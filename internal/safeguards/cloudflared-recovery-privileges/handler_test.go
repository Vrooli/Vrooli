package cloudflaredrecoveryprivileges

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func stubLookups(t *testing.T) func() {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origReadFile := hostreqkit.ReadFileFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	origRunningAsRoot := hostreqkit.RunningAsRootFn
	origWriteTemp := hostreqkit.WriteTempFileFn
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
		hostreqkit.RunningAsRootFn = origRunningAsRoot
		hostreqkit.WriteTempFileFn = origWriteTemp
	}
}

func newTestHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{
		Name:    "cloudflared_recovery_privileges",
		Handler: "cloudflared_recovery_privileges",
	})
}

func linuxReq() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name:     "cloudflared_recovery_privileges",
		Kind:     hostreqspec.KindSafeguard,
		Required: false,
	}
}

func linuxHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", SupportsSystemd: true}
}

func cloudflaredPresentStub() {
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "systemctl" && len(args) > 0 && args[0] == "list-unit-files" {
			return []byte("cloudflared.service enabled enabled\n"), nil
		}
		return nil, errors.New("unexpected CombinedOutputFn call")
	}
}

func cloudflaredAbsentStub() {
	hostreqkit.CombinedOutputFn = func(name string, _ ...string) ([]byte, error) {
		if name == "systemctl" {
			return []byte(""), nil
		}
		return nil, errors.New("unexpected CombinedOutputFn call")
	}
}

func TestNameAndKind(t *testing.T) {
	h := newTestHandler()
	if h.Name() != "cloudflared_recovery_privileges" {
		t.Fatalf("Name = %q", h.Name())
	}
	if h.Kind() != hostreqspec.KindSafeguard {
		t.Fatalf("Kind = %q", h.Kind())
	}
}

