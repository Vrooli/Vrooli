package onboard

import "strings"

// setupPresetEnvironments is the Bridge-side expansion seam. APIs and durable
// policy records carry only the preset name; only the final bootstrap argument
// builder may turn that name into the legacy node setup environment flag.
var setupPresetEnvironments = map[string]string{
	"development":        "development",
	"production":         "production",
	"minimal":            "minimal",
	"managed-connection": "minimal",
	"presence":           "minimal",
	"deployment-target":  "production",
	"production-runtime": "production",
	"development-runner": "development",
	"custom":             "development",
}

func knownSetupPreset(name string) bool {
	_, ok := setupPresetEnvironments[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

func environmentForSetupPreset(name string) (string, bool) {
	environment, ok := setupPresetEnvironments[strings.ToLower(strings.TrimSpace(name))]
	return environment, ok
}
