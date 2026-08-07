//go:build windows

package hostreqkit

import "errors"

// ErrSessionExecutionUnsupported is returned when a handler asks the
// cross-platform session seam to perform a Unix session operation on Windows.
// Callers must surface a manual/unsupported result instead of treating the
// operation as successfully executed.
var ErrSessionExecutionUnsupported = errors.New("invoking-user session execution is unsupported on Windows")

func RunAsInvokingUserWithSession(string, []string, EnsureOptions) error {
	return ErrSessionExecutionUnsupported
}

func RunAsInvokingUserWithSessionOutput(string, []string, EnsureOptions) ([]byte, error) {
	return nil, ErrSessionExecutionUnsupported
}
