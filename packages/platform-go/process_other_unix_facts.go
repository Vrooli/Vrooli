//go:build unix && !linux && !darwin

package platform

func processWorkingDir(int) (string, error) { return "", ErrUnsupported }

func processHasChildren(int) (bool, error) { return false, ErrUnsupported }
