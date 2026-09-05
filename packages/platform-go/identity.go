package platform

import (
	"bytes"
	"context"
	"errors"
	"io"
)

// ErrSessionExecutionUnsupported means the target platform has no supported
// way to enter the invoking user's interactive session. Callers must surface
// the typed limitation instead of silently running under the wrong identity.
var ErrSessionExecutionUnsupported = errors.New("platform: invoking-user session execution is unsupported")

// IdentityCommandOptions carries process streams without exposing secrets in
// argv or environment variables. Input is connected directly to stdin.
type IdentityCommandOptions struct {
	// Dir is the working directory for the command. It is especially important
	// for project-scoped control-plane commands launched after dropping from
	// sudo to the invoking operator.
	Dir    string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// RunAsInvokingUserInSession runs name and args as the operator who invoked
// the elevated process, inside that operator's native login/session context.
// It is the single seam for credential-store and keyring operations.
func RunAsInvokingUserInSession(ctx context.Context, name string, args []string, options IdentityCommandOptions) error {
	return runAsInvokingUserInSession(ctx, name, args, options)
}

// RunAsInvokingUserInSessionWithInput is the secret-safe variant. The input is
// never copied into argv, environment, or a temporary file.
func RunAsInvokingUserInSessionWithInput(ctx context.Context, name string, args []string, input []byte, options IdentityCommandOptions) error {
	return runAsInvokingUserInSessionWithInput(ctx, name, args, input, options)
}

// RunAsInvokingUserInSessionOutput is the bounded-output counterpart used by
// session-aware probes. It keeps the same identity/session policy while
// making the captured output explicit at the call site.
func RunAsInvokingUserInSessionOutput(ctx context.Context, name string, args []string, options IdentityCommandOptions) ([]byte, error) {
	var output bytes.Buffer
	options.Stdout = &output
	if err := RunAsInvokingUserInSession(ctx, name, args, options); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
