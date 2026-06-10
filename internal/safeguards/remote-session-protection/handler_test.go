package remotesessionprotection

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const procMeminfo = "/proc/meminfo"

// ── Helpers ──────────────────────────────────────────────────────────────────

func stubAll(t *testing.T) func() {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origReadFile := hostreqkit.ReadFileFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	origCompetingSwaps := CompetingSwapsFn
	// Default stubs: no commands found, no files exist, no desktop detected
	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	hostreqkit.ReadFileFn = func(string) ([]byte, error) { return nil, os.ErrNotExist }
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) { return nil, fmt.Errorf("stubbed") }
	hostreqkit.RunCommandFn = func(string, []string, hostreqkit.EnsureOptions) error { return nil }
	// Default: no competing swaps. Tests that exercise the dedup-warning path
	// override.
	CompetingSwapsFn = func() []string { return nil }
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
		CompetingSwapsFn = origCompetingSwaps
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

func linuxHost() hostreqkit.Host {
	return hostreqkit.Host{
		OS:              "linux",
		PackageManager:  "apt-get",
		SupportsSysctl:  true,
		SupportsSystemd: true,
	}
}

func baseReq() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name:     "remote_session_protection",
		Kind:     hostreqspec.KindSafeguard,
		Required: true,
	}
}

// fakeMeminfo returns /proc/meminfo content for the given RAM in GB.
func fakeMeminfo(ramGB int, swapGB int) string {
	memKB := ramGB * 1024 * 1024
	swapKB := swapGB * 1024 * 1024
	return fmt.Sprintf("MemTotal:       %d kB\nMemFree:        %d kB\nSwapTotal:      %d kB\n",
		memKB, memKB/2, swapKB)
}

// ── Name and Kind ────────────────────────────────────────────────────────────

func TestNameAndKind(t *testing.T) {
	h := newTestHandler()
	if h.Name() != "remote_session_protection" {
		t.Fatalf("Name = %q", h.Name())
	}
	if h.Kind() != hostreqspec.KindSafeguard {
		t.Fatalf("Kind = %q", h.Kind())
	}
}

// ── Inspect: early returns ───────────────────────────────────────────────────

func TestInspectManualRequirement(t *testing.T) {
	h := newTestHandler()
	req := baseReq()
	req.Manual = true
	status := h.Inspect(linuxHost(), req)
	if status.SupportClass != hostreqkit.SupportManualOnly {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
}

func TestInspectNonLinuxUnsupported(t *testing.T) {
	h := newTestHandler()
	status := h.Inspect(hostreqkit.Host{OS: "darwin"}, baseReq())
	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
}

func TestInspectNoSysctlNoSystemdNotApplicable(t *testing.T) {
	h := newTestHandler()
	host := linuxHost()
	host.SupportsSysctl = false
	host.SupportsSystemd = false
	status := h.Inspect(host, baseReq())
	if status.SupportClass != hostreqkit.SupportNotApplicable {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
}

// ── Inspect: static files (no desktop) ───────────────────────────────────────

func TestInspectAllStaticFilesPresent(t *testing.T) {
	restore := stubAll(t)
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

	status := newTestHandler().Inspect(linuxHost(), baseReq())
	if !status.Applied {
		t.Fatalf("expected Applied = true, notes: %v", status.Notes)
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectPartialStaticFilesMissing(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == sysctlPath {
			return []byte(sysctlContent), nil
		}
		return nil, os.ErrNotExist
	}

	status := newTestHandler().Inspect(linuxHost(), baseReq())
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

func TestInspectSysctlOnlyHost(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == sysctlPath {
			return []byte(sysctlContent), nil
		}
		return nil, os.ErrNotExist
	}

	host := linuxHost()
	host.SupportsSystemd = false
	status := newTestHandler().Inspect(host, baseReq())
	if !status.Applied {
		t.Fatal("sysctl-only host with matching file should be Applied")
	}
}

// ── Inspect: with desktop detected ──────────────────────────────────────────

func TestInspectDesktopDetectedAllPresent(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	alloc := memoryAllocation{
		systemMB:       32768,
		desktopMinMB:   4915,
		desktopLowMB:   6963,
		workloadHighMB: 27852,
		workloadMaxMB:  31129,
		targetSwapGB:   32,
	}

	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "systemctl" && len(args) > 0 && args[0] == "list-unit-files" {
			return []byte("gdm3.service enabled enabled\n"), nil
		}
		return nil, fmt.Errorf("stubbed")
	}

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		switch path {
		case sysctlPath:
			return []byte(sysctlContent), nil
		case systemdPath:
			return []byte(unitContent), nil
		case logindPath:
			return []byte(logindContent), nil
		case procMeminfo:
			return []byte(fakeMeminfo(32, 32)), nil
		case desktopSlicePath:
			return []byte(desktopSliceContent(alloc)), nil
		case workloadSlicePath:
			return []byte(workloadSliceContent()), nil
		case dockerDaemonJSON:
			return nil, os.ErrNotExist // Docker not configured, but not installed either
		default:
			return nil, os.ErrNotExist
		}
	}

	// Docker not on PATH, so docker config not checked
	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "", os.ErrNotExist
	}

	status := newTestHandler().Inspect(linuxHost(), baseReq())
	if !status.Applied {
		t.Fatalf("expected Applied = true, notes: %v", status.Notes)
	}
}

