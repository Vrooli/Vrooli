package hostreqkit

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestInstallCommandLinuxManagers(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "", os.ErrNotExist
	}

	tests := []struct {
		manager    string
		wantCmd    string
		wantPrefix string
	}{
		{"apt-get", "sudo", "apt-get install -y testpkg"},
		{"apt", "sudo", "apt-get install -y testpkg"},
		{"dnf", "sudo", "dnf install -y testpkg"},
		{"yum", "sudo", "yum install -y testpkg"},
		{"pacman", "sudo", "pacman -S --noconfirm testpkg"},
		{"apk", "sudo", "apk add testpkg"},
		{"brew", "brew", "install testpkg"},
	}

	for _, tt := range tests {
		t.Run(tt.manager, func(t *testing.T) {
			cmd, args, err := InstallCommand(Host{OS: "linux", PackageManager: tt.manager}, "testpkg", "ask")
			if err != nil {
				t.Fatalf("InstallCommand: %v", err)
			}
			if cmd != tt.wantCmd {
				t.Fatalf("command = %q, want %q", cmd, tt.wantCmd)
			}
			joined := strings.Join(args, " ")
			if !strings.Contains(joined, tt.wantPrefix) {
				t.Fatalf("args = %q, want containing %q", joined, tt.wantPrefix)
			}
		})
	}
}

func TestInstallCommandLinuxUnsupportedManager(t *testing.T) {
	_, _, err := InstallCommand(Host{OS: "linux", PackageManager: "unknown"}, "pkg", "ask")
	if err == nil {
		t.Fatal("expected error for unsupported package manager")
	}
}

func TestInstallCommandDarwinBrew(t *testing.T) {
	cmd, args, err := InstallCommand(Host{OS: "darwin", PackageManager: "brew"}, "jq", "ask")
	if err != nil {
		t.Fatalf("InstallCommand: %v", err)
	}
	if cmd != "brew" || strings.Join(args, " ") != "install jq" {
		t.Fatalf("got %s %v", cmd, args)
	}
}

func TestInstallCommandDarwinNoBrew(t *testing.T) {
	_, _, err := InstallCommand(Host{OS: "darwin", PackageManager: ""}, "jq", "ask")
	if err == nil || !strings.Contains(err.Error(), "Homebrew") {
		t.Fatalf("expected Homebrew error, got %v", err)
	}
}

func TestInstallCommandDarwinBrewWithWhitespace(t *testing.T) {
	cmd, args, err := InstallCommand(Host{OS: "darwin", PackageManager: " brew "}, "jq", "ask")
	if err != nil {
		t.Fatalf("InstallCommand: %v (whitespace around 'brew' should be trimmed)", err)
	}
	if cmd != "brew" || strings.Join(args, " ") != "install jq" {
		t.Fatalf("got %s %v", cmd, args)
	}
}

func TestInstallCommandWindowsWinget(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "winget" {
			return "C:\\winget.exe", nil
		}
		return "", os.ErrNotExist
	}

	cmd, args, err := InstallCommand(Host{OS: "windows", PackageManager: "winget"}, "Git.Git", "ask")
	if err != nil {
		t.Fatalf("InstallCommand: %v", err)
	}
	if cmd != "winget" {
		t.Fatalf("command = %q", cmd)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "Git.Git") || !strings.Contains(joined, "--accept-package-agreements") {
		t.Fatalf("args = %v", args)
	}
}

func TestInstallCommandWindowsNoWinget(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }

	_, _, err := InstallCommand(Host{OS: "windows"}, "pkg", "ask")
	if err == nil || !strings.Contains(err.Error(), "supported Windows package manager") {
		t.Fatalf("expected package manager error, got %v", err)
	}
}

