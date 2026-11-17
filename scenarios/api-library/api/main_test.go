package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type researchAssistantResolverFixture struct {
	urls  []string
	calls int
}

func (r *researchAssistantResolverFixture) ResolveScenarioURLDefault(_ context.Context, scenario string) (string, error) {
	r.calls++
	if scenario != "research-assistant" {
		return "", fmt.Errorf("unexpected scenario %q", scenario)
	}
	index := r.calls - 1
	if index >= len(r.urls) {
		index = len(r.urls) - 1
	}
	if index < 0 {
		return "", fmt.Errorf("no URL configured")
	}
	return r.urls[index], nil
}

func TestMain(m *testing.M) {
	// Setup test environment
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")
	os.Setenv("POSTGRES_DB", "test_api_library")
	os.Setenv("POSTGRES_USER", "test")
	os.Setenv("POSTGRES_PASSWORD", "test")
	os.Setenv("QDRANT_HOST", "localhost")
	os.Setenv("QDRANT_PORT", "6333")

	// Run tests
	code := m.Run()

	// Cleanup
	os.Exit(code)
}

func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal("Failed to decode response body")
	}

	if response["status"] != "healthy" {
		t.Errorf("handler returned wrong status: got %v want %v",
			response["status"], "healthy")
	}

	if response["service"] != "api-library" {
		t.Errorf("handler returned wrong service name: got %v want %v",
			response["service"], "api-library")
	}
}

func TestResearchAssistantResolutionObservesPortDrift(t *testing.T) {
	resolver := &researchAssistantResolverFixture{urls: []string{
		"http://localhost:16111",
		"http://localhost:17422",
	}}
	first, err := resolveResearchAssistantURL(context.Background(), resolver)
	if err != nil || first != "http://localhost:16111" {
		t.Fatalf("first research-assistant URL = %q, %v", first, err)
	}
	second, err := resolveResearchAssistantURL(context.Background(), resolver)
	if err != nil || second != "http://localhost:17422" {
		t.Fatalf("URL after target restart = %q, %v; want new port", second, err)
	}
	if resolver.calls != 2 {
		t.Fatalf("resolver calls = %d, want one resolution per trigger", resolver.calls)
	}
}

func TestResearchAssistantStaticBaseURLBypassesLocalDiscovery(t *testing.T) {
	t.Setenv(researchAssistantAPIBaseURLEnv, "https://research.example.test/base")
	got, err := resolveResearchAssistantURL(context.Background(), configuredResearchAssistantResolver())
	if err != nil || got != "https://research.example.test/base" {
		t.Fatalf("configured research-assistant URL = %q, %v", got, err)
	}
}

func TestResearchAssistantSearchURLPreservesDeploymentBasePath(t *testing.T) {
	got := researchAssistantSearchURL("https://research.example.test/base/")
	if got != "https://research.example.test/base/api/search" {
		t.Fatalf("research search URL = %q, want deployed base path preserved", got)
	}
}

func TestResearchAssistantUnavailableMessageExplainsExternalConfiguration(t *testing.T) {
	message := researchAssistantUnavailableMessage(fmt.Errorf("scenario_not_running: research-assistant"))
	if !strings.Contains(message, "scenario_not_running") || !strings.Contains(message, researchAssistantAPIBaseURLEnv) {
		t.Fatalf("unavailable message = %q, want missing-target detail and static configuration hint", message)
	}
}
