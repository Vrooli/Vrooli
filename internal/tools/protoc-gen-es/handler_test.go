package protocgenes

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
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

var baseRequirement = protocGenESBaseRequirement

func protocGenESBaseRequirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "protoc-gen-es", Kind: hostreqspec.KindTool}
}

// stub installs the test seams the handler reaches for and steers
// InvokingUserHomeDir at a clean tempdir so the user-dir probe in
// ResolveCommandForInvokingUser doesn't see the developer's real
// ~/.local/bin. Tests that need a specific home should override
// hostreqkit.ReadFileFn after stub() returns.
func stub(t *testing.T) (string, func()) {
	return hostreqkittest.StubInvokingUser(t)
}

// fakeFSExec emulates the small set of shell commands the protoc-gen-es
// handler issues via RunAsInvokingUser (mkdir -p, ln -sfn).
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
	hostreqkittest.RunInstallSupportedProbe(t, func() {
		hostreqkit.LookPathFn = func(name string) (string, error) {
			if name == "npm" {
				return "/usr/bin/npm", nil
			}
			return "", os.ErrNotExist
		}
	}, func() hostreqkit.ItemStatus {
		return newHandler().Inspect(hostreqkit.Host{OS: "linux"}, baseRequirement())
	}, "@2.12.0")
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
	hostreqkittest.AssertDryRunNote(t, out, "@2.12.0")
}

func TestApplyInstallsAndSymlinks(t *testing.T) {
	tmp, restore := stub(t)
	defer restore()
	t.Setenv("XDG_CACHE_HOME", "")

	binDir := filepath.Join(tmp, ".cache", "vrooli", "protoc-plugins", "node", "node_modules", ".bin")
	binPath := filepath.Join(binDir, "protoc-gen-es")

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "npm" {
			return "/usr/bin/npm", nil
		}
		// LookPath always misses for the plugin; the user-dir probe in
		// ResolveCommandForInvokingUser finds the symlink.
		return "", os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return []byte("v2.12.0\n"), nil
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		if name == "npm" {
			// Simulate npm placing the binary.
			if err := os.MkdirAll(binDir, 0o755); err != nil {
				return err
			}
			return os.WriteFile(binPath, []byte(shelltest.POSIXShebang()+"echo 2.12.0\n"), 0o755)
		}
		// mkdir -p / ln -sfn flow through fakeFSExec so the test
		// exercises the real filesystem boundary.
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
