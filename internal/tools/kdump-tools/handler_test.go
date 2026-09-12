//nolint:goconst // test data deliberately reuses stable command fixtures.
package kdumptools

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

type capturedCommand struct {
	Name string
	Args []string
}

var stubAll = kdumpToolsStubAll

func kdumpToolsStubAll(t *testing.T) (cmds *[]capturedCommand, cmdline *string, debconfInputs *[]string, restore func()) {
	t.Helper()
	origRun := hostreqkit.RunCommandFn
	origCombined := hostreqkit.CombinedOutputFn
	origInput := hostreqkit.CombinedOutputInputFn
	origLookPath := hostreqkit.LookPathFn
	origElevationFacts := hostreqkit.ElevationFactsFn
	origReadProc := ReadProcCmdlineFn
	origServiceState := KdumpServiceStateFn
	origConfigStatus := KdumpConfigStatusFn
	origSysrq := SysrqEnabledFn

	captured := []capturedCommand{}
	procCmdline := ""
	inputs := []string{}
	hostreqkit.ElevationFactsFn = func() hostreqkit.ElevationFacts {
		return hostreqkit.ElevationFacts{Platform: "linux", CanElevate: true, Mechanism: "test"}
	}

	// Default: service armed, kdump-config ready, sysrq enabled. Tests that
	// exercise the unhappy paths override these.
	KdumpServiceStateFn = func() (bool, bool) { return true, true }
	KdumpConfigStatusFn = func() (bool, string) { return true, "current state    : ready to kdump" }
	SysrqEnabledFn = func() bool { return true }

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		captured = append(captured, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		return nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		captured = append(captured, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		return nil, nil
	}
	hostreqkit.CombinedOutputInputFn = func(name, input string, args ...string) ([]byte, error) {
		captured = append(captured, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		inputs = append(inputs, input)
		return nil, nil
	}
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" || name == "kdump-config" {
			return "", fs.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}
	ReadProcCmdlineFn = func() (string, error) {
		if procCmdline == "" {
			return "", fs.ErrNotExist
		}
		return procCmdline, nil
	}

	return &captured, &procCmdline, &inputs, func() {
		hostreqkit.RunCommandFn = origRun
		hostreqkit.CombinedOutputFn = origCombined
		hostreqkit.CombinedOutputInputFn = origInput
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ElevationFactsFn = origElevationFacts
		ReadProcCmdlineFn = origReadProc
		KdumpServiceStateFn = origServiceState
		KdumpConfigStatusFn = origConfigStatus
		SysrqEnabledFn = origSysrq
	}
}

func newHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.ToolManifest{
		Name:           "kdump-tools",
		Handler:        "kdump_tools",
		Commands:       []string{"kdump-config"},
		VersionArgs:    []string{"show"},
		DefaultPackage: "kdump-tools",
	})
}

func aptHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", PackageManager: "apt-get", SupportsSystemd: true}
}

func req(manual bool) hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name: "kdump-tools", Kind: hostreqspec.KindTool, Required: true, Manual: manual,
	}
}

func setInstalled() {
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "kdump-config" {
			return "/usr/sbin/kdump-config", nil
		}
		if name == "sudo" {
			return "", fs.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}
}

func TestInspectNonLinux(t *testing.T) {
	_, _, _, restore := stubAll(t)
	defer restore()
	st := newHandler().Inspect(hostreqkit.Host{OS: "darwin"}, req(false))
	if st.SupportClass != hostreqkit.SupportUnsupported {
		t.Errorf("SupportClass = %q", st.SupportClass)
	}
}

func TestInspectNonAptUnsupported(t *testing.T) {
	_, _, _, restore := stubAll(t)
	defer restore()
	host := hostreqkit.Host{OS: "linux", PackageManager: "dnf", SupportsSystemd: true}
	st := newHandler().Inspect(host, req(false))
	if st.SupportClass != hostreqkit.SupportUnsupported {
		t.Errorf("SupportClass = %q", st.SupportClass)
	}
}