func TestInspectDesktopDetectedSliceMissing(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "systemctl" && len(args) > 0 && args[0] == "list-unit-files" {
			return []byte("lightdm.service enabled\n"), nil
		}
		return nil, fmt.Errorf("stubbed")
	}

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		switch path {
		case sysctlPath:
			return []byte(sysctlContent), nil
		case systemdPath:
			return []byte(unitContent), nil
		case logindPath:
			return []byte(logindContent), nil
		case procMeminfo:
			return []byte(fakeMeminfo(16, 16)), nil
		default:
			return nil, os.ErrNotExist
		}
	}

	status := newTestHandler().Inspect(linuxHost(), baseReq())
	if status.Applied {
		t.Fatal("expected Applied = false when desktop slice files missing")
	}
	foundDesktopSlice := false
	for _, note := range status.Notes {
		if strings.Contains(note, desktopSlicePath) {
			foundDesktopSlice = true
		}
	}
	if !foundDesktopSlice {
		t.Fatalf("expected desktop slice in pending, got %v", status.Notes)
	}
}

func TestInspectDesktopDetectedInsufficientSwap(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	alloc := memoryAllocation{
		systemMB:       32768,
		desktopMinMB:   4915,
		desktopLowMB:   6963,
		workloadHighMB: 27852,
		workloadMaxMB:  31129,
		targetSwapGB:   32,
	}

	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "systemctl" && len(args) > 0 && args[0] == "list-unit-files" {
			return []byte("gdm.service enabled\n"), nil
		}
		return nil, fmt.Errorf("stubbed")
	}

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		switch path {
		case sysctlPath:
			return []byte(sysctlContent), nil
		case systemdPath:
			return []byte(unitContent), nil
		case logindPath:
			return []byte(logindContent), nil
		case procMeminfo:
			return []byte(fakeMeminfo(32, 4)), nil // only 4GB swap, need 32GB
		case desktopSlicePath:
			return []byte(desktopSliceContent(alloc)), nil
		case workloadSlicePath:
			return []byte(workloadSliceContent()), nil
		default:
			return nil, os.ErrNotExist
		}
	}

	status := newTestHandler().Inspect(linuxHost(), baseReq())
	if status.Applied {
		t.Fatal("expected Applied = false when swap insufficient")
	}
	foundSwap := false
	for _, note := range status.Notes {
		if strings.Contains(note, swapFile) {
			foundSwap = true
		}
	}
	if !foundSwap {
		t.Fatalf("expected swapfile in pending, got %v", status.Notes)
	}
}

