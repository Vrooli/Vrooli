package permissions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testAdapter(t *testing.T) *Adapter {
	t.Helper()
	dir := t.TempDir()
	return &Adapter{
		SettingsPath: filepath.Join(dir, "settings.json"),
		Scope:        ScopeUser,
	}
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	a := testAdapter(t)
	p, err := a.Load()
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if len(p.BashDeny)+len(p.BashAsk)+len(p.BashAllow) != 0 {
		t.Errorf("expected empty policy, got %+v", p)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	a := testAdapter(t)
	in := Policy{
		BashDeny:  []string{"command(rm -rf)", "command(git push --force)"},
		BashAllow: []string{"command(git)"},
		BashAsk:   []string{"mcp(*)"},
	}
	if err := a.Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := a.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if strings.Join(got.BashDeny, ",") != "command(git push --force),command(rm -rf)" {
		t.Errorf("deny round-trip (want sorted): %+v", got.BashDeny)
	}
	if len(got.BashAllow) != 1 || got.BashAllow[0] != "command(git)" {
		t.Errorf("allow round-trip: %+v", got.BashAllow)
	}
	if len(got.BashAsk) != 1 || got.BashAsk[0] != "mcp(*)" {
		t.Errorf("ask round-trip: %+v", got.BashAsk)
	}
}

func TestSaveWritesNativePermissionsObject(t *testing.T) {
	a := testAdapter(t)
	if err := a.Save(Policy{BashDeny: []string{"command(rm -rf)"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(a.SettingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse settings: %v", err)
	}
	section, ok := doc["permissions"].(map[string]any)
	if !ok {
		t.Fatalf("expected native `permissions` object, got: %s", data)
	}
	if _, ok := section["deny"].([]any); !ok {
		t.Errorf("expected permissions.deny array, got: %s", data)
	}
}

func TestSavePreservesUnrelatedSettings(t *testing.T) {
	a := testAdapter(t)
	// Seed a realistic antigravity-cli settings.json including other top-level
	// keys and an unrelated key inside the `permissions` object.
	seed := `{
  "model": "gemini-3.5-flash",
  "toolPermission": "request-review",
  "trustedWorkspaces": ["/home/me/proj"],
  "permissions": {
    "allowNonWorkspaceAccess": false
  }
}`
	if err := os.WriteFile(a.SettingsPath, []byte(seed), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := a.Save(Policy{BashDeny: []string{"command(rm -rf)"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, _ := os.ReadFile(a.SettingsPath)
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if doc["model"] != "gemini-3.5-flash" {
		t.Errorf("model not preserved: %s", data)
	}
	if doc["toolPermission"] != "request-review" {
		t.Errorf("toolPermission not preserved: %s", data)
	}
	if tw, _ := doc["trustedWorkspaces"].([]any); len(tw) != 1 {
		t.Errorf("trustedWorkspaces not preserved: %s", data)
	}
	perm, _ := doc["permissions"].(map[string]any)
	if perm["allowNonWorkspaceAccess"] != false {
		t.Errorf("permissions.allowNonWorkspaceAccess clobbered: %s", data)
	}
	if _, ok := perm["deny"]; !ok {
		t.Errorf("permissions.deny missing: %s", data)
	}
}

func TestSaveRemovesEmptyPermissionsObject(t *testing.T) {
	a := testAdapter(t)
	if err := a.Save(Policy{BashDeny: []string{"command(rm -rf)"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Clearing all managed arrays drops the (otherwise empty) permissions object.
	if err := a.Save(Policy{}); err != nil {
		t.Fatalf("Save clear: %v", err)
	}
	data, _ := os.ReadFile(a.SettingsPath)
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, ok := doc["permissions"]; ok {
		t.Errorf("expected empty permissions object removed: %s", data)
	}
}

func TestFingerprintStableAndOrderInsensitive(t *testing.T) {
	a := Policy{BashDeny: []string{"Bash(b)", "Bash(a)"}}
	b := Policy{BashDeny: []string{"Bash(a)", "Bash(b)"}}
	if Fingerprint(a) != Fingerprint(b) {
		t.Error("fingerprint should be order-insensitive")
	}
	c := Policy{BashDeny: []string{"Bash(a)"}}
	if Fingerprint(a) == Fingerprint(c) {
		t.Error("fingerprint should change with content")
	}
}

func TestDefaultAdapterRejectsNonUserScope(t *testing.T) {
	if _, err := DefaultAdapter(Scope("project")); err == nil {
		t.Error("expected DefaultAdapter to reject project scope")
	}
	a, err := DefaultAdapter(ScopeUser)
	if err != nil {
		t.Fatalf("DefaultAdapter(user): %v", err)
	}
	if !strings.HasSuffix(a.SettingsPath, filepath.Join(".gemini", "antigravity-cli", "settings.json")) {
		t.Errorf("unexpected settings path: %s", a.SettingsPath)
	}
}
