// Package artifacts provides centralized artifact path management.
//
// This file defines all artifact locations for test-genie, ensuring consistency
// across phases and making it easy to locate test outputs.
//
// Directory Structure:
//
//	<scenario>/
//	├── coverage/                      # All test outputs under one root
//	│   ├── phase-results/             # Per-phase JSON summaries
//	│   │   ├── smoke.json
//	│   │   ├── unit.json
//	│   │   ├── playbooks.json
//	│   │   ├── lighthouse.json
//	│   │   └── ...
//	│   ├── ui-smoke/                  # UI smoke test artifacts
//	│   │   ├── latest.json
//	│   │   ├── screenshot.png
//	│   │   ├── console.json
//	│   │   ├── network.json
//	│   │   ├── dom.html
//	│   │   └── README.md
//	│   ├── automation/                # Playbook execution timelines
//	│   │   └── *.timeline.json
//	│   ├── lighthouse/                # Lighthouse performance reports
//	│   │   ├── <page-id>.json
//	│   │   ├── <page-id>.html
//	│   │   └── summary.json
//	│   ├── unit/                      # Unit test failure artifacts
//	│   │   └── <test-name>/README.md
//	│   ├── sync/                      # Requirement sync metadata
//	│   │   └── latest.json
//	│   ├── manual-validations/        # Manual validation logs
//	│   │   └── log.jsonl
//	│   └── vitest-requirements.json   # Vitest requirement mapping
//	└── test/
//	    └── artifacts/
//	        └── runtime/               # Runtime state (seeds)
//	            └── seed-state.json
package artifacts

import "path/filepath"

// ============================================================================
// Path Constants (relative to scenario root)
// ============================================================================

const (
	// CoverageRoot is the root directory for all coverage artifacts.
	CoverageRoot = "coverage"

	// LogsDir holds per-run phase logs.
	LogsDir = "coverage/logs"

	// LatestDir holds pointers and manifests for the most recent run.
	LatestDir = "coverage/latest"

	// PhaseResultsDir is where per-phase JSON summaries are written.
	// Each phase writes <phase>.json here (e.g., smoke.json, unit.json).
	PhaseResultsDir = "coverage/phase-results"

	// UISmokeDir holds UI smoke test artifacts (screenshots, console logs, etc).
	UISmokeDir = "coverage/ui-smoke"

	// AutomationDir holds playbook execution timeline artifacts.
	AutomationDir = "coverage/automation"

	// LighthouseDir holds Lighthouse performance audit reports.
	LighthouseDir = "coverage/lighthouse"

	// UnitDir holds unit test failure artifacts (READMEs with context).
	UnitDir = "coverage/unit"

	// SyncDir holds requirement synchronization metadata.
	SyncDir = "coverage/sync"

	// ManualValidationsDir holds manual validation logs.
	ManualValidationsDir = "coverage/manual-validations"

	// RuntimeDir holds runtime state like seed data.
	RuntimeDir = "coverage/runtime"

	// LatestManifestFile is the manifest describing the latest run artifacts.
	LatestManifestFile = "manifest.json"
)

// ============================================================================
// Common Filenames
// ============================================================================

const (
	// PhaseResultsSmoke is the filename for smoke phase results.
	PhaseResultsSmoke = "smoke.json"

	// PhaseResultsUnit is the filename for unit phase results.
	PhaseResultsUnit = "unit.json"

	// PhaseResultsPlaybooks is the filename for playbooks phase results.
	PhaseResultsPlaybooks = "playbooks.json"

	// PhaseResultsLighthouse is the filename for lighthouse phase results.
	PhaseResultsLighthouse = "lighthouse.json"

	// PhaseResultsPerformance is the filename for performance phase results.
	PhaseResultsPerformance = "performance.json"

	// SyncMetadataFile is the filename for sync metadata.
	SyncMetadataFile = "latest.json"

	// ManualValidationsLog is the filename for manual validation logs.
	ManualValidationsLog = "log.jsonl"

	// VitestRequirementsFile is the filename for vitest requirement mapping.
	VitestRequirementsFile = "vitest-requirements.json"

	// SeedStateFile is the filename for seed state.
	SeedStateFile = "seed-state.json"

	// UISmokeLatest is the filename for UI smoke test results.
	UISmokeLatest = "latest.json"

	// UISmokeScreenshot is the filename for UI smoke screenshot.
	UISmokeScreenshot = "screenshot.png"

	// UISmokeConsole is the filename for UI smoke console logs.
	UISmokeConsole = "console.json"

	// UISmokeNetwork is the filename for UI smoke network failures.
	UISmokeNetwork = "network.json"

	// UISmokeDOM is the filename for UI smoke DOM snapshot.
	UISmokeDOM = "dom.html"

	// UISmokeRaw is the filename for UI smoke raw response.
	UISmokeRaw = "raw.json"

	// UISmokeReadme is the filename for UI smoke README.
	UISmokeReadme = "README.md"

	// LighthouseSummary is the filename for Lighthouse summary.
	LighthouseSummary = "summary.json"
)

