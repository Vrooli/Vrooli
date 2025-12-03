package phases

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/uismoke"
)

// UISmokeModeNative enables the native Go implementation of UI smoke tests.
// Defaults to true. Set TEST_GENIE_UI_SMOKE_NATIVE=false to use legacy cached results.
var UISmokeModeNative = os.Getenv("TEST_GENIE_UI_SMOKE_NATIVE") != "false"

var (
	// standardStructureDirs defines directories required for a well-formed scenario.
	standardStructureDirs = []string{
		"api",
		"cli",
		"docs",
		"requirements",
		"test",
		"ui",
	}
	// standardStructureFiles defines files required for a well-formed scenario.
	standardStructureFiles = []string{
		filepath.Join("api", "main.go"),
		filepath.Join("cli", "install.sh"),
		filepath.Join(".vrooli", "service.json"),
		filepath.Join(".vrooli", "testing.json"),
		"README.md",
		"PRD.md",
		"Makefile",
	}
	jsonValidationSkipDirs = map[string]struct{}{
		".git":         {},
		"node_modules": {},
		"dist":         {},
		"build":        {},
		"coverage":     {},
		"artifacts":    {},
	}
)

// runStructurePhase validates the essential scenario layout without shelling out to bash scripts.
func runStructurePhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if err := ctx.Err(); err != nil {
		return RunReport{Err: err, FailureClassification: FailureClassSystem}
	}

	logPhaseInfo(logWriter, "Starting structure validation for %s", env.ScenarioName)

	expectations, err := loadStructureExpectations(env.ScenarioDir)
	if err != nil {
		logPhaseError(logWriter, "Failed to load structure expectations: %v", err)
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Fix .vrooli/testing.json so structure requirements can be parsed.",
		}
	}

	requiredDirs, requiredFiles := resolveStructureRequirements(env.ScenarioName, expectations)
	var observations []Observation

	// Section: Scenario Structure
	observations = append(observations, NewSectionObservation("🏗️", "Validating scenario structure..."))

	// Section: Directories
	observations = append(observations, NewSectionObservation("🔍", "Checking required directories..."))
	logPhaseInfo(logWriter, "Checking required directories...")
	if report := validateStructureDirs(env.ScenarioDir, requiredDirs, logWriter); report.Err != nil {
		return report
	}
	logPhaseSuccess(logWriter, "All required directories present (%d)", len(requiredDirs))
	observations = append(observations, NewSuccessObservation(fmt.Sprintf("All required directories present (%d checked)", len(requiredDirs))))

	// Section: Files
	observations = append(observations, NewSectionObservation("🔍", "Checking required files..."))
	logPhaseInfo(logWriter, "Checking required files...")
	if report := validateStructureFiles(env.ScenarioDir, requiredFiles, logWriter); report.Err != nil {
		return report
	}
	logPhaseSuccess(logWriter, "All required files present (%d)", len(requiredFiles))
	observations = append(observations, NewSuccessObservation(fmt.Sprintf("All required files present (%d checked)", len(requiredFiles))))

	// Section: Service Manifest
	observations = append(observations, NewSectionObservation("📋", "Validating service manifest..."))
	logPhaseInfo(logWriter, "Validating service manifest...")
	manifestPath := filepath.Join(env.ScenarioDir, ".vrooli", "service.json")
	if result := validateServiceManifest(manifestPath, env.ScenarioName, logWriter, expectations); result.Err != nil {
		return result
	}
	logPhaseSuccess(logWriter, "service.json validated")
	observations = append(observations, NewSuccessObservation("service.json name matches scenario directory"))
	observations = append(observations, NewSuccessObservation("Health checks defined"))

	// Section: JSON Validation
	if expectations.ValidateJSONFiles {
		observations = append(observations, NewSectionObservation("📄", "Validating JSON files..."))
		logPhaseInfo(logWriter, "Validating JSON files...")
		jsonCount, invalidFiles, err := scanScenarioJSON(env.ScenarioDir)
		if err != nil {
			logPhaseError(logWriter, "JSON scan failed: %v", err)
			return RunReport{
				Err:                   err,
				FailureClassification: FailureClassSystem,
				Remediation:           "Ensure JSON files under the scenario tree are readable.",
				Observations:          observations,
			}
		}
		if len(invalidFiles) > 0 {
			logPhaseError(logWriter, "Invalid JSON files found: %s", strings.Join(invalidFiles, ", "))
			return RunReport{
				Err:                   fmt.Errorf("invalid JSON detected: %s", strings.Join(invalidFiles, ", ")),
				FailureClassification: FailureClassMisconfiguration,
				Remediation:           "Fix the malformed JSON files listed in the error message.",
				Observations:          observations,
			}
		}
		logPhaseSuccess(logWriter, "All JSON files valid (%d)", jsonCount)
		observations = append(observations, NewSuccessObservation(fmt.Sprintf("All JSON files are valid (%d checked)", jsonCount)))
	}

	// Section: UI Smoke Test
	observations = append(observations, NewSectionObservation("🌐", "Running UI smoke test..."))
	logPhaseInfo(logWriter, "Checking UI smoke telemetry...")
	smokeResult := enforceUISmokeTelemetry(ctx, env, logWriter)
	if smokeResult.failure != nil {
		smokeResult.failure.Observations = append(smokeResult.failure.Observations, observations...)
		return *smokeResult.failure
	} else if smokeResult.skipped {
		logPhaseStep(logWriter, "⏭️  %s", smokeResult.observation)
		observations = append(observations, NewSkipObservation(smokeResult.observation))
	} else if smokeResult.observation != "" {
		logPhaseSuccess(logWriter, "%s", smokeResult.observation)
		observations = append(observations, NewSuccessObservation(smokeResult.observation))
	} else {
		logPhaseStep(logWriter, "⏭️  UI smoke telemetry not configured")
		observations = append(observations, NewSkipObservation("UI smoke telemetry not configured"))
	}

	// Final summary
	totalChecks := len(requiredDirs) + len(requiredFiles) + 2 // +2 for manifest checks
	observations = append(observations, Observation{Icon: "✅", Text: fmt.Sprintf("Structure validation completed (%d checks)", totalChecks)})

	logPhaseSuccess(logWriter, "Structure validation complete")
	return RunReport{Observations: observations}
}