func TestInspectDesktopDetectedDockerNotConfigured(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	alloc := memoryAllocation{
		systemMB:       32768,
		desktopMinMB:   4915,
		desktopLowMB:   6963,
		workloadHighMB: 27852,
		workloadMaxMB:  31129,
		targetSwapGB:   32,
	}

	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "systemctl" && len(args) > 0 && args[0] == "list-unit-files" {
			return []byte("gdm3.service enabled\n"), nil
		}
		return nil, fmt.Errorf("stubbed")
	}

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "docker" {
			return "/usr/bin/docker", nil
		}
		return "", os.ErrNotExist
	}

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		switch path {
		case sysctlPath:
			return []byte(sysctlContent), nil
		case systemdPath:
			return []byte(unitContent), nil
		case logindPath:
			return []byte(logindContent), nil
		case procMeminfo:
			return []byte(fakeMeminfo(32, 32)), nil
		case desktopSlicePath:
			return []byte(desktopSliceContent(alloc)), nil
		case workloadSlicePath:
			return []byte(workloadSliceContent()), nil
		case dockerDaemonJSON:
			return []byte(`{}`), nil // Docker installed but not configured
		default:
			return nil, os.ErrNotExist
		}
	}

	status := newTestHandler().Inspect(linuxHost(), baseReq())
	if status.Applied {
		t.Fatal("expected Applied = false when Docker not configured")
	}
	foundDocker := false
	for _, note := range status.Notes {
		if strings.Contains(note, dockerDaemonJSON) {
			foundDocker = true
		}
	}
	if !foundDocker {
		t.Fatalf("expected docker daemon.json in pending, got %v", status.Notes)
	}
}

// ── Apply: early returns ─────────────────────────────────────────────────────

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
	status, err := h.Apply(linuxHost(), hostreqkit.ItemStatus{
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
	status, err := h.Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionWouldApply {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

// ── Apply: static files (no desktop) ─────────────────────────────────────────

func TestApplyStaticCommandsNoDesktop(t *testing.T) {
	restore := stubAll(t)
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

	status, err := newTestHandler().Apply(linuxHost(), hostreqkit.ItemStatus{
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

	// Expect 8 commands: mkdir, install, sysctl -p, mkdir, mkdir, install, install, systemctl daemon-reload
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
	restore := stubAll(t)
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

	host := linuxHost()
	host.SupportsSystemd = false
	status, err := newTestHandler().Apply(host, hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Applied {
		t.Fatal("expected Applied = true")
	}
	if len(calls) != 3 {
		t.Fatalf("call count = %d, want 3; calls: %v", len(calls), calls)
	}
}

func TestApplyMkdirFailure(t *testing.T) {
	restore := stubAll(t)
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

	status, err := newTestHandler().Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("ExecutionState = %q, want failed", status.ExecutionState)
	}
}

// ── Apply: with desktop detected ─────────────────────────────────────────────

func TestApplyDesktopFullFlow(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		switch name {
		case "sudo", "mkdir", "install", "sysctl", "systemctl",
			"fallocate", "chmod", "mkswap", "swapon", "swapoff", "sh":
			return "/usr/bin/" + name, nil
		case "docker":
			return "/usr/bin/docker", nil
		}
		return "", os.ErrNotExist
	}

	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "systemctl" {
			if len(args) > 0 && args[0] == "list-unit-files" {
				return []byte("gdm3.service enabled enabled\n"), nil
			}
			if len(args) > 0 && args[0] == "is-active" {
				return []byte("active"), nil
			}
		}
		return nil, fmt.Errorf("stubbed")
	}

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		switch path {
		case procMeminfo:
			return []byte(fakeMeminfo(32, 4)), nil // 32GB RAM, 4GB swap (needs more)
		case "/etc/fstab":
			return []byte("# empty fstab\n"), nil
		case dockerDaemonJSON:
			return []byte(`{"log-driver":"json-file"}`), nil
		}
		return nil, os.ErrNotExist
	}

	var calls []string
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil
	}

	status, err := newTestHandler().Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Applied {
		t.Fatalf("expected Applied = true, notes: %v", status.Notes)
	}

	// Verify key commands were issued
	needles := []string{
		"sysctl -p",                // static sysctl
		"systemctl daemon-reload",  // static daemon-reload
		"fallocate",                // swap creation
		"mkswap",                   // swap format
		"swapon",                   // swap activate
		"systemctl restart docker", // docker restart
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
			t.Errorf("expected command containing %q, got %v", needle, calls)
		}
	}

	// Verify install commands for slice files
	foundDesktopSlice := false
	foundWorkloadSlice := false
	for _, call := range calls {
		if strings.Contains(call, "install") && strings.Contains(call, desktopSlicePath) {
			foundDesktopSlice = true
		}
		if strings.Contains(call, "install") && strings.Contains(call, workloadSlicePath) {
			foundWorkloadSlice = true
		}
	}
	if !foundDesktopSlice {
		t.Errorf("expected install for desktop slice, calls: %v", calls)
	}
	if !foundWorkloadSlice {
		t.Errorf("expected install for workload slice, calls: %v", calls)
	}
}