// ============================================================================
// Path Builders - Returns absolute paths given scenario root
// ============================================================================

// PhaseResultsPath returns the absolute path for a phase results file.
func PhaseResultsPath(scenarioDir, filename string) string {
	return filepath.Join(scenarioDir, PhaseResultsDir, filename)
}

// UISmokeArtifactPath returns the absolute path for a UI smoke artifact.
func UISmokeArtifactPath(scenarioDir, filename string) string {
	return filepath.Join(scenarioDir, UISmokeDir, filename)
}

// AutomationArtifactPath returns the absolute path for an automation/playbook artifact.
func AutomationArtifactPath(scenarioDir, filename string) string {
	return filepath.Join(scenarioDir, AutomationDir, filename)
}

// LighthouseArtifactPath returns the absolute path for a Lighthouse artifact.
func LighthouseArtifactPath(scenarioDir, filename string) string {
	return filepath.Join(scenarioDir, LighthouseDir, filename)
}

// UnitArtifactPath returns the absolute path for a unit test artifact.
func UnitArtifactPath(scenarioDir, testName string) string {
	return filepath.Join(scenarioDir, UnitDir, testName)
}

// SyncMetadataPath returns the absolute path for sync metadata.
func SyncMetadataPath(scenarioDir string) string {
	return filepath.Join(scenarioDir, SyncDir, SyncMetadataFile)
}

// ManualValidationsPath returns the absolute path for manual validations log.
func ManualValidationsPath(scenarioDir string) string {
	return filepath.Join(scenarioDir, ManualValidationsDir, ManualValidationsLog)
}

// SeedStatePath returns the absolute path for seed state.
func SeedStatePath(scenarioDir string) string {
	return filepath.Join(scenarioDir, RuntimeDir, SeedStateFile)
}

// ============================================================================
// Vitest Paths (may be in scenario root or ui/ subdirectory)
// ============================================================================

// VitestRequirementsPaths returns all possible paths for vitest requirements file.
// The first match found should be used.
func VitestRequirementsPaths(scenarioDir string) []string {
	return []string{
		filepath.Join(scenarioDir, "ui", "coverage", VitestRequirementsFile),
		filepath.Join(scenarioDir, CoverageRoot, VitestRequirementsFile),
		filepath.Join(scenarioDir, "test", "coverage", "vitest.json"), // Legacy
	}
}

// ============================================================================
// Legacy/Fallback Paths (for backwards compatibility)
// ============================================================================

// ============================================================================
// Directory Creation Helpers
// ============================================================================

// AllCoverageSubdirs returns all coverage subdirectories that may need creation.
func AllCoverageSubdirs(scenarioDir string) []string {
	return []string{
		filepath.Join(scenarioDir, LogsDir),
		filepath.Join(scenarioDir, LatestDir),
		filepath.Join(scenarioDir, PhaseResultsDir),
		filepath.Join(scenarioDir, UISmokeDir),
		filepath.Join(scenarioDir, AutomationDir),
		filepath.Join(scenarioDir, LighthouseDir),
		filepath.Join(scenarioDir, UnitDir),
		filepath.Join(scenarioDir, SyncDir),
		filepath.Join(scenarioDir, ManualValidationsDir),
		filepath.Join(scenarioDir, RuntimeDir),
	}
}

// ============================================================================
// Relative Path Helpers (for artifact references in JSON output)
// ============================================================================

// RelativePhaseResultsPath returns the relative path for a phase results file.
func RelativePhaseResultsPath(filename string) string {
	return filepath.Join(PhaseResultsDir, filename)
}

// RelativeUISmokeArtifactPath returns the relative path for a UI smoke artifact.
func RelativeUISmokeArtifactPath(filename string) string {
	return filepath.Join(UISmokeDir, filename)
}

// RelativeAutomationArtifactPath returns the relative path for an automation artifact.
func RelativeAutomationArtifactPath(filename string) string {
	return filepath.Join(AutomationDir, filename)
}

// RelativeLighthouseArtifactPath returns the relative path for a Lighthouse artifact.
func RelativeLighthouseArtifactPath(filename string) string {
	return filepath.Join(LighthouseDir, filename)
}

// RunLogsDir builds the absolute path for a specific run's logs.
func RunLogsDir(scenarioDir, runID string) string {
	return filepath.Join(scenarioDir, LogsDir, runID)
}

// LatestDirPath returns the absolute path for latest pointers.
func LatestDirPath(scenarioDir string) string {
	return filepath.Join(scenarioDir, LatestDir)
}

// LatestManifestPath returns the absolute path for the latest manifest file.
func LatestManifestPath(scenarioDir string) string {
	return filepath.Join(LatestDirPath(scenarioDir), LatestManifestFile)
}