func validateStructureDirs(scenarioDir string, required []string, logWriter io.Writer) RunReport {
	for _, rel := range required {
		abs := resolveStructurePath(scenarioDir, rel)
		if err := ensureDir(abs); err != nil {
			logPhaseError(logWriter, "Missing directory: %s", rel)
			return RunReport{
				Err:                   err,
				FailureClassification: FailureClassMisconfiguration,
				Remediation:           fmt.Sprintf("Create the '%s' directory to match the scenario template.", rel),
			}
		}
		logPhaseStep(logWriter, "  ✓ %s", rel)
	}
	return RunReport{}
}

func validateStructureFiles(scenarioDir string, required []string, logWriter io.Writer) RunReport {
	for _, rel := range required {
		abs := resolveStructurePath(scenarioDir, rel)
		if err := ensureFile(abs); err != nil {
			logPhaseError(logWriter, "Missing file: %s", rel)
			return RunReport{
				Err:                   err,
				FailureClassification: FailureClassMisconfiguration,
				Remediation:           fmt.Sprintf("Restore the file '%s' so structure validation can pass.", rel),
			}
		}
		logPhaseStep(logWriter, "  ✓ %s", rel)
	}
	return RunReport{}
}

func validateServiceManifest(manifestPath, scenarioName string, logWriter io.Writer, expectations *structureExpectations) RunReport {
	if err := ensureFile(manifestPath); err != nil {
		logPhaseError(logWriter, "service.json not found")
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Run `vrooli scenario init` or restore .vrooli/service.json.",
		}
	}
	logPhaseStep(logWriter, "  ✓ service.json exists")

	manifest, err := workspace.LoadServiceManifest(manifestPath)
	if err != nil {
		logPhaseError(logWriter, "Failed to parse service.json: %v", err)
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           fmt.Sprintf("Fix JSON syntax in %s so the manifest can be parsed.", manifestPath),
		}
	}
	logPhaseStep(logWriter, "  ✓ service.json valid JSON")

	if expectations.ValidateServiceName && manifest.Service.Name != scenarioName {
		logPhaseError(logWriter, "service.name '%s' does not match scenario '%s'", manifest.Service.Name, scenarioName)
		return RunReport{
			Err:                   fmt.Errorf("service.name '%s' does not match scenario '%s'", manifest.Service.Name, scenarioName),
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           fmt.Sprintf("Update service.name in %s to '%s'.", manifestPath, scenarioName),
		}
	}
	if manifest.Service.Name == "" {
		logPhaseError(logWriter, "service.name is required")
		return RunReport{
			Err:                   fmt.Errorf("service.name is required in %s", manifestPath),
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Set service.name to the scenario folder name in .vrooli/service.json.",
		}
	}
	logPhaseStep(logWriter, "  ✓ service.name matches scenario")

	if len(manifest.Lifecycle.Health.Checks) == 0 {
		logPhaseError(logWriter, "No health checks defined")
		return RunReport{
			Err:                   fmt.Errorf("lifecycle.health.checks must define at least one entry in %s", manifestPath),
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Add at least one health check entry under lifecycle.health.checks.",
		}
	}
	logPhaseStep(logWriter, "  ✓ health checks defined (%d)", len(manifest.Lifecycle.Health.Checks))

	return RunReport{}
}

