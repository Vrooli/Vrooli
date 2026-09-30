package control

import (
	"bytes"
	"encoding/json"
	"strings"
)

func extractJSONPayload(output []byte) (json.RawMessage, bool) {
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 {
		return nil, false
	}
	start := bytes.IndexByte(trimmed, '{')
	end := bytes.LastIndexByte(trimmed, '}')
	if start < 0 || end < start {
		return nil, false
	}
	candidate := bytes.TrimSpace(trimmed[start : end+1])
	if !json.Valid(candidate) {
		return nil, false
	}
	return append(json.RawMessage(nil), candidate...), true
}

func boolValue(value any, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "yes", "healthy", "running", "installed":
			return true
		case "false", "no", "stopped", "missing":
			return false
		}
	}
	return fallback
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return strings.Trim(strings.TrimSpace(string(data)), `"`)
	}
}
