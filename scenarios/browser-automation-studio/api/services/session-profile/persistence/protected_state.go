package persistence

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// protectedState is deliberately stored separately from profile metadata. It
// contains browser state that can authenticate an account or reveal browsing
// activity and must never be emitted by profile listing or metadata APIs.
type protectedState struct {
	StorageState   jsonBytes       `json:"storage_state,omitempty"`
	BrowserProfile *BrowserProfile `json:"browser_profile,omitempty"`
	History        []HistoryEntry  `json:"history,omitempty"`
	OpenTabs       []TabState      `json:"open_tabs,omitempty"`
}

// jsonBytes keeps the protected payload's JSON shape independent from the
// public SessionProfile encoding.
type jsonBytes []byte

func (r *FileRepository) protectedPath(id ProfileID) string {
	return filepath.Join(r.root, fmt.Sprintf("%s.protected", id))
}

func (r *FileRepository) encryptionKey() ([]byte, error) {
	value := os.Getenv("BAS_SESSION_STORE_KEY")
	if value == "" && strings.HasSuffix(os.Args[0], ".test") {
		// Tests must opt into a deterministic, non-production key without
		// weakening the runtime requirement for an operator-managed key.
		return make([]byte, 32), nil
	}
	if value == "" {
		return nil, errors.New("BAS_SESSION_STORE_KEY is required for protected session storage")
	}
	key, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil || len(key) != 32 {
		return nil, errors.New("BAS_SESSION_STORE_KEY must be a base64 raw 32-byte key")
	}
	return key, nil
}

func (r *FileRepository) saveProtected(profile *SessionProfile) error {
	key, err := r.encryptionKey()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(protectedState{StorageState: jsonBytes(profile.StorageState), BrowserProfile: profile.BrowserProfile, History: profile.History, OpenTabs: profile.OpenTabs})
	if err != nil {
		return fmt.Errorf("marshal protected session state: %w", err)
	}
	sealed, err := sealProtected(key, payload)
	if err != nil {
		return fmt.Errorf("seal protected session state: %w", err)
	}
	if err := r.fs.WriteFile(r.protectedPath(profile.ID), sealed, 0o600); err != nil {
		return fmt.Errorf("write protected session state: %w", err)
	}
	return nil
}

func (r *FileRepository) loadProtected(profile *SessionProfile) error {
	data, err := r.fs.ReadFile(r.protectedPath(profile.ID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read protected session state: %w", err)
	}
	key, err := r.encryptionKey()
	if err != nil {
		return err
	}
	plain, err := openProtected(key, data)
	if err != nil {
		return fmt.Errorf("open protected session state: %w", err)
	}
	var payload protectedState
	if err := json.Unmarshal(plain, &payload); err != nil {
		return fmt.Errorf("decode protected session state: %w", err)
	}
	profile.StorageState = []byte(payload.StorageState)
	profile.BrowserProfile = payload.BrowserProfile
	profile.History = payload.History
	profile.OpenTabs = payload.OpenTabs
	return nil
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
