package toolregistry

import (
	"context"
	"testing"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// mockProvider is a test double for ToolProvider.
type mockProvider struct {
	name       string
	tools      []*toolspb.ToolDefinition
	categories []*toolspb.ToolCategory
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return m.tools
}

func (m *mockProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return m.categories
}

func TestNewRegistry(t *testing.T) {
	cfg := RegistryConfig{
		ScenarioName:        "test-scenario",
		ScenarioVersion:     "1.0.0",
		ScenarioDescription: "Test scenario for unit tests",
	}

	reg := NewRegistry(cfg)

	if reg == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if reg.scenarioName != cfg.ScenarioName {
		t.Errorf("scenarioName = %q, want %q", reg.scenarioName, cfg.ScenarioName)
	}
	if reg.scenarioVersion != cfg.ScenarioVersion {
		t.Errorf("scenarioVersion = %q, want %q", reg.scenarioVersion, cfg.ScenarioVersion)
	}
	if reg.ProviderCount() != 0 {
		t.Errorf("ProviderCount = %d, want 0", reg.ProviderCount())
	}
}

func TestRegistry_RegisterProvider(t *testing.T) {
	reg := NewRegistry(RegistryConfig{ScenarioName: "test"})

	provider := &mockProvider{
		name: "test-provider",
		tools: []*toolspb.ToolDefinition{
			{Name: "test_tool", Description: "A test tool"},
		},
	}

	reg.RegisterProvider(provider)

	if reg.ProviderCount() != 1 {
		t.Errorf("ProviderCount = %d, want 1", reg.ProviderCount())
	}
}

func TestRegistry_RegisterProvider_Replaces(t *testing.T) {
	reg := NewRegistry(RegistryConfig{ScenarioName: "test"})

	provider1 := &mockProvider{
		name: "same-name",
		tools: []*toolspb.ToolDefinition{
			{Name: "tool_v1", Description: "Version 1"},
		},
	}
	provider2 := &mockProvider{
		name: "same-name",
		tools: []*toolspb.ToolDefinition{
			{Name: "tool_v2", Description: "Version 2"},
		},
	}

	reg.RegisterProvider(provider1)
	reg.RegisterProvider(provider2)

	// Should still be 1 provider (replaced, not added)
	if reg.ProviderCount() != 1 {
		t.Errorf("ProviderCount = %d, want 1", reg.ProviderCount())
	}

	// Should have the v2 tool
	ctx := context.Background()
	tool := reg.GetTool(ctx, "tool_v2")
	if tool == nil {
		t.Error("GetTool(tool_v2) returned nil, expected tool from provider2")
	}
}

func TestRegistry_UnregisterProvider(t *testing.T) {
	reg := NewRegistry(RegistryConfig{ScenarioName: "test"})

	provider := &mockProvider{name: "to-remove"}
	reg.RegisterProvider(provider)

	if reg.ProviderCount() != 1 {
		t.Fatalf("ProviderCount = %d, want 1", reg.ProviderCount())
	}

	reg.UnregisterProvider("to-remove")

	if reg.ProviderCount() != 0 {
		t.Errorf("ProviderCount after unregister = %d, want 0", reg.ProviderCount())
	}
}

func TestRegistry_GetManifest(t *testing.T) {
	reg := NewRegistry(RegistryConfig{
		ScenarioName:        "test-scenario",
		ScenarioVersion:     "2.0.0",
		ScenarioDescription: "A test scenario",
	})

	provider := &mockProvider{
		name: "test-provider",
		tools: []*toolspb.ToolDefinition{
			{Name: "tool_one", Description: "First tool", Category: "cat_a"},
			{Name: "tool_two", Description: "Second tool", Category: "cat_b"},
		},
		categories: []*toolspb.ToolCategory{
			{Id: "cat_a", Name: "Category A"},
			{Id: "cat_b", Name: "Category B"},
		},
	}
	reg.RegisterProvider(provider)

	ctx := context.Background()
	manifest := reg.GetManifest(ctx)

	if manifest == nil {
		t.Fatal("GetManifest returned nil")
	}

	// Check protocol version
	if manifest.ProtocolVersion != ToolProtocolVersion {
		t.Errorf("ProtocolVersion = %q, want %q", manifest.ProtocolVersion, ToolProtocolVersion)
	}

	// Check scenario info
	if manifest.Scenario == nil {
		t.Fatal("Scenario is nil")
	}
	if manifest.Scenario.Name != "test-scenario" {
		t.Errorf("Scenario.Name = %q, want %q", manifest.Scenario.Name, "test-scenario")
	}
	if manifest.Scenario.Version != "2.0.0" {
		t.Errorf("Scenario.Version = %q, want %q", manifest.Scenario.Version, "2.0.0")
	}

	// Check tools
	if len(manifest.Tools) != 2 {
		t.Errorf("len(Tools) = %d, want 2", len(manifest.Tools))
	}

	// Check categories
	if len(manifest.Categories) != 2 {
		t.Errorf("len(Categories) = %d, want 2", len(manifest.Categories))
	}

	// Check timestamp is set
	if manifest.GeneratedAt == nil {
		t.Error("GeneratedAt is nil")
	}
}

func TestRegistry_GetManifest_MultipleProviders(t *testing.T) {
	reg := NewRegistry(RegistryConfig{ScenarioName: "test"})

	provider1 := &mockProvider{
		name: "provider-1",
		tools: []*toolspb.ToolDefinition{
			{Name: "tool_a", Description: "From provider 1"},
		},
		categories: []*toolspb.ToolCategory{
			{Id: "shared_cat", Name: "Original Name"},
		},
	}
	provider2 := &mockProvider{
		name: "provider-2",
		tools: []*toolspb.ToolDefinition{
			{Name: "tool_b", Description: "From provider 2"},
		},
		categories: []*toolspb.ToolCategory{
			{Id: "shared_cat", Name: "Overridden Name"}, // Same ID, different name
		},
	}

	reg.RegisterProvider(provider1)
	reg.RegisterProvider(provider2)

	ctx := context.Background()
	manifest := reg.GetManifest(ctx)

	// Should have tools from both providers
	if len(manifest.Tools) != 2 {
		t.Errorf("len(Tools) = %d, want 2", len(manifest.Tools))
	}

	// Categories with same ID should be merged (later wins)
	if len(manifest.Categories) != 1 {
		t.Errorf("len(Categories) = %d, want 1 (merged)", len(manifest.Categories))
	}
}

func TestRegistry_GetTool(t *testing.T) {
	reg := NewRegistry(RegistryConfig{ScenarioName: "test"})

	provider := &mockProvider{
		name: "test-provider",
		tools: []*toolspb.ToolDefinition{
			{Name: "find_me", Description: "The tool to find"},
			{Name: "other_tool", Description: "Another tool"},
		},
	}
	reg.RegisterProvider(provider)

	ctx := context.Background()

	// Test finding existing tool
	tool := reg.GetTool(ctx, "find_me")
	if tool == nil {
		t.Fatal("GetTool(find_me) returned nil")
	}
	if tool.Name != "find_me" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "find_me")
	}

	// Test not finding non-existent tool
	notFound := reg.GetTool(ctx, "does_not_exist")
	if notFound != nil {
		t.Errorf("GetTool(does_not_exist) = %v, want nil", notFound)
	}
}

