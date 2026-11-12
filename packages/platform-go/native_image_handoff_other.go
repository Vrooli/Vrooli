//go:build !darwin && !windows

package platform

import "context"

func verifyImmutableNativePath(context.Context, string, bool) error { return ErrUnsupported }
func frozenOwnedServiceEnvironment(context.Context, ServiceRef) ([]string, error) {
	return nil, ErrUnsupported
}
