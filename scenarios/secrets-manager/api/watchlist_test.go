package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"testing"
)

func resetWatchlistKeyState(t *testing.T) {
	t.Helper()
	watchlistKeyOnce = sync.Once{}
	watchlistKeyBytes = nil
	watchlistKeyErr = nil
	watchlistKeyWarnOnce = sync.Once{}
}

func TestWatchlistKey_UnsetReturnsNotConfigured(t *testing.T) {
	t.Setenv(watchlistEncryptionKeyEnv, "")
	resetWatchlistKeyState(t)
	if _, err := watchlistKey(); !errors.Is(err, errWatchlistKeyNotConfigured) {
		t.Errorf("expected errWatchlistKeyNotConfigured, got %v", err)
	}
	if watchlistKeyAvailable() {
		t.Errorf("expected watchlistKeyAvailable to be false with no env var")
	}
}

func TestWatchlistKey_InvalidHex(t *testing.T) {
	t.Setenv(watchlistEncryptionKeyEnv, "not-hex")
	resetWatchlistKeyState(t)
	_, err := watchlistKey()
	if err == nil || !errors.Is(err, errWatchlistKeyInvalid) {
		t.Errorf("expected errWatchlistKeyInvalid, got %v", err)
	}
}

func TestWatchlistKey_WrongLength(t *testing.T) {
	t.Setenv(watchlistEncryptionKeyEnv, strings.Repeat("ab", 10)) // 10 bytes
	resetWatchlistKeyState(t)
	_, err := watchlistKey()
	if err == nil || !errors.Is(err, errWatchlistKeyInvalid) {
		t.Errorf("expected errWatchlistKeyInvalid for wrong length, got %v", err)
	}
}

func TestWatchlistEncryptDecrypt_RoundTrip(t *testing.T) {
	t.Setenv(watchlistEncryptionKeyEnv, strings.Repeat("ab", 32)) // 32 bytes hex
	resetWatchlistKeyState(t)
	plain := []byte("alice@example.com")
	ct, err := encryptWatchlistValue(plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if bytes.Equal(ct, plain) {
		t.Errorf("ciphertext should differ from plaintext")
	}
	decrypted, err := decryptWatchlistValue(ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(decrypted, plain) {
		t.Errorf("roundtrip mismatch: got %q want %q", decrypted, plain)
	}
}

func TestWatchlistDecrypt_WrongKey(t *testing.T) {
	// First encrypt with one key.
	t.Setenv(watchlistEncryptionKeyEnv, strings.Repeat("ab", 32))
	resetWatchlistKeyState(t)
	ct, err := encryptWatchlistValue([]byte("secret"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	// Switch to a different key and try to decrypt.
	otherKey := make([]byte, 32)
	for i := range otherKey {
		otherKey[i] = byte(i + 1)
	}
	t.Setenv(watchlistEncryptionKeyEnv, hex.EncodeToString(otherKey))
	resetWatchlistKeyState(t)
	if _, err := decryptWatchlistValue(ct); err == nil {
		t.Errorf("expected decrypt with wrong key to fail")
	}
}

func TestWatchlistEncrypt_NoncesDiffer(t *testing.T) {
	t.Setenv(watchlistEncryptionKeyEnv, strings.Repeat("cd", 32))
	resetWatchlistKeyState(t)
	a, err := encryptWatchlistValue([]byte("same"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := encryptWatchlistValue([]byte("same"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a, b) {
		t.Errorf("expected independent nonces to produce different ciphertexts")
	}
}
