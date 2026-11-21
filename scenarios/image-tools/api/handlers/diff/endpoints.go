package diff

import (
	"image-tools/internal/module"

	diffconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/diff/diff_v1connect"
)

// Endpoints describes the diff module's surface. ListDiffModes is a generated
// Connect procedure (discovery); the compare edge is a deliberate REST exception
// — two image payloads can't ride a proto field, but the result stays
// proto-typed (DiffResult).
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "diff_list_modes",
		Path:        diffconnect.DiffServiceListDiffModesProcedure,
		Method:      "POST",
		Summary:     "List image-comparison modes",
		Description: "Returns the comparison modes the diff engine supports (pixel, perceptual), each with a one-line summary. Pure discovery (executes nothing).",
		Category:    "diff",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"modes": "array<DiffModeInfo>"},
		},
		Examples: []module.Example{
			{Name: "List diff modes", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.diff.DiffService/ListDiffModes -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "diff_compare",
		Path:        "/api/v1/diff/compare",
		Method:      "POST",
		Summary:     "Compare two images (pixel + perceptual diff)",
		Description: "Compares two uploaded images and returns DiffResult: a headline verdict, pixel metrics (changed-pixel fraction + MAE/RMSE/PSNR), perceptual metrics (pHash distance + structural similarity), and a heat-map overlay blob highlighting the changed regions. Pure-Go, no model — runs on any host. The `base` and `compare` parts carry the images; the `params` part carries DiffParams (protojson).",
		Category:    "diff",
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_request", Description: "Malformed multipart form, missing base/compare image part, or invalid params"},
			{Status: 413, Code: "invalid_request", Description: "An image exceeds the ingest byte limit"},
			{Status: 422, Code: "invalid_request", Description: "An undecodable image"},
		},
		Examples: []module.Example{
			{Name: "Pixel-diff two images", Curl: "curl -X POST 'http://localhost:${API_PORT}/api/v1/diff/compare' -F base=@a.png -F compare=@b.png"},
			{Name: "Perceptual same-picture check", Curl: "curl -X POST 'http://localhost:${API_PORT}/api/v1/diff/compare' -F base=@a.png -F compare=@b.jpg -F 'params={\"mode\":\"DIFF_MODE_PERCEPTUAL\"}'"},
		},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonMultipartUpload,
			Note:   "Two image payloads cannot ride proto fields; the request is multipart. The result metadata is proto-typed (DiffResult).",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request: module.RESTPayload{
					Transport:   "multipart/form-data",
					Conformance: "transport_only",
				},
				Response: module.RESTPayload{
					ProtoFullName: "vrooli.image_tools.v1.diff.DiffResult",
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
