package sealing

import (
	"crypto/ecdh"
	"crypto/rand"
	"testing"
)

func TestEnvelopeRoundTripAndContextBinding(t *testing.T) {
	nodePrivate, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	private := nodePrivate
	public := nodePrivate.PublicKey().Bytes()
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
	first, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	firstPrivate := first
	secondPublic := second.PublicKey().Bytes()
	envelope, err := Seal(secondPublic, []byte("secret"), []byte("aad"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(firstPrivate, envelope, []byte("aad")); err == nil {
		t.Fatal("wrong recipient opened the envelope")
	}
}
