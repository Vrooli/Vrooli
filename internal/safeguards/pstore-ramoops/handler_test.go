//nolint:goconst // test data deliberately reuses stable command fixtures.
package pstoreramoops

import (
	"errors"
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

const (
	MemAddressEnvVar = "mem_address"
	MemSizeEnvVar    = "mem_size"
	ECCEnvVar        = "ecc"
)

var testConfig map[string]string

var stubAll = pstoreRamoopsStubAll

func pstoreRamoopsStubAll(t *testing.T) (
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

	testConfig = ramoopsConfig(env)
	ReadProcCmdlineFn = ramoopsProcReader(&procCmdline)
	hostreqkit.ReadFileFn = ramoopsReadFile(fileContents)
	hostreqkit.RunCommandFn = ramoopsRunCommand(&captured, fileContents, tempContents)
	hostreqkit.CombinedOutputFn = ramoopsCombinedOutput(&captured)
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" || name == "grub-script-check" {
			return "", fs.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}
	hostreqkit.WriteTempFileFn = ramoopsWriteTemp(tempContents)
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

func ramoopsConfig(env map[string]string) map[string]string { return env }

func ramoopsProcReader(procCmdline *string) func() (string, error) {
	return func() (string, error) {
		if *procCmdline == "" {
			return "", fs.ErrNotExist
		}
		return *procCmdline, nil
	}
}

func ramoopsCombinedOutput(captured *[]capturedCommand) func(string, ...string) ([]byte, error) {
	return func(name string, args ...string) ([]byte, error) {
		*captured = append(*captured, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		return nil, nil
	}
}

func ramoopsReadFile(files map[string]string) func(string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		if content, ok := files[path]; ok {
			return []byte(content), nil
		}
		return nil, fs.ErrNotExist
	}
}

func ramoopsRunCommand(captured *[]capturedCommand, files, temps map[string]string) func(string, []string, hostreqkit.EnsureOptions) error {
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

func ramoopsWriteTemp(temps map[string]string) func(string) (string, error) {
	sequence := 0
	return func(content string) (string, error) {
		sequence++
		path := "/tmp/vrooli-pstore-ramoops-test-" + strings.Repeat("a", sequence)
		temps[path] = content
		return path, nil
	}
}

func newHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{Name: "pstore_ramoops", Handler: "pstore_ramoops"})
}

var linuxHost = pstoreRamoopsLinuxHost

func pstoreRamoopsLinuxHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", PackageManager: "apt-get", SupportsSysctl: true, SupportsSystemd: true}
}

func req(manual bool) hostreqspec.ResolvedRequirement {
	config := map[string]any{}
	for key, value := range testConfig {
		config[key] = value
	}
	return hostreqspec.ResolvedRequirement{
		Name: "pstore_ramoops", Kind: hostreqspec.KindSafeguard, Required: true, Manual: manual,
		Config: config,
	}
}

func setEnv(env map[string]string, addr, size string) {
	env[MemAddressEnvVar] = addr
	env[MemSizeEnvVar] = size
}

func TestInspectMissingEnvIsNotApplicable(t *testing.T) {
	_, _, _, _, restore := stubAll(t)
	defer restore()
	st := newHandler().Inspect(linuxHost(), req(false))
	if st.SupportClass != hostreqkit.SupportNotApplicable {
		t.Errorf("SupportClass = %q", st.SupportClass)
	}
	if st.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Errorf("ExecutionState = %q", st.ExecutionState)
	}
	notes := strings.Join(st.Notes, " | ")
	if !strings.Contains(notes, MemAddressEnvVar) || !strings.Contains(notes, MemSizeEnvVar) {
		t.Errorf("note should mention both env vars: %q", notes)
	}
}

func TestInspectPartialEnvIsNotApplicable(t *testing.T) {
	_, _, env, _, restore := stubAll(t)
	defer restore()
	env[MemAddressEnvVar] = "0x70000000"
	// MemSizeEnvVar deliberately unset.
	st := newHandler().Inspect(linuxHost(), req(false))
	if st.SupportClass != hostreqkit.SupportNotApplicable {
		t.Errorf("SupportClass = %q (partial env should still be NotApplicable)", st.SupportClass)
	}
}

func TestInspectAppliedRamoopsActivationState(t *testing.T) {
	for _, tc := range []struct {
		name          string
		cmdline       string
		expectedState hostreqkit.ExecutionState
	}{
		{name: "active", cmdline: "BOOT_IMAGE=/boot/vmlinuz quiet ramoops.mem_address=0x70000000 ramoops.mem_size=0x100000 ramoops.ecc=1", expectedState: hostreqkit.ExecutionAlreadyPresent},
		{name: "reboot_required", cmdline: "BOOT_IMAGE=/boot/vmlinuz quiet", expectedState: hostreqkit.ExecutionRebootRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, files, env, cmdline, restore := stubAll(t)
			defer restore()
			setEnv(env, "0x70000000", "0x100000")
			files[grub.DefaultConfigPath] = `GRUB_CMDLINE_LINUX="quiet ramoops.mem_address=0x70000000 ramoops.mem_size=0x100000 ramoops.ecc=1"` + "\n"
			*cmdline = tc.cmdline

			st := newHandler().Inspect(linuxHost(), req(false))
			if !st.Applied {
				t.Errorf("expected Applied=true; status=%+v", st)
			}
			if st.ExecutionState != tc.expectedState {
				t.Errorf("ExecutionState = %q, want %q", st.ExecutionState, tc.expectedState)
			}
		})
	}
}

