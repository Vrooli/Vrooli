package analysis

import (
	"image-tools/internal/module"

	analysisconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/analysis/analysis_v1connect"
)

// Endpoints describes the analysis module's surface. ListAnalysisOperations is a
// generated Connect procedure (discovery); the analyze edge is a deliberate REST
// exception — image bytes can't ride a proto field, but the result stays
// proto-typed (AnalyzeResponse, see the proto_payloads declaration).
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "analysis_list_operations",
		Path:        analysisconnect.AnalysisServiceListAnalysisOperationsProcedure,
		Method:      "POST",
		Summary:     "List image→data analysis operations",
		Description: "Returns the analysis operation catalog (ocr, nsfw_classify, probe) with each op's model-backing and default model.",
		Category:    "analysis",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"operations": "array<AnalysisOperationInfo>"},
		},
		Examples: []module.Example{
			{Name: "List analysis operations", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.analysis.AnalysisService/ListAnalysisOperations -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "analysis_run",
		Path:        "/api/v1/analysis/{operation}",
		Method:      "POST",
		Summary:     "Run an analysis operation on an uploaded image",
		Description: "Executes one analysis op (ocr / nsfw_classify / probe) on a multipart-uploaded image. The `file` part carries the image bytes; the structured result is returned as AnalyzeResponse (protojson). `probe` is pure-Go and always available; `ocr`/`nsfw_classify` require their backend + model installed.",
		Category:    "analysis",
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_request", Description: "Malformed multipart form or missing image part"},
			{Status: 404, Code: "not_found", Description: "Unknown analysis operation"},
			{Status: 413, Code: "invalid_request", Description: "Image exceeds the ingest byte limit"},
			{Status: 422, Code: "invalid_request", Description: "Undecodable image"},
			{Status: 503, Code: "internal", Description: "Model-backed op's backend/model is not installed"},
		},
		Examples: []module.Example{
			{Name: "Probe an image", Curl: "curl -X POST 'http://localhost:${API_PORT}/api/v1/analysis/probe' -F file=@in.png"},
			{Name: "OCR an image", Curl: "curl -X POST 'http://localhost:${API_PORT}/api/v1/analysis/ocr' -F file=@scan.png"},
		},

		RESTException: &module.RESTException{
			Reason: module.RESTReasonMultipartUpload,
			Note:   "Image bytes cannot ride a proto field; the request is multipart. The result metadata is proto-typed (AnalyzeResponse).",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request: module.RESTPayload{
					Transport:   "multipart/form-data",
					Conformance: "transport_only",
				},
				Response: module.RESTPayload{
					ProtoFullName: "vrooli.image_tools.v1.analysis.AnalyzeResponse",
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
