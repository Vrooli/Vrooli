package wire

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"

	"research-desk/internal/home"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/shared"
)

// Error is the one mapping from an error's cause to its wire code; the
// message, which says what is wrong and what the operator can
// choose, is kept. A command-line next step the error carries
// (home.WithCommand) travels as a CommandHint detail, never in the message.
// An error with no known cause is internal and is logged with
// op.
func Error(logger *log.Logger, op string, err error) error {
	var cerr *connect.Error
	code := connect.CodeInternal
	switch {
	case err == nil:
		return nil
	case errors.As(err, &cerr):
		return err
	case errors.Is(err, ErrNotOperator), errors.Is(err, authn.ErrOperation), errors.Is(err, authn.ErrCapability):
		code = connect.CodePermissionDenied
	case errors.Is(err, home.ErrNotFound):
		code = connect.CodeNotFound
	case errors.Is(err, home.ErrInvalid):
		code = connect.CodeInvalidArgument
	case errors.Is(err, home.ErrRefused):
		code = connect.CodeFailedPrecondition
	case errors.Is(err, home.ErrUnavailable), errors.Is(err, context.DeadlineExceeded):
		code = connect.CodeUnavailable
	default:
		if logger == nil {
			logger = log.Default()
		}
		logger.Printf("%s: %v", op, err)
	}
	out := connect.NewError(code, err)
	if command := home.Command(err); command != "" {
		if detail, derr := connect.NewErrorDetail(&sharedv1.CommandHint{Command: command}); derr == nil {
			out.AddDetail(detail)
		}
	}
	return out
}