func TestApplyDesktopSwapSufficient(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		switch name {
		case "sudo", "mkdir", "install", "sysctl", "systemctl":
			return "/usr/bin/" + name, nil
		}
		return "", os.ErrNotExist
	}

	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "systemctl" && len(args) > 0 && args[0] == "list-unit-files" {
			return []byte("gdm3.service enabled\n"), nil
		}
		return nil, fmt.Errorf("stubbed")
	}

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == procMeminfo {
			return []byte(fakeMeminfo(16, 16)), nil // swap already sufficient
		}
		if path == "/etc/fstab" {
			return []byte(swapFile + " none swap sw 0 0\n"), nil
		}
		return nil, os.ErrNotExist
	}

	var calls []string
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil
	}

	status, err := newTestHandler().Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Applied {
		t.Fatalf("expected Applied = true, notes: %v", status.Notes)
	}

	// Swap commands should NOT appear since swap is already sufficient
	for _, call := range calls {
		if strings.Contains(call, "fallocate") || strings.Contains(call, "mkswap") {
			t.Fatalf("unexpected swap command when swap is sufficient: %s", call)
		}
	}
}

func TestApplyDesktopProtectionBestEffort(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" || name == "mkdir" || name == "install" || name == "sysctl" || name == "systemctl" {
			return "/usr/bin/" + name, nil
		}
		return "", os.ErrNotExist
	}

	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "systemctl" && len(args) > 0 && args[0] == "list-unit-files" {
			return []byte("sddm.service enabled\n"), nil
		}
		return nil, fmt.Errorf("stubbed")
	}

	// /proc/meminfo unreadable → calculateMemory fails → desktop protection fails
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		return nil, os.ErrNotExist
	}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		return nil
	}

	status, err := newTestHandler().Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatal(err)
	}
	// Static files still succeed, desktop protection is best-effort
	if !status.Applied {
		t.Fatalf("expected Applied = true (desktop is best-effort), notes: %v", status.Notes)
	}
	foundPartial := false
	for _, note := range status.Notes {
		if strings.Contains(note, "desktop protection partially applied") {
			foundPartial = true
		}
	}
	if !foundPartial {
		t.Fatalf("expected partial desktop protection note, got %v", status.Notes)
	}
}

// ── Memory calculation ───────────────────────────────────────────────────────

func TestCalculateMemory32GB(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == procMeminfo {
			return []byte(fakeMeminfo(32, 0)), nil
		}
		return nil, os.ErrNotExist
	}

	alloc, err := calculateMemory()
	if err != nil {
		t.Fatal(err)
	}
	if alloc.systemMB != 32768 {
		t.Fatalf("systemMB = %d, want 32768", alloc.systemMB)
	}
	// 15% of 32768 = 4915, which is > 4096 minimum
	if alloc.desktopMinMB != 4915 {
		t.Fatalf("desktopMinMB = %d, want 4915", alloc.desktopMinMB)
	}
	if alloc.desktopLowMB != 4915+desktopBufferMB {
		t.Fatalf("desktopLowMB = %d, want %d", alloc.desktopLowMB, 4915+desktopBufferMB)
	}
	if alloc.targetSwapGB != 32 {
		t.Fatalf("targetSwapGB = %d, want 32", alloc.targetSwapGB)
	}
}

