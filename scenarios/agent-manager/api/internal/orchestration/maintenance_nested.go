package orchestration

import (
	"context"
	"sync/atomic"
)

type maintenanceAdmissionKey struct{}

// This invocation-local capability cannot be established by a public request.
// It belongs to one owner and expires before that owner's real admission ends.
type maintenanceAdmission struct {
	owner  *Orchestrator
	active atomic.Bool
}

func (o *Orchestrator) hasMaintenanceAdmission(ctx context.Context) bool {
	admitted, ok := ctx.Value(maintenanceAdmissionKey{}).(*maintenanceAdmission)
	return ok && admitted != nil && admitted.owner == o && admitted.active.Load()
}

// admitMaintenanceContext covers a synchronous owner operation and its nested
// admissions with one gate hold. A nested caller cannot extend its lifetime.
func (o *Orchestrator) admitMaintenanceContext(ctx context.Context) (context.Context, func(), error) {
	if o.hasMaintenanceAdmission(ctx) {
		return ctx, func() {}, nil
	}
	release, err := o.admitMaintenance(ctx)
	if err != nil {
		return nil, nil, err
	}
	admitted := &maintenanceAdmission{owner: o}
	admitted.active.Store(true)
	return context.WithValue(ctx, maintenanceAdmissionKey{}, admitted), func() {
		if admitted.active.Swap(false) {
			release()
		}
	}, nil
}
