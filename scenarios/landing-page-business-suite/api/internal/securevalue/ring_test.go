package securevalue

import (
	"errors"
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

func TestParseRingRejectsInvalidShape(t *testing.T) {
	if _, err := ParseRing(`{"active":0,"keys":[]}`); err == nil {
		t.Fatal("accepted ring without active version")
	}
	if _, err := ParseRing(`{"active":1,"keys":[{"version":1,"key":"aA=="}]}`); err == nil {
		t.Fatal("accepted ring with short key")
	}
}
