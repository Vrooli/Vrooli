// Phase-ladder advancement and checkpoint persistence helpers.
//
// AdvancePhase is the single seam every phase calls to update the
// run.Phase ladder. Persistence to the checkpoint store and run repository
// is best-effort: failures emit a warn event but do not block the in-memory
// advance. The nil-guards on Deps.Runs/Deps.Checkpoints mirror each other so
// unit tests that drive a partially-wired phase function don't have to
// satisfy every persistence seam.

package phases

import (
	"context"
	"fmt"
	"sync"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// AdvancePhaseInput is the explicit input to AdvancePhase.
type AdvancePhaseInput struct {
	Deps       Deps
	Run        *domain.Run
	Checkpoint *domain.RunCheckpoint
	Mu         *sync.Mutex
	Phase      domain.RunPhase
}

// AdvancePhase updates the in-memory checkpoint to a new phase, persists
// it, and emits the corresponding info event. Persistence failures are
// warned but not load-bearing.
//
// When in.Mu is non-nil the run/checkpoint mutations happen under that
// lock so the heartbeat goroutine can't race a phase advance.
func AdvancePhase(ctx context.Context, in AdvancePhaseInput) {
	if in.Mu != nil {
		in.Mu.Lock()
	}
	if in.Checkpoint != nil {
		updated := in.Checkpoint.Update(in.Phase, 0)
		*in.Checkpoint = *updated
	}
	if in.Run != nil {
		in.Run.UpdateProgress(in.Phase, domain.PhaseToProgress(in.Phase))
	}
	if in.Mu != nil {
		in.Mu.Unlock()
	}

	var runID uuid.UUID
	if in.Run != nil {
		runID = in.Run.ID
	}
	SaveCheckpoint(ctx, SaveCheckpointInput{
		Deps:       in.Deps,
		Checkpoint: in.Checkpoint,
		Mu:         in.Mu,
		RunID:      runID,
	})

	if in.Deps.Runs != nil && in.Run != nil {
		if err := in.Deps.Runs.Update(ctx, in.Run); err != nil {
			EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn",
				"failed to persist phase update: "+err.Error())
		}
	}

	if in.Run != nil {
		EmitSystemEvent(ctx, in.Deps, in.Run.ID, "info",
			fmt.Sprintf("phase: %s", in.Phase.Description()))
	}
}

// SaveCheckpointInput is the explicit input to SaveCheckpoint.
type SaveCheckpointInput struct {
	Deps       Deps
	Checkpoint *domain.RunCheckpoint
	Mu         *sync.Mutex
	RunID      uuid.UUID
}

// SaveCheckpoint persists the supplied checkpoint if the repository is
// configured. It is best-effort and logs but does not return errors.
func SaveCheckpoint(ctx context.Context, in SaveCheckpointInput) {
	if in.Deps.Checkpoints == nil || in.Checkpoint == nil {
		return
	}
	if in.Mu != nil {
		in.Mu.Lock()
	}
	cp := *in.Checkpoint
	if in.Mu != nil {
		in.Mu.Unlock()
	}
	if err := in.Deps.Checkpoints.Save(ctx, &cp); err != nil {
		EmitSystemEvent(ctx, in.Deps, in.RunID, "warn",
			"failed to save checkpoint: "+err.Error())
	}
}
