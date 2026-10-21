package motions

import (
	"device-studio/internal/module"
	motionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/motions/motions_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "motions_candidate_save", Path: motionsconnect.MotionsServiceSaveMotionCandidateProcedure, Method: "POST", Summary: "Save motion candidate", Description: "Appends an immutable motion candidate revision at the expected head; local validity does not resolve external targets or qualify motion.", Category: "motions"},
	{ID: "motions_revision_get", Path: motionsconnect.MotionsServiceGetMotionRevisionProcedure, Method: "POST", Summary: "Get motion revision", Description: "Returns one immutable motion candidate revision after canonical digest verification.", Category: "motions"},
	{ID: "motions_head_get", Path: motionsconnect.MotionsServiceGetMotionHeadProcedure, Method: "POST", Summary: "Get motion head", Description: "Returns the current immutable motion candidate revision.", Category: "motions"},
	{ID: "motions_revision_list", Path: motionsconnect.MotionsServiceListMotionRevisionsProcedure, Method: "POST", Summary: "List motion revisions", Description: "Returns a bounded ascending page of immutable motion revisions.", Category: "motions"},
	{ID: "motions_candidate_list", Path: motionsconnect.MotionsServiceListMotionCandidatesProcedure, Method: "POST", Summary: "List motion candidate heads", Description: "Returns one latest immutable candidate revision per motion in stable identifier order.", Category: "motions"},
}
