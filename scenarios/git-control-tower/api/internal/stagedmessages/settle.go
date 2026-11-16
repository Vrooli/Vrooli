package stagedmessages

import (
	"context"

	"git-control-tower/internal/reviewjobs"
)

// MaxSettledPerPass bounds owner reads in one background pass.
const MaxSettledPerPass = 20

// Settle advances active drafting jobs so a result is durable even when no
// browser is reading the job. It also recovers durable admissions that crashed
// before dispatch by using their saved input and the same owner idempotency key.
// Pending cancellations are included until the owner confirms a terminal state.
func (s *Service) Settle(ctx context.Context, directory func(context.Context, string) (string, error)) (int, error) {
	if s.provider == nil || directory == nil {
		return 0, nil
	}
	// Serialize sweeps so concurrent notification passes cannot repeatedly
	// restart the bounded scan at the same active jobs.
	s.settleMu.Lock()
	defer s.settleMu.Unlock()
	states := []reviewjobs.State{reviewjobs.Queued, reviewjobs.Dispatching, reviewjobs.Running, reviewjobs.AwaitingInput, reviewjobs.Cancelling}
	active, err := s.store.ListRecoverableAdvisory(ctx, operation, states, MaxSettledPerPass, s.settleCursor)
	if err != nil {
		return 0, err
	}
	if len(active) == 0 && s.settleCursor != "" {
		s.settleCursor = ""
		active, err = s.store.ListRecoverableAdvisory(ctx, operation, states, MaxSettledPerPass, "")
		if err != nil {
			return 0, err
		}
	}
	if len(active) == 0 {
		s.settleCursor = ""
		return 0, nil
	}
	s.settleCursor = active[len(active)-1].ID
	settled := 0
	for _, row := range active {
		if ctx.Err() != nil {
			return settled, ctx.Err()
		}
		job, err := s.store.Get(ctx, row.ID)
		if err != nil {
			continue
		}
		input, err := decodeDraftInput([]byte(job.InputJSON))
		if err != nil {
			continue
		}
		if row.ExecutionID != "" && !row.CancelRequested {
			execution, ownerErr := s.provider.Get(ctx, job.ExecutionID, input)
			if ownerErr != nil || !terminal(reviewjobs.State(execution.Status)) {
				continue
			}
		}
		dir, err := directory(ctx, job.RepositoryID)
		if err != nil {
			continue
		}
		// Background settlement is an explicit lifecycle advance; public GETs
		// remain observational and cannot dispatch or mutate owner state.
		action := "get"
		if row.ExecutionID == "" && !row.CancelRequested {
			action = "advance"
		} else if row.CancelRequested && row.ExecutionID == "" {
			action = "advance"
		} else if row.CancelRequested {
			action = "cancel"
		}
		projected, err := s.advanceRead(ctx, job.RepositoryID, dir, job, action, 0)
		if err == nil && terminal(reviewjobs.State(projected.Status)) {
			settled++
		}
	}
	return settled, nil
}
