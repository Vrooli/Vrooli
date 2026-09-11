package interactive

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureClaudeProjectTrustedAddsOnlyRequestedProject(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, ".claude.json")
	original := []byte(`{"projects":{"/trusted":{"hasTrustDialogAccepted":true}},"lastVersion":"test"}`)
	if err := os.WriteFile(configPath, original, 0o640); err != nil {
		t.Fatal(err)
	}

	if err := ensureClaudeProjectTrusted(home, "/sandbox/merged"); err != nil {
		t.Fatalf("ensureClaudeProjectTrusted: %v", err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	projects := got["projects"].(map[string]any)
	if projects["/trusted"].(map[string]any)["hasTrustDialogAccepted"] != true {
		t.Fatal("existing trusted project was not preserved")
	}
	if projects["/sandbox/merged"].(map[string]any)["hasTrustDialogAccepted"] != true {
		t.Fatal("requested project was not trusted")
	}
	if got["lastVersion"] != "test" {
		t.Fatal("unrelated config content was not preserved")
	}
	if mode := func() os.FileMode { info, _ := os.Stat(configPath); return info.Mode().Perm() }(); mode != 0o640 {
		t.Fatalf("config mode = %o, want 640", mode)
	}
}

func TestEnsureClaudeProjectTrustedIsIdempotent(t *testing.T) {
	home := t.TempDir()
	if err := ensureClaudeProjectTrusted(home, "/sandbox/merged"); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureClaudeProjectTrusted(home, "/sandbox/merged"); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("second trust bootstrap changed an already trusted config")
	}
}
