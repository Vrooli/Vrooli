package protocgengo

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

var testManifest = hostreqkit.ToolManifest{
	Name:        "protoc-gen-go",
	Description: "Go protoc plugin",
	Commands:    []string{"protoc-gen-go"},
	VersionArgs: []string{"--version"},
	Handler:     "protoc_gen_go",
	Version:     "v1.36.11",
	Platforms:   []string{"linux", "macos", "windows"},
}

func newHandler() hostreqkit.Handler { return NewHandler(testManifest) }

func baseRequirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "protoc-gen-go", Kind: hostreqspec.KindTool}
}

func stub(t *testing.T) (restore func()) {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	origUserHome := UserHomeDirFn
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
		UserHomeDirFn = origUserHome
	}
}

func TestNameAndKind(t *testing.T) {
	h := newHandler()
	if h.Name() != "protoc-gen-go" {
		t.Fatalf("Name = %q", h.Name())
	}
	if h.Kind() != hostreqspec.KindTool {
		t.Fatalf("Kind = %q", h.Kind())
	}
}

func TestInspectAlreadyInstalled(t *testing.T) {
	defer stub(t)()
	hostreqkit.LookPathFn = func(name string) (string, error) {
		switch name {
		case "protoc-gen-go":
			return "/home/user/.local/bin/protoc-gen-go", nil
		case "go":
			return "/usr/local/go/bin/go", nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("protoc-gen-go v1.36.11\n"), nil
	}
	status := newHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, baseRequirement())
	if !status.Installed {
		t.Fatal("Installed should be true")
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
	if status.Version == "" {
		t.Fatal("Version should be populated when installed")
	}
}

func TestInspectMissingGoBlocksInstall(t *testing.T) {
	defer stub(t)()
	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	status := newHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, baseRequirement())
	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q (want unsupported when go is missing)", status.SupportClass)
	}
	if status.InstallSupported {
		t.Fatal("InstallSupported must be false when go is missing")
	}
}

func TestInspectGoPresentEnablesInstall(t *testing.T) {
	defer stub(t)()
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "go" {
			return "/usr/local/go/bin/go", nil
		}
		return "", os.ErrNotExist
	}
	status := newHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, baseRequirement())
	if !status.InstallSupported {
		t.Fatal("InstallSupported must be true when go is present")
	}
	if !strings.Contains(status.PackageName, "@v1.36.11") {
		t.Fatalf("PackageName = %q; want pinned version reference", status.PackageName)
	}
}

func TestApplyDryRunReportsCommand(t *testing.T) {
	defer stub(t)()
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "go" {
			return "/usr/local/go/bin/go", nil
		}
		return "", os.ErrNotExist
	}
	status := newHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, baseRequirement())
	out, err := newHandler().Apply(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionWouldInstall {
		t.Fatalf("ExecutionState = %q", out.ExecutionState)
	}
	matched := false
	for _, n := range out.Notes {
		if strings.Contains(n, "go install") && strings.Contains(n, "@v1.36.11") {
			matched = true
		}
	}
	if !matched {
		t.Fatalf("dry-run note should mention `go install ...@v1.36.11`, got %v", out.Notes)
	}
}

func TestApplyInvokesGoInstallAndSymlinks(t *testing.T) {
	defer stub(t)()

	tmpHome := t.TempDir()
	UserHomeDirFn = func() (string, error) { return tmpHome, nil }
	t.Setenv("GOBIN", "")
	t.Setenv("GOPATH", "")
	// Pre-create the binary at $HOME/go/bin so the post-install symlink finds a target.
	goBin := filepath.Join(tmpHome, "go", "bin")
	if err := os.MkdirAll(goBin, 0o755); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(goBin, "protoc-gen-go")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho v1.36.11\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var ranArgs []string
	hostreqkit.LookPathFn = func(name string) (string, error) {
		switch name {
		case "go":
			return "/usr/local/go/bin/go", nil
		case "protoc-gen-go":
			// First resolve attempt before install: fail.
			// After install, the symlink in ~/.local/bin will be detected.
			link := filepath.Join(tmpHome, ".local", "bin", "protoc-gen-go")
			if _, err := os.Lstat(link); err == nil {
				return link, nil
			}
			return "", os.ErrNotExist
		}
		return "", os.ErrNotExist
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		if name != "go" || len(args) < 2 || args[0] != "install" {
			return errors.New("unexpected command")
		}
		ranArgs = append([]string(nil), args...)
		return nil
	}
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return []byte("protoc-gen-go v1.36.11\n"), nil
	}

	status := newHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, baseRequirement())
	out, err := newHandler().Apply(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Fatalf("ExecutionState = %q (notes=%v)", out.ExecutionState, out.Notes)
	}
	if len(ranArgs) == 0 || !strings.Contains(ranArgs[1], "@v1.36.11") {
		t.Fatalf("install command = %v; want pinned @v1.36.11", ranArgs)
	}
	link := filepath.Join(tmpHome, ".local", "bin", "protoc-gen-go")
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("symlink not created: %v", err)
	}
	if target != binPath {
		t.Fatalf("symlink target = %q; want %q", target, binPath)
	}
}

func TestApplyAlreadyInstalledShortCircuits(t *testing.T) {
	defer stub(t)()
	status := hostreqkit.ItemStatus{Installed: true}
	out, err := newHandler().Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", out.ExecutionState)
	}
}

func TestPinnedVersionMatchesManifest(t *testing.T) {
	if testManifest.Version != defaultVersion {
		t.Fatalf("test manifest version %q drifted from handler defaultVersion %q", testManifest.Version, defaultVersion)
	}
}
