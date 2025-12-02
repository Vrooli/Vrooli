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
)

var (
	standardStructureDirs = []string{
		"api",
		"cli",
		"docs",
		"requirements",
		"test",
		filepath.Join("test", "phases"),
		"ui",
	}
	standardStructureFiles = []string{
		filepath.Join("api", "main.go"),
		filepath.Join("cli", "install.sh"),
		filepath.Join("test", "run-tests.sh"),
		filepath.Join("test", "phases", "test-structure.sh"),
		filepath.Join("test", "phases", "test-dependencies.sh"),
		filepath.Join("test", "phases", "test-unit.sh"),
		filepath.Join("test", "phases", "test-integration.sh"),
		filepath.Join("test", "phases", "test-business.sh"),
		filepath.Join("test", "phases", "test-performance.sh"),
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

	expectations, err := loadStructureExpectations(env.ScenarioDir)
	if err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Fix .vrooli/testing.json so structure requirements can be parsed.",
		}
	}

	requiredDirs, requiredFiles := resolveStructureRequirements(env.ScenarioName, expectations)
	var observations []string

	if report := validateStructureDirs(env.ScenarioDir, requiredDirs, logWriter); report.Err != nil {
		return report
	}
	observations = append(observations, fmt.Sprintf("directories validated: %d", len(requiredDirs)))

	if report := validateStructureFiles(env.ScenarioDir, requiredFiles, logWriter); report.Err != nil {
		return report
	}
	observations = append(observations, fmt.Sprintf("files validated: %d", len(requiredFiles)))

	manifestPath := filepath.Join(env.ScenarioDir, ".vrooli", "service.json")
	if result := validateServiceManifest(manifestPath, env.ScenarioName, logWriter, expectations); result.Err != nil {
		return result
	}

	if expectations.ValidateJSONFiles {
		jsonCount, invalidFiles, err := scanScenarioJSON(env.ScenarioDir)
		if err != nil {
			return RunReport{
				Err:                   err,
				FailureClassification: FailureClassSystem,
				Remediation:           "Ensure JSON files under the scenario tree are readable.",
				Observations:          observations,
			}
		}
		if len(invalidFiles) > 0 {
			return RunReport{
				Err:                   fmt.Errorf("invalid JSON detected: %s", strings.Join(invalidFiles, ", ")),
				FailureClassification: FailureClassMisconfiguration,
				Remediation:           "Fix the malformed JSON files listed in the error message.",
				Observations:          observations,
			}
		}
		logPhaseStep(logWriter, "validated %d JSON files", jsonCount)
		observations = append(observations, fmt.Sprintf("json files validated: %d", jsonCount))
	}

	runnerPath := filepath.Join(env.TestDir, "run-tests.sh")
	if err := ensureFile(runnerPath); err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Restore test/run-tests.sh from the scenario template.",
			Observations:          observations,
		}
	}
	content, err := os.ReadFile(runnerPath)
	if err != nil {
		return RunReport{
			Err:                   fmt.Errorf("failed to read %s: %w", runnerPath, err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Ensure the orchestrator script is readable by the scenario runtime.",
			Observations:          observations,
		}
	}
	if strings.Contains(string(content), "scripts/scenarios/testing") {
		return RunReport{
			Err:                   fmt.Errorf("%s must not reference scripts/scenarios/testing", runnerPath),
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Update run-tests.sh to call the scenario-local orchestrator instead of legacy scripts.",
			Observations:          observations,
		}
	}
	logPhaseStep(logWriter, "verified scenario-local orchestrator usage: %s", runnerPath)
	observations = append(observations, "scenario-local orchestrator enforced")

	if uiObservation, failure := enforceUISmokeTelemetry(ctx, env, logWriter); failure != nil {
		failure.Observations = append(failure.Observations, observations...)
		return *failure
	} else if uiObservation != "" {
		observations = append(observations, uiObservation)
	}

	logPhaseStep(logWriter, "structure validation complete")
	return RunReport{Observations: observations}
}

