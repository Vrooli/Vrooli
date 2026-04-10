package identity

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTokenRoundTrip(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!!")

	claims := &Claims{
		RunID:      uuid.New(),
		TaskID:     uuid.New(),
		ProfileKey: "test-profile",
		ScopePath:  "scenarios/test",
		IssuedAt:   time.Now().Unix(),
		ExpiresAt:  time.Now().Add(DefaultTTL).Unix(),
		Meta:       map[string]string{"foo": "bar"},
	}

	token, err := GenerateToken(claims, secret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	got, err := VerifyToken(token, secret)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}

	if got.RunID != claims.RunID {
		t.Errorf("RunID = %v, want %v", got.RunID, claims.RunID)
	}
	if got.TaskID != claims.TaskID {
		t.Errorf("TaskID = %v, want %v", got.TaskID, claims.TaskID)
	}
	if got.ProfileKey != claims.ProfileKey {
		t.Errorf("ProfileKey = %q, want %q", got.ProfileKey, claims.ProfileKey)
	}
	if got.ScopePath != claims.ScopePath {
		t.Errorf("ScopePath = %q, want %q", got.ScopePath, claims.ScopePath)
	}
	if got.Meta["foo"] != "bar" {
		t.Errorf("Meta[foo] = %q, want %q", got.Meta["foo"], "bar")
	}
}

func TestExpiredToken(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!!")

	claims := &Claims{
		RunID:     uuid.New(),
		TaskID:    uuid.New(),
		IssuedAt:  time.Now().Add(-2 * time.Hour).Unix(),
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(), // expired 1 hour ago
		Meta:      map[string]string{},
	}

	token, err := GenerateToken(claims, secret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	_, err = VerifyToken(token, secret)
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestTamperedClaims(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!!")

	claims := &Claims{
		RunID:     uuid.New(),
		TaskID:    uuid.New(),
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(DefaultTTL).Unix(),
		Meta:      map[string]string{},
	}

	token, err := GenerateToken(claims, secret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	// Tamper with the payload by replacing a character.
	tampered := "X" + token[1:]
	_, err = VerifyToken(tampered, secret)
	if err == nil {
		t.Error("expected error for tampered token, got nil")
	}
}

func TestTamperedSignature(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!!")

	claims := &Claims{
		RunID:     uuid.New(),
		TaskID:    uuid.New(),
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(DefaultTTL).Unix(),
		Meta:      map[string]string{},
	}

	token, err := GenerateToken(claims, secret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	// Verify with wrong secret.
	wrongSecret := []byte("wrong-secret-key-32-bytes-long!")
	token2, _ := GenerateToken(claims, wrongSecret)

	_, err = VerifyToken(token, wrongSecret)
	if err != ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}

	// Also verify token2 fails with original secret.
	_, err = VerifyToken(token2, secret)
	if err != ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature for token2 with original secret, got %v", err)
	}
}

func TestMalformedToken(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!!")

	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"no dot", "nodothere"},
		{"empty parts", "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyToken(tt.token, secret)
			if err == nil {
				t.Error("expected error for malformed token, got nil")
			}
		})
	}
}

func TestHashToken(t *testing.T) {
	hash := HashToken("test-token")
	if len(hash) != 64 { // SHA-256 hex = 64 chars
		t.Errorf("hash length = %d, want 64", len(hash))
	}

	// Same input produces same hash.
	if HashToken("test-token") != hash {
		t.Error("hash is not deterministic")
	}

	// Different input produces different hash.
	if HashToken("different-token") == hash {
		t.Error("different inputs produced same hash")
	}
}

func TestSecretPersistence(t *testing.T) {
	dir := t.TempDir()

	// First call should create the secret.
	secret1, err := LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret (create): %v", err)
	}
	if len(secret1) != secretSize {
		t.Fatalf("secret length = %d, want %d", len(secret1), secretSize)
	}

	// Second call should load the same secret.
	secret2, err := LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret (load): %v", err)
	}

	if string(secret1) != string(secret2) {
		t.Error("loaded secret does not match created secret")
	}
}

func TestSecretFilePermissions(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}

	path := filepath.Join(dir, secretFileName)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat secret file: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestSecretInvalidSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, secretFileName)

	// Write a file with wrong size.
	if err := os.WriteFile(path, []byte("too-short"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadOrCreateSecret(dir)
	if err == nil {
		t.Error("expected error for invalid size secret, got nil")
	}
}
