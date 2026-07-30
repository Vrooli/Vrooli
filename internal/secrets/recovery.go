package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// RecoveryEntry identifies a value to include in an encrypted recovery
// bundle. Callers supply descriptors; the authority never enumerates secrets.
type RecoveryEntry struct {
	Identity Identity `json:"identity"`
	Field    string   `json:"field"`
}

type recoveryPayload struct {
	Version int             `json:"version"`
	Values  []recoveryValue `json:"values"`
}
type recoveryValue struct {
	Identity Identity `json:"identity"`
	Field    string   `json:"field"`
	Value    string   `json:"value"`
}
type recoveryEnvelope struct {
	Version    int    `json:"version"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// ExportRecovery creates an opaque encrypted recovery bundle for explicitly
// selected credentials. The passphrase exists only in the caller's memory.
func (a *Authority) ExportRecovery(entries []RecoveryEntry, passphrase string) ([]byte, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("at least one recovery entry is required")
	}
	if strings.TrimSpace(passphrase) == "" {
		return nil, fmt.Errorf("recovery passphrase is required")
	}
	payload := recoveryPayload{Version: 1, Values: make([]recoveryValue, 0, len(entries))}
	for _, entry := range entries {
		identity, err := ParseIdentity(string(entry.Identity))
		if err != nil {
			return nil, err
		}
		field := strings.TrimSpace(entry.Field)
		if field == "" {
			return nil, fmt.Errorf("recovery field is required")
		}
		value, err := a.read(identity, field)
		if err != nil {
			return nil, fmt.Errorf("recover %s/%s: %w", identity, field, err)
		}
		payload.Values = append(payload.Values, recoveryValue{Identity: identity, Field: field, Value: value})
	}
	plain, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return encryptRecovery(plain, passphrase)
}

// RestoreRecovery decrypts and writes a bundle only through this authority.
func (a *Authority) RestoreRecovery(bundle []byte, passphrase string) error {
	if strings.TrimSpace(passphrase) == "" {
		return fmt.Errorf("recovery passphrase is required")
	}
	plain, err := decryptRecovery(bundle, passphrase)
	if err != nil {
		return err
	}
	var payload recoveryPayload
	if err := json.Unmarshal(plain, &payload); err != nil {
		return fmt.Errorf("decode recovery bundle: %w", err)
	}
	if payload.Version != 1 || len(payload.Values) == 0 {
		return fmt.Errorf("unsupported or empty recovery bundle")
	}
	for _, value := range payload.Values {
		if err := a.Put(value.Identity, value.Field, value.Value); err != nil {
			return fmt.Errorf("restore %s/%s: %w", value.Identity, value.Field, err)
		}
	}
	return nil
}

func (a *Authority) read(identity Identity, field string) (string, error) {
	if a == nil || a.store == nil {
		return "", ErrUnsupportedProvider
	}
	a.mu.Lock()
	value, err := a.store.Get(credentialService, string(identity)+":"+field)
	a.mu.Unlock()
	if err != nil || strings.TrimSpace(value) == "" {
		return "", ErrUnconfigured
	}
	return value, nil
}

func encryptRecovery(plain []byte, passphrase string) ([]byte, error) {
	salt, nonce := make([]byte, 32), make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	key, err := recoveryKey(passphrase, salt)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	sealed := cipher.NewGCM
	gcm, err := sealed(block)
	if err != nil {
		return nil, err
	}
	envelope := recoveryEnvelope{Version: 1, Salt: base64.StdEncoding.EncodeToString(salt), Nonce: base64.StdEncoding.EncodeToString(nonce), Ciphertext: base64.StdEncoding.EncodeToString(gcm.Seal(nil, nonce, plain, nil))}
	return json.Marshal(envelope)
}

func decryptRecovery(bundle []byte, passphrase string) ([]byte, error) {
	var envelope recoveryEnvelope
	if err := json.Unmarshal(bundle, &envelope); err != nil {
		return nil, fmt.Errorf("decode recovery envelope: %w", err)
	}
	if envelope.Version != 1 {
		return nil, fmt.Errorf("unsupported recovery bundle version")
	}
	salt, err := base64.StdEncoding.DecodeString(envelope.Salt)
	if err != nil {
		return nil, fmt.Errorf("decode recovery salt: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decode recovery nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode recovery ciphertext: %w", err)
	}
	key, err := recoveryKey(passphrase, salt)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt recovery bundle: authorization failed")
	}
	return plain, nil
}

func recoveryKey(passphrase string, salt []byte) ([]byte, error) {
	return pbkdf2.Key(sha256.New, passphrase, salt, 600_000, 32)
}
