package orchestration

import (
	"context"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
)

// panickingRunRepo panics on List to simulate a defect inside a reconciler
// sweep. All other RunRepository methods panic via the nil embedded interface,
// which is equally acceptable for this test: any panic must be contained.
type panickingRunRepo struct {
	repository.RunRepository
}

func (panickingRunRepo) List(context.Context, repository.RunListFilter) ([]*domain.Run, error) {
	panic("simulated reconciler sweep defect")
}

// A panic inside any reconciler sweep must be contained by reconcileGuarded so
// the reconciliation loop — the process-wide recovery safety net — keeps
// ticking instead of killing the API.
func TestReconcileGuarded_ContainsSweepPanic(t *testing.T) {
	rec := NewReconciler(panickingRunRepo{}, nil)

	defer func() {
		if v := recover(); v != nil {
			t.Fatalf("reconcileGuarded let a sweep panic escape: %v", v)
		}
	}()
	stats := rec.reconcileGuarded(context.Background())
	if stats.RunsChecked != 0 {
		t.Errorf("stats.RunsChecked = %d, want 0 from an aborted cycle", stats.RunsChecked)
	}
}
