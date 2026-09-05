package runtime

import "github.com/vrooli/api-core/retention"

// RetentionConflict remains available from the runtime package for callers
// that used the original validation seam. The implementation is shared with
// storage-manager through api-core/retention so the validator cannot drift.
type RetentionConflict = retention.RetentionConflict

// ValidateRetentionAgainstDurableData delegates to the canonical retention
// contract validator.
func ValidateRetentionAgainstDurableData(manifest []byte) ([]RetentionConflict, error) {
	return retention.ValidateRetentionAgainstDurableData(manifest)
}
