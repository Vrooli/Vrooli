//go:build !linux

package runtimesupervisor

import (
	"context"
	"fmt"
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

func InstallService(context.Context, ServiceInstallOptions) (ServiceInstallResult, error) {
	return ServiceInstallResult{}, fmt.Errorf("runtime supervisor service install is only implemented for Linux systemd user services")
}

func UninstallService(context.Context, ServiceInstallOptions) (ServiceInstallResult, error) {
	return ServiceInstallResult{}, fmt.Errorf("runtime supervisor service uninstall is only implemented for Linux systemd user services")
}
