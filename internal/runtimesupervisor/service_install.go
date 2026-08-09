package runtimesupervisor

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	platform "github.com/vrooli/platform-go"
)

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
	if err := ctx.Err(); err != nil {
		return ServiceInstallResult{}, err
	}
	result, err := platform.InstallService(platform.ServiceInstallOptions{
		HomeDir: opts.HomeDir, Executable: opts.Executable, SourceRoot: opts.SourceRoot, User: opts.User,
	})
	return ServiceInstallResult{UnitName: result.UnitName, UnitPath: result.UnitPath, Scope: result.Scope, Active: result.Active}, err
}

func UninstallService(ctx context.Context, opts ServiceInstallOptions) (ServiceInstallResult, error) {
	if err := ctx.Err(); err != nil {
		return ServiceInstallResult{}, err
	}
	result, err := platform.UninstallService(platform.ServiceInstallOptions{HomeDir: opts.HomeDir, User: opts.User})
	return ServiceInstallResult{UnitName: result.UnitName, UnitPath: result.UnitPath, Scope: result.Scope, Active: result.Active}, err
}

func ServiceStartHint() string { return platform.ServiceStartHint() }

// systemdUserUnitContent remains a pure rendering helper for the Linux
// contract tests. The lifecycle operation itself is owned by platform-go.
func systemdUserUnitContent(executable, home, sourceRoot string) string {
	sourceRoot = strings.TrimSpace(sourceRoot)
	extra := ""
	if sourceRoot != "" {
		extra = fmt.Sprintf("Environment=VROOLI_SOURCE_ROOT=%s\nWorkingDirectory=%s\n", strconv.Quote(sourceRoot), sourceRoot)
	}
	return fmt.Sprintf("[Unit]\nDescription=Vrooli runtime supervisor\nAfter=default.target\n\n[Service]\nType=simple\nEnvironment=HOME=%s\nEnvironment=VROOLI_RUNTIME_SUPERVISOR=on\n%sExecStart=%s --no-stale-check runtime supervisor run\nRestart=on-failure\nRestartSec=5s\n\n[Install]\nWantedBy=default.target\n", strconv.Quote(home), extra, strconv.Quote(executable))
}
