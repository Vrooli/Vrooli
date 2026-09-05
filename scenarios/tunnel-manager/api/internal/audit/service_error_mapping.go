package audit

import (
	"connectrpc.com/connect"
)

// ToConnectError translates audit domain errors into Connect's typed error
// model. The audit service declares no typed sentinels of its own — per-route
// problems are reported as findings, not errors, so the only error path is a
// manifest read failure, which maps to internal. Kept as a function (mirroring
// internal/routes) so the handler has a single translation point and a new
// sentinel only needs a branch here.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	return connect.NewError(connect.CodeInternal, err)
}
