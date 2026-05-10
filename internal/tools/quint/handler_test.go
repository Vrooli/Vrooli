package quint

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
	Name:        "quint",
	Description: "Quint formal specification CLI",
	Commands:    []string{"quint"},
	VersionArgs: []string{"--version"},
	Handler:     "quint",
	Version:     "0.32.0",
	Platforms:   []string{"linux", "macos", "windows"},
}

func newHandler() hostreqkit.Handler { return NewHandler(testManifest) }

func baseRequirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "quint", Kind: hostreqspec.KindTool}
}

func stub(t *testing.T) (string, func()) {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	origReadFile := hostreqkit.ReadFileFn
	origRoot := hostreqkit.RunningAsRootFn

	tmp := t.TempDir()
	hostreqkit.RunningAsRootFn = func() bool { return false }
	t.Setenv("USER", "alice")
	t.Setenv("HOME", tmp)
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == "/etc/passwd" {
			return []byte("alice:x:1000:1000:Alice:" + tmp + ":/bin/sh\n"), nil
		}
		return nil, os.ErrNotExist
	}
	return tmp, func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.RunningAsRootFn = origRoot
	}
}

func fakeFSExec(name string, args []string) error {
	switch name {
	case "mkdir":
		if len(args) >= 2 && args[0] == "-p" {
			return os.MkdirAll(args[1], 0o755)
		}
	case "ln":
		if len(args) >= 3 && args[0] == "-sfn" {
			_ = os.Remove(args[2])
			return os.Symlink(args[1], args[2])
		}
	}
	return errors.New("fakeFSExec: unsupported " + name + " " + strings.Join(args, " "))
}

func TestNameAndKind(t *testing.T) {
	h := newHandler()
	if h.Name() != "quint" {
		t.Fatalf("Name = %q", h.Name())
	}
	if h.Kind() != hostreqspec.KindTool {
		t.Fatalf("Kind = %q", h.Kind())
	}
}

func TestInspectMissingNpmIsUnsupported(t *testing.T) {
	_, restore := stub(t)
	defer restore()
	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	status := newHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, baseRequirement())
	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q (want unsupported)", status.SupportClass)
	}
}

func TestInspectNpmPresentEnablesInstall(t *testing.T) {
	_, restore := stub(t)
	defer restore()
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "npm" {
			return "/usr/bin/npm", nil
		}
		return "", os.ErrNotExist
	}
	status := newHandler().Inspect(hostreqkit.Host{OS: "linux"}, baseRequirement())
	if !status.InstallSupported {
		t.Fatal("InstallSupported should be true with npm available")
	}
	if !strings.Contains(status.PackageName, "@informalsystems/quint@0.32.0") {
		t.Fatalf("PackageName = %q; want pinned Quint package", status.PackageName)
	}
}

func TestApplyDryRunMentionsNpmInstall(t *testing.T) {
	_, restore := stub(t)
	defer restore()
	t.Setenv("XDG_CACHE_HOME", "")
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "npm" {
			return "/usr/bin/npm", nil
		}
		return "", os.ErrNotExist
	}
	status := newHandler().Inspect(hostreqkit.Host{OS: "linux"}, baseRequirement())
	out, err := newHandler().Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionWouldInstall {
		t.Fatalf("ExecutionState = %q", out.ExecutionState)
	}
	matched := false
	for _, n := range out.Notes {
		if strings.Contains(n, "npm install") && strings.Contains(n, "@informalsystems/quint@0.32.0") {
			matched = true
		}
	}
	if !matched {
		t.Fatalf("dry-run note should mention pinned Quint install, got %v", out.Notes)
	}
}

func TestApplyInstallsAndSymlinks(t *testing.T) {
	tmp, restore := stub(t)
	defer restore()
	t.Setenv("XDG_CACHE_HOME", "")

	binDir := filepath.Join(tmp, ".cache", "vrooli", "formal-tools", "quint", "node", "node_modules", ".bin")
	binPath := filepath.Join(binDir, "quint")

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "npm" {
			return "/usr/bin/npm", nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return []byte("0.32.0\n"), nil
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		if name == "npm" {
			if err := os.MkdirAll(binDir, 0o755); err != nil {
				return err
			}
			return os.WriteFile(binPath, []byte("#!/bin/sh\necho 0.32.0\n"), 0o755)
		}
		return fakeFSExec(name, args)
	}

	status := newHandler().Inspect(hostreqkit.Host{OS: "linux"}, baseRequirement())
	out, err := newHandler().Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Fatalf("ExecutionState = %q (notes=%v)", out.ExecutionState, out.Notes)
	}
	link := filepath.Join(tmp, ".local", "bin", "quint")
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("symlink not created: %v", err)
	}
	if target != binPath {
		t.Fatalf("symlink target = %q; want %q", target, binPath)
	}
}

func TestApplyAlreadyInstalledShortCircuits(t *testing.T) {
	_, restore := stub(t)
	defer restore()
	out, err := newHandler().Apply(hostreqkit.Host{OS: "linux"}, hostreqkit.ItemStatus{Installed: true}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", out.ExecutionState)
	}
}

func TestPinnedVersionMatchesManifest(t *testing.T) {
	if testManifest.Version != defaultVersion {
		t.Fatalf("test manifest version %q drifted from defaultVersion %q", testManifest.Version, defaultVersion)
	}
}
