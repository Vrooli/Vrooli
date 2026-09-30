package artifactpaths

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/vrooli/api-core/storage"
)

func TestLegacyVocabularyEquivalence(t *testing.T) {
	t.Parallel()
	root := filepath.Join(string(filepath.Separator), "repo", "scenarios", "demo")
	home := filepath.Join(string(filepath.Separator), "home", "op", ".vrooli")
	tests := map[string][2]string{
		"run":                 {RunDir(root, "r1"), filepath.Join(root, "coverage", "runs", "r1")},
		"scenario target":     {TargetRunDir(home, "scenario", "demo", root, "r1"), filepath.Join(root, "coverage", "runs", "r1")},
		"target":              {TargetRunDir(home, "package", "api-core", root, "r1"), filepath.Join(home, "state", "test-genie", "targets", "package", "api-core", "coverage", "runs", "r1")},
		"phase-dir":           {RunPhaseResultsDir(root, "r1"), filepath.Join(root, "coverage", "runs", "r1", "phase-results")},
		"ui":                  {RunUISmokeDir(root, "r1"), filepath.Join(root, "coverage", "runs", "r1", "ui-smoke")},
		"pages":               {RunUISmokePagesDir(root, "r1"), filepath.Join(root, "coverage", "runs", "r1", "ui-smoke", "pages")},
		"automation":          {RunAutomationDir(root, "r1"), filepath.Join(root, "coverage", "runs", "r1", "automation")},
		"unit":                {RunUnitDir(root, "r1"), filepath.Join(root, "coverage", "runs", "r1", "unit")},
		"findings":            {RunFindingsArtifactPath(root, "r1"), filepath.Join(root, "coverage", "runs", "r1", "findings.json")},
		"snapshot":            {RunSnapshotPath(root, "r1"), filepath.Join(root, "coverage", "runs", "r1", "run-snapshot.json")},
		"descriptor":          {RunDescriptorSnapshotPath(root, "r1"), filepath.Join(root, "coverage", "runs", "r1", "descriptor-snapshot.json")},
		"catalog":             {RunArtifactCatalogPath(root, "r1"), filepath.Join(root, "coverage", "runs", "r1", "artifact-catalog.json")},
		"index":               {RunsIndexPath(root), filepath.Join(root, "coverage", "runs.index.json")},
		"phase":               {PhaseResultsPath(root, "r1", "unit.json"), filepath.Join(root, "coverage", "runs", "r1", "phase-results", "unit.json")},
		"automation artifact": {AutomationArtifactPath(root, "r1", "flow.json"), filepath.Join(root, "coverage", "runs", "r1", "automation", "flow.json")},
		"unit artifact":       {UnitArtifactPath(root, "r1", "TestX"), filepath.Join(root, "coverage", "runs", "r1", "unit", "TestX")},
		"sync":                {SyncMetadataPath(root), filepath.Join(root, "coverage", "sync", "latest.json")},
		"manual":              {ManualValidationsPath(root), filepath.Join(root, "coverage", "manual-validations", "log.jsonl")},
		"seed":                {SeedStatePath(root), filepath.Join(root, "coverage", "runtime", "seed-state.json")},
		"logs":                {RunLogsDir(root, "r1"), filepath.Join(root, "coverage", "logs", "r1")},
		"latest dir":          {LatestDirPath(root), filepath.Join(root, "coverage", "latest")},
		"latest":              {LatestManifestPath(root), filepath.Join(root, "coverage", "latest", "manifest.json")},
		"cache":               {PhaseCacheDir(root), filepath.Join(root, "phase-cache")},
	}
	for name, pair := range tests {
		if pair[0] != pair[1] {
			t.Errorf("%s = %q, want %q", name, pair[0], pair[1])
		}
	}
	got := VitestRequirementsPaths(root)
	want := []string{filepath.Join(root, "ui", "coverage", "vitest-requirements.json"), filepath.Join(root, "coverage", "vitest-requirements.json"), filepath.Join(root, "test", "coverage", "vitest.json")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("vitest paths = %v, want %v", got, want)
	}
	if got, want := AllCoverageSubdirs(root), []string{
		filepath.Join(root, "coverage", "logs"), filepath.Join(root, "coverage", "latest"), filepath.Join(root, "coverage", "runs"),
		filepath.Join(root, "coverage", "sync"), filepath.Join(root, "coverage", "manual-validations"), filepath.Join(root, "coverage", "runtime"),
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("coverage subdirs = %v, want %v", got, want)
	}
	for name, pair := range map[string][2]string{
		"relative findings":   {RelativeRunFindingsArtifactPath("r1"), filepath.Join("coverage", "runs", "r1", "findings.json")},
		"relative phase":      {RelativePhaseResultsPath("r1", "unit.json"), filepath.Join("coverage", "runs", "r1", "phase-results", "unit.json")},
		"relative automation": {RelativeAutomationArtifactPath("r1", "flow.json"), filepath.Join("coverage", "runs", "r1", "automation", "flow.json")},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s = %q, want %q", name, pair[0], pair[1])
		}
	}
}

func TestGovernedRootsAreOwnerScoped(t *testing.T) {
	storageRoot := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", storageRoot)
	t.Setenv(storage.EnvScenario, "")
	t.Setenv(storage.EnvStorageNamespace, "")
	t.Setenv(storage.EnvVariant, "")

	runs, err := ScenarioRoot("demo")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(storageRoot, "test_runs", "demo"); runs != want {
		t.Fatalf("ScenarioRoot() = %q, want %q", runs, want)
	}
	cache, err := PhaseCacheRoot("demo")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(storageRoot, "cache", "vrooli", "test-genie", "demo"); cache != want {
		t.Fatalf("PhaseCacheRoot() = %q, want %q", cache, want)
	}
}

func TestScenarioRootIsolatesTestGenieShadowFromLive(t *testing.T) {
	storageRoot := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", storageRoot)
	t.Setenv(storage.EnvScenario, TestGenieOwner)
	t.Setenv(storage.EnvStorageNamespace, TestGenieOwner)
	t.Setenv(storage.EnvVariant, "live")

	live, err := ScenarioRoot("demo")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(storageRoot, "test_runs", "demo"); live != want {
		t.Fatalf("live ScenarioRoot() = %q, want %q", live, want)
	}

	t.Setenv(storage.EnvStorageNamespace, "test-genie_shadow")
	t.Setenv(storage.EnvVariant, "shadow")
	shadow, err := ScenarioRoot("demo")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(storageRoot, "test_runs", "test-genie_shadow", "demo"); shadow != want {
		t.Fatalf("shadow ScenarioRoot() = %q, want %q", shadow, want)
	}
	if shadow == live {
		t.Fatal("live and shadow resolved to the same artifact tree")
	}
}

func TestScenarioRootIgnoresImportingConsumersNamespace(t *testing.T) {
	storageRoot := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", storageRoot)
	t.Setenv(storage.EnvScenario, "scenario-structure-suite")
	t.Setenv(storage.EnvStorageNamespace, "scenario-structure-suite_shadow")
	t.Setenv(storage.EnvVariant, "shadow")

	got, err := ScenarioRoot("demo")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(storageRoot, "test_runs", "demo"); got != want {
		t.Fatalf("consumer ScenarioRoot() = %q, want live Test Genie root %q", got, want)
	}
}
