package phasekeys

import "strings"

var aliases = map[string]string{
	"unit-test":        "unit",
	"unit_test":        "unit",
	"unittest":         "unit",
	"integration-test": "integration",
	"integration_test": "integration",
	"integrationtest":  "integration",
	"e2e":              "playbooks",
	"business-logic":   "business",
	"business_logic":   "business",
	"struct":           "structure",
	"deps":             "dependencies",
	"perf":             "performance",
	"playbook":         "playbooks",
}

// NormalizeKey trims, lowercases, and applies orchestrator-wide phase aliases.
func NormalizeKey(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "" {
		return ""
	}
	if alias, ok := aliases[key]; ok {
		return alias
	}
	return key
}
