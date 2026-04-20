package orchestration

import "agent-manager/internal/modelregistry"

// healthMarkerAdapter bridges the executor's ModelHealthReporter seam to the
// modelregistry.HealthStore. Kept in this package to prevent the executor from
// importing the concrete store type — the executor only needs the narrow seam.
type healthMarkerAdapter struct {
	store *modelregistry.HealthStore
}

func newHealthMarkerAdapter(store *modelregistry.HealthStore) *healthMarkerAdapter {
	return &healthMarkerAdapter{store: store}
}

func (a *healthMarkerAdapter) MarkModelHealthy(runnerType, modelID string) {
	if a == nil || a.store == nil {
		return
	}
	a.store.Mark(runnerType, modelID, modelregistry.ModelHealthOK, "")
}

func (a *healthMarkerAdapter) MarkModelUnavailable(runnerType, modelID, message string) {
	if a == nil || a.store == nil {
		return
	}
	a.store.Mark(runnerType, modelID, modelregistry.ModelHealthFailed, message)
}
