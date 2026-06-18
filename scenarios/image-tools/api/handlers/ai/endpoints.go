package ai

import (
	"image-tools/internal/module"

	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai/ai_v1connect"
)

// Endpoints describes the ai module's surface. ListAIOperations is a generated
// Connect procedure (discovery); the submit edge is a deliberate REST exception
// — image bytes can't ride a proto field, but params and the submit result stay
// proto-typed (see the proto_payloads declaration).
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "ai_list_operations",
		Path:        aiconnect.AIServiceListAIOperationsProcedure,
		Method:      "POST",
		Summary:     "List AI generation & enhancement operations",
		Description: "Returns the model-backed operation catalog (which ops need an input image, which need a mask, which are prompt-driven) plus each op's seeded default model.",
		Category:    "ai",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"operations": "array<AIOperationInfo>"},
		},
		Examples: []module.Example{
			{Name: "List AI operations", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.ai.AIService/ListAIOperations -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "ai_submit",
		Path:        "/api/v1/ai/{operation}",
		Method:      "POST",
		Summary:     "Submit a model-backed AI operation",
		Description: "Submits one AI generation/enhancement op on optionally-uploaded image(s). The `file` part carries the input image (when required), `mask` carries a mask (inpaint/object_removal), and `params` carries AIParams as protojson. The op runs asynchronously on the durable job queue: the response (SubmitAIResponse) carries the job id + ETA + the resolved model/tier + any fallback warnings. Wait via the jobs verbs and fetch the output blob by the job's result ref.",
		Category:    "ai",
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_request", Description: "Malformed multipart form, missing required image/mask part, or invalid params"},
			{Status: 404, Code: "not_found", Description: "Unknown AI operation"},
			{Status: 409, Code: "invalid_request", Description: "Selected model is not installed (run `image-tools models install <id>`)"},
			{Status: 413, Code: "invalid_request", Description: "Image exceeds the ingest byte limit"},
			{Status: 422, Code: "invalid_request", Description: "No enabled model fits this host, or the model override is invalid"},
			{Status: 503, Code: "internal", Description: "No standalone backend is available and BYOK is not enabled"},
		},
		Examples: []module.Example{
			{Name: "Generate from a prompt", Curl: "curl -X POST 'http://localhost:${API_PORT}/api/v1/ai/text_to_image' -F 'params={\"prompt\":\"a red bicycle\",\"width\":512,\"height\":512}'"},
			{Name: "Upscale an image", Curl: "curl -X POST 'http://localhost:${API_PORT}/api/v1/ai/upscale' -F file=@in.png -F 'params={\"scale\":4}'"},
		},

		RESTException: &module.RESTException{
			Reason: module.RESTReasonMultipartUpload,
			Note:   "Image/mask bytes cannot ride a proto field; the request is multipart (file + mask + protojson params). Parameters stay proto-typed (AIParams) and the submit result is proto-typed (SubmitAIResponse).",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request: module.RESTPayload{
					ProtoFullName: "vrooli.image_tools.v1.ai.AIParams",
					Transport:     "multipart/form-data",
					Conformance:   "transport_only",
				},
				Response: module.RESTPayload{
					ProtoFullName: "vrooli.image_tools.v1.ai.SubmitAIResponse",
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