func TestInspectPendingWhenFileMissing(t *testing.T) {
	_, files, env, _, restore := stubAll(t)
	defer restore()
	setEnv(env, "0x70000000", "0x100000")
	files[grub.DefaultConfigPath] = `GRUB_CMDLINE_LINUX="quiet"` + "\n"

	st := newHandler().Inspect(linuxHost(), req(false))
	if st.Applied {
		t.Error("expected Applied=false")
	}
	if st.ExecutionState != hostreqkit.ExecutionPending {
		t.Errorf("ExecutionState = %q, want pending", st.ExecutionState)
	}
}

func TestApplyHappyPathReturnsRebootRequired(t *testing.T) {
	cmds, files, env, _, restore := stubAll(t)
	defer restore()
	setEnv(env, "0x70000000", "0x100000")
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
		t.Errorf("ExecutionState = %q, want reboot_required", out.ExecutionState)
	}
	notes := strings.Join(out.Notes, " | ")
	if !strings.Contains(notes, "update-grub") {
		t.Errorf("note should instruct operator to run update-grub: %q", notes)
	}
	// Two install calls expected: backup + new content.
	installCount := 0
	for _, c := range *cmds {
		if c.Name == "install" {
			installCount++
		}
	}
	if installCount != 2 {
		t.Errorf("install count = %d, want 2 (backup + new); commands=%v", installCount, *cmds)
	}
	// Verify no update-grub was run — that's the operator's job.
	for _, c := range *cmds {
		if c.Name == "update-grub" || (c.Name == "sudo" && len(c.Args) > 0 && c.Args[0] == "update-grub") {
			t.Errorf("safeguard ran update-grub, must not: %v", c)
		}
	}
}

func TestApplyValidationFailureSurfaced(t *testing.T) {
	_, files, env, _, restore := stubAll(t)
	defer restore()
	setEnv(env, "0x70000000", "0x100000")
	files[grub.DefaultConfigPath] = `GRUB_CMDLINE_LINUX="quiet"` + "\n"
	grub.ValidateGrubConfigFn = func(content string, opts hostreqkit.EnsureOptions) (bool, string) {
		return false, "synthetic syntax error"
	}

	st := newHandler().Inspect(linuxHost(), req(false))
	out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionFailed {
		t.Errorf("ExecutionState = %q, want failed", out.ExecutionState)
	}
	notes := strings.Join(out.Notes, " | ")
	if !strings.Contains(notes, "synthetic syntax error") {
		t.Errorf("notes missing root cause: %q", notes)
	}
}

func TestApplyBackupFailureBlocksWrite(t *testing.T) {
	cmds, files, env, _, restore := stubAll(t)
	defer restore()
	setEnv(env, "0x70000000", "0x100000")
	files[grub.DefaultConfigPath] = `GRUB_CMDLINE_LINUX="quiet"` + "\n"

	// Track install calls and fail the first.
	calls := 0
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		*cmds = append(*cmds, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		if name == "install" {
			calls++
			if calls == 1 {
				return errors.New("synthetic backup failure")
			}
		}
		return nil
	}

	st := newHandler().Inspect(linuxHost(), req(false))
	out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionFailed {
		t.Errorf("ExecutionState = %q, want failed", out.ExecutionState)
	}
	if calls != 1 {
		t.Errorf("install attempts = %d, want 1 (second install must not run after backup failure)", calls)
	}
}

func TestApplyIdempotentFileAppliedNoCmdline(t *testing.T) {
	cmds, files, env, _, restore := stubAll(t)
	defer restore()
	setEnv(env, "0x70000000", "0x100000")
	// File already has all three params; /proc/cmdline does not (no reboot yet).
	files[grub.DefaultConfigPath] = `GRUB_CMDLINE_LINUX="quiet ramoops.mem_address=0x70000000 ramoops.mem_size=0x100000 ramoops.ecc=1"` + "\n"

	st := newHandler().Inspect(linuxHost(), req(false))
	// Inspect already returns RebootRequired — Apply should respect that.
	if st.ExecutionState != hostreqkit.ExecutionRebootRequired {
		t.Fatalf("Inspect should return RebootRequired; got %q", st.ExecutionState)
	}

	out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionRebootRequired {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
	for _, c := range *cmds {
		if c.Name == "install" {
			t.Errorf("idempotent re-run wrote files: %v", c)
		}
	}
}

func TestECCDefaultAndOverride(t *testing.T) {
	_, _, env, _, restore := stubAll(t)
	defer restore()
	if got := configECC(nil); got != "1" {
		t.Errorf("default ecc() = %q, want 1", got)
	}
	env[ECCEnvVar] = "16"
	if got := configECC(map[string]any{"ecc": "16"}); got != "16" {
		t.Errorf("override ecc() = %q, want 16", got)
	}
}

func TestContainsToken(t *testing.T) {
	cases := map[string]map[string]bool{
		"BOOT_IMAGE=/x quiet ramoops.ecc=1": {
			"ramoops.ecc=1":    true,
			"ramoops.eccfoo=1": false,
			"quiet":            true,
			"missing":          false,
		},
	}
	for cmdline, want := range cases {
		for token, expected := range want {
			if got := containsToken(cmdline, token); got != expected {
				t.Errorf("containsToken(%q, %q) = %v, want %v", cmdline, token, got, expected)
			}
		}
	}
}
