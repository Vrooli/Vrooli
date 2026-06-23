package phasekeys

import "strings"

var aliases = map[string]string{
	"unit-test": "unit",
	"unit_test": "unit",
	"unittest":  "unit",
	// The integration-test aliases are retained on purpose: phasekeys is shared
	// with the requirements/coverage classifier (internal/requirements/parsing),
	// which keeps "integration" as a test-classification bucket inferred from
	// `*_integration_test.go` refs. That is independent of the (now-removed)
	// test-genie orchestrator `integration` phase — the alias only maps a string;
	// "integration" simply no longer resolves to a registered phase, so the
	// orchestrator filters it out of any selection.
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
