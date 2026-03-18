package domain

import (
	"testing"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// =============================================================================
// EffectiveTool Helper Method Tests
// =============================================================================

func TestEffectiveTool_GetName(t *testing.T) {
	tests := []struct {
		name     string
		tool     EffectiveTool
		expected string
	}{
		{
			name:     "nil tool",
			tool:     EffectiveTool{Tool: nil},
			expected: "",
		},
		{
			name:     "has name",
			tool:     EffectiveTool{Tool: &toolspb.ToolDefinition{Name: "test_tool"}},
			expected: "test_tool",
		},
		{
			name:     "empty name",
			tool:     EffectiveTool{Tool: &toolspb.ToolDefinition{Name: ""}},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tool.GetName(); got != tc.expected {
				t.Errorf("GetName() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestEffectiveTool_GetDescription(t *testing.T) {
	tests := []struct {
		name     string
		tool     EffectiveTool
		expected string
	}{
		{
			name:     "nil tool",
			tool:     EffectiveTool{Tool: nil},
			expected: "",
		},
		{
			name:     "has description",
			tool:     EffectiveTool{Tool: &toolspb.ToolDefinition{Description: "A test tool"}},
			expected: "A test tool",
		},
		{
			name:     "empty description",
			tool:     EffectiveTool{Tool: &toolspb.ToolDefinition{Description: ""}},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tool.GetDescription(); got != tc.expected {
				t.Errorf("GetDescription() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestEffectiveTool_GetMetadata(t *testing.T) {
	tests := []struct {
		name    string
		tool    EffectiveTool
		wantNil bool
	}{
		{
			name:    "nil tool",
			tool:    EffectiveTool{Tool: nil},
			wantNil: true,
		},
		{
			name:    "nil metadata",
			tool:    EffectiveTool{Tool: &toolspb.ToolDefinition{Name: "test"}},
			wantNil: true,
		},
		{
			name: "has metadata",
			tool: EffectiveTool{Tool: &toolspb.ToolDefinition{
				Name:     "test",
				Metadata: &toolspb.ToolMetadata{EnabledByDefault: true},
			}},
			wantNil: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.tool.GetMetadata()
			if tc.wantNil && got != nil {
				t.Errorf("GetMetadata() = %v, want nil", got)
			}
			if !tc.wantNil && got == nil {
				t.Error("GetMetadata() = nil, want non-nil")
			}
		})
	}
}

func TestEffectiveTool_IsLongRunning(t *testing.T) {
	tests := []struct {
		name     string
		tool     EffectiveTool
		expected bool
	}{
		{
			name:     "nil tool",
			tool:     EffectiveTool{Tool: nil},
			expected: false,
		},
		{
			name:     "nil metadata",
			tool:     EffectiveTool{Tool: &toolspb.ToolDefinition{Name: "test"}},
			expected: false,
		},
		{
			name: "not long running",
			tool: EffectiveTool{Tool: &toolspb.ToolDefinition{
				Name:     "test",
				Metadata: &toolspb.ToolMetadata{LongRunning: false},
			}},
			expected: false,
		},
		{
			name: "is long running",
			tool: EffectiveTool{Tool: &toolspb.ToolDefinition{
				Name:     "test",
				Metadata: &toolspb.ToolMetadata{LongRunning: true},
			}},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tool.IsLongRunning(); got != tc.expected {
				t.Errorf("IsLongRunning() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestEffectiveTool_HasAsyncBehavior(t *testing.T) {
	tests := []struct {
		name     string
		tool     EffectiveTool
		expected bool
	}{
		{
			name:     "nil tool",
			tool:     EffectiveTool{Tool: nil},
			expected: false,
		},
		{
			name:     "nil metadata",
			tool:     EffectiveTool{Tool: &toolspb.ToolDefinition{Name: "test"}},
			expected: false,
		},
		{
			name: "nil async behavior",
			tool: EffectiveTool{Tool: &toolspb.ToolDefinition{
				Name:     "test",
				Metadata: &toolspb.ToolMetadata{},
			}},
			expected: false,
		},
		{
			name: "has async behavior",
			tool: EffectiveTool{Tool: &toolspb.ToolDefinition{
				Name: "test",
				Metadata: &toolspb.ToolMetadata{
					AsyncBehavior: &toolspb.AsyncBehavior{},
				},
			}},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tool.HasAsyncBehavior(); got != tc.expected {
				t.Errorf("HasAsyncBehavior() = %v, want %v", got, tc.expected)
			}
		})
	}
}
