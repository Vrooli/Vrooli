package module_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cleanup-manager/internal/module"

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
		ID:          "cleanup_plan_create",
		Path:        "/vrooli.cleanup_manager.v1.cleanup.CleanupService/CreatePlan",
		Method:      http.MethodPost,
		Summary:     "Create cleanup plan",
		Description: "Creates a preview-only cleanup plan",
		Category:    "cleanup",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scope": "ObservationScope"},
		},
		Response: &module.Schema{Type: "object"},
		Errors: []module.ErrorDesc{{
			Status:      http.StatusBadRequest,
			Code:        "invalid_argument",
			Description: "Invalid cleanup scope",
		}},
		Examples: []module.Example{{
			Name: "Create cleanup plan",
			Curl: "curl http://localhost:${API_PORT}/vrooli.cleanup_manager.v1.cleanup.CleanupService/CreatePlan",
		}},
	}

	data, err := json.Marshal(descriptor)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, "cleanup_plan_create", got["id"])
	require.Equal(t, "/vrooli.cleanup_manager.v1.cleanup.CleanupService/CreatePlan", got["path"])
	require.Equal(t, http.MethodPost, got["method"])
	require.Contains(t, got, "request")
	require.Contains(t, got, "response")
	require.Contains(t, got, "errors")
	require.Contains(t, got, "examples")
	require.NotContains(t, got, "cli_mapping")
}
