//go:build !linux && !darwin && !windows

package hostreqkit

import "errors"

var ErrSessionExecutionUnsupported = errors.New("invoking-user session execution is unsupported on this platform")

func RunAsInvokingUserWithSession(string, []string, EnsureOptions) error {
	return ErrSessionExecutionUnsupported
}

func RunAsInvokingUserWithSessionOutput(string, []string, EnsureOptions) ([]byte, error) {
	return nil, ErrSessionExecutionUnsupported
}
