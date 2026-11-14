//go:build !windows

package platform

import "context"

func prepareOwnedServicePeerObservation(context.Context, string) (func() error, error) {
	return func() error { return nil }, nil
}
