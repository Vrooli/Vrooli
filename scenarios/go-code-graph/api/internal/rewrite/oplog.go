package rewrite

import "context"

// OperationLog records the per-op outcome of every non-dry-run Apply.
// Production wires *SQLiteStore (which satisfies both PlanStore and
// OperationLog); tests wire NoopOperationLog to skip the write.
//
// Dry-run callers MUST NOT trigger RecordApply — the synthetic
// success/echo loop in Service.Apply short-circuits before the
// executor runs and would record misleading rows.
type OperationLog interface {
	RecordApply(ctx context.Context, planID PlanID, results []OperationResult) error
}

// NoopOperationLog is the test/seam default — discards every call.
type NoopOperationLog struct{}

// RecordApply is a no-op.
func (NoopOperationLog) RecordApply(context.Context, PlanID, []OperationResult) error { return nil }

// Compile-time assertion.
var _ OperationLog = NoopOperationLog{}
