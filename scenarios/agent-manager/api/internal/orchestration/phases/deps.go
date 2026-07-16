package phases

import (
	"context"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"
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

// ModelHealthReporter receives runtime classifications of model availability.
// Implemented by the orchestration health-store adapter.
type ModelHealthReporter interface {
	MarkModelHealthy(runnerType, modelID string)
	MarkModelUnavailable(runnerType, modelID, message string)
}

// WorkspaceSandboxEnsurer is the run-time dependency seam for making the
// workspace-sandbox scenario available when a sandboxed run needs it.
//
// It must not create sandboxes or apply diffs; sandbox.Provider owns HTTP
// operations. Production implementations delegate process startup to Vrooli
// lifecycle so lifecycle's cross-process scenario lock remains authoritative.
type WorkspaceSandboxEnsurer interface {
	EnsureAvailable(ctx context.Context) error
}

// StructuredResultResolver applies the typed-output policy to an already
// canonical final result. Implementations may use a constrained extractor, but
// must locally validate every candidate before returning success.
type StructuredResultResolver interface {
	Resolve(context.Context, *domain.ResultSpec, *domain.RunResult) *domain.StructuredResult
}

// Deps bundles the common dependencies every phase shares. Each phase's
// input struct carries a Deps so the call site is explicit about what the
// phase touches without each phase repeating the same handful of fields.
//
// Deps is constructed once per run by RunExecutor and passed read-only into
// each phase. Phase mutations to the run/checkpoint happen via pointers
// carried in each phase's specific input struct, not via Deps.
type Deps struct {
	Runs              repository.RunRepository
	Events            event.Store
	Broadcaster       EventBroadcaster
	Checkpoints       repository.CheckpointRepository
	Gate              *emit.Gate
	Levers            config.Levers
	WorkspaceSandbox  WorkspaceSandboxEnsurer
	StructuredResults StructuredResultResolver
}