func TestInstallCommandWindowsManagers(t *testing.T) {
	tests := []struct {
		manager string
		command string
		args    string
	}{
		{manager: "winget", command: "winget", args: "install --id pkg --accept-package-agreements --accept-source-agreements"},
		{manager: "choco", command: "choco", args: "install pkg -y"},
		{manager: "scoop", command: "scoop", args: "install pkg"},
	}
	for _, tt := range tests {
		t.Run(tt.manager, func(t *testing.T) {
			cmd, args, err := InstallCommand(Host{OS: "windows", PackageManager: tt.manager}, "pkg", "ask")
			if err != nil {
				t.Fatalf("InstallCommand: %v", err)
			}
			if cmd != tt.command || strings.Join(args, " ") != tt.args {
				t.Fatalf("got %s %v, want %s %s", cmd, args, tt.command, tt.args)
			}
		})
	}
}

func TestInstallCommandUnknownOS(t *testing.T) {
	_, _, err := InstallCommand(Host{OS: "freebsd"}, "pkg", "ask")
	if err == nil || !strings.Contains(err.Error(), "freebsd") {
		t.Fatalf("expected unsupported error, got %v", err)
	}
}

func TestInstallCommandEmptyOS(t *testing.T) {
	_, _, err := InstallCommand(Host{}, "pkg", "ask")
	if err == nil || !strings.Contains(err.Error(), "this platform") {
		t.Fatalf("expected platform error, got %v", err)
	}
}

func TestWithSudoAskMode(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "", os.ErrNotExist
	}

	cmd, args, err := WithSudo("ask", "apt-get", []string{"install", "-y", "jq"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "sudo" {
		t.Fatalf("command = %q", cmd)
	}
	if args[0] != "apt-get" {
		t.Fatalf("args[0] = %q", args[0])
	}
}

func TestWithSudoEmptyModeDefaultsToSkip(t *testing.T) {
	// Empty `--sudo-mode` is the operator typing `vrooli setup` with no
	// flag. We default to "skip" so the run is non-interactive — items
	// requiring root land in the Needs-sudo group via the typed sentinel
	// and the action block points at `sudo vrooli setup`.
	restore := stubLookups(t)
	defer restore()
	origRoot := RunningAsRootFn
	defer func() { RunningAsRootFn = origRoot }()
	RunningAsRootFn = func() bool { return false }

	LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "", os.ErrNotExist
	}

	_, _, err := WithSudo("", "apt-get", []string{"install"})
	if !IsSudoSkipped(err) {
		t.Fatalf("empty mode should default to skip (got err=%v)", err)
	}
}

func TestWithSudoRootSkipsWrap(t *testing.T) {
	// Already running as root: WithSudo must be a no-op regardless of mode,
	// otherwise `sudo vrooli setup` would self-fail with ErrSudoSkipped on
	// every privileged command.
	restore := stubLookups(t)
	defer restore()
	origRoot := RunningAsRootFn
	defer func() { RunningAsRootFn = origRoot }()
	RunningAsRootFn = func() bool { return true }

	LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "", os.ErrNotExist
	}

	for _, mode := range []string{"", "ask", "skip", "error"} {
		t.Run(mode, func(t *testing.T) {
			cmd, args, err := WithSudo(mode, "apt-get", []string{"install"})
			if err != nil {
				t.Fatalf("mode=%q: unexpected error %v", mode, err)
			}
			if cmd != "apt-get" || strings.Join(args, " ") != "install" {
				t.Fatalf("mode=%q: expected bare command, got %s %v", mode, cmd, args)
			}
		})
	}
}

func TestWithSudoSkipMode(t *testing.T) {
	restore := stubAvailableSudo(t)
	defer restore()

	_, _, err := WithSudo("skip", "apt-get", []string{"install"})
	if err == nil || !strings.Contains(err.Error(), "skip") {
		t.Fatalf("expected skip error, got %v", err)
	}
}

func TestWithSudoErrorMode(t *testing.T) {
	restore := stubAvailableSudo(t)
	defer restore()

	_, _, err := WithSudo("error", "apt-get", []string{"install"})
	if err == nil || !strings.Contains(err.Error(), "error") {
		t.Fatalf("expected error mode error, got %v", err)
	}
}

