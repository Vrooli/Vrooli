package validator

import (
	"testing"
)

// TestStepDefinitionsMatchGetAvailableNodeTypes verifies that all node types
// from GetAvailableNodeTypes() have corresponding step definitions.
// This prevents drift between schema_provider.go and step_definitions.go.
func TestStepDefinitionsMatchGetAvailableNodeTypes(t *testing.T) {
	nodeTypes := GetAvailableNodeTypes()
	definitions := GetStepDefinitions()

	// Build a map of defined step types
	definedTypes := make(map[string]bool)
	for _, def := range definitions {
		definedTypes[def.Type] = true
	}

	// Verify each node type has a step definition
	for _, nodeType := range nodeTypes {
		if !definedTypes[nodeType] {
			t.Errorf("node type %q from GetAvailableNodeTypes() has no step definition", nodeType)
		}
	}

	// Verify each step definition corresponds to a node type
	nodeTypeSet := make(map[string]bool)
	for _, nt := range nodeTypes {
		nodeTypeSet[nt] = true
	}

	for _, def := range definitions {
		if !nodeTypeSet[def.Type] {
			t.Errorf("step definition %q is not in GetAvailableNodeTypes()", def.Type)
		}
	}
}

// TestCLISupportedTypesAreSubset verifies that CLI-supported types
// are a subset of all node types.
func TestCLISupportedTypesAreSubset(t *testing.T) {
	allTypes := GetAvailableNodeTypes()
	cliTypes := GetCLISupportedStepTypes()

	allTypeSet := make(map[string]bool)
	for _, nt := range allTypes {
		allTypeSet[nt] = true
	}

	for _, cliType := range cliTypes {
		if !allTypeSet[cliType] {
			t.Errorf("CLI-supported type %q is not in GetAvailableNodeTypes()", cliType)
		}
	}
}

// TestStepDefinitionsCLISupportedCount verifies the expected count of CLI-supported types.
// Update this test if intentionally adding/removing CLI support for types.
func TestStepDefinitionsCLISupportedCount(t *testing.T) {
	cliTypes := GetCLISupportedStepTypes()

	// Per the plan: 14 types support CLI --step syntax
	// Types: navigate, click, type, assert, wait, screenshot, evaluate,
	//        hover, focus, blur, select, keyboard, shortcut, extract
	expectedCount := 14
	if len(cliTypes) != expectedCount {
		t.Errorf("expected %d CLI-supported types, got %d: %v", expectedCount, len(cliTypes), cliTypes)
	}
}

// TestStepDefinitionsNonCLISupported verifies which types are NOT CLI-supported.
// These are types that require workflow JSON files instead of --step syntax.
func TestStepDefinitionsNonCLISupported(t *testing.T) {
	expectedNonCLI := []string{"subflow", "dragDrop", "loop"}

	for _, stepType := range expectedNonCLI {
		def, ok := GetStepDefinition(stepType)
		if !ok {
			t.Errorf("expected non-CLI type %q not found in step definitions", stepType)
			continue
		}
		if def.CLISupported {
			t.Errorf("type %q should not be CLI-supported", stepType)
		}
	}
}

// TestStepDefinitionsHaveDescriptions verifies all definitions have descriptions.
func TestStepDefinitionsHaveDescriptions(t *testing.T) {
	for _, def := range GetStepDefinitions() {
		if def.Description == "" {
			t.Errorf("step type %q has no description", def.Type)
		}
	}
}

// TestCLISupportedDefinitionsHaveExamples verifies CLI-supported types have examples.
func TestCLISupportedDefinitionsHaveExamples(t *testing.T) {
	for _, def := range GetCLISupportedStepDefinitions() {
		if len(def.Examples) == 0 {
			t.Errorf("CLI-supported step type %q has no examples", def.Type)
		}
	}
}

// TestStepDefinitionLookup verifies the O(1) lookup works correctly.
func TestStepDefinitionLookup(t *testing.T) {
	// Test existing type
	def, ok := GetStepDefinition("navigate")
	if !ok {
		t.Fatal("expected to find 'navigate' definition")
	}
	if def.Type != "navigate" {
		t.Errorf("expected type 'navigate', got %q", def.Type)
	}

	// Test non-existing type
	_, ok = GetStepDefinition("nonexistent")
	if ok {
		t.Error("expected not to find 'nonexistent' definition")
	}
}

