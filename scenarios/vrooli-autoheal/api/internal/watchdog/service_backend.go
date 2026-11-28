package watchdog

import platformgo "github.com/vrooli/platform-go"

type serviceBackend interface {
	Status(platformgo.NativeServiceOptions) (platformgo.NativeServiceResult, error)
}

type nativeServiceBackend struct{}

func (nativeServiceBackend) Status(options platformgo.NativeServiceOptions) (platformgo.NativeServiceResult, error) {
	return platformgo.NativeServiceStatus(options)
}