func TestInspectNotInstalled(t *testing.T) {
	_, _, _, restore := stubAll(t)
	defer restore()
	st := newHandler().Inspect(aptHost(), req(false))
	if st.Installed {
		t.Error("expected Installed=false")
	}
	if st.ExecutionState != hostreqkit.ExecutionPending {
		t.Errorf("ExecutionState = %q", st.ExecutionState)
	}
}

func TestInspectInstalledNoCrashkernelIsRebootRequired(t *testing.T) {
	_, cmdline, _, restore := stubAll(t)
	defer restore()
	setInstalled()
	*cmdline = "BOOT_IMAGE=/boot/vmlinuz quiet"

	st := newHandler().Inspect(aptHost(), req(false))
	if st.ExecutionState != hostreqkit.ExecutionRebootRequired {
		t.Errorf("ExecutionState = %q, want reboot_required", st.ExecutionState)
	}
	notes := strings.Join(st.Notes, " | ")
	if !strings.Contains(notes, "crashkernel_reserve") {
		t.Errorf("note should redirect to crashkernel_reserve safeguard: %q", notes)
	}
}

func TestInspectInstalledWithCrashkernelIsAlreadyPresent(t *testing.T) {
	_, cmdline, _, restore := stubAll(t)
	defer restore()
	setInstalled()
	*cmdline = "BOOT_IMAGE=/boot/vmlinuz crashkernel=512M-:256M quiet"

	st := newHandler().Inspect(aptHost(), req(false))
	if st.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Errorf("ExecutionState = %q", st.ExecutionState)
	}
}

func TestApplyHappyPathPostReboot(t *testing.T) {
	_, cmdline, debconfInputs, restore := stubAll(t)
	defer restore()
	*cmdline = "BOOT_IMAGE=/boot/vmlinuz crashkernel=512M-:256M quiet"

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		if name == "env" {
			// Simulate apt install completing.
			setInstalled()
		}
		return nil
	}

	st := newHandler().Inspect(aptHost(), req(false))
	out, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
	if len(*debconfInputs) != 1 {
		t.Errorf("debconf preseed not called exactly once: got %d inputs", len(*debconfInputs))
	}
	if len(*debconfInputs) > 0 && !strings.Contains((*debconfInputs)[0], "kdump-tools/use_kdump boolean true") {
		t.Errorf("debconf input wrong: %q", (*debconfInputs)[0])
	}
}

func TestApplyHappyPathPreReboot(t *testing.T) {
	_, _, _, restore := stubAll(t)
	defer restore()
	// crashkernel NOT in /proc/cmdline — operator hasn't rebooted after
	// crashkernel_reserve safeguard. Apply should still install but report
	// reboot_required.

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		if name == "env" {
			setInstalled()
		}
		return nil
	}

	st := newHandler().Inspect(aptHost(), req(false))
	out, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionRebootRequired {
		t.Errorf("ExecutionState = %q, want reboot_required", out.ExecutionState)
	}
}

