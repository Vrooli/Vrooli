package investigations

import (
	"context"
	"time"
)

type Run struct {
	ID              string
	EntryID         string
	ExecutionMode   string
	Status          string
	SkipReason      string
	ExitCode        int
	TimedOut        bool
	StartedAt       time.Time
	CompletedAt     time.Time
	DurationSeconds float64
	HostOS          string
	HostArch        string
	ResultJSON      string
	StderrTail      string
	AnomalyID       string
	Findings        []Finding
}

type Finding struct {
	Severity   string
	Code       string
	Summary    string
	DetailJSON string
}

type Repository interface {
	SaveRun(context.Context, Run) error
	GetRun(context.Context, string) (Run, error)
	ListRuns(context.Context, string, time.Time, int) ([]Run, error)
	CountRunsBefore(context.Context, time.Time) (int64, error)
	PruneRunsBefore(context.Context, time.Time) (int64, error)
}
