//go:build darwin

package resources

func nativeOpenRGBAdoptionPaths(home string) ([]string, error) {
	return openRGBMacAdoptionPaths(home), nil
}