func TestCalculateMemory2GB(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == procMeminfo {
			return []byte(fakeMeminfo(2, 0)), nil
		}
		return nil, os.ErrNotExist
	}

	alloc, err := calculateMemory()
	if err != nil {
		t.Fatal(err)
	}
	// 15% of 2048 = 307, below 4096 minimum → clamped to 4096
	if alloc.desktopMinMB != desktopMinMB {
		t.Fatalf("desktopMinMB = %d, want %d (clamped to minimum)", alloc.desktopMinMB, desktopMinMB)
	}
	if alloc.targetSwapGB != 2 {
		t.Fatalf("targetSwapGB = %d, want 2", alloc.targetSwapGB)
	}
}

func TestCalculateMemory128GB(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == procMeminfo {
			return []byte(fakeMeminfo(128, 0)), nil
		}
		return nil, os.ErrNotExist
	}

	alloc, err := calculateMemory()
	if err != nil {
		t.Fatal(err)
	}
	// swap capped at 64GB
	if alloc.targetSwapGB != maxSwapGB {
		t.Fatalf("targetSwapGB = %d, want %d (capped)", alloc.targetSwapGB, maxSwapGB)
	}
}

func TestCalculateMemoryBadMeminfo(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == procMeminfo {
			return []byte("garbage content\n"), nil
		}
		return nil, os.ErrNotExist
	}

	_, err := calculateMemory()
	if err == nil {
		t.Fatal("expected error for unparseable meminfo")
	}
}

func TestParseMeminfoField(t *testing.T) {
	content := "MemTotal:       33554432 kB\nMemFree:        16777216 kB\nSwapTotal:      8388608 kB\n"

	mem, swap, err := hostinventory.ParseLinuxMeminfo(content)
	if err != nil {
		t.Fatalf("ParseLinuxMeminfo: %v", err)
	}
	if v := mem.TotalBytes / 1024; v != 33554432 {
		t.Fatalf("MemTotal = %d, want 33554432", v)
	}
	if v := swap.TotalBytes / 1024; v != 8388608 {
		t.Fatalf("SwapTotal = %d, want 8388608", v)
	}
}

// ── Desktop detection ────────────────────────────────────────────────────────

func TestIsDesktopInstalledDisplayManager(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "systemctl" && len(args) > 0 && args[0] == "list-unit-files" {
			return []byte("gdm3.service enabled enabled\nsshd.service enabled enabled\n"), nil
		}
		return nil, fmt.Errorf("stubbed")
	}

	if !isDesktopInstalled() {
		t.Fatal("expected desktop detected via gdm3")
	}
}

func TestIsDesktopInstalledXrdp(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "systemctl" && len(args) > 0 && args[0] == "list-unit-files" {
			return []byte("xrdp.service enabled\n"), nil
		}
		return nil, fmt.Errorf("stubbed")
	}

	if !isDesktopInstalled() {
		t.Fatal("expected desktop detected via xrdp")
	}
}

func TestIsDesktopInstalledGuiProcess(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "pgrep" {
			for _, a := range args {
				if a == "Xorg" {
					return []byte("12345\n"), nil
				}
			}
		}
		return nil, fmt.Errorf("not found")
	}

	if !isDesktopInstalled() {
		t.Fatal("expected desktop detected via Xorg process")
	}
}

func TestIsDesktopInstalledHeadless(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	// Default stubs return errors → no desktop detected
	if isDesktopInstalled() {
		t.Fatal("expected no desktop on headless stub")
	}
}

// ── Content generators ───────────────────────────────────────────────────────

func TestDesktopSliceContent(t *testing.T) {
	alloc := memoryAllocation{desktopMinMB: 4096, desktopLowMB: 6144}
	content := desktopSliceContent(alloc)
	if !strings.Contains(content, "MemoryMin=4096M") {
		t.Fatalf("missing MemoryMin, got:\n%s", content)
	}
	if !strings.Contains(content, "MemoryLow=6144M") {
		t.Fatalf("missing MemoryLow, got:\n%s", content)
	}
	if !strings.Contains(content, "ManagedOOMPreference=omit") {
		t.Fatalf("missing ManagedOOMPreference, got:\n%s", content)
	}
}

