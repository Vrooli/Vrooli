package permissions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func newTestAdapter(t *testing.T) *Adapter {
	t.Helper()
	dir := t.TempDir()
	return &Adapter{
		SettingsPath: filepath.Join(dir, "config.toml"),
		Scope:        ScopeUser,
	}
}

func TestLoadMissingReturnsEmpty(t *testing.T) {
	a := newTestAdapter(t)
	p, err := a.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(p.BashDeny)+len(p.BashAsk)+len(p.BashAllow) != 0 {
		t.Fatalf("expected empty policy, got %+v", p)
	}
}

func TestSaveAddsVrooliSection(t *testing.T) {
	a := newTestAdapter(t)
	p := Policy{BashDeny: []string{"git stash *"}, BashAsk: []string{"rm *"}, BashAllow: []string{"ls *"}}
	if err := a.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, err := os.ReadFile(a.SettingsPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var doc map[string]any
	if err := toml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse: %v\n%s", err, raw)
	}
	section := doc[vrooliSectionKey].(map[string]any)
	perms := section["permissions"].(map[string]any)
	if got := perms["bash_deny"].([]any); len(got) != 1 || got[0] != "git stash *" {
		t.Errorf("bash_deny wrong: %v", got)
	}
	if got := perms["bash_ask"].([]any); len(got) != 1 || got[0] != "rm *" {
		t.Errorf("bash_ask wrong: %v", got)
	}
	if got := perms["bash_allow"].([]any); len(got) != 1 || got[0] != "ls *" {
		t.Errorf("bash_allow wrong: %v", got)
	}
}

func TestSavePreservesUnknownTopLevelKeys(t *testing.T) {
	a := newTestAdapter(t)
	initial := `
model = "gpt-5"
approval_policy = "on-request"

[profiles.default]
sandbox_mode = "workspace-write"

[vrooli]
other_key = "stays"
`
	if err := os.WriteFile(a.SettingsPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := a.Save(Policy{BashDeny: []string{"git stash *"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, _ := os.ReadFile(a.SettingsPath)
	var doc map[string]any
	if err := toml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse: %v\n%s", err, raw)
	}
	if doc["model"] != "gpt-5" {
		t.Errorf("model not preserved: %v", doc["model"])
	}
	if doc["approval_policy"] != "on-request" {
		t.Errorf("approval_policy not preserved: %v", doc["approval_policy"])
	}
	profiles := doc["profiles"].(map[string]any)
	def := profiles["default"].(map[string]any)
	if def["sandbox_mode"] != "workspace-write" {
		t.Errorf("[profiles.default].sandbox_mode not preserved: %v", def)
	}
	section := doc[vrooliSectionKey].(map[string]any)
	if section["other_key"] != "stays" {
		t.Errorf("[vrooli].other_key not preserved: %v", section)
	}
}

func TestSaveClearsSectionWhenEmpty(t *testing.T) {
	a := newTestAdapter(t)
	if err := a.Save(Policy{BashDeny: []string{"x"}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := a.Save(Policy{}); err != nil {
		t.Fatalf("clear: %v", err)
	}
	raw, _ := os.ReadFile(a.SettingsPath)
	if strings.Contains(string(raw), "bash_deny") {
		t.Errorf("bash_deny should be cleared: %s", raw)
	}
}

func TestFingerprintStableAcrossOrder(t *testing.T) {
	a := Fingerprint(Policy{BashDeny: []string{"a", "b"}})
	b := Fingerprint(Policy{BashDeny: []string{"b", "a"}})
	if a != b {
		t.Errorf("fingerprint should be order-independent: %s vs %s", a, b)
	}
}

func TestRoundTripIdempotent(t *testing.T) {
	a := newTestAdapter(t)
	p := Policy{BashDeny: []string{"git stash *"}, BashAsk: []string{"rm *"}}
	if err := a.Save(p); err != nil {
		t.Fatalf("save1: %v", err)
	}
	first, _ := os.ReadFile(a.SettingsPath)
	loaded, err := a.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := a.Save(loaded); err != nil {
		t.Fatalf("save2: %v", err)
	}
	second, _ := os.ReadFile(a.SettingsPath)
	if string(first) != string(second) {
		t.Errorf("round-trip not byte-identical:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

func TestAdminScopeUsesRequirementsFile(t *testing.T) {
	dir := t.TempDir()
	a := &Adapter{
		SettingsPath: filepath.Join(dir, "requirements.toml"),
		Scope:        ScopeAdmin,
	}
	if err := a.Save(Policy{BashDeny: []string{"x"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !strings.HasSuffix(a.StatePath(), ".vrooli-permissions-state.admin.json") {
		t.Errorf("admin scope state path wrong: %s", a.StatePath())
	}
}

func TestStateRoundTrip(t *testing.T) {
	a := newTestAdapter(t)
	p := Policy{BashDeny: []string{"git stash *"}}
	if err := a.Save(p); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := a.WriteState(p, "test-0.1"); err != nil {
		t.Fatalf("write state: %v", err)
	}
	st, err := a.LoadState()
	if err != nil || st == nil {
		t.Fatalf("load state: %v, %v", st, err)
	}
	if st.Fingerprint != Fingerprint(p) {
		t.Errorf("fingerprint mismatch: %s vs %s", st.Fingerprint, Fingerprint(p))
	}
	if st.SchemaVersion != StateSchemaVersion {
		t.Errorf("schema version: %d", st.SchemaVersion)
	}
	if st.Scope != ScopeUser {
		t.Errorf("scope not recorded: %v", st.Scope)
	}
}
