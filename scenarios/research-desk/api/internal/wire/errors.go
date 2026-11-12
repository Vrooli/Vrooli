package wire

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	"research-desk/internal/home"
)

// Error is the one mapping from an error's cause to its wire code (rulings
// E4); the message, which says what to do, is kept. An error with no known
// cause is internal and is logged with op.
func Error(logger *log.Logger, op string, err error) error {
	var cerr *connect.Error
	code := connect.CodeInternal
	switch {
	case err == nil:
		return nil
	case errors.As(err, &cerr):
		return err
	case errors.Is(err, ErrNotOperator):
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
	return connect.NewError(code, err)
}
