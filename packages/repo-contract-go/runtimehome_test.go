package repocontract

import (
	"path/filepath"
	"testing"
)

func TestRuntimeHomeDirName(t *testing.T) {
	c := validContract(t)
	if got := c.RuntimeHomeDirName(); got != ".vrooli" {
		t.Fatalf("RuntimeHomeDirName() = %q, want %q", got, ".vrooli")
	}
	if overrides := c.RuntimeHomeEnvOverrides(); len(overrides) != 0 {
		t.Fatalf("RuntimeHomeEnvOverrides() = %v, want empty", overrides)
	}
}

func TestRuntimeHomeRoot(t *testing.T) {
	c := validContract(t)
	home := filepath.Join(string(filepath.Separator), "tmp", "vrooli-home")
	root, err := c.RuntimeHome(home)
	if err != nil {
		t.Fatalf("RuntimeHome: %v", err)
	}
	if want := filepath.Join(home, ".vrooli"); root != want {
		t.Fatalf("RuntimeHome = %q, want %q", root, want)
	}
	if _, err := c.RuntimeHome(""); err == nil {
		t.Fatal("RuntimeHome(\"\") expected error")
	}
}

func TestRuntimeHomeEntry(t *testing.T) {
	c := validContract(t)
	home := filepath.Join(string(filepath.Separator), "tmp", "vrooli-home")

	cases := []struct {
		key         string
		wantRel     string
		kind        string
		regenerable bool
		format      string
		sensitive   bool
	}{
		{"secrets", "secrets.json", "file", false, "json", true},
		{"runtime_db", "state/runtime.db", "file", false, "sqlite", false},
		{"data", "data", "dir", false, "", false},
		{"logs", "logs", "dir", true, "", false},
	}
	for _, tc := range cases {
		entry, err := c.RuntimeHomeEntry(home, tc.key)
		if err != nil {
			t.Fatalf("RuntimeHomeEntry(%q): %v", tc.key, err)
		}
		if want := filepath.Join(home, ".vrooli", filepath.FromSlash(tc.wantRel)); entry.AbsPath != want {
			t.Errorf("%s AbsPath = %q, want %q", tc.key, entry.AbsPath, want)
		}
		if entry.Kind != tc.kind {
			t.Errorf("%s Kind = %q, want %q", tc.key, entry.Kind, tc.kind)
		}
		if entry.Regenerable != tc.regenerable {
			t.Errorf("%s Regenerable = %v, want %v", tc.key, entry.Regenerable, tc.regenerable)
		}
		if entry.Format != tc.format {
			t.Errorf("%s Format = %q, want %q", tc.key, entry.Format, tc.format)
		}
		if entry.Sensitive != tc.sensitive {
			t.Errorf("%s Sensitive = %v, want %v", tc.key, entry.Sensitive, tc.sensitive)
		}
	}

	if _, err := c.RuntimeHomeEntry(home, "does-not-exist"); err == nil {
		t.Fatal("RuntimeHomeEntry(unknown) expected error")
	}
}

func TestRuntimeHomeEntriesSortedAndComplete(t *testing.T) {
	c := validContract(t)
	home := filepath.Join(string(filepath.Separator), "tmp", "vrooli-home")
	entries, err := c.RuntimeHomeEntries(home)
	if err != nil {
		t.Fatalf("RuntimeHomeEntries: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("RuntimeHomeEntries returned nothing")
	}
	// Sorted by key.
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Key > entries[i].Key {
			t.Fatalf("entries not sorted: %q before %q", entries[i-1].Key, entries[i].Key)
		}
	}
	// Durable vs regenerable split is queryable.
	durable := map[string]bool{}
	for _, e := range entries {
		durable[e.Key] = !e.Regenerable
	}
	if !durable["data"] {
		t.Error("data must be durable (regenerable=false)")
	}
	if durable["logs"] || durable["cache"] || durable["bin"] {
		t.Error("logs/cache/bin must be regenerable")
	}
}

func TestScopedRuntimePath(t *testing.T) {
	c := validContract(t)
	home := filepath.Join(string(filepath.Separator), "tmp", "vrooli-home")

	secrets, err := c.ScopedRuntimePath(home, "scenario_secrets", map[string]string{"scenario": "data-backup-manager"})
	if err != nil {
		t.Fatalf("ScopedRuntimePath scenario_secrets: %v", err)
	}
	if want := filepath.Join(home, ".vrooli", "scenarios", "data-backup-manager", "secrets.json"); secrets != want {
		t.Fatalf("scenario_secrets = %q, want %q", secrets, want)
	}

	state, err := c.ScopedRuntimePath(home, "project_state", map[string]string{"project_key": "abc123"})
	if err != nil {
		t.Fatalf("ScopedRuntimePath project_state: %v", err)
	}
	if want := filepath.Join(home, ".vrooli", "state", "projects", "abc123"); state != want {
		t.Fatalf("project_state = %q, want %q", state, want)
	}

	if _, err := c.ScopedRuntimePath(home, "nope", nil); err == nil {
		t.Fatal("ScopedRuntimePath(unknown key) expected error")
	}
	if _, err := c.ScopedRuntimePath(home, "scenario_secrets", map[string]string{"scenario": "../escape"}); err == nil {
		t.Fatal("ScopedRuntimePath with traversal param expected error")
	}
	if _, err := c.ScopedRuntimePath(home, "scenario_secrets", nil); err == nil {
		t.Fatal("ScopedRuntimePath with unresolved placeholder expected error")
	}
}

// TestUserSecretPathsMatchRuntimeHome pins that the userpaths.go helpers resolve
// to exactly the runtime_home-derived entries (single source of truth).
func TestUserSecretPathsMatchRuntimeHome(t *testing.T) {
	c := validContract(t)
	home := filepath.Join(string(filepath.Separator), "tmp", "vrooli-home")

	plaintext, err := UserPlaintextSecretsPath(home)
	if err != nil {
		t.Fatalf("UserPlaintextSecretsPath: %v", err)
	}
	wantPlain, _ := c.RuntimeHomeEntry(home, "secrets")
	if plaintext != wantPlain.AbsPath {
		t.Errorf("UserPlaintextSecretsPath = %q, want %q", plaintext, wantPlain.AbsPath)
	}

	encrypted, err := UserEncryptedSecretsPath(home)
	if err != nil {
		t.Fatalf("UserEncryptedSecretsPath: %v", err)
	}
	wantEnc, _ := c.RuntimeHomeEntry(home, "secrets_enc")
	if encrypted != wantEnc.AbsPath {
		t.Errorf("UserEncryptedSecretsPath = %q, want %q", encrypted, wantEnc.AbsPath)
	}
}
