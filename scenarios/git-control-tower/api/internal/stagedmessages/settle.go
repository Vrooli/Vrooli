package stagedmessages

import (
	"context"
	"encoding/json"

	"git-control-tower/internal/reviewjobs"
)

// MaxSettledPerPass bounds owner reads in one background pass.
const MaxSettledPerPass = 20

// Settle advances dispatched drafting jobs whose owner execution has ended,
// so a result is durable even when no browser is reading the job. It only
// reads the owner: a job without an execution handle or with a pending
// cancellation is left to the operator's own requests, and an owner read
// error changes nothing here, so a background outage never records an error
// on a job the operator is following. It reports how many jobs it settled.
func (s *Service) Settle(ctx context.Context, directory func(context.Context, string) (string, error)) (int, error) {
	if s.provider == nil || directory == nil {
		return 0, nil
	}
	active, err := s.store.ListAdvisory(ctx, operation, []reviewjobs.State{reviewjobs.Queued, reviewjobs.Running}, MaxSettledPerPass)
	if err != nil {
		return 0, err
	}
	settled := 0
	for _, row := range active {
		if ctx.Err() != nil {
			return settled, ctx.Err()
		}
		if row.ExecutionID == "" || row.CancelRequested {
			continue
		}
		job, err := s.store.Get(ctx, row.ID)
		if err != nil {
			continue
		}
		var input DraftInput
		if json.Unmarshal([]byte(job.InputJSON), &input) != nil {
			continue
		}
		ex, err := s.provider.Get(ctx, job.ExecutionID, input)
		if err != nil || !terminal(reviewjobs.State(ex.Status)) {
			continue
		}
		dir, err := directory(ctx, job.RepositoryID)
		if err != nil {
			continue
		}
		// The ordinary read path owns validation and the lifecycle CAS.
		if _, err := s.Job(ctx, job.RepositoryID, dir, job.ID, "", "get", 0); err == nil {
			settled++
		}
	}
	return settled, nil
}
