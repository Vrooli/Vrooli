package inference

import (
	"ai-gateway/internal/module"

	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "inference_run",
		Path:        inferenceconnect.InferenceServiceRunProcedure,
		Method:      "POST",
		Summary:     "Run schema-constrained typed inference",
		Description: "Executes one provider-neutral inference request and locally validates the returned JSON before marking it successful.",
		Category:    "inference",
		Request:     &module.Schema{Type: "RunRequest", Properties: map[string]string{"source": "string", "schema_json": "string", "instruction": "string", "role": "string", "turns": "Turn[]", "attachments": "Attachment[]"}},
		Response:    &module.Schema{Type: "RunResponse", Properties: map[string]string{"value_json": "string", "provider": "string", "model": "string", "validated": "bool", "usage": "Usage", "error": "InferenceError"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "The source, role, or schema is invalid"}},
	},
	{
		ID:          "inference_run_batch",
		Path:        inferenceconnect.InferenceServiceRunBatchProcedure,
		Method:      "POST",
		Summary:     "Run ordered schema-constrained typed inference batch",
		Description: "Runs up to 256 sources against one schema and returns ordered results with aggregate usage.",
		Category:    "inference",
		Request:     &module.Schema{Type: "RunBatchRequest", Properties: map[string]string{"items": "RunBatchItem[]", "schema_json": "string", "instruction": "string", "role": "string"}},
		Response:    &module.Schema{Type: "RunBatchResponse", Properties: map[string]string{"results": "RunResponse[]", "usage": "Usage"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "The batch size or schema is invalid"}},
	},
	{
		ID:          "inference_embed",
		Path:        inferenceconnect.InferenceServiceEmbedProcedure,
		Method:      "POST",
		Summary:     "Return gateway-owned embedding vectors",
		Description: "Resolves the embedding.default role and returns one vector per input text in request order.",
		Category:    "inference",
		Request:     &module.Schema{Type: "EmbedRequest", Properties: map[string]string{"texts": "string[]", "role": "string", "sampling": "SamplingControls"}},
		Response:    &module.Schema{Type: "EmbedResponse", Properties: map[string]string{"vectors": "EmbeddingVector[]", "provider": "string", "model": "string", "dimension": "int32", "usage": "Usage", "error": "InferenceError"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "The role, texts, or sampling controls are invalid"}},
	},
}
