package netconsole

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/modules"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const netconsoleModprobe = "modprobe"

type capturedCommand struct {
	Name string
	Args []string
}

const TargetEnvVar = "target"

var testConfig map[string]string

// stubAll swaps every shared seam (env, FS, exec, modules.Stat) for fakes
// and returns a restore func plus access to the capture state.
var stubAll = netconsoleStubAll

func netconsoleStubAll(t *testing.T) (
	cmds *[]capturedCommand,
	files map[string]string,
	envValues map[string]string,
	restore func(),
) {
	t.Helper()
	origRead := hostreqkit.ReadFileFn
	origRun := hostreqkit.RunCommandFn
	origLookPath := hostreqkit.LookPathFn
	origWriteTemp := hostreqkit.WriteTempFileFn
	origStat := modules.StatFn
	origRoot := hostreqkit.RunningAsRootFn
	hostreqkit.RunningAsRootFn = func() bool { return true }

	captured := []capturedCommand{}
	fileContents := map[string]string{}
	env := map[string]string{}
	tempContents := map[string]string{}

	testConfig = env

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if c, ok := fileContents[path]; ok {
			return []byte(c), nil
		}
		return nil, fs.ErrNotExist
	}

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "", fs.ErrNotExist // disable sudo wrapping in tests
		}
		return "/usr/bin/" + name, nil
	}

	hostreqkit.WriteTempFileFn = func(content string) (string, error) {
		path := "/tmp/vrooli-netconsole-test-" + content[:1] // crude but unique-ish
		tempContents[path] = content
		return path, nil
	}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		captured = append(captured, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		// Simulate `install -m 644 <tmp> <dst>`.
		if name == "install" && len(args) >= 4 {
			tmp := args[len(args)-2]
			dst := args[len(args)-1]
			if c, ok := tempContents[tmp]; ok {
				fileContents[dst] = c
			}
		}
		return nil
	}

	modules.StatFn = func(path string) (os.FileInfo, error) {
		// Default: module not loaded. Tests opt back in.
		return nil, fs.ErrNotExist
	}

	return &captured, fileContents, env, func() {
		testConfig = nil
		hostreqkit.ReadFileFn = origRead
		hostreqkit.RunCommandFn = origRun
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.RunningAsRootFn = origRoot
		hostreqkit.WriteTempFileFn = origWriteTemp
		modules.StatFn = origStat
	}
}

func newHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{Name: "netconsole", Handler: "netconsole"})
}

var linuxHost = netconsoleLinuxHost

func netconsoleLinuxHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", PackageManager: "apt-get", SupportsSysctl: true, SupportsSystemd: true}
}

func req(manual bool) hostreqspec.ResolvedRequirement {
	config := map[string]any{}
	if target := testConfig[TargetEnvVar]; target != "" {
		config["target"] = target
	}
	return hostreqspec.ResolvedRequirement{
		Name:     "netconsole",
		Kind:     hostreqspec.KindSafeguard,
		Required: true,
		Manual:   manual,
		Config:   config,
	}
}

func TestInspectMissingTargetIsNotApplicable(t *testing.T) {
	_, _, _, restore := stubAll(t)
	defer restore()
	// env[TargetEnvVar] not set

	st := newHandler().Inspect(linuxHost(), req(false))
	if st.SupportClass != hostreqkit.SupportNotApplicable {
		t.Errorf("SupportClass = %q, want not_applicable", st.SupportClass)
	}
	if st.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Errorf("ExecutionState = %q", st.ExecutionState)
	}
	notes := strings.Join(st.Notes, " | ")
	if !strings.Contains(notes, TargetEnvVar) {
		t.Errorf("note should reference %s; got %q", TargetEnvVar, notes)
	}
}

func TestInspectAlreadyAppliedWhenAllPresent(t *testing.T) {
	_, files, env, restore := stubAll(t)
	defer restore()
	target := "6666@10.0.0.5/dev,6666@10.0.0.6/00:11:22:33:44:55"
	env[TargetEnvVar] = target
	files[modules.LoadFilePath(ModuleName)] = expectedLoadContent()
	files[modules.OptionsFilePath(ModuleName)] = expectedOptionsContent(target)
	modules.StatFn = func(path string) (os.FileInfo, error) {
		if path == "/sys/module/netconsole" {
			return nil, nil
		}
		return nil, fs.ErrNotExist
	}

	st := newHandler().Inspect(linuxHost(), req(false))
	if !st.Applied {
		t.Errorf("expected Applied=true; got %+v", st)
	}
	if st.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Errorf("ExecutionState = %q", st.ExecutionState)
	}
}

