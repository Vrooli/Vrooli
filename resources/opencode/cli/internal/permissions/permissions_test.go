package permissions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestAdapter(t *testing.T) *Adapter {
	t.Helper()
	dir := t.TempDir()
	return &Adapter{SettingsPath: filepath.Join(dir, "opencode.json")}
}

func readDoc(t *testing.T, a *Adapter) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(a.SettingsPath)
	if err != nil {
		t.Fatalf("read opencode.json: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse opencode.json: %v\n%s", err, raw)
	}
	return doc
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

func TestSaveAddsBashEntriesAndSidecar(t *testing.T) {
	a := newTestAdapter(t)
	p := Policy{
		BashDeny:  []string{"git stash *", "rm -rf /*"},
		BashAsk:   []string{"sudo *"},
		BashAllow: []string{"ls *"},
	}
	if err := a.Save(p, "test"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	doc := readDoc(t, a)
	perm, ok := doc["permission"].(map[string]any)
	if !ok {
		t.Fatalf("missing permission object: %v", doc)
	}
	bash, ok := perm["bash"].(map[string]any)
	if !ok {
		t.Fatalf("missing permission.bash map: %v", perm)
	}
	if bash["git stash *"] != "deny" || bash["rm -rf /*"] != "deny" {
		t.Errorf("deny entries wrong: %v", bash)
	}
	if bash["sudo *"] != "ask" {
		t.Errorf("ask entry wrong: %v", bash)
	}
	if bash["ls *"] != "allow" {
		t.Errorf("allow entry wrong: %v", bash)
	}

	// The managed list lives in the sidecar, NOT inline in opencode.json.
	st, err := a.LoadState()
	if err != nil || st == nil {
		t.Fatalf("load state: %v, %v", st, err)
	}
	if len(st.ManagedBash) != 4 {
		t.Fatalf("sidecar should list 4 managed patterns, got %v", st.ManagedBash)
	}
}

// TestOpencodeJSONHasOnlySchemaKeys is the regression guard for the host
// block: opencode rejects unknown top-level keys, so a permissions write must
// never inject a Vrooli-private key (e.g. x-vrooli-managed-permissions) into
// opencode.json.
func TestOpencodeJSONHasOnlySchemaKeys(t *testing.T) {
	a := newTestAdapter(t)
	if err := a.Save(Policy{BashDeny: []string{"git stash *"}, BashAsk: []string{"sudo *"}}, "test"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	doc := readDoc(t, a)
	for k := range doc {
		if strings.HasPrefix(k, "x-vrooli") {
			t.Errorf("opencode.json must not carry Vrooli-private top-level key %q (opencode rejects unknown keys)", k)
		}
	}
	// Sanity: the permission object is still present.
	if _, ok := doc["permission"]; !ok {
		t.Errorf("expected permission object in %v", doc)
	}
}

// TestLegacyManagedKeyMigratedAndStripped covers the migration path: an old
// opencode.json that still carries the retired inline managed-list key must
// (a) have those patterns treated as managed on Load, and (b) drop the key
// from opencode.json on the next Save while recording the list in the sidecar.
func TestLegacyManagedKeyMigratedAndStripped(t *testing.T) {
	a := newTestAdapter(t)
	initial := `{
  "model": "anthropic/claude-opus-4-7",
  "permission": {"bash": {"git stash *": "deny", "echo *": "allow"}},
  "x-vrooli-managed-permissions": ["git stash *"]
}`
	if err := os.WriteFile(a.SettingsPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Load migrates: the legacy-keyed pattern is reported as managed; the
	// hand-written "echo *" is not.
	p, err := a.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(p.BashDeny) != 1 || p.BashDeny[0] != "git stash *" {
		t.Fatalf("legacy managed pattern not surfaced: %+v", p)
	}
	if len(p.BashAllow) != 0 {
		t.Errorf("hand-written 'echo *' must not be reported as managed: %+v", p)
	}

	// Save strips the legacy key and records the list in the sidecar.
	if err := a.Save(p, "test"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	doc := readDoc(t, a)
	if _, ok := doc["x-vrooli-managed-permissions"]; ok {
		t.Errorf("legacy key must be stripped from opencode.json: %v", doc)
	}
	bash := doc["permission"].(map[string]any)["bash"].(map[string]any)
	if bash["echo *"] != "allow" {
		t.Errorf("hand-written entry lost during migration: %v", bash)
	}
	if bash["git stash *"] != "deny" {
		t.Errorf("managed entry lost during migration: %v", bash)
	}
	st, err := a.LoadState()
	if err != nil || st == nil || len(st.ManagedBash) != 1 || st.ManagedBash[0] != "git stash *" {
		t.Errorf("sidecar should record the migrated managed list: %+v (%v)", st, err)
	}
}

func TestSavePreservesUnknownTopLevelKeys(t *testing.T) {
	a := newTestAdapter(t)
	initial := `{
  "model": "claude-opus-4-7",
  "permission": {"edit": "allow", "bash": {"echo *": "allow"}}
}`
	if err := os.WriteFile(a.SettingsPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := a.Save(Policy{BashDeny: []string{"git stash *"}}, "test"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	doc := readDoc(t, a)
	if doc["model"] != "claude-opus-4-7" {
		t.Errorf("model not preserved: %v", doc["model"])
	}
	perm := doc["permission"].(map[string]any)
	if perm["edit"] != "allow" {
		t.Errorf("permission.edit not preserved: %v", perm)
	}
	bash := perm["bash"].(map[string]any)
	if bash["echo *"] != "allow" {
		t.Errorf("hand-written bash entry lost: %v", bash)
	}
	if bash["git stash *"] != "deny" {
		t.Errorf("managed entry missing: %v", bash)
	}
}

func TestSavePreservesHandWrittenBashEntries(t *testing.T) {
	a := newTestAdapter(t)
	// User wrote this pattern by hand — it must survive subsequent
	// Vrooli-managed writes that don't mention it.
	initial := `{"permission": {"bash": {"sudo apt *": "deny"}}}`
	if err := os.WriteFile(a.SettingsPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := a.Save(Policy{BashDeny: []string{"git stash *"}}, "test"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	doc := readDoc(t, a)
	bash := doc["permission"].(map[string]any)["bash"].(map[string]any)
	if bash["sudo apt *"] != "deny" {
		t.Errorf("hand-written deny lost: %v", bash)
	}
	if bash["git stash *"] != "deny" {
		t.Errorf("managed deny missing: %v", bash)
	}
}

func TestSaveOverwritesPriorManagedEntries(t *testing.T) {
	a := newTestAdapter(t)
	if err := a.Save(Policy{BashDeny: []string{"old"}}, "test"); err != nil {
		t.Fatalf("save1: %v", err)
	}
	if err := a.Save(Policy{BashDeny: []string{"new"}}, "test"); err != nil {
		t.Fatalf("save2: %v", err)
	}
	raw, _ := os.ReadFile(a.SettingsPath)
	if strings.Contains(string(raw), `"old"`) {
		t.Errorf("old managed entry not removed: %s", raw)
	}
	if !strings.Contains(string(raw), `"new"`) {
		t.Errorf("new managed entry missing: %s", raw)
	}
}

func TestSaveClearsBashWhenEmpty(t *testing.T) {
	a := newTestAdapter(t)
	if err := a.Save(Policy{BashDeny: []string{"x"}}, "test"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := a.Save(Policy{}, "test"); err != nil {
		t.Fatalf("clear: %v", err)
	}
	// No managed entry should remain in either file.
	raw, _ := os.ReadFile(a.SettingsPath)
	if strings.Contains(string(raw), `"x"`) {
		t.Errorf("managed entry should be gone from opencode.json: %s", raw)
	}
	st, err := a.LoadState()
	if err != nil || st == nil {
		t.Fatalf("load state: %v, %v", st, err)
	}
	if len(st.ManagedBash) != 0 {
		t.Errorf("sidecar managed list should be empty after clear: %v", st.ManagedBash)
	}
}

func TestSaveClearsBashKeepsHandWritten(t *testing.T) {
	a := newTestAdapter(t)
	// Seed with hand-written, then write Vrooli-managed, then clear.
	if err := os.WriteFile(a.SettingsPath, []byte(`{"permission": {"bash": {"echo *": "allow"}}}`), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := a.Save(Policy{BashDeny: []string{"git stash *"}}, "test"); err != nil {
		t.Fatalf("save managed: %v", err)
	}
	if err := a.Save(Policy{}, "test"); err != nil {
		t.Fatalf("clear: %v", err)
	}
	raw, _ := os.ReadFile(a.SettingsPath)
	if !strings.Contains(string(raw), `"echo *"`) {
		t.Errorf("hand-written entry should still be present after clear: %s", raw)
	}
	if strings.Contains(string(raw), `"git stash *"`) {
		t.Errorf("managed entry should be cleared: %s", raw)
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
	if err := a.Save(p, "test"); err != nil {
		t.Fatalf("save1: %v", err)
	}
	first, _ := os.ReadFile(a.SettingsPath)
	loaded, err := a.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := a.Save(loaded, "test"); err != nil {
		t.Fatalf("save2: %v", err)
	}
	second, _ := os.ReadFile(a.SettingsPath)
	if string(first) != string(second) {
		t.Errorf("round-trip not byte-identical:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

func TestLoadIgnoresHandWrittenForPolicy(t *testing.T) {
	a := newTestAdapter(t)
	// No sidecar → all bash entries are "hand-written" from the
	// adapter's point of view; Load should report empty Policy but
	// the file is left alone.
	if err := os.WriteFile(a.SettingsPath, []byte(`{"permission": {"bash": {"echo *": "allow"}}}`), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	p, err := a.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(p.BashAllow)+len(p.BashAsk)+len(p.BashDeny) != 0 {
		t.Errorf("hand-written entries should not be reported as managed: %+v", p)
	}
}

func TestStateRoundTrip(t *testing.T) {
	a := newTestAdapter(t)
	p := Policy{BashDeny: []string{"git stash *"}}
	if err := a.Save(p, "test-0.1"); err != nil {
		t.Fatalf("save: %v", err)
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
	if st.WrittenByVer != "test-0.1" {
		t.Errorf("writtenByVersion not recorded: %q", st.WrittenByVer)
	}
	if len(st.ManagedBash) != 1 || st.ManagedBash[0] != "git stash *" {
		t.Errorf("managed list not recorded in sidecar: %v", st.ManagedBash)
	}
}

func TestSaveProjectsConfiguredToolBeforePlugin(t *testing.T) {
	dir := t.TempDir()
	a := &Adapter{SettingsPath: filepath.Join(dir, "opencode.json"), PluginPath: filepath.Join(dir, "plugins", "vrooli-policy.js")}
	t.Setenv("VROOLI_AGENT_POLICY_RUNNER", "policy-runner")
	if err := a.Save(Policy{BashDeny: []string{"pnpm add *"}}, "test"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(a.PluginPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "tool.execute.before") || !strings.Contains(string(raw), "policy-runner") {
		t.Fatalf("unexpected plugin projection: %s", raw)
	}
}
