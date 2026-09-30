package support

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAdminEmail(t *testing.T) {
	t.Run("explicit wins", func(t *testing.T) {
		t.Setenv("ADMIN_DEFAULT_EMAIL", "env@example.test")
		if got := ResolveAdminEmail("flag@example.test"); got != "flag@example.test" {
			t.Fatalf("ResolveAdminEmail = %q, want explicit value", got)
		}
	})
	t.Run("environment fallback", func(t *testing.T) {
		t.Setenv("ADMIN_DEFAULT_EMAIL", "env@example.test")
		if got := ResolveAdminEmail(""); got != "env@example.test" {
			t.Fatalf("ResolveAdminEmail = %q, want environment value", got)
		}
	})
	t.Run("seeded default", func(t *testing.T) {
		t.Setenv("ADMIN_DEFAULT_EMAIL", "")
		if got := ResolveAdminEmail(""); got != DefaultAdminEmail {
			t.Fatalf("ResolveAdminEmail = %q, want %q", got, DefaultAdminEmail)
		}
	})
}

func TestResolveAdminPasswordExplicit(t *testing.T) {
	t.Run("literal", func(t *testing.T) {
		got, err := ResolveAdminPassword("hunter2")
		if err != nil {
			t.Fatalf("ResolveAdminPassword returned error: %v", err)
		}
		if got != "hunter2" {
			t.Fatalf("ResolveAdminPassword = %q, want literal", got)
		}
	})
	t.Run("file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "password.txt")
		if err := os.WriteFile(path, []byte("file-secret\n"), 0o600); err != nil {
			t.Fatalf("write password file: %v", err)
		}
		got, err := ResolveAdminPassword("@" + path)
		if err != nil {
			t.Fatalf("ResolveAdminPassword returned error: %v", err)
		}
		if got != "file-secret" {
			t.Fatalf("ResolveAdminPassword = %q, want trimmed file value", got)
		}
	})
}
