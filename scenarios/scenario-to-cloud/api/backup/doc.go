// Package backup is the cloud-side owner of data recovery for a deployment:
// recovery points, restores, verification, schema-aware rollback admission,
// migration posture and retention.
//
// Consistency and encryption live in packages/recoverypoint (shared with the
// target-local `vrooli cloud-target data` verbs) so the cloud side and the
// target read and write one on-disk format. This package adds what only the
// cloud side knows: the deployment record, the closure's declared bindings,
// the release/schema/configuration/credential-version references a point is
// bound to, the backup owner (data-backup-manager) registration, protection
// by active releases and non-terminal operations, and the measured RTO/RPO
// against certification/budgets.json.
//
// Rules:
//
//   - A database binding is captured only through its owner's declared
//     database-native tooling; a generic file copy is never a backup.
//   - A recovery point carries references only. The recovery key is a
//     reference resolvable outside the target's failure domain, never material.
//   - Restore never writes into a binding that already holds data
//     (restore_target_not_clean) and never reports success for a subset.
//   - A code rollback across an incompatible schema is refused with a typed
//     forward-repair or restore plan; a destructive schema change without a
//     recovery point is refused with recovery_point_required.
//   - A recovery point referenced by an active or retained release or by a
//     non-terminal operation is protected and cannot be deleted.
package backup
