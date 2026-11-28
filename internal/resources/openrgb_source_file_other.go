//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package resources

import "os"

func openOpenRGBSourceFileNative(path string) (*os.File, error) { return os.Open(path) }
