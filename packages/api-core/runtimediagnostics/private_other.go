//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd && !dragonfly

package runtimediagnostics

import "errors"

func ensurePrivateDirectory(string) error {
	return errors.New("private runtime diagnostics sockets are unsupported on this platform")
}