func TestWithSudoInvalidMode(t *testing.T) {
	restore := stubAvailableSudo(t)
	defer restore()

	_, _, err := WithSudo("bogus", "apt-get", []string{"install"})
	if err == nil || !strings.Contains(err.Error(), "invalid sudo mode") {
		t.Fatalf("expected invalid mode error, got %v", err)
	}
}

func TestWithSudoUnavailableFailsClosed(t *testing.T) {
	restore := stubUnavailableLookups(t)
	defer restore()

	_, _, err := WithSudo("ask", "apt-get", []string{"install", "-y", "jq"})
	if !errors.Is(err, ErrElevationUnavailable) {
		t.Fatalf("expected typed elevation error, got %v", err)
	}
}

func TestWithSudoUnavailableErrorModeFailsClosed(t *testing.T) {
	restore := stubUnavailableLookups(t)
	defer restore()

	_, _, err := WithSudo("error", "apt-get", []string{"install", "-y", "jq"})
	if !errors.Is(err, ErrElevationUnavailable) {
		t.Fatalf("expected typed elevation error, got %v", err)
	}
}

func TestWithSudoUnavailableSkipModeFailsClosed(t *testing.T) {
	restore := stubUnavailableLookups(t)
	defer restore()

	_, _, err := WithSudo("skip", "apt-get", []string{"install"})
	if !errors.Is(err, ErrElevationUnavailable) {
		t.Fatalf("expected typed elevation error, got %v", err)
	}
}

func TestWithSudoWindowsRequiresManualElevation(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	ElevationFactsFn = func() ElevationFacts { return ElevationFacts{Platform: "windows", Mechanism: "windows-uac"} }
	_, _, err := WithSudo("ask", "winget", []string{"install", "pkg"})
	if !errors.Is(err, ErrElevationRequired) || !strings.Contains(err.Error(), "winget install pkg") {
		t.Fatalf("expected exact manual elevation command, got %v", err)
	}
}

func TestWithSudoElevationMatrixNeverFailsOpen(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	tests := []struct {
		name       string
		facts      ElevationFacts
		mode       string
		wantCmd    string
		wantErr    error
		wantPrefix string
	}{
		{name: "linux elevated skip", facts: ElevationFacts{Platform: "linux", Elevated: true, CanElevate: true}, mode: "skip", wantCmd: "apt-get"},
		{name: "linux sudo ask", facts: ElevationFacts{Platform: "linux", CanElevate: true, Mechanism: "sudo"}, mode: "ask", wantCmd: "sudo", wantPrefix: "apt-get install"},
		{name: "linux sudo skip", facts: ElevationFacts{Platform: "linux", CanElevate: true, Mechanism: "sudo"}, mode: "skip", wantErr: ErrSudoSkipped},
		{name: "linux no elevation", facts: ElevationFacts{Platform: "linux", Mechanism: "none"}, mode: "ask", wantErr: ErrElevationUnavailable},
		{name: "darwin no elevation", facts: ElevationFacts{Platform: "darwin", Mechanism: "none"}, mode: "ask", wantErr: ErrElevationUnavailable},
		{name: "windows uac", facts: ElevationFacts{Platform: "windows", Mechanism: "windows-uac"}, mode: "ask", wantErr: ErrElevationRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ElevationFactsFn = func() ElevationFacts { return tt.facts }
			cmd, args, err := WithSudo(tt.mode, "apt-get", []string{"install", "pkg"})
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				if cmd != "" || args != nil {
					t.Fatalf("failed command returned executable: %q %v", cmd, args)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cmd != tt.wantCmd || (tt.wantPrefix != "" && !strings.Contains(strings.Join(args, " "), tt.wantPrefix)) {
				t.Fatalf("command = %q %v", cmd, args)
			}
		})
	}
}

