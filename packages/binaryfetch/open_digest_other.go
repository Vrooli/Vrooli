//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package binaryfetch

import "os"

// The portable adapter checks regular type/identity after open. It does not
// promise to interrupt an OS open if a special file replaces the input.
func openDigestFile(path string) (*os.File, error) { return os.Open(path) }
