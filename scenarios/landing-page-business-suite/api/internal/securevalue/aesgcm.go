// Package securevalue provides the authenticated encryption primitive used for
// sensitive values persisted by the API.
package securevalue

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Encrypt returns plaintext sealed with AES-GCM. The encoded payload prefixes
// the ciphertext with its random nonce, so callers can persist one value.
//
// A nil key deliberately preserves plaintext. Callers decide whether that is
// acceptable (for example, development-only storage); this package never
// chooses an environment or secret policy on their behalf.
func Encrypt(key []byte, plaintext string) (string, error) {
	if key == nil {
		return plaintext, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt opens a value returned by Encrypt.
//
// A nil key deliberately preserves ciphertext for the same development-only
// policy controlled by the caller.
func Decrypt(key []byte, ciphertext string) (string, error) {
	if key == nil {
		return ciphertext, nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
