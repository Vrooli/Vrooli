//go:build !linux

package baselinefloor

import "io/fs"

// cloneFile is a no-op on non-Linux platforms: there is no portable reflink
// primitive, so the copy ladder always uses the native-Go deep-copy floor.
// (macOS APFS clonefile() is a future accelerator behind its own build tag; the
// portable floor already works everywhere, so it is not required for correctness.)
//
// Returning (false, nil) tells copyRegularFile to fall back to the deep copy.
func cloneFile(dst, src string, mode fs.FileMode) (bool, error) {
	return false, nil
}
