//go:build linux

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func installService(options ServiceInstallOptions) (ServiceInstallResult, error) {
	if !options.User {
		return ServiceInstallResult{}, fmt.Errorf("platform: system service install requires explicit broker support")
	}
	home, err := resolvedHome(options.HomeDir)
	if err != nil {
		return ServiceInstallResult{}, err
	}
	executable := options.Executable
	if executable == "" {
		executable, err = os.Executable()
		if err != nil {
			return ServiceInstallResult{}, fmt.Errorf("platform: resolve executable: %w", err)
		}
	}
	path := filepath.Join(home, ".config", "systemd", "user", "vrooli-runtime-supervisor.service")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: create systemd unit dir: %w", err)
	}
	content := systemdUnitContent(executable, home, options.SourceRoot)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: write systemd unit: %w", err)
	}
	if output, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: systemctl daemon-reload: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if output, err := exec.Command("systemctl", "--user", "enable", "--now", "vrooli-runtime-supervisor.service").CombinedOutput(); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: systemctl enable: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return ServiceInstallResult{UnitName: "vrooli-runtime-supervisor.service", UnitPath: path, Scope: "user", Active: true}, nil
}

func uninstallService(options ServiceInstallOptions) (ServiceInstallResult, error) {
	home, err := resolvedHome(options.HomeDir)
	if err != nil {
		return ServiceInstallResult{}, err
	}
	path := filepath.Join(home, ".config", "systemd", "user", "vrooli-runtime-supervisor.service")
	_, _ = exec.Command("systemctl", "--user", "disable", "--now", "vrooli-runtime-supervisor.service").CombinedOutput()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return ServiceInstallResult{}, fmt.Errorf("platform: remove systemd unit: %w", err)
	}
	return ServiceInstallResult{UnitName: "vrooli-runtime-supervisor.service", UnitPath: path, Scope: "user", Active: false}, nil
}

func supportsService(user bool) bool {
	return user && exec.Command("systemctl", "--user", "--version").Run() == nil
}

func serviceStartHint() string { return "systemctl --user start vrooli-runtime-supervisor.service" }

func systemdUnitContent(executable, home, sourceRoot string) string {
	sourceRoot = strings.TrimSpace(sourceRoot)
	sourceEnv := ""
	workingDir := ""
	if sourceRoot != "" {
		sourceEnv = fmt.Sprintf("Environment=VROOLI_SOURCE_ROOT=%s\n", strconv.Quote(sourceRoot))
		workingDir = fmt.Sprintf("WorkingDirectory=%s\n", strconv.Quote(sourceRoot))
	}
	return fmt.Sprintf("[Unit]\nDescription=Vrooli runtime supervisor\nAfter=default.target\n\n[Service]\nType=simple\nEnvironment=HOME=%s\nEnvironment=VROOLI_RUNTIME_SUPERVISOR=on\n%s%sExecStart=%s --no-stale-check runtime supervisor run\nRestart=on-failure\nRestartSec=5s\n\n[Install]\nWantedBy=default.target\n", strconv.Quote(home), sourceEnv, workingDir, strconv.Quote(executable))
}

func resolvedHome(home string) (string, error) {
	if strings.TrimSpace(home) != "" {
		return home, nil
	}
	return HomeDir()
}