func TestWorkloadSliceContent(t *testing.T) {
	content := workloadSliceContent()
	if !strings.Contains(content, "MemoryHigh=85%") {
		t.Fatalf("missing MemoryHigh, got:\n%s", content)
	}
	if !strings.Contains(content, "MemoryMax=95%") {
		t.Fatalf("missing MemoryMax, got:\n%s", content)
	}
	if !strings.Contains(content, "ManagedOOMMemoryPressure=kill") {
		t.Fatalf("missing ManagedOOMMemoryPressure, got:\n%s", content)
	}
}

// ── Docker config ────────────────────────────────────────────────────────────

func TestDockerConfigAppliedTrue(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == dockerDaemonJSON {
			return []byte(`{
				"exec-opts": ["native.cgroupdriver=systemd"],
				"cgroup-parent": "workload.slice",
				"log-driver": "json-file"
			}`), nil
		}
		return nil, os.ErrNotExist
	}

	if !dockerConfigApplied() {
		t.Fatal("expected docker config to be detected as applied")
	}
}

func TestDockerConfigAppliedMissingCgroupParent(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == dockerDaemonJSON {
			return []byte(`{"exec-opts": ["native.cgroupdriver=systemd"]}`), nil
		}
		return nil, os.ErrNotExist
	}

	if dockerConfigApplied() {
		t.Fatal("expected docker config NOT applied when cgroup parent missing")
	}
}

func TestDockerConfigAppliedNoFile(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	if dockerConfigApplied() {
		t.Fatal("expected docker config NOT applied when file missing")
	}
}

func TestDockerConfigMergesExisting(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" || name == "mkdir" || name == "install" {
			return "/usr/bin/" + name, nil
		}
		return "", os.ErrNotExist
	}

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == dockerDaemonJSON {
			return []byte(`{"log-driver":"json-file","storage-driver":"overlay2"}`), nil
		}
		return nil, os.ErrNotExist
	}

	var targetedDocker bool
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		for _, arg := range args {
			if arg == dockerDaemonJSON {
				targetedDocker = true
			}
		}
		return nil
	}

	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("not running")
	}

	err := applyDockerConfig(hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatal(err)
	}
	if !targetedDocker {
		t.Fatal("expected docker daemon.json to be targeted by install command")
	}
}

// ── Swap management ──────────────────────────────────────────────────────────

func TestReadCurrentSwapGB(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == procMeminfo {
			return []byte(fakeMeminfo(32, 16)), nil
		}
		return nil, os.ErrNotExist
	}

	if got := readCurrentSwapGB(); got != 16 {
		t.Fatalf("readCurrentSwapGB = %d, want 16", got)
	}
}

func TestReadCurrentSwapGBNoMeminfo(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	if got := readCurrentSwapGB(); got != 0 {
		t.Fatalf("readCurrentSwapGB = %d, want 0 when meminfo unreadable", got)
	}
}

func TestEnsureSwapSkipsWhenSufficient(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == procMeminfo {
			return []byte(fakeMeminfo(16, 16)), nil
		}
		return nil, os.ErrNotExist
	}

	var calls []string
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		calls = append(calls, name)
		return nil
	}

	alloc := memoryAllocation{targetSwapGB: 16}
	if err := ensureSwap(alloc, hostreqkit.EnsureOptions{SudoMode: "ask"}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Fatalf("expected no commands when swap sufficient, got %v", calls)
	}
}

