//go:build !darwin && !windows

package resources

func nativeOpenRGBAdoptionPaths(string) ([]string, error) { return nil, nil }
