//go:build !windows

package platform

func nativeProcessSecurityProof(int) (string, error) { return "", ErrUnsupported }
