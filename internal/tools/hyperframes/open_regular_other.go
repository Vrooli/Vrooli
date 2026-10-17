//go:build !unix

package hyperframes

import "os"

// Linux is the declared execution target. The common adapter checks
// types/inodes elsewhere; Unix-specific nonblocking replacement evidence does
// not establish a non-Unix guarantee.
func openRegularFile(path string) (*os.File, error) { return os.Open(path) }
