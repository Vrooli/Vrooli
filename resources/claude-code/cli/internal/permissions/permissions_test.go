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
	return &Adapter{
		SettingsPath:  filepath.Join(dir, "settings.json"),
		HookScriptDir: filepath.Join(dir, ".vrooli-hooks"),
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
	if !p.Hooks {
		t.Fatal("expected Hooks=true default on missing file")
	}
}

func TestSaveAddsDenyAndHookEntry(t *testing.T) {
	a := newTestAdapter(t)
	p := Policy{BashDeny: []string{"Bash(git stash*)", "Bash(rm -rf /*)"}, Hooks: true}
	if err := a.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	raw, err := os.ReadFile(a.SettingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse settings: %v\n---\n%s", err, raw)
	}

	perms, ok := doc["permissions"].(map[string]any)
	if !ok {
		t.Fatalf("missing permissions object: %v", doc)
	}
	denyAny, ok := perms["deny"].([]any)
	if !ok || len(denyAny) != 2 {
		t.Fatalf("expected 2 deny entries, got %v", perms["deny"])
	}

	hooks, ok := doc["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("missing hooks: %v", doc)
	}
	preToolUse, ok := hooks["PreToolUse"].([]any)
	if !ok || len(preToolUse) != 1 {
		t.Fatalf("expected 1 PreToolUse entry, got %v", hooks["PreToolUse"])
	}
	entry := preToolUse[0].(map[string]any)
	if entry["managedBy"] != ManagedByMarker {
		t.Fatalf("expected managedBy=%q, got %v", ManagedByMarker, entry["managedBy"])
	}
	if entry["matcher"] != "Bash" {
		t.Fatalf("matcher: %v", entry["matcher"])
	}

	info, err := os.Stat(a.HookScriptPath())
	if err != nil {
		t.Fatalf("managed shell hook was not materialized: %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("managed shell hook mode = %o, want 700", info.Mode().Perm())
	}
	hookEntries := entry["hooks"].([]any)
	hookCommand := hookEntries[0].(map[string]any)["command"].(string)
	if !strings.Contains(hookCommand, HookScriptName) || !strings.Contains(hookCommand, "Bash(git stash*)") {
		t.Fatalf("hook command does not carry the managed script and pattern: %q", hookCommand)
	}
}

func TestSavePreservesUnknownTopLevelKeys(t *testing.T) {
	a := newTestAdapter(t)
	initial := `{
  "env": {"FOO": "bar"},
  "permissions": {"allow": ["Bash(ls)"], "extraField": 42},
  "model": "claude-opus-4-7"
}`
	if err := os.WriteFile(a.SettingsPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	p, err := a.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	p.BashDeny = []string{"Bash(git stash*)"}
	if err := a.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	raw, _ := os.ReadFile(a.SettingsPath)
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse: %v\n%s", err, raw)
	}
	if env, ok := doc["env"].(map[string]any); !ok || env["FOO"] != "bar" {
		t.Errorf("env not preserved: %v", doc["env"])
	}
	if doc["model"] != "claude-opus-4-7" {
		t.Errorf("model not preserved: %v", doc["model"])
	}
	perms := doc["permissions"].(map[string]any)
	if perms["extraField"] == nil {
		t.Errorf("extraField under permissions not preserved: %v", perms)
	}
	if allow, _ := perms["allow"].([]any); len(allow) != 1 || allow[0] != "Bash(ls)" {
		t.Errorf("allow not preserved: %v", perms["allow"])
	}
}

func TestSavePreservesHandWrittenHookEntries(t *testing.T) {
	a := newTestAdapter(t)
	initial := `{
  "hooks": {
    "PreToolUse": [
      {"matcher": "Edit", "hooks": [{"type": "command", "command": "/my/hand-written.sh"}]}
    ]
  }
}`
	if err := os.WriteFile(a.SettingsPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	p := Policy{BashDeny: []string{"Bash(git stash*)"}, Hooks: true}
	if err := a.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, _ := os.ReadFile(a.SettingsPath)
	var doc map[string]any
	_ = json.Unmarshal(raw, &doc)
	entries := doc["hooks"].(map[string]any)["PreToolUse"].([]any)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (hand-written + managed), got %d: %s", len(entries), raw)
	}
	var sawHandWritten, sawManaged bool
	for _, e := range entries {
		em := e.(map[string]any)
		if em["matcher"] == "Edit" {
			sawHandWritten = true
		}
		if em["managedBy"] == ManagedByMarker {
			sawManaged = true
		}
	}
	if !sawHandWritten {
		t.Errorf("hand-written hook lost: %s", raw)
	}
	if !sawManaged {
		t.Errorf("managed hook not added: %s", raw)
	}
}

func TestSaveOverwritesPriorManagedHook(t *testing.T) {
	a := newTestAdapter(t)
	if err := a.Save(Policy{BashDeny: []string{"Bash(old)"}, Hooks: true}); err != nil {
		t.Fatalf("first save: %v", err)
	}
	if err := a.Save(Policy{BashDeny: []string{"Bash(new)"}, Hooks: true}); err != nil {
		t.Fatalf("second save: %v", err)
	}
	raw, _ := os.ReadFile(a.SettingsPath)
	if !strings.Contains(string(raw), "Bash(new)") {
		t.Errorf("missing new pattern: %s", raw)
	}
	if strings.Contains(string(raw), "Bash(old)") {
		t.Errorf("old pattern not replaced: %s", raw)
	}
	// Exactly one managed entry.
	count := strings.Count(string(raw), `"managedBy": "`+ManagedByMarker+`"`)
	if count != 1 {
		t.Errorf("expected 1 managed entry, got %d: %s", count, raw)
	}
}

func TestSaveRemovesManagedHookWhenDenyEmpty(t *testing.T) {
	a := newTestAdapter(t)
	if err := a.Save(Policy{BashDeny: []string{"Bash(x)"}, Hooks: true}); err != nil {
		t.Fatalf("seed save: %v", err)
	}
	if err := a.Save(Policy{Hooks: true}); err != nil {
		t.Fatalf("clear save: %v", err)
	}
	raw, _ := os.ReadFile(a.SettingsPath)
	if strings.Contains(string(raw), ManagedByMarker) {
		t.Errorf("managed entry should be removed when no deny patterns remain: %s", raw)
	}
}

func TestFingerprintStableAcrossOrder(t *testing.T) {
	a := Fingerprint(Policy{BashDeny: []string{"a", "b"}, Hooks: true})
	b := Fingerprint(Policy{BashDeny: []string{"b", "a"}, Hooks: true})
	if a != b {
		t.Errorf("fingerprint should be order-independent: %s vs %s", a, b)
	}
}

func TestRoundTripIdempotent(t *testing.T) {
	a := newTestAdapter(t)
	p := Policy{BashDeny: []string{"Bash(git stash*)"}, BashAsk: []string{"Bash(rm *)"}, Hooks: true}
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

func TestStateRoundTrip(t *testing.T) {
	a := newTestAdapter(t)
	p := Policy{BashDeny: []string{"Bash(x)"}, Hooks: true}
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
}

func TestReconcileHookPreservesAndUpdatesIdentifiedEntries(t *testing.T) {
	a := newTestAdapter(t)
	hook := map[string]any{"type": "command", "command": "echo one"}
	result, err := a.ReconcileHook("Stop", "web-console-tts", hook)
	if err != nil || result.Status != "applied" {
		t.Fatalf("first reconcile: %+v, %v", result, err)
	}
	hook["command"] = "echo two"
	result, err = a.ReconcileHook("Stop", "web-console-tts", hook)
	if err != nil || result.Status != "applied" {
		t.Fatalf("second reconcile: %+v, %v", result, err)
	}

	raw, err := os.ReadFile(a.SettingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	group := doc["hooks"].(map[string]any)["Stop"].([]any)[0].(map[string]any)
	entries := group["hooks"].([]any)
	if len(entries) != 1 || entries[0].(map[string]any)["command"] != "echo two" {
		t.Fatalf("unexpected hook projection: %s", raw)
	}
}

func TestRemoveHookIsIdempotentAndPreservesOtherHooks(t *testing.T) {
	a := newTestAdapter(t)
	first := map[string]any{"type": "command", "command": "echo one"}
	second := map[string]any{"type": "command", "command": "echo two"}
	if _, err := a.ReconcileHook("Stop", "first", first); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ReconcileHook("Stop", "second", second); err != nil {
		t.Fatal(err)
	}
	result, err := a.RemoveHook("Stop", "first")
	if err != nil || result.Status != "removed" {
		t.Fatalf("remove: %+v, %v", result, err)
	}
	result, err = a.RemoveHook("Stop", "first")
	if err != nil || result.Status != "unchanged" {
		t.Fatalf("repeat remove: %+v, %v", result, err)
	}
	raw, err := os.ReadFile(a.SettingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"_id": "second"`) || strings.Contains(string(raw), `"_id": "first"`) {
		t.Fatalf("other hook was not preserved: %s", raw)
	}
}
