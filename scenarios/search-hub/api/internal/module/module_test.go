package module_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"search-hub/internal/module"

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
		ID:          "providers_register",
		Path:        "/vrooli.search_hub.v1.registry.RegistryService/RegisterProvider",
		Method:      http.MethodPost,
		Summary:     "Register provider",
		Description: "Registers a provider descriptor",
		Category:    "registry",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"descriptor": "ProviderDescriptor"},
		},
		Response: &module.Schema{Type: "object"},
		Errors: []module.ErrorDesc{{
			Status:      http.StatusBadRequest,
			Code:        "invalid_argument",
			Description: "Invalid descriptor",
		}},
		Examples: []module.Example{{
			Name: "Register",
			Curl: "curl http://localhost:${API_PORT}/vrooli.search_hub.v1.registry.RegistryService/RegisterProvider",
		}},
	}

	data, err := json.Marshal(descriptor)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, "providers_register", got["id"])
	require.Equal(t, "/vrooli.search_hub.v1.registry.RegistryService/RegisterProvider", got["path"])
	require.Equal(t, http.MethodPost, got["method"])
	require.Contains(t, got, "request")
	require.Contains(t, got, "response")
	require.Contains(t, got, "errors")
	require.Contains(t, got, "examples")
	require.NotContains(t, got, "cli_mapping")
}
