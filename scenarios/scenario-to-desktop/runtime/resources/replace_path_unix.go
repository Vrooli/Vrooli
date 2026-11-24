//go:build !windows

package resources

import "os"

func replacePath(source, destination string) error {
	return os.Rename(source, destination)
}
