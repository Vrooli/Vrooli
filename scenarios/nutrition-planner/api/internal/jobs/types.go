// Package jobs owns the durable state-machine vocabulary for optional work.
// Applying a job result remains an explicit domain operation elsewhere.
package jobs

import (
	"context"
	"errors"
	"fmt"
)

type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("job %q not found", e.ID) }

type ErrForbidden struct{ ID string }

func (e ErrForbidden) Error() string { return fmt.Sprintf("job %q is outside the workspace", e.ID) }

type ErrIdempotencyConflict struct{ Key string }

func (e ErrIdempotencyConflict) Error() string {
	return fmt.Sprintf("job dedup key %q was reused with different input", e.Key)
}

type State string

const (
	Queued           State = "queued"
	Running          State = "running"
	WaitingForReview State = "waiting_for_review"
	Succeeded        State = "succeeded"
	Failed           State = "failed"
	Canceled         State = "canceled"
)

type Job struct {
	ID, WorkspaceID, Type, DedupKey, Provider, ResultReference, ErrorCode string
	RequestHash                                                           string
	State                                                                 State
	InputRevisions                                                        []string
	Attempts                                                              int
	BudgetUnits                                                           int
}

type TransitionInput struct {
	WorkspaceID, ID, Provider, ResultReference, ErrorCode string
	NextState                                             State
	Attempts                                              int
}

type Repository interface {
	Create(context.Context, Job) (Job, error)
	List(context.Context, string) ([]Job, error)
	Get(context.Context, string, string) (Job, error)
	Transition(context.Context, TransitionInput) (Job, error)
}

func New(id, workspaceID, kind, dedup string) (Job, error) {
	if id == "" || workspaceID == "" || kind == "" || dedup == "" {
		return Job{}, errors.New("job requires id, workspace, type, and dedup key")
	}
	return Job{ID: id, WorkspaceID: workspaceID, Type: kind, DedupKey: dedup, State: Queued}, nil
}

func (j Job) Transition(next State) (Job, error) {
	allowed := map[State]map[State]bool{
		Queued:           {Running: true, Canceled: true},
		Running:          {WaitingForReview: true, Succeeded: true, Failed: true, Canceled: true},
		WaitingForReview: {Running: true, Canceled: true},
	}
	if !allowed[j.State][next] {
		return j, fmt.Errorf("job %q cannot transition from %s to %s", j.ID, j.State, next)
	}
	j.State = next
	return j, nil
}
