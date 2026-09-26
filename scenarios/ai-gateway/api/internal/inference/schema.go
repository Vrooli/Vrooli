package inference

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
)

const (
	MaxSchemaBytes = 32 * 1024
	MaxSchemaDepth = 32
)

// SchemaError identifies the JSON Schema construct that made a request
// unenforceable. Callers can surface Construct without parsing prose.
type SchemaError struct {
	Construct string
	Message   string
}

func (e *SchemaError) Error() string {
	if e == nil {
		return ""
	}
	if e.Construct == "" {
		return e.Message
	}
	return fmt.Sprintf("unsupported schema construct %q: %s", e.Construct, e.Message)
}

var supportedSchemaKeywords = map[string]struct{}{
	"$id": {}, "$schema": {}, "title": {}, "description": {},
	"type": {}, "enum": {}, "const": {}, "required": {},
	"properties": {}, "items": {}, "pattern": {}, "minimum": {},
	"maximum": {},
}

// SchemaGate accepts only the subset that the inference backends and the
// local validator can enforce. Descriptions are accepted as metadata but are
// never used as instruction; instruction is a separate RPC field.
type SchemaGate struct{}

func (SchemaGate) Parse(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, &SchemaError{Construct: "schema_json", Message: "schema_json is required"}
	}
	if len(raw) > MaxSchemaBytes {
		return nil, &SchemaError{Construct: "schema_json", Message: fmt.Sprintf("schema exceeds %d bytes", MaxSchemaBytes)}
	}
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, &SchemaError{Construct: "schema_json", Message: fmt.Sprintf("schema must be valid JSON: %v", err)}
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return nil, &SchemaError{Construct: "schema_json", Message: "schema must contain one JSON value"}
	}
	root, ok := value.(map[string]any)
	if !ok {
		return nil, &SchemaError{Construct: "schema", Message: "schema root must be an object"}
	}
	if err := validateSchemaNode(root, 0); err != nil {
		return nil, err
	}
	return root, nil
}

func validateSchemaNode(node map[string]any, depth int) error {
	if depth > MaxSchemaDepth {
		return &SchemaError{Construct: "depth", Message: fmt.Sprintf("schema exceeds maximum depth %d", MaxSchemaDepth)}
	}
	for key := range node {
		if _, ok := supportedSchemaKeywords[key]; !ok {
			return &SchemaError{Construct: key, Message: "the construct is outside the enforceable subset"}
		}
	}
	if raw, ok := node["type"]; ok {
		typeName, ok := raw.(string)
		if !ok {
			return &SchemaError{Construct: "type", Message: "type must be one string"}
		}
		switch typeName {
		case "object", "array", "string", "number", "integer", "boolean", "null":
		default:
			return &SchemaError{Construct: "type", Message: fmt.Sprintf("type %q is unsupported", typeName)}
		}
	}
	if raw, ok := node["required"]; ok {
		items, ok := raw.([]any)
		if !ok {
			return &SchemaError{Construct: "required", Message: "required must be an array of strings"}
		}
		for _, item := range items {
			name, ok := item.(string)
			if !ok || strings.TrimSpace(name) == "" {
				return &SchemaError{Construct: "required", Message: "required entries must be non-empty strings"}
			}
		}
	}
	if raw, ok := node["properties"]; ok {
		properties, ok := raw.(map[string]any)
		if !ok {
			return &SchemaError{Construct: "properties", Message: "properties must be an object of schemas"}
		}
		for name, child := range properties {
			childSchema, ok := child.(map[string]any)
			if !ok {
				return &SchemaError{Construct: "properties", Message: fmt.Sprintf("property %q must be a schema object", name)}
			}
			if err := validateSchemaNode(childSchema, depth+1); err != nil {
				return err
			}
		}
	}
	if raw, ok := node["items"]; ok {
		items, ok := raw.(map[string]any)
		if !ok {
			return &SchemaError{Construct: "items", Message: "items must be a schema object"}
		}
		if err := validateSchemaNode(items, depth+1); err != nil {
			return err
		}
	}
	if raw, ok := node["pattern"]; ok {
		pattern, ok := raw.(string)
		if !ok {
			return &SchemaError{Construct: "pattern", Message: "pattern must be a string"}
		}
		if _, err := regexp.Compile(pattern); err != nil {
			return &SchemaError{Construct: "pattern", Message: fmt.Sprintf("pattern is invalid: %v", err)}
		}
	}
	for _, key := range []string{"minimum", "maximum"} {
		if raw, ok := node[key]; ok {
			if _, ok := numberValue(raw); !ok {
				return &SchemaError{Construct: key, Message: fmt.Sprintf("%s must be a number", key)}
			}
		}
	}
	return nil
}