func TestWithSudoSkipModeReturnsTypedSentinel(t *testing.T) {
	restore := stubAvailableSudo(t)
	defer restore()

	_, _, err := WithSudo("skip", "apt-get", []string{"install"})
	if !IsSudoSkipped(err) {
		t.Fatalf("IsSudoSkipped(%v) = false, want true", err)
	}
}

func TestWithSudoErrorModeReturnsTypedSentinel(t *testing.T) {
	restore := stubAvailableSudo(t)
	defer restore()

	_, _, err := WithSudo("error", "apt-get", []string{"install"})
	if !IsSudoSkipped(err) {
		t.Fatalf("IsSudoSkipped(%v) = false, want true", err)
	}
}

func TestIsSudoSkippedRecognizesWrappedErrors(t *testing.T) {
	wrapped := fmt.Errorf("apt install kdump-tools failed: %w: automatic install skipped because --sudo-mode=skip", ErrSudoSkipped)
	if !IsSudoSkipped(wrapped) {
		t.Fatalf("expected wrapped sentinel to be detected: %v", wrapped)
	}
}

func TestSudoAvailable(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "", os.ErrNotExist
	}
	if !SudoAvailable() {
		t.Fatal("sudo should be available")
	}

	LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	if SudoAvailable() {
		t.Fatal("sudo should not be available")
	}
}

func TestPackageNameForHost(t *testing.T) {
	m := ToolManifest{
		DefaultPackage: "fallback",
		Packages: map[string]string{
			"brew":    "brew-pkg",
			"apt-get": "apt-pkg",
		},
	}

	if got := m.PackageNameForHost(Host{PackageManager: "brew"}); got != "brew-pkg" {
		t.Fatalf("brew = %q", got)
	}
	if got := m.PackageNameForHost(Host{PackageManager: "apt-get"}); got != "apt-pkg" {
		t.Fatalf("apt-get = %q", got)
	}
	if got := m.PackageNameForHost(Host{PackageManager: "dnf"}); got != "fallback" {
		t.Fatalf("dnf fallback = %q", got)
	}
	if got := m.PackageNameForHost(Host{PackageManager: ""}); got != "fallback" {
		t.Fatalf("empty = %q", got)
	}
}

func TestPackageNameForHostEmptyManifest(t *testing.T) {
	m := ToolManifest{}
	if got := m.PackageNameForHost(Host{PackageManager: "brew"}); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestAptRepoInstallerInspectPlatforms(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	LookPathFn = func(name string) (string, error) {
		if name == "vault" {
			return "/usr/bin/vault", nil
		}
		return "", os.ErrNotExist
	}
	installer := AptRepoInstaller{
		Manifest: ToolManifest{
			Name: "vault", Commands: []string{"vault"},
			Packages: map[string]string{"brew": "hashicorp/tap/vault", "winget": "Hashicorp.Vault"},
		},
		AptPackage: "vault",
	}
	requirement := hostreqspec.ResolvedRequirement{Name: "vault", Kind: hostreqspec.KindTool}
	tests := []struct {
		name        string
		host        Host
		wantPkg     string
		wantStatus  SupportClass
		wantInstall bool
	}{
		{name: "apt", host: Host{OS: "linux", PackageManager: "apt"}, wantPkg: "vault", wantInstall: true, wantStatus: SupportSupported},
		{name: "brew", host: Host{OS: "darwin", PackageManager: "brew"}, wantPkg: "hashicorp/tap/vault", wantInstall: true, wantStatus: SupportSupported},
		{name: "winget", host: Host{OS: "windows", PackageManager: "winget"}, wantPkg: "Hashicorp.Vault", wantInstall: true, wantStatus: SupportSupported},
		{name: "unsupported", host: Host{OS: "linux", PackageManager: "dnf"}, wantStatus: SupportUnsupported},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status := installer.Inspect(test.host, requirement)
			if status.SupportClass != test.wantStatus {
				t.Fatalf("SupportClass = %q, want %q", status.SupportClass, test.wantStatus)
			}
			if status.PackageName != test.wantPkg {
				t.Fatalf("PackageName = %q, want %q", status.PackageName, test.wantPkg)
			}
			if status.InstallSupported != test.wantInstall {
				t.Fatalf("InstallSupported = %t, want %t", status.InstallSupported, test.wantInstall)
			}
		})
	}
}

