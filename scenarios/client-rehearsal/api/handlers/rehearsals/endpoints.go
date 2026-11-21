package rehearsals

import (
	"client-rehearsal/internal/module"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/rehearsals/rehearsals_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "rehearsals_create_and_run", Path: connect.RehearsalServiceCreateAndRunRehearsalProcedure, Method: "POST", Summary: "Import and run a synthetic workflow", Description: "Scrubs secret-shaped values, runs declared operations through the synthetic adapter, and returns evidence awaiting explicit review.", Category: "rehearsals"},
	{ID: "rehearsals_list", Path: connect.RehearsalServiceListRehearsalsProcedure, Method: "POST", Summary: "List rehearsals", Description: "Lists recent rehearsal records and their operator review states.", Category: "rehearsals"},
	{ID: "rehearsals_get", Path: connect.RehearsalServiceGetRehearsalProcedure, Method: "POST", Summary: "Get a rehearsal", Description: "Returns the sanitized workflow provenance, execution trace, repair evidence, and handoff.", Category: "rehearsals"},
	{ID: "rehearsals_record_review", Path: connect.RehearsalServiceRecordReviewProcedure, Method: "POST", Summary: "Record operator review", Description: "Explicitly approves the handoff or requests changes; execution success alone does not approve it.", Category: "rehearsals"},
}
