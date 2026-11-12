//go:build !darwin && !windows

package platform

import (
	"context"
	"os"
)

func nativeLauncherPublisherRoot() (string, error) { return "", ErrUnsupported }
func openPublishedNativeLauncher(context.Context, string, string) (*os.File, error) {
	return nil, ErrUnsupported
}
