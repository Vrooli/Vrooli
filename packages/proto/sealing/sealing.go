// Package sealing provides the small, dependency-free envelope used for
// operator secrets crossing the Bridge control plane. It deliberately lives
// in the shared wire module so the CLI and node helper cannot implement
// subtly different formats.
package sealing

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
)

const (
	version       = "VCS1"
	publicKeySize = 32
	maxPlaintext  = 64 * 1024
)

// Context creates the canonical additional-authentication-data binding for a
// cleanup authorization envelope. Empty fields remain positional so target,
// operation, and plan substitutions cannot produce the same bytes.
func Context(machineID, nodeID, target, scope, planHash, operationID, operatorID string) []byte {
	return []byte(strings.Join([]string{"vrooli-cleanup-context-v1", machineID, nodeID, target, scope, planHash, operationID, operatorID}, "\x00"))
}

// CredentialContext binds a fleet credential envelope to the exact node,
// address, and source generation it was created for. A frame cannot be
// replayed for another node, field, or rotation generation.
func CredentialContext(nodeID, logicalID, field string, generation int64) []byte {
	return []byte(strings.Join([]string{"vrooli-credential-context-v1", nodeID, logicalID, field, fmt.Sprint(generation)}, "\x00"))
}

// PrivateKeyFromRaw loads the independently stored X25519 private key used for
// envelope decryption. It intentionally accepts only raw X25519 material; no
// No Ed25519-to-X25519 conversion exists in this package.
func PrivateKeyFromRaw(raw []byte) (*ecdh.PrivateKey, error) {
	if len(raw) != publicKeySize {
		return nil, fmt.Errorf("sealing: X25519 private key must be %d bytes", publicKeySize)
	}
	return ecdh.X25519().NewPrivateKey(raw)
}

// Seal encrypts plaintext to the recipient's X25519 public key. The returned
// bytes are safe to persist and relay as opaque data. AAD binds the envelope
// to the operation context (machine, target, plan hash, and operator id).
func Seal(recipientPublic, plaintext, aad []byte) ([]byte, error) {
	if len(recipientPublic) != publicKeySize {
		return nil, errors.New("sealing: recipient public key must be 32 bytes")
	}
	if len(plaintext) == 0 || len(plaintext) > maxPlaintext {
		return nil, errors.New("sealing: plaintext must be between 1 and 65536 bytes")
	}
	recipient, err := ecdh.X25519().NewPublicKey(recipientPublic)
	if err != nil {
		return nil, fmt.Errorf("sealing: recipient key: %w", err)
	}
	ephemeral, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("sealing: ephemeral key: %w", err)
	}
	shared, err := ephemeral.ECDH(recipient)
	if err != nil {
		return nil, fmt.Errorf("sealing: derive shared key: %w", err)
	}
	defer zero(shared)
	key := derive(shared, ephemeral.PublicKey().Bytes(), recipientPublic, aad)
	defer zero(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("sealing: cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("sealing: gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("sealing: nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, aad)
	out := make([]byte, 0, len(version)+publicKeySize+len(nonce)+len(ciphertext))
	out = append(out, version...)
	out = append(out, ephemeral.PublicKey().Bytes()...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

// Open decrypts an envelope using the node's derived private key and verifies
// the same context binding supplied to Seal.
func Open(private *ecdh.PrivateKey, envelope, aad []byte) ([]byte, error) {
	if private == nil || private.Curve() != ecdh.X25519() {
		return nil, errors.New("sealing: X25519 private key is required")
	}
	if len(envelope) < len(version)+publicKeySize+12+16 || string(envelope[:len(version)]) != version {
		return nil, errors.New("sealing: malformed envelope")
	}
	offset := len(version)
	ephemeralRaw := envelope[offset : offset+publicKeySize]
	offset += publicKeySize
	ephemeral, err := ecdh.X25519().NewPublicKey(ephemeralRaw)
	if err != nil {
		return nil, fmt.Errorf("sealing: ephemeral key: %w", err)
	}
	shared, err := private.ECDH(ephemeral)
	if err != nil {
		return nil, fmt.Errorf("sealing: derive shared key: %w", err)
	}
	defer zero(shared)
	key := derive(shared, ephemeralRaw, private.PublicKey().Bytes(), aad)
	defer zero(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("sealing: cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("sealing: gcm: %w", err)
	}
	if len(envelope) < offset+gcm.NonceSize()+gcm.Overhead() {
		return nil, errors.New("sealing: truncated envelope")
	}
	nonce := envelope[offset : offset+gcm.NonceSize()]
	ciphertext := envelope[offset+gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, errors.New("sealing: authentication failed")
	}
	if len(plaintext) == 0 || len(plaintext) > maxPlaintext {
		zero(plaintext)
		return nil, errors.New("sealing: invalid plaintext size")
	}
	return plaintext, nil
}

func derive(shared, ephemeral, recipient, aad []byte) []byte {
	salt := []byte("vrooli-cleanup-sealing-v1")
	extractor := hmac.New(sha256.New, salt)
	_, _ = extractor.Write(shared)
	prk := extractor.Sum(nil)
	defer zero(prk)
	expander := hmac.New(sha256.New, prk)
	_, _ = expander.Write([]byte("envelope"))
	_, _ = expander.Write(ephemeral)
	_, _ = expander.Write(recipient)
	_, _ = expander.Write(aad)
	_, _ = expander.Write([]byte{1})
	return expander.Sum(nil)
}

func zero(raw []byte) {
	for i := range raw {
		raw[i] = 0
	}
}