func TestGoInstallInstallerInspectPresentAbsentAndVersionMismatch(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	manifest := ToolManifest{Name: "plugin", Commands: []string{"plugin"}, VersionArgs: []string{"--version"}}
	installer := GoInstallInstaller{Manifest: manifest, ModulePath: "example/plugin", Version: "v1.2.3", BinaryName: "plugin", Kind: InstallKindGo}
	requirement := hostreqspec.ResolvedRequirement{Name: "plugin", Kind: hostreqspec.KindTool}

	LookPathFn = func(name string) (string, error) {
		if name == "go" {
			return "/usr/local/go/bin/go", nil
		}
		if name == "plugin" {
			return "/usr/local/bin/plugin", nil
		}
		return "", os.ErrNotExist
	}
	CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "plugin" {
			return []byte("plugin v9.9.9\n"), nil
		}
		return nil, os.ErrNotExist
	}
	present := installer.Inspect(Host{OS: "linux"}, requirement)
	if !present.Installed || present.ExecutionState != ExecutionAlreadyPresent {
		t.Fatalf("present status = %+v", present)
	}
	if present.Version != "plugin v9.9.9" {
		t.Fatalf("observed version = %q", present.Version)
	}
	if present.PackageName != "example/plugin@v1.2.3" {
		t.Fatalf("package reference = %q", present.PackageName)
	}

	LookPathFn = func(name string) (string, error) {
		if name == "go" {
			return "/usr/local/go/bin/go", nil
		}
		return "", os.ErrNotExist
	}
	absent := installer.Inspect(Host{OS: "linux"}, requirement)
	if absent.Installed || !absent.InstallSupported {
		t.Fatalf("absent status = %+v", absent)
	}
}

func TestSysctlApplierInspectStates(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	applier := SysctlApplier{
		ConfigPath: "/etc/sysctl.d/test.conf",
		Parameters: []SysctlParameter{{Name: "net.test.one", Value: 4}, {Name: "net.test.minimum", Value: 8, Minimum: true, ReadFailure: -1}},
	}
	requirement := hostreqspec.ResolvedRequirement{Name: "test", Kind: hostreqspec.KindSafeguard}
	fullConfig := applier.ConfigContent()
	ReadFileFn = func(path string) ([]byte, error) {
		switch path {
		case "/proc/sys/net/test/one":
			return []byte("4"), nil
		case "/proc/sys/net/test/minimum":
			return []byte("9"), nil
		case applier.ConfigPath:
			return []byte(fullConfig), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	ready := applier.Inspect(Host{OS: "linux", SupportsSysctl: true}, requirement)
	if !ready.Applied || ready.ExecutionState != ExecutionAlreadyPresent {
		t.Fatalf("already-applied status = %+v", ready)
	}

	ReadFileFn = func(path string) ([]byte, error) {
		if path == "/proc/sys/net/test/one" {
			return []byte("2"), nil
		}
		if path == "/proc/sys/net/test/minimum" {
			return []byte("9"), nil
		}
		return []byte(fullConfig), nil
	}
	needsApply := applier.Inspect(Host{OS: "linux", SupportsSysctl: true}, requirement)
	if needsApply.Applied || needsApply.ExecutionState != ExecutionPending {
		t.Fatalf("needs-apply status = %+v", needsApply)
	}

	ReadFileFn = func(string) ([]byte, error) { return nil, os.ErrPermission }
	readFailure := applier.Inspect(Host{OS: "linux", SupportsSysctl: true}, requirement)
	if readFailure.Applied || readFailure.ExecutionState != ExecutionPending {
		t.Fatalf("read-failure status = %+v", readFailure)
	}
}
