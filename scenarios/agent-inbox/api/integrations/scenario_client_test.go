package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-inbox/domain"
)

func TestScenarioClient_FetchToolManifest(t *testing.T) {
	manifest := &domain.ToolManifest{
		ProtocolVersion: domain.ToolProtocolVersion,
		Scenario: domain.ScenarioInfo{
			Name:        "test-scenario",
			Version:     "1.0.0",
			Description: "Test scenario",
		},
		Tools: []domain.ToolDefinition{
			{
				Name:        "test_tool",
				Description: "A test tool",
				Metadata: domain.ToolMetadata{
					EnabledByDefault: true,
				},
			},
		},
		GeneratedAt: time.Now(),
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tools" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(manifest)
	}))
	defer server.Close()

	// Create client with mock URL resolver
	client := NewScenarioClientWithDeps(
		server.Client(),
		&staticURLResolver{url: server.URL},
		ScenarioClientConfig{
			Timeout:  10 * time.Second,
			CacheTTL: 60 * time.Second,
		},
	)

	// Fetch manifest
	result, err := client.FetchToolManifest(context.Background(), "test-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Scenario.Name != "test-scenario" {
		t.Errorf("expected scenario name 'test-scenario', got %q", result.Scenario.Name)
	}

	if len(result.Tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(result.Tools))
	}
}

func TestScenarioClient_FetchToolManifest_Caching(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		manifest := &domain.ToolManifest{
			ProtocolVersion: domain.ToolProtocolVersion,
			Scenario:        domain.ScenarioInfo{Name: "test"},
			GeneratedAt:     time.Now(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(manifest)
	}))
	defer server.Close()

	client := NewScenarioClientWithDeps(
		server.Client(),
		&staticURLResolver{url: server.URL},
		ScenarioClientConfig{
			Timeout:  10 * time.Second,
			CacheTTL: 60 * time.Second,
		},
	)

	ctx := context.Background()

	// First call should hit server
	_, err := client.FetchToolManifest(ctx, "test")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 server call, got %d", callCount)
	}

	// Second call should use cache
	_, err = client.FetchToolManifest(ctx, "test")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected still 1 server call (cached), got %d", callCount)
	}

	// Invalidate cache and call again
	client.InvalidateCache("test")
	_, err = client.FetchToolManifest(ctx, "test")
	if err != nil {
		t.Fatalf("third call failed: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 server calls after invalidate, got %d", callCount)
	}
}

func TestScenarioClient_FetchToolManifest_InvalidProtocolVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manifest := &domain.ToolManifest{
			ProtocolVersion: "99.0", // Invalid version
			GeneratedAt:     time.Now(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(manifest)
	}))
	defer server.Close()

	client := NewScenarioClientWithDeps(
		server.Client(),
		&staticURLResolver{url: server.URL},
		ScenarioClientConfig{
			Timeout:  10 * time.Second,
			CacheTTL: 60 * time.Second,
		},
	)

	_, err := client.FetchToolManifest(context.Background(), "test")
	if err == nil {
		t.Error("expected error for invalid protocol version")
	}
}

func TestScenarioClient_FetchMultiple(t *testing.T) {
	scenarios := map[string]*domain.ToolManifest{
		"scenario-a": {
			ProtocolVersion: domain.ToolProtocolVersion,
			Scenario:        domain.ScenarioInfo{Name: "scenario-a"},
			Tools:           []domain.ToolDefinition{{Name: "tool_a"}},
		},
		"scenario-b": {
			ProtocolVersion: domain.ToolProtocolVersion,
			Scenario:        domain.ScenarioInfo{Name: "scenario-b"},
			Tools:           []domain.ToolDefinition{{Name: "tool_b"}},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return a generic manifest for any request
		json.NewEncoder(w).Encode(scenarios["scenario-a"])
	}))
	defer server.Close()

	client := NewScenarioClientWithDeps(
		server.Client(),
		&staticURLResolver{url: server.URL},
		ScenarioClientConfig{
			Timeout:  10 * time.Second,
			CacheTTL: 60 * time.Second,
		},
	)

	results, errors := client.FetchMultiple(context.Background(), []string{"scenario-a", "scenario-b"})

	if len(errors) != 0 {
		t.Errorf("expected no errors, got %v", errors)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestScenarioClient_CheckScenarioStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manifest := &domain.ToolManifest{
			ProtocolVersion: domain.ToolProtocolVersion,
			Scenario:        domain.ScenarioInfo{Name: "test"},
			Tools:           []domain.ToolDefinition{{Name: "tool_1"}, {Name: "tool_2"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(manifest)
	}))
	defer server.Close()

	client := NewScenarioClientWithDeps(
		server.Client(),
		&staticURLResolver{url: server.URL},
		ScenarioClientConfig{
			Timeout:  10 * time.Second,
			CacheTTL: 60 * time.Second,
		},
	)

	status := client.CheckScenarioStatus(context.Background(), "test")

	if !status.Available {
		t.Error("expected scenario to be available")
	}

	if status.ToolCount != 2 {
		t.Errorf("expected 2 tools, got %d", status.ToolCount)
	}

	if status.Scenario != "test" {
		t.Errorf("expected scenario 'test', got %q", status.Scenario)
	}
}

// staticURLResolver always returns the same URL.
type staticURLResolver struct {
	url string
}

func (r *staticURLResolver) ResolveScenarioURL(ctx context.Context, scenarioName string) (string, error) {
	return r.url, nil
}