func validateStructureDirs(scenarioDir string, required []string, logWriter io.Writer) RunReport {
	for _, rel := range required {
		abs := resolveStructurePath(scenarioDir, rel)
		if err := ensureDir(abs); err != nil {
			return RunReport{
				Err:                   err,
				FailureClassification: FailureClassMisconfiguration,
				Remediation:           fmt.Sprintf("Create the '%s' directory to match the scenario template.", rel),
			}
		}
		logPhaseStep(logWriter, "directory verified: %s", abs)
	}
	return RunReport{}
}

func validateStructureFiles(scenarioDir string, required []string, logWriter io.Writer) RunReport {
	for _, rel := range required {
		abs := resolveStructurePath(scenarioDir, rel)
		if err := ensureFile(abs); err != nil {
			return RunReport{
				Err:                   err,
				FailureClassification: FailureClassMisconfiguration,
				Remediation:           fmt.Sprintf("Restore the file '%s' so structure validation can pass.", rel),
			}
		}
		logPhaseStep(logWriter, "file verified: %s", abs)
	}
	return RunReport{}
}

func validateServiceManifest(manifestPath, scenarioName string, logWriter io.Writer, expectations *structureExpectations) RunReport {
	if err := ensureFile(manifestPath); err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Run `vrooli scenario init` or restore .vrooli/service.json.",
		}
	}
	logPhaseStep(logWriter, "manifest located: %s", manifestPath)
	manifest, err := workspace.LoadServiceManifest(manifestPath)
	if err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           fmt.Sprintf("Fix JSON syntax in %s so the manifest can be parsed.", manifestPath),
		}
	}
	if expectations.ValidateServiceName && manifest.Service.Name != scenarioName {
		return RunReport{
			Err:                   fmt.Errorf("service.name '%s' does not match scenario '%s'", manifest.Service.Name, scenarioName),
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           fmt.Sprintf("Update service.name in %s to '%s'.", manifestPath, scenarioName),
		}
	}
	if manifest.Service.Name == "" {
		return RunReport{
			Err:                   fmt.Errorf("service.name is required in %s", manifestPath),
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Set service.name to the scenario folder name in .vrooli/service.json.",
		}
	}
	if len(manifest.Lifecycle.Health.Checks) == 0 {
		return RunReport{
			Err:                   fmt.Errorf("lifecycle.health.checks must define at least one entry in %s", manifestPath),
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Add at least one health check entry under lifecycle.health.checks.",
		}
	}
	logPhaseStep(logWriter, "manifest validated for scenario '%s'", scenarioName)
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

func enforceUISmokeTelemetry(ctx context.Context, env workspace.Environment, logWriter io.Writer) (string, *RunReport) {
	status, err := fetchScenarioStatus(ctx, env, logWriter)
	if err != nil {
		logPhaseWarn(logWriter, "ui smoke telemetry unavailable: %v", err)
		return "", nil
	}
	if status.Diagnostics.UISmoke == nil {
		return "ui smoke telemetry not reported", nil
	}
	smoke := status.Diagnostics.UISmoke
	if strings.EqualFold(smoke.Status, "passed") {
		if smoke.Bundle != nil && !smoke.Bundle.Fresh {
			reason := strings.TrimSpace(smoke.Bundle.Reason)
			if reason == "" {
				reason = "bundle marked stale"
			}
			return "", &RunReport{
				Err:                   fmt.Errorf("ui bundle stale: %s", reason),
				FailureClassification: FailureClassMisconfiguration,
				Remediation:           "Rebuild or restart the UI so bundles are regenerated before re-running structure tests.",
			}
		}
		duration := int(smoke.DurationMs)
		if duration < 0 {
			duration = 0
		}
		return fmt.Sprintf("ui smoke passed (%dms)", duration), nil
	}
	message := strings.TrimSpace(smoke.Message)
	if message == "" {
		message = "UI smoke reported a failure"
	}
	return "", &RunReport{
		Err:                   fmt.Errorf("ui smoke status '%s': %s", smoke.Status, message),
		FailureClassification: FailureClassSystem,
		Remediation:           "Investigate the UI smoke failure (see scenario status diagnostics) and restart the scenario before retrying.",
	}
}