// TestStepDefinitionsUniqueTypes verifies no duplicate type names.
func TestStepDefinitionsUniqueTypes(t *testing.T) {
	seen := make(map[string]bool)
	for _, def := range GetStepDefinitions() {
		if seen[def.Type] {
			t.Errorf("duplicate step type: %q", def.Type)
		}
		seen[def.Type] = true
	}
}

// TestNavigateRequireOneOf verifies navigate's requireOneOf constraint.
func TestNavigateRequireOneOf(t *testing.T) {
	def, ok := GetStepDefinition("navigate")
	if !ok {
		t.Fatal("expected to find 'navigate' definition")
	}

	if len(def.RequireOneOf) == 0 {
		t.Error("navigate should have requireOneOf constraint")
		return
	}

	// Should require one of: url or scenario
	found := false
	for _, group := range def.RequireOneOf {
		hasURL := false
		hasScenario := false
		for _, key := range group {
			if key == "url" {
				hasURL = true
			}
			if key == "scenario" {
				hasScenario = true
			}
		}
		if hasURL && hasScenario {
			found = true
			break
		}
	}

	if !found {
		t.Error("navigate's requireOneOf should include [url, scenario]")
	}
}

// TestAssertRequiresAssertMode verifies assert has assertMode as required.
func TestAssertRequiresAssertMode(t *testing.T) {
	def, ok := GetStepDefinition("assert")
	if !ok {
		t.Fatal("expected to find 'assert' definition")
	}

	found := false
	for _, kv := range def.RequiredKVs {
		if kv.Key == "assertMode" {
			found = true
			break
		}
	}

	if !found {
		t.Error("assert should require assertMode key")
	}
}

// TestTypeRequiresText verifies type has text as required.
func TestTypeRequiresText(t *testing.T) {
	def, ok := GetStepDefinition("type")
	if !ok {
		t.Fatal("expected to find 'type' definition")
	}

	found := false
	for _, kv := range def.RequiredKVs {
		if kv.Key == "text" {
			found = true
			break
		}
	}

	if !found {
		t.Error("type should require text key")
	}
}

// TestSelectRequireOneOfOption verifies select requires one of the option keys.
func TestSelectRequireOneOfOption(t *testing.T) {
	def, ok := GetStepDefinition("select")
	if !ok {
		t.Fatal("expected to find 'select' definition")
	}

	if len(def.RequireOneOf) == 0 {
		t.Error("select should have requireOneOf constraint for option selection")
		return
	}

	// Should require one of: optionText, optionValue, or optionIndex
	expectedKeys := map[string]bool{"optionText": true, "optionValue": true, "optionIndex": true}
	for _, group := range def.RequireOneOf {
		matchCount := 0
		for _, key := range group {
			if expectedKeys[key] {
				matchCount++
			}
		}
		if matchCount == 3 {
			return // Found the expected group
		}
	}

	t.Error("select's requireOneOf should include [optionText, optionValue, optionIndex]")
}

// TestWaitRequireOneOfDurationOrSelector verifies wait requires duration or selector.
func TestWaitRequireOneOfDurationOrSelector(t *testing.T) {
	def, ok := GetStepDefinition("wait")
	if !ok {
		t.Fatal("expected to find 'wait' definition")
	}

	if len(def.RequireOneOf) == 0 {
		t.Error("wait should have requireOneOf constraint")
		return
	}

	// Should require one of: durationMs or selector
	found := false
	for _, group := range def.RequireOneOf {
		hasDuration := false
		hasSelector := false
		for _, key := range group {
			if key == "durationMs" {
				hasDuration = true
			}
			if key == "selector" {
				hasSelector = true
			}
		}
		if hasDuration && hasSelector {
			found = true
			break
		}
	}

	if !found {
		t.Error("wait's requireOneOf should include [durationMs, selector]")
	}
}
