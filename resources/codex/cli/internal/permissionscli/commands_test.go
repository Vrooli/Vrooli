package permissionscli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/codex/cli/internal/permissions"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliutil"
)

func newTestHandlers(t *testing.T, kind cliutil.CallerKind) (*Handlers, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	h := &Handlers{
		AdapterFor: func(scope permissions.Scope) (*permissions.Adapter, error) {
			if scope == "" {
				scope = permissions.ScopeUser
			}
			file := "config.toml"
			if scope == permissions.ScopeAdmin {
				file = "requirements.toml"
			}
			return &permissions.Adapter{
				SettingsPath: filepath.Join(dir, file),
				Scope:        scope,
			}, nil
		},
		DetectCaller:   func() cliutil.CallerKind { return kind },
		Policy:         agentharness.DefaultPolicy(),
		CLIVersion:     "test-0.0",
		VersionCommand: []string{"true"},
		VersionRunner:  func(ctx context.Context, args []string) (string, error) { return "codex-cli 0.131.0", nil },
		Stdout:         stdout,
		Stderr:         stderr,
	}
	return h, stdout, stderr
}

func TestDenyAsHumanWrites(t *testing.T) {
	h, stdout, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.shared().Deny([]string{"git stash *"}); err != nil {
		t.Fatalf("Deny: %v", err)
	}
	a, _ := h.AdapterFor(permissions.ScopeUser)
	p, err := a.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(p.BashDeny) != 1 || p.BashDeny[0] != "git stash *" {
		t.Errorf("expected deny pattern saved, got %+v", p)
	}
	if !strings.Contains(stdout.String(), "git stash *") {
		t.Errorf("expected stdout to confirm write: %s", stdout.String())
	}
}

func TestDenyAsAgentRefusedWithoutOverride(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindExternalAgent)
	err := h.shared().Deny([]string{"git stash *"})
	if err == nil {
		t.Fatal("expected deny error")
	}
	if !strings.Contains(err.Error(), agentharness.OverrideFlag) {
		t.Errorf("expected override flag mentioned in error: %v", err)
	}
	a, _ := h.AdapterFor(permissions.ScopeUser)
	p, _ := a.Load()
	if len(p.BashDeny) != 0 {
		t.Errorf("agent should not have written: %+v", p)
	}
}

func TestDenyAsAgentAllowedWithOverride(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindExternalAgent)
	if err := h.shared().Deny([]string{"-i-was-explicitly-authorized", "git stash *"}); err != nil {
		t.Fatalf("Deny with override: %v", err)
	}
	a, _ := h.AdapterFor(permissions.ScopeUser)
	p, _ := a.Load()
	if len(p.BashDeny) != 1 {
		t.Errorf("expected deny pattern saved, got %+v", p)
	}
}

func TestAdminScopeWritesRequirementsFile(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.shared().Deny([]string{"-scope", "admin", "git stash *"}); err != nil {
		t.Fatalf("Deny admin: %v", err)
	}
	a, _ := h.AdapterFor(permissions.ScopeAdmin)
	p, _ := a.Load()
	if len(p.BashDeny) != 1 {
		t.Errorf("admin scope write missing: %+v", p)
	}
	if !strings.HasSuffix(a.SettingsPath, "requirements.toml") {
		t.Errorf("admin scope should target requirements.toml: %s", a.SettingsPath)
	}
}

func TestUserAndAdminScopesAreSeparate(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.shared().Deny([]string{"user-pat"}); err != nil {
		t.Fatalf("user Deny: %v", err)
	}
	if err := h.shared().Deny([]string{"-scope", "admin", "admin-pat"}); err != nil {
		t.Fatalf("admin Deny: %v", err)
	}
	userA, _ := h.AdapterFor(permissions.ScopeUser)
	adminA, _ := h.AdapterFor(permissions.ScopeAdmin)
	up, _ := userA.Load()
	ap, _ := adminA.Load()
	if len(up.BashDeny) != 1 || up.BashDeny[0] != "user-pat" {
		t.Errorf("user scope: %+v", up)
	}
	if len(ap.BashDeny) != 1 || ap.BashDeny[0] != "admin-pat" {
		t.Errorf("admin scope: %+v", ap)
	}
}

