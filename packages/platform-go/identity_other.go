//go:build !linux && !darwin && !windows

package platform

import "context"

func runAsInvokingUserInSession(context.Context, string, []string, IdentityCommandOptions) error {
	return ErrSessionExecutionUnsupported
}

func runAsInvokingUserInSessionWithInput(context.Context, string, []string, []byte, IdentityCommandOptions) error {
	return ErrSessionExecutionUnsupported
}
