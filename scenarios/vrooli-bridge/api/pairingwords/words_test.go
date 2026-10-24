package pairingwords

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func TestDeriveIsStableAndThreeWords(t *testing.T) {
	cp, _, _ := ed25519.GenerateKey(rand.Reader)
	node, _, _ := ed25519.GenerateKey(rand.Reader)
	control := base64.StdEncoding.EncodeToString(cp)
	public := base64.StdEncoding.EncodeToString(node)
	first, err := String(control, public)
	if err != nil {
		t.Fatal(err)
	}
	second, err := String(control, public)
	if err != nil || first != second || len(strings.Fields(first)) != 3 {
		t.Fatalf("confirmation = %q / %q, err=%v", first, second, err)
	}
	otherPublic, _, _ := ed25519.GenerateKey(rand.Reader)
	other, err := String(control, base64.StdEncoding.EncodeToString(otherPublic))
	if err != nil || other == first {
		t.Fatalf("different node key should change confirmation: %q / %q", first, other)
	}
}

// A fingerprint is the second value an operator compares, so it must be
// derived from the key rather than from anything the sender chose, and it must
// refuse a key it cannot decode instead of returning a plausible-looking
// string.
func TestFingerprintIsKeyDerivedAndFailsClosed(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))
	got := Fingerprint(key)
	if !strings.HasPrefix(got, "ed25519:") {
		t.Fatalf("Fingerprint() = %q, want an ed25519-labelled value", got)
	}
	if got != Fingerprint(key) {
		t.Error("Fingerprint() is not stable for the same key")
	}

	other := make([]byte, ed25519.PublicKeySize)
	other[0] = 1
	if Fingerprint(base64.StdEncoding.EncodeToString(other)) == got {
		t.Error("two different keys produced the same fingerprint")
	}

	for _, bad := range []string{"", "not-base64!!", base64.StdEncoding.EncodeToString([]byte("short"))} {
		if Fingerprint(bad) != "" {
			t.Errorf("Fingerprint(%q) returned a value for an undecodable key", bad)
		}
	}
}
