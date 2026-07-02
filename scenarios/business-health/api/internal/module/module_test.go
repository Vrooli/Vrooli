package module_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"business-health/internal/module"

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
		ID:          "notes_create",
		Path:        "/api/v1/notes",
		Method:      http.MethodPost,
		Summary:     "Create note",
		Description: "Creates a note",
		Category:    "notes",
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
			Curl: "curl http://localhost:${API_PORT}/api/v1/notes",
		}},
	}

	data, err := json.Marshal(descriptor)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, "notes_create", got["id"])
	require.Equal(t, "/api/v1/notes", got["path"])
	require.Equal(t, http.MethodPost, got["method"])
	require.Contains(t, got, "request")
	require.Contains(t, got, "response")
	require.Contains(t, got, "errors")
	require.Contains(t, got, "examples")
	require.NotContains(t, got, "cli_mapping")
}