func validateValue(value any, schema map[string]any, path string) error {
	if raw, ok := schema["enum"]; ok {
		values, ok := raw.([]any)
		if !ok {
			return fmt.Errorf("%s: enum must be an array", path)
		}
		matched := false
		for _, candidate := range values {
			if jsonEqual(value, candidate) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("%s: value is not one of enum members", path)
		}
	}
	if constant, ok := schema["const"]; ok && !jsonEqual(value, constant) {
		return fmt.Errorf("%s: value does not equal const", path)
	}
	if typeName, ok := schema["type"].(string); ok && !matchesType(value, typeName) {
		return fmt.Errorf("%s: value does not match type %q", path, typeName)
	}
	switch typed := value.(type) {
	case map[string]any:
		// Assertions are checked rather than forced: ValidateJSON is exported,
		// so a schema that never passed SchemaGate.Parse can reach here and
		// must produce an error rather than a panic.
		if required, ok := schema["required"].([]any); ok {
			for _, name := range required {
				key, ok := name.(string)
				if !ok {
					return fmt.Errorf("%s: required entries must be strings", path)
				}
				if _, exists := typed[key]; !exists {
					return fmt.Errorf("%s: required property %q is missing", path, key)
				}
			}
		}
		if properties, ok := schema["properties"].(map[string]any); ok {
			for name, child := range properties {
				childValue, exists := typed[name]
				if !exists {
					continue
				}
				childSchema, ok := child.(map[string]any)
				if !ok {
					return fmt.Errorf("%s.%s: property schema must be an object", path, name)
				}
				if err := validateValue(childValue, childSchema, path+"."+name); err != nil {
					return err
				}
			}
		}
	case []any:
		if itemSchema, ok := schema["items"].(map[string]any); ok {
			for index, item := range typed {
				if err := validateValue(item, itemSchema, fmt.Sprintf("%s[%d]", path, index)); err != nil {
					return err
				}
			}
		}
	case string:
		if pattern, ok := schema["pattern"].(string); ok {
			matched, err := regexp.MatchString(pattern, typed)
			if err != nil || !matched {
				return fmt.Errorf("%s: value does not match pattern", path)
			}
		}
	case json.Number:
		number, _ := typed.Float64()
		if raw, ok := schema["minimum"]; ok {
			minimum, _ := numberValue(raw)
			if number < minimum {
				return fmt.Errorf("%s: value is below minimum", path)
			}
		}
		if raw, ok := schema["maximum"]; ok {
			maximum, _ := numberValue(raw)
			if number > maximum {
				return fmt.Errorf("%s: value is above maximum", path)
			}
		}
	}
	return nil
}

func ValidateJSON(schema map[string]any, raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("value is not valid JSON: %w", err)
	}
	return validateValue(value, schema, "$")
}

func matchesType(value any, typeName string) bool {
	switch typeName {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := numberValue(value)
		return ok
	case "integer":
		number, ok := numberValue(value)
		return ok && number == math.Trunc(number)
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "null":
		return value == nil
	default:
		return true
	}
}

func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case float64:
		return typed, true
	default:
		return 0, false
	}
}

func jsonEqual(left, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}