func resolveStructureRequirements(scenarioName string, expectations *structureExpectations) (dirs []string, files []string) {
	dirs = append([]string{}, standardStructureDirs...)
	files = append([]string{}, standardStructureFiles...)
	files = append(files, filepath.Join("cli", scenarioName))

	if expectations != nil {
		dirs = append(dirs, expectations.AdditionalDirs...)
		files = append(files, expectations.AdditionalFiles...)
		dirs = filterStructurePaths(dirs, expectations.ExcludedDirs)
		files = filterStructurePaths(files, expectations.ExcludedFiles)
	}

	dirs = deduplicateStructurePaths(dirs)
	files = deduplicateStructurePaths(files)
	return dirs, files
}

func filterStructurePaths(paths, excludes []string) []string {
	if len(excludes) == 0 {
		return paths
	}
	excludeSet := make(map[string]struct{}, len(excludes))
	for _, path := range excludes {
		excludeSet[canonicalStructurePath(path)] = struct{}{}
	}
	var filtered []string
	for _, path := range paths {
		clean := canonicalStructurePath(path)
		if clean == "" {
			continue
		}
		if _, skip := excludeSet[clean]; skip {
			continue
		}
		filtered = append(filtered, clean)
	}
	return filtered
}

func deduplicateStructurePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	var result []string
	for _, path := range paths {
		clean := canonicalStructurePath(path)
		if clean == "" {
			continue
		}
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		result = append(result, clean)
	}
	return result
}

func resolveStructurePath(baseDir, rel string) string {
	if rel == "" {
		return baseDir
	}
	osSpecific := filepath.FromSlash(rel)
	if filepath.IsAbs(osSpecific) {
		return osSpecific
	}
	return filepath.Join(baseDir, osSpecific)
}

func canonicalStructurePath(path string) string {
	clean := filepath.Clean(path)
	clean = strings.TrimPrefix(clean, "./")
	if clean == "." {
		return ""
	}
	return filepath.ToSlash(clean)
}

func scanScenarioJSON(root string) (int, []string, error) {
	var invalid []string
	count := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if _, skip := jsonValidationSkipDirs[entry.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			return nil
		}
		count++
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		if !json.Valid(data) {
			if rel, relErr := filepath.Rel(root, path); relErr == nil {
				invalid = append(invalid, filepath.ToSlash(rel))
			} else {
				invalid = append(invalid, path)
			}
		}
		return nil
	})
	return count, invalid, err
}

// uiSmokeOutcome holds the result of a UI smoke telemetry check.
type uiSmokeOutcome struct {
	observation string     // Human-readable result message
	skipped     bool       // True if the test was skipped (not a failure)
	failure     *RunReport // Non-nil if the test failed or was blocked
}

