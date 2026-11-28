package operatorsession

import (
	"context"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLocalOwnerTokenProvider(t *testing.T) {
	t.Setenv(operatorSessionDirEnv, filepath.Join(t.TempDir(), "missing"))
	if token, err := LocalOwnerTokenProvider()(context.Background()); err != nil || token != "" {
		t.Fatalf("missing store = (%q, %v), want empty token and nil error", token, err)
	}

	dir := t.TempDir()
	t.Setenv(operatorSessionDirEnv, dir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if token, err := LocalOwnerTokenProvider()(context.Background()); err != nil || token != "" {
		t.Fatalf("unenrolled store = (%q, %v), want empty token and nil error", token, err)
	}

	private, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	enrollment := Enrollment{OperatorID: "operator-1", IdentityProvider: IdentityProviderAuthenticator, Mode: ModePersonal, Reference: "enrollment-1", EnrolledAt: time.Now(), ScopeCeiling: []string{"read"}}
	if err := (&FileStore{Dir: dir}).Save(ed25519.PrivateKey(private), enrollment); err != nil {
		t.Fatal(err)
	}
	token, err := LocalOwnerTokenProvider()(context.Background())
	if err != nil || !strings.HasPrefix(token, LocalSessionScheme+" ") {
		t.Fatalf("enrolled store = (%q, %v), want LocalSession credential", token, err)
	}
}
