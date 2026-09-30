package permissionscli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/grok/cli/internal/permissions"

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
				HooksDir:     filepath.Join(dir, "hooks"),
				Scope:        scope,
			}, nil
		},
		DetectCaller:   func() cliutil.CallerKind { return kind },
		Policy:         agentharness.DefaultPolicy(),
		CLIVersion:     "test-0.0",
		VersionCommand: []string{"true"},
		VersionRunner:  func(ctx context.Context, args []string) (string, error) { return "grok 0.2.72", nil },
		Stdout:         stdout,
		Stderr:         stderr,
	}
	return h, stdout, stderr
}

func TestDenyAsHumanWritesAndInstallsHook(t *testing.T) {
	h, stdout, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.Deny([]string{"Bash(rm -rf *)"}); err != nil {
		t.Fatalf("Deny: %v", err)
	}
	a, _ := h.AdapterFor(permissions.ScopeUser)
	p, err := a.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(p.BashDeny) != 1 || p.BashDeny[0] != "Bash(rm -rf *)" {
		t.Errorf("expected deny pattern saved, got %+v", p)
	}
	if _, err := os.Stat(a.HookConfigPath()); err != nil {
		t.Errorf("expected PreToolUse deny hook installed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Bash(rm -rf *)") {
		t.Errorf("expected stdout to confirm write: %s", stdout.String())
	}
}

func TestDenyAsAgentRefusedWithoutOverride(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindExternalAgent)
	err := h.Deny([]string{"Bash(git stash*)"})
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
	if err := h.Deny([]string{"-i-was-explicitly-authorized", "Bash(git stash*)"}); err != nil {
		t.Fatalf("Deny with override: %v", err)
	}
	a, _ := h.AdapterFor(permissions.ScopeUser)
	p, _ := a.Load()
	if len(p.BashDeny) != 1 {
		t.Errorf("expected deny pattern saved, got %+v", p)
	}
}

func TestVrooliAgentAlsoRefused(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindVrooliAgent)
	if err := h.Deny([]string{"Bash(x)"}); err == nil {
		t.Fatal("vrooli-agent must also be refused without override")
	}
}

func TestListIsAllowedForAgents(t *testing.T) {
	h, stdout, _ := newTestHandlers(t, cliutil.CallerKindExternalAgent)
	a, _ := h.AdapterFor(permissions.ScopeUser)
	if err := a.Save(permissions.Policy{BashDeny: []string{"Bash(rm -rf *)"}, Hooks: true}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := h.List(nil); err != nil {
		t.Fatalf("List: %v", err)
	}
	if !strings.Contains(stdout.String(), "Bash(rm -rf *)") {
		t.Errorf("expected list to include seeded pattern: %s", stdout.String())
	}
}

func TestDriftCheckCleanThenDetectsHandEdit(t *testing.T) {
	h, stdout, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.Deny([]string{"Bash(x)"}); err != nil {
		t.Fatalf("seed deny: %v", err)
	}
	if err := h.DriftCheck(nil); err != nil {
		t.Fatalf("drift-check clean: %v", err)
	}
	if !strings.Contains(stdout.String(), "clean") {
		t.Errorf("expected clean output: %s", stdout.String())
	}
	a, _ := h.AdapterFor(permissions.ScopeUser)
	p, _ := a.Load()
	p.BashDeny = append(p.BashDeny, "Bash(extra*)")
	if err := a.Save(p); err != nil {
		t.Fatalf("hand edit: %v", err)
	}
	if err := h.DriftCheck(nil); err == nil {
		t.Fatal("expected drift error after hand edit")
	}
}

func TestResetClearsAllAndRemovesHook(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.Deny([]string{"Bash(rm -rf *)"}); err != nil {
		t.Fatalf("Deny: %v", err)
	}
	a, _ := h.AdapterFor(permissions.ScopeUser)
	if _, err := os.Stat(a.HookConfigPath()); err != nil {
		t.Fatalf("expected hook before reset: %v", err)
	}
	if err := h.Reset(nil); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	p, _ := a.Load()
	if len(p.BashDeny)+len(p.BashAsk)+len(p.BashAllow) != 0 {
		t.Errorf("expected cleared policy, got %+v", p)
	}
	if _, err := os.Stat(a.HookConfigPath()); !os.IsNotExist(err) {
		t.Errorf("expected hook removed after reset, stat err=%v", err)
	}
}

func TestRemoveOnePattern(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	a, _ := h.AdapterFor(permissions.ScopeUser)
	if err := a.Save(permissions.Policy{BashDeny: []string{"Bash(a)", "Bash(b)"}, Hooks: true}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := h.Remove([]string{"Bash(a)"}); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	p, _ := a.Load()
	if len(p.BashDeny) != 1 || p.BashDeny[0] != "Bash(b)" {
		t.Errorf("expected only 'Bash(b)' to remain, got %+v", p.BashDeny)
	}
}

func TestPatternMovesBetweenLists(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.Deny([]string{"Bash(git stash*)"}); err != nil {
		t.Fatalf("Deny: %v", err)
	}
	if err := h.Allow([]string{"Bash(git stash*)"}); err != nil {
		t.Fatalf("Allow: %v", err)
	}
	a, _ := h.AdapterFor(permissions.ScopeUser)
	p, _ := a.Load()
	if len(p.BashDeny) != 0 {
		t.Errorf("pattern should have moved out of deny: %+v", p.BashDeny)
	}
	if len(p.BashAllow) != 1 || p.BashAllow[0] != "Bash(git stash*)" {
		t.Errorf("pattern should be in allow: %+v", p.BashAllow)
	}
	// Moving the only Bash deny out should also remove the hook.
	if _, err := os.Stat(a.HookConfigPath()); !os.IsNotExist(err) {
		t.Errorf("expected hook removed once deny empty, stat err=%v", err)
	}
}

func TestAdminScopeWritesRequirementsFile(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.Deny([]string{"-scope", "admin", "Bash(rm -rf *)"}); err != nil {
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

func TestUnknownScopeRejected(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	err := h.Deny([]string{"-scope", "bogus", "Bash(x)"})
	if err == nil || !strings.Contains(err.Error(), "unknown --scope") {
		t.Fatalf("expected unknown-scope error, got %v", err)
	}
}

func TestDoctorAffirmsEnforcement(t *testing.T) {
	h, stdout, stderr := newTestHandlers(t, cliutil.CallerKindHuman)
	h.VersionRunner = func(ctx context.Context, args []string) (string, error) { return "grok 9.9.9", nil }
	if err := h.Doctor([]string{"-pinned-version", "0.2.72"}); err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	if !strings.Contains(stdout.String(), "grok 9.9.9") {
		t.Errorf("expected installed version in stdout: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "pinned") {
		t.Errorf("expected drift warning in stderr: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "natively honours") {
		t.Errorf("expected enforcement affirmation in stdout: %s", stdout.String())
	}
}
