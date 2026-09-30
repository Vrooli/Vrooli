package permissions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/vrooli/agentharness"
)

func testExecution() *agentharness.ExecutionPermissions {
	enabled := true
	return &agentharness.ExecutionPermissions{Filesystem: agentharness.ExecutionFilesystem{Workspace: "write"}, Network: agentharness.ExecutionNetwork{Enabled: &enabled, Domains: map[string]string{"localhost": "allow", "127.0.0.1": "allow"}}, Approval: agentharness.ExecutionApproval{Policy: "on-request", Reviewer: "user"}}
}

func TestExecutionStagingActivationAndNativeDrift(t *testing.T) {
	a := newTestAdapter(t)
	if err := os.WriteFile(a.SettingsPath, []byte("model = 'preserved'\napprovals_reviewer = 'user'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	e := testExecution()
	e.Approval.Reviewer = "auto_review"
	p := Policy{Execution: e}
	if err := a.Save(p); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(a.SettingsPath)
	var doc map[string]any
	if err := toml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["default_permissions"] != nil || doc["approval_policy"] != nil || doc["approvals_reviewer"] != "user" {
		t.Fatalf("staging activated settings: %s", raw)
	}
	network := doc["permissions"].(map[string]any)[ExecutionProfile].(map[string]any)["network"].(map[string]any)
	if network["enabled"] != false {
		t.Fatal("staged profile allows unfiltered networking")
	}
	live, err := a.Load()
	if err != nil {
		t.Fatal(err)
	}
	if live.NativeFingerprint != DesiredExecutionFingerprint(e, false) {
		t.Fatal("staged projection differs")
	}
	p.ExecutionActive = true
	if err = a.Save(p); err != nil {
		t.Fatal(err)
	}
	live, err = a.Load()
	if err != nil {
		t.Fatal(err)
	}
	if live.NativeFingerprint != DesiredExecutionFingerprint(e, true) {
		t.Fatal("active projection differs")
	}
	raw, _ = os.ReadFile(a.SettingsPath)
	altered := strings.Replace(string(raw), "network_proxy = true", "network_proxy = false", 1)
	if altered == string(raw) {
		t.Fatal("fixture did not disable proxy")
	}
	if err = os.WriteFile(a.SettingsPath, []byte(altered), 0o600); err != nil {
		t.Fatal(err)
	}
	live, err = a.Load()
	if err != nil {
		t.Fatal(err)
	}
	if live.NativeFingerprint == DesiredExecutionFingerprint(e, true) {
		t.Fatal("proxy drift undetected")
	}
	info, _ := os.Stat(a.SettingsPath)
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatal("mode widened")
	}
	if a.LastBackup == "" {
		t.Fatal("missing recovery backup")
	}
	backupInfo, _ := os.Stat(a.LastBackup)
	if runtime.GOOS != "windows" && backupInfo.Mode().Perm() != 0o600 {
		t.Fatal("backup not private")
	}
}

func TestExecutionConflictsAndSnapshotRejectBeforeWrite(t *testing.T) {
	for _, initial := range []string{"sandbox_mode = 'workspace-write'\n", "[profiles.old]\nsandbox_mode = 'read-only'\n", "[permissions.vrooli]\nextends = ':workspace'\n"} {
		t.Run(initial, func(t *testing.T) {
			a := newTestAdapter(t)
			os.WriteFile(a.SettingsPath, []byte(initial), 0o600)
			err := a.Save(Policy{Execution: testExecution(), ExecutionActive: true})
			if err == nil {
				t.Fatal("accepted conflicting settings")
			}
			raw, _ := os.ReadFile(a.SettingsPath)
			if string(raw) != initial {
				t.Fatal("failure mutated native config")
			}
		})
	}
	a := newTestAdapter(t)
	live, err := a.Load()
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(a.SettingsPath, []byte("model = 'concurrent'\n"), 0o600)
	if err = a.Save(Policy{Execution: testExecution(), SnapshotDigest: live.SnapshotDigest}); err == nil {
		t.Fatal("stale plan accepted")
	}
}

func TestExecutionCODExHomeAndAdminPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	a, err := DefaultAdapter(ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	if a.SettingsPath != filepath.Join(home, "config.toml") || a.HookPath != filepath.Join(home, "hooks.json") {
		t.Fatalf("CODEX_HOME ignored: %+v", a)
	}
	a, err = DefaultAdapter(ScopeAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(a.SettingsPath, home) {
		t.Fatal("admin path is user-controlled")
	}
	if err = a.CheckExecutionProjection(Policy{Execution: testExecution()}); err == nil {
		t.Fatal("unsupported admin execution accepted")
	}
}

func TestHookOnlyDriftAndConfigRollback(t *testing.T) {
	a := newTestAdapter(t)
	a.HookPath = filepath.Join(filepath.Dir(a.SettingsPath), "hooks.json")
	p := Policy{BashDeny: []string{"rm -rf /"}}
	if err := a.Save(p); err != nil {
		t.Fatal(err)
	}
	live, err := a.Load()
	if err != nil {
		t.Fatal(err)
	}
	if live.HookFingerprint != a.DesiredHookFingerprint(p) {
		t.Fatal("projected hook differs")
	}
	os.WriteFile(a.HookPath, []byte(`{"hooks":{}}`), 0o600)
	live, err = a.Load()
	if err != nil {
		t.Fatal(err)
	}
	if live.HookFingerprint == a.DesiredHookFingerprint(p) {
		t.Fatal("missing hook not detected")
	}
	// JSON null is valid JSON, but cannot be a native hook document.
	os.WriteFile(a.HookPath, []byte(`null`), 0o600)
	before, _ := os.ReadFile(a.SettingsPath)
	err = a.Save(Policy{BashDeny: []string{"different"}})
	if err == nil {
		t.Fatal("invalid hook document accepted")
	}
	after, _ := os.ReadFile(a.SettingsPath)
	if string(before) != string(after) {
		t.Fatal("config not restored after hook failure")
	}
}

func TestWritableRootsKeepProtectedDirectoriesReadOnly(t *testing.T) {
	e := testExecution()
	root := t.TempDir()
	e.Filesystem.WritableRoots = []string{root}
	profile := nativeProfile(e, true)
	raw, _ := json.Marshal(profile)
	if !strings.Contains(string(raw), `".codex":"read"`) {
		t.Fatal("protected path missing")
	}
}

func TestHookWriteFailureRestoresConfig(t *testing.T) {
	a := newTestAdapter(t)
	a.HookPath = filepath.Join(filepath.Dir(a.SettingsPath), "hooks.json")
	if err := a.Save(Policy{BashDeny: []string{"original"}}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(a.SettingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(a.HookPath+".vrooli-lock", []byte("busy"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = a.Save(Policy{BashDeny: []string{"replacement"}})
	if err == nil || !strings.Contains(err.Error(), "rollback") {
		t.Fatalf("missing recovery result: %v", err)
	}
	after, err := os.ReadFile(a.SettingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("hook write failure left changed config")
	}
	if a.LastBackup == "" {
		t.Fatal("backup missing")
	}
}
