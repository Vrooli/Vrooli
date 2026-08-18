package runtimesupervisor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
)

// LogFileName is the supervisor's log basename inside the standard Vrooli log
// tree. Every scenario already gets <home>/.vrooli/logs/<name>.log; the control
// plane's own daemon uses the same tree so operators have one place to look.
const LogFileName = "runtime-supervisor.log"

type ServiceInstallOptions struct {
	HomeDir    string
	Executable string
	SourceRoot string
	User       bool
}

type ServiceInstallResult struct {
	UnitName string `json:"unit_name"`
	UnitPath string `json:"unit_path"`
	Scope    string `json:"scope"`
	Active   bool   `json:"active"`
	LogPath  string `json:"log_path,omitempty"`
	// Executable is the absolute path the installed unit will run, and
	// ExecutableIsCanonical reports whether that is the installed CLI rather
	// than a build output or other transient binary. Surfaced so an operator
	// sees what the unit was pinned to at install time instead of discovering
	// it later from /proc.
	Executable            string `json:"executable,omitempty"`
	ExecutableIsCanonical bool   `json:"executable_is_canonical"`
}

// LogPath resolves the supervisor's log file from the runtime_home authority.
// platform-go deliberately does not know the Vrooli home contract, so the path
// is resolved here and passed down as a plain absolute path.
func LogPath(homeDir string) (string, error) {
	root, err := repocontract.RuntimeHomeEntryPath(homeDir, repocontract.HomeKeyLogs)
	if err != nil {
		return "", fmt.Errorf("resolve runtime supervisor log path: %w", err)
	}
	return filepath.Join(root, LogFileName), nil
}

// cliBinaryName is the installed control-plane binary the service unit runs.
const cliBinaryName = "vrooli"

// ExecutablePath resolves the binary a service unit should name, preferring the
// canonical installed CLI over whichever binary happens to be performing the
// install.
//
// A unit records an absolute path and keeps running it across restarts and
// reboots, but the process doing the install is routinely NOT the installed
// binary: it is a build output under .vrooli/build, a `go run` temp file, or a
// one-off build in someone's checkout. Deriving ExecStart from os.Executable()
// therefore pins the fleet's supervisor to whatever scratch path the installer
// happened to live at — and that path is rewritten by every subsequent build.
// This happened in practice: an install run from .vrooli/build/vrooli left the
// supervisor unit pointing at a file that `make install` overwrites.
//
// Reports whether the returned path is the canonical installed one, so callers
// can tell an operator when a unit is being pinned somewhere less durable.
func ExecutablePath(homeDir, requested string) (string, bool, error) {
	binDir, err := repocontract.RuntimeHomeEntryPath(homeDir, repocontract.HomeKeyBin)
	if err == nil {
		installed := filepath.Join(binDir, cliBinaryName)
		if runtime.GOOS == "windows" {
			installed += ".exe"
		}
		if isExecutableFile(installed) {
			return installed, true, nil
		}
	}
	// No installed CLI yet — a first-run or development install. Naming the
	// requesting binary is better than refusing, but it is not canonical.
	if strings.TrimSpace(requested) != "" {
		return requested, false, nil
	}
	running, err := os.Executable()
	if err != nil {
		return "", false, fmt.Errorf("resolve executable for the runtime supervisor service: %w", err)
	}
	return running, false, nil
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	// Windows has no executable bit; existence of the file is the signal there.
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode().Perm()&0o111 != 0
}

func InstallService(ctx context.Context, opts ServiceInstallOptions) (ServiceInstallResult, error) {
	if err := ctx.Err(); err != nil {
		return ServiceInstallResult{}, err
	}
	logPath, err := LogPath(opts.HomeDir)
	if err != nil {
		return ServiceInstallResult{}, err
	}
	executable, canonical, err := ExecutablePath(opts.HomeDir, opts.Executable)
	if err != nil {
		return ServiceInstallResult{}, err
	}
	result, err := platform.InstallService(platform.ServiceInstallOptions{
		HomeDir: opts.HomeDir, Executable: executable, SourceRoot: opts.SourceRoot, User: opts.User,
		LogPath: logPath,
	})
	return ServiceInstallResult{
		UnitName: result.UnitName, UnitPath: result.UnitPath, Scope: result.Scope,
		Active: result.Active, LogPath: logPath,
		Executable: executable, ExecutableIsCanonical: canonical,
	}, err
}

func UninstallService(ctx context.Context, opts ServiceInstallOptions) (ServiceInstallResult, error) {
	if err := ctx.Err(); err != nil {
		return ServiceInstallResult{}, err
	}
	result, err := platform.UninstallService(platform.ServiceInstallOptions{HomeDir: opts.HomeDir, User: opts.User})
	return ServiceInstallResult{UnitName: result.UnitName, UnitPath: result.UnitPath, Scope: result.Scope, Active: result.Active}, err
}

func ServiceStartHint() string { return platform.ServiceStartHint() }
