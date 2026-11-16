package analysis

import "music-tools/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID: "analysis-submit", Path: "/api/v1/analyze", Method: "POST",
		Summary: "Queue durable music analysis on the shared serialized GPU lane.", Category: "analysis",
		Request:       &module.Schema{Type: "Request", Properties: map[string]string{"source_blob_key": "string (required)", "options": "object (optional, resource-specific)"}},
		Response:      &module.Schema{Type: "AcceptedJob", Properties: map[string]string{"job_id": "string", "state": "string", "operation": "string", "source_sha256": "string"}},
		Examples:      []module.Example{{Name: "Analyze a managed audio blob", Curl: `curl -X POST "$API/api/v1/analyze" -H 'Content-Type: application/json' -d '{"source_blob_key":"library/reference.wav"}'`}},
		RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Submits a durable asynchronous operation; poll or cancel through the shared Music Tools jobs endpoints."},
	},
	{
		ID: "analysis-separate", Path: "/api/v1/separate", Method: "POST",
		Summary: "Queue durable vocal, drum, bass and other stem separation.", Category: "analysis",
		Request:       &module.Schema{Type: "Request", Properties: map[string]string{"source_blob_key": "string (required)", "options": "object (optional, resource-specific; required stems remain vocals/drums/bass/other)"}},
		Response:      &module.Schema{Type: "AcceptedJob", Properties: map[string]string{"job_id": "string", "state": "string", "operation": "string", "source_sha256": "string"}},
		Examples:      []module.Example{{Name: "Separate a managed audio blob", Curl: `curl -X POST "$API/api/v1/separate" -H 'Content-Type: application/json' -d '{"source_blob_key":"library/reference.wav"}'`}},
		RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Stem artifacts and the decomposition receipt remain in Music Tools managed storage and are referenced from the durable job."},
	},
	{ID: "analysis-result", Path: "/api/v1/analysis/{id}/result", Method: "GET", Summary: "Read the persisted analysis result for a completed analysis job.", Category: "analysis", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Reads the immutable Music Tools managed result referenced by the durable job."}},
	{ID: "analysis-stem", Path: "/api/v1/analysis/{id}/stems/{name}", Method: "GET", Summary: "Stream one persisted separated stem by name.", Category: "analysis", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Streams one of vocals, drums, bass or other from Music Tools managed storage."}},
	{ID: "analysis-library-sync", Path: "/api/v1/analysis/library-sync", Method: "GET", Summary: "Inspect durable Music Library result forwarding state.", Category: "analysis", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Reports pending, delivered, and blocked result forwarding without exposing result blobs."}},
}
