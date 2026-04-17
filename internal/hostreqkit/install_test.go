package hostreqkit

import (
	"os"
	"strings"
	"testing"
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

func TestInstallCommandWindowsWinget(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "winget" {
			return "C:\\winget.exe", nil
		}
		return "", os.ErrNotExist
	}

	cmd, args, err := InstallCommand(Host{OS: "windows"}, "Git.Git", "ask")
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
	if err == nil || !strings.Contains(err.Error(), "winget") {
		t.Fatalf("expected winget error, got %v", err)
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

func TestWithSudoEmptyModeDefaultsToAsk(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "", os.ErrNotExist
	}

	cmd, _, err := WithSudo("", "apt-get", []string{"install"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "sudo" {
		t.Fatalf("empty mode should default to sudo, got %q", cmd)
	}
}

func TestWithSudoSkipMode(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "", os.ErrNotExist
	}

	_, _, err := WithSudo("skip", "apt-get", []string{"install"})
	if err == nil || !strings.Contains(err.Error(), "skip") {
		t.Fatalf("expected skip error, got %v", err)
	}
}

func TestWithSudoErrorMode(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "", os.ErrNotExist
	}

	_, _, err := WithSudo("error", "apt-get", []string{"install"})
	if err == nil || !strings.Contains(err.Error(), "error") {
		t.Fatalf("expected error mode error, got %v", err)
	}
}

func TestWithSudoInvalidMode(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "", os.ErrNotExist
	}

	_, _, err := WithSudo("bogus", "apt-get", []string{"install"})
	if err == nil || !strings.Contains(err.Error(), "invalid sudo mode") {
		t.Fatalf("expected invalid mode error, got %v", err)
	}
}

func TestWithSudoUnavailableFallsThrough(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }

	cmd, args, err := WithSudo("ask", "apt-get", []string{"install", "-y", "jq"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "apt-get" {
		t.Fatalf("command = %q, want apt-get (no sudo)", cmd)
	}
	if strings.Join(args, " ") != "install -y jq" {
		t.Fatalf("args = %v", args)
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
