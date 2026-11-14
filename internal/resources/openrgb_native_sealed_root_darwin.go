//go:build darwin

package resources

func nativeOpenRGBSealedRoot() (string, error) {
	return "/Library/PrivilegedHelperTools/com.vrooli.openrgb", nil
}
