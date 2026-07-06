package module_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-gateway/internal/module"

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
		ID:          "gateway_validate",
		Path:        "/vrooli.ai_gateway.v1.gateway.GatewayService/ValidateRequest",
		Method:      http.MethodPost,
		Summary:     "Validate gateway request",
		Description: "Validates a provider-neutral gateway request",
		Category:    "gateway",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"role": "string"},
		},
		Response: &module.Schema{Type: "object"},
		Errors: []module.ErrorDesc{{
			Status:      http.StatusBadRequest,
			Code:        "invalid_argument",
			Description: "Missing role",
		}},
		Examples: []module.Example{{
			Name: "Validate",
			Curl: "curl http://localhost:${API_PORT}/vrooli.ai_gateway.v1.gateway.GatewayService/ValidateRequest",
		}},
	}

	data, err := json.Marshal(descriptor)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, "gateway_validate", got["id"])
	require.Equal(t, "/vrooli.ai_gateway.v1.gateway.GatewayService/ValidateRequest", got["path"])
	require.Equal(t, http.MethodPost, got["method"])
	require.Contains(t, got, "request")
	require.Contains(t, got, "response")
	require.Contains(t, got, "errors")
	require.Contains(t, got, "examples")
	require.NotContains(t, got, "cli_mapping")
}
