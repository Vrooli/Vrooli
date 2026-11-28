package work

import (
	"fmt"
	"time"
)

type WorkItem struct {
	ID, Title, Description string
	RemainingMinutes       int
	SourceLabel            string
	CreatedAt, UpdatedAt   time.Time
}

type CreateInput struct {
	Title, Description, SourceLabel string
	RemainingMinutes                int
}

type TodayPlan struct {
	Entries          []TodayPlanEntry
	PlannedMinutes   int
	AvailableMinutes int
	BreathingRoom    int
}

type TodayPlanEntry struct {
	WorkItemID, Title, SourceLabel string
	StartMinutes, DurationMinutes  int
}

type ErrWorkItemNotFound struct{ ID string }

func (e ErrWorkItemNotFound) Error() string { return fmt.Sprintf("work item %q not found", e.ID) }

type ErrInvalidWorkItem struct{ Field, Reason string }

func (e ErrInvalidWorkItem) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }
