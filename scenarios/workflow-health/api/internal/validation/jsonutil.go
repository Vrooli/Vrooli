package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func readWorkflowDoc(root, relPath string) (map[string]any, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func hasLegacyReset(root, relPath string) bool {
	doc, err := readWorkflowDoc(root, relPath)
	if err != nil {
		return false
	}
	return strings.EqualFold(getString(doc, "metadata", "labels", "reset"), "database") ||
		strings.EqualFold(getString(doc, "metadata", "reset"), "database")
}

func hasExplicitConfirmation(root, relPath string) bool {
	doc, err := readWorkflowDoc(root, relPath)
	if err != nil {
		return false
	}
	value := strings.ToLower(strings.TrimSpace(getString(doc, "metadata", "labels", "requires_confirmation")))
	return value == "true" || value == "yes"
}

func hasRoutedIsolation(root, relPath string) bool {
	doc, err := readWorkflowDoc(root, relPath)
	if err != nil {
		return false
	}
	value := strings.ToLower(strings.TrimSpace(getString(doc, "metadata", "labels", "routed_isolation")))
	return value == "true" || value == "yes" || value == "routed"
}

func setNestedString(m map[string]any, value string, path ...string) {
	if len(path) == 0 {
		return
	}
	current := m
	for _, key := range path[:len(path)-1] {
		next, ok := current[key].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[key] = next
		}
		current = next
	}
	current[path[len(path)-1]] = value
}

func getString(m map[string]any, path ...string) string {
	var current any = m
	for _, key := range path {
		next, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = next[key]
	}
	value, ok := current.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
