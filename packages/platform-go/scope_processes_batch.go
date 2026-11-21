package platform

import "context"

// ScopeProcessObservation is one aligned result from ScopeProcessesBatchContext.
// Err is non-nil unless membership is readable or the exact collected scope is
// proven gone and inactive.
type ScopeProcessObservation struct {
	PIDs []int
	Err  error
}

// ScopeProcessesBatchContext reads several exact scopes together. On Linux,
// scopes whose cgroups are gone share bounded systemd state queries, avoiding
// one systemctl process per historical scope while retaining the exact
// inactive-unit proof required for absence.
func ScopeProcessesBatchContext(ctx context.Context, refs []ScopeRef) []ScopeProcessObservation {
	return scopeProcessesBatchContext(ctx, refs)
}
