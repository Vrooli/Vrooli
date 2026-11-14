package ops

import (
	"image-tools/internal/module"

	opsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops/ops_v1connect"
)

// Endpoints describes the ops module's surface. ListOperations is a generated
// Connect procedure (discovery); the run + blob edges are deliberate REST
// exceptions — image bytes can't ride a proto field, but params and result
// metadata stay proto-typed (see the proto_payloads declarations).
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "ops_import_link",
		Path:        "/api/v1/import/link",
		Method:      "POST",
		Summary:     "Import an image from a public link",
		Description: "Fetches a public HTTP(S) image URL through an SSRF-safe bounded downloader, validates raster dimensions and stores the bytes as an Image Tools input blob.",
		Category:    "ops",
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_request", Description: "URL is invalid or resolves to a non-public address"},
			{Status: 413, Code: "invalid_request", Description: "Remote image exceeds the encoded byte limit"},
			{Status: 422, Code: "invalid_request", Description: "Remote response is not a supported image or violates raster limits"},
		},
		Examples: []module.Example{
			{Name: "Import a public image URL", Curl: "curl -X POST 'http://localhost:${API_PORT}/api/v1/import/link' -H 'Content-Type: application/json' -d '{\"url\":\"https://example.org/image.png\"}'"},
		},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonThirdPartyShape,
			Note:   "This small REST action accepts a URL and returns an opaque input reference; the referenced image bytes are fetched server-side and never exposed as a proxy response.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "json", Conformance: "transport_only"},
				Response: module.RESTPayload{Transport: "json", Conformance: "transport_only"},
				Error:    module.RESTPayload{ProtoFullName: "vrooli.image_tools.v1.shared.ErrorEnvelope", Transport: "json", Conformance: "protojson"},
			},
		},
	},
	{
		ID:          "ops_list_operations",
		Path:        opsconnect.OpsServiceListOperationsProcedure,
		Method:      "POST",
		Summary:     "List deterministic operations",
		Description: "Returns the deterministic operation catalog plus the formats the codec layer can decode and encode.",
		Category:    "ops",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"operations":        "array<OperationInfo>",
				"decodable_formats": "array<string>",
				"encodable_formats": "array<string>",
			},
		},
		Examples: []module.Example{
			{Name: "List operations", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.ops.OpsService/ListOperations -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "ops_run",
		Path:        "/api/v1/ops/{operation}",
		Method:      "POST",
		Summary:     "Run a deterministic operation on an uploaded image",
		Description: "Executes one deterministic operation on a multipart-uploaded image. The `file` part carries the image bytes; the `params` part carries OpParams as protojson; an optional `overlay` part carries a watermark image. `output=bytes` (default) streams the result; `output=blob` returns RunOpResponse JSON; a `path` query writes the result to a caller-owned host path.",
		Category:    "ops",
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_request", Description: "Malformed multipart form, missing image part, or invalid params"},
			{Status: 404, Code: "not_found", Description: "Unknown operation"},
			{Status: 413, Code: "invalid_request", Description: "Image exceeds the ingest byte limit"},
			{Status: 422, Code: "invalid_request", Description: "Undecodable image or invalid operation parameters"},
		},
		Examples: []module.Example{
			{Name: "Resize an image", Curl: "curl -X POST 'http://localhost:${API_PORT}/api/v1/ops/resize?output=bytes' -F file=@in.png -F 'params={\"resize\":{\"width\":256}}' -o out.png"},
		},

		RESTException: &module.RESTException{
			Reason: module.RESTReasonMultipartUpload,
			Note:   "Image bytes cannot ride a proto field; the request is multipart (file + protojson params). Parameters stay proto-typed (OpParams) and the result metadata is proto-typed (RunOpResponse).",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request: module.RESTPayload{
					ProtoFullName: "vrooli.image_tools.v1.ops.OpParams",
					Transport:     "multipart/form-data",
					Conformance:   "transport_only",
				},
				Response: module.RESTPayload{
					ProtoFullName: "vrooli.image_tools.v1.ops.RunOpResponse",
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
	{
		ID:          "ops_blob_get",
		Path:        "/api/v1/blobs/{key}",
		Method:      "GET",
		Summary:     "Fetch a result blob",
		Description: "Serves the bytes of a managed result blob by key. Browser-facing binary fetch with no generated client; used by the UI to display op inputs/outputs.",
		Category:    "ops",
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No blob with that key"},
		},
		Examples: []module.Example{
			{Name: "Fetch a blob", Curl: "curl http://localhost:${API_PORT}/api/v1/blobs/out/<id>.png -o out.png"},
		},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "Plain GET serving opaque image bytes to a browser <img>/download; no generated client applies and the body is binary, not proto.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{Transport: "none", Conformance: "none"},
				Error: module.RESTPayload{
					ProtoFullName: "vrooli.image_tools.v1.shared.ErrorEnvelope",
					Transport:     "json",
					Conformance:   "protojson",
				},
			},
		},
	},
}
