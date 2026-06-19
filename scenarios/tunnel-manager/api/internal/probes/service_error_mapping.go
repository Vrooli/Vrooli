package probes

import (
	"connectrpc.com/connect"
)

// ToConnectError translates domain errors into Connect's typed error
// model. The probes domain has no input-validation or not-found sentinels
// today — RunProbes/ListProbes/Classify either succeed or fail on a
// repository/transport read — so every error maps to internal and callers
// never depend on raw storage details. Typed sentinels added later (e.g.
// an ErrInvalidProbeFilter) get their own branch above the catch-all,
// mirroring internal/routes/service_error_mapping.go.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	return connect.NewError(connect.CodeInternal, err)
}
