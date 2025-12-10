package agents

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

// --- Test Execution Provider (Mock) ---

// MockExecutionProvider is a test double for ExecutionProvider.
type MockExecutionProvider struct {
	NameValue         string
	Available         bool
	AvailableReason   string
	BuildCommandFunc  func(ctx context.Context, params ExecutionParams) (*exec.Cmd, error)
	SessionIDResponse string
}

func (p *MockExecutionProvider) Name() string {
	return p.NameValue
}

func (p *MockExecutionProvider) IsAvailable(ctx context.Context) AvailabilityResult {
	return AvailabilityResult{
		Available:  p.Available,
		BinaryPath: "/mock/path",
		Reason:     p.AvailableReason,
	}
}

func (p *MockExecutionProvider) BuildCommand(ctx context.Context, params ExecutionParams) (*exec.Cmd, error) {
	if p.BuildCommandFunc != nil {
		return p.BuildCommandFunc(ctx, params)
	}
	return exec.Command("echo", "mock"), nil
}

func (p *MockExecutionProvider) ExtractSessionID(output string) string {
	return p.SessionIDResponse
}

// --- ResourceOpenCodeProvider Tests ---

func TestResourceOpenCodeProvider_Name(t *testing.T) {
	provider := NewResourceOpenCodeProvider()

	if provider.Name() != "resource-opencode" {
		t.Errorf("Expected name 'resource-opencode', got '%s'", provider.Name())
	}
}

func TestResourceOpenCodeProvider_BuildCommand_RequiresPrompt(t *testing.T) {
	provider := &ResourceOpenCodeProvider{
		binaryPath: "/mock/resource-opencode",
	}

	_, err := provider.BuildCommand(context.Background(), ExecutionParams{
		Model: "test-model",
		// Prompt is missing
	})

	if err == nil {
		t.Error("Expected error for missing prompt")
	}
	if !strings.Contains(err.Error(), "prompt") {
		t.Errorf("Expected error about prompt, got: %v", err)
	}
}

func TestResourceOpenCodeProvider_BuildCommand_RequiresModel(t *testing.T) {
	provider := &ResourceOpenCodeProvider{
		binaryPath: "/mock/resource-opencode",
	}

	_, err := provider.BuildCommand(context.Background(), ExecutionParams{
		Prompt: "test prompt",
		// Model is missing
	})

	if err == nil {
		t.Error("Expected error for missing model")
	}
	if !strings.Contains(err.Error(), "model") {
		t.Errorf("Expected error about model, got: %v", err)
	}
}

