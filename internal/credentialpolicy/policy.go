// Package credentialpolicy contains cryptographic policy shared by the
// authority and encrypted credential-store adapters.
package credentialpolicy

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

const RecoveryPBKDF2Iterations = 600_000

// Envelope is the algorithm-independent portion of a credential envelope.
// Callers own the on-disk or wire framing; this type owns the cryptographic
// operation and the authenticated context shared by every credential path.
type Envelope struct {
	Version    int
	Purpose    string
	Nonce      []byte
	Ciphertext []byte
}

const envelopeContext = "vrooli.credential-envelope"

var ErrInvalidEnvelope = errors.New("invalid credential envelope")

// Seal encrypts plaintext under key and authenticates the format version and
// purpose. Purpose is mandatory so one credential artifact cannot be replayed
// as a different artifact type.
func Seal(key, plaintext []byte, purpose string, version int) (Envelope, error) {
	if len(key) == 0 {
		return Envelope{}, fmt.Errorf("%w: empty key", ErrInvalidEnvelope)
	}
	if strings.TrimSpace(purpose) == "" {
		return Envelope{}, fmt.Errorf("%w: empty purpose", ErrInvalidEnvelope)
	}
	if version <= 0 {
		return Envelope{}, fmt.Errorf("%w: invalid version", ErrInvalidEnvelope)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return Envelope{}, fmt.Errorf("create credential cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return Envelope{}, fmt.Errorf("create credential AEAD: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return Envelope{}, fmt.Errorf("generate credential nonce: %w", err)
	}
	return Envelope{
		Version:    version,
		Purpose:    purpose,
		Nonce:      nonce,
		Ciphertext: gcm.Seal(nil, nonce, plaintext, AdditionalData(version, purpose)),
	}, nil
}

// Open authenticates and decrypts an envelope. It never returns plaintext on
// an authentication failure.
func Open(key []byte, envelope Envelope) ([]byte, error) {
	if len(key) == 0 || envelope.Version <= 0 || strings.TrimSpace(envelope.Purpose) == "" {
		return nil, fmt.Errorf("%w: incomplete envelope", ErrInvalidEnvelope)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create credential cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create credential AEAD: %w", err)
	}
	if len(envelope.Nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("%w: nonce is %d bytes, want %d", ErrInvalidEnvelope, len(envelope.Nonce), gcm.NonceSize())
	}
	plain, err := gcm.Open(nil, envelope.Nonce, envelope.Ciphertext, AdditionalData(envelope.Version, envelope.Purpose))
	if err != nil {
		return nil, fmt.Errorf("%w: authentication failed", ErrInvalidEnvelope)
	}
	return plain, nil
}

// AdditionalData is exported for compatibility readers that need to compose a
// larger context (for example a provider name inside a key-wrap purpose).
func AdditionalData(version int, purpose string) []byte {
	result := make([]byte, 0, len(envelopeContext)+len(purpose)+16)
	result = appendLengthPrefixed(result, []byte(envelopeContext))
	result = binary.BigEndian.AppendUint32(result, uint32(version))
	return appendLengthPrefixed(result, []byte(purpose))
}

func appendLengthPrefixed(dst, value []byte) []byte {
	dst = binary.BigEndian.AppendUint32(dst, uint32(len(value)))
	return append(dst, value...)
}
