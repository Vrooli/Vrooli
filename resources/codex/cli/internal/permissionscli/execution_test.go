package permissionscli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/resources/codex/cli/internal/permissions"
)

const executionFixture = `{"schema_version":"v2","rules":[],"execution":{"filesystem":{"workspace":"write"},"network":{"enabled":true,"domains":{"localhost":"allow"}},"approval":{"policy":"on-request","reviewer":"auto_review"}}}`

func TestExecutionPlanReconcileDigestActivationAndVersion(t *testing.T) {
	h, out, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	path := filepath.Join(t.TempDir(), "execution.json")
	os.WriteFile(path, []byte(executionFixture), 0o600)
	if err := h.Plan([]string{"--document", path, "--json"}); err == nil {
		t.Fatal("unsupported installed version accepted")
	}
	h.VersionRunner = func(context.Context, []string) (string, error) { return "codex-cli 0.156.1", nil }
	args := []string{"--document", path, "--activate", "--json"}
	if err := h.Plan(args); err != nil {
		t.Fatal(err)
	}
	var plan agentharness.PermissionPlanResult
	if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Execution == nil || plan.Execution.Desired == nil || !plan.Execution.NativeNetworkEnabled || !plan.Execution.Activate || plan.Execution.RuntimeEvidence != "unverified" {
		t.Fatal("missing honest execution evidence")
	}
	a, _ := h.AdapterFor(permissions.ScopeUser)
	if _, err := os.Stat(a.SettingsPath); !os.IsNotExist(err) {
		t.Fatal("plan wrote config")
	}
	if err := h.Reconcile(args); err == nil {
		t.Fatal("accepted execution without preview digest")
	}
	if err := h.Reconcile(append(args, "--expected-digest", "stale")); err == nil {
		t.Fatal("accepted wrong preview digest")
	}
	out.Reset()
	if err := h.Reconcile(append(args, "--expected-digest", plan.PreviewDigest)); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(a.SettingsPath)
	if !strings.Contains(string(raw), "network_proxy = true") || !strings.Contains(string(raw), "auto_review") {
		t.Fatalf("activation missing: %s", raw)
	}
	out.Reset()
	if err := h.Plan(args); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Drift {
		t.Fatalf("reconcile not idempotent: %s", out.String())
	}
	if err := h.DriftCheck(nil); err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), "network_proxy = true", "network_proxy = false", 1))
	os.WriteFile(a.SettingsPath, raw, 0o600)
	if err := h.DriftCheck(nil); err == nil {
		t.Fatal("native drift not detected")
	}
	if err := h.Reconcile(append(args, "--expected-digest", plan.PreviewDigest)); err == nil {
		t.Fatal("changed native bytes accepted")
	}
}

func TestExecutionReconcileAgentGateAndActivationMismatch(t *testing.T) {
	h, out, _ := newTestHandlers(t, cliutil.CallerKindExternalAgent)
	h.VersionRunner = func(context.Context, []string) (string, error) { return "codex-cli 0.156.1", nil }
	path := filepath.Join(t.TempDir(), "execution.json")
	os.WriteFile(path, []byte(executionFixture), 0o600)
	if err := h.Plan([]string{"--document", path, "--json"}); err != nil {
		t.Fatal(err)
	}
	var plan agentharness.PermissionPlanResult
	json.Unmarshal(out.Bytes(), &plan)
	args := []string{"--document", path, "--expected-digest", plan.PreviewDigest}
	if err := h.Reconcile(args); err == nil {
		t.Fatal("agent bypassed authorization gate")
	}
	if err := h.Reconcile(append(args, "--activate", "--i-was-explicitly-authorized")); err == nil {
		t.Fatal("activation not bound to preview")
	}
	if err := h.Reconcile(append(args, "--i-was-explicitly-authorized")); err != nil {
		t.Fatal(err)
	}
}

func TestExecutionStateOnlyRecoveryAndStaleStatePreview(t *testing.T) {
	h, out, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	h.VersionRunner = func(context.Context, []string) (string, error) { return "codex-cli 0.156.1", nil }
	path := filepath.Join(t.TempDir(), "execution.json")
	if err := os.WriteFile(path, []byte(executionFixture), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"--document", path, "--json"}
	preview := func() agentharness.PermissionPlanResult {
		t.Helper()
		out.Reset()
		if err := h.Plan(args); err != nil {
			t.Fatal(err)
		}
		var plan agentharness.PermissionPlanResult
		if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
			t.Fatal(err)
		}
		return plan
	}
	plan := preview()
	if err := h.Reconcile(append(args, "--expected-digest", plan.PreviewDigest)); err != nil {
		t.Fatal(err)
	}
	a, _ := h.AdapterFor(permissions.ScopeUser)
	configBefore, err := os.ReadFile(a.SettingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(a.StatePath()); err != nil {
		t.Fatal(err)
	}
	plan = preview()
	if !plan.StateDrift || !plan.Drift || plan.LiveFingerprint != plan.DesiredFingerprint {
		t.Fatal("state-only drift not represented")
	}
	if err = h.Reconcile(append(args, "--expected-digest", plan.PreviewDigest)); err != nil {
		t.Fatal(err)
	}
	configAfter, err := os.ReadFile(a.SettingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(configBefore) != string(configAfter) {
		t.Fatal("state repair rewrote native config")
	}
	plan = preview()
	if plan.StateDrift || plan.Drift {
		t.Fatal("state repair not idempotent")
	}
	if err = os.Remove(a.StatePath()); err != nil {
		t.Fatal(err)
	}
	if err = h.Reconcile(append(args, "--expected-digest", plan.PreviewDigest)); err == nil {
		t.Fatal("stale state preview accepted")
	}
}

func TestExecutionPreviewBindsRuntimeVersionAndExecutable(t *testing.T) {
	h, out, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	version := "codex-cli 0.156.1"
	h.VersionRunner = func(_ context.Context, args []string) (string, error) {
		if len(args) != 2 || args[1] != "--version" {
			t.Fatalf("unexpected version command: %v", args)
		}
		return version, nil
	}
	path := filepath.Join(t.TempDir(), "execution.json")
	os.WriteFile(path, []byte(executionFixture), 0o600)
	binary := filepath.Join(t.TempDir(), "codex")
	args := []string{"--document", path, "--codex-executable", binary, "--json"}
	if err := h.Plan(args); err != nil {
		t.Fatal(err)
	}
	var plan agentharness.PermissionPlanResult
	json.Unmarshal(out.Bytes(), &plan)
	if plan.Execution.VersionCommand[0] != binary {
		t.Fatal("preview omitted executable evidence")
	}
	version = "codex-cli 0.156.2"
	if err := h.Reconcile(append(args, "--expected-digest", plan.PreviewDigest)); err == nil {
		t.Fatal("runtime version change not bound to preview")
	}
}

func TestUnqualifiedFilteredNetworkVersionCannotWrite(t *testing.T) {
	h, _, _ := newTestHandlers(t, cliutil.CallerKindHuman)
	h.VersionRunner = func(context.Context, []string) (string, error) { return "codex-cli 0.141.0", nil }
	path := filepath.Join(t.TempDir(), "execution.json")
	if err := os.WriteFile(path, []byte(executionFixture), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, run := range []func([]string) error{h.Plan, h.Reconcile} {
		if err := run([]string{"--document", path, "--activate"}); err == nil {
			t.Fatal("unqualified filtered networking was accepted")
		}
	}
	a, _ := h.AdapterFor(permissions.ScopeUser)
	if _, err := os.Stat(a.SettingsPath); !os.IsNotExist(err) {
		t.Fatal("unsupported networking wrote config")
	}
}
