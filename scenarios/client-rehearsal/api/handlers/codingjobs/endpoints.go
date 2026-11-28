package codingjobs

import (
	"client-rehearsal/internal/module"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/codingjobs/codingjobs_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "coding_jobs_create", Path: connect.CodingJobServiceCreateCodingJobProcedure, Method: "POST", Summary: "Create a coding practice job", Description: "Persists the solver-visible task and source seal reference without exposing the frozen evaluator or reference fix.", Category: "coding-jobs"},
	{ID: "coding_jobs_record_stage", Path: connect.CodingJobServiceRecordCodingJobStageProcedure, Method: "POST", Summary: "Record a coding job stage", Description: "Appends a role-checked event using optimistic revision control.", Category: "coding-jobs"},
	{ID: "coding_jobs_record_evaluation", Path: connect.CodingJobServiceRecordCodingJobEvaluationProcedure, Method: "POST", Summary: "Record a frozen evaluation receipt", Description: "Stores one immutable Test Genie receipt reference, digest, and closed three-way score.", Category: "coding-jobs"},
	{ID: "coding_jobs_get", Path: connect.CodingJobServiceGetCodingJobProcedure, Method: "POST", Summary: "Get a coding practice job", Description: "Reads a durable coding job and bounded event history.", Category: "coding-jobs"},
	{ID: "coding_jobs_list", Path: connect.CodingJobServiceListCodingJobsProcedure, Method: "POST", Summary: "List coding practice jobs", Description: "Lists recent coding practice jobs with an explicit bounded limit.", Category: "coding-jobs"},
}
