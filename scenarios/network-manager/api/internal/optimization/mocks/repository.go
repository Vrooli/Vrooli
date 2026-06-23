package mocks

import (
	"context"
	"sync"

	"network-manager/internal/optimization"
)

type Repository struct {
	mu   sync.Mutex
	runs map[string]optimization.Run
}

func NewRepository() *Repository {
	return &Repository{runs: map[string]optimization.Run{}}
}

func (r *Repository) SaveRun(_ context.Context, run optimization.Run) (optimization.Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run.Candidates = nil
	r.runs[run.ID] = run
	return run, nil
}

func (r *Repository) GetRun(_ context.Context, id string) (optimization.Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[id]
	if !ok {
		return optimization.Run{}, optimization.ErrNotFound
	}
	return cloneRun(run), nil
}

func (r *Repository) UpdateRun(_ context.Context, run optimization.Run) (optimization.Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.runs[run.ID]
	if !ok {
		return optimization.Run{}, optimization.ErrNotFound
	}
	run.Candidates = current.Candidates
	r.runs[run.ID] = run
	return run, nil
}

func (r *Repository) SaveCandidate(_ context.Context, c optimization.Candidate) (optimization.Candidate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[c.RunID]
	if !ok {
		return optimization.Candidate{}, optimization.ErrNotFound
	}
	run.Candidates = append(run.Candidates, c)
	r.runs[c.RunID] = run
	return c, nil
}

func (r *Repository) UpdateCandidate(_ context.Context, c optimization.Candidate) (optimization.Candidate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[c.RunID]
	if !ok {
		return optimization.Candidate{}, optimization.ErrNotFound
	}
	for i := range run.Candidates {
		if run.Candidates[i].ID == c.ID {
			run.Candidates[i] = c
			r.runs[c.RunID] = run
			return c, nil
		}
	}
	return optimization.Candidate{}, optimization.ErrCandidateNotFound
}

func (r *Repository) SaveApproval(_ context.Context, approval optimization.ApprovalRecord) (optimization.ApprovalRecord, error) {
	return approval, nil
}

func (r *Repository) SaveRollback(_ context.Context, rollback optimization.RollbackRecord) (optimization.RollbackRecord, error) {
	return rollback, nil
}

func cloneRun(run optimization.Run) optimization.Run {
	run.Candidates = append([]optimization.Candidate(nil), run.Candidates...)
	return run
}

var _ optimization.Repository = (*Repository)(nil)
