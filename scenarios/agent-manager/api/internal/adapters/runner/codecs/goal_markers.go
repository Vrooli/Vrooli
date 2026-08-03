package codecs

import (
	"encoding/json"
	"strings"
)

// declaredGoalStatus applies only the marker names declared by one codec.
// Values remain bounded metadata; transcript bodies are never retained here.
func declaredGoalStatus(line string, markerTypes ...string) (string, bool, bool) {
	var value any
	if json.Unmarshal([]byte(line), &value) != nil {
		return "", false, false
	}
	allowed := map[string]bool{}
	for _, marker := range markerTypes {
		allowed[marker] = true
	}
	var visit func(any) (string, bool, bool)
	visit = func(current any) (string, bool, bool) {
		object, ok := current.(map[string]any)
		if !ok {
			if array, ok := current.([]any); ok {
				for _, child := range array {
					if condition, met, found := visit(child); found {
						return condition, met, true
					}
				}
			}
			return "", false, false
		}
		marker, _ := object["type"].(string)
		if allowed[marker] {
			condition, _ := object["condition"].(string)
			met, _ := object["met"].(bool)
			if strings.TrimSpace(condition) != "" {
				return condition, met, true
			}
		}
		for _, child := range object {
			if condition, met, found := visit(child); found {
				return condition, met, true
			}
		}
		return "", false, false
	}
	return visit(value)
}
