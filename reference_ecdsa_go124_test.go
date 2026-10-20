//go:build go1.24

package tls

// Go 1.24 changed ECDSA nonce derivation. Keep matching reference transcripts
// without changing the signing implementation used by the library.
const modernECDSATranscripts = true
