package byokstore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// KeyEnvVar names the env var consulted before falling back to a
// persisted key file.
const KeyEnvVar = "AUDIO_TOOLS_DB_KEY"

// Encryptor wraps a 256-bit AES-GCM key.
type Encryptor struct {
	aead cipher.AEAD
}

// NewEncryptor builds an Encryptor from a 32-byte key.
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("byokstore: key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Encryptor{aead: aead}, nil
}

// Seal returns ciphertext = nonce || gcm(plaintext).
func (e *Encryptor) Seal(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ct := e.aead.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ct...), nil
}

// Open reverses Seal. Returns ErrInvalidCiphertext when the input is
// malformed.
func (e *Encryptor) Open(ciphertext []byte) ([]byte, error) {
	ns := e.aead.NonceSize()
	if len(ciphertext) < ns+e.aead.Overhead() {
		return nil, ErrInvalidCiphertext
	}
	nonce, ct := ciphertext[:ns], ciphertext[ns:]
	return e.aead.Open(nil, nonce, ct, nil)
}

// ErrInvalidCiphertext indicates the input is shorter than the AEAD
// requires or otherwise corrupt.
var ErrInvalidCiphertext = errors.New("byokstore: invalid ciphertext")

// LoadOrCreateKey reads the key from AUDIO_TOOLS_DB_KEY (hex). If the
// env var is empty it reads/persists a key file at keyPath, creating a
// new random key on first boot. Returns the 32-byte key.
func LoadOrCreateKey(keyPath string) ([]byte, error) {
	if hexKey := os.Getenv(KeyEnvVar); hexKey != "" {
		k, err := hex.DecodeString(hexKey)
		if err != nil {
			return nil, fmt.Errorf("byokstore: invalid %s: %w", KeyEnvVar, err)
		}
		if len(k) != 32 {
			return nil, fmt.Errorf("byokstore: %s must be 32 bytes (64 hex chars)", KeyEnvVar)
		}
		return k, nil
	}
	if b, err := os.ReadFile(keyPath); err == nil {
		k, decErr := hex.DecodeString(string(b))
		if decErr == nil && len(k) == 32 {
			return k, nil
		}
		// Fall through to regenerate on corrupt file.
	}
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyPath, []byte(hex.EncodeToString(k)), 0o600); err != nil {
		return nil, err
	}
	return k, nil
}
