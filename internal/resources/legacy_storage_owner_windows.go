//go:build windows

package resources

import "os"

// Windows does not expose POSIX ownership metadata through os.FileInfo. The
// migration therefore remains fail-closed instead of guessing whether a
// legacy mount is owned by the invoking user.
func legacyStorageOwnerUID(os.FileInfo) (uint32, bool) {
	return 0, false
}
