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
//	│   │   ├── ui-health.json
//	│   │   ├── unit.json
//	│   │   ├── playbooks.json
//	│   │   └── ...
//	│   ├── ui-smoke/                  # Legacy per-page visual artifact tree
//	│   │   ├── latest.json
//	│   │   ├── screenshot.png
//	│   │   ├── console.json
//	│   │   ├── network.json
//	│   │   ├── dom.html
//	│   │   └── README.md
//	│   ├── automation/                # Playbook execution timelines
//	│   │   └── *.timeline.json
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

	// RunsDir is the root for append-only, runID-keyed run artifacts.
	// Each run writes under coverage/runs/<runID>/{phase-results,ui-smoke,...}.
	// The ui-smoke path is retained as a historical storage directory for per-page
	// visual artifacts; active visual judgment is delegated to ui-health.
	RunsDir = "coverage/runs"

	// RunsIndexFile is the append-only index of all runs, stored under coverage/.
	RunsIndexFile = "runs.index.json"

	// PhaseResultsSubdir is the per-run subdirectory for phase JSON summaries.
	// Each phase writes <phase>.json here (e.g., ui-health.json, unit.json).
	PhaseResultsSubdir = "phase-results"

	// UISmokeSubdir is the historical per-run subdirectory for page visual artifacts.
	UISmokeSubdir = "ui-smoke"

	// AutomationSubdir is the per-run subdirectory for playbook timelines.
	AutomationSubdir = "automation"

	// UnitSubdir is the per-run subdirectory for unit test failure artifacts.
	UnitSubdir = "unit"

	// SyncDir holds requirement synchronization metadata.
	SyncDir = "coverage/sync"

	// ManualValidationsDir holds manual validation logs.
	ManualValidationsDir = "coverage/manual-validations"

	// RuntimeDir holds runtime state like seed data.
	RuntimeDir = "coverage/runtime"

	// LatestManifestFile is the manifest describing the latest run artifacts.
	LatestManifestFile = "manifest.json"

	// FindingsArtifactFile is the per-run combined findings document. It
	// gathers every phase's normalized findings (zero-finding phases
	// included) into one file shaped as the `--from-audit` ingest contract,
	// so the campaign nudge can point at a file that already exists on disk.
	FindingsArtifactFile = "findings.json"

	// RunSnapshotFile is the canonical versioned terminal run record. It is
	// written before the compact index flips terminal so durable readers never
	// observe a success-shaped index entry without its full terminal evidence.
	RunSnapshotFile = "run-snapshot.json"

	// DescriptorSnapshotFile is the effective provider descriptor catalog and
	// applicability decision set frozen before phase execution begins.
	DescriptorSnapshotFile = "descriptor-snapshot.json"

	// ArtifactCatalogFile is the versioned, run-scoped inventory of evidence
	// bytes. The catalog stores private storage locators on disk; API projections
	// expose only opaque artifact IDs.
	ArtifactCatalogFile = "artifact-catalog.json"
)

// ============================================================================
// Common Filenames
// ============================================================================

const (
	// PhaseResultsSmoke is the legacy filename for retired browser-render results.
	PhaseResultsSmoke = "smoke.json"

	// PhaseResultsUnit is the filename for unit phase results.
	PhaseResultsUnit = "unit.json"

	// PhaseResultsPlaybooks is the filename for playbooks phase results.
	PhaseResultsPlaybooks = "playbooks.json"

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
)

// ============================================================================
// Path Builders - Returns absolute paths given scenario root
// ============================================================================

// RunDir returns the absolute path for a specific run's artifact root.
func RunDir(scenarioDir, runID string) string {
	return filepath.Join(scenarioDir, RunsDir, runID)
}

// TargetRunDir keeps artifacts for non-scenario targets out of shipped source
// trees. Scenario paths preserve the historical layout; all other targets use
// the runtime home state area.
func TargetRunDir(runtimeHome, targetKind, targetID, scenarioDir, runID string) string {
	if targetKind == "" || targetKind == "scenario" {
		return RunDir(scenarioDir, runID)
	}
	return filepath.Join(runtimeHome, "state", "test-genie", "targets", targetKind, targetID, RunsDir, runID)
}

// RunPhaseResultsDir returns the absolute per-run phase-results directory.
func RunPhaseResultsDir(scenarioDir, runID string) string {
	return filepath.Join(RunDir(scenarioDir, runID), PhaseResultsSubdir)
}

// RunUISmokeDir returns the absolute per-run legacy visual artifact directory.
func RunUISmokeDir(scenarioDir, runID string) string {
	return filepath.Join(RunDir(scenarioDir, runID), UISmokeSubdir)
}