func TestInspectManualRequirement(t *testing.T) {
	h := newTestHandler()
	req := linuxReq()
	req.Manual = true
	status := h.Inspect(linuxHost(), req)
	if status.SupportClass != hostreqkit.SupportManualOnly {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
	if status.ExecutionState != hostreqkit.ExecutionManualActionRequired {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectNonLinuxNotApplicable(t *testing.T) {
	h := newTestHandler()
	status := h.Inspect(hostreqkit.Host{OS: "darwin"}, linuxReq())
	if status.SupportClass != hostreqkit.SupportNotApplicable {
		t.Fatalf("SupportClass = %q, want NotApplicable", status.SupportClass)
	}
}

func TestInspectNoSystemdNotApplicable(t *testing.T) {
	h := newTestHandler()
	host := linuxHost()
	host.SupportsSystemd = false
	status := h.Inspect(host, linuxReq())
	if status.SupportClass != hostreqkit.SupportNotApplicable {
		t.Fatalf("SupportClass = %q, want NotApplicable", status.SupportClass)
	}
}

func TestInspectNoCloudflaredUnitNotApplicable(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	cloudflaredAbsentStub()
	hostreqkit.RunningAsRootFn = func() bool { return false }
	t.Setenv("USER", "alice")

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if status.SupportClass != hostreqkit.SupportNotApplicable {
		t.Fatalf("SupportClass = %q, want NotApplicable; notes: %v", status.SupportClass, status.Notes)
	}
}

func TestInspectAlreadyPresent(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	cloudflaredPresentStub()
	hostreqkit.RunningAsRootFn = func() bool { return false }
	t.Setenv("USER", "alice")

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == sudoersPath {
			return []byte(buildSudoersContent("alice")), nil
		}
		return nil, os.ErrNotExist
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if !status.Applied {
		t.Fatalf("expected Applied=true; notes: %v", status.Notes)
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectMissingPromotesToRequiredUnderRoot(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	cloudflaredPresentStub()
	hostreqkit.RunningAsRootFn = func() bool { return true }
	hostreqkit.ReadFileFn = func(string) ([]byte, error) { return nil, os.ErrNotExist }
	t.Setenv("SUDO_USER", "alice")

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatalf("expected Applied=false when grant missing")
	}
	if !status.Required {
		t.Fatalf("expected self-promotion to Required under root")
	}
	if !strings.Contains(strings.Join(status.Notes, " "), sudoersPath) {
		t.Errorf("expected notes to mention %q; got %v", sudoersPath, status.Notes)
	}
}

func TestApplyDryRunWritesNothing(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	cloudflaredPresentStub()
	hostreqkit.RunningAsRootFn = func() bool { return false }
	t.Setenv("USER", "alice")

	calls := 0
	hostreqkit.RunCommandFn = func(string, []string, hostreqkit.EnsureOptions) error {
		calls++
		return nil
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	out, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionWouldApply {
		t.Fatalf("ExecutionState = %q, want WouldApply", out.ExecutionState)
	}
	if calls != 0 {
		t.Errorf("dry-run ran %d commands; want 0", calls)
	}
}

func TestApplyAlreadyAppliedShortCircuits(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	calls := 0
	hostreqkit.RunCommandFn = func(string, []string, hostreqkit.EnsureOptions) error {
		calls++
		return nil
	}

	h := newTestHandler()
	out, err := h.Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
		Applied:      true,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", out.ExecutionState)
	}
	if calls != 0 {
		t.Errorf("already-applied path ran %d commands; want 0", calls)
	}
}

func TestApplyNotApplicableShortCircuits(t *testing.T) {
	h := newTestHandler()
	out, err := h.Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportNotApplicable,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Fatalf("ExecutionState = %q", out.ExecutionState)
	}
}

func TestApplyVisudoInvalidAbortsWithoutWriting(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	cloudflaredPresentStub()
	hostreqkit.RunningAsRootFn = func() bool { return true } // bypass WithSudo skip
	t.Setenv("SUDO_USER", "alice")
	hostreqkit.WriteTempFileFn = func(string) (string, error) { return "/tmp/fake-sudoers", nil }

	var installCalled bool
	hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		if name == "visudo" {
			return errors.New("invalid sudoers syntax")
		}
		if name == "install" {
			installCalled = true
		}
		_ = args
		return nil
	}

	h := newTestHandler()
	out, err := h.Apply(linuxHost(), hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("ExecutionState = %q, want Failed", out.ExecutionState)
	}
	if installCalled {
		t.Error("install must not run when visudo validation fails")
	}
}

func TestApplyHappyPathValidatesThenInstalls(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	cloudflaredPresentStub()
	hostreqkit.RunningAsRootFn = func() bool { return true }
	t.Setenv("SUDO_USER", "alice")
	hostreqkit.WriteTempFileFn = func(string) (string, error) { return "/tmp/fake-sudoers", nil }

	var order []string
	var installArgs []string
	hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		order = append(order, name)
		if name == "install" {
			installArgs = args
		}
		return nil
	}

	h := newTestHandler()
	out, err := h.Apply(linuxHost(), hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("ExecutionState = %q, want Applied; notes: %v", out.ExecutionState, out.Notes)
	}
	if len(order) != 2 || order[0] != "visudo" || order[1] != "install" {
		t.Fatalf("command order = %v, want [visudo install]", order)
	}
	// 0440 perms, exact temp→dest install.
	if strings.Join(installArgs, " ") != "-m 0440 /tmp/fake-sudoers "+sudoersPath {
		t.Errorf("install args = %v", installArgs)
	}
}

func TestBuildSudoersContentExactLine(t *testing.T) {
	content := buildSudoersContent("alice")
	wantLine := "alice ALL=(root) NOPASSWD: /usr/bin/systemctl restart cloudflared, /usr/bin/systemctl reset-failed cloudflared"
	if !strings.Contains(content, wantLine) {
		t.Errorf("content missing exact grant line %q; got:\n%s", wantLine, content)
	}
	if !strings.Contains(content, "Managed by Vrooli") {
		t.Error("content missing managed-by marker")
	}
	if !strings.HasSuffix(content, "\n") {
		t.Error("content must end with a newline")
	}
	if strings.Contains(content, "*") {
		t.Error("grant must not contain wildcards")
	}
}
