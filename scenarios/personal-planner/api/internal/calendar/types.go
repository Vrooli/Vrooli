package calendar

import (
	"fmt"
	"time"
)

type Allocation struct {
	ID, WorkItemID, Title, SourceLabel, LocalDate, State, CarriedFromID string
	StartMinutes, DurationMinutes                                       int
	CreatedAt                                                           time.Time
}

type CreateInput struct {
	WorkItemID, LocalDate         string
	StartMinutes, DurationMinutes int
}

type CarryForwardInput struct {
	AllocationID, TargetLocalDate string
	StartMinutes                  int
}

type Today struct {
	Allocations                                                                 []Allocation
	PlannedMinutes, AvailableMinutes, BreathingRoomMinutes, ExternalBusyMinutes int
	ExternalEventCount                                                          int
	ExternalFreshness                                                           string
}

type DateRange struct{ Allocations []Allocation }

type Routine struct {
	ID, Title, Kind, Timezone, StartDate, EndDate  string
	Weekdays                                       []int
	StartMinute, DurationMinutes, FrequencyPerWeek int
	Revision                                       int64
	Active                                         bool
}

type RoutineOccurrence struct {
	RoutineID, Title, LocalDate, Kind string
	StartMinute, DurationMinutes      int
	Generated                         bool
}

type ErrRoutineNotFound struct{ ID string }

func (e ErrRoutineNotFound) Error() string { return fmt.Sprintf("routine %q not found", e.ID) }

type ErrRoutineRevisionConflict struct{ ID string }

func (e ErrRoutineRevisionConflict) Error() string {
	return fmt.Sprintf("routine %q changed; reload before changing an occurrence", e.ID)
}

type CreateRoutineInput struct {
	Title, Kind, Timezone, StartDate, EndDate      string
	Weekdays                                       []int
	StartMinute, DurationMinutes, FrequencyPerWeek int
}

type ErrInvalidAllocation struct{ Field, Reason string }

func (e ErrInvalidAllocation) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

type ErrWorkItemNotFound struct{ ID string }

func (e ErrWorkItemNotFound) Error() string { return fmt.Sprintf("work item %q not found", e.ID) }

type ErrAllocationConflict struct{}

func (e ErrAllocationConflict) Error() string {
	return "allocation overlaps an existing accepted allocation"
}

type ErrAllocationNotFound struct{ ID string }

func (e ErrAllocationNotFound) Error() string { return fmt.Sprintf("allocation %q not found", e.ID) }

type ErrAllocationAlreadyCarried struct{ ID string }

func (e ErrAllocationAlreadyCarried) Error() string {
	return fmt.Sprintf("allocation %q was already carried forward", e.ID)
}
