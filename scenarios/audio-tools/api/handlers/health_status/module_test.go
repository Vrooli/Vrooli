package health_status_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"audio-tools/handlers/health_status"
	"audio-tools/internal/capabilities"
	capmocks "audio-tools/internal/capabilities/mocks"

	"github.com/gorilla/mux"
)

func describeRequest(t *testing.T, registry *capabilities.Registry) *httptest.ResponseRecorder {
	t.Helper()
	router := mux.NewRouter()
	health_status.Module(health_status.Deps{Registry: registry}).Mount(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/capabilities/describe", nil))
	return response
}

func TestDescribeCapabilitiesReturnsSharedRegistryJSON(t *testing.T) {
	registry := capabilities.NewRegistry(
		[]capabilities.Def{{
			ID: "ollama", Name: "Ollama", Description: "Local model inference",
			DependencyKind: capabilities.DependencyResource, DependencySlug: "ollama",
		}},
		map[string]capabilities.Checker{
			"ollama": capmocks.NewFakeChecker(capabilities.StatusAvailable, "healthy"),
		},
		time.Minute,
	)
	response := describeRequest(t, registry)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body struct {
		Definitions []capabilities.Def   `json:"definitions"`
		States      []capabilities.State `json:"states"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Definitions) != 1 || len(body.States) != 1 || body.States[0].Status != capabilities.StatusAvailable {
		t.Fatalf("unexpected description: %+v", body)
	}
}

func TestDescribeCapabilitiesWithoutRegistryReturnsUnavailable(t *testing.T) {
	response := describeRequest(t, nil)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestDescribeCapabilitiesRejectsInvalidRegistry(t *testing.T) {
	registry := capabilities.NewRegistry(
		[]capabilities.Def{{
			Name: "OpenRouter", Description: "Cloud model inference",
			DependencyKind: capabilities.DependencyResource, DependencySlug: "openrouter",
		}},
		map[string]capabilities.Checker{},
		time.Minute,
	)
	response := describeRequest(t, registry)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}
