package validator

// StepDefinition defines the CLI syntax and documentation for a workflow step type.
// This is the single source of truth for CLI step syntax documentation.
type StepDefinition struct {
	Type         string         `json:"type"`
	Description  string         `json:"description"`
	Positional   *PositionalDef `json:"positional,omitempty"`
	RequiredKVs  []KVDef        `json:"requiredKVs,omitempty"`
	OptionalKVs  []KVDef        `json:"optionalKVs,omitempty"`
	RequireOneOf [][]string     `json:"requireOneOf,omitempty"`
	Examples     []StepExample  `json:"examples,omitempty"`
	CLISupported bool           `json:"cliSupported"`
}

// PositionalDef defines a positional argument for a step type.
type PositionalDef struct {
	Name        string `json:"name"`
	MapsTo      string `json:"mapsTo"`
	Description string `json:"description"`
}

// KVDef defines a key-value argument for a step type.
type KVDef struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// StepExample provides an example CLI usage for a step type.
type StepExample struct {
	Description string `json:"description"`
	CLI         string `json:"cli"`
}

// stepDefinitions is the canonical list of step type definitions.
// This drives both CLI documentation output and anti-drift tests.
var stepDefinitions = []StepDefinition{
	{
		Type:        "navigate",
		Description: "Navigate browser to a URL or scenario",
		Positional: &PositionalDef{
			Name:        "url",
			MapsTo:      "url",
			Description: "Target URL to navigate to",
		},
		OptionalKVs: []KVDef{
			{Key: "scenario", Type: "string", Description: "Scenario name (alternative to URL)"},
			{Key: "path", Type: "string", Description: "Path within scenario"},
			{Key: "waitUntil", Type: "string", Description: "Wait condition (load, domcontentloaded, networkidle)"},
		},
		RequireOneOf: [][]string{{"url", "scenario"}},
		Examples: []StepExample{
			{Description: "Navigate to URL", CLI: `--step navigate "http://example.com"`},
			{Description: "Navigate with wait condition", CLI: `--step navigate "http://example.com" waitUntil=networkidle`},
			{Description: "Navigate to scenario", CLI: `--step navigate scenario=my-app path=/dashboard`},
		},
		CLISupported: true,
	},
	{
		Type:        "click",
		Description: "Click on an element",
		Positional: &PositionalDef{
			Name:        "selector",
			MapsTo:      "selector",
			Description: "CSS selector or @selector/ reference",
		},
		OptionalKVs: []KVDef{
			{Key: "clickCount", Type: "int", Description: "Number of clicks (1-3)"},
			{Key: "button", Type: "string", Description: "Mouse button (left, right, middle)"},
			{Key: "resilience.maxAttempts", Type: "int", Description: "Retry attempts on failure"},
			{Key: "resilience.retryDelayMs", Type: "int", Description: "Delay between retries (ms)"},
		},
		Examples: []StepExample{
			{Description: "Click a button", CLI: `--step click "#submit"`},
			{Description: "Double-click with selector reference", CLI: `--step click @selector/dashboard.item clickCount=2`},
			{Description: "Right-click", CLI: `--step click "#context-menu" button=right`},
		},
		CLISupported: true,
	},
	{
		Type:        "type",
		Description: "Type text into an input element",
		Positional: &PositionalDef{
			Name:        "selector",
			MapsTo:      "selector",
			Description: "CSS selector or @selector/ reference for the input",
		},
		RequiredKVs: []KVDef{
			{Key: "text", Type: "string", Description: "Text to type (or use value=)"},
		},
		OptionalKVs: []KVDef{
			{Key: "delay", Type: "int", Description: "Delay between keystrokes (ms)"},
			{Key: "clear", Type: "bool", Description: "Clear existing content first"},
		},
		Examples: []StepExample{
			{Description: "Type into email field", CLI: `--step type "#email" text=test@example.com`},
			{Description: "Type with delay", CLI: `--step type "#search" text="hello world" delay=50`},
		},
		CLISupported: true,
	},
	{
		Type:        "assert",
		Description: "Verify element state or content",
		Positional: &PositionalDef{
			Name:        "selector",
			MapsTo:      "selector",
			Description: "CSS selector or @selector/ reference to assert on",
		},
		RequiredKVs: []KVDef{
			{Key: "assertMode", Type: "string", Description: "Assertion type: exists, not_exists, visible, contains_text, exact_text"},
		},
		OptionalKVs: []KVDef{
			{Key: "expectedText", Type: "string", Description: "Text to match (for contains_text, exact_text modes)"},
			{Key: "timeoutMs", Type: "int", Description: "Max wait time for assertion"},
		},
		Examples: []StepExample{
			{Description: "Assert element exists", CLI: `--step assert "[data-testid='dashboard']" assertMode=exists`},
			{Description: "Assert text content", CLI: `--step assert "#message" assertMode=contains_text expectedText="Success"`},
			{Description: "Assert element not visible", CLI: `--step assert ".loading-spinner" assertMode=not_exists`},
		},
		CLISupported: true,
	},
	{
		Type:        "wait",
		Description: "Wait for a duration or element",
		Positional: &PositionalDef{
			Name:        "selector",
			MapsTo:      "selector",
			Description: "CSS selector to wait for (optional)",
		},
		OptionalKVs: []KVDef{
			{Key: "durationMs", Type: "int", Description: "Fixed wait duration in milliseconds"},
			{Key: "selector", Type: "string", Description: "Wait for element to appear"},
			{Key: "state", Type: "string", Description: "Element state to wait for (visible, hidden, attached, detached)"},
			{Key: "timeoutMs", Type: "int", Description: "Max wait time"},
		},
		RequireOneOf: [][]string{{"durationMs", "selector"}},
		Examples: []StepExample{
			{Description: "Wait 2 seconds", CLI: `--step wait durationMs=2000`},
			{Description: "Wait for element", CLI: `--step wait "#results"`},
			{Description: "Wait for element to be visible", CLI: `--step wait selector="#modal" state=visible`},
		},
		CLISupported: true,
	},
	{
		Type:        "screenshot",
		Description: "Capture a screenshot",
		Positional: &PositionalDef{
			Name:        "selector",
			MapsTo:      "selector",
			Description: "CSS selector to screenshot (optional, defaults to full page)",
		},
		OptionalKVs: []KVDef{
			{Key: "fullPage", Type: "bool", Description: "Capture full scrollable page"},
			{Key: "name", Type: "string", Description: "Custom screenshot filename"},
		},
		Examples: []StepExample{
			{Description: "Screenshot viewport", CLI: `--step screenshot`},
			{Description: "Full page screenshot", CLI: `--step screenshot fullPage=true`},
			{Description: "Screenshot specific element", CLI: `--step screenshot "#chart"`},
		},
		CLISupported: true,
	},
	{
		Type:        "evaluate",
		Description: "Execute JavaScript in the page context",
		Positional: &PositionalDef{
			Name:        "expression",
			MapsTo:      "expression",
			Description: "JavaScript expression to evaluate",
		},
		OptionalKVs: []KVDef{
			{Key: "script", Type: "string", Description: "Alternative key for expression"},
			{Key: "code", Type: "string", Description: "Alternative key for expression"},
		},
		Examples: []StepExample{
			{Description: "Get page title", CLI: `--step evaluate "document.title"`},
			{Description: "Scroll to bottom", CLI: `--step evaluate "window.scrollTo(0, document.body.scrollHeight)"`},
			{Description: "Set localStorage", CLI: `--step evaluate "localStorage.setItem('theme', 'dark')"`},
		},
		CLISupported: true,
	},
	{
		Type:        "hover",
		Description: "Hover over an element",
		Positional: &PositionalDef{
			Name:        "selector",
			MapsTo:      "selector",
			Description: "CSS selector or @selector/ reference to hover",
		},
		OptionalKVs: []KVDef{
			{Key: "resilience.maxAttempts", Type: "int", Description: "Retry attempts on failure"},
		},
		Examples: []StepExample{
			{Description: "Hover over menu item", CLI: `--step hover "#dropdown-trigger"`},
			{Description: "Hover with selector reference", CLI: `--step hover @selector/nav.menuItem`},
		},
		CLISupported: true,
	},
	{
		Type:        "focus",
		Description: "Focus on an element",
		Positional: &PositionalDef{
			Name:        "selector",
			MapsTo:      "selector",
			Description: "CSS selector or @selector/ reference to focus",
		},
		OptionalKVs: []KVDef{
			{Key: "resilience.maxAttempts", Type: "int", Description: "Retry attempts on failure"},
		},
		Examples: []StepExample{
			{Description: "Focus input field", CLI: `--step focus "#search-input"`},
			{Description: "Focus with selector reference", CLI: `--step focus @selector/form.emailField`},
		},
		CLISupported: true,
	},
	{
		Type:        "blur",
		Description: "Remove focus from an element",
		Positional: &PositionalDef{
			Name:        "selector",
			MapsTo:      "selector",
			Description: "CSS selector or @selector/ reference to blur",
		},
		OptionalKVs: []KVDef{
			{Key: "resilience.maxAttempts", Type: "int", Description: "Retry attempts on failure"},
		},
		Examples: []StepExample{
			{Description: "Blur input field", CLI: `--step blur "#search-input"`},
			{Description: "Blur to trigger validation", CLI: `--step blur @selector/form.emailField`},
		},
		CLISupported: true,
	},
	{
		Type:        "select",
		Description: "Select an option from a dropdown",
		Positional: &PositionalDef{
			Name:        "selector",
			MapsTo:      "selector",
			Description: "CSS selector for the <select> element",
		},
		OptionalKVs: []KVDef{
			{Key: "optionText", Type: "string", Description: "Option visible text to select"},
			{Key: "optionValue", Type: "string", Description: "Option value attribute to select"},
			{Key: "optionIndex", Type: "int", Description: "Option index to select (0-based)"},
		},
		RequireOneOf: [][]string{{"optionText", "optionValue", "optionIndex"}},
		Examples: []StepExample{
			{Description: "Select by text", CLI: `--step select "#country" optionText="United States"`},
			{Description: "Select by value", CLI: `--step select "#currency" optionValue="USD"`},
			{Description: "Select by index", CLI: `--step select "#priority" optionIndex=2`},
		},
		CLISupported: true,
	},
	{
		Type:        "keyboard",
		Description: "Press a keyboard key",
		Positional: &PositionalDef{
			Name:        "key",
			MapsTo:      "key",
			Description: "Key to press (e.g., Enter, Tab, Escape, ArrowDown)",
		},
		OptionalKVs: []KVDef{
			{Key: "delay", Type: "int", Description: "Delay before key press (ms)"},
		},
		Examples: []StepExample{
			{Description: "Press Enter", CLI: `--step keyboard "Enter"`},
			{Description: "Press Escape", CLI: `--step keyboard "Escape"`},
			{Description: "Press Tab", CLI: `--step keyboard "Tab"`},
		},
		CLISupported: true,
	},
	{
		Type:        "shortcut",
		Description: "Press a keyboard shortcut",
		Positional: &PositionalDef{
			Name:        "keys",
			MapsTo:      "keys",
			Description: "Key combination (e.g., Control+a, Meta+Shift+s)",
		},
		Examples: []StepExample{
			{Description: "Select all", CLI: `--step shortcut "Control+a"`},
			{Description: "Copy", CLI: `--step shortcut "Control+c"`},
			{Description: "Save (macOS)", CLI: `--step shortcut "Meta+s"`},
		},
		CLISupported: true,
	},
	{
		Type:        "extract",
		Description: "Extract data from an element",
		Positional: &PositionalDef{
			Name:        "selector",
			MapsTo:      "selector",
			Description: "CSS selector or @selector/ reference to extract from",
		},
		OptionalKVs: []KVDef{
			{Key: "attribute", Type: "string", Description: "Attribute to extract (default: textContent)"},
			{Key: "outputKey", Type: "string", Description: "Key to store extracted value"},
		},
		Examples: []StepExample{
			{Description: "Extract text content", CLI: `--step extract "#price"`},
			{Description: "Extract href attribute", CLI: `--step extract "a.download-link" attribute=href`},
			{Description: "Extract with custom key", CLI: `--step extract "#total" outputKey=totalAmount`},
		},
		CLISupported: true,
	},
	{
		Type:        "subflow",
		Description: "Execute another workflow as a subflow",
		OptionalKVs: []KVDef{
			{Key: "workflowId", Type: "string", Description: "ID of workflow to execute"},
			{Key: "workflowPath", Type: "string", Description: "Path to workflow JSON file"},
			{Key: "params", Type: "object", Description: "Parameters to pass to subflow"},
		},
		Examples: []StepExample{
			{Description: "Use workflow JSON for subflows", CLI: "(Use workflow JSON file instead of --step)"},
		},
		CLISupported: false,
	},
	{
		Type:        "dragDrop",
		Description: "Drag an element and drop it on another",
		OptionalKVs: []KVDef{
			{Key: "sourceSelector", Type: "string", Description: "CSS selector for drag source"},
			{Key: "targetSelector", Type: "string", Description: "CSS selector for drop target"},
			{Key: "sourcePosition", Type: "object", Description: "Position within source element"},
			{Key: "targetPosition", Type: "object", Description: "Position within target element"},
		},
		Examples: []StepExample{
			{Description: "Use workflow JSON for drag-drop", CLI: "(Use workflow JSON file instead of --step)"},
		},
		CLISupported: false,
	},
	{
		Type:        "loop",
		Description: "Loop over elements or iterations",
		OptionalKVs: []KVDef{
			{Key: "selector", Type: "string", Description: "CSS selector for elements to loop over"},
			{Key: "iterations", Type: "int", Description: "Number of iterations"},
			{Key: "steps", Type: "array", Description: "Steps to execute in each iteration"},
		},
		Examples: []StepExample{
			{Description: "Use workflow JSON for loops", CLI: "(Use workflow JSON file instead of --step)"},
		},
		CLISupported: false,
	},
}

