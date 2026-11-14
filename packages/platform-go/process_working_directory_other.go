//go:build !windows

package platform

import "os"

func processWorkingDirectoryMatches(pid int, expected string) (bool, error) {
	path, err := ProcessWorkingDir(pid)
	if err != nil {
		return false, err
	}
	actual, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	wanted, err := os.Stat(expected)
	if err != nil {
		return false, err
	}
	return actual.IsDir() && wanted.IsDir() && os.SameFile(actual, wanted), nil
}
