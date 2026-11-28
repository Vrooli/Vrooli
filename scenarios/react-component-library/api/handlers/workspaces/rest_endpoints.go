package workspaces

import (
	"net/http"

	"react-component-library/internal/module"
)

func workspaceBinaryREST(note string) *module.RESTException {
	return &module.RESTException{
		Reason: module.RESTReasonBrowserSurface,
		Note:   note,
		ProtoPayloads: &module.RESTProtoPayloads{
			Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
			Response: module.RESTPayload{Transport: "binary", Conformance: "external_shape"},
			Error:    module.RESTPayload{Transport: "text", Conformance: "external_shape"},
		},
	}
}

var captureEndpoints = []module.EndpointDescriptor{{
	ID: "workspaces_capture_receipt_screenshot", Path: "/api/workspaces/{id}/states/{stateId}/capture-receipts/{receiptId}/screenshot", Method: http.MethodGet,
	Summary:     "Read the screenshot from a workspace capture receipt",
	Description: "Serves the receipt screenshot as image bytes with private cache headers; artifact access is scoped to the owning workspace and state.",
	Category:    "workspaces", Response: &module.Schema{Type: "binary", Properties: map[string]string{"body": "screenshot bytes"}},
	Errors:        []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No screenshot for that workspace capture receipt"}},
	RESTException: workspaceBinaryREST("Screenshots are browser-rendered image artifacts; this endpoint streams bytes directly."),
}}

var comparisonEndpoints = []module.EndpointDescriptor{{
	ID: "workspaces_visual_comparison_heatmap", Path: "/api/workspaces/{id}/states/{stateId}/visual-comparisons/{comparisonId}/heatmap", Method: http.MethodGet,
	Summary:     "Read a visual comparison heatmap",
	Description: "Serves the heatmap PNG associated with the selected comparison; artifact lookup remains scoped to the workspace and state.",
	Category:    "workspaces", Response: &module.Schema{Type: "binary", Properties: map[string]string{"body": "PNG heatmap bytes"}},
	Errors:        []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No heatmap for that comparison"}, {Status: 502, Code: "bad_gateway", Description: "Image Tools could not return the heatmap"}},
	RESTException: workspaceBinaryREST("Heatmaps are browser-rendered PNG artifacts; this endpoint streams bytes directly."),
}}
