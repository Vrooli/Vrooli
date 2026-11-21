package resources

import (
	"errors"
	"os"
)

// A mutable acquired/adopted source is never execution authority. Pin a
// regular opened object before copying it into the privileged pending store;
// the caller still proves every copied byte against the release profile.
func openOpenRGBSourceFile(path string, expected os.FileInfo) (*os.File, error) {
	var err error
	if expected == nil {
		expected, err = os.Stat(path)
		if err != nil {
			return nil, err
		}
	}
	if !expected.Mode().IsRegular() {
		return nil, errors.New("OpenRGB source is not a regular file")
	}
	file, err := openOpenRGBSourceFileNative(path)
	if err != nil {
		return nil, err
	}
	actual, err := file.Stat()
	if err != nil || !actual.Mode().IsRegular() || !os.SameFile(expected, actual) {
		file.Close()
		return nil, errors.Join(errors.New("OpenRGB source object changed before copying"), err)
	}
	return file, nil
}