// UISmokePagesSubdir is the per-run subdirectory holding all-pages visual
// captures (one directory per page) produced under the baseline capture profile.
const UISmokePagesSubdir = "pages"

// RunUISmokePagesDir returns the absolute per-run all-pages visual capture
// directory. It uses the historical ui-smoke storage path for compatibility
// with existing run artifacts.
func RunUISmokePagesDir(scenarioDir, runID string) string {
	return filepath.Join(RunUISmokeDir(scenarioDir, runID), UISmokePagesSubdir)
}

// RunAutomationDir returns the absolute per-run automation directory.
func RunAutomationDir(scenarioDir, runID string) string {
	return filepath.Join(RunDir(scenarioDir, runID), AutomationSubdir)
}

// RunUnitDir returns the absolute per-run unit directory.
func RunUnitDir(scenarioDir, runID string) string {
	return filepath.Join(RunDir(scenarioDir, runID), UnitSubdir)
}

// RunFindingsArtifactPath returns the absolute path to a run's combined
// findings document (coverage/runs/<runID>/findings.json).
func RunFindingsArtifactPath(scenarioDir, runID string) string {
	return filepath.Join(RunDir(scenarioDir, runID), FindingsArtifactFile)
}

// RunSnapshotPath returns the canonical terminal snapshot path for a run.
func RunSnapshotPath(scenarioDir, runID string) string {
	return filepath.Join(RunDir(scenarioDir, runID), RunSnapshotFile)
}

// RunDescriptorSnapshotPath returns the immutable planning-time descriptor
// snapshot path for a run.
func RunDescriptorSnapshotPath(scenarioDir, runID string) string {
	return filepath.Join(RunDir(scenarioDir, runID), DescriptorSnapshotFile)
}

// RunArtifactCatalogPath returns the private on-disk artifact catalog path for
// a run. Consumers must use the typed artifact API rather than reading it.
func RunArtifactCatalogPath(scenarioDir, runID string) string {
	return filepath.Join(RunDir(scenarioDir, runID), ArtifactCatalogFile)
}

// RelativeRunFindingsArtifactPath returns the scenario-relative path to a
// run's combined findings document, suitable for embedding in the campaign
// nudge command (`--from-audit coverage/runs/<runID>/findings.json`).
func RelativeRunFindingsArtifactPath(runID string) string {
	return filepath.Join(RunsDir, runID, FindingsArtifactFile)
}

// RunsIndexPath returns the absolute path to the append-only runs index.
func RunsIndexPath(scenarioDir string) string {
	return filepath.Join(scenarioDir, CoverageRoot, RunsIndexFile)
}

// PhaseResultsPath returns the absolute path for a phase results file in a run.
func PhaseResultsPath(scenarioDir, runID, filename string) string {
	return filepath.Join(RunPhaseResultsDir(scenarioDir, runID), filename)
}

// AutomationArtifactPath returns the absolute path for an automation/playbook artifact in a run.
func AutomationArtifactPath(scenarioDir, runID, filename string) string {
	return filepath.Join(RunAutomationDir(scenarioDir, runID), filename)
}

// UnitArtifactPath returns the absolute path for a unit test artifact in a run.
func UnitArtifactPath(scenarioDir, runID, testName string) string {
	return filepath.Join(RunUnitDir(scenarioDir, runID), testName)
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

// AllCoverageSubdirs returns the scenario-global coverage subdirectories that
// may need creation. Per-run directories under coverage/runs/<runID>/ are
// created lazily by writers once the runID is known.
func AllCoverageSubdirs(scenarioDir string) []string {
	return []string{
		filepath.Join(scenarioDir, LogsDir),
		filepath.Join(scenarioDir, LatestDir),
		filepath.Join(scenarioDir, RunsDir),
		filepath.Join(scenarioDir, SyncDir),
		filepath.Join(scenarioDir, ManualValidationsDir),
		filepath.Join(scenarioDir, RuntimeDir),
	}
}

// ============================================================================
// Relative Path Helpers (for artifact references in JSON output)
// ============================================================================

// RelativePhaseResultsPath returns the run-relative path for a phase results file.
func RelativePhaseResultsPath(runID, filename string) string {
	return filepath.Join(RunsDir, runID, PhaseResultsSubdir, filename)
}

// RelativeAutomationArtifactPath returns the run-relative path for an automation artifact.
func RelativeAutomationArtifactPath(runID, filename string) string {
	return filepath.Join(RunsDir, runID, AutomationSubdir, filename)
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
