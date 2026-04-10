package main

// knownDependencies maps resources to their required dependencies.
// Used by config validation (handleConfigValidate) and setup ordering (handleSetupOrder).
var knownDependencies = map[string][]string{
	"postgis":        {"postgres"},
	"judge0":         {"postgres", "redis"},
	"n8n":            {"postgres"},
	"nextcloud":      {"postgres", "redis"},
	"wikijs":         {"postgres"},
	"erpnext":        {"postgres", "redis"},
	"keycloak":       {"postgres"},
	"home-assistant": {"postgres"},
}
