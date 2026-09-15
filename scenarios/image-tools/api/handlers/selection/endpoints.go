package selection

import (
	"image-tools/internal/module"

	selectionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/selection/selection_v1connect"
)

// Endpoints describes the selection module's surface. ListRegionClasses +
// SuggestEdits are generated Connect procedures (discovery + the pure
// contextual-edit compiler); the segment edge is a deliberate REST exception —
// image bytes can't ride a proto field, but the result stays proto-typed
// (SegmentResult).
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "selection_list_region_classes",
		Path:        selectionconnect.SelectionServiceListRegionClassesProcedure,
		Method:      "POST",
		Summary:     "List region classes + their contextual edit menus",
		Description: "Returns the region classes the classifier can label (person, sky, foliage, background, object), each with its contextual edit menu.",
		Category:    "selection",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"classes": "array<RegionClassInfo>"},
		},
		Examples: []module.Example{
			{Name: "List region classes", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.selection.SelectionService/ListRegionClasses -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "selection_suggest_edits",
		Path:        selectionconnect.SelectionServiceSuggestEditsProcedure,
		Method:      "POST",
		Summary:     "Suggest contextual edits for a region class",
		Description: "Returns the contextual edit menu for a region class — each edit compiled to a ready-to-submit AI request shape (operation + prompt + requires_mask). Unknown/empty class returns the generic object menu. The compose-seam; executes nothing.",
		Category:    "selection",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"region_class": "string"}},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"region_class": "string", "edits": "array<SuggestedEdit>"},
		},
		Examples: []module.Example{
			{Name: "Suggest edits for a sky", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.selection.SelectionService/SuggestEdits -H 'Content-Type: application/json' -d '{\"region_class\":\"sky\"}'"},
		},
	},
	{
		ID:          "selection_segment",
		Path:        "/api/v1/selection/segment",
		Method:      "POST",
		Summary:     "Smart-select a region from an uploaded image",
		Description: "Turns a click point / box / auto-subject prompt into a mask that snaps to the region's silhouette (built-in region-grow, runs on any host), classifies the region, stores the mask as a blob, and returns SegmentResult (mask_ref + class + contextual edit menu). The `file` part carries the image; the `params` part carries SegmentParams (protojson).",
		Category:    "selection",
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_request", Description: "Malformed multipart form, missing image part, or invalid params"},
			{Status: 413, Code: "invalid_request", Description: "Image exceeds the ingest byte limit"},
			{Status: 422, Code: "invalid_request", Description: "Undecodable image or an invalid mode/seed (e.g. point mode without a point)"},
		},
		Examples: []module.Example{
			{Name: "Point-select the center of an image", Curl: "curl -X POST 'http://localhost:${API_PORT}/api/v1/selection/segment' -F file=@in.png -F 'params={\"mode\":\"SEGMENT_MODE_POINT\",\"points\":[{\"x\":0.5,\"y\":0.5}]}'"},
			{Name: "Auto-extract the subject", Curl: "curl -X POST 'http://localhost:${API_PORT}/api/v1/selection/segment' -F file=@in.png -F 'params={\"mode\":\"SEGMENT_MODE_AUTO\"}'"},
		},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonMultipartUpload,
			Note:   "Image bytes cannot ride a proto field; the request is multipart. The result metadata is proto-typed (SegmentResult).",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request: module.RESTPayload{
					Transport:   "multipart/form-data",
					Conformance: "transport_only",
				},
				Response: module.RESTPayload{
					ProtoFullName: "vrooli.image_tools.v1.selection.SegmentResult",
					Transport:     "json",
					Conformance:   "protojson",
				},
				Error: module.RESTPayload{
					ProtoFullName: "vrooli.image_tools.v1.shared.ErrorEnvelope",
					Transport:     "json",
					Conformance:   "protojson",
				},
			},
		},
	},
}
