//go:build !windows && !darwin

package platform

import (
	"context"
	"os"
)

func openCurrentNativeSystemSupervisorImage(context.Context) (*os.File, error) {
	return nil, ErrUnsupported
}
func openInstalledNativeSupervisorImage(context.Context) (*os.File, error) {
	return nil, ErrUnsupported
}
func openOwnedServiceRootImage(context.Context, ServiceRef) (*os.File, error) {
	return nil, ErrUnsupported
}
func openOwnedServiceChildImage(context.Context, ServiceRef, OwnedImageChildIdentity) (*os.File, error) {
	return nil, ErrUnsupported
}
