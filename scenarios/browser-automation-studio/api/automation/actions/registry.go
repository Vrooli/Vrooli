package actions

// Registry is the canonical registry of all supported action types and their metadata.
// This is the single source of truth used by both recording and execution systems.
var Registry = map[ActionType]ActionMetadata{
	// Navigation
	Navigate: {
		Type:               Navigate,
		Category:           CategoryNavigation,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: true,
		RequiresElement:    false,
		CanFail:            true,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Navigate to a URL",
	},

	// Mouse interactions
	Click: {
		Type:               Click,
		Category:           CategoryMouse,
		NeedsSelectorWait:  true,
		TriggersDOMChanges: true,
		RequiresElement:    true,
		CanFail:            true,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Click an element",
	},
	Hover: {
		Type:               Hover,
		Category:           CategoryMouse,
		NeedsSelectorWait:  true,
		TriggersDOMChanges: false,
		RequiresElement:    true,
		CanFail:            true,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Hover over an element",
	},
	DragDrop: {
		Type:               DragDrop,
		Category:           CategoryMouse,
		NeedsSelectorWait:  true,
		TriggersDOMChanges: true,
		RequiresElement:    true,
		CanFail:            true,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Drag element to target",
	},
	Scroll: {
		Type:               Scroll,
		Category:           CategoryMouse,
		NeedsSelectorWait:  true,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            false,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Scroll the page or element",
	},
	Gesture: {
		Type:               Gesture,
		Category:           CategoryMouse,
		NeedsSelectorWait:  true,
		TriggersDOMChanges: false,
		RequiresElement:    true,
		CanFail:            true,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Perform touch gesture",
	},
	Rotate: {
		Type:               Rotate,
		Category:           CategoryMouse,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            false,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Rotate device orientation",
	},

	// Keyboard/input
	TypeInput: {
		Type:               TypeInput,
		Category:           CategoryKeyboard,
		NeedsSelectorWait:  true,
		TriggersDOMChanges: true,
		RequiresElement:    true,
		CanFail:            true,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Type text into an element",
	},
	Keyboard: {
		Type:               Keyboard,
		Category:           CategoryKeyboard,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: true,
		RequiresElement:    false,
		CanFail:            false,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Press keyboard key(s)",
	},
	Shortcut: {
		Type:               Shortcut,
		Category:           CategoryKeyboard,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: true,
		RequiresElement:    false,
		CanFail:            false,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Execute keyboard shortcut",
	},

	// Focus management
	Focus: {
		Type:               Focus,
		Category:           CategoryFocus,
		NeedsSelectorWait:  true,
		TriggersDOMChanges: false,
		RequiresElement:    true,
		CanFail:            true,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Focus an element",
	},
	Blur: {
		Type:               Blur,
		Category:           CategoryFocus,
		NeedsSelectorWait:  true,
		TriggersDOMChanges: false,
		RequiresElement:    true,
		CanFail:            true,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Remove focus from element",
	},

	// Form interactions
	Select: {
		Type:               Select,
		Category:           CategoryForm,
		NeedsSelectorWait:  true,
		TriggersDOMChanges: true,
		RequiresElement:    true,
		CanFail:            true,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Select option from dropdown",
	},
	UploadFile: {
		Type:               UploadFile,
		Category:           CategoryForm,
		NeedsSelectorWait:  true,
		TriggersDOMChanges: true,
		RequiresElement:    true,
		CanFail:            true,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Upload file to input",
	},

	// Waiting and assertions
	Wait: {
		Type:               Wait,
		Category:           CategoryAssertion,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            true,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Wait for condition",
	},
	Assert: {
		Type:               Assert,
		Category:           CategoryAssertion,
		NeedsSelectorWait:  true,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            true,
		ProducesData:       true,
		IsControlFlow:      false,
		Description:        "Assert condition is true",
	},
	Screenshot: {
		Type:               Screenshot,
		Category:           CategoryAssertion,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            false,
		ProducesData:       true,
		IsControlFlow:      false,
		Description:        "Capture screenshot",
	},

	// Data extraction and manipulation
	Extract: {
		Type:               Extract,
		Category:           CategoryData,
		NeedsSelectorWait:  true,
		TriggersDOMChanges: false,
		RequiresElement:    true,
		CanFail:            true,
		ProducesData:       true,
		IsControlFlow:      false,
		Description:        "Extract data from element",
	},
	Evaluate: {
		Type:               Evaluate,
		Category:           CategoryData,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: true,
		RequiresElement:    false,
		CanFail:            true,
		ProducesData:       true,
		IsControlFlow:      false,
		Description:        "Evaluate JavaScript expression",
	},
	SetVariable: {
		Type:               SetVariable,
		Category:           CategoryData,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            false,
		ProducesData:       true,
		IsControlFlow:      false,
		Description:        "Set execution variable",
	},
	UseVariable: {
		Type:               UseVariable,
		Category:           CategoryData,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            true,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Use execution variable",
	},

	// Context switching
	TabSwitch: {
		Type:               TabSwitch,
		Category:           CategoryContext,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            true,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Switch browser tab",
	},
	FrameSwitch: {
		Type:               FrameSwitch,
		Category:           CategoryContext,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            true,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Switch iframe context",
	},

	// Control flow
	Conditional: {
		Type:               Conditional,
		Category:           CategoryControlFlow,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            false,
		ProducesData:       false,
		IsControlFlow:      true,
		Description:        "Conditional branch",
	},
	Loop: {
		Type:               Loop,
		Category:           CategoryControlFlow,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            false,
		ProducesData:       false,
		IsControlFlow:      true,
		Description:        "Loop iteration",
	},
	Subflow: {
		Type:               Subflow,
		Category:           CategoryControlFlow,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: true,
		RequiresElement:    false,
		CanFail:            true,
		ProducesData:       true,
		IsControlFlow:      true,
		Description:        "Execute child workflow",
	},

	// Cookie and storage management
	SetCookie: {
		Type:               SetCookie,
		Category:           CategoryStorage,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            false,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Set browser cookie",
	},
	GetCookie: {
		Type:               GetCookie,
		Category:           CategoryStorage,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            true,
		ProducesData:       true,
		IsControlFlow:      false,
		Description:        "Get browser cookie",
	},
	ClearCookie: {
		Type:               ClearCookie,
		Category:           CategoryStorage,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            false,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Clear browser cookies",
	},
	SetStorage: {
		Type:               SetStorage,
		Category:           CategoryStorage,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            false,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Set localStorage/sessionStorage",
	},
	GetStorage: {
		Type:               GetStorage,
		Category:           CategoryStorage,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            true,
		ProducesData:       true,
		IsControlFlow:      false,
		Description:        "Get localStorage/sessionStorage",
	},
	ClearStorage: {
		Type:               ClearStorage,
		Category:           CategoryStorage,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            false,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Clear localStorage/sessionStorage",
	},

	// Network
	NetworkMock: {
		Type:               NetworkMock,
		Category:           CategoryNetwork,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            false,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Mock network request",
	},

	// Custom/extensibility
	Custom: {
		Type:               Custom,
		Category:           CategoryCustom,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            true,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Custom action",
	},
}

// AllTypes returns a slice of all registered action types.
func AllTypes() []ActionType {
	types := make([]ActionType, 0, len(Registry))
	for t := range Registry {
		types = append(types, t)
	}
	return types
}

// TypesByCategory returns action types filtered by category.
func TypesByCategory(category ActionCategory) []ActionType {
	var types []ActionType
	for t, meta := range Registry {
		if meta.Category == category {
			types = append(types, t)
		}
	}
	return types
}

// InteractionTypes returns action types that interact with DOM elements.
func InteractionTypes() []ActionType {
	var types []ActionType
	for t, meta := range Registry {
		if meta.RequiresElement {
			types = append(types, t)
		}
	}
	return types
}

// DOMChangingTypes returns action types that might modify the DOM.
func DOMChangingTypes() []ActionType {
	var types []ActionType
	for t, meta := range Registry {
		if meta.TriggersDOMChanges {
			types = append(types, t)
		}
	}
	return types
}
