package flows

import (
	"connectrpc.com/connect"
)

// ToConnectError translates flows-domain failures into Connect codes.
// Discovery failures (unknown flow id, missing contract path, schema
// error) all surface as flat errors today; we map any error containing
// "not found" or "no flow" to NotFound, everything else to Internal.
// Real sentinels can be added later without changing call sites.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	if isNotFound(err) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	if isInvalid(err) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func isNotFound(err error) bool {
	msg := err.Error()
	return contains(msg, "no flow matches") || contains(msg, "not found") || contains(msg, "no contract")
}

func isInvalid(err error) bool {
	msg := err.Error()
	return contains(msg, "requires a flow id") || contains(msg, "must be one of") || contains(msg, "invalid language")
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
