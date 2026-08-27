package crashkernelreserve

import (
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/grub"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

type capturedCommand struct {
	Name string
	Args []string
}

const CrashkernelEnvVar = "reservation"

var testConfig map[string]string

var stubAll = crashkernelStubAll

func crashkernelStubAll(t *testing.T) (
	cmds *[]capturedCommand,
	files map[string]string,
	envValues map[string]string,
	cmdline *string,
	restore func(),
) {
	t.Helper()
	origReadProc := ReadProcCmdlineFn
	origRead := hostreqkit.ReadFileFn
	origRun := hostreqkit.RunCommandFn
	origCombined := hostreqkit.CombinedOutputFn
	origLookPath := hostreqkit.LookPathFn
	origWriteTemp := hostreqkit.WriteTempFileFn
	origNow := grub.NowFn
	origValidate := grub.ValidateGrubConfigFn
	origRoot := hostreqkit.RunningAsRootFn
	hostreqkit.RunningAsRootFn = func() bool { return true }

	captured := []capturedCommand{}
	fileContents := map[string]string{}
	tempContents := map[string]string{}
	env := map[string]string{}
	procCmdline := ""

	testConfig = env
	ReadProcCmdlineFn = func() (string, error) {
		if procCmdline == "" {
			return "", fs.ErrNotExist
		}
		return procCmdline, nil
	}
	hostreqkit.ReadFileFn = crashReadFile(fileContents)
	hostreqkit.RunCommandFn = crashRunCommand(&captured, fileContents, tempContents)
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		captured = append(captured, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		return nil, nil
	}
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" || name == "grub-script-check" {
			return "", fs.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}
	hostreqkit.WriteTempFileFn = crashWriteTemp(tempContents)
	grub.NowFn = func() time.Time {
		return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	}
	grub.ValidateGrubConfigFn = func(content string, opts hostreqkit.EnsureOptions) (bool, string) {
		return true, ""
	}

	return &captured, fileContents, env, &procCmdline, func() {
		testConfig = nil
		ReadProcCmdlineFn = origReadProc
		hostreqkit.ReadFileFn = origRead
		hostreqkit.RunCommandFn = origRun
		hostreqkit.CombinedOutputFn = origCombined
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.RunningAsRootFn = origRoot
		hostreqkit.WriteTempFileFn = origWriteTemp
		grub.NowFn = origNow
		grub.ValidateGrubConfigFn = origValidate
	}
}

func crashReadFile(files map[string]string) func(string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		if content, ok := files[path]; ok {
			return []byte(content), nil
		}
		return nil, fs.ErrNotExist
	}
}

func crashRunCommand(captured *[]capturedCommand, files, temps map[string]string) func(string, []string, hostreqkit.EnsureOptions) error {
	return func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		*captured = append(*captured, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		if name == "install" && len(args) >= 4 {
			if content, ok := temps[args[len(args)-2]]; ok {
				files[args[len(args)-1]] = content
			}
		}
		return nil
	}
}

func crashWriteTemp(temps map[string]string) func(string) (string, error) {
	sequence := 0
	return func(content string) (string, error) {
		sequence++
		path := "/tmp/vrooli-crashkernel-test-" + strings.Repeat("a", sequence)
		temps[path] = content
		return path, nil
	}
}

func newHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{Name: "crashkernel_reserve", Handler: "crashkernel_reserve"})
}

var linuxHost = crashkernelLinuxHost

func crashkernelLinuxHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", PackageManager: "apt-get", SupportsSysctl: true, SupportsSystemd: true}
}

func req(manual bool) hostreqspec.ResolvedRequirement {
	config := map[string]any{}
	if value := testConfig[CrashkernelEnvVar]; value != "" {
		config["reservation"] = value
	}
	return hostreqspec.ResolvedRequirement{
		Name: "crashkernel_reserve", Kind: hostreqspec.KindSafeguard, Required: true, Manual: manual,
		Config: config,
	}
}

func TestCrashkernelValueDefault(t *testing.T) {
	_, _, _, _, restore := stubAll(t)
	defer restore()
	if got := crashkernelValue(); got != DefaultCrashkernel {
		t.Errorf("default crashkernelValue() = %q, want %q", got, DefaultCrashkernel)
	}
}

func TestCrashkernelValueOverride(t *testing.T) {
	_, _, env, _, restore := stubAll(t)
	defer restore()
	env[CrashkernelEnvVar] = "768M"
	if got := crashkernelValue(map[string]any{"reservation": "768M"}); got != "768M" {
		t.Errorf("override crashkernelValue() = %q", got)
	}
}