func TestApplyDryRunNotInstalled(t *testing.T) {
	cmds, _, _, restore := stubAll(t)
	defer restore()

	st := newHandler().Inspect(aptHost(), req(false))
	out, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionWouldInstall {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
	for _, c := range *cmds {
		if c.Name == "env" || c.Name == "apt-get" {
			t.Errorf("DryRun ran %s: %v", c.Name, c)
		}
	}
}

func TestApplyDebconfFailureSurfaced(t *testing.T) {
	_, _, _, restore := stubAll(t)
	defer restore()

	hostreqkit.CombinedOutputInputFn = func(name, input string, args ...string) ([]byte, error) {
		return nil, errors.New("synthetic debconf failure")
	}

	st := newHandler().Inspect(aptHost(), req(false))
	out, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionFailed {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
	if !strings.Contains(strings.Join(out.Notes, " | "), "synthetic debconf failure") {
		t.Errorf("notes missing root cause: %v", out.Notes)
	}
}

func TestApplyAlreadyInstalledShortCircuits(t *testing.T) {
	cmds, cmdline, _, restore := stubAll(t)
	defer restore()
	setInstalled()
	*cmdline = "BOOT_IMAGE=/boot/vmlinuz crashkernel=512M-:256M quiet"

	st := newHandler().Inspect(aptHost(), req(false))
	if st.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("setup: Inspect should return AlreadyPresent")
	}
	out, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
	for _, c := range *cmds {
		if c.Name == "env" || c.Name == "debconf-set-selections" {
			t.Errorf("short-circuit ran %s: %v", c.Name, c)
		}
	}
}

func TestApplySudoSkipReturnsTypedSentinel(t *testing.T) {
	_, _, _, restore := stubAll(t)
	defer restore()

	// sudo IS available — that's the only way --sudo-mode=skip is meaningful.
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		if name == "kdump-config" {
			return "", fs.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}

	st := newHandler().Inspect(aptHost(), req(false))
	out, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{SudoMode: "skip"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionFailed {
		t.Errorf("ExecutionState = %q, want failed", out.ExecutionState)
	}
	if !strings.Contains(strings.Join(out.Notes, " | "), "sudo skipped") {
		t.Errorf("notes should mention sudo: %v", out.Notes)
	}
}

func TestApplySudoModePreseedRoutesThroughSudo(t *testing.T) {
	cmds, cmdline, _, restore := stubAll(t)
	defer restore()
	*cmdline = "BOOT_IMAGE=/boot/vmlinuz crashkernel=512M-:256M quiet"

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		if name == "kdump-config" {
			return "", fs.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		*cmds = append(*cmds, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		if name == "sudo" {
			// Apt install completes — flip kdump-config availability.
			hostreqkit.LookPathFn = func(name string) (string, error) {
				if name == "kdump-config" {
					return "/usr/sbin/kdump-config", nil
				}
				if name == "sudo" {
					return "/usr/bin/sudo", nil
				}
				return "/usr/bin/" + name, nil
			}
		}
		return nil
	}

	st := newHandler().Inspect(aptHost(), req(false))
	if _, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// Preseed should have been invoked under sudo: first captured command for
	// debconf-set-selections is `sudo debconf-set-selections`.
	var found bool
	for _, c := range *cmds {
		if c.Name == "sudo" && len(c.Args) > 0 && c.Args[0] == "debconf-set-selections" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected `sudo debconf-set-selections` in commands, got %+v", *cmds)
	}
}

// --- New: arming probes (Phase 2 of 2026-05-07 host-stability work) -------

func TestInspectInstalledServiceDisabledIsPending(t *testing.T) {
	_, cmdline, _, restore := stubAll(t)
	defer restore()
	*cmdline = "BOOT_IMAGE=/vmlinuz crashkernel=512M ro quiet"
	setInstalled()
	KdumpServiceStateFn = func() (bool, bool) { return false, false }

	st := newHandler().Inspect(aptHost(), req(false))
	if st.ExecutionState != hostreqkit.ExecutionPending {
		t.Fatalf("ExecutionState = %q, want Pending; notes: %v", st.ExecutionState, st.Notes)
	}
	if !containsNote(st.Notes, "service not enabled") || !containsNote(st.Notes, "service not active") {
		t.Errorf("expected notes to call out enabled+active gaps; got %v", st.Notes)
	}
}

func TestInspectInstalledKdumpConfigNotReadyAddsWarning(t *testing.T) {
	_, cmdline, _, restore := stubAll(t)
	defer restore()
	*cmdline = "crashkernel=512M"
	setInstalled()
	KdumpConfigStatusFn = func() (bool, string) {
		return false, "current state    : not ready (no capture kernel loaded)"
	}

	st := newHandler().Inspect(aptHost(), req(false))
	// Service is armed; we still report AlreadyPresent but surface the warning.
	if st.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q, want AlreadyPresent", st.ExecutionState)
	}
	if !containsNote(st.Notes, "kdump-config status` reports not-ready") {
		t.Errorf("expected kdump-config diagnostic note; got %v", st.Notes)
	}
}

func TestInspectInstalledSysrqDisabledAddsInformationalNote(t *testing.T) {
	_, cmdline, _, restore := stubAll(t)
	defer restore()
	*cmdline = "crashkernel=512M"
	setInstalled()
	SysrqEnabledFn = func() bool { return false }

	st := newHandler().Inspect(aptHost(), req(false))
	if st.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q, want AlreadyPresent", st.ExecutionState)
	}
	if !containsNote(st.Notes, "kernel.sysrq is disabled") {
		t.Errorf("expected sysrq note; got %v", st.Notes)
	}
}

func TestApplyArmsServiceWhenInstalledButDisabled(t *testing.T) {
	cmds, cmdline, _, restore := stubAll(t)
	defer restore()
	*cmdline = "crashkernel=512M"
	setInstalled()

	// First probe: not armed. After Apply runs `systemctl enable --now`, flip
	// to armed so the post-arm re-probe passes.
	state := struct{ enabled, active bool }{false, false}
	KdumpServiceStateFn = func() (bool, bool) { return state.enabled, state.active }

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		*cmds = append(*cmds, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		if name == "systemctl" && len(args) >= 3 && args[0] == "enable" && args[1] == "--now" && args[2] == ServiceName {
			state.enabled, state.active = true, true
		}
		return nil
	}

	st := newHandler().Inspect(aptHost(), req(false))
	if st.ExecutionState != hostreqkit.ExecutionPending {
		t.Fatalf("Inspect ExecutionState = %q, want Pending", st.ExecutionState)
	}

	hostreqkit.ElevationFactsFn = func() hostreqkit.ElevationFacts {
		return hostreqkit.ElevationFacts{Platform: "linux", Elevated: true, CanElevate: true, Mechanism: "test"}
	}
	st, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{SudoMode: "skip"})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if st.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Fatalf("Apply ExecutionState = %q, want Installed; notes: %v", st.ExecutionState, st.Notes)
	}

	found := false
	for _, c := range *cmds {
		if c.Name == "systemctl" && len(c.Args) >= 3 && c.Args[0] == "enable" && c.Args[1] == "--now" && c.Args[2] == ServiceName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected systemctl enable --now %s in commands; got %+v", ServiceName, *cmds)
	}
}

