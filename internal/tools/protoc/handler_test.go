//nolint:goconst // test data deliberately reuses stable command fixtures.
package protoc

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

var testManifest = hostreqkit.ToolManifest{
	Name:        "protoc",
	Description: "Protocol Buffers compiler",
	Commands:    []string{"protoc"},
	VersionArgs: []string{"--version"},
	Handler:     "protoc",
	Version:     "34.1",
	Packages: map[string]string{
		"apt":     "protobuf-compiler",
		"apt-get": "protobuf-compiler",
		"brew":    "protobuf",
		"winget":  "Google.Protobuf",
	},
	DefaultPackage: "protobuf-compiler",
	Platforms:      []string{"linux", "macos", "windows"},
}

func newHandler() hostreqkit.Handler { return NewHandler(testManifest) }

var baseRequirement = protocBaseRequirement

func protocBaseRequirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "protoc", Kind: hostreqspec.KindTool}
}

func stub(t *testing.T) (restore func()) {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
	}
}

func TestInspectLinuxAptNotInstalled(t *testing.T) {
	defer stub(t)()
	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	status := newHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, baseRequirement())
	if !status.InstallSupported {
		t.Fatal("InstallSupported should be true on apt")
	}
	if status.PackageName != "protobuf-compiler" {
		t.Fatalf("PackageName = %q", status.PackageName)
	}
}

func TestInspectDarwinBrew(t *testing.T) {
	defer stub(t)()
	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	status := newHandler().Inspect(hostreqkit.Host{OS: "darwin", PackageManager: "brew"}, baseRequirement())
	if status.PackageName != "protobuf" {
		t.Fatalf("PackageName = %q (want darwin brew mapping)", status.PackageName)
	}
}

func TestInspectInstalledReportsVersion(t *testing.T) {
	defer stub(t)()
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "protoc" {
			return "/usr/bin/protoc", nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return []byte("libprotoc 34.1\n"), nil
	}
	status := newHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, baseRequirement())
	if !status.Installed || status.Version == "" {
		t.Fatalf("expected installed+version, got Installed=%v Version=%q", status.Installed, status.Version)
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyDryRunReportsCommand(t *testing.T) {
	defer stub(t)()
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "", os.ErrNotExist
	}
	status := newHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, baseRequirement())
	out, err := newHandler().Apply(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, status, hostreqkit.EnsureOptions{DryRun: true, SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionWouldInstall {
		t.Fatalf("ExecutionState = %q", out.ExecutionState)
	}
	matched := false
	for _, n := range out.Notes {
		if strings.Contains(n, "apt-get") && strings.Contains(n, "protobuf-compiler") {
			matched = true
		}
	}
	if !matched {
		t.Fatalf("dry-run note should mention apt-get install protobuf-compiler, got %v", out.Notes)
	}
}

func TestApplyInvokesPackageManager(t *testing.T) {
	defer stub(t)()
	installed := false
	hostreqkit.LookPathFn = func(name string) (string, error) {
		switch name {
		case "sudo":
			return "/usr/bin/sudo", nil
		case "protoc":
			if installed {
				return "/usr/bin/protoc", nil
			}
		}
		return "", os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return []byte("libprotoc 34.1\n"), nil
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		if name != "sudo" || len(args) < 4 || args[1] != "install" {
			return errors.New("unexpected command: " + name + " " + strings.Join(args, " "))
		}
		installed = true
		return nil
	}
	status := newHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, baseRequirement())
	out, err := newHandler().Apply(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, status, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Fatalf("ExecutionState = %q (notes=%v)", out.ExecutionState, out.Notes)
	}
}
