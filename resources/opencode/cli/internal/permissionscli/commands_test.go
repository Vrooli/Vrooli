package permissionscli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"resource-opencode/cli/internal/permissions"

	"github.com/vrooli/cli-core/agentpolicy"
	"github.com/vrooli/cli-core/cliutil"
)

func newTestHandlers(t *testing.T, kind cliutil.CallerKind) (*Handlers, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	h := &Handlers{
		Adapter:        &permissions.Adapter{SettingsPath: filepath.Join(dir, "opencode.json")},
		DetectCaller:   func() cliutil.CallerKind { return kind },
		Policy:         agentpolicy.DefaultPolicy(),
		CLIVersion:     "test-0.0",
		VersionCommand: []string{"true"},
		VersionRunner:  func(ctx context.Context, args []string) (string, error) { return "1.0.0", nil },
		Stdout:         stdout,
		Stderr:         stderr,
	}
	return h, stdout, stderr
}

func TestDenyAsHumanWrites(t *testing.T) {
	h, stdout, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.Deny([]string{"git stash *"}); err != nil {
		t.Fatalf("Deny: %v", err)
	}
	p, err := h.Adapter.Load()
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
	err := h.Deny([]string{"git stash *"})
	if err == nil {
		t.Fatal("expected deny error")
	}
	if !strings.Contains(err.Error(), agentpolicy.OverrideFlag) {
		t.Errorf("expected override flag mentioned in error: %v", err)
	}
	p, _ := h.Adapter.Load()
	if len(p.BashDeny) != 0 {
		t.Errorf("agent should not have written: %+v", p)
	}
}

func TestDenyAsAgentAllowedWithOverride(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindExternalAgent)
	if err := h.Deny([]string{"-i-was-explicitly-authorized", "git stash *"}); err != nil {
		t.Fatalf("Deny with override: %v", err)
	}
	p, _ := h.Adapter.Load()
	if len(p.BashDeny) != 1 {
		t.Errorf("expected deny pattern saved, got %+v", p)
	}
}

func TestListIsAlwaysAllowedForAgents(t *testing.T) {
	h, stdout, _ := newTestHandlers(t, cliutil.CallerKindExternalAgent)
	if err := h.Adapter.Save(permissions.Policy{BashDeny: []string{"rm -rf /*"}}, "test"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := h.List(nil); err != nil {
		t.Fatalf("List: %v", err)
	}
	if !strings.Contains(stdout.String(), "rm -rf /*") {
		t.Errorf("expected list to include seeded pattern: %s", stdout.String())
	}
}

func TestDriftCheckClean(t *testing.T) {
	h, stdout, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.Deny([]string{"x"}); err != nil {
		t.Fatalf("seed deny: %v", err)
	}
	if err := h.DriftCheck(nil); err != nil {
		t.Fatalf("drift-check clean: %v", err)
	}
	if !strings.Contains(stdout.String(), "clean") {
		t.Errorf("expected clean output: %s", stdout.String())
	}
}

func TestDriftCheckDetectsHandEdit(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.Deny([]string{"x"}); err != nil {
		t.Fatalf("seed deny: %v", err)
	}
	// Simulate a genuine out-of-band hand-edit: flip the managed entry's
	// action directly in opencode.json, bypassing the adapter, so the sidecar
	// fingerprint goes stale. (Editing via Save would keep the sidecar in
	// sync and not be drift.)
	raw, err := os.ReadFile(h.Adapter.SettingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	edited := strings.Replace(string(raw), `"x": "deny"`, `"x": "allow"`, 1)
	if edited == string(raw) {
		t.Fatalf("expected to rewrite the managed entry; file was:\n%s", raw)
	}
	if err := os.WriteFile(h.Adapter.SettingsPath, []byte(edited), 0o644); err != nil {
		t.Fatalf("hand edit: %v", err)
	}
	if err := h.DriftCheck(nil); err == nil {
		t.Fatal("expected drift error")
	}
}

func TestMigrateStripsLegacyKey(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	// Seed a pre-1.0 config that opencode would reject.
	initial := `{
  "model": "anthropic/claude-opus-4-7",
  "permission": {"bash": {"git stash *": "deny"}},
  "x-vrooli-managed-permissions": ["git stash *"]
}`
	if err := os.WriteFile(h.Adapter.SettingsPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := h.Migrate(nil); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	raw, _ := os.ReadFile(h.Adapter.SettingsPath)
	if strings.Contains(string(raw), "x-vrooli-managed-permissions") {
		t.Errorf("legacy key not stripped: %s", raw)
	}
	if !strings.Contains(string(raw), `"git stash *": "deny"`) {
		t.Errorf("managed entry should survive migration: %s", raw)
	}
	// Idempotent: a second run is a no-op and still valid.
	if err := h.Migrate(nil); err != nil {
		t.Fatalf("Migrate (2nd): %v", err)
	}
	st, err := h.Adapter.LoadState()
	if err != nil || st == nil || len(st.ManagedBash) != 1 {
		t.Errorf("sidecar should record migrated list: %+v (%v)", st, err)
	}
}

func TestResetClearsAll(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.Deny([]string{"x"}); err != nil {
		t.Fatalf("Deny: %v", err)
	}
	if err := h.Reset(nil); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	p, _ := h.Adapter.Load()
	if len(p.BashDeny)+len(p.BashAsk)+len(p.BashAllow) != 0 {
		t.Errorf("expected cleared policy, got %+v", p)
	}
}

func TestRemoveOnePattern(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.Adapter.Save(permissions.Policy{BashDeny: []string{"a", "b"}}, "test"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := h.Remove([]string{"a"}); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	p, _ := h.Adapter.Load()
	if len(p.BashDeny) != 1 || p.BashDeny[0] != "b" {
		t.Errorf("expected only 'b' to remain, got %+v", p.BashDeny)
	}
}

func TestDoctorRunsAndChecksPin(t *testing.T) {
	h, stdout, stderr := newTestHandlers(t, cliutil.CallerKindHuman)
	h.VersionRunner = func(ctx context.Context, args []string) (string, error) { return "opencode 2.5.0", nil }
	if err := h.Doctor([]string{"-pinned-version", "1.0"}); err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	if !strings.Contains(stdout.String(), "opencode 2.5.0") {
		t.Errorf("expected installed version in stdout: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "pinned") {
		t.Errorf("expected drift warning in stderr: %s", stderr.String())
	}
}

func TestVrooliAgentAlsoRefused(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindVrooliAgent)
	err := h.Deny([]string{"x"})
	if err == nil {
		t.Fatal("vrooli-agent must also be refused without override")
	}
}

func TestPatternMovesBetweenLists(t *testing.T) {
	// OpenCode is last-match-wins, but our managed entries should not
	// duplicate a pattern across deny/ask/allow. Moving an existing
	// pattern to a new bucket should replace, not add.
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	if err := h.Deny([]string{"git stash *"}); err != nil {
		t.Fatalf("Deny: %v", err)
	}
	if err := h.Ask([]string{"git stash *"}); err != nil {
		t.Fatalf("Ask: %v", err)
	}
	p, _ := h.Adapter.Load()
	if len(p.BashDeny) != 0 {
		t.Errorf("pattern should have moved out of deny: %+v", p.BashDeny)
	}
	if len(p.BashAsk) != 1 || p.BashAsk[0] != "git stash *" {
		t.Errorf("pattern should be in ask: %+v", p.BashAsk)
	}
}
