package util

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// executablePath is overridable in tests to control os.Executable behavior.
var executablePath = os.Executable

// ============================================================================
// Scenario Location Decisions
// ============================================================================
// These types and functions determine where a scenario exists and what
// lifecycle behaviors apply based on its location.

// Location constants define the two environments where scenarios can exist.
// Staging: Generated scenarios awaiting promotion (in landing-manager/generated/)
// Production: Promoted scenarios ready for independent operation (in /scenarios/)
const (
	LocationStaging    = "staging"
	LocationProduction = "production"
)

// ScenarioLocation represents where a scenario is located and provides
// context about its lifecycle state for routing decisions.
type ScenarioLocation struct {
	Path     string
	Location string // LocationStaging or LocationProduction
	Found    bool
}

// IsStaging returns true if the scenario is in the staging area.
// Decision: Staging scenarios are in landing-manager/generated/ and may need
// explicit path arguments for lifecycle commands.
func (loc ScenarioLocation) IsStaging() bool {
	return loc.Found && loc.Location == LocationStaging
}

// IsProduction returns true if the scenario is in the production area.
// Decision: Production scenarios are in /scenarios/ and use standard CLI commands.
func (loc ScenarioLocation) IsProduction() bool {
	return loc.Found && loc.Location == LocationProduction
}

// RequiresPathArg determines if lifecycle commands need --path argument.
// Decision: Staging scenarios are not in the standard scenarios/ directory,
// so they need explicit path hints. Production scenarios don't.
func (loc ScenarioLocation) RequiresPathArg() bool {
	return loc.IsStaging()
}

// GetVrooliRoot returns the Vrooli root directory path.
// Decision Priority:
//  1. VROOLI_ROOT environment variable (explicit override)
//  2. Default: ~/Vrooli (standard installation location)
func GetVrooliRoot() string {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return root
	}

	if scenarioRoot, err := ResolveScenarioRoot(); err == nil {
		if candidate := strings.TrimSpace(GetVrooliRootFromScenario(scenarioRoot)); candidate != "" && candidate != string(filepath.Separator) {
			return candidate
		}
	}

	if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
		return filepath.Join(home, "Vrooli")
	}

	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}

	return "."
}

// ResolveScenarioPath finds a scenario in staging or production.
// Decision Priority:
//  1. Check staging first (landing-manager/generated/{id})
//  2. Check production second (/scenarios/{id})
//  3. Return not-found if neither exists
//
// This ordering ensures that recently generated scenarios are found first,
// which is the common case during development and testing workflows.
func ResolveScenarioPath(scenarioID string) ScenarioLocation {
	vrooliRoot := GetVrooliRoot()

	// Decision: Check staging first - recently generated scenarios are more likely
	stagingPath := filepath.Join(vrooliRoot, "scenarios", "landing-manager", "generated", scenarioID)
	if _, err := os.Stat(stagingPath); err == nil {
		return ScenarioLocation{Path: stagingPath, Location: LocationStaging, Found: true}
	}

	// Decision: Fall back to production for promoted scenarios
	productionPath := filepath.Join(vrooliRoot, "scenarios", scenarioID)
	if _, err := os.Stat(productionPath); err == nil {
		return ScenarioLocation{Path: productionPath, Location: LocationProduction, Found: true}
	}

	return ScenarioLocation{Found: false}
}

// StagingPath returns the expected staging path for a scenario ID.
// Use this to construct paths without resolving location.
func StagingPath(scenarioID string) string {
	return filepath.Join(GetVrooliRoot(), "scenarios", "landing-manager", "generated", scenarioID)
}

// ProductionPath returns the expected production path for a scenario ID.
// Use this to construct paths without resolving location.
func ProductionPath(scenarioID string) string {
	return filepath.Join(GetVrooliRoot(), "scenarios", scenarioID)
}

// ============================================================================
// CLI Output Interpretation Decisions
// ============================================================================
// These functions classify CLI command output to determine what happened.

// scenarioNotFoundPatterns capture CLI output that specifically indicates a scenario is missing.
var scenarioNotFoundPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)scenario[^\n]*not found`),
	regexp.MustCompile(`(?i)no such scenario`),
	regexp.MustCompile(`(?i)no lifecycle log found`),
	regexp.MustCompile(`(?i)scenario[^\n]*does not exist`),
}

// IsScenarioNotFound checks if a command error indicates a missing scenario.
// Decision: Returns true if the output contains any known "not found" indicator.
// This is used to distinguish "not found" errors from other failure modes.
func IsScenarioNotFound(output string) bool {
	for _, pattern := range scenarioNotFoundPatterns {
		if pattern.MatchString(output) {
			return true
		}
	}
	return false
}

// ============================================================================
// Output Sanitization Decisions
// ============================================================================
// These functions clean up output before exposing it to clients.

// SensitivePathPatterns are regex patterns for paths that should be sanitized.
var SensitivePathPatterns = []*regexp.Regexp{
	regexp.MustCompile(`/home/[^/\s]+`),
	regexp.MustCompile(`/Users/[^/\s]+`),
}

// SanitizeCommandOutput removes sensitive information from command output.
// Decision: Replaces home directory paths with generic placeholders to avoid
// leaking usernames and system structure to API clients.
func SanitizeCommandOutput(output string) string {
	for _, pattern := range SensitivePathPatterns {
		output = pattern.ReplaceAllString(output, "/home/user")
	}
	return output
}

// ============================================================================
// Path Resolution Decisions
// ============================================================================
// These functions determine filesystem locations for various components.

// ResolveScenarioRoot finds the landing-manager scenario root directory.
// Decision: Assumes binary resides in /scenarios/landing-manager/api,
// so the scenario root is one directory up from the executable.
func ResolveScenarioRoot() (string, error) {
	execPath, err := executablePath()
	if err != nil {
		return "", err
	}
	execDir := filepath.Dir(execPath)
	return filepath.Dir(execDir), nil
}

// GetVrooliRootFromScenario derives Vrooli root from scenario root.
// Decision: Assumes standard directory structure where scenario is
// two levels below Vrooli root (/Vrooli/scenarios/{name}).
func GetVrooliRootFromScenario(scenarioRoot string) string {
	return filepath.Dir(filepath.Dir(scenarioRoot))
}

// ResolvePackagesDir finds the absolute path to the Vrooli packages directory.
func ResolvePackagesDir() (string, error) {
	scenarioRoot, err := ResolveScenarioRoot()
	if err != nil {
		return "", err
	}
	vrooliRoot := GetVrooliRootFromScenario(scenarioRoot)
	return filepath.Join(vrooliRoot, "packages"), nil
}

// GenerationRoot resolves the base directory for generated scenarios.
// Decision Priority:
//  1. GEN_OUTPUT_DIR environment variable (testing/CI override)
//  2. Default: {scenario_root}/generated (standard location)
func GenerationRoot() (string, error) {
	if override := strings.TrimSpace(os.Getenv("GEN_OUTPUT_DIR")); override != "" {
		return filepath.Abs(override)
	}

	scenarioRoot, err := ResolveScenarioRoot()
	if err != nil {
		return "", err
	}
	defaultRoot := filepath.Join(scenarioRoot, "generated")
	return filepath.Abs(defaultRoot)
}

// ResolveGenerationPath returns an absolute path for the generated scenario.
func ResolveGenerationPath(slug string) (string, error) {
	root, err := GenerationRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, slug), nil
}
