package tls

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func TestRestlsConcurrentClientReadWrite(t *testing.T) {
	target, stop := startRestlsTestTLSTarget(t, VersionTLS13, nil)
	defer stop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clientRaw, serverRaw := localPipe(t)
	defer clientRaw.Close()
	defer serverRaw.Close()
	clientRaw.SetDeadline(time.Now().Add(10 * time.Second))
	serverRaw.SetDeadline(time.Now().Add(10 * time.Second))
	const password = "concurrent-restls-regression"
	ready := make(chan net.Conn, 1)
	errors := make(chan error, 5)
	go func() {
		c, err := RestlsServer(ctx, serverRaw, &RestlsServerConfig{ServerHostname: target, Password: password})
		if err != nil {
			errors <- err
			return
		}
		ready <- c
	}()
	clientConfig := newRestlsServerTestClientConfig(t, password, "tls13", "", "chrome", VersionTLS13)
	client := UClient(clientRaw, clientConfig, HelloChrome_Auto)
	defer client.Close()
	if err := client.HandshakeContext(ctx); err != nil {
		t.Fatal(err)
	}
	// Complete the switch from camouflage TLS to authenticated ResTLS records
	// before exercising application-level full-duplex traffic.
	if _, err := client.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	var server net.Conn
	select {
	case server = <-ready:
	case err := <-errors:
		t.Fatal(err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	defer server.Close()
	go func() {
		var ping [4]byte
		_, err := io.ReadFull(server, ping[:])
		if err == nil {
			_, err = server.Write([]byte("pong"))
		}
		errors <- err
	}()
	var pong [4]byte
	if _, err := io.ReadFull(client, pong[:]); err != nil || string(pong[:]) != "pong" {
		t.Fatalf("initial exchange: %q, %v", pong, err)
	}
	if err := <-errors; err != nil {
		t.Fatal(err)
	}
	upload := bytes.Repeat([]byte{0x41}, 256<<10)
	download := bytes.Repeat([]byte{0x42}, 256<<10)
	go func() { _, err := client.Write(upload); errors <- err }()
	go func() { _, err := server.Write(download); errors <- err }()
	for _, transfer := range []struct {
		reader net.Conn
		want   []byte
	}{{client, download}, {server, upload}} {
		transfer := transfer
		go func() {
			got := make([]byte, len(transfer.want))
			_, err := io.ReadFull(transfer.reader, got)
			if err == nil && !bytes.Equal(got, transfer.want) {
				err = fmt.Errorf("corrupted full-duplex transfer")
			}
			errors <- err
		}()
	}
	for n := 0; n < 4; n++ {
		select {
		case err := <-errors:
			if err != nil {
				t.Fatal(err)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
}
