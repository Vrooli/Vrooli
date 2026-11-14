//go:build darwin

package platform

import (
	"context"
	"os"
)

func enrolledNativeOperatorResourceStorageOptions(ctx context.Context) (NativeSystemServiceBrokerOptions, bool, error) {
	options, configured, err := nativeDarwinSystemBrokerEngine().enrolledResourceStorageOptions(ctx)
	if err != nil || !configured {
		return options, configured, err
	}
	if os.Geteuid() != 0 && uint32(os.Geteuid()) != options.OperatorUID {
		return NativeSystemServiceBrokerOptions{}, false, ErrOperationIdentity
	}
	return options, true, nil
}
