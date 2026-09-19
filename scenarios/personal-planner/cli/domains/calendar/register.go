package calendar

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "calendar"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"CalendarService.CarryForwardAllocation":     cliapp.ProtoMutation(h.carryForwardCall, h.carryForwardReport),
		"CalendarService.ListRoutines":                cliapp.ProtoList(h.listRoutinesCall, h.listRoutinesReport),
		"CalendarService.CreateRoutine":               cliapp.ProtoMutation(h.createRoutineCall, h.createRoutineReport),
		"CalendarService.ListRoutineOccurrences":      cliapp.ProtoList(h.occurrencesCall, h.occurrencesReport),
		"CalendarService.SkipRoutineOccurrence":       cliapp.ProtoMutation(h.skipOccurrenceCall, h.skipOccurrenceReport),
		"CalendarService.RescheduleRoutineOccurrence": cliapp.ProtoMutation(h.rescheduleOccurrenceCall, h.rescheduleOccurrenceReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("calendar: load from manifest: %w", err)
	}
	return group, nil
}