func TestRegistry_ListToolNames(t *testing.T) {
	reg := NewRegistry(RegistryConfig{ScenarioName: "test"})

	provider := &mockProvider{
		name: "test-provider",
		tools: []*toolspb.ToolDefinition{
			{Name: "alpha"},
			{Name: "beta"},
			{Name: "gamma"},
		},
	}
	reg.RegisterProvider(provider)

	ctx := context.Background()
	names := reg.ListToolNames(ctx)

	if len(names) != 3 {
		t.Errorf("len(names) = %d, want 3", len(names))
	}

	// Check all names are present (order may vary)
	expected := map[string]bool{"alpha": true, "beta": true, "gamma": true}
	for _, name := range names {
		if !expected[name] {
			t.Errorf("unexpected tool name: %q", name)
		}
		delete(expected, name)
	}
	if len(expected) > 0 {
		t.Errorf("missing tool names: %v", expected)
	}
}

func TestRegistry_ToolCount(t *testing.T) {
	reg := NewRegistry(RegistryConfig{ScenarioName: "test"})

	ctx := context.Background()

	// Empty registry
	if count := reg.ToolCount(ctx); count != 0 {
		t.Errorf("ToolCount (empty) = %d, want 0", count)
	}

	// Add providers with tools
	reg.RegisterProvider(&mockProvider{
		name:  "p1",
		tools: []*toolspb.ToolDefinition{{Name: "t1"}, {Name: "t2"}},
	})
	reg.RegisterProvider(&mockProvider{
		name:  "p2",
		tools: []*toolspb.ToolDefinition{{Name: "t3"}},
	})

	if count := reg.ToolCount(ctx); count != 3 {
		t.Errorf("ToolCount = %d, want 3", count)
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	reg := NewRegistry(RegistryConfig{ScenarioName: "test"})
	ctx := context.Background()

	// Start multiple goroutines accessing the registry
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			reg.RegisterProvider(&mockProvider{
				name:  "provider",
				tools: []*toolspb.ToolDefinition{{Name: "tool"}},
			})
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = reg.GetManifest(ctx)
			_ = reg.GetTool(ctx, "tool")
			_ = reg.ProviderCount()
		}
		done <- true
	}()

	// Wait for both to complete (no race detector panic = success)
	<-done
	<-done
}

