package securevalue

import (
	"errors"
	"strings"
	"testing"
)

func TestRingRotationRetainsHistoricalReaders(t *testing.T) {
	ring, err := NewRing()
	if err != nil {
		t.Fatal(err)
	}
	first, err := EncryptRing(ring, "before")
	if err != nil {
		t.Fatal(err)
	}
	ring, err = ring.Rotate()
	if err != nil {
		t.Fatal(err)
	}
	second, err := EncryptRing(ring, "after")
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := DecryptRing(ring, first); got != "before" {
		t.Fatalf("historical plaintext = %q", got)
	}
	if got, _ := DecryptRing(ring, second); got != "after" {
		t.Fatalf("active plaintext = %q", got)
	}
}

func TestRingRejectsUnknownVersion(t *testing.T) {
	ring, err := NewRing()
	if err != nil {
		t.Fatal(err)
	}
	_, err = DecryptRing(ring, "v99:invalid")
	if !errors.Is(err, ErrUnknownKeyVersion) {
		t.Fatalf("error = %v, want unknown version", err)
	}
}

func TestRingReadsLegacyCiphertext(t *testing.T) {
	ring, err := NewRing()
	if err != nil {
		t.Fatal(err)
	}
	key, err := ring.ActiveKey()
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := Encrypt(key, "legacy")
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecryptRing(ring, legacy)
	if err != nil || got != "legacy" {
		t.Fatalf("legacy decrypt = %q, %v", got, err)
	}
}

func TestDecryptRingDoesNotMistakeLegacyBase64ForVersionPrefix(t *testing.T) {
	ring, err := NewRing()
	if err != nil {
		t.Fatal(err)
	}
	key, err := ring.ActiveKey()
	if err != nil {
		t.Fatal(err)
	}
	var legacy string
	for i := 0; i < 10000; i++ {
		candidate, err := Encrypt(key, "legacy")
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(candidate, "v") && !strings.Contains(candidate, ":") {
			legacy = candidate
			break
		}
	}
	if legacy == "" {
		t.Fatal("failed to generate a legacy ciphertext beginning with v")
	}
	got, err := DecryptRing(ring, legacy)
	if err != nil || got != "legacy" {
		t.Fatalf("legacy ciphertext beginning with v decrypted as %q: %v", got, err)
	}
}

func TestParseRingRejectsInvalidShape(t *testing.T) {
	if _, err := ParseRing(`{"active":0,"keys":[]}`); err == nil {
		t.Fatal("accepted ring without active version")
	}
	if _, err := ParseRing(`{"active":1,"keys":[{"version":1,"key":"aA=="}]}`); err == nil {
		t.Fatal("accepted ring with short key")
	}
}
