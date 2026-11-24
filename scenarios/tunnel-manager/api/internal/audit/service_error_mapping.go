package audit

import "github.com/vrooli/api-core/connectx"

// ToConnectError keeps the audit package's handler-facing translation point
// while using the shared internal-error fallback. Audit failures have no typed
// sentinels; per-route problems are returned as findings.
func ToConnectError(err error) error {
	return connectx.WrapError(err, nil)
}
