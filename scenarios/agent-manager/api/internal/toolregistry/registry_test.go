package toolregistry

import (
	"context"
	"testing"

	"agent-manager/internal/domain"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// mockToolProvider is a test double for ToolProvider.
type mockToolProvider struct {
	name       string
	tools      []*toolspb.ToolDefinition
	categories []*toolspb.ToolCategory
}

func (m *mockToolProvider) Name() string {
	return m.name
}

func (m *mockToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return m.tools
}

func (m *mockToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return m.categories
}

func TestNewRegistry(t *testing.T) {
	cfg := RegistryConfig{
		ScenarioName:        "test-scenario",
		ScenarioVersion:     "1.0.0",
		ScenarioDescription: "Test scenario description",
	}

	registry := NewRegistry(cfg)

	if registry == nil {
		t.Fatal("expected registry to be non-nil")
	}

	if registry.ProviderCount() != 0 {
		t.Errorf("expected 0 providers, got %d", registry.ProviderCount())
	}
}

func TestRegistry_RegisterProvider(t *testing.T) {
	registry := NewRegistry(RegistryConfig{
		ScenarioName:    "test",
		ScenarioVersion: "1.0.0",
	})

	provider := &mockToolProvider{
		name: "test-provider",
		tools: []*toolspb.ToolDefinition{
			{Name: "test_tool", Description: "A test tool"},
		},
	}

	registry.RegisterProvider(provider)

	if registry.ProviderCount() != 1 {
		t.Errorf("expected 1 provider, got %d", registry.ProviderCount())
	}
}

func TestRegistry_RegisterProvider_Replaces(t *testing.T) {
	registry := NewRegistry(RegistryConfig{
		ScenarioName:    "test",
		ScenarioVersion: "1.0.0",
	})

	provider1 := &mockToolProvider{
		name: "same-name",
		tools: []*toolspb.ToolDefinition{
			{Name: "tool_v1", Description: "Version 1"},
		},
	}
	provider2 := &mockToolProvider{
		name: "same-name",
		tools: []*toolspb.ToolDefinition{
			{Name: "tool_v2", Description: "Version 2"},
		},
	}

	registry.RegisterProvider(provider1)
	registry.RegisterProvider(provider2)

	if registry.ProviderCount() != 1 {
		t.Errorf("expected 1 provider after replacement, got %d", registry.ProviderCount())
	}

	// Verify it's the second provider's tools
	manifest := registry.GetManifest(context.Background())
	if len(manifest.Tools) != 1 || manifest.Tools[0].Name != "tool_v2" {
		t.Errorf("expected tool_v2, got %+v", manifest.Tools)
	}
}

func TestRegistry_UnregisterProvider(t *testing.T) {
	registry := NewRegistry(RegistryConfig{
		ScenarioName:    "test",
		ScenarioVersion: "1.0.0",
	})

	provider := &mockToolProvider{name: "test-provider"}
	registry.RegisterProvider(provider)
	registry.UnregisterProvider("test-provider")

	if registry.ProviderCount() != 0 {
		t.Errorf("expected 0 providers after unregister, got %d", registry.ProviderCount())
	}
}

func TestRegistry_GetManifest(t *testing.T) {
	cfg := RegistryConfig{
		ScenarioName:        "my-scenario",
		ScenarioVersion:     "2.0.0",
		ScenarioDescription: "My scenario description",
	}
	registry := NewRegistry(cfg)

	provider := &mockToolProvider{
		name: "test-provider",
		tools: []*toolspb.ToolDefinition{
			{Name: "tool_a", Description: "Tool A", Category: "cat1"},
			{Name: "tool_b", Description: "Tool B", Category: "cat2"},
		},
		categories: []*toolspb.ToolCategory{
			{Id: "cat1", Name: "Category 1"},
			{Id: "cat2", Name: "Category 2"},
		},
	}
	registry.RegisterProvider(provider)

	manifest := registry.GetManifest(context.Background())

	// Verify protocol version
	if manifest.ProtocolVersion != domain.ToolProtocolVersion {
		t.Errorf("expected protocol version %s, got %s", domain.ToolProtocolVersion, manifest.ProtocolVersion)
	}

	// Verify scenario info
	if manifest.Scenario.Name != "my-scenario" {
		t.Errorf("expected scenario name 'my-scenario', got %s", manifest.Scenario.Name)
	}
	if manifest.Scenario.Version != "2.0.0" {
		t.Errorf("expected scenario version '2.0.0', got %s", manifest.Scenario.Version)
	}
	if manifest.Scenario.Description != "My scenario description" {
		t.Errorf("expected description 'My scenario description', got %s", manifest.Scenario.Description)
	}

	// Verify tools
	if len(manifest.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(manifest.Tools))
	}

	// Verify categories
	if len(manifest.Categories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(manifest.Categories))
	}

	// Verify GeneratedAt is set
	if manifest.GeneratedAt == nil || manifest.GeneratedAt.AsTime().IsZero() {
		t.Error("expected GeneratedAt to be set")
	}
}

