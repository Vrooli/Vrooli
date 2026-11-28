package permissionscli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/resources/codex/cli/internal/permissions"
)

const testDeclaration = `{"schema_version":"` + permissions.ManagedHooksSchema + `","runner":"codex","hooks":[
 {"id":"vrooli-model-hook","event":"PreToolUse","matcher":"spawn_agent","program":"vrooli-policy-runner","args":["model-hook","--runner","codex"]}]}`

// managedHookHandlers builds handlers over a user-scope adapter with a hooks.json
// path, a declaration file, and a fake Codex trust answer.
func managedHookHandlers(t *testing.T, kind cliutil.CallerKind, trust string, probeErr error) (*Handlers, *permissions.Adapter, string) {
	t.Helper()
	t.Setenv("VROOLI_AGENT_POLICY_RUNNER", "")
	t.Setenv("HOME", t.TempDir())
	h, _, _ := newTestHandlers(t, kind)
	inner := h.AdapterFor
	h.AdapterFor = func(scope permissions.Scope) (*permissions.Adapter, error) {
		a, err := inner(scope)
		if err == nil && scope != permissions.ScopeAdmin {
			a.HookPath = filepath.Join(filepath.Dir(a.SettingsPath), "hooks.json")
		}
		return a, err
	}
	h.ManagedHooksPath = filepath.Join(t.TempDir(), "managed-hooks.json")
	if err := os.WriteFile(h.ManagedHooksPath, []byte(testDeclaration), 0o600); err != nil {
		t.Fatal(err)
	}
	h.TrustProbe = func(_ context.Context, _ string) ([]permissions.HookTrust, error) {
		if probeErr != nil {
			return nil, probeErr
		}
		hooks, err := h.declaration()
		if err != nil {
			return nil, err
		}
		return []permissions.HookTrust{{Command: hooks.Hooks[0].CommandLine(), Status: trust}}, nil
	}
	a, _ := h.AdapterFor(permissions.ScopeUser)
	return h, a, h.ManagedHooksPath
}

func TestHooksPlanWritesNothingAndApplyInstallsThenReportsTrust(t *testing.T) {
	h, a, _ := managedHookHandlers(t, cliutil.CallerKindHuman, permissions.TrustUntrusted, nil)
	stdout := h.Stdout.(interface{ String() string })
	if err := h.runManagedHooks("hooks plan", nil, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(a.HookPath); !os.IsNotExist(err) {
		t.Fatalf("plan wrote hooks.json: %v", err)
	}
	if !strings.Contains(stdout.String(), "would add") {
		t.Fatalf("plan output: %s", stdout.String())
	}
	if err := h.runManagedHooks("hooks apply", nil, true); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	for _, want := range []string{"added", "model-hook --runner codex", "NOT TRUSTED", "/hooks"} {
		if !strings.Contains(out, want) {
			t.Errorf("apply output lacks %q:\n%s", want, out)
		}
	}
	if err := h.managedHookDrift(a); err != nil {
		t.Fatalf("drift after apply: %v", err)
	}
}

func TestHooksApplyIsGatedForAgentsUnlessExplicitlyAuthorized(t *testing.T) {
	h, a, _ := managedHookHandlers(t, cliutil.CallerKindExternalAgent, permissions.TrustTrusted, nil)
	if err := h.runManagedHooks("hooks apply", nil, true); err == nil {
		t.Fatal("agent apply was not gated")
	}
	if _, err := os.Stat(a.HookPath); !os.IsNotExist(err) {
		t.Fatalf("gated apply wrote hooks.json: %v", err)
	}
	if err := h.runManagedHooks("hooks apply", []string{"--i-was-explicitly-authorized"}, true); err != nil {
		t.Fatal(err)
	}
}

func TestDriftCheckFlagsAMissingManagedHook(t *testing.T) {
	h, a, _ := managedHookHandlers(t, cliutil.CallerKindHuman, permissions.TrustTrusted, nil)
	if err := h.managedHookDrift(a); err == nil || !strings.Contains(err.Error(), "resource-codex hooks apply") {
		t.Fatalf("missing hook not reported: %v", err)
	}
}

func TestTrustUnavailableIsReportedNotFatal(t *testing.T) {
	h, _, _ := managedHookHandlers(t, cliutil.CallerKindHuman, "", errors.New("codex not found"))
	stdout := h.Stdout.(interface{ String() string })
	if err := h.runManagedHooks("hooks apply", nil, true); err != nil {
		t.Fatalf("an unanswerable trust probe must not fail apply: %v", err)
	}
	if !strings.Contains(stdout.String(), "trust unknown") || !strings.Contains(stdout.String(), "/hooks") {
		t.Fatalf("output: %s", stdout.String())
	}
}

func TestEnsureStepInstallsTheDeclaredHooks(t *testing.T) {
	h, a, _ := managedHookHandlers(t, cliutil.CallerKindExternalAgent, permissions.TrustTrusted, nil)
	var out strings.Builder
	if err := ManagedHooksStep(h).Run(&out, &out); err != nil {
		t.Fatal(err)
	}
	if err := h.managedHookDrift(a); err != nil {
		t.Fatalf("hook not installed by ensure: %v", err)
	}
}
