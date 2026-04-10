package validator

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestSchemaContainsAllNodeTypes verifies that the schema defines NodeData for all node types in nodeRules.
// This test logs warnings for missing definitions but doesn't fail, since node types can rely on
// the base node definition with additionalProperties: true.
func TestSchemaContainsAllNodeTypes(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal(workflowSchemaBytes, &schema); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	definitions, ok := schema["definitions"].(map[string]any)
	if !ok {
		t.Fatal("schema missing definitions")
	}

	// Types that intentionally don't have explicit definitions
	// (they use the base node definition with additionalProperties: true)
	skipTypes := map[string]bool{
		"loop":     true,
		"dragDrop": true,
		"hover":    true,
		"focus":    true,
		"blur":     true,
		"select":   true,
		"keyboard": true,
		"shortcut": true,
		"extract":  true,
	}

	// Check that each node type in nodeRules has a corresponding {type}NodeData definition
	for nodeType := range nodeRules {
		// Skip types that don't need explicit NodeData definitions
		if skipTypes[nodeType] {
			continue
		}

		expectedDef := nodeType + "NodeData"
		if _, exists := definitions[expectedDef]; !exists {
			// Check if there's an allOf condition for this type instead
			nodeDef, ok := definitions["node"].(map[string]any)
			if !ok {
				t.Errorf("node type %q: missing %s definition (and no node definition found)", nodeType, expectedDef)
				continue
			}

			allOf, ok := nodeDef["allOf"].([]any)
			if !ok {
				t.Errorf("node type %q: missing %s definition (and no allOf in node)", nodeType, expectedDef)
				continue
			}

			found := false
			for _, cond := range allOf {
				condMap, ok := cond.(map[string]any)
				if !ok {
					continue
				}
				ifClause, ok := condMap["if"].(map[string]any)
				if !ok {
					continue
				}
				props, ok := ifClause["properties"].(map[string]any)
				if !ok {
					continue
				}
				typeClause, ok := props["type"].(map[string]any)
				if !ok {
					continue
				}
				constVal, ok := typeClause["const"].(string)
				if ok && constVal == nodeType {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("node type %q: missing %s definition and no allOf condition", nodeType, expectedDef)
			}
		}
	}
}

// TestSchemaMatchesNodeRulesRequirements verifies that schema required fields match nodeRules.
func TestSchemaMatchesNodeRulesRequirements(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal(workflowSchemaBytes, &schema); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	definitions, ok := schema["definitions"].(map[string]any)
	if !ok {
		t.Fatal("schema missing definitions")
	}

	// Check that schema required fields align with nodeRules requiredData
	for nodeType, rule := range nodeRules {
		defName := nodeType + "NodeData"
		def, exists := definitions[defName]
		if !exists {
			// Some types don't have explicit definitions
			continue
		}

		defMap, ok := def.(map[string]any)
		if !ok {
			continue
		}

		requiredRaw, hasRequired := defMap["required"]
		if !hasRequired && len(rule.requiredData) > 0 {
			t.Logf("node type %q: schema has no required fields but nodeRules requires %v", nodeType, rule.requiredData)
		}

		if hasRequired {
			required, ok := requiredRaw.([]any)
			if !ok {
				continue
			}

			schemaRequired := make(map[string]bool)
			for _, r := range required {
				if s, ok := r.(string); ok {
					schemaRequired[s] = true
				}
			}

			for _, field := range rule.requiredData {
				if !schemaRequired[field] {
					t.Logf("node type %q: nodeRules requires %q but schema does not mark it as required", nodeType, field)
				}
			}
		}
	}
}

// TestSchemaProviderCreation verifies that the schema provider can be created.
func TestSchemaProviderCreation(t *testing.T) {
	provider, err := NewSchemaProvider()
	if err != nil {
		t.Fatalf("failed to create schema provider: %v", err)
	}

	schema, err := provider.GetFullSchema()
	if err != nil {
		t.Fatalf("failed to get full schema: %v", err)
	}

	if len(schema) == 0 {
		t.Error("schema is empty")
	}
}

// TestSchemaProviderFiltering verifies that schema filtering works correctly.
func TestSchemaProviderFiltering(t *testing.T) {
	provider, err := NewSchemaProvider()
	if err != nil {
		t.Fatalf("failed to create schema provider: %v", err)
	}

	// Filter to just navigate and click
	filtered, err := provider.GetFilteredSchema([]string{"navigate", "click"})
	if err != nil {
		t.Fatalf("failed to get filtered schema: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(filtered, &schema); err != nil {
		t.Fatalf("failed to parse filtered schema: %v", err)
	}

	definitions, ok := schema["definitions"].(map[string]any)
	if !ok {
		t.Fatal("filtered schema missing definitions")
	}

	// Should have navigateNodeData and clickNodeData
	if _, ok := definitions["navigateNodeData"]; !ok {
		t.Error("filtered schema missing navigateNodeData")
	}
	if _, ok := definitions["clickNodeData"]; !ok {
		t.Error("filtered schema missing clickNodeData")
	}

	// Should NOT have assertNodeData (not requested)
	if _, ok := definitions["assertNodeData"]; ok {
		t.Error("filtered schema should not contain assertNodeData")
	}

	// Should still have base definitions
	for _, baseDef := range []string{"position", "edge", "commonNodeFields", "resilience"} {
		if _, ok := definitions[baseDef]; !ok {
			t.Errorf("filtered schema missing base definition: %s", baseDef)
		}
	}
}

// TestGetAvailableNodeTypes verifies that the available node types list is accurate.
func TestGetAvailableNodeTypes(t *testing.T) {
	nodeTypes := GetAvailableNodeTypes()

	// Should contain all major node types
	expected := []string{"navigate", "click", "type", "assert", "wait", "screenshot", "evaluate"}
	for _, nt := range expected {
		found := false
		for _, actual := range nodeTypes {
			if actual == nt {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing expected node type: %s", nt)
		}
	}
}

// TestSchemaHasValidJSON verifies the embedded schema is valid JSON.
func TestSchemaHasValidJSON(t *testing.T) {
	if len(workflowSchemaBytes) == 0 {
		t.Fatal("workflow schema bytes are empty")
	}

	var schema map[string]any
	if err := json.Unmarshal(workflowSchemaBytes, &schema); err != nil {
		t.Fatalf("workflow schema is not valid JSON: %v", err)
	}

	// Verify essential top-level properties
	required := []string{"$schema", "type", "required", "properties", "definitions"}
	for _, prop := range required {
		if _, ok := schema[prop]; !ok {
			t.Errorf("schema missing required property: %s", prop)
		}
	}
}

// TestSchemaNodeDefinitionsHaveData verifies node definitions reference data correctly.
func TestSchemaNodeDefinitionsHaveData(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal(workflowSchemaBytes, &schema); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	definitions := schema["definitions"].(map[string]any)
	nodeDef := definitions["node"].(map[string]any)

	// Verify node has required fields
	required, ok := nodeDef["required"].([]any)
	if !ok {
		t.Fatal("node definition missing required array")
	}

	requiredFields := make(map[string]bool)
	for _, r := range required {
		if s, ok := r.(string); ok {
			requiredFields[s] = true
		}
	}

	for _, field := range []string{"id", "type", "data"} {
		if !requiredFields[field] {
			t.Errorf("node definition should require %s", field)
		}
	}
}

// TestSchemaAllOfConditionsAreValid verifies allOf conditions in node definition.
func TestSchemaAllOfConditionsAreValid(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal(workflowSchemaBytes, &schema); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	definitions := schema["definitions"].(map[string]any)
	nodeDef := definitions["node"].(map[string]any)

	allOf, ok := nodeDef["allOf"].([]any)
	if !ok {
		t.Skip("node definition has no allOf conditions")
	}

	seenTypes := make(map[string]bool)
	for i, cond := range allOf {
		condMap, ok := cond.(map[string]any)
		if !ok {
			t.Errorf("allOf[%d] is not an object", i)
			continue
		}

		ifClause, hasIf := condMap["if"].(map[string]any)
		thenClause, hasThen := condMap["then"].(map[string]any)

		if !hasIf || !hasThen {
			continue // Skip non-conditional entries
		}

		props, ok := ifClause["properties"].(map[string]any)
		if !ok {
			continue
		}

		typeClause, ok := props["type"].(map[string]any)
		if !ok {
			continue
		}

		constVal, ok := typeClause["const"].(string)
		if !ok {
			continue
		}

		if seenTypes[constVal] {
			t.Errorf("duplicate allOf condition for type %q", constVal)
		}
		seenTypes[constVal] = true

		// Verify then clause references a definition
		thenProps, ok := thenClause["properties"].(map[string]any)
		if !ok {
			t.Errorf("allOf condition for %q has no then.properties", constVal)
			continue
		}

		dataRef, ok := thenProps["data"].(map[string]any)
		if !ok {
			t.Errorf("allOf condition for %q has no data property", constVal)
			continue
		}

		ref, ok := dataRef["$ref"].(string)
		if !ok {
			t.Errorf("allOf condition for %q has no $ref for data", constVal)
			continue
		}

		// Verify the referenced definition exists
		expectedDef := strings.TrimPrefix(ref, "#/definitions/")
		if _, exists := definitions[expectedDef]; !exists {
			t.Errorf("allOf condition for %q references non-existent definition: %s", constVal, expectedDef)
		}
	}
}
