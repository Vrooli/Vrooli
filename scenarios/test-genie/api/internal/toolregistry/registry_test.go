package toolregistry

import (
	"context"
	"reflect"
	"testing"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

type stubProvider struct {
	name       string
	tools      []*toolspb.ToolDefinition
	categories []*toolspb.ToolCategory
}

func (s stubProvider) Name() string { return s.name }

func (s stubProvider) Tools(context.Context) []*toolspb.ToolDefinition {
	return s.tools
}

func (s stubProvider) Categories(context.Context) []*toolspb.ToolCategory {
	return s.categories
}

func TestRegistryPreservesRegistrationOrderAndCategoryOverrides(t *testing.T) {
	registry := NewRegistry(RegistryConfig{
		ScenarioName:        "test-genie",
		ScenarioVersion:     "1.0.0",
		ScenarioDescription: "testing",
	})

	registry.RegisterProvider(stubProvider{
		name: "first",
		tools: []*toolspb.ToolDefinition{
			{Name: "alpha"},
		},
		categories: []*toolspb.ToolCategory{
			{Id: "shared", Name: "First"},
			{Id: "alpha", Name: "Alpha"},
		},
	})
	registry.RegisterProvider(stubProvider{
		name: "second",
		tools: []*toolspb.ToolDefinition{
			{Name: "beta"},
		},
		categories: []*toolspb.ToolCategory{
			{Id: "shared", Name: "Override"},
			{Id: "beta", Name: "Beta"},
		},
	})

	if got := registry.ListToolNames(context.Background()); !reflect.DeepEqual(got, []string{"alpha", "beta"}) {
		t.Fatalf("expected deterministic tool order, got %v", got)
	}

	manifest := registry.GetManifest(context.Background())
	if manifest.Scenario.Name != "test-genie" {
		t.Fatalf("expected scenario metadata to be preserved, got %+v", manifest.Scenario)
	}
	if len(manifest.Categories) != 3 {
		t.Fatalf("expected three manifest categories, got %+v", manifest.Categories)
	}
	if got := []string{manifest.Categories[0].Id, manifest.Categories[1].Id, manifest.Categories[2].Id}; !reflect.DeepEqual(got, []string{"shared", "alpha", "beta"}) {
		t.Fatalf("expected stable category order, got %v", got)
	}
	if manifest.Categories[0].Name != "Override" {
		t.Fatalf("expected later provider to override shared category, got %+v", manifest.Categories)
	}
}

func TestRegistryLookupAndBuiltInProviders(t *testing.T) {
	registry := NewRegistry(RegistryConfig{})
	registry.RegisterProvider(NewFixToolProvider())
	registry.RegisterProvider(NewRequirementsToolProvider())

	if registry.ProviderCount() != 2 {
		t.Fatalf("expected two registered providers, got %d", registry.ProviderCount())
	}
	if tool := registry.GetTool(context.Background(), "spawn_fix"); tool == nil || tool.Category != "fix_operations" {
		t.Fatalf("expected spawn_fix tool to be discoverable, got %+v", tool)
	}
	if tool := registry.GetTool(context.Background(), "sync_requirements"); tool == nil || tool.Category != "requirements" {
		t.Fatalf("expected sync_requirements tool to be discoverable, got %+v", tool)
	}
	if tool := registry.GetTool(context.Background(), "missing"); tool != nil {
		t.Fatalf("expected unknown tool lookup to return nil, got %+v", tool)
	}
}