func enforceUISmokeTelemetry(ctx context.Context, env workspace.Environment, logWriter io.Writer) uiSmokeOutcome {
	// Use native Go implementation if enabled
	if UISmokeModeNative {
		return runNativeUISmoke(ctx, env, logWriter)
	}

	// Fall back to reading cached results
	return readCachedUISmokeTelemetry(ctx, env, logWriter)
}

// runNativeUISmoke executes the UI smoke test using the new Go implementation.
func runNativeUISmoke(ctx context.Context, env workspace.Environment, logWriter io.Writer) uiSmokeOutcome {
	logPhaseStep(logWriter, "running native UI smoke test...")

	result, err := uismoke.RunForPhase(ctx, env.ScenarioName, env.ScenarioDir, logWriter)
	if err != nil {
		logPhaseWarn(logWriter, "ui smoke execution failed: %v", err)
		return uiSmokeOutcome{failure: &RunReport{
			Err:                   err,
			FailureClassification: FailureClassSystem,
			Remediation:           "Check browserless availability and scenario UI configuration.",
		}}
	}

	if result.Skipped {
		return uiSmokeOutcome{observation: result.Message, skipped: true}
	}

	if result.Blocked {
		return uiSmokeOutcome{failure: &RunReport{
			Err:                   result.ToError(),
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           result.Message,
		}}
	}

	if !result.Success {
		// Check if it's a bundle staleness issue
		if fresh, reason := result.GetBundleStatus(); !fresh {
			return uiSmokeOutcome{failure: &RunReport{
				Err:                   fmt.Errorf("ui bundle stale: %s", reason),
				FailureClassification: FailureClassMisconfiguration,
				Remediation:           "Rebuild or restart the UI so bundles are regenerated before re-running structure tests.",
			}}
		}

		return uiSmokeOutcome{failure: &RunReport{
			Err:                   result.ToError(),
			FailureClassification: FailureClassSystem,
			Remediation:           "Investigate the UI smoke failure (see artifacts in coverage/<scenario>/ui-smoke/) and fix the underlying issue.",
		}}
	}

	return uiSmokeOutcome{observation: result.FormatObservation()}
}

// readCachedUISmokeTelemetry reads UI smoke results from cached scenario status.
func readCachedUISmokeTelemetry(ctx context.Context, env workspace.Environment, logWriter io.Writer) uiSmokeOutcome {
	status, err := fetchScenarioStatus(ctx, env, logWriter)
	if err != nil {
		logPhaseWarn(logWriter, "ui smoke telemetry unavailable: %v", err)
		return uiSmokeOutcome{skipped: true, observation: "ui smoke telemetry unavailable"}
	}
	if status.Diagnostics.UISmoke == nil {
		return uiSmokeOutcome{skipped: true, observation: "ui smoke telemetry not reported"}
	}
	smoke := status.Diagnostics.UISmoke
	if strings.EqualFold(smoke.Status, "passed") {
		if smoke.Bundle != nil && !smoke.Bundle.Fresh {
			reason := strings.TrimSpace(smoke.Bundle.Reason)
			if reason == "" {
				reason = "bundle marked stale"
			}
			return uiSmokeOutcome{failure: &RunReport{
				Err:                   fmt.Errorf("ui bundle stale: %s", reason),
				FailureClassification: FailureClassMisconfiguration,
				Remediation:           "Rebuild or restart the UI so bundles are regenerated before re-running structure tests.",
			}}
		}
		duration := int(smoke.DurationMs)
		if duration < 0 {
			duration = 0
		}
		return uiSmokeOutcome{observation: fmt.Sprintf("ui smoke passed (%dms)", duration)}
	}
	if strings.EqualFold(smoke.Status, "skipped") {
		return uiSmokeOutcome{skipped: true, observation: smoke.Message}
	}
	message := strings.TrimSpace(smoke.Message)
	if message == "" {
		message = "UI smoke reported a failure"
	}
	return uiSmokeOutcome{failure: &RunReport{
		Err:                   fmt.Errorf("ui smoke status '%s': %s", smoke.Status, message),
		FailureClassification: FailureClassSystem,
		Remediation:           "Investigate the UI smoke failure (see scenario status diagnostics) and restart the scenario before retrying.",
	}}
}
