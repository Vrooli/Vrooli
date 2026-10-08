// Package artifactpaths is the sanctioned repository-wide authority for generated-artifact names.
package artifactpaths

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/storage"
)

const (
	TestGenieOwner          = "test-genie"
	CoverageRoot            = "coverage"
	LogsDir                 = "coverage/logs"
	LatestDir               = "coverage/latest"
	RunsDir                 = "coverage/runs"
	RunsIndexFile           = "runs.index.json"
	PhaseResultsSubdir      = "phase-results"
	UISmokeSubdir           = "ui-smoke"
	UISmokePagesSubdir      = "pages"
	AutomationSubdir        = "automation"
	UnitSubdir              = "unit"
	SyncDir                 = "coverage/sync"
	ManualValidationsDir    = "coverage/manual-validations"
	RuntimeDir              = "coverage/runtime"
	LatestManifestFile      = "manifest.json"
	FindingsArtifactFile    = "findings.json"
	RunSnapshotFile         = "run-snapshot.json"
	DescriptorSnapshotFile  = "descriptor-snapshot.json"
	ArtifactCatalogFile     = "artifact-catalog.json"
	PhaseResultsSmoke       = "smoke.json"
	PhaseResultsUnit        = "unit.json"
	PhaseResultsPlaybooks   = "playbooks.json"
	PhaseResultsPerformance = "performance.json"
	SyncMetadataFile        = "latest.json"
	ManualValidationsLog    = "log.jsonl"
	VitestRequirementsFile  = "vitest-requirements.json"
	SeedStateFile           = "seed-state.json"
)

// ScenarioRoot resolves test-genie's private run-artifact root for a scenario.
// Callers may construct logical names beneath the returned root, but they do
// not choose its physical class or runtime-home placement.
func ScenarioRoot(scenarioID string) (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return "", err
	}
	// Preserve the established live layout (test-runs/<target>) while giving a
	// non-live test-genie instance its own owner tree
	// (test-runs/test-genie_<variant>/<target>). Other processes import this
	// package to read Test Genie's live artifacts, so only consume the injected
	// namespace when the current lifecycle owner is Test Genie itself.
	if strings.TrimSpace(os.Getenv(storage.EnvScenario)) == TestGenieOwner {
		namespace, namespaceErr := storage.ScenarioNamespace(TestGenieOwner)
		if namespaceErr != nil {
			return "", namespaceErr
		}
		if namespace != TestGenieOwner {
			return resolver.Path(storage.Options{ScenarioID: namespace}, storage.ClassTestRuns, scenarioID)
		}
	}
	return resolver.Path(storage.Options{ScenarioID: scenarioID}, storage.ClassTestRuns, "")
}

// ScenarioRootForDir resolves the governed run-artifact root for a source
// scenario directory without preserving any physical relationship to that
// directory. The basename is identity only; placement remains contract-owned.
func ScenarioRootForDir(scenarioDir string) (string, error) {
	return ScenarioRoot(filepath.Base(filepath.Clean(scenarioDir)))
}

// PhaseCacheRoot resolves a target-keyed namespace beneath test-genie's cache
// class. Test-genie owns the cache even though target inputs key its contents.
func PhaseCacheRoot(targetID string) (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return "", err
	}
	return resolver.Path(storage.Options{ScenarioID: TestGenieOwner}, storage.ClassCache, targetID)
}

func RunDir(root, runID string) string { return scenarioPath(root, CoverageRoot, "runs", runID) }
func TargetRunDir(home, kind, id, root, runID string) string {
	if kind == "" || kind == "scenario" {
		return RunDir(root, runID)
	}
	return rootPath(home, "state", TestGenieOwner, "targets", kind, id, CoverageRoot, "runs", runID)
}

func RunPhaseResultsDir(root, runID string) string {
	return scenarioPath(root, CoverageRoot, "runs", runID, PhaseResultsSubdir)
}

func RunUISmokeDir(root, runID string) string {
	return scenarioPath(root, CoverageRoot, "runs", runID, UISmokeSubdir)
}

func RunUISmokePagesDir(root, runID string) string {
	return scenarioPath(root, CoverageRoot, "runs", runID, UISmokeSubdir, UISmokePagesSubdir)
}

func RunAutomationDir(root, runID string) string {
	return scenarioPath(root, CoverageRoot, "runs", runID, AutomationSubdir)
}

func RunUnitDir(root, runID string) string {
	return scenarioPath(root, CoverageRoot, "runs", runID, UnitSubdir)
}

func RunFindingsArtifactPath(root, runID string) string {
	return scenarioPath(root, CoverageRoot, "runs", runID, FindingsArtifactFile)
}

