package handoffs

import (
	"persona/internal/module"

	handoffsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/handoffs/handoffs_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "handoffs_open", Path: handoffsconnect.HandoffsServiceOpenHandoffProcedure, Method: "POST", Summary: "Open a resumable human handoff", Category: "handoffs"},
	{ID: "handoffs_get", Path: handoffsconnect.HandoffsServiceGetHandoffProcedure, Method: "POST", Summary: "Read a handoff checkpoint", Category: "handoffs"},
	{ID: "handoffs_list", Path: handoffsconnect.HandoffsServiceListHandoffsProcedure, Method: "POST", Summary: "List handoffs", Category: "handoffs"},
	{ID: "handoffs_complete", Path: handoffsconnect.HandoffsServiceCompleteHandoffProcedure, Method: "POST", Summary: "Complete a human handoff", Category: "handoffs"},
	{ID: "handoffs_cancel", Path: handoffsconnect.HandoffsServiceCancelHandoffProcedure, Method: "POST", Summary: "Cancel a handoff", Category: "handoffs"},
	{ID: "handoffs_resume", Path: handoffsconnect.HandoffsServiceResumeHandoffProcedure, Method: "POST", Summary: "Resume from a completed checkpoint", Category: "handoffs"},
	{ID: "handoffs_prepare_enrolment", Path: handoffsconnect.HandoffsServicePrepareEnrolmentProcedure, Method: "POST", Summary: "Prepare one enrolment handoff", Category: "handoffs"},
}
