package calendar

import "context"

type Repository interface {
	ListToday(context.Context, string) (Today, error)
	ListRange(context.Context, string, string) (DateRange, error)
	Create(context.Context, Allocation) (Allocation, error)
	Preview(context.Context, PreviewInput) (PlacementProposal, error)
	ApplyProposal(context.Context, ApplyProposalInput) (Allocation, error)
	PreviewSchedule(context.Context, SchedulePreviewInput) (ScheduleProposal, error)
	ApplyScheduleProposal(context.Context, ApplyScheduleProposalInput) ([]Allocation, error)
	CarryForward(context.Context, CarryForwardInput) (Allocation, error)
	WorkItem(context.Context, string) (Allocation, error)
	ListRoutines(context.Context) ([]Routine, error)
	CreateRoutine(context.Context, CreateRoutineInput) (Routine, error)
	ListRoutineOccurrences(context.Context, string, string) ([]RoutineOccurrence, error)
	SkipRoutineOccurrence(context.Context, string, string, int64) error
	RescheduleRoutineOccurrence(context.Context, string, string, int, int64) error
}