func TestResourceOpenCodeProvider_BuildCommand_IncludesAllParams(t *testing.T) {
	provider := &ResourceOpenCodeProvider{
		binaryPath: "/usr/bin/resource-opencode",
	}

	cmd, err := provider.BuildCommand(context.Background(), ExecutionParams{
		Prompt:         "write tests",
		Model:          "claude-3-opus",
		Provider:       "anthropic",
		WorkingDir:     "/home/test/project",
		TimeoutSeconds: 300,
		MaxTurns:       50,
		AllowedTools:   []string{"read", "write", "bash(go test)"},
		AgentID:        "agent-123",
		Scenario:       "test-genie",
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that key arguments are included
	args := strings.Join(cmd.Args, " ")

	requiredParts := []string{
		"agents", "run",
		"--prompt", "write tests",
		"--model", "claude-3-opus",
		"--provider", "anthropic",
		"--timeout", "300",
		"--max-turns", "50",
		"--allowed-tools",
		"--directory", "/home/test/project",
	}

	for _, part := range requiredParts {
		if !strings.Contains(args, part) {
			t.Errorf("Expected command to contain '%s', got: %s", part, args)
		}
	}

	// Check working directory is set
	if cmd.Dir != "/home/test/project" {
		t.Errorf("Expected working dir '/home/test/project', got '%s'", cmd.Dir)
	}
}

func TestResourceOpenCodeProvider_BuildCommand_DefaultsProvider(t *testing.T) {
	provider := &ResourceOpenCodeProvider{
		binaryPath: "/usr/bin/resource-opencode",
	}

	cmd, err := provider.BuildCommand(context.Background(), ExecutionParams{
		Prompt: "test",
		Model:  "test-model",
		// Provider not specified - should default to openrouter
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, "--provider openrouter") {
		t.Errorf("Expected default provider 'openrouter', got: %s", args)
	}
}

func TestResourceOpenCodeProvider_ExtractSessionID(t *testing.T) {
	provider := NewResourceOpenCodeProvider()

	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "standard format",
			output:   "Starting agent...\nCreated OpenCode session: abc123-def456\nRunning...",
			expected: "abc123-def456",
		},
		{
			name:     "no session ID",
			output:   "Starting agent...\nRunning...\nDone.",
			expected: "",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "",
		},
		{
			name:     "prefix only",
			output:   "Created OpenCode session:",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.ExtractSessionID(tt.output)
			if result != tt.expected {
				t.Errorf("Expected session ID '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// --- ExecutionProviderSelector Tests ---

func TestExecutionProviderSelector_SelectsFirstAvailable(t *testing.T) {
	unavailableProvider := &MockExecutionProvider{
		NameValue:       "unavailable",
		Available:       false,
		AvailableReason: "not installed",
	}
	availableProvider := &MockExecutionProvider{
		NameValue:       "available",
		Available:       true,
		AvailableReason: "ready",
	}

	selector := NewExecutionProviderSelector(
		[]ExecutionProvider{unavailableProvider, availableProvider},
		nil,
	)

	result := selector.SelectProvider(context.Background())

	if !result.Available {
		t.Error("Expected available provider to be found")
	}
	if result.ProviderName != "available" {
		t.Errorf("Expected provider 'available', got '%s'", result.ProviderName)
	}
	if result.CheckedCount != 2 {
		t.Errorf("Expected 2 providers checked, got %d", result.CheckedCount)
	}
}

func TestExecutionProviderSelector_UsesFallback(t *testing.T) {
	unavailableProvider := &MockExecutionProvider{
		NameValue: "unavailable",
		Available: false,
	}
	fallbackProvider := &MockExecutionProvider{
		NameValue: "fallback",
		Available: true,
	}

	selector := NewExecutionProviderSelector(
		[]ExecutionProvider{unavailableProvider},
		fallbackProvider,
	)

	result := selector.SelectProvider(context.Background())

	if !result.Available {
		t.Error("Expected fallback provider to be available")
	}
	if result.ProviderName != "fallback" {
		t.Errorf("Expected provider 'fallback', got '%s'", result.ProviderName)
	}
	if !result.UsedFallback {
		t.Error("Expected UsedFallback to be true")
	}
}

func TestExecutionProviderSelector_NoProvidersAvailable(t *testing.T) {
	unavailableProvider := &MockExecutionProvider{
		NameValue: "unavailable",
		Available: false,
	}

	selector := NewExecutionProviderSelector(
		[]ExecutionProvider{unavailableProvider},
		nil, // No fallback
	)

	result := selector.SelectProvider(context.Background())

	if result.Available {
		t.Error("Expected no providers to be available")
	}
	if result.Provider != nil {
		t.Error("Expected Provider to be nil when none available")
	}
	if !strings.Contains(result.Reason, "no execution providers available") {
		t.Errorf("Expected reason about no providers, got: %s", result.Reason)
	}
}

func TestExecutionProviderSelector_ListProviders(t *testing.T) {
	availableProvider := &MockExecutionProvider{
		NameValue:       "primary",
		Available:       true,
		AvailableReason: "ready",
	}
	fallbackProvider := &MockExecutionProvider{
		NameValue:       "backup",
		Available:       true,
		AvailableReason: "ready",
	}

	selector := NewExecutionProviderSelector(
		[]ExecutionProvider{availableProvider},
		fallbackProvider,
	)

	infos := selector.ListProviders(context.Background())

	if len(infos) != 2 {
		t.Fatalf("Expected 2 providers, got %d", len(infos))
	}

	// First should be primary
	if infos[0].Name != "primary" {
		t.Errorf("Expected first provider 'primary', got '%s'", infos[0].Name)
	}
	if infos[0].IsFallback {
		t.Error("First provider should not be fallback")
	}

	// Second should be fallback
	if !strings.Contains(infos[1].Name, "backup") {
		t.Errorf("Expected second provider to contain 'backup', got '%s'", infos[1].Name)
	}
	if !infos[1].IsFallback {
		t.Error("Second provider should be marked as fallback")
	}
}

// --- Decision Function Tests ---

func TestProviderSelection_Decision_FirstAvailable(t *testing.T) {
	// This tests the decision criteria documented in SelectProvider:
	// "Return the first one that is available"

	provider1 := &MockExecutionProvider{NameValue: "first", Available: true}
	provider2 := &MockExecutionProvider{NameValue: "second", Available: true}

	selector := NewExecutionProviderSelector(
		[]ExecutionProvider{provider1, provider2},
		nil,
	)

	result := selector.SelectProvider(context.Background())

	// Decision: First available wins, even if second is also available
	if result.ProviderName != "first" {
		t.Errorf("Decision violated: expected first available provider 'first', got '%s'", result.ProviderName)
	}
}

func TestProviderSelection_Decision_FallbackOnlyIfNoOthers(t *testing.T) {
	// This tests the decision criteria:
	// "If none available and fallback exists, use fallback"

	primaryUnavailable := &MockExecutionProvider{NameValue: "primary", Available: false}
	fallback := &MockExecutionProvider{NameValue: "fallback", Available: true}

	selector := NewExecutionProviderSelector(
		[]ExecutionProvider{primaryUnavailable},
		fallback,
	)

	result := selector.SelectProvider(context.Background())

	// Decision: Fallback used only when primary exhausted
	if result.ProviderName != "fallback" {
		t.Errorf("Decision violated: expected fallback when primary unavailable")
	}
	if !result.UsedFallback {
		t.Error("Decision violated: UsedFallback should be true")
	}
}