// TestRealProviders verifies that the actual tool providers work correctly.
func TestRealProviders(t *testing.T) {
	reg := NewRegistry(RegistryConfig{
		ScenarioName:        "scenario-to-cloud",
		ScenarioVersion:     "1.0.0",
		ScenarioDescription: "Test",
	})

	// Register all real providers
	reg.RegisterProvider(NewDeploymentToolProvider())
	reg.RegisterProvider(NewInspectionToolProvider())
	reg.RegisterProvider(NewValidationToolProvider())

	ctx := context.Background()
	manifest := reg.GetManifest(ctx)

	// Should have 11 tools total (4 + 5 + 2)
	expectedToolCount := 11
	if len(manifest.Tools) != expectedToolCount {
		t.Errorf("len(Tools) = %d, want %d", len(manifest.Tools), expectedToolCount)
	}

	// Verify key tools exist
	keyTools := []string{
		"create_deployment",
		"execute_deployment",
		"stop_deployment",
		"start_deployment",
		"check_deployment_status",
		"list_deployments",
		"inspect_deployment",
		"get_deployment_logs",
		"get_live_state",
		"validate_manifest",
		"run_preflight",
	}

	for _, toolName := range keyTools {
		tool := reg.GetTool(ctx, toolName)
		if tool == nil {
			t.Errorf("missing expected tool: %s", toolName)
		}
	}

	// Verify categories
	expectedCategories := 4 // deployment_lifecycle, deployment_status, deployment_inspection, validation
	if len(manifest.Categories) != expectedCategories {
		t.Errorf("len(Categories) = %d, want %d", len(manifest.Categories), expectedCategories)
	}

	// Verify execute_deployment has async behavior
	execTool := reg.GetTool(ctx, "execute_deployment")
	if execTool == nil {
		t.Fatal("execute_deployment tool not found")
	}
	if execTool.Metadata == nil {
		t.Fatal("execute_deployment has nil Metadata")
	}
	if execTool.Metadata.AsyncBehavior == nil {
		t.Error("execute_deployment should have AsyncBehavior defined")
	}
	if !execTool.Metadata.RequiresApproval {
		t.Error("execute_deployment should require approval")
	}
	if !execTool.Metadata.LongRunning {
		t.Error("execute_deployment should be long running")
	}
}
