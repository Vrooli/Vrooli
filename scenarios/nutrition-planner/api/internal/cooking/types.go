package cooking

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Timer struct {
	ID              string    `json:"id"`
	StepID          string    `json:"step_id"`
	DurationSeconds int64     `json:"duration_seconds"`
	StartedAt       time.Time `json:"started_at"`
	PausedAt        time.Time `json:"paused_at,omitempty"`
	ElapsedSeconds  int64     `json:"elapsed_seconds"`
}

type Session struct {
	ID, WorkspaceID, RecipeID, MethodID string
	RecipeRevision                      int64
	Scale                               string
	CurrentStepIndex                    int
	CompletedSteps                      []string
	Timers                              []Timer
	Status                              string
	ActualYield, YieldUnit              string
	Version                             int64
	StartedAt, UpdatedAt, FinishedAt    time.Time
}

type Repository interface {
	Create(context.Context, Session) (Session, error)
	Get(context.Context, string, string) (Session, error)
	List(context.Context, string) ([]Session, error)
	Save(context.Context, string, string, string, Session, int64) (Session, error)
}

type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("cooking session %q not found", e.ID) }

type ErrConflict struct{ ID string }

func (e ErrConflict) Error() string {
	return fmt.Sprintf("cooking session %q changed or is already finished", e.ID)
}

var ErrInvalid = errors.New("invalid cooking session")
