ECDSA reference transcripts for Go 1.24 and newer.

Server-* originated in getmikan/utls commit
32502cba39f41e96f4885b91ef6d3b48084b4f61.
Client-* were recorded against OpenSSL 1.1.1w using Go 1.26.9:
  go test -run ^TestHandshakeClientCertECDSA$ -update -v .

The original pre-Go-1.24 transcripts remain in the parent directory and are
still exercised by the Go 1.20 CI job. No production signing behavior changes.
