package permissions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func testAdapter(t *testing.T, scope Scope) *Adapter {
	t.Helper()
	dir := t.TempDir()
	file := "config.toml"
	if scope == ScopeAdmin {
		file = "requirements.toml"
	}
	return &Adapter{
		SettingsPath: filepath.Join(dir, file),
		HooksDir:     filepath.Join(dir, "hooks"),
		Scope:        scope,
	}
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	a := testAdapter(t, ScopeUser)
	p, err := a.Load()
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if len(p.BashDeny)+len(p.BashAsk)+len(p.BashAllow) != 0 {
		t.Errorf("expected empty policy, got %+v", p)
	}
	if !p.Hooks {
		t.Error("expected Hooks default true for a fresh policy")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	a := testAdapter(t, ScopeUser)
	in := Policy{
		BashDeny:  []string{"Bash(rm -rf *)", "Bash(git push --force*)"},
		BashAllow: []string{"Bash(git *)"},
		BashAsk:   []string{"Bash(npm publish*)"},
		Hooks:     true,
	}
	if err := a.Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := a.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if strings.Join(got.BashDeny, ",") != "Bash(git push --force*),Bash(rm -rf *)" {
		t.Errorf("deny round-trip (want sorted): %+v", got.BashDeny)
	}
	if len(got.BashAllow) != 1 || got.BashAllow[0] != "Bash(git *)" {
		t.Errorf("allow round-trip: %+v", got.BashAllow)
	}
	if len(got.BashAsk) != 1 || got.BashAsk[0] != "Bash(npm publish*)" {
		t.Errorf("ask round-trip: %+v", got.BashAsk)
	}
}

func TestSaveWritesNativePermissionSection(t *testing.T) {
	a := testAdapter(t, ScopeUser)
	if err := a.Save(Policy{BashDeny: []string{"Bash(rm -rf *)"}, Hooks: true}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(a.SettingsPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var doc map[string]any
	if err := toml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	section, ok := doc["permission"].(map[string]any)
	if !ok {
		t.Fatalf("expected native [permission] table, got: %s", data)
	}
	if _, ok := section["deny"].([]any); !ok {
		t.Errorf("expected [permission].deny array, got: %s", data)
	}
}

func TestSavePreservesUnrelatedConfig(t *testing.T) {
	a := testAdapter(t, ScopeUser)
	// Seed a realistic grok config.toml including an array-of-tables.
	seed := `[cli]
installer = "internal"

[marketplace]
official_marketplace_auto_installed = true

[[marketplace.sources]]
name = "xAI Official"
git = "https://github.com/xai-org/plugin-marketplace.git"

[permission]
rules = [{ action = "allow", tool = "read" }]
`
	if err := os.WriteFile(a.SettingsPath, []byte(seed), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := a.Save(Policy{BashDeny: []string{"Bash(rm -rf *)"}, Hooks: true}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, _ := os.ReadFile(a.SettingsPath)
	var doc map[string]any
	if err := toml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Unrelated top-level tables survive.
	if cli, _ := doc["cli"].(map[string]any); cli["installer"] != "internal" {
		t.Errorf("[cli] not preserved: %s", data)
	}
	mk, _ := doc["marketplace"].(map[string]any)
	if mk["official_marketplace_auto_installed"] != true {
		t.Errorf("[marketplace] not preserved: %s", data)
	}
	srcs, _ := mk["sources"].([]any)
	if len(srcs) != 1 {
		t.Fatalf("[[marketplace.sources]] not preserved: %s", data)
	}
	// Pre-existing [permission].rules survives alongside our deny.
	perm, _ := doc["permission"].(map[string]any)
	if _, ok := perm["rules"]; !ok {
		t.Errorf("[permission].rules clobbered: %s", data)
	}
	if _, ok := perm["deny"]; !ok {
		t.Errorf("[permission].deny missing: %s", data)
	}
}

func TestHookMaterializedForBashDeny(t *testing.T) {
	a := testAdapter(t, ScopeUser)
	if err := a.Save(Policy{BashDeny: []string{"Bash(rm -rf *)"}, Hooks: true}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Shared script exists and is executable.
	info, err := os.Stat(a.HookScriptPath())
	if err != nil {
		t.Fatalf("hook script missing: %v", err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("hook script not executable: %v", info.Mode())
	}
	// Hook JSON references the script and the inner glob (not the wrapper).
	data, err := os.ReadFile(a.HookConfigPath())
	if err != nil {
		t.Fatalf("hook json missing: %v", err)
	}
	var doc struct {
		Hooks struct {
			PreToolUse []struct {
				Matcher string `json:"matcher"`
				Hooks   []struct {
					Type    string `json:"type"`
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"PreToolUse"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse hook json: %v", err)
	}
	if len(doc.Hooks.PreToolUse) != 1 || doc.Hooks.PreToolUse[0].Matcher != "Bash" {
		t.Fatalf("unexpected hook shape: %s", data)
	}
	cmd := doc.Hooks.PreToolUse[0].Hooks[0].Command
	if !strings.Contains(cmd, "vrooli-bash-deny.sh") {
		t.Errorf("hook command missing script path: %s", cmd)
	}
	if !strings.Contains(cmd, "rm -rf *") || strings.Contains(cmd, "Bash(") {
		t.Errorf("hook should pass the inner glob, not the Bash() wrapper: %s", cmd)
	}
}

func TestHookRemovedWhenNoBashDeny(t *testing.T) {
	a := testAdapter(t, ScopeUser)
	if err := a.Save(Policy{BashDeny: []string{"Bash(rm -rf *)"}, Hooks: true}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(a.HookConfigPath()); err != nil {
		t.Fatalf("expected hook present after deny: %v", err)
	}
	// Clearing the deny list removes the hook JSON.
	if err := a.Save(Policy{BashAllow: []string{"Bash(git *)"}, Hooks: true}); err != nil {
		t.Fatalf("Save clear: %v", err)
	}
	if _, err := os.Stat(a.HookConfigPath()); !os.IsNotExist(err) {
		t.Errorf("expected hook JSON removed when no Bash deny remain, stat err=%v", err)
	}
}

func TestHookSkipsNonBashDenyPatterns(t *testing.T) {
	a := testAdapter(t, ScopeUser)
	// Only non-Bash deny patterns → native config enforces them, but no
	// Bash hook should be created.
	if err := a.Save(Policy{BashDeny: []string{"Read(/secrets/**)", "Grep"}, Hooks: true}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(a.HookConfigPath()); !os.IsNotExist(err) {
		t.Errorf("non-Bash deny should not create a Bash hook, stat err=%v", err)
	}
	// But the native deny rules are still persisted.
	p, _ := a.Load()
	if len(p.BashDeny) != 2 {
		t.Errorf("native deny rules should persist: %+v", p.BashDeny)
	}
}

func TestRenderHookExtractsGlobs(t *testing.T) {
	a := testAdapter(t, ScopeUser)
	h := a.RenderHook(Policy{BashDeny: []string{"Bash(git stash*)", "Read(x)"}, Hooks: true})
	if h == nil {
		t.Fatal("expected non-nil hook render")
	}
	if a.RenderHook(Policy{BashDeny: []string{"Read(x)"}, Hooks: true}) != nil {
		t.Error("expected nil render when no Bash() deny patterns present")
	}
	if a.RenderHook(Policy{BashDeny: []string{"Bash(x)"}, Hooks: false}) != nil {
		t.Error("expected nil render when Hooks disabled")
	}
}

func TestFingerprintStableAndOrderInsensitive(t *testing.T) {
	a := Policy{BashDeny: []string{"Bash(b)", "Bash(a)"}, Hooks: true}
	b := Policy{BashDeny: []string{"Bash(a)", "Bash(b)"}, Hooks: true}
	if Fingerprint(a) != Fingerprint(b) {
		t.Error("fingerprint should be order-insensitive")
	}
	c := Policy{BashDeny: []string{"Bash(a)"}, Hooks: true}
	if Fingerprint(a) == Fingerprint(c) {
		t.Error("fingerprint should change with content")
	}
}

func TestAdminScopeUsesDistinctFiles(t *testing.T) {
	user := testAdapter(t, ScopeUser)
	admin := testAdapter(t, ScopeAdmin)
	if !strings.HasSuffix(user.SettingsPath, "config.toml") {
		t.Errorf("user scope should target config.toml: %s", user.SettingsPath)
	}
	if !strings.HasSuffix(admin.SettingsPath, "requirements.toml") {
		t.Errorf("admin scope should target requirements.toml: %s", admin.SettingsPath)
	}
	if filepath.Base(user.HookConfigPath()) == filepath.Base(admin.HookConfigPath()) {
		t.Error("user and admin hook JSON filenames must differ")
	}
	if filepath.Base(user.StatePath()) == filepath.Base(admin.StatePath()) {
		t.Error("user and admin state filenames must differ")
	}
}
