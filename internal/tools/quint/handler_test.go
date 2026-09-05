//nolint:goconst // test data deliberately reuses stable command fixtures.
package quint

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
	"github.com/vrooli/vrooli/internal/testenv"
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

var baseRequirement = quintBaseRequirement

func quintBaseRequirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "quint", Kind: hostreqspec.KindTool}
}

func stub(t *testing.T) (string, func()) {
	return hostreqkittest.StubInvokingUser(t)
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
	}, "@informalsystems/quint@0.32.0")
}

func TestApplyDryRunMentionsNpmInstall(t *testing.T) {
	_, restore := stub(t)
	defer restore()
	testenv.SetIdentityEnv(t, map[string]string{"XDG_CACHE_HOME": ""})
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
	hostreqkittest.AssertDryRunNote(t, out, "@informalsystems/quint@0.32.0")
}

func TestApplyInstallsAndSymlinks(t *testing.T) {
	tmp, restore := stub(t)
	defer restore()
	testenv.SetIdentityEnv(t, map[string]string{"XDG_CACHE_HOME": ""})

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
			return os.WriteFile(binPath, []byte(shelltest.POSIXShebang()+"echo 0.32.0\n"), 0o755)
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
