//go:build !linux && !darwin && !windows

package platform

func processAliveObserved(int) (bool, error) { return false, ErrUnsupported }
