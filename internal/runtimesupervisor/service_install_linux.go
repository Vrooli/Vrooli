//go:build linux

package runtimesupervisor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/config"
)

const systemdUserUnitName = "vrooli-runtime-supervisor.service"

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
}

func InstallService(ctx context.Context, opts ServiceInstallOptions) (ServiceInstallResult, error) {
	if !opts.User {
		return ServiceInstallResult{}, fmt.Errorf("runtime supervisor system service install is not implemented; use --user")
	}
	exe := strings.TrimSpace(opts.Executable)
	if exe == "" {
		var err error
		exe, err = os.Executable()
		if err != nil {
			return ServiceInstallResult{}, fmt.Errorf("resolve vrooli executable: %w", err)
		}
	}
	home := strings.TrimSpace(opts.HomeDir)
	if home == "" {
		var err error
		home, err = config.HomeDir()
		if err != nil {
			return ServiceInstallResult{}, fmt.Errorf("resolve user home: %w", err)
		}
	}
	unitPath, err := systemdUserUnitPath()
	if err != nil {
		return ServiceInstallResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("create systemd user unit dir: %w", err)
	}
	if err := os.WriteFile(unitPath, []byte(systemdUserUnitContent(exe, home, opts.SourceRoot)), 0o644); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("write runtime supervisor systemd unit: %w", err)
	}
	if err := runSystemctlUser(ctx, "daemon-reload"); err != nil {
		return ServiceInstallResult{}, err
	}
	if err := runSystemctlUser(ctx, "enable", "--now", systemdUserUnitName); err != nil {
		return ServiceInstallResult{}, err
	}
	return ServiceInstallResult{UnitName: systemdUserUnitName, UnitPath: unitPath, Scope: "user", Active: true}, nil
}

func UninstallService(ctx context.Context, opts ServiceInstallOptions) (ServiceInstallResult, error) {
	if !opts.User {
		return ServiceInstallResult{}, fmt.Errorf("runtime supervisor system service uninstall is not implemented; use --user")
	}
	unitPath, err := systemdUserUnitPath()
	if err != nil {
		return ServiceInstallResult{}, err
	}
	if err := runSystemctlUser(ctx, "disable", "--now", systemdUserUnitName); err != nil {
		return ServiceInstallResult{}, err
	}
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return ServiceInstallResult{}, fmt.Errorf("remove runtime supervisor systemd unit: %w", err)
	}
	if err := runSystemctlUser(ctx, "daemon-reload"); err != nil {
		return ServiceInstallResult{}, err
	}
	return ServiceInstallResult{UnitName: systemdUserUnitName, UnitPath: unitPath, Scope: "user", Active: false}, nil
}

func systemdUserUnitPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(configDir, "systemd", "user", systemdUserUnitName), nil
}

func systemdUserUnitContent(executable string, home string, sourceRoot string) string {
	sourceRoot = strings.TrimSpace(sourceRoot)
	sourceRootEnv := ""
	workingDirectory := ""
	if sourceRoot != "" {
		sourceRootEnv = fmt.Sprintf("Environment=VROOLI_SOURCE_ROOT=%s\n", systemdValue(sourceRoot))
		workingDirectory = fmt.Sprintf("WorkingDirectory=%s\n", sourceRoot)
	}
	return fmt.Sprintf(`[Unit]
Description=Vrooli runtime supervisor
After=default.target

[Service]
Type=simple
Environment=HOME=%s
Environment=VROOLI_RUNTIME_SUPERVISOR=on
%s%sExecStart=%s --no-stale-check runtime supervisor run
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=default.target
`, systemdValue(home), sourceRootEnv, workingDirectory, systemdValue(executable))
}

func systemdValue(value string) string {
	return strconv.Quote(value)
}

func runSystemctlUser(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "systemctl", append([]string{"--user"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl --user %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