func TestRegistry_GetManifest_MultipleProviders(t *testing.T) {
	registry := NewRegistry(RegistryConfig{
		ScenarioName:    "test",
		ScenarioVersion: "1.0.0",
	})

	provider1 := &mockToolProvider{
		name: "provider-1",
		tools: []*toolspb.ToolDefinition{
			{Name: "tool_from_p1", Description: "From provider 1"},
		},
		categories: []*toolspb.ToolCategory{
			{Id: "shared", Name: "From P1"},
		},
	}
	provider2 := &mockToolProvider{
		name: "provider-2",
		tools: []*toolspb.ToolDefinition{
			{Name: "tool_from_p2", Description: "From provider 2"},
		},
		categories: []*toolspb.ToolCategory{
			{Id: "shared", Name: "From P2"}, // Overrides P1's category
		},
	}

	registry.RegisterProvider(provider1)
	registry.RegisterProvider(provider2)

	manifest := registry.GetManifest(context.Background())

	// Both tools should be present
	if len(manifest.Tools) != 2 {
		t.Errorf("expected 2 tools from both providers, got %d", len(manifest.Tools))
	}

	// Category should be deduplicated (later provider wins)
	if len(manifest.Categories) != 1 {
		t.Errorf("expected 1 category after dedup, got %d", len(manifest.Categories))
	}
}

func TestRegistry_GetTool(t *testing.T) {
	registry := NewRegistry(RegistryConfig{
		ScenarioName:    "test",
		ScenarioVersion: "1.0.0",
	})

	provider := &mockToolProvider{
		name: "test-provider",
		tools: []*toolspb.ToolDefinition{
			{Name: "my_tool", Description: "My tool description"},
		},
	}
	registry.RegisterProvider(provider)

	// Find existing tool
	tool := registry.GetTool(context.Background(), "my_tool")
	if tool == nil {
		t.Fatal("expected to find tool 'my_tool'")
	}
	if tool.Description != "My tool description" {
		t.Errorf("expected description 'My tool description', got %s", tool.Description)
	}

	// Tool not found
	notFound := registry.GetTool(context.Background(), "nonexistent")
	if notFound != nil {
		t.Errorf("expected nil for nonexistent tool, got %+v", notFound)
	}
}

func TestRegistry_ListToolNames(t *testing.T) {
	registry := NewRegistry(RegistryConfig{
		ScenarioName:    "test",
		ScenarioVersion: "1.0.0",
	})

	provider := &mockToolProvider{
		name: "test-provider",
		tools: []*toolspb.ToolDefinition{
			{Name: "alpha"},
			{Name: "beta"},
			{Name: "gamma"},
		},
	}
	registry.RegisterProvider(provider)

	names := registry.ListToolNames(context.Background())

	if len(names) != 3 {
		t.Errorf("expected 3 tool names, got %d", len(names))
	}

	// Verify all names are present
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}
	for _, expected := range []string{"alpha", "beta", "gamma"} {
		if !nameSet[expected] {
			t.Errorf("expected tool name %s in list", expected)
		}
	}
}

func TestRegistry_EmptyRegistry(t *testing.T) {
	registry := NewRegistry(RegistryConfig{
		ScenarioName:    "empty",
		ScenarioVersion: "1.0.0",
	})

	manifest := registry.GetManifest(context.Background())

	if len(manifest.Tools) != 0 {
		t.Errorf("expected empty tools list, got %d", len(manifest.Tools))
	}
	if len(manifest.Categories) != 0 {
		t.Errorf("expected empty categories list, got %d", len(manifest.Categories))
	}

	names := registry.ListToolNames(context.Background())
	if len(names) != 0 {
		t.Errorf("expected empty names list, got %d", len(names))
	}
}
