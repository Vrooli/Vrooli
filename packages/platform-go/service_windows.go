//go:build windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const runtimeSupervisorService = "VrooliRuntimeSupervisor"

func installService(options ServiceInstallOptions) (ServiceInstallResult, error) {
	if !options.User {
		return ServiceInstallResult{}, fmt.Errorf("platform: system service install requires explicit broker support")
	}
	executable := options.Executable
	if executable == "" {
		var err error
		executable, err = os.Executable()
		if err != nil {
			return ServiceInstallResult{}, fmt.Errorf("platform: resolve executable: %w", err)
		}
	}
	executable, err := filepath.Abs(executable)
	if err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: resolve executable path: %w", err)
	}
	machine, err := mgr.Connect()
	if err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: connect to Windows SCM: %w", err)
	}
	defer machine.Disconnect()

	service, err := machine.CreateService(runtimeSupervisorService, executable, mgr.Config{
		DisplayName:    "Vrooli Runtime Supervisor",
		StartType:      mgr.StartAutomatic,
		ErrorControl:   mgr.ErrorNormal,
		ServiceType:    windows.SERVICE_WIN32_OWN_PROCESS,
		Description:    "Supervises Vrooli scenario runtimes",
		BinaryPathName: executable,
	})
	if err != nil {
		if !errors.Is(err, windows.ERROR_SERVICE_EXISTS) {
			return ServiceInstallResult{}, fmt.Errorf("platform: create Windows service: %w", err)
		}
		service, err = machine.OpenService(runtimeSupervisorService)
		if err != nil {
			return ServiceInstallResult{}, fmt.Errorf("platform: open existing Windows service: %w", err)
		}
	}
	defer service.Close()
	if err := service.Start(); err != nil && !errors.Is(err, windows.ERROR_SERVICE_ALREADY_RUNNING) {
		return ServiceInstallResult{}, fmt.Errorf("platform: start Windows service: %w", err)
	}
	return ServiceInstallResult{UnitName: runtimeSupervisorService, Scope: "machine", Active: true}, nil
}

func uninstallService(options ServiceInstallOptions) (ServiceInstallResult, error) {
	if !options.User {
		return ServiceInstallResult{}, fmt.Errorf("platform: system service removal requires explicit broker support")
	}
	machine, err := mgr.Connect()
	if err != nil {
		return ServiceInstallResult{}, fmt.Errorf("platform: connect to Windows SCM: %w", err)
	}
	defer machine.Disconnect()
	service, err := machine.OpenService(runtimeSupervisorService)
	if err != nil {
		if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
			return ServiceInstallResult{UnitName: runtimeSupervisorService, Scope: "machine", Active: false}, nil
		}
		return ServiceInstallResult{}, fmt.Errorf("platform: open Windows service: %w", err)
	}
	defer service.Close()
	if _, err := service.Control(svc.Stop); err != nil && !errors.Is(err, windows.ERROR_SERVICE_NOT_ACTIVE) {
		return ServiceInstallResult{}, fmt.Errorf("platform: stop Windows service: %w", err)
	}
	if err := service.Delete(); err != nil && !errors.Is(err, windows.ERROR_SERVICE_MARKED_FOR_DELETE) {
		return ServiceInstallResult{}, fmt.Errorf("platform: delete Windows service: %w", err)
	}
	return ServiceInstallResult{UnitName: runtimeSupervisorService, Scope: "machine", Active: false}, nil
}

func supportsService(user bool) bool {
	if !user {
		return false
	}
	machine, err := mgr.Connect()
	if err != nil {
		return false
	}
	_ = machine.Disconnect()
	return true
}

func serviceStartHint() string { return "sc start VrooliRuntimeSupervisor" }
