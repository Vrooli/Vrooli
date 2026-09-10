package secrets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestKeyUsable(t *testing.T) {
	good := "sk-or-" + makeString(40)
	cases := map[string]bool{
		good:                        true,
		"sk-or-short":               false,
		"sk-proj-" + makeString(40): false,
		"auto-null-123":             false,
		"":                          false,
		"[ERROR] nope":              false,
	}
	for k, want := range cases {
		if got := KeyUsable(k); got != want {
			t.Errorf("KeyUsable(%q)=%v want %v", k, got, want)
		}
	}
}

func TestResolveOpenRouterKey_EnvFirst(t *testing.T) {
	key := "sk-or-" + makeString(40)
	got := ResolveOpenRouterKey(Options{
		Getenv: func(k string) string {
			if k == "OPENROUTER_API_KEY" {
				return key
			}
			return ""
		},
	})
	if got != key {
		t.Errorf("env key not used: %q", got)
	}
}

func TestResolveOpenRouterKeyReturnsEmptyWithoutInjection(t *testing.T) {
	if got := ResolveOpenRouterKey(Options{Getenv: func(string) string { return "" }}); got != "" {
		t.Errorf("unconfigured key = %q", got)
	}
}

func TestSyncOpenRouterAuthPreservesProvidersAndWritesProtectedAuth(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opencode", "auth.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"anthropic":{"type":"api","key":"keep-me"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	changed, err := SyncOpenRouterAuth(path, "sk-or-abcdefghijklmnopqrstuvwxyz1234567890")
	if err != nil || !changed {
		t.Fatalf("SyncOpenRouterAuth() = %v, %v", changed, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("auth mode = %o, want 600", info.Mode().Perm())
	}
	var auth map[string]map[string]string
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, &auth); err != nil {
		t.Fatal(err)
	}
	if auth["anthropic"]["key"] != "keep-me" {
		t.Fatalf("unrelated provider was not preserved: %#v", auth)
	}
	if auth["openrouter"]["key"] == "" || auth["openrouter"]["type"] != "api" {
		t.Fatalf("OpenRouter auth was not written: %#v", auth["openrouter"])
	}

	changed, err = SyncOpenRouterAuth(path, "sk-or-abcdefghijklmnopqrstuvwxyz1234567890")
	if err != nil || changed {
		t.Fatalf("idempotent SyncOpenRouterAuth() = %v, %v", changed, err)
	}
}

func makeString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}
