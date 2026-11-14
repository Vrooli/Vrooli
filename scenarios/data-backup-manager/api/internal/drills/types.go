package drills

import (
	"fmt"
	"time"
)

type Status string

const (
	StatusRequested Status = "requested"
	StatusRunning   Status = "running"
	StatusVerified  Status = "verified"
	StatusFailed    Status = "failed"
)

type Drill struct {
	ID             string
	PlanID         string
	TargetID       string
	DestinationID  string
	SnapshotID     string
	RestoreID      string
	Status         Status
	Scheduled      bool
	Error          string
	NextAction     string
	RequestedAt    time.Time
	StartedAt      time.Time
	FinishedAt     time.Time
	IdempotencyKey string
}

type Plan struct {
	ID             string
	TargetIDs      []string
	DestinationIDs []string
	Enabled        bool
	DrillSchedule  string
}

type Snapshot struct {
	ID          string
	CompletedAt time.Time
}

type Restore struct {
	ID     string
	Status string
	Error  string
}

type Preview struct {
	Eligible      bool
	PlanID        string
	TargetID      string
	DestinationID string
	SnapshotID    string
	Warnings      []string
	Reason        string
}

type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("drill %q not found", e.ID) }

type ErrInvalid struct {
	Field  string
	Reason string
}

func (e ErrInvalid) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

type ErrAlreadyActive struct{ ID string }

func (e ErrAlreadyActive) Error() string { return fmt.Sprintf("drill already active: %s", e.ID) }