func TestApplyArmingFailureSurfacesFailed(t *testing.T) {
	_, cmdline, _, restore := stubAll(t)
	defer restore()
	*cmdline = "crashkernel=512M"
	setInstalled()
	KdumpServiceStateFn = func() (bool, bool) { return false, false }

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		return errors.New("masked")
	}

	st := newHandler().Inspect(aptHost(), req(false))
	st, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{SudoMode: "skip"})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if st.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("ExecutionState = %q, want Failed", st.ExecutionState)
	}
}

func TestApplyDryRunSurfacesArmingIntent(t *testing.T) {
	_, cmdline, _, restore := stubAll(t)
	defer restore()
	*cmdline = "crashkernel=512M"
	setInstalled()
	KdumpServiceStateFn = func() (bool, bool) { return false, false }

	st := newHandler().Inspect(aptHost(), req(false))
	st, err := newHandler().Apply(aptHost(), st, hostreqkit.EnsureOptions{DryRun: true, SudoMode: "skip"})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if st.ExecutionState != hostreqkit.ExecutionWouldApply {
		t.Fatalf("ExecutionState = %q, want WouldApply", st.ExecutionState)
	}
}

func containsNote(notes []string, want string) bool {
	for _, n := range notes {
		if strings.Contains(n, want) {
			return true
		}
	}
	return false
}
