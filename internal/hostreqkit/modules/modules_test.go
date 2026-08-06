package modules

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

type capturedCommand struct {
	Name string
	Args []string
}

// stubAll swaps every package-level seam used by EnsureLoadAtBoot/Modprobe
// for capturing fakes and returns a restore func plus accessors.
func stubAll(t *testing.T) (cmds *[]capturedCommand, files map[string]string, restore func()) {
	t.Helper()
	origRead := hostreqkit.ReadFileFn
	origRun := hostreqkit.RunCommandFn
	origCombined := hostreqkit.CombinedOutputFn
	origLookPath := hostreqkit.LookPathFn
	origWriteTemp := hostreqkit.WriteTempFileFn
	origStat := StatFn
	origElevation := hostreqkit.ElevationFactsFn
	hostreqkit.ElevationFactsFn = func() hostreqkit.ElevationFacts {
		return hostreqkit.ElevationFacts{Platform: "linux", Elevated: true, CanElevate: true, Mechanism: "test"}
	}

	captured := []capturedCommand{}
	fileContents := map[string]string{}

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if c, ok := fileContents[path]; ok {
			return []byte(c), nil
		}
		return nil, fs.ErrNotExist
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		captured = append(captured, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		// Simulate `install -m 0644 <tmp> <dst>` by capturing the tempfile
		// contents under the destination path. This lets later FileContentMatches
		// calls observe the write.
		if name == "install" && len(args) >= 4 {
			tmp := args[len(args)-2]
			dst := args[len(args)-1]
			if c, ok := tempContents[tmp]; ok {
				fileContents[dst] = c
			}
		}
		return nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		captured = append(captured, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		return nil, nil
	}
	hostreqkit.LookPathFn = func(name string) (string, error) {
		// Disable sudo wrapping in tests — without sudo on PATH, WithSudo
		// returns the unwrapped command and the captured Name field is the
		// real command (install / modprobe / mkdir), making assertions clean.
		if name == "sudo" {
			return "", fs.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}
	tempContents = map[string]string{}
	hostreqkit.WriteTempFileFn = func(content string) (string, error) {
		path := "/tmp/vrooli-modules-" + time.Now().Format("150405.000000000")
		tempContents[path] = content
		return path, nil
	}
	StatFn = func(path string) (os.FileInfo, error) {
		return nil, fs.ErrNotExist
	}

	return &captured, fileContents, func() {
		hostreqkit.ReadFileFn = origRead
		hostreqkit.ElevationFactsFn = origElevation
		hostreqkit.RunCommandFn = origRun
		hostreqkit.CombinedOutputFn = origCombined
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.WriteTempFileFn = origWriteTemp
		StatFn = origStat
	}
}

// tempContents is populated by the WriteTempFileFn stub so RunCommandFn can
// "install" tempfile content into the in-memory filesystem map.
var tempContents map[string]string

func TestIsLoadedTrueWhenSysModuleExists(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	StatFn = func(path string) (os.FileInfo, error) {
		if path == "/sys/module/netconsole" {
			return nil, nil
		}
		return nil, fs.ErrNotExist
	}

	if !IsLoaded("netconsole") {
		t.Error("expected IsLoaded=true")
	}
	if IsLoaded("missing") {
		t.Error("expected IsLoaded=false for missing module")
	}
}

func TestIsLoadedFalseOnEmptyName(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	if IsLoaded("") {
		t.Error("empty name should be IsLoaded=false")
	}
	if IsLoaded("  \t") {
		t.Error("whitespace name should be IsLoaded=false")
	}
}

func TestEnsureLoadAtBootWritesBothFiles(t *testing.T) {
	cmds, files, restore := stubAll(t)
	defer restore()

	out, err := EnsureLoadAtBoot(
		"netconsole",
		map[string]string{"netconsole": "6666@10.0.0.5:6666"},
		"ask",
		hostreqkit.EnsureOptions{},
	)
	if err != nil {
		t.Fatalf("EnsureLoadAtBoot: %v", err)
	}
	if !out.LoadFileChanged || !out.OptionsFileChanged {
		t.Errorf("Outcome = %+v, want both true", out)
	}
	loadPath := LoadFilePath("netconsole")
	if got := files[loadPath]; !strings.Contains(got, "netconsole") || !strings.Contains(got, ManagedHeader) {
		t.Errorf("load file content unexpected:\n%q", got)
	}
	optPath := OptionsFilePath("netconsole")
	if got := files[optPath]; !strings.Contains(got, "options netconsole netconsole=6666@10.0.0.5:6666") {
		t.Errorf("options file content unexpected:\n%q", got)
	}
	// Must run mkdir -p for both target dirs and install for both files —
	// but we're in single-file installs path, so each file gets its own pair.
	mkdirCount := 0
	installCount := 0
	for _, c := range *cmds {
		if c.Name == "mkdir" {
			mkdirCount++
		}
		if c.Name == "install" {
			installCount++
		}
	}
	if mkdirCount != 2 || installCount != 2 {
		t.Errorf("got mkdir=%d install=%d, want 2/2 (commands=%v)", mkdirCount, installCount, *cmds)
	}
}

func TestEnsureLoadAtBootNilOptionsSkipsModprobeFile(t *testing.T) {
	cmds, files, restore := stubAll(t)
	defer restore()

	out, err := EnsureLoadAtBoot("amd64_edac", nil, "ask", hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("EnsureLoadAtBoot: %v", err)
	}
	if !out.LoadFileChanged {
		t.Error("LoadFileChanged should be true")
	}
	if out.OptionsFileChanged {
		t.Error("OptionsFileChanged should be false when options is nil")
	}
	if _, ok := files[OptionsFilePath("amd64_edac")]; ok {
		t.Error("modprobe.d file should not be written when options is nil")
	}
	installCount := 0
	for _, c := range *cmds {
		if c.Name == "install" {
			installCount++
		}
	}
	if installCount != 1 {
		t.Errorf("install count = %d, want 1", installCount)
	}
}

func TestEnsureLoadAtBootIdempotent(t *testing.T) {
	cmds, files, restore := stubAll(t)
	defer restore()
	// Pre-populate the FS with the exact content the helper would write.
	files[LoadFilePath("netconsole")] = renderLoadFile("netconsole")
	files[OptionsFilePath("netconsole")] = renderOptionsFile("netconsole", map[string]string{"netconsole": "x"})

	out, err := EnsureLoadAtBoot(
		"netconsole",
		map[string]string{"netconsole": "x"},
		"ask",
		hostreqkit.EnsureOptions{},
	)
	if err != nil {
		t.Fatalf("EnsureLoadAtBoot: %v", err)
	}
	if out.LoadFileChanged || out.OptionsFileChanged {
		t.Errorf("Outcome = %+v, want zero (idempotent)", out)
	}
	if len(*cmds) != 0 {
		t.Errorf("idempotent run executed commands: %v", *cmds)
	}
}

func TestEnsureLoadAtBootDryRunWritesNothing(t *testing.T) {
	cmds, _, restore := stubAll(t)
	defer restore()

	out, err := EnsureLoadAtBoot(
		"netconsole",
		map[string]string{"netconsole": "x"},
		"ask",
		hostreqkit.EnsureOptions{DryRun: true},
	)
	if err != nil {
		t.Fatalf("EnsureLoadAtBoot: %v", err)
	}
	// Outcome still reports "would change".
	if !out.LoadFileChanged || !out.OptionsFileChanged {
		t.Errorf("DryRun Outcome = %+v, want both true (reports would-be change)", out)
	}
	if len(*cmds) != 0 {
		t.Errorf("DryRun ran commands: %v", *cmds)
	}
}

func TestEnsureLoadAtBootOnlyOptionsChanged(t *testing.T) {
	cmds, files, restore := stubAll(t)
	defer restore()
	// Pre-populate only the load file. Options file is missing — should be written.
	files[LoadFilePath("netconsole")] = renderLoadFile("netconsole")

	out, err := EnsureLoadAtBoot(
		"netconsole",
		map[string]string{"netconsole": "x"},
		"ask",
		hostreqkit.EnsureOptions{},
	)
	if err != nil {
		t.Fatalf("EnsureLoadAtBoot: %v", err)
	}
	if out.LoadFileChanged {
		t.Error("LoadFileChanged should be false (already matched)")
	}
	if !out.OptionsFileChanged {
		t.Error("OptionsFileChanged should be true")
	}
	installCount := 0
	for _, c := range *cmds {
		if c.Name == "install" {
			installCount++
		}
	}
	if installCount != 1 {
		t.Errorf("install count = %d, want 1 (only the options file)", installCount)
	}
}

func TestEnsureLoadAtBootRejectsEmptyName(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	if _, err := EnsureLoadAtBoot("", nil, "ask", hostreqkit.EnsureOptions{}); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestEnsureLoadAtBootInstallFailureSurfacesError(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		if name == "install" {
			return errors.New("synthetic install failure")
		}
		return nil
	}
	_, err := EnsureLoadAtBoot("netconsole", nil, "ask", hostreqkit.EnsureOptions{})
	if err == nil {
		t.Fatal("expected error from install failure")
	}
	if !strings.Contains(err.Error(), "synthetic install failure") {
		t.Errorf("error chain missing root cause: %v", err)
	}
}

func TestRenderOptionsFileDeterministicKeyOrder(t *testing.T) {
	// Map iteration is randomized in Go; we must sort keys before rendering or
	// FileContentMatches will flap.
	a := renderOptionsFile("foo", map[string]string{"b": "2", "a": "1", "c": "3"})
	b := renderOptionsFile("foo", map[string]string{"c": "3", "a": "1", "b": "2"})
	if a != b {
		t.Errorf("non-deterministic options output:\n%q\nvs\n%q", a, b)
	}
	if !strings.Contains(a, "a=1 b=2 c=3") {
		t.Errorf("expected sorted options, got %q", a)
	}
}

func TestModprobePassesOptionsAsKVArgs(t *testing.T) {
	cmds, _, restore := stubAll(t)
	defer restore()

	err := Modprobe("netconsole", map[string]string{"netconsole": "x"}, "ask", hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Modprobe: %v", err)
	}
	// One modprobe call expected (sudo wrapping is internal to RunPrivilegedCommand).
	found := false
	for _, c := range *cmds {
		if c.Name == "modprobe" || (c.Name == "sudo" && len(c.Args) > 0 && c.Args[0] == "modprobe") {
			found = true
			args := c.Args
			if c.Name == "sudo" {
				args = c.Args[1:]
			}
			// Expect: ["netconsole", "netconsole=x"]
			if len(args) < 2 || args[0] != "netconsole" || args[1] != "netconsole=x" {
				t.Errorf("modprobe args = %v, want [netconsole netconsole=x]", args)
			}
			break
		}
	}
	if !found {
		t.Errorf("modprobe not invoked: commands=%v", *cmds)
	}
}

func TestModprobeNoOptions(t *testing.T) {
	cmds, _, restore := stubAll(t)
	defer restore()

	if err := Modprobe("amd64_edac", nil, "ask", hostreqkit.EnsureOptions{}); err != nil {
		t.Fatalf("Modprobe: %v", err)
	}
	for _, c := range *cmds {
		if c.Name == "modprobe" || (c.Name == "sudo" && len(c.Args) > 0 && c.Args[0] == "modprobe") {
			args := c.Args
			if c.Name == "sudo" {
				args = c.Args[1:]
			}
			if len(args) != 1 || args[0] != "amd64_edac" {
				t.Errorf("modprobe args = %v, want [amd64_edac]", args)
			}
			return
		}
	}
	t.Errorf("modprobe not invoked: commands=%v", *cmds)
}

func TestModprobeDryRunIsNoOp(t *testing.T) {
	cmds, _, restore := stubAll(t)
	defer restore()

	if err := Modprobe("netconsole", nil, "ask", hostreqkit.EnsureOptions{DryRun: true}); err != nil {
		t.Fatalf("Modprobe: %v", err)
	}
	if len(*cmds) != 0 {
		t.Errorf("DryRun ran commands: %v", *cmds)
	}
}

func TestModprobeRejectsEmptyName(t *testing.T) {
	_, _, restore := stubAll(t)
	defer restore()
	if err := Modprobe("", nil, "ask", hostreqkit.EnsureOptions{}); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestPathConstructors(t *testing.T) {
	if got := LoadFilePath("netconsole"); got != "/etc/modules-load.d/99-vrooli-netconsole.conf" {
		t.Errorf("LoadFilePath = %q", got)
	}
	if got := OptionsFilePath("amd64_edac"); got != "/etc/modprobe.d/99-vrooli-amd64_edac.conf" {
		t.Errorf("OptionsFilePath = %q", got)
	}
}
