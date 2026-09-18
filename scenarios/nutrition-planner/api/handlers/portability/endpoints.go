package portability

import (
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/portability/portability_v1connect"
	"nutrition-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "portability_export_recipes", Path: connect.PortabilityServiceExportRecipesProcedure, Method: "POST", Summary: "Export recipes in native format", Category: "portability"},
	{ID: "portability_export_workspace", Path: connect.PortabilityServiceExportWorkspaceProcedure, Method: "POST", Summary: "Export supported workspace data", Category: "portability"},
	{ID: "portability_preview_workspace_import", Path: connect.PortabilityServicePreviewWorkspaceImportProcedure, Method: "POST", Summary: "Validate a workspace import before applying it", Category: "portability"},
	{ID: "portability_apply_workspace_import", Path: connect.PortabilityServiceApplyWorkspaceImportProcedure, Method: "POST", Summary: "Apply a validated workspace restore atomically", Category: "portability"},
	{ID: "portability_export_groceries_csv", Path: connect.PortabilityServiceExportGroceriesCSVProcedure, Method: "POST", Summary: "Export a grocery checklist as CSV", Category: "portability"},
	{ID: "portability_export_recipe_pdf", Path: connect.PortabilityServiceExportRecipePDFProcedure, Method: "POST", Summary: "Export one recipe as PDF", Category: "portability"},
	{ID: "portability_export_weekly_pdf", Path: connect.PortabilityServiceExportWeeklyPDFProcedure, Method: "POST", Summary: "Export the weekly plan as PDF", Category: "portability"},
}
