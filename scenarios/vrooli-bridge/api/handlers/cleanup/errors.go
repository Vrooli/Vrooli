package cleanup

import "connectrpc.com/connect"

func connectErrorInvalid(err error) error { return connect.NewError(connect.CodeInvalidArgument, err) }
func connectErrorFailedPrecondition(err error) error {
	return connect.NewError(connect.CodeFailedPrecondition, err)
}
func connectErrorUnavailable(err error) error { return connect.NewError(connect.CodeUnavailable, err) }
func connectErrorAlreadyExists(err error) error {
	return connect.NewError(connect.CodeAlreadyExists, err)
}
