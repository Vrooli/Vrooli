package focus

import (
	"fmt"
	"time"
)

const (
	StateRunning = "running"
	StatePaused  = "paused"
	StateEnded   = "ended"
	ModeOpen     = "open"
	ModePomodoro = "pomodoro"
	ModeTimed    = "timed"
	ModeUntimed  = "untimed"
)

type Session struct {
	ID, WorkItemID, Title, Mode, State string
	StartedAt, EndedAt                 time.Time
	ActiveStartedAt                    *time.Time
	ActiveSeconds, WallSeconds         int64
	Revision                           int64
}

type Actual struct {
	ID, WorkItemID, Title, LocalDate, Certainty, Note string
	ReportedMinutes, Revision                         int64
	CreatedAt                                         time.Time
}

type Correction struct {
	ID, ActualID, PreviousCertainty, NewCertainty, Reason string
	PreviousMinutes, NewMinutes                           int64
	CreatedAt                                             time.Time
}

type RecordActualInput struct {
	WorkItemID, Title, LocalDate, Certainty, Note string
	ReportedMinutes                               int64
}

type CorrectActualInput struct {
	ID, Certainty, Note string
	ReportedMinutes     int64
	ExpectedRevision    int64
}

type StartInput struct {
	WorkItemID, Title, Mode string
}

type ErrInvalidFocus struct{ Field, Reason string }

func (e ErrInvalidFocus) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

type ErrSessionNotFound struct{ ID string }

func (e ErrSessionNotFound) Error() string { return fmt.Sprintf("focus session %q not found", e.ID) }

type ErrSessionConflict struct{ Reason string }

func (e ErrSessionConflict) Error() string {
	return fmt.Sprintf("focus session conflict: %s", e.Reason)
}

type ErrInvalidTransition struct{ State, Action string }

func (e ErrInvalidTransition) Error() string {
	return fmt.Sprintf("cannot %s a %s session", e.Action, e.State)
}

type ErrRevisionConflict struct{ ID string }

func (e ErrRevisionConflict) Error() string {
	return fmt.Sprintf("focus session %q changed; refresh before retrying", e.ID)
}

type ErrActualNotFound struct{ ID string }

func (e ErrActualNotFound) Error() string { return fmt.Sprintf("actual %q not found", e.ID) }
