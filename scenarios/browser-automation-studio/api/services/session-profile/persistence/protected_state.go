package persistence

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// profileDocument has one commit point for metadata and authenticated browser state.
// The encrypted payload is the complete SessionProfile, including its identity.
type profileDocument struct {
	Version    int    `json:"version"`
	KeyVersion int    `json:"key_version"`
	Sealed     []byte `json:"sealed"`
}

func (r *FileRepository) encodeProfile(profile *SessionProfile) ([]byte, error) {
	ring, err := r.encryptionKeys()
	if err != nil {
		return nil, err
	}
	plain, err := json.Marshal(profile)
	if err != nil {
		return nil, fmt.Errorf("encode profile: %w", err)
	}
	sealed, err := sealProtected(ring.Keys[ring.Active], plain)
	if err != nil {
		return nil, fmt.Errorf("seal profile: %w", err)
	}
	return json.Marshal(profileDocument{Version: 1, KeyVersion: ring.Active, Sealed: sealed})
}

func (r *FileRepository) decodeProfile(id ProfileID, data []byte) (*SessionProfile, error) {
	var document profileDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse profile document: %w", err)
	}
	if document.Version != 1 || document.KeyVersion < 1 || len(document.Sealed) == 0 {
		return nil, errors.New("unsupported or incomplete profile document; restore or convert the saved profile offline")
	}
	ring, err := r.encryptionKeys()
	if err != nil {
		return nil, err
	}
	key, ok := ring.Keys[document.KeyVersion]
	if !ok {
		return nil, fmt.Errorf("profile requires missing encryption key version %d; restore the credential keyring", document.KeyVersion)
	}
	plain, err := openProtected(key, document.Sealed)
	if err != nil {
		return nil, fmt.Errorf("open protected profile: %w", err)
	}
	var profile SessionProfile
	if err := json.Unmarshal(plain, &profile); err != nil {
		return nil, fmt.Errorf("decode profile: %w", err)
	}
	if profile.ID != id {
		return nil, errors.New("saved profile identity does not match its address")
	}
	return &profile, nil
}

func sealProtected(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return append(nonce, gcm.Seal(nil, nonce, plaintext, nil)...), nil
}

func openProtected(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("protected session payload is truncated")
	}
	return gcm.Open(nil, ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():], nil)
}
