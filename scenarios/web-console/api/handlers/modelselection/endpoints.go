package modelselection

import (
	modelselectionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/modelselection/modelselection_v1connect"
	"web-console/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "modelselection_get_role_model_choices", Path: modelselectionconnect.ModelSelectionServiceGetRoleModelChoicesProcedure, Method: "POST", Summary: "Get coding role model choices", Description: "Returns resource-declared model/source choices and policy provenance without claiming provider availability.", Category: "modelselection"},
}
