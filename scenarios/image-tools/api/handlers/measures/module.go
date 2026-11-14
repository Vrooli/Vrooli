package measures

import (
	"encoding/json"
	"net/http"

	"image-tools/internal/httpx"
	internalmeasures "image-tools/internal/measures"
	"image-tools/internal/module"

	"github.com/gorilla/mux"
)

// Module exposes owner-collected model cold/warm/peak-memory profiles. The
// endpoint reads the same Recorder that receives profiles from benchmark runs.
func Module(recorder *internalmeasures.Recorder) module.Module {
	return module.Module{
		Name: "measures",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/measures/usage", func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet {
					w.Header().Set("Allow", http.MethodGet)
					httpx.WriteError(w, http.StatusMethodNotAllowed, httpx.CodeInvalidRequest, "method not allowed")
					return
				}
				usage, err := recorder.ModelUsageSummary(req.Context(), req.URL.Query().Get("model_id"))
				if err != nil {
					httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "model usage query failed")
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(usage); err != nil {
					return
				}
			}).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/measures/models", func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet {
					w.Header().Set("Allow", http.MethodGet)
					httpx.WriteError(w, http.StatusMethodNotAllowed, httpx.CodeInvalidRequest, "method not allowed")
					return
				}
				profiles, err := recorder.ModelProfiles(req.Context(), req.URL.Query().Get("model_id"), req.URL.Query().Get("operation"))
				if err != nil {
					httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "measure query failed")
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(profiles); err != nil {
					return
				}
			}).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the recorder's additive schema contribution.
func Schema() string { return internalmeasures.Schema() }

var Endpoints = []module.EndpointDescriptor{{
	ID:          "model_usage",
	Path:        "/api/v1/measures/usage",
	Method:      http.MethodGet,
	Summary:     "Report each model's successful durable job use",
	Description: "Returns only recorded successful model-backed job counts and last-success identity; missing models remain unknown.",
	Category:    "measures",
	Request:     &module.Schema{Type: "object", Properties: map[string]string{"model_id": "string query parameter (optional)"}},
	Response:    &module.Schema{Type: "array", Properties: map[string]string{"model_id": "string", "successful_jobs": "integer", "last_successful_at": "string", "last_operation": "string", "last_tier": "string"}},
	Errors:      []module.ErrorDesc{{Status: http.StatusInternalServerError, Code: httpx.CodeInternal, Description: "The successful model-use query could not be read"}},
	RESTException: &module.RESTException{
		Reason: module.RESTReasonOpsProbe,
		Note:   "Read-only model-use accounting from durable job traces; absent model rows mean use history is unknown.",
		ProtoPayloads: &module.RESTProtoPayloads{
			Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
			Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
			Error:    module.RESTPayload{ProtoFullName: "vrooli.image_tools.v1.shared.ErrorEnvelope", Transport: "json", Conformance: "protojson"},
		},
	},
}, {
	ID:          "model_profiles",
	Path:        "/api/v1/measures/models",
	Method:      http.MethodGet,
	Summary:     "Query model performance profiles",
	Description: "Returns owner-recorded cold, warm, and peak-memory profile summaries, including unavailable results.",
	Category:    "measures",
	Request:     &module.Schema{Type: "object", Properties: map[string]string{"model_id": "string query parameter (optional)", "operation": "string query parameter (optional)"}},
	Response:    &module.Schema{Type: "array", Properties: map[string]string{"model_id": "string", "operation": "string", "profile": "string", "status": "string", "samples": "integer", "latency_p50_ms": "integer", "latency_p95_ms": "integer", "peak_memory_bytes": "integer", "device": "string", "last_evidence": "string", "last_recorded_at": "string"}},
	Errors:      []module.ErrorDesc{{Status: http.StatusInternalServerError, Code: httpx.CodeInternal, Description: "The profile query could not be read"}},
	RESTException: &module.RESTException{
		Reason: module.RESTReasonOpsProbe,
		Note:   "Read-only profile query for owner-recorded model measurements; query parameters select model/operation and the response uses the declared profile shape rather than a Connect procedure.",
		ProtoPayloads: &module.RESTProtoPayloads{
			Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
			Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
			Error:    module.RESTPayload{ProtoFullName: "vrooli.image_tools.v1.shared.ErrorEnvelope", Transport: "json", Conformance: "protojson"},
		},
	},
}}