func TestEnsureSwapFallocateFallbackToDd(t *testing.T) {
	restore := stubAll(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == procMeminfo {
			return []byte(fakeMeminfo(8, 0)), nil
		}
		if path == "/etc/fstab" {
			return []byte(swapFile + " none swap sw 0 0\n"), nil // already in fstab
		}
		return nil, os.ErrNotExist
	}

	var calls []string
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		cmd := name + " " + strings.Join(args, " ")
		calls = append(calls, cmd)
		// Fail fallocate, succeed dd
		for _, a := range args {
			if a == "fallocate" {
				return fmt.Errorf("fallocate not supported")
			}
		}
		return nil
	}

	alloc := memoryAllocation{targetSwapGB: 8}
	if err := ensureSwap(alloc, hostreqkit.EnsureOptions{SudoMode: "ask"}); err != nil {
		t.Fatal(err)
	}

	foundDd := false
	for _, call := range calls {
		if strings.Contains(call, "dd") && strings.Contains(call, "if=/dev/zero") {
			foundDd = true
		}
	}
	if !foundDd {
		t.Fatalf("expected dd fallback, got %v", calls)
	}
}

// ── Competing-swap detection (Phase 4.5 — 2026-05-07 host-stability) ─────────

func TestInspectCompetingSwapNoteAppearsWhenExtraSwapActive(t *testing.T) {
	restore := stubAll(t)
	defer restore()
	CompetingSwapsFn = func() []string { return []string{"/swap.img"} }

	status := newTestHandler().Inspect(linuxHost(), hostreqspec.ResolvedRequirement{
		Name:     "remote_session_protection",
		Kind:     hostreqspec.KindSafeguard,
		Required: true,
	})

	joined := strings.Join(status.Notes, " | ")
	if !strings.Contains(joined, "/swap.img") {
		t.Errorf("expected note to name the competing swap path; got %v", status.Notes)
	}
	if !strings.Contains(joined, "competing swap area") {
		t.Errorf("expected `competing swap area` phrasing in note; got %v", status.Notes)
	}
	if !strings.Contains(joined, "swapoff") {
		t.Errorf("expected dedupe hint with swapoff in note; got %v", status.Notes)
	}
}

func TestInspectCompetingSwapNoteAbsentWhenNoExtras(t *testing.T) {
	restore := stubAll(t)
	defer restore()
	// stubAll defaults CompetingSwapsFn to return nil — leave it that way.

	status := newTestHandler().Inspect(linuxHost(), hostreqspec.ResolvedRequirement{
		Name:     "remote_session_protection",
		Kind:     hostreqspec.KindSafeguard,
		Required: true,
	})

	for _, n := range status.Notes {
		if strings.Contains(n, "competing swap area") {
			t.Errorf("did not expect competing-swap note when none active; got: %v", status.Notes)
		}
	}
}

func TestCompetingSwapsFnFiltersManagedFile(t *testing.T) {
	origCombinedOutput := hostreqkit.CombinedOutputFn
	defer func() { hostreqkit.CombinedOutputFn = origCombinedOutput }()

	// Mimic `swapon --show=NAME --noheadings` output with both files active.
	// The managed swapFile must be filtered out; everything else surfaces.
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "swapon" {
			return []byte("/swap.img\n" + swapFile + "\n/mnt/encrypted-swap\n"), nil
		}
		return nil, fmt.Errorf("unexpected: %s %v", name, args)
	}

	got := CompetingSwapsFn()
	wantContains := []string{"/swap.img", "/mnt/encrypted-swap"}
	if len(got) != len(wantContains) {
		t.Fatalf("got %d competing swaps, want %d (%v)", len(got), len(wantContains), got)
	}
	for _, want := range wantContains {
		found := false
		for _, g := range got {
			if g == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %q in competing swaps; got %v", want, got)
		}
	}
	for _, g := range got {
		if g == swapFile {
			t.Errorf("managed swap file %q must be filtered out; got %v", swapFile, got)
		}
	}
}

func TestCompetingSwapsFnReturnsNilOnSwaponError(t *testing.T) {
	origCombinedOutput := hostreqkit.CombinedOutputFn
	defer func() { hostreqkit.CombinedOutputFn = origCombinedOutput }()
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return nil, fmt.Errorf("swapon: command not found")
	}
	if got := CompetingSwapsFn(); got != nil {
		t.Errorf("expected nil on swapon error; got %v", got)
	}
}
