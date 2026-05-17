package byokstore

import (
	"bytes"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func mustEncryptor(t *testing.T, hexKey string) *Encryptor {
	t.Helper()
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		t.Fatalf("decode key: %v", err)
	}
	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	return enc
}

func TestEncryptorRoundTrip(t *testing.T) {
	enc := mustEncryptor(t, "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	plaintext := []byte("sk-test-byok-secret-payload")

	ct, err := enc.Seal(plaintext)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if bytes.Contains(ct, plaintext) {
		t.Fatalf("ciphertext leaked plaintext bytes")
	}

	got, err := enc.Open(ct)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("round-trip mismatch: %q != %q", got, plaintext)
	}
}

func TestEncryptorTamperRejected(t *testing.T) {
	enc := mustEncryptor(t, "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	ct, err := enc.Seal([]byte("payload"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	// Flip a byte deep in the ciphertext body (past the nonce).
	tampered := append([]byte(nil), ct...)
	tampered[len(tampered)-1] ^= 0x01
	if _, err := enc.Open(tampered); err == nil {
		t.Fatalf("expected open to reject tampered ciphertext, got nil error")
	}
}

func TestEncryptorWrongKeyRejected(t *testing.T) {
	encA := mustEncryptor(t, "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	encB := mustEncryptor(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	ct, err := encA.Seal([]byte("secret"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if _, err := encB.Open(ct); err == nil {
		t.Fatalf("expected open to fail under a different key, got nil error")
	}
}

func TestEncryptorShortCiphertextRejected(t *testing.T) {
	enc := mustEncryptor(t, "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	if _, err := enc.Open([]byte("short")); !errors.Is(err, ErrInvalidCiphertext) {
		t.Fatalf("expected ErrInvalidCiphertext, got %v", err)
	}
}

func TestNewEncryptorRejectsShortKey(t *testing.T) {
	if _, err := NewEncryptor(make([]byte, 16)); err == nil {
		t.Fatalf("expected NewEncryptor to reject a 16-byte key")
	}
}

func TestSealNonDeterministic(t *testing.T) {
	enc := mustEncryptor(t, "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	plaintext := []byte("payload")
	a, err := enc.Seal(plaintext)
	if err != nil {
		t.Fatalf("seal a: %v", err)
	}
	b, err := enc.Seal(plaintext)
	if err != nil {
		t.Fatalf("seal b: %v", err)
	}
	if bytes.Equal(a, b) {
		t.Fatalf("expected distinct nonces to produce distinct ciphertexts")
	}
}

func TestFingerprintDeterministicAndRedacted(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "***"},
		{"abc", "***"},
		{"abcde", "***"},
		{"abcdef", "a***ef"},
		{"abcdefg", "a***fg"},
		{"abcdefghij", "ab***ghij"},
		{"sk-test-byok-secret-payload", "sk***load"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := Fingerprint(tc.in)
			if got != tc.want {
				t.Fatalf("Fingerprint(%q) = %q, want %q", tc.in, got, tc.want)
			}
			// Same input must yield same fingerprint (determinism).
			if again := Fingerprint(tc.in); again != got {
				t.Fatalf("fingerprint changed across calls: %q vs %q", got, again)
			}
		})
	}
}

func TestLoadOrCreateKeyEnvWins(t *testing.T) {
	hexKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	t.Setenv(KeyEnvVar, hexKey)
	dir := t.TempDir()
	got, err := LoadOrCreateKey(filepath.Join(dir, "key.hex"))
	if err != nil {
		t.Fatalf("LoadOrCreateKey: %v", err)
	}
	want, _ := hex.DecodeString(hexKey)
	if !bytes.Equal(got, want) {
		t.Fatalf("env-supplied key not returned")
	}
}

func TestLoadOrCreateKeyPersists(t *testing.T) {
	t.Setenv(KeyEnvVar, "")
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.hex")

	k1, err := LoadOrCreateKey(keyPath)
	if err != nil {
		t.Fatalf("first LoadOrCreateKey: %v", err)
	}
	if len(k1) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(k1))
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("expected key file written: %v", err)
	}

	k2, err := LoadOrCreateKey(keyPath)
	if err != nil {
		t.Fatalf("second LoadOrCreateKey: %v", err)
	}
	if !bytes.Equal(k1, k2) {
		t.Fatalf("key not stable across loads")
	}
}
