//go:build !unix

package packagegov

import "io/fs"

// linkCount reports a single link where the platform does not expose link
// counts; DetachShippedFiles then leaves files in place.
func linkCount(fs.FileInfo) uint64 { return 1 }
