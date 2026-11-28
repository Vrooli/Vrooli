//go:build !linux && !darwin && !windows

package platform

import "fmt"

func installNativeService(NativeServiceOptions) (NativeServiceResult, error) {
	return NativeServiceResult{}, fmt.Errorf("platform: native services unsupported")
}

func uninstallNativeService(NativeServiceOptions) (NativeServiceResult, error) {
	return NativeServiceResult{}, fmt.Errorf("platform: native services unsupported")
}

func startNativeService(NativeServiceOptions) error {
	return fmt.Errorf("platform: native services unsupported")
}

func stopNativeService(NativeServiceOptions) error {
	return fmt.Errorf("platform: native services unsupported")
}

func restartNativeService(NativeServiceOptions) error {
	return fmt.Errorf("platform: native services unsupported")
}

func nativeServiceStatus(NativeServiceOptions) (NativeServiceResult, error) {
	return NativeServiceResult{}, fmt.Errorf("platform: native services unsupported")
}

func nativeServiceLogs(NativeServiceOptions, int) ([]byte, error) {
	return nil, fmt.Errorf("platform: native service logs unsupported")
}
