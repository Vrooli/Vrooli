package actions

import (
	"testing"
)

func TestActionTypeString(t *testing.T) {
	tests := []struct {
		action   ActionType
		expected string
	}{
		{Navigate, "navigate"},
		{Click, "click"},
		{TypeInput, "type"},
		{SetVariable, "setVariable"},
		{Loop, "loop"},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if got := tt.action.String(); got != tt.expected {
				t.Errorf("ActionType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestActionTypeIsValid(t *testing.T) {
	tests := []struct {
		action   ActionType
		expected bool
	}{
		{Navigate, true},
		{Click, true},
		{TypeInput, true},
		{Custom, true},
		{ActionType("unknown"), false},
		{ActionType(""), false},
		{ActionType("Navigate"), false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if got := tt.action.IsValid(); got != tt.expected {
				t.Errorf("ActionType.IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseActionType(t *testing.T) {
	tests := []struct {
		input    string
		expected ActionType
	}{
		{"navigate", Navigate},
		{"click", Click},
		{"type", TypeInput},
		{"setVariable", SetVariable},
		{"unknown", Custom},
		{"", Custom},
		{"Navigate", Custom}, // case-sensitive, returns Custom for unknown
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseActionType(tt.input); got != tt.expected {
				t.Errorf("ParseActionType(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestValidateActionType(t *testing.T) {
	tests := []struct {
		input       string
		expectError bool
	}{
		{"navigate", false},
		{"click", false},
		{"type", false},
		{"custom", false},
		{"unknown", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateActionType(tt.input)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateActionType(%q) error = %v, expectError %v", tt.input, err, tt.expectError)
			}
			if err != nil {
				if _, ok := err.(*UnsupportedActionError); !ok {
					t.Errorf("ValidateActionType(%q) returned wrong error type: %T", tt.input, err)
				}
			}
		})
	}
}

func TestUnsupportedActionError(t *testing.T) {
	err := &UnsupportedActionError{ActionType: "foobar"}
	expected := "unsupported action type: foobar"
	if got := err.Error(); got != expected {
		t.Errorf("UnsupportedActionError.Error() = %q, want %q", got, expected)
	}
}

func TestRegistryCompleteness(t *testing.T) {
	// All action type constants should be in the registry
	expectedTypes := []ActionType{
		Navigate, Click, Hover, DragDrop, Scroll, Gesture, Rotate,
		TypeInput, Keyboard, Shortcut,
		Focus, Blur,
		Select, UploadFile,
		Wait, Assert, Screenshot,
		Extract, Evaluate, SetVariable, UseVariable,
		TabSwitch, FrameSwitch,
		Conditional, Loop, Subflow,
		SetCookie, GetCookie, ClearCookie, SetStorage, GetStorage, ClearStorage,
		NetworkMock,
		Custom,
	}

	for _, at := range expectedTypes {
		t.Run(string(at), func(t *testing.T) {
			if _, ok := Registry[at]; !ok {
				t.Errorf("ActionType %q not found in Registry", at)
			}
		})
	}

	// Registry should have exactly the expected number of entries
	if len(Registry) != len(expectedTypes) {
		t.Errorf("Registry has %d entries, expected %d", len(Registry), len(expectedTypes))
	}
}

func TestRegistryMetadataConsistency(t *testing.T) {
	for actionType, meta := range Registry {
		t.Run(string(actionType), func(t *testing.T) {
			// Type field should match the key
			if meta.Type != actionType {
				t.Errorf("Registry[%q].Type = %q, want %q", actionType, meta.Type, actionType)
			}

			// Category should be set
			if meta.Category == "" {
				t.Errorf("Registry[%q].Category is empty", actionType)
			}

			// Description should be set
			if meta.Description == "" {
				t.Errorf("Registry[%q].Description is empty", actionType)
			}
		})
	}
}

func TestGetMetadata(t *testing.T) {
	// Known action type
	meta := GetMetadata(Click)
	if meta.Type != Click {
		t.Errorf("GetMetadata(Click).Type = %v, want %v", meta.Type, Click)
	}
	if meta.Category != CategoryMouse {
		t.Errorf("GetMetadata(Click).Category = %v, want %v", meta.Category, CategoryMouse)
	}
	if !meta.NeedsSelectorWait {
		t.Error("GetMetadata(Click).NeedsSelectorWait should be true")
	}
	if !meta.TriggersDOMChanges {
		t.Error("GetMetadata(Click).TriggersDOMChanges should be true")
	}
	if !meta.RequiresElement {
		t.Error("GetMetadata(Click).RequiresElement should be true")
	}

	// Unknown action type should return default metadata
	unknownMeta := GetMetadata(ActionType("unknown"))
	if unknownMeta.Type != ActionType("unknown") {
		t.Errorf("GetMetadata(unknown).Type = %v, want unknown", unknownMeta.Type)
	}
	if unknownMeta.Category != CategoryCustom {
		t.Errorf("GetMetadata(unknown).Category = %v, want %v", unknownMeta.Category, CategoryCustom)
	}
}

func TestNeedsSelectorWait(t *testing.T) {
	tests := []struct {
		action   ActionType
		expected bool
	}{
		{Click, true},
		{TypeInput, true},
		{Hover, true},
		{Focus, true},
		{Extract, true},
		{Navigate, false},
		{Wait, false},
		{SetVariable, false},
		{Screenshot, false},
		{Loop, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if got := NeedsSelectorWait(tt.action); got != tt.expected {
				t.Errorf("NeedsSelectorWait(%q) = %v, want %v", tt.action, got, tt.expected)
			}
		})
	}
}

func TestTriggersDOMChanges(t *testing.T) {
	tests := []struct {
		action   ActionType
		expected bool
	}{
		{Navigate, true},
		{Click, true},
		{TypeInput, true},
		{Select, true},
		{Evaluate, true},
		{Hover, false},
		{Wait, false},
		{Screenshot, false},
		{GetCookie, false},
		{Extract, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if got := TriggersDOMChanges(tt.action); got != tt.expected {
				t.Errorf("TriggersDOMChanges(%q) = %v, want %v", tt.action, got, tt.expected)
			}
		})
	}
}

func TestRequiresElement(t *testing.T) {
	tests := []struct {
		action   ActionType
		expected bool
	}{
		{Click, true},
		{TypeInput, true},
		{Hover, true},
		{Focus, true},
		{Extract, true},
		{Select, true},
		{Navigate, false},
		{Wait, false},
		{Screenshot, false},
		{SetVariable, false},
		{Keyboard, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if got := RequiresElement(tt.action); got != tt.expected {
				t.Errorf("RequiresElement(%q) = %v, want %v", tt.action, got, tt.expected)
			}
		})
	}
}

func TestIsControlFlow(t *testing.T) {
	tests := []struct {
		action   ActionType
		expected bool
	}{
		{Conditional, true},
		{Loop, true},
		{Subflow, true},
		{Click, false},
		{Navigate, false},
		{Wait, false},
		{SetVariable, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if got := IsControlFlow(tt.action); got != tt.expected {
				t.Errorf("IsControlFlow(%q) = %v, want %v", tt.action, got, tt.expected)
			}
		})
	}
}

func TestGetCategory(t *testing.T) {
	tests := []struct {
		action   ActionType
		expected ActionCategory
	}{
		{Navigate, CategoryNavigation},
		{Click, CategoryMouse},
		{TypeInput, CategoryKeyboard},
		{Focus, CategoryFocus},
		{Select, CategoryForm},
		{Wait, CategoryAssertion},
		{Extract, CategoryData},
		{TabSwitch, CategoryContext},
		{Loop, CategoryControlFlow},
		{SetCookie, CategoryStorage},
		{NetworkMock, CategoryNetwork},
		{Custom, CategoryCustom},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if got := GetCategory(tt.action); got != tt.expected {
				t.Errorf("GetCategory(%q) = %v, want %v", tt.action, got, tt.expected)
			}
		})
	}
}

func TestAllTypes(t *testing.T) {
	types := AllTypes()
	if len(types) != len(Registry) {
		t.Errorf("AllTypes() returned %d types, expected %d", len(types), len(Registry))
	}

	// Verify all returned types are valid
	for _, at := range types {
		if !at.IsValid() {
			t.Errorf("AllTypes() returned invalid type: %q", at)
		}
	}
}

func TestTypesByCategory(t *testing.T) {
	mouseTypes := TypesByCategory(CategoryMouse)
	expectedMouse := []ActionType{Click, Hover, DragDrop, Scroll, Gesture, Rotate}

	if len(mouseTypes) != len(expectedMouse) {
		t.Errorf("TypesByCategory(CategoryMouse) returned %d types, expected %d", len(mouseTypes), len(expectedMouse))
	}

	// All returned types should be mouse category
	for _, at := range mouseTypes {
		if GetCategory(at) != CategoryMouse {
			t.Errorf("TypesByCategory(CategoryMouse) returned non-mouse type: %q", at)
		}
	}

	// Control flow types
	controlTypes := TypesByCategory(CategoryControlFlow)
	for _, at := range controlTypes {
		if !IsControlFlow(at) {
			t.Errorf("TypesByCategory(CategoryControlFlow) returned non-control-flow type: %q", at)
		}
	}
}

func TestInteractionTypes(t *testing.T) {
	interactionTypes := InteractionTypes()

	// All returned types should require elements
	for _, at := range interactionTypes {
		if !RequiresElement(at) {
			t.Errorf("InteractionTypes() returned type that doesn't require element: %q", at)
		}
	}

	// Should include known interaction types
	expectedInteractions := map[ActionType]bool{
		Click: true, TypeInput: true, Hover: true, Focus: true, Extract: true,
	}
	for at := range expectedInteractions {
		found := false
		for _, it := range interactionTypes {
			if it == at {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("InteractionTypes() missing expected type: %q", at)
		}
	}
}

func TestDOMChangingTypes(t *testing.T) {
	domTypes := DOMChangingTypes()

	// All returned types should trigger DOM changes
	for _, at := range domTypes {
		if !TriggersDOMChanges(at) {
			t.Errorf("DOMChangingTypes() returned type that doesn't trigger DOM changes: %q", at)
		}
	}

	// Should include known DOM-changing types
	expectedDOM := map[ActionType]bool{
		Navigate: true, Click: true, TypeInput: true, Select: true,
	}
	for at := range expectedDOM {
		found := false
		for _, dt := range domTypes {
			if dt == at {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DOMChangingTypes() missing expected type: %q", at)
		}
	}
}

func TestLoopConstants(t *testing.T) {
	// Verify loop control flow targets exist and are non-empty
	if LoopContinueTarget == "" {
		t.Error("LoopContinueTarget should not be empty")
	}
	if LoopBreakTarget == "" {
		t.Error("LoopBreakTarget should not be empty")
	}

	// Verify loop edge handles exist and are non-empty
	handles := []string{
		LoopHandleBody, LoopHandleAfter, LoopHandleBreak, LoopHandleContinue,
		LoopConditionBody, LoopConditionAfter, LoopConditionBreak, LoopConditionContinue,
	}
	for _, h := range handles {
		if h == "" {
			t.Error("Loop handle constant should not be empty")
		}
	}
}

func TestDefaultMetadata(t *testing.T) {
	// Test that unknown action types get sensible defaults
	meta := GetMetadata(ActionType("totallyUnknown"))

	if meta.Type != ActionType("totallyUnknown") {
		t.Errorf("default metadata Type = %q, want %q", meta.Type, "totallyUnknown")
	}
	if meta.Category != CategoryCustom {
		t.Errorf("default metadata Category = %q, want %q", meta.Category, CategoryCustom)
	}
	if meta.NeedsSelectorWait {
		t.Error("default metadata NeedsSelectorWait should be false")
	}
	if meta.TriggersDOMChanges {
		t.Error("default metadata TriggersDOMChanges should be false")
	}
	if meta.RequiresElement {
		t.Error("default metadata RequiresElement should be false")
	}
	if !meta.CanFail {
		t.Error("default metadata CanFail should be true")
	}
	if meta.ProducesData {
		t.Error("default metadata ProducesData should be false")
	}
	if meta.IsControlFlow {
		t.Error("default metadata IsControlFlow should be false")
	}
}

func TestCategoryConstants(t *testing.T) {
	// Verify all categories are used in the registry
	usedCategories := make(map[ActionCategory]bool)
	for _, meta := range Registry {
		usedCategories[meta.Category] = true
	}

	expectedCategories := []ActionCategory{
		CategoryNavigation, CategoryMouse, CategoryKeyboard, CategoryFocus,
		CategoryForm, CategoryAssertion, CategoryData, CategoryContext,
		CategoryControlFlow, CategoryStorage, CategoryNetwork, CategoryCustom,
	}

	for _, cat := range expectedCategories {
		if !usedCategories[cat] {
			t.Errorf("Category %q is defined but not used in Registry", cat)
		}
	}
}
