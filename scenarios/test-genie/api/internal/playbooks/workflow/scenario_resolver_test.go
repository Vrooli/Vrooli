package workflow

import (
	"context"
	"errors"
	"testing"
)

func TestResolveScenarioURLs_BasicResolution(t *testing.T) {
	definition := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "nav-1",
				"type": "navigate",
				"data": map[string]any{
					"destinationType": "scenario",
					"scenario":        "my-app",
					"scenarioPath":    "/dashboard",
				},
			},
		},
		"edges": []any{},
	}

	resolver := func(ctx context.Context, scenario string) (string, error) {
		if scenario == "my-app" {
			return "http://localhost:3000", nil
		}
		return "", errors.New("unknown scenario")
	}

	err := ResolveScenarioURLs(context.Background(), definition, resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify transformation
	nodes := definition["nodes"].([]any)
	node := nodes[0].(map[string]any)
	data := node["data"].(map[string]any)

	if data["destinationType"] != "url" {
		t.Errorf("expected destinationType=url, got %v", data["destinationType"])
	}
	if data["url"] != "http://localhost:3000/dashboard" {
		t.Errorf("expected url=http://localhost:3000/dashboard, got %v", data["url"])
	}
	if _, ok := data["scenario"]; ok {
		t.Error("scenario field should be removed after resolution")
	}
	if _, ok := data["scenarioPath"]; ok {
		t.Error("scenarioPath field should be removed after resolution")
	}
}

func TestResolveScenarioURLs_NoScenarioPath(t *testing.T) {
	definition := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "nav-1",
				"type": "navigate",
				"data": map[string]any{
					"destinationType": "scenario",
					"scenario":        "my-app",
				},
			},
		},
	}

	resolver := func(ctx context.Context, scenario string) (string, error) {
		return "http://localhost:3000", nil
	}

	err := ResolveScenarioURLs(context.Background(), definition, resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := definition["nodes"].([]any)
	data := nodes[0].(map[string]any)["data"].(map[string]any)

	if data["url"] != "http://localhost:3000" {
		t.Errorf("expected url=http://localhost:3000, got %v", data["url"])
	}
}

func TestResolveScenarioURLs_URLDestinationUnchanged(t *testing.T) {
	definition := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "nav-1",
				"type": "navigate",
				"data": map[string]any{
					"destinationType": "url",
					"url":             "https://example.com",
				},
			},
		},
	}

	resolver := func(ctx context.Context, scenario string) (string, error) {
		t.Error("resolver should not be called for url destinations")
		return "", nil
	}

	err := ResolveScenarioURLs(context.Background(), definition, resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := definition["nodes"].([]any)
	data := nodes[0].(map[string]any)["data"].(map[string]any)

	if data["destinationType"] != "url" {
		t.Errorf("destinationType should remain url")
	}
	if data["url"] != "https://example.com" {
		t.Errorf("url should remain unchanged")
	}
}

func TestResolveScenarioURLs_ResolverError(t *testing.T) {
	definition := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "nav-1",
				"type": "navigate",
				"data": map[string]any{
					"destinationType": "scenario",
					"scenario":        "unknown-app",
				},
			},
		},
	}

	resolver := func(ctx context.Context, scenario string) (string, error) {
		return "", errors.New("scenario not found")
	}

	err := ResolveScenarioURLs(context.Background(), definition, resolver)
	if err == nil {
		t.Fatal("expected error for failed resolution")
	}
	if !errors.Is(err, err) || err.Error() == "" {
		t.Errorf("error should contain useful information, got: %v", err)
	}
}

func TestResolveScenarioURLs_EmptyScenarioName(t *testing.T) {
	definition := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "nav-1",
				"type": "navigate",
				"data": map[string]any{
					"destinationType": "scenario",
					"scenario":        "",
				},
			},
		},
	}

	resolver := func(ctx context.Context, scenario string) (string, error) {
		return "http://localhost:3000", nil
	}

	err := ResolveScenarioURLs(context.Background(), definition, resolver)
	if err == nil {
		t.Fatal("expected error for empty scenario name")
	}
}

