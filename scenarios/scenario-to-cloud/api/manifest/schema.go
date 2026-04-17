package manifest

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"scenario-to-cloud/domain"
	"strings"
)

//go:embed schema.json
var schemaJSON []byte

var parsedSchema map[string]interface{}

func init() {
	if err := json.Unmarshal(schemaJSON, &parsedSchema); err != nil {
		panic(fmt.Sprintf("manifest schema.json is invalid: %v", err))
	}
}

// SchemaJSON returns the canonical manifest JSON Schema document.
func SchemaJSON() []byte {
	out := make([]byte, len(schemaJSON))
	copy(out, schemaJSON)
	return out
}

// SchemaMap returns the parsed schema map.
func SchemaMap() map[string]interface{} {
	out := make(map[string]interface{}, len(parsedSchema))
	for k, v := range parsedSchema {
		out[k] = v
	}
	return out
}

// ValidateStructure validates a raw manifest object against schema.json.
func ValidateStructure(raw map[string]interface{}) []domain.ValidationIssue {
	issues := validateNode(parsedSchema, raw, "")
	return stableIssues(issues)
}

func validateNode(schema map[string]interface{}, value interface{}, path string) []domain.ValidationIssue {
	var issues []domain.ValidationIssue

	if expectedType, ok := schema["type"].(string); ok {
		if !matchesType(expectedType, value) {
			issues = append(issues, schemaIssue(path, fmt.Sprintf("expected %s", expectedType), "Use the manifest schema via `scenario-to-cloud manifest schema` for the exact contract."))
			return issues
		}
	}

	if enumVals, ok := schema["enum"].([]interface{}); ok {
		if !matchesEnum(enumVals, value) {
			issues = append(issues, schemaIssue(path, "value must be one of the allowed enum values", "Check the schema enum values and update the manifest field."))
		}
	}

	switch typed := value.(type) {
	case map[string]interface{}:
		issues = append(issues, validateObject(schema, typed, path)...)
	case []interface{}:
		issues = append(issues, validateArray(schema, typed, path)...)
	case float64:
		issues = append(issues, validateNumberConstraints(schema, typed, path)...)
	case string:
		issues = append(issues, validateStringConstraints(schema, typed, path)...)
	}

	return issues
}

func validateObject(schema, value map[string]interface{}, path string) []domain.ValidationIssue {
	var issues []domain.ValidationIssue

	if required, ok := schema["required"].([]interface{}); ok {
		for _, req := range required {
			name, ok := req.(string)
			if !ok {
				continue
			}
			if _, exists := value[name]; !exists {
				issues = append(issues, schemaIssue(joinPath(path, name), "required field is missing", "Add the required field."))
			}
		}
	}

	properties, _ := asMap(schema["properties"])
	additional := schema["additionalProperties"]

	for key, propValue := range value {
		if propSchemaRaw, ok := properties[key]; ok {
			propSchema, ok := asMap(propSchemaRaw)
			if !ok {
				continue
			}
			issues = append(issues, validateNode(propSchema, propValue, joinPath(path, key))...)
			continue
		}

		switch allowed := additional.(type) {
		case bool:
			if !allowed {
				issues = append(issues, schemaIssue(joinPath(path, key), "unknown field is not allowed", "Remove this field or update to a supported field."))
			}
		case map[string]interface{}:
			issues = append(issues, validateNode(allowed, propValue, joinPath(path, key))...)
		case nil:
			// JSON schema default is allowed when unspecified
		default:
			if m, ok := asMap(allowed); ok {
				issues = append(issues, validateNode(m, propValue, joinPath(path, key))...)
			}
		}
	}

	return issues
}

func validateArray(schema map[string]interface{}, value []interface{}, path string) []domain.ValidationIssue {
	itemsSchemaRaw, ok := schema["items"]
	if !ok {
		return nil
	}
	itemsSchema, ok := asMap(itemsSchemaRaw)
	if !ok {
		return nil
	}

	issues := make([]domain.ValidationIssue, 0)
	for i, item := range value {
		issues = append(issues, validateNode(itemsSchema, item, fmt.Sprintf("%s[%d]", path, i))...)
	}
	return issues
}

func validateNumberConstraints(schema map[string]interface{}, value float64, path string) []domain.ValidationIssue {
	var issues []domain.ValidationIssue
	if min, ok := schema["minimum"].(float64); ok && value < min {
		issues = append(issues, schemaIssue(path, fmt.Sprintf("must be >= %.0f", min), "Set the number within the allowed range."))
	}
	if max, ok := schema["maximum"].(float64); ok && value > max {
		issues = append(issues, schemaIssue(path, fmt.Sprintf("must be <= %.0f", max), "Set the number within the allowed range."))
	}
	if expectedType, ok := schema["type"].(string); ok && expectedType == "integer" {
		if math.Trunc(value) != value {
			issues = append(issues, schemaIssue(path, "must be an integer", "Use an integer value."))
		}
	}
	return issues
}

func validateStringConstraints(schema map[string]interface{}, value, path string) []domain.ValidationIssue {
	var issues []domain.ValidationIssue
	if minLength, ok := schema["minLength"].(float64); ok && len(value) < int(minLength) {
		issues = append(issues, schemaIssue(path, fmt.Sprintf("must be at least %.0f characters", minLength), "Set a non-empty value that satisfies the schema constraint."))
	}
	return issues
}

func schemaIssue(path, message, hint string) domain.ValidationIssue {
	return domain.ValidationIssue{
		Path:     strings.TrimPrefix(path, "."),
		Message:  message,
		Hint:     hint,
		Severity: domain.SeverityError,
	}
}

func joinPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func matchesType(expectedType string, value interface{}) bool {
	switch expectedType {
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "number":
		_, ok := value.(float64)
		return ok
	case "integer":
		n, ok := value.(float64)
		return ok && math.Trunc(n) == n
	default:
		return true
	}
}

func matchesEnum(enumValues []interface{}, value interface{}) bool {
	for _, v := range enumValues {
		if fmt.Sprintf("%v", v) == fmt.Sprintf("%v", value) {
			return true
		}
	}
	return false
}

func asMap(v interface{}) (map[string]interface{}, bool) {
	m, ok := v.(map[string]interface{})
	return m, ok
}

// ValidateRaw validates raw manifest data in two passes:
// 1) schema structural validation
// 2) semantic/runtime validation + normalization
func ValidateRaw(raw map[string]interface{}) (domain.CloudManifest, []domain.ValidationIssue, error) {
	structural := ValidateStructure(raw)

	blob, err := json.Marshal(raw)
	if err != nil {
		return domain.CloudManifest{}, structural, fmt.Errorf("marshal raw manifest: %w", err)
	}

	var parsed domain.CloudManifest
	if err := json.Unmarshal(blob, &parsed); err != nil {
		issues := append(structural, domain.ValidationIssue{
			Path:     "",
			Message:  "manifest payload cannot be decoded",
			Hint:     err.Error(),
			Severity: domain.SeverityError,
		})
		return domain.CloudManifest{}, stableIssues(issues), nil
	}

	normalized, semantic := ValidateAndNormalize(parsed)
	all := append(structural, semantic...)
	return normalized, stableIssues(all), nil
}
