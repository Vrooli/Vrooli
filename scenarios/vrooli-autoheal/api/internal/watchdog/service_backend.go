package watchdog

import platformgo "github.com/vrooli/platform-go"

type serviceBackend interface {
	Install(platformgo.NativeServiceOptions) (platformgo.NativeServiceResult, error)
	Uninstall(platformgo.NativeServiceOptions) (platformgo.NativeServiceResult, error)
	Status(platformgo.NativeServiceOptions) (platformgo.NativeServiceResult, error)
}

type nativeServiceBackend struct{}

func (nativeServiceBackend) Install(options platformgo.NativeServiceOptions) (platformgo.NativeServiceResult, error) {
	return platformgo.InstallNativeService(options)
}

func (nativeServiceBackend) Uninstall(options platformgo.NativeServiceOptions) (platformgo.NativeServiceResult, error) {
	return platformgo.UninstallNativeService(options)
}

func (nativeServiceBackend) Status(options platformgo.NativeServiceOptions) (platformgo.NativeServiceResult, error) {
	return platformgo.NativeServiceStatus(options)
}
