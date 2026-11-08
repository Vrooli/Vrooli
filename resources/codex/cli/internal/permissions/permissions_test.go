package permissions

import (
	"encoding/json"
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

func readIntent(t *testing.T, a *Adapter) Intent {
	t.Helper()
	raw, err := os.ReadFile(a.IntentPath())
	if err != nil {
		t.Fatalf("read intent: %v", err)
	}
	var intent Intent
	if err := json.Unmarshal(raw, &intent); err != nil {
		t.Fatalf("parse intent: %v\n%s", err, raw)
	}
	return intent
}

func TestLoadMissingReturnsEmpty(t *testing.T) {
	a := newTestAdapter(t)
	p, err := a.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(p.BashDeny)+len(p.BashAsk)+len(p.BashAllow) != 0 || p.LegacyTable {
		t.Fatalf("expected empty policy, got %+v", p)
	}
}

func TestSaveRecordsRulesInTheSidecarAndNeverInConfig(t *testing.T) {
	a := newTestAdapter(t)
	p := Policy{BashDeny: []string{"git stash *"}, BashAsk: []string{"rm *"}, BashAllow: []string{"ls *"}}
	if err := a.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(a.SettingsPath); !os.IsNotExist(err) {
		t.Fatalf("rule-only save created config.toml: %v", err)
	}
	intent := readIntent(t, a)
	if intent.SchemaVersion != IntentSchemaVersion || intent.SettingsPath != a.SettingsPath {
		t.Fatalf("intent header = %+v", intent)
	}
	if strings.Join(intent.BashDeny, ",") != "git stash *" || strings.Join(intent.BashAsk, ",") != "rm *" || strings.Join(intent.BashAllow, ",") != "ls *" {
		t.Fatalf("intent rules = %+v", intent)
	}
	loaded, err := a.Load()
	if err != nil || Fingerprint(loaded) != Fingerprint(p) {
		t.Fatalf("Load = %+v, %v", loaded, err)
	}
}

func TestRuleOnlySaveLeavesAnExistingConfigByteForByte(t *testing.T) {
	a := newTestAdapter(t)
	initial := "# my notes\nmodel = \"gpt-5\"\n\n[profiles.default]\nsandbox_mode = \"workspace-write\" # keep\n"
	if err := os.WriteFile(a.SettingsPath, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := a.Save(Policy{BashDeny: []string{"git stash *"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got, _ := os.ReadFile(a.SettingsPath); string(got) != initial {
		t.Fatalf("config rewritten:\n%s", got)
	}
	if a.LastBackup != "" {
		t.Fatalf("backup taken for an untouched config: %s", a.LastBackup)
	}
}

func TestSaveClearsRules(t *testing.T) {
	a := newTestAdapter(t)
	if err := a.Save(Policy{BashDeny: []string{"x"}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := a.Save(Policy{}); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if intent := readIntent(t, a); len(intent.BashDeny) != 0 {
		t.Fatalf("deny should be cleared: %+v", intent)
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
	first, _ := os.ReadFile(a.IntentPath())
	loaded, err := a.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := a.Save(loaded); err != nil {
		t.Fatalf("save2: %v", err)
	}
	second, _ := os.ReadFile(a.IntentPath())
	if string(first) != string(second) {
		t.Errorf("round-trip not byte-identical:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

func TestIntentFromAFutureSchemaIsRefused(t *testing.T) {
	a := newTestAdapter(t)
	if err := os.WriteFile(a.IntentPath(), []byte(`{"schemaVersion":99,"settingsPath":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Load(); err == nil || !strings.Contains(err.Error(), "schema 99") {
		t.Fatalf("future intent schema accepted: %v", err)
	}
}

func TestAdminScopeKeepsItsIntentApartAndNeverWritesRequirements(t *testing.T) {
	dir := t.TempDir()
	a := &Adapter{
		SettingsPath: filepath.Join(dir, "requirements.toml"),
		Scope:        ScopeAdmin,
	}
	if err := a.Save(Policy{BashDeny: []string{"x"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !strings.HasSuffix(a.StatePath(), ".vrooli-permissions-state.admin.json") || !strings.HasSuffix(a.IntentPath(), ".vrooli-permissions-intent.admin.json") {
		t.Errorf("admin scope sidecars: %s %s", a.StatePath(), a.IntentPath())
	}
	if _, err := os.Stat(a.SettingsPath); !os.IsNotExist(err) {
		t.Fatalf("admin rule save wrote requirements.toml: %v", err)
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

func TestSaveProjectsPreToolHookWhenConfigured(t *testing.T) {
	a := newTestAdapter(t)
	a.HookPath = filepath.Join(filepath.Dir(a.SettingsPath), "hooks.json")
	t.Setenv("VROOLI_AGENT_POLICY_RUNNER", "policy-runner")
	if err := a.Save(Policy{BashDeny: []string{"pnpm add *"}}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(a.HookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "PreToolUse") || !strings.Contains(string(raw), "policy-runner hook --runner codex") {
		t.Fatalf("unexpected hook projection: %s", raw)
	}
	// Codex rejects the whole hooks file for an unknown top-level key, so
	// ownership markers may only sit on the hook entries.
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for key := range doc {
		if key != "hooks" && key != "description" {
			t.Fatalf("hooks.json carries non-native top-level key %q: %s", key, raw)
		}
	}
}

func TestDefaultAdapterFollowsASessionHomeLinkToTheSharedConfig(t *testing.T) {
	shared := t.TempDir()
	session := t.TempDir()
	if err := os.WriteFile(filepath.Join(shared, "config.toml"), []byte("model = 'x'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(shared, "config.toml"), filepath.Join(session, "config.toml")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	t.Setenv("CODEX_HOME", session)
	a, err := DefaultAdapter(ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	resolved, _ := filepath.EvalSymlinks(shared)
	if filepath.Dir(a.SettingsPath) != resolved || filepath.Dir(a.IntentPath()) != resolved || filepath.Dir(a.HookPath) != resolved {
		t.Fatalf("adapter paths do not follow the link: %+v intent=%s", a, a.IntentPath())
	}
	if _, err := a.Load(); err != nil {
		t.Fatalf("Load through a session home: %v", err)
	}
}

func decodeTOML(t *testing.T, data []byte) map[string]any {
	t.Helper()
	doc := map[string]any{}
	if err := toml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse: %v\n%s", err, data)
	}
	return doc
}
