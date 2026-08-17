package sealing

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestEnvelopeRoundTripAndContextBinding(t *testing.T) {
	_, nodePrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	private, err := PrivateKeyFromEd25519Seed(nodePrivate.Seed())
	if err != nil {
		t.Fatal(err)
	}
	public, err := PublicKeyFromEd25519(nodePrivate.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	aad := Context("machine", "node", "mini.local", "all", "plan-hash", "operation", "operator")
	envelope, err := Seal(public, []byte("operator passphrase"), aad)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := Open(private, envelope, aad)
	if err != nil {
		t.Fatal(err)
	}
	if string(opened) != "operator passphrase" {
		t.Fatalf("opened plaintext = %q", opened)
	}
	if _, err := Open(private, envelope, Context("machine", "node", "other.local", "all", "plan-hash", "operation", "operator")); err == nil {
		t.Fatal("envelope accepted for a different target")
	}
}

func TestEnvelopeRejectsWrongRecipient(t *testing.T) {
	_, first, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	firstPrivate, err := PrivateKeyFromEd25519Seed(first.Seed())
	if err != nil {
		t.Fatal(err)
	}
	secondPublic, err := PublicKeyFromEd25519(second.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := Seal(secondPublic, []byte("secret"), []byte("aad"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(firstPrivate, envelope, []byte("aad")); err == nil {
		t.Fatal("wrong recipient opened the envelope")
	}
}
