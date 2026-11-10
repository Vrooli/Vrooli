package quality

import (
	qualityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality/quality_v1connect"
	"image-tools/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "quality_profiles", Path: qualityconnect.QualityServiceListQualityProfilesProcedure, Method: "POST", Summary: "List image-quality evaluation profiles", Description: "Reference metrics and local-judge rubrics for image, matte, mask, text and structured outputs.", Category: "quality"},
	{ID: "quality_evaluate", Path: "/api/v1/quality/evaluate", Method: "POST", Summary: "Evaluate an output set against references and regression budgets", Description: "Multipart manifest and image parts; schedules a durable quality evaluation. Wait for its job and read the retained EvaluationReport. A failed quality gate is a completed report with passed=false.", Category: "quality", RESTException: &module.RESTException{Reason: module.RESTReasonMultipartUpload, Note: "Image set payloads use multipart; manifest, baseline and result metadata use the quality protobuf contract.", ProtoPayloads: &module.RESTProtoPayloads{Request: module.RESTPayload{Transport: "multipart/form-data", Conformance: "transport_only"}, Response: module.RESTPayload{ProtoFullName: "vrooli.image_tools.v1.quality.SubmitEvaluationResponse", Transport: "json", Conformance: "protojson"}, Error: module.RESTPayload{ProtoFullName: "vrooli.image_tools.v1.shared.ErrorEnvelope", Transport: "json", Conformance: "protojson"}}}},
}
