package operatingmode

import (
	"fmt"
	"strings"
)

// validateDeclaredOutput checks a resolved envelope object against a phase's
// declared output schema (L1 extraction validation, reused as the L3 contract
// gate). It walks every declared field by dotted path against the generic
// envelope map — the same MapFieldLookup the guard evaluator uses — so field
// presence, type, enum membership, and numeric/length bounds are enforced from
// data with no mode-specific vocabulary.
//
// It returns two disjoint lists: missing names required-but-absent fields (the
// ladder abstains when non-empty), and violations names present-but-invalid
// fields (wrong type, out-of-enum, out-of-bounds). A field that is optional and
// absent is neither missing nor a violation.
func validateDeclaredOutput(declared *DeclaredOutput, envelope map[string]any) (missing, violations []string) {
	if declared == nil {
		return nil, nil
	}
	lookup := NewMapFieldLookup(envelope)
	walkDeclaredFields(declared.Fields, "", lookup, &missing, &violations)
	return missing, violations
}

// walkDeclaredFields validates one level of declared fields under prefix,
// recursing into any nested Fields. Field names may themselves be dotted paths
// (the on-disk schema uses flat "progress.decision" names), so the effective
// path is prefix + name.
func walkDeclaredFields(fields []OutputField, prefix string, lookup FieldLookup, missing, violations *[]string) {
	for _, field := range fields {
		name := strings.TrimSpace(field.Name)
		if name == "" {
			continue
		}
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		value, present := lookup.Lookup(path)
		switch {
		case !present || value == nil:
			if field.Required {
				*missing = append(*missing, path)
			}
		default:
			if v := violateField(field, path, value); v != "" {
				*violations = append(*violations, v)
			}
		}
		if len(field.Fields) > 0 {
			walkDeclaredFields(field.Fields, path, lookup, missing, violations)
		}
	}
}

// violateField reports the first contract violation for a present field value,
// or "" when the value satisfies the declared type/enum/bounds. Type checking is
// permissive for an empty declared type (treated as "any").
func violateField(field OutputField, path string, value any) string {
	if !typeMatches(field.Type, value) {
		return fmt.Sprintf("%s: expected %s, got %s", path, field.Type, jsonKind(value))
	}
	if len(field.Enum) > 0 {
		matched := false
		for _, member := range field.Enum {
			if scalarEqual(value, member) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Sprintf("%s: %q not in declared enum", path, renderGuardValue(value))
		}
	}
	if field.Minimum != nil || field.Maximum != nil {
		if num, ok := toFloat(value); ok {
			if field.Minimum != nil && num < *field.Minimum {
				return fmt.Sprintf("%s: %v below minimum %v", path, num, *field.Minimum)
			}
			if field.Maximum != nil && num > *field.Maximum {
				return fmt.Sprintf("%s: %v above maximum %v", path, num, *field.Maximum)
			}
		}
	}
	if field.MinLength != nil || field.MaxLength != nil {
		if length, ok := lengthOf(value); ok {
			if field.MinLength != nil && length < *field.MinLength {
				return fmt.Sprintf("%s: length %d below min_length %d", path, length, *field.MinLength)
			}
			if field.MaxLength != nil && length > *field.MaxLength {
				return fmt.Sprintf("%s: length %d above max_length %d", path, length, *field.MaxLength)
			}
		}
	}
	return ""
}

// typeMatches reports whether value matches the declared JSON type. An empty
// declared type accepts any value. Integer accepts a whole-valued number (JSON
// has no separate integer kind after decoding).
func typeMatches(declaredType string, value any) bool {
	switch strings.TrimSpace(declaredType) {
	case "", "any":
		return true
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean", "bool":
		_, ok := value.(bool)
		return ok
	case "number":
		_, ok := toFloat(value)
		return ok
	case "integer":
		num, ok := toFloat(value)
		return ok && num == float64(int64(num))
	case "object":
		_, ok := coerceToMap(value)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	default:
		// Unknown declared type: don't fabricate a violation.
		return true
	}
}

// lengthOf returns the length of a string or array value for MinLength/MaxLength
// checks; ok is false for values that have no meaningful length.
func lengthOf(value any) (int, bool) {
	switch v := value.(type) {
	case string:
		return len(v), true
	case []any:
		return len(v), true
	default:
		return 0, false
	}
}

// jsonKind names the decoded JSON kind of a value for violation messages.
func jsonKind(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64, float32, int, int32, int64:
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "value"
	}
}

// requiredFieldNames returns the dotted paths of every required declared field,
// used to describe an abstain when no envelope was parsed at all.
func requiredFieldNames(declared *DeclaredOutput) []string {
	if declared == nil {
		return nil
	}
	var names []string
	collectRequiredFieldNames(declared.Fields, "", &names)
	return names
}

func collectRequiredFieldNames(fields []OutputField, prefix string, names *[]string) {
	for _, field := range fields {
		name := strings.TrimSpace(field.Name)
		if name == "" {
			continue
		}
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		if field.Required {
			*names = append(*names, path)
		}
		collectRequiredFieldNames(field.Fields, path, names)
	}
}
