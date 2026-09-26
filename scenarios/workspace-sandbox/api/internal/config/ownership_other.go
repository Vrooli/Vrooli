//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris && !windows

package config

import "os"

func validateAuthoritativeDirectory(_ string, _ os.FileInfo) error { return nil }
