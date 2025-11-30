// Package validation provides centralized input validation for the landing-manager API.
// All validation patterns and functions are defined here to ensure consistency
// and avoid duplication across handlers.
package validation

import (
	"fmt"
	"regexp"
	"strings"
)

// Validation patterns - unified across all handlers
var (
	// TemplateIDPattern validates template identifiers (1-64 chars, lowercase alphanumeric with hyphens/underscores)
	TemplateIDPattern = regexp.MustCompile(`^[a-z0-9_-]{1,64}$`)

	// ScenarioNamePattern validates human-readable scenario names (1-128 chars, allows spaces)
	ScenarioNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_\s-]{1,128}$`)

	// ScenarioSlugPattern validates URL-safe scenario slugs (1-64 chars, lowercase)
	ScenarioSlugPattern = regexp.MustCompile(`^[a-z0-9_-]{1,64}$`)

	// ScenarioIDPattern validates scenario identifiers - unified with slug pattern for consistency
	ScenarioIDPattern = regexp.MustCompile(`^[a-z0-9_-]{1,64}$`)

	// PersonaIDPattern validates persona identifiers (1-64 chars, lowercase alphanumeric with hyphens/underscores)
	PersonaIDPattern = regexp.MustCompile(`^[a-z0-9_-]{1,64}$`)
)

// ValidateTemplateID ensures the template ID is safe and well-formed
func ValidateTemplateID(id string) error {
	if id == "" {
		return fmt.Errorf("template_id is required")
	}
	if !TemplateIDPattern.MatchString(id) {
		return fmt.Errorf("invalid template_id format: must contain only lowercase letters, numbers, hyphens, and underscores (1-64 chars)")
	}
	return nil
}

// ValidateScenarioName validates scenario name input for human-readable display
func ValidateScenarioName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > 128 {
		return fmt.Errorf("name too long (max 128 characters)")
	}
	if !ScenarioNamePattern.MatchString(name) {
		return fmt.Errorf("invalid name: must contain only letters, numbers, spaces, hyphens, and underscores")
	}
	return nil
}

// ValidateScenarioSlug validates URL-safe scenario slug input
func ValidateScenarioSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug is required")
	}
	if !ScenarioSlugPattern.MatchString(slug) {
		return fmt.Errorf("invalid slug: must contain only lowercase letters, numbers, hyphens, and underscores (1-64 chars)")
	}
	return nil
}

// ValidateScenarioID ensures the scenario ID is safe for filesystem and command usage
func ValidateScenarioID(scenarioID string) error {
	if scenarioID == "" {
		return fmt.Errorf("scenario_id is required")
	}
	if !ScenarioIDPattern.MatchString(scenarioID) {
		return fmt.Errorf("invalid scenario_id: must contain only lowercase letters, numbers, hyphens, and underscores (1-64 chars)")
	}
	// Additional check: prevent hidden files and parent directory references
	if strings.HasPrefix(scenarioID, ".") || strings.Contains(scenarioID, "..") {
		return fmt.Errorf("invalid scenario_id: cannot start with . or contain ..")
	}
	return nil
}

// ValidatePersonaID validates persona ID input (optional field)
func ValidatePersonaID(id string) error {
	if id == "" {
		return nil // persona_id is optional
	}
	if !PersonaIDPattern.MatchString(id) {
		return fmt.Errorf("invalid persona_id format: must contain only lowercase letters, numbers, hyphens, and underscores (1-64 chars)")
	}
	return nil
}

// ValidateBrief validates the customization brief content
func ValidateBrief(brief string) error {
	if len(brief) > 10000 {
		return fmt.Errorf("brief too long (max 10000 characters)")
	}
	return nil
}

// ValidateTailParam validates the log tail parameter
func ValidateTailParam(tail string) (string, bool) {
	if tail == "" {
		return "50", true
	}
	// Must be a positive integer within reasonable bounds
	var n int
	if _, err := fmt.Sscanf(tail, "%d", &n); err != nil || n < 1 || n > 10000 {
		return "50", false // return default but indicate invalid input
	}
	return tail, true
}
