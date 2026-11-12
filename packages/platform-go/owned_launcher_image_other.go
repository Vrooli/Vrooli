//go:build !windows

package platform

import "os"

func openOwnedLauncherImage(ref ServiceRef) (*os.File, error) {
	return OpenProcessExecutable(ref.RootPID)
}
