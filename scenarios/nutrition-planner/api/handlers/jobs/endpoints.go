package jobs

import (
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/jobs/jobs_v1connect"
	"nutrition-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "jobs_list", Path: connect.JobsServiceListJobsProcedure, Method: "POST", Summary: "List optional background jobs", Category: "jobs"},
	{ID: "jobs_create", Path: connect.JobsServiceCreateJobProcedure, Method: "POST", Summary: "Queue an optional background job", Category: "jobs"},
	{ID: "jobs_transition", Path: connect.JobsServiceTransitionJobProcedure, Method: "POST", Summary: "Advance a background job state", Category: "jobs"},
	{ID: "jobs_cancel", Path: connect.JobsServiceCancelJobProcedure, Method: "POST", Summary: "Cancel a background job", Category: "jobs"},
}
