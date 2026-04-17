package remotesessionprotection

import (
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
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
	}
}

func newTestHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{
		Name:        "remote_session_protection",
		Description: "test safeguard",
		Handler:     "remote_session_protection",
		Platforms:   []string{"linux"},
	})
}

func TestNameAndKind(t *testing.T) {
	h := newTestHandler()
	if h.Name() != "remote_session_protection" {
		t.Fatalf("Name = %q", h.Name())
	}
	if h.Kind() != hostreqspec.KindSafeguard {
		t.Fatalf("Kind = %q", h.Kind())
	}
}

func TestInspectManualRequirement(t *testing.T) {
	h := newTestHandler()
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, hostreqspec.ResolvedRequirement{
		Name:   "remote_session_protection",
		Kind:   hostreqspec.KindSafeguard,
		Manual: true,
	})
	if status.SupportClass != hostreqkit.SupportManualOnly {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
	if status.ExecutionState != hostreqkit.ExecutionManualActionRequired {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectNonLinuxUnsupported(t *testing.T) {
	h := newTestHandler()
	status := h.Inspect(hostreqkit.Host{OS: "darwin"}, hostreqspec.ResolvedRequirement{
		Name: "remote_session_protection",
		Kind: hostreqspec.KindSafeguard,
	})
	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
	if status.ExecutionState != hostreqkit.ExecutionUnsupported {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectNoSysctlNoSystemdNotApplicable(t *testing.T) {
	h := newTestHandler()
	status := h.Inspect(hostreqkit.Host{
		OS:              "linux",
		SupportsSysctl:  false,
		SupportsSystemd: false,
	}, hostreqspec.ResolvedRequirement{
		Name: "remote_session_protection",
		Kind: hostreqspec.KindSafeguard,
	})
	if status.SupportClass != hostreqkit.SupportNotApplicable {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
	if status.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectAllFilesPresent(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		switch path {
		case sysctlPath:
			return []byte(sysctlContent), nil
		case systemdPath:
			return []byte(unitContent), nil
		case logindPath:
			return []byte(logindContent), nil
		default:
			return nil, os.ErrNotExist
		}
	}

	h := newTestHandler()
	status := h.Inspect(hostreqkit.Host{
		OS:              "linux",
		SupportsSysctl:  true,
		SupportsSystemd: true,
	}, hostreqspec.ResolvedRequirement{
		Name: "remote_session_protection",
		Kind: hostreqspec.KindSafeguard,
	})
	if !status.Applied {
		t.Fatal("expected Applied = true")
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectPartialFilesMissing(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == sysctlPath {
			return []byte(sysctlContent), nil
		}
		return nil, os.ErrNotExist
	}

	h := newTestHandler()
	status := h.Inspect(hostreqkit.Host{
		OS:              "linux",
		SupportsSysctl:  true,
		SupportsSystemd: true,
	}, hostreqspec.ResolvedRequirement{
		Name: "remote_session_protection",
		Kind: hostreqspec.KindSafeguard,
	})
	if status.Applied {
		t.Fatal("expected Applied = false when systemd files missing")
	}
	foundPending := false
	for _, note := range status.Notes {
		if strings.Contains(note, "pending managed files") {
			foundPending = true
		}
	}
	if !foundPending {
		t.Fatalf("expected pending note, got %v", status.Notes)
	}
}

func TestInspectSysctlOnly(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == sysctlPath {
			return []byte(sysctlContent), nil
		}
		return nil, os.ErrNotExist
	}

	h := newTestHandler()
	status := h.Inspect(hostreqkit.Host{
		OS:              "linux",
		SupportsSysctl:  true,
		SupportsSystemd: false,
	}, hostreqspec.ResolvedRequirement{
		Name: "remote_session_protection",
		Kind: hostreqspec.KindSafeguard,
	})
	if !status.Applied {
		t.Fatal("sysctl-only host with matching file should be Applied")
	}
}

func TestApplyUnsupportedReturnsEarly(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(hostreqkit.Host{OS: "darwin"}, hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportUnsupported,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionUnsupported {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyNotApplicableReturnsEarly(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(hostreqkit.Host{OS: "linux"}, hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportNotApplicable,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyManualReturnsEarly(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(hostreqkit.Host{OS: "linux"}, hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportManualOnly,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionManualActionRequired {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyAlreadyAppliedSkips(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(hostreqkit.Host{OS: "linux"}, hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
		Applied:      true,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyDryRun(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(hostreqkit.Host{
		OS:              "linux",
		SupportsSysctl:  true,
		SupportsSystemd: true,
	}, hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionWouldApply {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyRunsAllCommands(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" || name == "mkdir" || name == "install" || name == "sysctl" || name == "systemctl" {
			return "/usr/bin/" + name, nil
		}
		return "", os.ErrNotExist
	}

	var calls []string
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil
	}

	h := newTestHandler()
	status, err := h.Apply(hostreqkit.Host{
		OS:              "linux",
		SupportsSysctl:  true,
		SupportsSystemd: true,
	}, hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Applied {
		t.Fatal("expected Applied = true")
	}
	if status.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}

	// Expect 8 sudo commands: mkdir, install, sysctl -p, mkdir, mkdir, install, install, systemctl daemon-reload
	if len(calls) != 8 {
		t.Fatalf("call count = %d, want 8; calls: %v", len(calls), calls)
	}

	needles := []string{
		"mkdir -p",
		"sysctl -p " + sysctlPath,
		"install -m 0644",
		"systemctl daemon-reload",
	}
	for _, needle := range needles {
		found := false
		for _, call := range calls {
			if strings.Contains(call, needle) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected command containing %q, got %v", needle, calls)
		}
	}
}

func TestApplySysctlOnlyHost(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" || name == "mkdir" || name == "install" || name == "sysctl" {
			return "/usr/bin/" + name, nil
		}
		return "", os.ErrNotExist
	}

	var calls []string
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil
	}

	h := newTestHandler()
	status, err := h.Apply(hostreqkit.Host{
		OS:              "linux",
		SupportsSysctl:  true,
		SupportsSystemd: false,
	}, hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Applied {
		t.Fatal("expected Applied = true")
	}
	// Only sysctl commands: mkdir, install, sysctl -p
	if len(calls) != 3 {
		t.Fatalf("call count = %d, want 3; calls: %v", len(calls), calls)
	}
}

func TestApplyMkdirFailure(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "", os.ErrNotExist
	}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		for _, arg := range args {
			if arg == "mkdir" {
				return os.ErrPermission
			}
		}
		return nil
	}

	h := newTestHandler()
	status, err := h.Apply(hostreqkit.Host{
		OS:              "linux",
		SupportsSysctl:  true,
		SupportsSystemd: true,
	}, hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("ExecutionState = %q, want failed", status.ExecutionState)
	}
}
