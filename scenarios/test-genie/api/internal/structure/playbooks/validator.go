package playbooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"test-genie/internal/shared"
)

// Re-export shared types for convenience.
type (
	FailureClass    = shared.FailureClass
	ObservationType = shared.ObservationType
	Observation     = shared.Observation
	Result          = shared.Result
)

// Re-export constants.
const (
	FailureClassNone             = shared.FailureClassNone
	FailureClassMisconfiguration = shared.FailureClassMisconfiguration

	ObservationSuccess = shared.ObservationSuccess
	ObservationWarning = shared.ObservationWarning
	ObservationInfo    = shared.ObservationInfo
)

// Re-export constructor functions.
var (
	NewSuccessObservation = shared.NewSuccessObservation
	NewWarningObservation = shared.NewWarningObservation
	NewInfoObservation    = shared.NewInfoObservation

	OK          = shared.OK
	OKWithCount = shared.OKWithCount
)

// twoDigitPrefixRE matches directory names starting with two digits and a hyphen.
var twoDigitPrefixRE = regexp.MustCompile(`^[0-9]{2}-`)

// Config holds configuration for playbooks validation.
type Config struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// Enabled controls whether playbooks validation runs.
	// When false, validation is skipped entirely.
	Enabled bool

	// Strict controls whether validation failures block the structure phase.
	// When false (default), issues are reported as warnings.
	// When true, issues are reported as errors and block the phase.
	Strict bool
}

// Validator validates the structure of test/playbooks/ directories.
type Validator interface {
	// Validate checks the playbooks directory structure.
	// Returns a Result with observations (warnings or errors based on config).
	Validate() Result
}

// validator is the default implementation of Validator.
type validator struct {
	config    Config
	logWriter io.Writer
}

// New creates a new playbooks validator.
func New(config Config, logWriter io.Writer) Validator {
	if logWriter == nil {
		logWriter = io.Discard
	}
	return &validator{
		config:    config,
		logWriter: logWriter,
	}
}

// Validate implements Validator.
func (v *validator) Validate() Result {
	if !v.config.Enabled {
		return OK().WithObservations(
			NewInfoObservation("Playbooks validation disabled"),
		)
	}

	playboooksDir := filepath.Join(v.config.ScenarioDir, "test", "playbooks")

	// Check if playbooks directory exists
	if _, err := os.Stat(playboooksDir); os.IsNotExist(err) {
		return OK().WithObservations(
			NewInfoObservation("No test/playbooks/ directory found (optional)"),
		)
	}

	shared.LogInfo(v.logWriter, "Validating playbooks structure...")

	var observations []Observation
	var issues int

	// Validate registry.json
	registryObs, registryIssues := v.validateRegistry(playboooksDir)
	observations = append(observations, registryObs...)
	issues += registryIssues

	// Validate capabilities/ directory structure
	capabilitiesObs, capabilitiesIssues := v.validatePrefixedDir(playboooksDir, "capabilities")
	observations = append(observations, capabilitiesObs...)
	issues += capabilitiesIssues

	// Validate journeys/ directory structure
	journeysObs, journeysIssues := v.validatePrefixedDir(playboooksDir, "journeys")
	observations = append(observations, journeysObs...)
	issues += journeysIssues

	// Validate __subflows/ fixtures
	subflowsObs, subflowsIssues := v.validateSubflows(playboooksDir)
	observations = append(observations, subflowsObs...)
	issues += subflowsIssues

	// Validate __seeds/ directory
	seedsObs, seedsIssues := v.validateSeeds(playboooksDir)
	observations = append(observations, seedsObs...)
	issues += seedsIssues

	// Return result
	if issues > 0 {
		if v.config.Strict {
			return shared.Fail(
				fmt.Errorf("playbooks structure has %d issue(s)", issues),
				FailureClassMisconfiguration,
				"Run 'test-genie registry build' and review warnings above",
			).WithObservations(observations...)
		}
		// Non-strict mode: return success with warnings
		observations = append(observations, NewWarningObservation(
			fmt.Sprintf("Playbooks structure has %d issue(s) (informational)", issues),
		))
		return OK().WithObservations(observations...)
	}

	observations = append(observations, NewSuccessObservation("Playbooks structure valid"))
	shared.LogSuccess(v.logWriter, "Playbooks structure valid")
	return OK().WithObservations(observations...)
}