func TestListIsAlwaysAllowedForAgents(t *testing.T) {
	h, stdout, _ := newTestHandlers(t, cliutil.CallerKindExternalAgent)
	a, _ := h.AdapterFor(permissions.ScopeUser)
	if err := a.Save(permissions.Policy{BashDeny: []string{"rm -rf /*"}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := h.shared().List(nil); err != nil {
		t.Fatalf("List: %v", err)
	}
	if !strings.Contains(stdout.String(), "rm -rf /*") {
		t.Errorf("expected list to include seeded pattern: %s", stdout.String())
	}
}

func TestDriftCheckClean(t *testing.T) {
	h, stdout, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.shared().Deny([]string{"x"}); err != nil {
		t.Fatalf("seed deny: %v", err)
	}
	if err := h.shared().DriftCheck(nil); err != nil {
		t.Fatalf("drift-check clean: %v", err)
	}
	if !strings.Contains(stdout.String(), "clean") {
		t.Errorf("expected clean output: %s", stdout.String())
	}
}

func TestDriftCheckDetectsHandEdit(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.shared().Deny([]string{"x"}); err != nil {
		t.Fatalf("seed deny: %v", err)
	}
	a, _ := h.AdapterFor(permissions.ScopeUser)
	p, _ := a.Load()
	p.BashDeny = append(p.BashDeny, "extra")
	if err := a.Save(p); err != nil {
		t.Fatalf("hand edit save: %v", err)
	}
	if err := h.shared().DriftCheck(nil); err == nil {
		t.Fatal("expected drift error")
	}
}

func TestResetClearsAll(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.shared().Deny([]string{"x"}); err != nil {
		t.Fatalf("Deny: %v", err)
	}
	if err := h.shared().Reset(nil); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	a, _ := h.AdapterFor(permissions.ScopeUser)
	p, _ := a.Load()
	if len(p.BashDeny)+len(p.BashAsk)+len(p.BashAllow) != 0 {
		t.Errorf("expected cleared policy, got %+v", p)
	}
}

func TestRemoveOnePattern(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	a, _ := h.AdapterFor(permissions.ScopeUser)
	if err := a.Save(permissions.Policy{BashDeny: []string{"a", "b"}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := h.shared().Remove([]string{"a"}); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	p, _ := a.Load()
	if len(p.BashDeny) != 1 || p.BashDeny[0] != "b" {
		t.Errorf("expected only 'b' to remain, got %+v", p.BashDeny)
	}
}

func TestDoctorMentionsEnforcementCaveat(t *testing.T) {
	h, stdout, stderr := newTestHandlers(t, cliutil.CallerKindHuman)
	h.VersionRunner = func(ctx context.Context, args []string) (string, error) { return "codex-cli 2.5.0", nil }
	if err := h.shared().Doctor([]string{"-pinned-version", "1.0"}); err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	if !strings.Contains(stdout.String(), "codex-cli 2.5.0") {
		t.Errorf("expected installed version in stdout: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "pinned") {
		t.Errorf("expected drift warning in stderr: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "does NOT enforce") {
		t.Errorf("expected enforcement caveat in stdout: %s", stdout.String())
	}
}

func TestVrooliAgentAlsoRefused(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindVrooliAgent)
	err := h.shared().Deny([]string{"x"})
	if err == nil {
		t.Fatal("vrooli-agent must also be refused without override")
	}
}

func TestUnknownScopeRejected(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	err := h.shared().Deny([]string{"-scope", "bogus", "x"})
	if err == nil || !strings.Contains(err.Error(), "unknown --scope") {
		t.Fatalf("expected unknown-scope error, got %v", err)
	}
}

func TestPatternMovesBetweenLists(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.shared().Deny([]string{"git stash *"}); err != nil {
		t.Fatalf("Deny: %v", err)
	}
	if err := h.shared().Ask([]string{"git stash *"}); err != nil {
		t.Fatalf("Ask: %v", err)
	}
	a, _ := h.AdapterFor(permissions.ScopeUser)
	p, _ := a.Load()
	if len(p.BashDeny) != 0 {
		t.Errorf("pattern should have moved out of deny: %+v", p.BashDeny)
	}
	if len(p.BashAsk) != 1 || p.BashAsk[0] != "git stash *" {
		t.Errorf("pattern should be in ask: %+v", p.BashAsk)
	}
}

func TestEnsureStepMovesTheLegacyTableOutOfTheSharedConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	config := filepath.Join(home, "config.toml")
	if err := os.WriteFile(config, []byte("model = 'm'\n\n[vrooli.permissions]\nbash_deny = ['rm -rf /']\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	h, stdout, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	h.AdapterFor = permissions.DefaultAdapter
	if err := h.shared().Doctor(nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "retired [vrooli] table") {
		t.Fatalf("doctor does not report the pending move: %s", stdout.String())
	}
	var out bytes.Buffer
	step := IntentMigrationStep()
	if err := step.Run(&out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "moved the legacy [vrooli] table") {
		t.Fatalf("ensure output: %s", out.String())
	}
	if got, _ := os.ReadFile(config); string(got) != "model = 'm'\n" {
		t.Fatalf("config after ensure: %q", got)
	}
	out.Reset()
	if err := step.Run(&out, &out); err != nil || out.Len() != 0 {
		t.Fatalf("second ensure = %q, %v; want silent no-op", out.String(), err)
	}
	stdout.Reset()
	if err := h.shared().List(nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "rm -rf /") || !strings.Contains(stdout.String(), "intent: "+filepath.Join(home, ".vrooli-permissions-intent.json")) {
		t.Fatalf("list after ensure: %s", stdout.String())
	}
}

func TestDriftCheckReportsAMissingHookEvenWithoutState(t *testing.T) {
	h, _, stderr := newTestHandlers(t, cliutil.CallerKindHuman)
	inner := h.AdapterFor
	h.AdapterFor = func(scope permissions.Scope) (*permissions.Adapter, error) {
		a, err := inner(scope)
		if err == nil {
			a.HookPath = filepath.Join(filepath.Dir(a.SettingsPath), "hooks.json")
		}
		return a, err
	}
	a, _ := h.AdapterFor(permissions.ScopeUser)
	// Intent recorded by an earlier version that never projected the hook.
	if err := os.WriteFile(a.IntentPath(), []byte(`{"schemaVersion":1,"settingsPath":"x","bashDeny":["rm -rf /"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := h.shared().DriftCheck(nil); err == nil || !strings.Contains(stderr.String(), "PreToolUse hook") {
		t.Fatalf("missing hook not reported: %v %s", err, stderr.String())
	}
}
