package focus

import (
	focusconnect "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/focus/focus_v1connect"
	"personal-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "focus_current", Path: focusconnect.FocusServiceGetCurrentSessionProcedure, Method: "POST", Summary: "Get current focus session", Description: "Returns the one running or paused focus session, if present.", Category: "focus"},
	{ID: "focus_start", Path: focusconnect.FocusServiceStartFocusProcedure, Method: "POST", Summary: "Start focus", Description: "Starts one durable focus session.", Category: "focus"},
	{ID: "focus_pause", Path: focusconnect.FocusServicePauseFocusProcedure, Method: "POST", Summary: "Pause focus", Description: "Pauses active time without ending the session.", Category: "focus"},
	{ID: "focus_resume", Path: focusconnect.FocusServiceResumeFocusProcedure, Method: "POST", Summary: "Resume focus", Description: "Resumes a paused focus session.", Category: "focus"},
	{ID: "focus_end", Path: focusconnect.FocusServiceEndFocusProcedure, Method: "POST", Summary: "End focus", Description: "Ends a focus session while preserving active and wall time separately.", Category: "focus"},
	{ID: "focus_record_actual", Path: focusconnect.FocusServiceRecordManualActualProcedure, Method: "POST", Summary: "Record manual actual", Description: "Records approximate or observed active time without fabricating a timer interval.", Category: "focus"},
	{ID: "focus_list_actuals", Path: focusconnect.FocusServiceListActualsProcedure, Method: "POST", Summary: "List actuals", Description: "Lists manual actual records, optionally for a local date.", Category: "focus"},
	{ID: "focus_list_actual_corrections", Path: focusconnect.FocusServiceListActualCorrectionsProcedure, Method: "POST", Summary: "List actual corrections", Description: "Lists bounded correction provenance for manual actual records.", Category: "focus"},
	{ID: "focus_correct_actual", Path: focusconnect.FocusServiceCorrectActualProcedure, Method: "POST", Summary: "Correct actual", Description: "Corrects a manual actual while retaining correction provenance.", Category: "focus"},
}
