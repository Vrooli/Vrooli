package phases

import (
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/modelregistry"
	"agent-manager/internal/orchestration/emit"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

// EventBroadcaster broadcasts events to WebSocket clients in real-time.
//
// Defined in this package (rather than the parent orchestration package) so
// the per-phase functions can reference it without creating an import cycle.
// The orchestration service's WebSocket hub adapter satisfies this interface
// structurally.
type EventBroadcaster interface {
	BroadcastEvent(event *domain.RunEvent)
	BroadcastRunStatus(run *domain.Run)
	BroadcastProgress(runID uuid.UUID, phase domain.RunPhase, percent int, action string)
}

// ModelChainResolver returns the ordered preset chain for a runner+preset pair.
// Implemented by modelregistry.Store. Injected so model-level fallback
// can walk the chain at execution time without persisting derived state on the run.
type ModelChainResolver interface {
	ResolvePreset(runner string, preset string) (modelregistry.PresetChain, bool)
}

// ModelHealthReporter receives runtime classifications of model availability.
// Implemented by the orchestration health-store adapter so the executor does
// not import modelregistry's HealthStore type directly (keeps the seam small).
type ModelHealthReporter interface {
	MarkModelHealthy(runnerType, modelID string)
	MarkModelUnavailable(runnerType, modelID, message string)
}

// Deps bundles the common dependencies every phase shares. Each phase's
// input struct carries a Deps so the call site is explicit about what the
// phase touches without each phase repeating the same handful of fields.
//
// Deps is constructed once per run by RunExecutor and passed read-only into
// each phase. Phase mutations to the run/checkpoint happen via pointers
// carried in each phase's specific input struct, not via Deps.
type Deps struct {
	Runs        repository.RunRepository
	Events      event.Store
	Broadcaster EventBroadcaster
	Checkpoints repository.CheckpointRepository
	Gate        *emit.Gate
	Levers      config.Levers
}
