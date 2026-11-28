//go:build windows

package resources

import "golang.org/x/sys/windows"

func replacePath(source, destination string) error {
	return windows.Rename(source, destination)
}
