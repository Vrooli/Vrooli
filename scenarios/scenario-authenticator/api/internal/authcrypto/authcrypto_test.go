package authcrypto

import (
	"crypto/rand"
	"crypto/rsa"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testAud = "scenario-authenticator:default"

func newTestSigner(t *testing.T, dir string) *Signer {
	t.Helper()
	keys, err := LoadOrGenerate(dir)
	if err != nil {
		t.Fatalf("LoadOrGenerate: %v", err)
	}
	return NewSigner(keys, SignerConfig{Issuer: "scenario-authenticator", Expiry: time.Hour})
}

// TestKeyPersistenceRoundTrip proves a key written by one process load is
// loaded — not regenerated — by the next, so a token signed before "restart"
// still verifies after. This is the core anti-rotation guarantee.
func TestKeyPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()

	s1 := newTestSigner(t, dir)
	tok, err := s1.Sign(TokenInput{UserID: "u1", Email: "a@b.co", Roles: []string{"user"}, Audience: testAud})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	// Simulate a process restart: a brand-new Keys load from the same dir.
	s2 := newTestSigner(t, dir)
	if s1.Keys().KID() != s2.Keys().KID() {
		t.Fatalf("kid changed across reload: %q vs %q", s1.Keys().KID(), s2.Keys().KID())
	}
	if _, err := s2.Validate(tok, testAud); err != nil {
		t.Fatalf("token signed before reload failed to validate after reload: %v", err)
	}
}

// TestKIDInHeader asserts the §8 correction: the JWT header carries the same
// kid JWKS publishes.
func TestKIDInHeader(t *testing.T) {
	dir := t.TempDir()
	s := newTestSigner(t, dir)
	tok, err := s.Sign(TokenInput{UserID: "u1", Audience: testAud})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	parsed, _, err := jwt.NewParser().ParseUnverified(tok, &Claims{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := parsed.Header["kid"]; got != s.Keys().KID() {
		t.Fatalf("kid header = %v, want %v", got, s.Keys().KID())
	}
	if got := s.Keys().JWKS().Keys[0].Kid; got != s.Keys().KID() {
		t.Fatalf("jwks kid = %v, want %v", got, s.Keys().KID())
	}
}

// TestRS256MethodLockRejectsNoneAndHS is the algorithm-confusion defense: a
// `none` or HS256 token must never validate against the RSA verifier.
func TestRS256MethodLockRejectsNoneAndHS(t *testing.T) {
	dir := t.TempDir()
	s := newTestSigner(t, dir)

	claims := &Claims{UserID: "evil", RegisteredClaims: jwt.RegisteredClaims{
		Issuer: "scenario-authenticator", Audience: jwt.ClaimStrings{testAud},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}

	noneTok, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("mint none token: %v", err)
	}
	if _, err := s.Validate(noneTok, testAud); err == nil {
		t.Fatal("none-signed token was accepted — algorithm-confusion hole")
	}

	hsTok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("mint hs token: %v", err)
	}
	if _, err := s.Validate(hsTok, testAud); err == nil {
		t.Fatal("HS256-signed token was accepted — algorithm-confusion hole")
	}
}

// TestAudienceMismatchRejected covers OT-P0-008: a token minted for a different
// realm/aud is rejected even though only one realm exists.
func TestAudienceMismatchRejected(t *testing.T) {
	dir := t.TempDir()
	s := newTestSigner(t, dir)
	tok, err := s.Sign(TokenInput{UserID: "u1", Audience: "scenario-authenticator:other-realm"})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := s.Validate(tok, testAud); err == nil {
		t.Fatal("cross-aud token accepted — cross-tenant leak")
	}
	// Same token validates against its own audience.
	if _, err := s.Validate(tok, "scenario-authenticator:other-realm"); err != nil {
		t.Fatalf("token rejected against its own audience: %v", err)
	}
}

// TestExpiredTokenRejected covers exp enforcement.
func TestExpiredTokenRejected(t *testing.T) {
	dir := t.TempDir()
	keys, err := LoadOrGenerate(dir)
	if err != nil {
		t.Fatalf("LoadOrGenerate: %v", err)
	}
	past := time.Now().Add(-2 * time.Hour)
	s := NewSigner(keys, SignerConfig{Issuer: "scenario-authenticator", Expiry: time.Hour, Now: func() time.Time { return past }})
	tok, err := s.Sign(TokenInput{UserID: "u1", Audience: testAud})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	// Validate with the real clock — token is long expired.
	live := NewSigner(keys, SignerConfig{Issuer: "scenario-authenticator", Expiry: time.Hour})
	if _, err := live.Validate(tok, testAud); err == nil {
		t.Fatal("expired token accepted")
	}
}

// TestFatalOnWriteFailure asserts the §8 correction: an unwritable key dir is a
// boot-fatal error, not a silent regenerate.
func TestFatalOnWriteFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission semantics")
	}
	// A path under a regular file can never be created as a directory.
	parent := t.TempDir()
	notADir := filepath.Join(parent, "file")
	if err := os.WriteFile(notADir, []byte("x"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := LoadOrGenerate(filepath.Join(notADir, "keys")); err == nil {
		t.Fatal("expected fatal error when key dir is unwritable, got nil")
	}
}

func mustKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	return priv, &priv.PublicKey
}

// TestNewKeysFromPair exercises the in-memory constructor used by other
// packages' tests.
func TestNewKeysFromPair(t *testing.T) {
	priv, pub := mustKeyPair(t)
	keys := NewKeysFromPair(priv, pub)
	if keys.KID() == "" {
		t.Fatal("empty kid")
	}
	s := NewSigner(keys, SignerConfig{Issuer: "scenario-authenticator", Expiry: time.Hour})
	tok, err := s.Sign(TokenInput{UserID: "u1", Audience: testAud})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := s.Validate(tok, testAud); err != nil {
		t.Fatalf("validate: %v", err)
	}
}
