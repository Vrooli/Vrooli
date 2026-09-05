package module_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tech-tree-designer/internal/module"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestModuleMountRegistersRoutes(t *testing.T) {
	m := module.Module{
		Name: "demo",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/demo", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			}).Methods(http.MethodPost)
		},
	}
	router := mux.NewRouter()

	m.Mount(router)

	req := httptest.NewRequest(http.MethodPost, "/demo", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)
}

func TestEndpointDescriptorJSONShape(t *testing.T) {
	descriptor := module.EndpointDescriptor{
		ID:          "health",
		Path:        "/health",
		Method:      http.MethodGet,
		Summary:     "Service health check",
		Description: "Returns API readiness.",
		Category:    "system",
		Response:    &module.Schema{Type: "object"},
		Examples: []module.Example{{
			Name: "Check",
			Curl: "curl http://localhost:${API_PORT}/health",
		}},

		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{ProtoFullName: "vrooli.tech_tree_designer.v1.health.Response", Transport: "json", Conformance: "protojson"},
				Error:    module.RESTPayload{ProtoFullName: "vrooli.tech_tree_designer.v1.errors.ErrorEnvelope", Transport: "json", Conformance: "protojson"},
			},
		},
	}

	data, err := json.Marshal(descriptor)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, "health", got["id"])
	require.Equal(t, "/health", got["path"])
	require.Equal(t, http.MethodGet, got["method"])
	require.Contains(t, got, "response")
	require.Contains(t, got, "examples")
	require.NotContains(t, got, "cli_mapping")
	require.Contains(t, got, "rest_exception")
}
