//go:build !darwin && !windows

package platform

import "context"

func enrolledNativeOperatorResourceStorageOptions(context.Context) (NativeSystemServiceBrokerOptions, bool, error) {
	return NativeSystemServiceBrokerOptions{}, false, ErrUnsupported
}
