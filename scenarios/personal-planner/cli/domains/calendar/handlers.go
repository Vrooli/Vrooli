package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/calendar"
	vc "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/calendar/calendar_v1connect"
)

type handlers struct{ client vc.CalendarServiceClient }

func newHandlers(c *cliapp.ScenarioApp) *handlers {
	httpClient, base := cliapp.NewConnectHTTPClient(c)
	return &handlers{client: vc.NewCalendarServiceClient(httpClient, base)}
}

func (h *handlers) listRoutinesCall(_ cliapp.OperationContext) (*v.ListRoutinesResponse, error) {
	response, err := h.client.ListRoutines(context.Background(), connect.NewRequest(&v.ListRoutinesRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list routines", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) listRoutinesReport(_ cliapp.OperationContext, response *v.ListRoutinesResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Active routines: %d", len(response.Routines))}, ResultsHeading: "Routines", Results: routineLines(response.Routines)}
}

func (h *handlers) createRoutineCall(c cliapp.OperationContext) (*v.CreateRoutineResponse, error) {
	weekdays, err := parseJSONInts(c.Flag("weekdays-json"))
	if err != nil {
		return nil, fmt.Errorf("weekdays-json must be a JSON array: %w", err)
	}
	start, err := parseIntFlag(c, "start-minute")
	if err != nil {
		return nil, err
	}
	duration, err := parseIntFlag(c, "duration-minutes")
	if err != nil {
		return nil, err
	}
	frequency := 1
	if strings.TrimSpace(c.Flag("frequency-per-week")) != "" {
		frequency, err = parseIntFlag(c, "frequency-per-week")
		if err != nil {
			return nil, err
		}
	}
	response, err := h.client.CreateRoutine(context.Background(), connect.NewRequest(&v.CreateRoutineRequest{Title: c.Flag("title"), Kind: c.Flag("kind"), Timezone: c.Flag("timezone"), StartDate: c.Flag("start-date"), EndDate: c.Flag("end-date"), Weekdays: weekdays, StartMinute: int32(start), DurationMinutes: int32(duration), FrequencyPerWeek: int32(frequency)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create routine", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) createRoutineReport(_ cliapp.OperationContext, response *v.CreateRoutineResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created %s routine %q.", response.Routine.Kind, response.Routine.Title)}, NextCommand: []string{"`calendar routines` — inspect active routine definitions"}}
}

func (h *handlers) carryForwardCall(c cliapp.OperationContext) (*v.CarryForwardAllocationResponse, error) {
	start, err := parseIntFlag(c, "start-minute")
	if err != nil {
		return nil, err
	}
	response, err := h.client.CarryForwardAllocation(context.Background(), connect.NewRequest(&v.CarryForwardAllocationRequest{AllocationId: c.Flag("allocation-id"), TargetLocalDate: c.Flag("target-date"), StartMinutes: int32(start)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("carry forward allocation", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) carryForwardReport(_ cliapp.OperationContext, response *v.CarryForwardAllocationResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Carried unfinished work to %s at %02d:%02d.", response.Allocation.LocalDate, response.Allocation.StartMinutes/60, response.Allocation.StartMinutes%60)}, NextCommand: []string{"`calendar routines` — inspect related planning state"}}
}

func (h *handlers) occurrencesCall(c cliapp.OperationContext) (*v.ListRoutineOccurrencesResponse, error) {
	response, err := h.client.ListRoutineOccurrences(context.Background(), connect.NewRequest(&v.ListRoutineOccurrencesRequest{StartLocalDate: c.Flag("start-date"), EndLocalDate: c.Flag("end-date")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list routine occurrences", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) occurrencesReport(_ cliapp.OperationContext, response *v.ListRoutineOccurrencesResponse) cliapp.ListReport {
	results := make([]string, len(response.Occurrences))
	for i, occurrence := range response.Occurrences {
		results[i] = fmt.Sprintf("%s %02d:%02d %dm %s (%s)", occurrence.LocalDate, occurrence.StartMinute/60, occurrence.StartMinute%60, occurrence.DurationMinutes, occurrence.Title, occurrence.Kind)
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Generated occurrences: %d", len(results))}, ResultsHeading: "Routine occurrences", Results: results}
}

func (h *handlers) skipOccurrenceCall(c cliapp.OperationContext) (*v.SkipRoutineOccurrenceResponse, error) {
	revision, err := parseIntFlag(c, "revision")
	if err != nil {
		return nil, err
	}
	response, err := h.client.SkipRoutineOccurrence(context.Background(), connect.NewRequest(&v.SkipRoutineOccurrenceRequest{RoutineId: c.Flag("routine-id"), LocalDate: c.Flag("date"), ExpectedRevision: int64(revision)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("skip routine occurrence", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) skipOccurrenceReport(_ cliapp.OperationContext, _ *v.SkipRoutineOccurrenceResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Skipped routine occurrence."}, NextCommand: []string{"`calendar routine-occurrences --start-date <date> --end-date <date>` — verify the projection"}}
}

func (h *handlers) rescheduleOccurrenceCall(c cliapp.OperationContext) (*v.RescheduleRoutineOccurrenceResponse, error) {
	startMinute, err := parseIntFlag(c, "start-minute")
	if err != nil {
		return nil, err
	}
	revision, err := parseIntFlag(c, "revision")
	if err != nil {
		return nil, err
	}
	response, err := h.client.RescheduleRoutineOccurrence(context.Background(), connect.NewRequest(&v.RescheduleRoutineOccurrenceRequest{RoutineId: c.Flag("routine-id"), LocalDate: c.Flag("date"), StartMinute: int32(startMinute), ExpectedRevision: int64(revision)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("reschedule routine occurrence", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) rescheduleOccurrenceReport(_ cliapp.OperationContext, _ *v.RescheduleRoutineOccurrenceResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Rescheduled routine occurrence."}, NextCommand: []string{"`calendar routine-occurrences --start-date <date> --end-date <date>` — verify the projection"}}
}

func routineLines(items []*v.Routine) []string {
	results := make([]string, len(items))
	for i, item := range items {
		results[i] = fmt.Sprintf("%s [%s] %s %dm", item.Title, item.Kind, item.Timezone, item.DurationMinutes)
	}
	return results
}

func parseJSONInts(value string) ([]int32, error) {
	var values []int32
	if err := json.Unmarshal([]byte(value), &values); err != nil {
		return nil, err
	}
	return values, nil
}

func parseIntFlag(c cliapp.OperationContext, name string) (int, error) {
	value, err := strconv.Atoi(c.Flag(name))
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return value, nil
}
