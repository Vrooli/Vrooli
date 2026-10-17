//go:build !linux && !darwin && !windows

package platform

import "fmt"

func installService(ServiceInstallOptions) (ServiceInstallResult, error) {
	return ServiceInstallResult{}, fmt.Errorf("platform: service supervisor unsupported on this operating system")
}

func uninstallService(ServiceInstallOptions) (ServiceInstallResult, error) {
	return ServiceInstallResult{}, fmt.Errorf("platform: service supervisor unsupported on this operating system")
}
func startInstalledService(ServiceInstallOptions) (bool, error) { return false, nil }
func supportsService(bool) bool                                 { return false }
func serviceStartHint() string                                  { return "" }

func readHostLogs(HostLogOptions) (HostLogResult, error) {
	return HostLogResult{}, fmt.Errorf("platform: host logs unsupported")
}
