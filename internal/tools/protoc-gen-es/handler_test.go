package protocgenes

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
	Name:        "protoc-gen-es",
	Description: "TS/JS protoc plugin",
	Commands:    []string{"protoc-gen-es"},
	VersionArgs: []string{"--version"},
	Handler:     "protoc_gen_es",
	Version:     "2.12.0",
	Platforms:   []string{"linux", "macos", "windows"},
}

func newHandler() hostreqkit.Handler { return NewHandler(testManifest) }

func baseRequirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "protoc-gen-es", Kind: hostreqspec.KindTool}
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
	if h.Name() != "protoc-gen-es" {
		t.Fatalf("Name = %q", h.Name())
	}
	if h.Kind() != hostreqspec.KindTool {
		t.Fatalf("Kind = %q", h.Kind())
	}
}

func TestInspectMissingNpmIsUnsupported(t *testing.T) {
	defer stub(t)()
	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	status := newHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, baseRequirement())
	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q (want unsupported)", status.SupportClass)
	}
}

func TestInspectNpmPresentEnablesInstall(t *testing.T) {
	defer stub(t)()
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
	if !strings.Contains(status.PackageName, "@2.12.0") {
		t.Fatalf("PackageName = %q; want pinned version", status.PackageName)
	}
}

func TestApplyDryRunMentionsNpmInstall(t *testing.T) {
	defer stub(t)()
	tmp := t.TempDir()
	UserHomeDirFn = func() (string, error) { return tmp, nil }
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
		if strings.Contains(n, "npm install") && strings.Contains(n, "@2.12.0") {
			matched = true
		}
	}
	if !matched {
		t.Fatalf("dry-run note should mention `npm install ...@2.12.0`, got %v", out.Notes)
	}
}

func TestApplyInstallsAndSymlinks(t *testing.T) {
	defer stub(t)()
	tmp := t.TempDir()
	UserHomeDirFn = func() (string, error) { return tmp, nil }
	t.Setenv("XDG_CACHE_HOME", "")

	binDir := filepath.Join(tmp, ".cache", "vrooli", "protoc-plugins", "node", "node_modules", ".bin")
	binPath := filepath.Join(binDir, "protoc-gen-es")

	hostreqkit.LookPathFn = func(name string) (string, error) {
		switch name {
		case "npm":
			return "/usr/bin/npm", nil
		case "protoc-gen-es":
			link := filepath.Join(tmp, ".local", "bin", "protoc-gen-es")
			if _, err := os.Lstat(link); err == nil {
				return link, nil
			}
			return "", os.ErrNotExist
		}
		return "", os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return []byte("v2.12.0\n"), nil
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		if name != "npm" {
			return errors.New("unexpected command")
		}
		// Simulate npm placing the binary.
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			return err
		}
		return os.WriteFile(binPath, []byte("#!/bin/sh\necho 2.12.0\n"), 0o755)
	}

	status := newHandler().Inspect(hostreqkit.Host{OS: "linux"}, baseRequirement())
	out, err := newHandler().Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Fatalf("ExecutionState = %q (notes=%v)", out.ExecutionState, out.Notes)
	}
	link := filepath.Join(tmp, ".local", "bin", "protoc-gen-es")
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