// validateRegistry checks that registry.json exists and is valid JSON.
func (v *validator) validateRegistry(playbooksDir string) ([]Observation, int) {
	var observations []Observation
	var issues int

	registryPath := filepath.Join(playbooksDir, "registry.json")
	data, err := os.ReadFile(registryPath)
	if os.IsNotExist(err) {
		observations = append(observations, NewWarningObservation(
			"Missing registry.json - run 'test-genie registry build'",
		))
		issues++
		return observations, issues
	}
	if err != nil {
		observations = append(observations, NewWarningObservation(
			fmt.Sprintf("Cannot read registry.json: %v", err),
		))
		issues++
		return observations, issues
	}

	// Validate JSON structure
	var registry struct {
		Note        string `json:"_note"`
		Scenario    string `json:"scenario"`
		GeneratedAt string `json:"generated_at"`
		Playbooks   []struct {
			File         string   `json:"file"`
			Description  string   `json:"description"`
			Order        string   `json:"order"`
			Requirements []string `json:"requirements"`
			Fixtures     []string `json:"fixtures"`
			Reset        string   `json:"reset"`
		} `json:"playbooks"`
	}

	if err := json.Unmarshal(data, &registry); err != nil {
		observations = append(observations, NewWarningObservation(
			fmt.Sprintf("Invalid JSON in registry.json: %v", err),
		))
		issues++
		return observations, issues
	}

	// Check for stale registry (generated_at is null or missing)
	if registry.GeneratedAt == "" || registry.GeneratedAt == "null" {
		observations = append(observations, NewWarningObservation(
			"Registry may be stale (no generated_at) - run 'test-genie registry build'",
		))
		issues++
	}

	observations = append(observations, NewInfoObservation(
		fmt.Sprintf("Registry has %d playbook(s)", len(registry.Playbooks)),
	))

	return observations, issues
}

// validatePrefixedDir checks that directories use two-digit prefixes (e.g., 01-foundation).
func (v *validator) validatePrefixedDir(playbooksDir, dirName string) ([]Observation, int) {
	var observations []Observation
	var issues int

	dirPath := filepath.Join(playbooksDir, dirName)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// Directory not required
		return observations, issues
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		observations = append(observations, NewWarningObservation(
			fmt.Sprintf("Cannot read %s/: %v", dirName, err),
		))
		issues++
		return observations, issues
	}

	var invalidDirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Skip hidden directories and internal directories
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "__") {
			continue
		}
		if !twoDigitPrefixRE.MatchString(name) {
			invalidDirs = append(invalidDirs, name)
		}
	}

	if len(invalidDirs) > 0 {
		observations = append(observations, NewWarningObservation(
			fmt.Sprintf("%s/ has directories without NN- prefix: %s", dirName, strings.Join(invalidDirs, ", ")),
		))
		issues++
	}

	return observations, issues
}

// validateSubflows checks that fixtures in __subflows/ declare fixture_id.
func (v *validator) validateSubflows(playbooksDir string) ([]Observation, int) {
	var observations []Observation
	var issues int

	subflowsDir := filepath.Join(playbooksDir, "__subflows")
	if _, err := os.Stat(subflowsDir); os.IsNotExist(err) {
		// Directory not required
		return observations, issues
	}

	entries, err := os.ReadDir(subflowsDir)
	if err != nil {
		observations = append(observations, NewWarningObservation(
			fmt.Sprintf("Cannot read __subflows/: %v", err),
		))
		issues++
		return observations, issues
	}

	var missingFixtureID []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}

		filePath := filepath.Join(subflowsDir, name)
		data, err := os.ReadFile(filePath)
		if err != nil {
			observations = append(observations, NewWarningObservation(
				fmt.Sprintf("Cannot read %s: %v", name, err),
			))
			issues++
			continue
		}

		var workflow struct {
			Metadata struct {
				FixtureID string `json:"fixture_id"`
			} `json:"metadata"`
		}

		if err := json.Unmarshal(data, &workflow); err != nil {
			observations = append(observations, NewWarningObservation(
				fmt.Sprintf("Invalid JSON in %s: %v", name, err),
			))
			issues++
			continue
		}

		if workflow.Metadata.FixtureID == "" {
			missingFixtureID = append(missingFixtureID, name)
		}
	}

	if len(missingFixtureID) > 0 {
		observations = append(observations, NewWarningObservation(
			fmt.Sprintf("__subflows/ fixtures missing fixture_id: %s", strings.Join(missingFixtureID, ", ")),
		))
		issues++
	}

	return observations, issues
}

// validateSeeds checks that __seeds/ has apply.sh if the directory exists.
func (v *validator) validateSeeds(playbooksDir string) ([]Observation, int) {
	var observations []Observation
	var issues int

	seedsDir := filepath.Join(playbooksDir, "__seeds")
	if _, err := os.Stat(seedsDir); os.IsNotExist(err) {
		// Directory not required
		return observations, issues
	}

	applyPath := filepath.Join(seedsDir, "apply.sh")
	if _, err := os.Stat(applyPath); os.IsNotExist(err) {
		observations = append(observations, NewWarningObservation(
			"__seeds/ directory exists but missing apply.sh",
		))
		issues++
	}

	return observations, issues
}
