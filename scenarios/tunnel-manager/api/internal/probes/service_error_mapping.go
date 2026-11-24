package probes

import "github.com/vrooli/api-core/connectx"

// ToConnectError keeps the probes package's handler-facing translation point
// while using the shared internal-error fallback. Probe operations currently
// have no typed sentinels and fail only on repository or transport reads.
func ToConnectError(err error) error {
	return connectx.WrapError(err, nil)
}