func RunSnapshotPath(root, runID string) string {
	return scenarioPath(root, CoverageRoot, "runs", runID, RunSnapshotFile)
}

func RunDescriptorSnapshotPath(root, runID string) string {
	return scenarioPath(root, CoverageRoot, "runs", runID, DescriptorSnapshotFile)
}

func RunArtifactCatalogPath(root, runID string) string {
	return scenarioPath(root, CoverageRoot, "runs", runID, ArtifactCatalogFile)
}

func RelativeRunFindingsArtifactPath(runID string) string {
	return logicalRelative(CoverageRoot, "runs", runID, FindingsArtifactFile)
}
func RunsIndexPath(root string) string { return scenarioPath(root, CoverageRoot, RunsIndexFile) }
func RunsRootPath(root string) string  { return scenarioPath(root, CoverageRoot, "runs") }
func RequirementsSnapshotPath(root string) string {
	return scenarioPath(root, CoverageRoot, "requirements-sync", SyncMetadataFile)
}

func PhaseResultsPath(root, runID, name string) string {
	return scenarioPath(root, CoverageRoot, "runs", runID, PhaseResultsSubdir, name)
}

func AutomationArtifactPath(root, runID, name string) string {
	return scenarioPath(root, CoverageRoot, "runs", runID, AutomationSubdir, name)
}

func UnitArtifactPath(root, runID, name string) string {
	return scenarioPath(root, CoverageRoot, "runs", runID, UnitSubdir, name)
}

func SyncMetadataPath(root string) string {
	return scenarioPath(root, CoverageRoot, "sync", SyncMetadataFile)
}

func ManualValidationsPath(root string) string {
	return scenarioPath(root, CoverageRoot, "manual-validations", ManualValidationsLog)
}

func SeedStatePath(root string) string {
	return scenarioPath(root, CoverageRoot, "runtime", SeedStateFile)
}

func VitestRequirementsPaths(root string) []string {
	return []string{scenarioPath(root, "ui", CoverageRoot, VitestRequirementsFile), scenarioPath(root, CoverageRoot, VitestRequirementsFile), scenarioPath(root, "test", CoverageRoot, "vitest.json")}
}

func AllCoverageSubdirs(root string) []string {
	return []string{scenarioPath(root, CoverageRoot, "logs"), scenarioPath(root, CoverageRoot, "latest"), scenarioPath(root, CoverageRoot, "runs"), scenarioPath(root, CoverageRoot, "sync"), scenarioPath(root, CoverageRoot, "manual-validations"), scenarioPath(root, CoverageRoot, "runtime")}
}

func RelativePhaseResultsPath(runID, name string) string {
	return logicalRelative(CoverageRoot, "runs", runID, PhaseResultsSubdir, name)
}

func RelativeAutomationArtifactPath(runID, name string) string {
	return logicalRelative(CoverageRoot, "runs", runID, AutomationSubdir, name)
}
func RunLogsDir(root, runID string) string { return scenarioPath(root, CoverageRoot, "logs", runID) }
func LatestDirPath(root string) string     { return scenarioPath(root, CoverageRoot, "latest") }
func LatestManifestPath(root string) string {
	return scenarioPath(root, CoverageRoot, "latest", LatestManifestFile)
}
func PhaseCacheDir(root string) string { return scenarioPath(root, "phase-cache") }

// ScenarioPath resolves a logical artifact domain beneath the current
// compatibility scenario root. It is the adoption seam for artifact families
// that do not yet warrant a dedicated named helper.
func ScenarioPath(root, domain string, segments ...string) string {
	return scenarioPath(root, append([]string{domain}, segments...)...)
}

func scenarioPath(root string, parts ...string) string {
	return filepath.Join(root, logicalRelative(parts[0], parts[1:]...))
}

func rootPath(root string, parts ...string) string {
	return filepath.Join(root, logicalRelative(parts[0], parts[1:]...))
}

// logicalRelative routes the logical name through storage.ArtifactPath. Phase 9 replaces
// the compatibility root join above with the resolved absolute class path.
func logicalRelative(domain string, segments ...string) string {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli"})
	if err != nil {
		panic(err)
	}
	opts := storage.Options{ScenarioID: TestGenieOwner, RootOverride: filepath.Join(os.TempDir(), "vrooli-artifact-authority")}
	path, err := resolver.ArtifactPath(opts, storage.ArtifactRef{Owner: TestGenieOwner, Domain: domain, Class: storage.ClassData, Segments: segments})
	if err != nil {
		panic(err)
	}
	base, err := resolver.Path(opts, storage.ClassData, "")
	if err != nil {
		panic(err)
	}
	relative, err := filepath.Rel(base, path)
	if err != nil {
		panic(err)
	}
	return relative
}
