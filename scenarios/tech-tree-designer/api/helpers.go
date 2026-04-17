package main

import (
	"encoding/json"
	"strings"
)

func normalizeStageType(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return "foundation"
	}
	if mapped, ok := stageTypeAliases[trimmed]; ok {
		return mapped
	}
	return strings.ReplaceAll(trimmed, " ", "_")
}

func encodeExamplesArray(values []string) ([]byte, error) {
	if len(values) == 0 {
		return []byte("[]"), nil
	}
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		text := strings.TrimSpace(value)
		if text == "" {
			continue
		}
		trimmed = append(trimmed, text)
	}
	if len(trimmed) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(trimmed)
}

func prettifyStageType(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "Stage"
	}
	parts := strings.Split(strings.ReplaceAll(trimmed, "-", "_"), "_")
	for index, part := range parts {
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		parts[index] = strings.ToUpper(lower[:1]) + lower[1:]
	}
	return strings.Join(parts, " ")
}
