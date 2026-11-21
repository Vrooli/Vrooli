package manifestschema

import (
	auditrules "structure-health/internal/packs/auditrules"
)

// Violation aliases the shared rule-framework type, matching the other
// config-pack rules.
type Violation = auditrules.Violation

func newViolation(filePath string, line int, message string) Violation {
	if line <= 0 {
		line = 1
	}
	return Violation{
		Type:           "config_manifest_schema",
		Severity:       "high",
		Title:          "Scenario manifest schema violation",
		Description:    message,
		FilePath:       filePath,
		LineNumber:     line,
		Recommendation: "Repair .vrooli/service.json against .vrooli/schemas/service.schema.json, or correct the schema if it no longer describes the implemented contract.",
		Standard:       "configuration-v1",
	}
}