// stepDefinitionMap provides O(1) lookup by step type.
var stepDefinitionMap map[string]*StepDefinition

func init() {
	stepDefinitionMap = make(map[string]*StepDefinition, len(stepDefinitions))
	for i := range stepDefinitions {
		stepDefinitionMap[stepDefinitions[i].Type] = &stepDefinitions[i]
	}
}

// GetStepDefinitions returns all step definitions.
func GetStepDefinitions() []StepDefinition {
	result := make([]StepDefinition, len(stepDefinitions))
	copy(result, stepDefinitions)
	return result
}

// GetStepDefinition returns the definition for a specific step type.
func GetStepDefinition(stepType string) (*StepDefinition, bool) {
	def, ok := stepDefinitionMap[stepType]
	return def, ok
}

// GetCLISupportedStepTypes returns the list of step types that support CLI --step syntax.
func GetCLISupportedStepTypes() []string {
	var result []string
	for _, def := range stepDefinitions {
		if def.CLISupported {
			result = append(result, def.Type)
		}
	}
	return result
}

// GetCLISupportedStepDefinitions returns only the step definitions that support CLI --step syntax.
func GetCLISupportedStepDefinitions() []StepDefinition {
	var result []StepDefinition
	for _, def := range stepDefinitions {
		if def.CLISupported {
			result = append(result, def)
		}
	}
	return result
}
