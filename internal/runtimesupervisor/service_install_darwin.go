//go:build darwin

package runtimesupervisor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/config"
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
	plan := newLaunchdServicePlan(home, os.Getuid())
	plistPath := plan.PlistPath
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("create LaunchAgents dir: %w", err)
	}
	if err := os.WriteFile(plistPath, []byte(launchAgentPlistContent(exe, home, opts.SourceRoot)), 0o644); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("write runtime supervisor launchd plist: %w", err)
	}
	// Re-bootstrapping an already-loaded agent fails, so boot it out first and
	// ignore the "not loaded" failure on a fresh install.
	_ = runLaunchctl(ctx, "bootout", plan.ServiceTarget)
	if err := runLaunchctl(ctx, "enable", plan.ServiceTarget); err != nil {
		return ServiceInstallResult{}, err
	}
	if err := runLaunchctl(ctx, "bootstrap", plan.DomainTarget, plistPath); err != nil {
		return ServiceInstallResult{}, err
	}
	return ServiceInstallResult{UnitName: launchdLabel, UnitPath: plistPath, Scope: "user", Active: true}, nil
}

func UninstallService(ctx context.Context, opts ServiceInstallOptions) (ServiceInstallResult, error) {
	if !opts.User {
		return ServiceInstallResult{}, fmt.Errorf("runtime supervisor system service uninstall is not implemented; use --user")
	}
	home := strings.TrimSpace(opts.HomeDir)
	if home == "" {
		var err error
		home, err = config.HomeDir()
		if err != nil {
			return ServiceInstallResult{}, fmt.Errorf("resolve user home: %w", err)
		}
	}
	plan := newLaunchdServicePlan(home, os.Getuid())
	plistPath := plan.PlistPath
	// Ignore bootout failure: the agent may already be unloaded, and the plist
	// removal below is the durable part of the uninstall.
	_ = runLaunchctl(ctx, "bootout", plan.ServiceTarget)
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return ServiceInstallResult{}, fmt.Errorf("remove runtime supervisor launchd plist: %w", err)
	}
	return ServiceInstallResult{UnitName: launchdLabel, UnitPath: plistPath, Scope: "user", Active: false}, nil
}

// ServiceStartHint is the platform-native command that starts the installed
// supervisor service, shown in `runtime supervisor status` next steps.
func ServiceStartHint() string {
	return fmt.Sprintf("launchctl kickstart %s", newLaunchdServicePlan("", os.Getuid()).ServiceTarget)
}

func runLaunchctl(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "launchctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
