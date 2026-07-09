package lifecycle

import (
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"

	lifecycleconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/lifecycle/lifecycle_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "lifecycle_generate_scenario", Path: lifecycleconnect.TemplateLifecycleServiceGenerateScenarioProcedure, Method: "POST", Summary: "Generate scenario", Description: "Generates a scenario from a governed scenario template.", Category: "lifecycle"},
	{ID: "lifecycle_orient_scenario", Path: lifecycleconnect.TemplateLifecycleServiceOrientScenarioProcedure, Method: "POST", Summary: "Orient scenario", Description: "Reports or finalizes scenario orientation progress.", Category: "lifecycle"},
	{ID: "lifecycle_detemplate_scenario", Path: lifecycleconnect.TemplateLifecycleServiceDetemplateScenarioProcedure, Method: "POST", Summary: "Detemplate scenario", Description: "Removes the template example domain from a generated scenario.", Category: "lifecycle"},
	{ID: "lifecycle_validate_template", Path: lifecycleconnect.TemplateLifecycleServiceValidateTemplateProcedure, Method: "POST", Summary: "Validate template", Description: "Runs the scenario-template validation engine.", Category: "lifecycle"},
	{ID: "lifecycle_drift_report", Path: lifecycleconnect.TemplateLifecycleServiceDriftReportProcedure, Method: "POST", Summary: "Template drift report", Description: "Compares generated scenarios against current template hashes.", Category: "lifecycle"},
	{ID: "lifecycle_cleanup_runs", Path: lifecycleconnect.TemplateLifecycleServiceCleanupRunsProcedure, Method: "POST", Summary: "Cleanup template runs", Description: "Cleans retained or stale deep-validation workspaces.", Category: "lifecycle"},
	{ID: "design_list", Path: lifecycleconnect.DesignKitServiceListDesignKitsProcedure, Method: "POST", Summary: "List design kits", Description: "Lists scenario design kits.", Category: "design"},
	{ID: "design_get", Path: lifecycleconnect.DesignKitServiceGetDesignKitProcedure, Method: "POST", Summary: "Get design kit", Description: "Returns one scenario design kit.", Category: "design"},
	{ID: "design_validate", Path: lifecycleconnect.DesignKitServiceValidateDesignKitsProcedure, Method: "POST", Summary: "Validate design kits", Description: "Validates scenario design kits.", Category: "design"},
}
