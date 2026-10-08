package validation

import (
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"

	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/validation/validation_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "validation_runs_run", Path: validationconnect.ValidationRunServiceRunTemplateValidationProcedure, Method: "POST", Summary: "Run template validation", Description: "Runs the current template validation engine and persists the result.", Category: "validation"},
	{ID: "validation_runs_list", Path: validationconnect.ValidationRunServiceListValidationRunsProcedure, Method: "POST", Summary: "List validation runs", Description: "Returns persisted template validation runs.", Category: "validation"},
	{ID: "validation_runs_get", Path: validationconnect.ValidationRunServiceGetValidationRunProcedure, Method: "POST", Summary: "Get validation run", Description: "Returns one persisted template validation run.", Category: "validation"},
	{ID: "validation_drift_record", Path: validationconnect.ValidationRunServiceRecordFleetDriftProcedure, Method: "POST", Summary: "Record fleet drift", Description: "Runs the current fleet drift engine and persists a snapshot.", Category: "validation"},
	{ID: "validation_drift_list", Path: validationconnect.ValidationRunServiceListDriftSnapshotsProcedure, Method: "POST", Summary: "List drift snapshots", Description: "Returns persisted template drift snapshots.", Category: "validation"},
}
