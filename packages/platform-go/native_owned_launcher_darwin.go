//go:build darwin

package platform

import (
	"context"
	"os"
	"runtime"
)

func nativeLauncherPublisherRoot() (string, error) {
	return "/Library/PrivilegedHelperTools/com.vrooli.openrgb/launchers", nil
}
func openPublishedNativeLauncher(ctx context.Context, path, digest string) (*os.File, error) {
	return (darwinPublishedLauncherReader{
		path:   verifyImmutableNativePath,
		format: func(file *os.File) error { return validateDarwinBrokerMachO(file, runtime.GOARCH) },
	}).open(ctx, path, digest)
}
