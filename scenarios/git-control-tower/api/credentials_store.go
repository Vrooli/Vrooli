package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// StoredCredential is the internal representation with plaintext token.
type StoredCredential struct {
	ID        string         `json:"id"`
	Remote    string         `json:"remote"`
	URL       string         `json:"url"`
	Type      CredentialType `json:"type"`
	Username  string         `json:"username,omitempty"`
	Token     string         `json:"token,omitempty"` // Plaintext token (encrypted at rest)
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// CredentialsStore manages encrypted credential storage.
type CredentialsStore struct {
	mu        sync.RWMutex
	storePath string
	key       []byte
}

// credentialsFile is the JSON structure stored on disk (encrypted).
type credentialsFile struct {
	Version     int                `json:"version"`
	Credentials []StoredCredential `json:"credentials"`
}

// NewCredentialsStore creates a new credentials store.
// The store path defaults to ~/.config/git-control-tower/credentials.enc
func NewCredentialsStore(storePath string) (*CredentialsStore, error) {
	if storePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		storePath = filepath.Join(homeDir, ".config", "git-control-tower", "credentials.enc")
	}

	// Derive encryption key from machine-specific data
	key, err := deriveEncryptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(storePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &CredentialsStore{
		storePath: storePath,
		key:       key,
	}, nil
}

// deriveEncryptionKey derives a 32-byte key for AES-256.
// Uses machine ID and a salt for derivation.
func deriveEncryptionKey() ([]byte, error) {
	// Read machine ID for key derivation
	// On Linux: /etc/machine-id
	// Fallback: hostname + user
	var machineData string

	if data, err := os.ReadFile("/etc/machine-id"); err == nil {
		machineData = strings.TrimSpace(string(data))
	} else {
		// Fallback: use hostname + username
		hostname, _ := os.Hostname()
		username := os.Getenv("USER")
		if username == "" {
			username = os.Getenv("USERNAME")
		}
		machineData = hostname + username
	}

	// Add salt
	salt := "git-control-tower-credentials-v1"
	combined := machineData + salt

	// SHA-256 produces exactly 32 bytes (256 bits) for AES-256
	hash := sha256.Sum256([]byte(combined))
	return hash[:], nil
}

// loadCredentials loads and decrypts credentials from disk.
func (s *CredentialsStore) loadCredentials() ([]StoredCredential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []StoredCredential{}, nil
		}
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	if len(data) == 0 {
		return []StoredCredential{}, nil
	}

	// Decrypt the data
	plaintext, err := s.decrypt(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	var file credentialsFile
	if err := json.Unmarshal(plaintext, &file); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return file.Credentials, nil
}

// saveCredentials encrypts and saves credentials to disk.
func (s *CredentialsStore) saveCredentials(creds []StoredCredential) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file := credentialsFile{
		Version:     1,
		Credentials: creds,
	}

	plaintext, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize credentials: %w", err)
	}

	// Encrypt the data
	ciphertext, err := s.encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	// Write atomically using temp file
	tempPath := s.storePath + ".tmp"
	if err := os.WriteFile(tempPath, ciphertext, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	if err := os.Rename(tempPath, s.storePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	return nil
}

// encrypt uses AES-256-GCM to encrypt data.
func (s *CredentialsStore) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
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

	// Prepend nonce to ciphertext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt uses AES-256-GCM to decrypt data.
func (s *CredentialsStore) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// GetCredential retrieves a credential by ID.
func (s *CredentialsStore) GetCredential(id string) (*StoredCredential, error) {
	creds, err := s.loadCredentials()
	if err != nil {
		return nil, err
	}

	for _, cred := range creds {
		if cred.ID == id {
			return &cred, nil
		}
	}
	return nil, nil
}

// GetCredentialByRemote retrieves a credential by remote name.
func (s *CredentialsStore) GetCredentialByRemote(remote string) (*StoredCredential, error) {
	creds, err := s.loadCredentials()
	if err != nil {
		return nil, err
	}

	for _, cred := range creds {
		if cred.Remote == remote {
			return &cred, nil
		}
	}
	return nil, nil
}

// SaveCredential saves or updates a credential.
func (s *CredentialsStore) SaveCredential(cred StoredCredential) error {
	creds, err := s.loadCredentials()
	if err != nil {
		return err
	}

	// Update or append
	found := false
	for i, c := range creds {
		if c.ID == cred.ID {
			cred.CreatedAt = c.CreatedAt // Preserve original creation time
			cred.UpdatedAt = time.Now()
			creds[i] = cred
			found = true
			break
		}
	}

	if !found {
		cred.CreatedAt = time.Now()
		cred.UpdatedAt = cred.CreatedAt
		creds = append(creds, cred)
	}

	return s.saveCredentials(creds)
}

// DeleteCredential removes a credential by ID.
func (s *CredentialsStore) DeleteCredential(id string) error {
	creds, err := s.loadCredentials()
	if err != nil {
		return err
	}

	filtered := make([]StoredCredential, 0, len(creds))
	for _, c := range creds {
		if c.ID != id {
			filtered = append(filtered, c)
		}
	}

	return s.saveCredentials(filtered)
}

// ListCredentials returns all stored credentials.
func (s *CredentialsStore) ListCredentials() ([]StoredCredential, error) {
	return s.loadCredentials()
}

// maskToken masks a token for display, showing only first 4 and last 4 characters.
// Example: "ghp_xxxxxxxxxxxx1234" -> "ghp_****...1234"
func maskToken(token string) string {
	if token == "" {
		return ""
	}

	length := len(token)
	if length <= 8 {
		// Token too short, show all as asterisks
		return strings.Repeat("*", length)
	}

	// Show first 4 and last 4 characters
	prefix := token[:4]
	suffix := token[length-4:]
	return prefix + "****..." + suffix
}

// ToCredential converts a StoredCredential to a Credential (masks token).
func (sc *StoredCredential) ToCredential() Credential {
	return Credential{
		ID:           sc.ID,
		Remote:       sc.Remote,
		URL:          sc.URL,
		Type:         sc.Type,
		Username:     sc.Username,
		TokenMasked:  maskToken(sc.Token),
		IsConfigured: sc.Token != "" && sc.Username != "",
		CreatedAt:    sc.CreatedAt,
		UpdatedAt:    sc.UpdatedAt,
	}
}

// credentialIDFromRemote generates a stable ID from remote name.
func credentialIDFromRemote(remote string) string {
	// Use remote name as ID (simple and stable)
	return "cred-" + remote
}
