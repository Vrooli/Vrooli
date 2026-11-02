//go:build !unix

package services

import "errors"

func diskFreeBytes(string) (uint64, error) {
	return 0, errors.New("free disk measurement is not supported on this platform")
}