func TestInspectPendingWhenModuleNotLoaded(t *testing.T) {
	_, files, env, restore := stubAll(t)
	defer restore()
	target := "x"
	env[TargetEnvVar] = target
	files[modules.LoadFilePath(ModuleName)] = expectedLoadContent()
	files[modules.OptionsFilePath(ModuleName)] = expectedOptionsContent(target)
	// modules.StatFn defaults to "not loaded".

	st := newHandler().Inspect(linuxHost(), req(false))
	if st.Applied {
		t.Error("expected Applied=false (module not loaded)")
	}
	if st.ExecutionState != hostreqkit.ExecutionPending {
		t.Errorf("ExecutionState = %q, want pending", st.ExecutionState)
	}
	notes := strings.Join(st.Notes, " | ")
	if !strings.Contains(notes, "/sys/module/netconsole") {
		t.Errorf("pending note should call out missing module: %q", notes)
	}
}

func checkApplyInstallsAndLoadsNetconsole(t *testing.T) {
	cmds, _, env, restore := stubAll(t)
	defer restore()
	target := "6666@10.0.0.5/dev,6666@10.0.0.6/00:11:22:33:44:55"
	env[TargetEnvVar] = target

	st := newHandler().Inspect(linuxHost(), req(false))
	if st.Applied {
		t.Fatal("Inspect should not report Applied for fresh state")
	}

	out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !out.Applied {
		t.Errorf("Applied = false, want true; status=%+v", out)
	}
	if out.ExecutionState != hostreqkit.ExecutionApplied {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
	// Expect at least one install (load file) plus one modprobe.
	sawInstall := false
	sawModprobe := false
	for _, c := range *cmds {
		if c.Name == "install" {
			sawInstall = true
		}
		if c.Name == netconsoleModprobe {
			sawModprobe = true
		}
	}
	if !sawInstall {
		t.Error("no install command captured")
	}
	if !sawModprobe {
		t.Error("no modprobe command captured")
	}
}

func TestApplyDryRunReportsWouldApply(t *testing.T) {
	cmds, _, env, restore := stubAll(t)
	defer restore()
	env[TargetEnvVar] = "x"

	st := newHandler().Inspect(linuxHost(), req(false))
	out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionWouldApply {
		t.Errorf("ExecutionState = %q, want would_apply", out.ExecutionState)
	}
	if len(*cmds) != 0 {
		t.Errorf("DryRun ran commands: %v", *cmds)
	}
}

func TestApplyNotApplicableShortCircuits(t *testing.T) {
	cmds, _, _, restore := stubAll(t)
	defer restore()
	st := hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportNotApplicable}

	out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Errorf("ExecutionState = %q", out.ExecutionState)
	}
	if len(*cmds) != 0 {
		t.Errorf("not_applicable ran commands: %v", *cmds)
	}
}

func TestApplyModprobeFailureSurfacedAsFailed(t *testing.T) {
	_, _, env, restore := stubAll(t)
	defer restore()
	env[TargetEnvVar] = "x"

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		if name == netconsoleModprobe {
			return errors.New("synthetic modprobe failure")
		}
		// Allow installs through (they don't matter for this test).
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
	if out.Applied {
		t.Error("Applied should be false after modprobe failure")
	}
	notes := strings.Join(out.Notes, " | ")
	if !strings.Contains(notes, "synthetic modprobe failure") {
		t.Errorf("note should chain modprobe error: %q", notes)
	}
}

func TestApplySkipsModprobeWhenAlreadyLoaded(t *testing.T) {
	cmds, _, env, restore := stubAll(t)
	defer restore()
	env[TargetEnvVar] = "x"
	// Pretend module is already loaded — Apply must still run install (config
	// might be stale) but skip modprobe.
	modules.StatFn = func(path string) (os.FileInfo, error) {
		if path == "/sys/module/netconsole" {
			return nil, nil
		}
		return nil, fs.ErrNotExist
	}

	st := newHandler().Inspect(linuxHost(), req(false))
	if _, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	for _, c := range *cmds {
		if c.Name == netconsoleModprobe {
			t.Errorf("modprobe should be skipped when module already loaded; commands=%v", *cmds)
		}
	}
}
