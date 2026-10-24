package coverage

import "connectrpc.com/connect"

// ToConnectError translates coverage errors into Connect's typed error model.
// Coverage has no caller-supplied input to validate (both RPCs take only
// booleans), so every error here is an internal composition failure from a
// downstream seam. Per-item accept failures are NOT errors — they ride back in
// AcceptResult.Failed so partial failures are never swallowed.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	return connect.NewError(connect.CodeInternal, err)
}
