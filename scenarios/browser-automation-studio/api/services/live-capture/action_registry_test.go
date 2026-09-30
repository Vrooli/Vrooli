package livecapture

import (
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/driver"
)

func TestGenerateTypeLabel(t *testing.T) {
	tests := []struct {
		name     string
		action   driver.RecordedAction
		expected string
	}{
		{
			name:     "no payload",
			action:   driver.RecordedAction{ActionType: "type"},
			expected: "Type text",
		},
		{
			name: "with text",
			action: driver.RecordedAction{
				ActionType: "type",
				Payload:    map[string]interface{}{"text": "hello"},
			},
			expected: `Type: "hello"`,
		},
		{
			name: "long text truncated",
			action: driver.RecordedAction{
				ActionType: "type",
				Payload:    map[string]interface{}{"text": "this is a very long text that should be truncated"},
			},
			expected: `Type: "this is a very long ..."`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTypeLabel(tt.action)
			if result != tt.expected {
				t.Errorf("generateTypeLabel() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestGenerateNavigateLabel(t *testing.T) {
	action := driver.RecordedAction{
		ActionType: "navigate",
		URL:        "https://example.com",
	}

	result := generateNavigateLabel(action)
	expected := "Navigate to https://example.com"

	if result != expected {
		t.Errorf("generateNavigateLabel() = %q, expected %q", result, expected)
	}
}

func TestGenerateKeypressLabel(t *testing.T) {
	tests := []struct {
		name     string
		action   driver.RecordedAction
		expected string
	}{
		{
			name:     "no payload",
			action:   driver.RecordedAction{ActionType: "keypress"},
			expected: "Press key",
		},
		{
			name: "with key",
			action: driver.RecordedAction{
				ActionType: "keypress",
				Payload:    map[string]interface{}{"key": "Enter"},
			},
			expected: "Press Enter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateKeypressLabel(tt.action)
			if result != tt.expected {
				t.Errorf("generateKeypressLabel() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
