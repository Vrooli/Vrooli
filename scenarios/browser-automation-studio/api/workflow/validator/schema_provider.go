package validator

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SchemaProvider provides access to the workflow schema.
type SchemaProvider interface {
	GetFullSchema() (json.RawMessage, error)
	GetFilteredSchema(nodeTypes []string) (json.RawMessage, error)
}

type schemaProvider struct {
	rawSchema json.RawMessage
	parsed    map[string]any
}

// NewSchemaProvider creates a new schema provider using the embedded workflow schema.
func NewSchemaProvider() (SchemaProvider, error) {
	if len(workflowSchemaBytes) == 0 {
		return nil, fmt.Errorf("workflow schema not loaded")
	}

	var parsed map[string]any
	if err := json.Unmarshal(workflowSchemaBytes, &parsed); err != nil {
		return nil, fmt.Errorf("parse workflow schema: %w", err)
	}

	return &schemaProvider{
		rawSchema: workflowSchemaBytes,
		parsed:    parsed,
	}, nil
}

// GetFullSchema returns the complete workflow schema.
func (p *schemaProvider) GetFullSchema() (json.RawMessage, error) {
	return p.rawSchema, nil
}

// GetFilteredSchema returns a schema filtered to include only the specified node types.
// If nodeTypes is empty, returns the full schema.
func (p *schemaProvider) GetFilteredSchema(nodeTypes []string) (json.RawMessage, error) {
	if len(nodeTypes) == 0 {
		return p.rawSchema, nil
	}

	// Create a filtered copy of the schema
	filtered := deepCopyMap(p.parsed)

	definitions, ok := filtered["definitions"].(map[string]any)
	if !ok {
		// No definitions to filter, return as-is
		result, err := json.MarshalIndent(filtered, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal filtered schema: %w", err)
		}
		return result, nil
	}

	// Build set of requested node types
	requested := make(map[string]bool)
	for _, nt := range nodeTypes {
		requested[strings.TrimSpace(nt)] = true
	}

	// Build list of definitions to keep
	// Always keep: position, edge, commonNodeFields, resilience, node (base)
	alwaysKeep := map[string]bool{
		"position":         true,
		"edge":             true,
		"commonNodeFields": true,
		"resilience":       true,
		"node":             true,
	}

	// Filter definitions
	filteredDefs := make(map[string]any)
	for name, def := range definitions {
		// Keep always-required definitions
		if alwaysKeep[name] {
			filteredDefs[name] = def
			continue
		}

		// Keep node data definitions for requested types
		// Pattern: {nodeType}NodeData
		if strings.HasSuffix(name, "NodeData") {
			nodeType := strings.TrimSuffix(name, "NodeData")
			if requested[nodeType] {
				filteredDefs[name] = def
			}
		}
	}
	filtered["definitions"] = filteredDefs

	// Filter the allOf conditions in the node definition to only include requested types
	if nodeDef, ok := filteredDefs["node"].(map[string]any); ok {
		if allOf, ok := nodeDef["allOf"].([]any); ok {
			filteredAllOf := make([]any, 0)
			for _, condition := range allOf {
				condMap, ok := condition.(map[string]any)
				if !ok {
					continue
				}
				ifClause, ok := condMap["if"].(map[string]any)
				if !ok {
					filteredAllOf = append(filteredAllOf, condition)
					continue
				}
				props, ok := ifClause["properties"].(map[string]any)
				if !ok {
					filteredAllOf = append(filteredAllOf, condition)
					continue
				}
				typeClause, ok := props["type"].(map[string]any)
				if !ok {
					filteredAllOf = append(filteredAllOf, condition)
					continue
				}
				constVal, ok := typeClause["const"].(string)
				if !ok {
					filteredAllOf = append(filteredAllOf, condition)
					continue
				}
				// Only include if this type was requested
				if requested[constVal] {
					filteredAllOf = append(filteredAllOf, condition)
				}
			}
			nodeDef["allOf"] = filteredAllOf
		}
	}

	result, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal filtered schema: %w", err)
	}

	return result, nil
}

// GetAvailableNodeTypes returns a list of all node types defined in the schema.
func GetAvailableNodeTypes() []string {
	return []string{
		"navigate",
		"click",
		"type",
		"assert",
		"wait",
		"screenshot",
		"evaluate",
		"subflow",
		"hover",
		"focus",
		"blur",
		"select",
		"keyboard",
		"shortcut",
		"extract",
		"dragDrop",
		"gesture",
		"loop",
	}
}

// deepCopyMap creates a deep copy of a map.
func deepCopyMap(m map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range m {
		switch val := v.(type) {
		case map[string]any:
			result[k] = deepCopyMap(val)
		case []any:
			result[k] = deepCopySlice(val)
		default:
			result[k] = v
		}
	}
	return result
}

// deepCopySlice creates a deep copy of a slice.
func deepCopySlice(s []any) []any {
	result := make([]any, len(s))
	for i, v := range s {
		switch val := v.(type) {
		case map[string]any:
			result[i] = deepCopyMap(val)
		case []any:
			result[i] = deepCopySlice(val)
		default:
			result[i] = v
		}
	}
	return result
}
