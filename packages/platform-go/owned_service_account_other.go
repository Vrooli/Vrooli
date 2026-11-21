//go:build !darwin && !windows

package platform

import "context"

func resolveOwnedServiceAccount(context.Context, string) (OwnedServiceAccount, error) {
	return OwnedServiceAccount{}, ErrUnsupported
}

func prepareOwnedServiceAccount(context.Context, string) (OwnedServiceAccount, error) {
	return OwnedServiceAccount{}, ErrUnsupported
}
