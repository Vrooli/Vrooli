//go:build windows

package changedetect

// isCharDevWhiteout is always false on Windows: overlayfs whiteouts are a
// Linux overlay filesystem construct (a rdev=0 character device) that
// cannot exist on a Windows filesystem, so the overlay change detector has
// no deletions to recognize this way.
func isCharDevWhiteout(string) bool {
	return false
}