func TestResolveScenarioURLs_NestedWorkflowDefinition(t *testing.T) {
	// Simulates a fixture that's been expanded with nested workflowDefinition
	definition := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "subflow-1",
				"type": "subflow",
				"data": map[string]any{
					"workflowDefinition": map[string]any{
						"nodes": []any{
							map[string]any{
								"id":   "nested-nav",
								"type": "navigate",
								"data": map[string]any{
									"destinationType": "scenario",
									"scenario":        "nested-app",
									"scenarioPath":    "/nested",
								},
							},
						},
					},
				},
			},
		},
	}

	resolver := func(ctx context.Context, scenario string) (string, error) {
		if scenario == "nested-app" {
			return "http://localhost:4000", nil
		}
		return "", errors.New("unknown scenario")
	}

	err := ResolveScenarioURLs(context.Background(), definition, resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify nested node was transformed
	outerNode := definition["nodes"].([]any)[0].(map[string]any)
	nestedDef := outerNode["data"].(map[string]any)["workflowDefinition"].(map[string]any)
	nestedNode := nestedDef["nodes"].([]any)[0].(map[string]any)
	nestedData := nestedNode["data"].(map[string]any)

	if nestedData["destinationType"] != "url" {
		t.Errorf("nested node destinationType should be url, got %v", nestedData["destinationType"])
	}
	if nestedData["url"] != "http://localhost:4000/nested" {
		t.Errorf("nested node url incorrect, got %v", nestedData["url"])
	}
}

func TestResolveScenarioURLs_NilResolver(t *testing.T) {
	definition := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "nav-1",
				"type": "navigate",
				"data": map[string]any{
					"destinationType": "scenario",
					"scenario":        "my-app",
				},
			},
		},
	}

	// With nil resolver, should return nil (let BAS validation catch it)
	err := ResolveScenarioURLs(context.Background(), definition, nil)
	if err != nil {
		t.Fatalf("expected nil error with nil resolver, got: %v", err)
	}
}

func TestResolveScenarioURLs_MultipleNodes(t *testing.T) {
	definition := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "nav-1",
				"type": "navigate",
				"data": map[string]any{
					"destinationType": "scenario",
					"scenario":        "app-a",
				},
			},
			map[string]any{
				"id":   "click-1",
				"type": "click",
				"data": map[string]any{
					"selector": "#button",
				},
			},
			map[string]any{
				"id":   "nav-2",
				"type": "navigate",
				"data": map[string]any{
					"destinationType": "scenario",
					"scenario":        "app-b",
					"scenarioPath":    "/page",
				},
			},
		},
	}

	resolver := func(ctx context.Context, scenario string) (string, error) {
		switch scenario {
		case "app-a":
			return "http://localhost:3001", nil
		case "app-b":
			return "http://localhost:3002", nil
		default:
			return "", errors.New("unknown")
		}
	}

	err := ResolveScenarioURLs(context.Background(), definition, resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := definition["nodes"].([]any)

	// Check first navigate node
	data1 := nodes[0].(map[string]any)["data"].(map[string]any)
	if data1["url"] != "http://localhost:3001" {
		t.Errorf("first nav url incorrect: %v", data1["url"])
	}

	// Check click node unchanged
	data2 := nodes[1].(map[string]any)["data"].(map[string]any)
	if data2["selector"] != "#button" {
		t.Errorf("click node should be unchanged")
	}

	// Check second navigate node
	data3 := nodes[2].(map[string]any)["data"].(map[string]any)
	if data3["url"] != "http://localhost:3002/page" {
		t.Errorf("second nav url incorrect: %v", data3["url"])
	}
}

func TestBuildFullURL(t *testing.T) {
	tests := []struct {
		baseURL  string
		path     string
		expected string
	}{
		{"http://localhost:3000", "/dashboard", "http://localhost:3000/dashboard"},
		{"http://localhost:3000/", "/dashboard", "http://localhost:3000/dashboard"},
		{"http://localhost:3000", "dashboard", "http://localhost:3000/dashboard"},
		{"http://localhost:3000/", "dashboard", "http://localhost:3000/dashboard"},
		{"http://localhost:3000", "", "http://localhost:3000"},
		{"http://localhost:3000/", "", "http://localhost:3000/"},
		{"http://localhost:3000", "/", "http://localhost:3000/"},
		{"http://localhost:3000/api", "/v1", "http://localhost:3000/api/v1"},
	}

	for _, tt := range tests {
		result := buildFullURL(tt.baseURL, tt.path)
		if result != tt.expected {
			t.Errorf("buildFullURL(%q, %q) = %q, expected %q", tt.baseURL, tt.path, result, tt.expected)
		}
	}
}
