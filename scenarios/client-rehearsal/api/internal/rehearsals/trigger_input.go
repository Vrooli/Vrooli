package rehearsals

import (
	"encoding/json"
	"fmt"
)

const (
	maxTriggerInputBytes  = 64 * 1024
	maxTriggerInputDepth  = 16
	maxTriggerInputValues = 4096
)

// normalizeTriggerInput takes a per-run event object, applies the same
// secret-shaped-key replacement used for workflow imports, and enforces a
// bounded JSON tree. The caller must not persist the returned map.
func normalizeTriggerInput(input map[string]any) (map[string]any, uint32, error) {
	if input == nil {
		return nil, 0, nil
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, 0, ErrInvalid{Field: "trigger_input", Reason: "must be a valid JSON object"}
	}
	if len(raw) > maxTriggerInputBytes {
		return nil, 0, ErrInvalid{Field: "trigger_input", Reason: "must not exceed 64 KiB"}
	}
	var normalized map[string]any
	if err := json.Unmarshal(raw, &normalized); err != nil || normalized == nil {
		return nil, 0, ErrInvalid{Field: "trigger_input", Reason: "must be a JSON object"}
	}
	count := 0
	if err := validateTriggerInputValue(normalized, 0, &count); err != nil {
		return nil, 0, err
	}
	replacements := sanitize(normalized)
	return normalized, replacements, nil
}

func validateTriggerInputValue(value any, depth int, count *int) error {
	*count++
	if *count > maxTriggerInputValues {
		return ErrInvalid{Field: "trigger_input", Reason: "must contain no more than 4096 values"}
	}
	switch typed := value.(type) {
	case map[string]any:
		if depth >= maxTriggerInputDepth {
			return ErrInvalid{Field: "trigger_input", Reason: "must not exceed 16 object/array levels"}
		}
		for key, child := range typed {
			if !safeSyntheticFieldName(key) {
				return ErrInvalid{Field: "trigger_input", Reason: "contains an unsafe object key"}
			}
			if err := validateTriggerInputValue(child, depth+1, count); err != nil {
				return err
			}
		}
	case []any:
		if depth >= maxTriggerInputDepth {
			return ErrInvalid{Field: "trigger_input", Reason: "must not exceed 16 object/array levels"}
		}
		for _, child := range typed {
			if err := validateTriggerInputValue(child, depth+1, count); err != nil {
				return err
			}
		}
	case nil, string, bool, float64:
		return nil
	default:
		return ErrInvalid{Field: "trigger_input", Reason: fmt.Sprintf("contains unsupported JSON value type %T", value)}
	}
	return nil
}
