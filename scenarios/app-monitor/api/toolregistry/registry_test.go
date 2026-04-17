package toolregistry

import (
	"context"
	"testing"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

type testProvider struct {
	name       string
	tools      []*toolspb.ToolDefinition
	categories []*toolspb.ToolCategory
}

func (p *testProvider) Name() string { return p.name }
func (p *testProvider) Tools(context.Context) []*toolspb.ToolDefinition {
	return p.tools
}

func (p *testProvider) Categories(context.Context) []*toolspb.ToolCategory {
	return p.categories
}

func TestRegistryLifecycle(t *testing.T) {
	registry := NewRegistry(RegistryConfig{
		ScenarioName:        "app-monitor",
		ScenarioVersion:     "1.0.0",
		ScenarioDescription: "test",
	})

	if registry.ProviderCount() != 0 {
		t.Fatalf("expected zero providers, got %d", registry.ProviderCount())
	}

	provider := &testProvider{
		name: "core",
		tools: []*toolspb.ToolDefinition{
			{Name: "list_apps"},
			{Name: "get_app"},
		},
		categories: []*toolspb.ToolCategory{
			{Id: "app_discovery", Name: "Discovery"},
		},
	}

	registry.RegisterProvider(provider)

	if registry.ProviderCount() != 1 {
		t.Fatalf("expected one provider, got %d", registry.ProviderCount())
	}
	if registry.ToolCount(context.Background()) != 2 {
		t.Fatalf("expected two tools, got %d", registry.ToolCount(context.Background()))
	}

	manifest := registry.GetManifest(context.Background())
	if manifest == nil {
		t.Fatal("expected manifest")
	}
	if manifest.ProtocolVersion != ToolProtocolVersion {
		t.Fatalf("expected protocol version %q, got %q", ToolProtocolVersion, manifest.ProtocolVersion)
	}
	if len(manifest.Tools) != 2 {
		t.Fatalf("expected 2 tools in manifest, got %d", len(manifest.Tools))
	}

	found := registry.GetTool(context.Background(), "get_app")
	if found == nil {
		t.Fatal("expected get_app tool to be discoverable")
	}

	registry.UnregisterProvider("core")
	if registry.ProviderCount() != 0 {
		t.Fatalf("expected zero providers after unregister, got %d", registry.ProviderCount())
	}
}
