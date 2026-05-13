package operatingmode

import "strings"

func ensurePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	return payload
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}
