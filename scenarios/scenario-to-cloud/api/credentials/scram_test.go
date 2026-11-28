package credentials

import (
	"strings"
	"testing"
)

// [REQ:STC-P0-032] The database role is prepared with a SCRAM-SHA-256
// verifier: deterministic for a given salt, PostgreSQL's format, and never
// containing the plaintext.
func TestScramSHA256VerifierMatchesPostgresFormat(t *testing.T) {
	salt := []byte("0123456789abcdef")
	verifier, err := ScramSHA256Verifier("canary-db-password", salt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(verifier, "SCRAM-SHA-256$4096:MDEyMzQ1Njc4OWFiY2RlZg==$") {
		t.Fatalf("verifier = %s", verifier)
	}
	again, _ := ScramSHA256Verifier("canary-db-password", salt)
	if again != verifier {
		t.Fatal("verifier is not deterministic for a fixed salt")
	}
	if strings.Contains(verifier, "canary-db-password") {
		t.Fatal("verifier contains the plaintext")
	}
	random1, _ := ScramSHA256Verifier("canary-db-password", nil)
	random2, _ := ScramSHA256Verifier("canary-db-password", nil)
	if random1 == random2 {
		t.Fatal("random salts collided")
	}
	if _, err := ScramSHA256Verifier("", salt); err == nil {
		t.Fatal("empty password accepted")
	}
}
