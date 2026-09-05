//nolint:goconst // test data deliberately reuses stable command fixtures.
package protocgengo

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
	"github.com/vrooli/vrooli/internal/testenv"
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

var baseRequirement = protocGenGoBaseRequirement

func protocGenGoBaseRequirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "protoc-gen-go", Kind: hostreqspec.KindTool}
}

func stub(t *testing.T) (restore func()) {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	origReadFile := hostreqkit.ReadFileFn
	origRoot := hostreqkit.RunningAsRootFn
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.RunningAsRootFn = origRoot
	}
}

// fakeFSExec is a tiny shell emulator that translates the handful of
// shell commands the protoc-gen handlers issue (mkdir -p, ln -sfn) into
// real filesystem operations. Tests use it to verify the handler's
// behavior without coupling to a real shell.
func fakeFSExec(name string, args []string) error {
	switch name {
	case "mkdir":
		// Expected: ["-p", path].
		if len(args) >= 2 && args[0] == "-p" {
			return os.MkdirAll(args[1], 0o755)
		}
	case "ln":
		// Expected: ["-sfn", source, link].
		if len(args) >= 3 && args[0] == "-sfn" {
			_ = os.Remove(args[2])
			return os.Symlink(args[1], args[2])
		}
	}
	return errors.New("fakeFSExec: unsupported command " + name + " " + strings.Join(args, " "))
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
	// Steer the user-dir probe at an empty tmpdir so the test isn't
	// polluted by the real developer's ~/.local/bin or ~/go/bin.
	tmpHome := testenv.RuntimeHome(t)
	hostreqkit.RunningAsRootFn = func() bool { return false }
	testenv.AsCurrentUser(t, "alice")
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == "/etc/passwd" {
			return []byte("alice:x:1000:1000:Alice:" + tmpHome + ":/bin/sh\n"), nil
		}
		return nil, os.ErrNotExist
	}
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

	tmpHome := testenv.RuntimeHome(t)
	// Steer InvokingUserHomeDir at the temp dir by stubbing the passwd
	// lookup seam. We stay non-root for this test (RunAsInvokingUser is
	// a no-op shell-out under that mode), exercising the basic flow.
	hostreqkit.RunningAsRootFn = func() bool { return false }
	testenv.AsCurrentUser(t, "alice")
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == "/etc/passwd" {
			return []byte("alice:x:1000:1000:Alice:" + tmpHome + ":/bin/sh\n"), nil
		}
		return nil, os.ErrNotExist
	}
	t.Setenv("GOBIN", "")
	t.Setenv("GOPATH", "")
	goBin := filepath.Join(tmpHome, "go", "bin")
	if err := os.MkdirAll(goBin, 0o755); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(goBin, "protoc-gen-go")

	var goInstallArgs []string
	hostreqkit.LookPathFn = func(name string) (string, error) {
		switch name {
		case "go":
			return "/usr/local/go/bin/go", nil
		case "protoc-gen-go":
			// LookPath always misses; ResolveCommandForInvokingUser's
			// user-dir probe finds the binary post-install.
			return "", os.ErrNotExist
		}
		return "", os.ErrNotExist
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		if name == "go" {
			if len(args) < 2 || args[0] != "install" {
				return errors.New("unexpected go args")
			}
			goInstallArgs = append([]string(nil), args...)
			// Simulate `go install` writing the binary.
			return os.WriteFile(binPath, []byte(shelltest.POSIXShebang()+""), 0o755)
		}
		// mkdir -p, ln -sfn run via RunAsInvokingUser → emulate via
		// fakeFSExec so the test exercises the real filesystem boundary.
		return fakeFSExec(name, args)
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
	if len(goInstallArgs) == 0 || !strings.Contains(goInstallArgs[1], "@v1.36.11") {
		t.Fatalf("install command = %v; want pinned @v1.36.11", goInstallArgs)
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

func TestApplyDropsPrivilegesWhenRoot(t *testing.T) {
	// Reproduces the bug from the user's `sudo vrooli setup` run:
	// with the old code, `go install` ran as root with HOME=/root,
	// dropping the binary in /root/go/bin instead of the operator's
	// home. The fix wraps with `sudo -u $SUDO_USER -H` — confirm the
	// wrap actually happens.
	defer stub(t)()

	tmpHome := t.TempDir()
	hostreqkit.RunningAsRootFn = func() bool { return true }
	testenv.SetSudoUser(t, "alice")
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == "/etc/passwd" {
			return []byte("alice:x:1000:1000:Alice:" + tmpHome + ":/bin/sh\n"), nil
		}
		return nil, os.ErrNotExist
	}
	t.Setenv("GOBIN", "")
	t.Setenv("GOPATH", "")

	goBin := filepath.Join(tmpHome, "go", "bin")
	if err := os.MkdirAll(goBin, 0o755); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(goBin, "protoc-gen-go")

	var sudoCalls []string
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "go" {
			return "/usr/local/go/bin/go", nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		// Under sudo'd flow every per-user command should be wrapped
		// with `sudo -u alice -H -- ...` — record and assert.
		sudoCalls = append(sudoCalls, name+" "+strings.Join(args, " "))
		if name == "sudo" {
			// Honor mkdir / ln / go install so the post-install state
			// is realistic. Strip leading sudo args (-u alice -H --).
			for i, a := range args {
				if a == "--" && i+1 < len(args) {
					inner, innerArgs := args[i+1], args[i+2:]
					if inner == "go" && len(innerArgs) >= 1 && innerArgs[0] == "install" {
						return os.WriteFile(binPath, []byte(shelltest.POSIXShebang()+""), 0o755)
					}
					return fakeFSExec(inner, innerArgs)
				}
			}
		}
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
		t.Fatalf("ExecutionState = %q (notes=%v); expected installed under sudo'd flow", out.ExecutionState, out.Notes)
	}

	// Every command we issued should have been wrapped with sudo -u alice.
	wantWraps := []string{
		"sudo -u alice -H -- go install",
		"sudo -u alice -H -- mkdir -p",
		"sudo -u alice -H -- ln -sfn",
	}
	for _, want := range wantWraps {
		var found bool
		for _, c := range sudoCalls {
			if strings.HasPrefix(c, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing sudo-wrapped invocation %q in:\n  %s", want, strings.Join(sudoCalls, "\n  "))
		}
	}
}
