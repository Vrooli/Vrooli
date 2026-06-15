package module_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"quality-health/internal/module"

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
		ID:          "audit_run",
		Path:        "/api/v1/audit",
		Method:      http.MethodPost,
		Summary:     "Run audit",
		Description: "Runs an audit",
		Category:    "audit",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"title": "string"},
		},
		Response: &module.Schema{Type: "object"},
		Errors: []module.ErrorDesc{{
			Status:      http.StatusBadRequest,
			Code:        "invalid_argument",
			Description: "Missing title",
		}},
		Examples: []module.Example{{
			Name: "Create",
			Curl: "curl http://localhost:${API_PORT}/api/v1/audit",
		}},
		CLIMapping: &module.CLIMapping{
			Command: "quality-health audit run",
			Args:    []string{"--scenario", "<scenario>"},
		},
	}

	data, err := json.Marshal(descriptor)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, "audit_run", got["id"])
	require.Equal(t, "/api/v1/audit", got["path"])
	require.Equal(t, http.MethodPost, got["method"])
	require.Contains(t, got, "request")
	require.Contains(t, got, "response")
	require.Contains(t, got, "errors")
	require.Contains(t, got, "examples")
	require.Contains(t, got, "cli_mapping")
}
