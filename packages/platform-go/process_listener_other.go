//go:build !windows && !darwin

package platform

func nativeProcessTCPListeners(int) ([]ProcessTCPListener, error) { return nil, ErrUnsupported }

func nativeTCPListeners(uint16) ([]ProcessTCPListener, error) { return nil, ErrUnsupported }
