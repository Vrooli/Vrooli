package integrations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-inbox/domain"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestScenarioClient_FetchToolManifest(t *testing.T) {
	manifest := &toolspb.ToolManifest{
		ProtocolVersion: domain.ToolProtocolVersion,
		Scenario: &toolspb.ScenarioInfo{
			Name:        "test-scenario",
			Version:     "1.0.0",
			Description: "Test scenario",
		},
		Tools: []*toolspb.ToolDefinition{
			{
				Name:        "test_tool",
				Description: "A test tool",
				Metadata: &toolspb.ToolMetadata{
					EnabledByDefault: true,
				},
			},
		},
		GeneratedAt: timestamppb.Now(),
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tools" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		data, _ := protojson.Marshal(manifest)
		w.Write(data)
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
		manifest := &toolspb.ToolManifest{
			ProtocolVersion: domain.ToolProtocolVersion,
			Scenario:        &toolspb.ScenarioInfo{Name: "test"},
			GeneratedAt:     timestamppb.Now(),
		}
		w.Header().Set("Content-Type", "application/json")
		data, _ := protojson.Marshal(manifest)
		w.Write(data)
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
		manifest := &toolspb.ToolManifest{
			ProtocolVersion: "99.0", // Invalid version
			GeneratedAt:     timestamppb.Now(),
		}
		w.Header().Set("Content-Type", "application/json")
		data, _ := protojson.Marshal(manifest)
		w.Write(data)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manifest := &toolspb.ToolManifest{
			ProtocolVersion: domain.ToolProtocolVersion,
			Scenario:        &toolspb.ScenarioInfo{Name: "scenario-a"},
			Tools:           []*toolspb.ToolDefinition{{Name: "tool_a"}},
			GeneratedAt:     timestamppb.Now(),
		}
		w.Header().Set("Content-Type", "application/json")
		data, _ := protojson.Marshal(manifest)
		w.Write(data)
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
		manifest := &toolspb.ToolManifest{
			ProtocolVersion: domain.ToolProtocolVersion,
			Scenario:        &toolspb.ScenarioInfo{Name: "test"},
			Tools: []*toolspb.ToolDefinition{
				{Name: "tool_1"},
				{Name: "tool_2"},
			},
			GeneratedAt: timestamppb.Now(),
		}
		w.Header().Set("Content-Type", "application/json")
		data, _ := protojson.Marshal(manifest)
		w.Write(data)
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
