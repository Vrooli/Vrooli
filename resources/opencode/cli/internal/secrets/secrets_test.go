package secrets

import (
	"context"
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

func TestParseExportedKey(t *testing.T) {
	out := "export FOO=bar\nexport OPENROUTER_API_KEY=\"sk-or-xyz\"\nBAZ=qux\n"
	if got := parseExportedKey(out, "OPENROUTER_API_KEY"); got != "sk-or-xyz" {
		t.Errorf("got %q", got)
	}
	if got := parseExportedKey(out, "MISSING"); got != "" {
		t.Errorf("missing key should be empty, got %q", got)
	}
}

func TestResolveOpenRouterKey_EnvFirst(t *testing.T) {
	key := "sk-or-" + makeString(40)
	got := ResolveOpenRouterKey(context.Background(), Options{
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

func TestResolveOpenRouterKey_VaultExport(t *testing.T) {
	key := "sk-or-" + makeString(40)
	got := ResolveOpenRouterKey(context.Background(), Options{
		Getenv:    func(string) string { return "" },
		LookVault: func(string) (string, error) { return "/fake/resource-vault", nil },
		RunVault: func(ctx context.Context, bin string, args ...string) ([]byte, error) {
			return []byte("export OPENROUTER_API_KEY=" + key + "\n"), nil
		},
	})
	if got != key {
		t.Errorf("vault key not resolved: %q", got)
	}
}

func TestResolveOpenRouterKey_CredentialsFile(t *testing.T) {
	dir := t.TempDir()
	cred := filepath.Join(dir, "cred.json")
	key := "sk-or-" + makeString(40)
	body, _ := json.Marshal(map[string]any{"data": map[string]string{"apiKey": key}})
	if err := os.WriteFile(cred, body, 0o600); err != nil {
		t.Fatal(err)
	}
	got := ResolveOpenRouterKey(context.Background(), Options{
		Getenv:    func(string) string { return "" },
		CredFile:  cred,
		LookVault: func(string) (string, error) { return "", os.ErrNotExist }, // no vault → fall through
	})
	if got != key {
		t.Errorf("credentials-file key not resolved: %q", got)
	}
}

func TestSyncAuth_MergesProviders(t *testing.T) {
	dir := t.TempDir()
	auth := filepath.Join(dir, "auth.json")
	// Pre-existing provider must be preserved.
	if err := os.WriteFile(auth, []byte(`{"anthropic":{"type":"api","key":"sk-ant-x"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	key := "sk-or-" + makeString(40)
	if err := SyncAuth(auth, key); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(auth)
	var m map[string]map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m["openrouter"]["key"] != key {
		t.Errorf("openrouter key not written: %v", m["openrouter"])
	}
	if m["anthropic"]["key"] != "sk-ant-x" {
		t.Errorf("existing provider clobbered: %v", m["anthropic"])
	}
}

func TestSyncAuth_NoopOnUnusableKey(t *testing.T) {
	dir := t.TempDir()
	auth := filepath.Join(dir, "auth.json")
	if err := SyncAuth(auth, "not-a-key"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(auth); !os.IsNotExist(err) {
		t.Errorf("auth file should not be created for an unusable key")
	}
}

func makeString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}
