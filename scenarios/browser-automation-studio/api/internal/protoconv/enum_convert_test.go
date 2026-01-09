package protoconv

import (
	"testing"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

func TestStringToActionType(t *testing.T) {
	tests := []struct {
		input    string
		expected basactions.ActionType
	}{
		// Canonical names
		{"navigate", basactions.ActionType_ACTION_TYPE_NAVIGATE},
		{"click", basactions.ActionType_ACTION_TYPE_CLICK},
		{"input", basactions.ActionType_ACTION_TYPE_INPUT},
		{"wait", basactions.ActionType_ACTION_TYPE_WAIT},
		{"assert", basactions.ActionType_ACTION_TYPE_ASSERT},
		{"scroll", basactions.ActionType_ACTION_TYPE_SCROLL},
		{"select", basactions.ActionType_ACTION_TYPE_SELECT},
		{"evaluate", basactions.ActionType_ACTION_TYPE_EVALUATE},
		{"keyboard", basactions.ActionType_ACTION_TYPE_KEYBOARD},
		{"hover", basactions.ActionType_ACTION_TYPE_HOVER},
		{"screenshot", basactions.ActionType_ACTION_TYPE_SCREENSHOT},
		{"focus", basactions.ActionType_ACTION_TYPE_FOCUS},
		{"blur", basactions.ActionType_ACTION_TYPE_BLUR},
		{"subflow", basactions.ActionType_ACTION_TYPE_SUBFLOW},
		{"setVariable", basactions.ActionType_ACTION_TYPE_SET_VARIABLE},
		{"set_variable", basactions.ActionType_ACTION_TYPE_SET_VARIABLE},
		{"loop", basactions.ActionType_ACTION_TYPE_LOOP},
		{"conditional", basactions.ActionType_ACTION_TYPE_CONDITIONAL},

		// Aliases
		{"goto", basactions.ActionType_ACTION_TYPE_NAVIGATE},
		{"type", basactions.ActionType_ACTION_TYPE_INPUT},
		{"fill", basactions.ActionType_ACTION_TYPE_INPUT},
		{"selectoption", basactions.ActionType_ACTION_TYPE_SELECT},
		{"eval", basactions.ActionType_ACTION_TYPE_EVALUATE},
		{"keypress", basactions.ActionType_ACTION_TYPE_KEYBOARD},
		{"press", basactions.ActionType_ACTION_TYPE_KEYBOARD},

		// Case insensitivity
		{"NAVIGATE", basactions.ActionType_ACTION_TYPE_NAVIGATE},
		{"Click", basactions.ActionType_ACTION_TYPE_CLICK},
		{"  INPUT  ", basactions.ActionType_ACTION_TYPE_INPUT},

		// Unknown
		{"unknown", basactions.ActionType_ACTION_TYPE_UNSPECIFIED},
		{"", basactions.ActionType_ACTION_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := StringToActionType(tt.input)
			if result != tt.expected {
				t.Errorf("StringToActionType(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestActionTypeToString(t *testing.T) {
	tests := []struct {
		input    basactions.ActionType
		expected string
	}{
		{basactions.ActionType_ACTION_TYPE_NAVIGATE, "navigate"},
		{basactions.ActionType_ACTION_TYPE_CLICK, "click"},
		{basactions.ActionType_ACTION_TYPE_INPUT, "input"},
		{basactions.ActionType_ACTION_TYPE_WAIT, "wait"},
		{basactions.ActionType_ACTION_TYPE_ASSERT, "assert"},
		{basactions.ActionType_ACTION_TYPE_SCROLL, "scroll"},
		{basactions.ActionType_ACTION_TYPE_SELECT, "select"},
		{basactions.ActionType_ACTION_TYPE_EVALUATE, "evaluate"},
		{basactions.ActionType_ACTION_TYPE_KEYBOARD, "keyboard"},
		{basactions.ActionType_ACTION_TYPE_HOVER, "hover"},
		{basactions.ActionType_ACTION_TYPE_SCREENSHOT, "screenshot"},
		{basactions.ActionType_ACTION_TYPE_FOCUS, "focus"},
		{basactions.ActionType_ACTION_TYPE_BLUR, "blur"},
		{basactions.ActionType_ACTION_TYPE_SUBFLOW, "subflow"},
		{basactions.ActionType_ACTION_TYPE_SET_VARIABLE, "setVariable"},
		{basactions.ActionType_ACTION_TYPE_LOOP, "loop"},
		{basactions.ActionType_ACTION_TYPE_CONDITIONAL, "conditional"},
		{basactions.ActionType_ACTION_TYPE_UNSPECIFIED, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := ActionTypeToString(tt.input)
			if result != tt.expected {
				t.Errorf("ActionTypeToString(%v) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestActionTypeRoundtrip(t *testing.T) {
	// Test that converting to string and back gives the same type
	actionTypes := []basactions.ActionType{
		basactions.ActionType_ACTION_TYPE_NAVIGATE,
		basactions.ActionType_ACTION_TYPE_CLICK,
		basactions.ActionType_ACTION_TYPE_INPUT,
		basactions.ActionType_ACTION_TYPE_WAIT,
		basactions.ActionType_ACTION_TYPE_ASSERT,
		basactions.ActionType_ACTION_TYPE_SCROLL,
		basactions.ActionType_ACTION_TYPE_SELECT,
		basactions.ActionType_ACTION_TYPE_EVALUATE,
		basactions.ActionType_ACTION_TYPE_KEYBOARD,
		basactions.ActionType_ACTION_TYPE_HOVER,
		basactions.ActionType_ACTION_TYPE_SCREENSHOT,
		basactions.ActionType_ACTION_TYPE_FOCUS,
		basactions.ActionType_ACTION_TYPE_BLUR,
		basactions.ActionType_ACTION_TYPE_SUBFLOW,
		basactions.ActionType_ACTION_TYPE_SET_VARIABLE,
		basactions.ActionType_ACTION_TYPE_LOOP,
		basactions.ActionType_ACTION_TYPE_CONDITIONAL,
	}

	for _, original := range actionTypes {
		str := ActionTypeToString(original)
		result := StringToActionType(str)
		if result != original {
			t.Errorf("Roundtrip failed: %v -> %q -> %v", original, str, result)
		}
	}
}
