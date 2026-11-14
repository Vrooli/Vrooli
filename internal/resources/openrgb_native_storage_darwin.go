//go:build darwin

package resources

func nativeOpenRGBMachineRoot() (string, error) {
	return "/Library/Application Support/Vrooli/resources/openrgb", nil
}
