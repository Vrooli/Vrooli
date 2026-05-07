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

func stubAll(t *testing.T) (cmds *[]capturedCommand, cmdline *string, debconfInputs *[]string, restore func()) {
	t.Helper()
	origRun := hostreqkit.RunCommandFn
	origCombined := hostreqkit.CombinedOutputFn
	origInput := hostreqkit.CombinedOutputInputFn
	origLookPath := hostreqkit.LookPathFn
	origReadProc := ReadProcCmdlineFn

	captured := []capturedCommand{}
	procCmdline := ""
	inputs := []string{}

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
		ReadProcCmdlineFn = origReadProc
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

func TestNameAndKind(t *testing.T) {
	h := newHandler()
	if h.Name() != "kdump-tools" {
		t.Errorf("Name = %q", h.Name())
	}
	if h.Kind() != hostreqspec.KindTool {
		t.Errorf("Kind = %q", h.Kind())
	}
}
