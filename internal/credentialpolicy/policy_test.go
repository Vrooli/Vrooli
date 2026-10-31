package credentialpolicy

import (
	"bytes"
	"testing"
)

func TestSealOpenBindsVersionAndPurpose(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	envelope, err := Seal(key, []byte("fixture"), "recovery-bundle", 3)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Open(key, envelope)
	if err != nil || string(got) != "fixture" {
		t.Fatalf("Open() = %q, %v", got, err)
	}
	for _, altered := range []Envelope{
		{Version: 3, Purpose: "other", Nonce: envelope.Nonce, Ciphertext: envelope.Ciphertext},
		{Version: 4, Purpose: envelope.Purpose, Nonce: envelope.Nonce, Ciphertext: envelope.Ciphertext},
	} {
		if _, err := Open(key, altered); err == nil {
			t.Fatal("Open accepted an envelope with altered authenticated context")
		}
	}
}
