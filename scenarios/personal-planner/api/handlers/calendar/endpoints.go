package calendar

import (
	c "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/calendar/calendar_v1connect"
	"personal-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "calendar_today", Path: c.CalendarServiceListTodayAllocationsProcedure, Method: "POST", Summary: "List accepted allocations", Description: "Lists accepted allocations for the current local day and reports measured planned capacity.", Category: "calendar"},
	{ID: "calendar_range", Path: c.CalendarServiceListAllocationsProcedure, Method: "POST", Summary: "List accepted allocation range", Description: "Lists accepted allocations for an explicit inclusive local-date range.", Category: "calendar"},
	{ID: "calendar_create", Path: c.CalendarServiceCreateAllocationProcedure, Method: "POST", Summary: "Place work on the calendar", Description: "Creates an accepted, non-overlapping calendar allocation for a work item.", Category: "calendar"},
	{ID: "calendar_preview", Path: c.CalendarServicePreviewAllocationProcedure, Method: "POST", Summary: "Preview placement", Description: "Calculates the next open placement without changing accepted schedule state.", Category: "calendar"},
	{ID: "calendar_apply_proposal", Path: c.CalendarServiceApplyAllocationProposalProcedure, Method: "POST", Summary: "Apply placement proposal", Description: "Atomically applies a still-current placement proposal with an idempotency key.", Category: "calendar"},
	{ID: "calendar_preview_schedule", Path: c.CalendarServicePreviewScheduleProcedure, Method: "POST", Summary: "Preview a multi-item schedule", Description: "Builds a deterministic one-day proposal for selected work without changing accepted schedule state.", Category: "calendar"},
	{ID: "calendar_apply_schedule", Path: c.CalendarServiceApplyScheduleProposalProcedure, Method: "POST", Summary: "Apply a multi-item schedule proposal", Description: "Atomically applies all feasible changes in a current multi-item schedule proposal.", Category: "calendar"},
	{ID: "calendar_carry_forward", Path: c.CalendarServiceCarryForwardAllocationProcedure, Method: "POST", Summary: "Carry unfinished work forward", Description: "Preserves the original accepted placement as history and creates one new accepted placement on a selected date.", Category: "calendar"},
	{ID: "calendar_list_routines", Path: c.CalendarServiceListRoutinesProcedure, Method: "POST", Summary: "List routines", Description: "Lists active fixed and flexible routine definitions.", Category: "calendar"},
	{ID: "calendar_create_routine", Path: c.CalendarServiceCreateRoutineProcedure, Method: "POST", Summary: "Create routine", Description: "Creates a timezone-aware fixed or flexible routine definition.", Category: "calendar"},
	{ID: "calendar_routine_occurrences", Path: c.CalendarServiceListRoutineOccurrencesProcedure, Method: "POST", Summary: "List routine occurrences", Description: "Expands routine demand across a bounded local-date range.", Category: "calendar"},
	{ID: "calendar_skip_routine_occurrence", Path: c.CalendarServiceSkipRoutineOccurrenceProcedure, Method: "POST", Summary: "Skip routine occurrence", Description: "Skips one generated routine occurrence without changing the series definition.", Category: "calendar"},
	{ID: "calendar_reschedule_routine_occurrence", Path: c.CalendarServiceRescheduleRoutineOccurrenceProcedure, Method: "POST", Summary: "Reschedule routine occurrence", Description: "Moves one generated routine occurrence without changing the series definition.", Category: "calendar"},
}
