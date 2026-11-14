//go:build !darwin && !windows

package platform

import "context"

func readNativeSystemServiceBrokerDescriptor(string) (NativeSystemServiceBrokerOptions, error) {
	return NativeSystemServiceBrokerOptions{}, ErrUnsupported
}
func runNativeSystemSupervisor(context.Context, NativeSystemServiceBrokerOptions, func(context.Context) error) error {
	return ErrUnsupported
}