func TestInspectAlreadyAppliedAndActive(t *testing.T) {
	_, files, _, cmdline, restore := stubAll(t)
	defer restore()
	files[grub.DefaultConfigPath] = `GRUB_CMDLINE_LINUX="quiet crashkernel=512M-:256M"` + "\n"
	*cmdline = "BOOT_IMAGE=/boot/vmlinuz crashkernel=512M-:256M quiet"

	st := newHandler().Inspect(linuxHost(), req(false))
	if !st.Applied {
		t.Errorf("expected Applied=true; status=%+v", st)
	}
	if st.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Errorf("ExecutionState = %q", st.ExecutionState)
	}
}

func TestInspectFileAppliedRebootRequired(t *testing.T) {
	_, files, _, cmdline, restore := stubAll(t)
	defer restore()
	files[grub.DefaultConfigPath] = `GRUB_CMDLINE_LINUX="quiet crashkernel=512M-:256M"` + "\n"
	*cmdline = "BOOT_IMAGE=/boot/vmlinuz quiet" // no crashkernel — pre-update-grub

	st := newHandler().Inspect(linuxHost(), req(false))
	if !st.Applied {
		t.Errorf("expected Applied=true (file written)")
	}
	if st.ExecutionState != hostreqkit.ExecutionRebootRequired {
		t.Errorf("ExecutionState = %q, want reboot_required", st.ExecutionState)
	}
}

func TestInspectPending(t *testing.T) {
	_, files, _, _, restore := stubAll(t)
	defer restore()
	files[grub.DefaultConfigPath] = `GRUB_CMDLINE_LINUX="quiet"` + "\n"

	st := newHandler().Inspect(linuxHost(), req(false))
	if st.Applied {
		t.Error("expected Applied=false")
	}
	if st.ExecutionState != hostreqkit.ExecutionPending {
		t.Errorf("ExecutionState = %q", st.ExecutionState)
	}
}

func TestInspectStaleValueIsPending(t *testing.T) {
	_, files, _, _, restore := stubAll(t)
	defer restore()
	files[grub.DefaultConfigPath] = `GRUB_CMDLINE_LINUX="quiet crashkernel=128M"` + "\n"

	st := newHandler().Inspect(linuxHost(), req(false))
	// File has crashkernel but with a different value than our default; we
	// should be Pending so Apply replaces it.
	if st.Applied {
		t.Errorf("Applied should be false when file has stale value; status=%+v", st)
	}
	if st.ExecutionState != hostreqkit.ExecutionPending {
		t.Errorf("ExecutionState = %q", st.ExecutionState)
	}
}

func TestApplyHappyPath(t *testing.T) {
	cmds, files, _, _, restore := stubAll(t)
	defer restore()
	files[grub.DefaultConfigPath] = `GRUB_CMDLINE_LINUX="quiet"` + "\n"

	st := newHandler().Inspect(linuxHost(), req(false))
	out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !out.Applied {
		t.Errorf("Applied = false; status=%+v", out)
	}
	if out.ExecutionState != hostreqkit.ExecutionRebootRequired {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
	for _, c := range *cmds {
		if c.Name == "update-grub" {
			t.Error("safeguard ran update-grub; must not")
		}
	}
}

func TestApplyOverrideValue(t *testing.T) {
	_, files, env, _, restore := stubAll(t)
	defer restore()
	env[CrashkernelEnvVar] = "768M"
	files[grub.DefaultConfigPath] = `GRUB_CMDLINE_LINUX="quiet"` + "\n"

	st := newHandler().Inspect(linuxHost(), req(false))
	out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionRebootRequired {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
	written := files[grub.DefaultConfigPath]
	if !strings.Contains(written, "crashkernel=768M") {
		t.Errorf("written config missing override value:\n%s", written)
	}
}

func TestApplyShortCircuitsOnSupportClasses(t *testing.T) {
	cases := []struct {
		name string
		sc   hostreqkit.SupportClass
		want hostreqkit.ExecutionState
	}{
		{"unsupported", hostreqkit.SupportUnsupported, hostreqkit.ExecutionUnsupported},
		{"not_applicable", hostreqkit.SupportNotApplicable, hostreqkit.ExecutionNotApplicable},
		{"manual_only", hostreqkit.SupportManualOnly, hostreqkit.ExecutionManualActionRequired},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cmds, _, _, _, restore := stubAll(t)
			defer restore()
			st := hostreqkit.ItemStatus{SupportClass: c.sc}
			out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{})
			if err != nil {
				t.Fatalf("Apply: %v", err)
			}
			if out.ExecutionState != c.want {
				t.Errorf("ExecutionState = %q, want %q", out.ExecutionState, c.want)
			}
			for _, cmd := range *cmds {
				if cmd.Name == "install" {
					t.Errorf("commands ran: %v", *cmds)
				}
			}
		})
	}
}
