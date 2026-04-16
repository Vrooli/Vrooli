package render

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ItemLabel(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "(empty)"
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(text)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err == nil {
		for _, key := range []string{"title", "name", "label"} {
			if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}

	return strings.TrimSpace(string(raw))
}

func ConfidencePercent(value float64) string {
	return fmt.Sprintf("%.1f%%", value*100)
}
