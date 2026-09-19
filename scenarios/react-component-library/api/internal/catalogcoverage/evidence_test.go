package catalogcoverage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"react-component-library/internal/gates"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	db "github.com/vrooli/api-core/databasetest"
)

func TestEvidenceStorePersistsAndListsRows(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(Schema)))
	store := NewEvidenceStore(database)
	require.NoError(t, store.Save(context.Background(), []GateEvidence{{AssetID: "controls.button", Target: "react-vite", Gate: "visual", Result: "pass", SourceRevision: "rev-1"}}))
	rows, err := store.List(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "rev-1", rows[0].SourceRevision)
}

func TestEvidenceStoreRecoversAfterCanceledSchemaProbe(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(Schema)))
	store := NewEvidenceStore(database)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	err := store.Save(canceled, []GateEvidence{{AssetID: "controls.button", Target: "react-vite", Gate: "visual", Result: "pass", SourceRevision: "rev-canceled"}})
	require.ErrorIs(t, err, context.Canceled)
	// A disconnected matrix must not cache its request-scoped cancellation as
	// a permanent schema error. The next valid request must be able to persist.
	require.NoError(t, store.Save(context.Background(), []GateEvidence{{AssetID: "controls.button", Target: "react-vite", Gate: "visual", Result: "pass", SourceRevision: "rev-recovered"}}))
}

func TestEvidenceFromUnmeasuredResultNeverWritesPass(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "controls")
	componentDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button")
	versionDir := filepath.Join(componentDir, "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.MkdirAll(versionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "button.json"), []byte(`{"kind":"catalog-asset","asset":{"id":"controls.button","kind":"component"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(`{"catalogId":"controls.button","latest":"1.0.0"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "Button.tsx"), []byte("export const Button = () => null;"), 0o644))
	rows, err := EvidenceFromResult(context.Background(), root, GateDefinition{ID: "rtl", Attribution: "attributable", AppliesTo: []string{"component"}}, gates.Result{Status: "unmeasured", InspectedAssets: []string{"controls.button"}}, nil)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "unmeasured", rows[0].Result)
}

func TestEvidenceFromResultSkipsNonCatalogObservations(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "controls")
	componentDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button")
	versionDir := filepath.Join(componentDir, "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.MkdirAll(versionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "button.json"), []byte(`{"kind":"catalog-asset","asset":{"id":"controls.button","kind":"component"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(`{"catalogId":"controls.button","latest":"1.0.0"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "Button.tsx"), []byte("export const Button = () => null;"), 0o644))
	rows, err := EvidenceFromResult(context.Background(), root, GateDefinition{ID: "types", Attribution: "attributable", AppliesTo: []string{"component"}}, gates.Result{
		InspectedAssets: []string{"workbench.conformance", "__corpus__.dependency-rank"},
	}, nil)
	if err != nil {
		t.Fatalf("pseudo-asset evidence: %v", err)
	}
	for _, row := range rows {
		if row.AssetID == "workbench.conformance" || strings.HasPrefix(row.AssetID, "__corpus__") {
			t.Fatalf("pseudo-asset row persisted: %+v", row)
		}
	}
}

func TestMergedEvidenceTreatsStalePersistedRowsAsAbsent(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "controls")
	componentDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button")
	versionDir := filepath.Join(componentDir, "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.MkdirAll(versionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "button.json"), []byte(`{"kind":"catalog-asset","asset":{"id":"controls.button","name":"Button","kind":"component","targets":["react-vite"],"target":{"maturity":"verified"}},"api":{"variants":{}}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json"), []byte(`{"gates":[{"id":"api","rung":"implemented","blocking":true,"appliesTo":["component"]}]}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(`{"catalogId":"controls.button","latest":"1.0.0"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "Button.tsx"), []byte(`export const Button = () => null;`), 0o644))
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(Schema)))
	// MergedEvidence runs the immutable released-version gate against the
	// same domain database used by production. This fixture has no component
	// index rows, but it still declares the table so the gate is measured
	// rather than failing while trying to open a legacy database path.
	_, err := database.ExecContext(context.Background(), `CREATE TABLE component_versions (status TEXT NOT NULL, source_path TEXT NOT NULL, content_sha256 TEXT NOT NULL)`)
	require.NoError(t, err)
	store := NewEvidenceStore(database)
	require.NoError(t, store.Save(context.Background(), []GateEvidence{{AssetID: "controls.button", Target: "react-vite", Gate: "visual", Result: "pass", SourceRevision: "stale"}}))
	evidence, err := MergedEvidence(context.Background(), root, store)
	require.NoError(t, err)
	for _, item := range evidence {
		if item.Gate == "visual" {
			t.Fatalf("stale visual evidence survived merge: %+v", item)
		}
	}
}

func TestCurrentRevisionAcceptsRepositoryAndLibraryRoots(t *testing.T) {
	repoRoot := t.TempDir()
	scenarioRoot := filepath.Join(repoRoot, "scenarios", "react-component-library")
	assetDir := filepath.Join(scenarioRoot, "catalog", "assets", "controls")
	componentDir := filepath.Join(scenarioRoot, "library", "components", "Button")
	versionDir := filepath.Join(componentDir, "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.MkdirAll(versionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "button.json"), []byte(`{"kind":"catalog-asset","asset":{"id":"controls.button"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(`{"catalogId":"controls.button","latest":"1.0.0"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "Button.tsx"), []byte("export const Button = () => null;"), 0o644))
	fromRepository, err := CurrentRevision(repoRoot, "controls.button")
	require.NoError(t, err)
	fromLibrary, err := CurrentRevision(filepath.Join(scenarioRoot, "library"), "controls.button")
	require.NoError(t, err)
	require.Equal(t, fromRepository, fromLibrary)
}

func TestCurrentRevisionForVersionChangesOnlyTheRequestedVersion(t *testing.T) {
	root := t.TempDir()
	scenarioRoot := filepath.Join(root, "scenarios", "react-component-library")
	assetDir := filepath.Join(scenarioRoot, "catalog", "assets", "controls")
	componentDir := filepath.Join(scenarioRoot, "library", "components", "Button")
	for _, version := range []string{"1.0.0", "1.1.0"} {
		require.NoError(t, os.MkdirAll(filepath.Join(componentDir, "versions", version), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(componentDir, "versions", version, "Button.tsx"), []byte("export const Button = '"+version+"';"), 0o644))
	}
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "button.json"), []byte(`{"asset":{"id":"controls.button"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(`{"catalogId":"controls.button","libraryId":"rcl:Button","latest":"1.1.0"}`), 0o644))
	first, err := CurrentRevisionForVersion(root, "rcl:Button", "1.0.0")
	require.NoError(t, err)
	second, err := CurrentRevisionForVersion(root, "rcl:Button", "1.1.0")
	require.NoError(t, err)
	require.NotEqual(t, first, second)
	changed := filepath.Join(componentDir, "versions", "1.0.0", "Button.tsx")
	require.NoError(t, os.WriteFile(changed, []byte("export const Button = 'changed';"), 0o644))
	firstChanged, err := CurrentRevisionForVersion(root, "rcl:Button", "1.0.0")
	require.NoError(t, err)
	secondUnchanged, err := CurrentRevisionForVersion(root, "rcl:Button", "1.1.0")
	require.NoError(t, err)
	require.NotEqual(t, first, firstChanged)
	require.Equal(t, second, secondUnchanged)
}

func TestCurrentRevisionForVersionIgnoresLockResolutionTimestamp(t *testing.T) {
	root := t.TempDir()
	writeAsset(t, root, "button", "Button", "")
	lock := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "button", "versions", "1.0.0", "dependencies.json")
	require.NoError(t, os.WriteFile(lock, []byte(`{"schemaVersion":2,"libraryId":"rcl:Button","version":"1.0.0","resolvedAt":"2026-01-01T00:00:00Z","dependencies":[]}`), 0o644))
	first, err := CurrentRevisionForVersion(root, "rcl:Button", "1.0.0")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(lock, []byte(`{"schemaVersion":2,"libraryId":"rcl:Button","version":"1.0.0","resolvedAt":"2099-01-01T00:00:00Z","dependencies":[]}`), 0o644))
	second, err := CurrentRevisionForVersion(root, "rcl:Button", "1.0.0")
	require.NoError(t, err)
	require.Equal(t, first, second, "generator bookkeeping must not invalidate evidence")
}

func TestCurrentRevisionForVersionChangesWhenStoryContractChanges(t *testing.T) {
	root := t.TempDir()
	writeAsset(t, root, "button", "Button", "")
	story := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "button", "versions", "1.0.0", "story.json")
	require.NoError(t, os.WriteFile(story, []byte(`{"schemaVersion":5,"stories":[{"id":"default"}]}`), 0o644))
	first, err := CurrentRevisionForVersion(root, "rcl:Button", "1.0.0")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(story, []byte(`{"schemaVersion":5,"stories":[{"id":"updated"}]}`), 0o644))
	second, err := CurrentRevisionForVersion(root, "rcl:Button", "1.0.0")
	require.NoError(t, err)
	require.NotEqual(t, first, second, "story contract changes must invalidate browser evidence")
}

func writeAsset(t *testing.T, root, name, libraryName, dependencies string) {
	t.Helper()
	scenarioRoot := filepath.Join(root, "scenarios", "react-component-library")
	assetDir := filepath.Join(scenarioRoot, "catalog", "assets", "controls")
	componentDir := filepath.Join(scenarioRoot, "library", "components", name)
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(componentDir, "versions", "1.0.0"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, name+".json"),
		[]byte(`{"kind":"catalog-asset","asset":{"id":"controls.`+name+`","kind":"component"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(componentDir, "component.json"),
		[]byte(`{"catalogId":"controls.`+name+`","libraryId":"rcl:`+libraryName+`","latest":"1.0.0"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(componentDir, "versions", "1.0.0", libraryName+".tsx"),
		[]byte("export const "+libraryName+" = () => null;"), 0o644))
	// Dependency edges live in the generated per-version lock, which is where
	// the revision index reads them from.
	require.NoError(t, os.WriteFile(filepath.Join(componentDir, "versions", "1.0.0", "dependencies.json"),
		[]byte(`{"schemaVersion":1,"libraryId":"rcl:`+libraryName+`","version":"1.0.0","dependencies":[`+dependencies+`]}`), 0o644))
}

// The manifest glob in the original revision hash was one directory short of
// where manifests live, so neither the manifest nor any version source reached
// the digest and an edited component kept looking current forever.
func TestRevisionTracksManifestAndVersionSource(t *testing.T) {
	root := t.TempDir()
	writeAsset(t, root, "button", "Button", "")
	before, err := CurrentRevision(root, "controls.button")
	require.NoError(t, err)

	source := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "button", "versions", "1.0.0", "Button.tsx")
	require.NoError(t, os.WriteFile(source, []byte("export const Button = () => <span />;"), 0o644))
	afterSource, err := CurrentRevision(root, "controls.button")
	require.NoError(t, err)
	require.NotEqual(t, before, afterSource, "editing version source must change the asset revision")

	manifest := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "button", "component.json")
	require.NoError(t, os.WriteFile(manifest, []byte(`{"catalogId":"controls.button","libraryId":"rcl:Button","latest":"1.0.0","kind":"control"}`), 0o644))
	afterManifest, err := CurrentRevision(root, "controls.button")
	require.NoError(t, err)
	require.NotEqual(t, afterSource, afterManifest, "editing the manifest must change the asset revision")
}

// Adoption copies a transitive closure, so a dependent's cached verdict cannot
// outlive a change to what it is built on.
func TestDependencyEditInvalidatesDependents(t *testing.T) {
	root := t.TempDir()
	writeAsset(t, root, "stack", "Stack", "")
	writeAsset(t, root, "card", "Card", `{"libraryId":"rcl:Stack","version":"1.0.0","rank":3}`)

	before, err := CurrentRevision(root, "controls.card")
	require.NoError(t, err)
	independent, err := CurrentRevision(root, "controls.stack")
	require.NoError(t, err)

	stackSource := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "stack", "versions", "1.0.0", "Stack.tsx")
	require.NoError(t, os.WriteFile(stackSource, []byte("export const Stack = () => <div />;"), 0o644))

	afterDependent, err := CurrentRevision(root, "controls.card")
	require.NoError(t, err)
	afterDependency, err := CurrentRevision(root, "controls.stack")
	require.NoError(t, err)
	require.NotEqual(t, before, afterDependent, "a dependency change must invalidate its dependents")
	require.NotEqual(t, independent, afterDependency)
}

func TestMajorLineIsolation(t *testing.T) {
	root := t.TempDir()
	writeAsset(t, root, "leaf", "Leaf", "")
	leafRoot := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "leaf")
	secondVersion := filepath.Join(leafRoot, "versions", "2.0.0")
	require.NoError(t, os.MkdirAll(secondVersion, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(secondVersion, "Leaf.tsx"), []byte("export const Leaf = '2';"), 0o644))
	writeAsset(t, root, "mid", "Mid", `{"libraryId":"rcl:Leaf","major":1,"observed":"1.0.0","rank":3}`)

	before, err := CurrentRevisionForVersion(root, "rcl:Mid", "1.0.0")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(secondVersion, "Leaf.tsx"), []byte("export const Leaf = '2 changed';"), 0o644))
	after, err := CurrentRevisionForVersion(root, "rcl:Mid", "1.0.0")
	require.NoError(t, err)
	require.Equal(t, before, after, "a change on an unimported major line must not invalidate the dependent")
}

func TestStaleAssetsByGateReportsOnlyTheChangedAsset(t *testing.T) {
	root := t.TempDir()
	writeAsset(t, root, "button", "Button", "")
	writeAsset(t, root, "card", "Card", "")
	revisions, err := BuildRevisionIndex(root)
	require.NoError(t, err)
	definitions := []GateDefinition{
		{ID: "types", Attribution: "attributable", AppliesTo: []string{"component"}},
		{ID: "visual", Attribution: "attributable", AppliesTo: []string{"component"}},
		{ID: "tokens", Attribution: "corpus", AppliesTo: []string{"component"}},
	}
	persisted := []GateEvidence{
		{AssetID: "controls.button", Gate: "types", SourceRevision: revisions["controls.button"], RecordedAt: "2026-01-02T00:00:00Z"},
		{AssetID: "controls.card", Gate: "types", SourceRevision: revisions["controls.card"], RecordedAt: "2026-01-02T00:00:00Z"},
		{AssetID: "controls.button", Gate: "visual", SourceRevision: revisions["controls.button"], RecordedAt: "2026-01-02T00:00:00Z"},
	}

	stale := staleAssetsByGate(root, persisted, definitions, revisions)
	require.Empty(t, stale["types"], "every applicable asset is current, so the runner can be skipped")
	require.Equal(t, map[string]bool{"controls.card": true}, stale["visual"],
		"only the asset without current evidence is stale — not the whole corpus")
	_, corpusTracked := stale["tokens"]
	require.False(t, corpusTracked, "corpus gates are not attributable and always recompute")

	// Editing one component must make that component stale and leave the other current.
	source := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "button", "versions", "1.0.0", "Button.tsx")
	require.NoError(t, os.WriteFile(source, []byte("export const Button = () => <b />;"), 0o644))
	revisions, err = BuildRevisionIndex(root)
	require.NoError(t, err)
	stale = staleAssetsByGate(root, persisted, definitions, revisions)
	require.Equal(t, map[string]bool{"controls.button": true}, stale["types"],
		"one edited component must not invalidate the corpus")
}

func TestRecomputeEvidenceDoesNotFabricateTypesPass(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "controls")
	componentDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button")
	versionDir := filepath.Join(componentDir, "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.MkdirAll(versionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "button.json"), []byte(`{"kind":"catalog-asset","asset":{"id":"controls.button","name":"Button","kind":"component","targets":["react-vite"]},"api":{}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json"), []byte(`{"gates":[{"id":"types","rung":"scaffolded","blocking":true,"appliesTo":["component"]}]}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(`{"catalogId":"controls.button","latest":"1.0.0"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "Button.tsx"), []byte(`export const Button = () => null;`), 0o644))

	evidence, err := RecomputeEvidence(root)
	require.NoError(t, err)
	require.Len(t, evidence, 1)
	require.Equal(t, "types", evidence[0].Gate)
	// The fixture repository has no deterministic type runner boundary, so
	// calibration quarantines the gate instead of turning a runner failure into
	// either a pass or a durable corpus fail.
	require.Equal(t, "unmeasured", evidence[0].Result)
}

func TestDeriveExperienceEvidenceRequiresDeclaredCaptureQuality(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "catalog", "assets", "controls")
	componentDir := filepath.Join(root, "library", "components", "Button")
	versionDir := filepath.Join(componentDir, "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.MkdirAll(versionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "button.json"), []byte(`{"kind":"catalog-asset","asset":{"id":"controls.button"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(`{"catalogId":"controls.button","latest":"1.0.0"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "Button.tsx"), []byte("export const Button = () => null;"), 0o644))
	definitions := []GateDefinition{
		{ID: "accessibility", ExperienceClaimTypes: []string{"element-present"}},
		{ID: "responsive", ExperienceClaimTypes: []string{"visible-without-scroll"}, ExperienceMinimumViewports: 2},
		{ID: "visual", ExperienceClaimTypes: []string{"spacing"}, ExperienceRequiresCapture: true},
	}
	evidence, err := deriveExperienceEvidence(root, []ExperienceCapture{
		{AssetID: "controls.button", Target: "react-vite", ClaimType: "element-present", Verdict: "passed"},
		{AssetID: "controls.button", Target: "react-vite", ClaimType: "visible-without-scroll", Verdict: "passed", Viewport: "desktop"},
		{AssetID: "controls.button", Target: "react-vite", ClaimType: "spacing", Verdict: "passed"},
	}, definitions)
	require.NoError(t, err)
	require.Len(t, evidence, 3)
	results := map[string]string{}
	for _, item := range evidence {
		results[item.Gate] = item.Result
	}
	require.Equal(t, "pass", results["accessibility"])
	require.Equal(t, "skipped", results["responsive"])
	require.Equal(t, "skipped", results["visual"])

	evidence, err = deriveExperienceEvidence(root, []ExperienceCapture{
		{AssetID: "controls.button", Target: "react-vite", ClaimType: "visible-without-scroll", Verdict: "passed", Viewport: "desktop"},
		{AssetID: "controls.button", Target: "react-vite", ClaimType: "visible-without-scroll", Verdict: "passed", Viewport: "mobile"},
		{AssetID: "controls.button", Target: "react-vite", ClaimType: "spacing", Verdict: "passed", CaptureRef: "capture-1"},
	}, definitions)
	require.NoError(t, err)
	results = map[string]string{}
	for _, item := range evidence {
		results[item.Gate] = item.Result
	}
	require.Equal(t, "pass", results["responsive"])
	require.Equal(t, "pass", results["visual"])
}

// The whole point of the cache is that only recomputation frequency changes.
// A warm pass that reuses persisted verdicts must produce the same evidence a
// cold pass does, asset for asset and gate for gate — including the
// "unmeasured" rows a catalog asset with no implementation receives, which an
// earlier draft of this change dropped on warm passes and would have quietly
// moved every reported rung.
func TestWarmPassProducesIdenticalEvidenceToColdPass(t *testing.T) {
	root := t.TempDir()
	scenarioRoot := filepath.Join(root, "scenarios", "react-component-library")
	writeAsset(t, root, "stack", "Stack", "")
	writeAsset(t, root, "card", "Card", `{"libraryId":"rcl:Stack","version":"1.0.0","rank":3}`)
	// A declared asset nobody has built yet.
	require.NoError(t, os.WriteFile(filepath.Join(scenarioRoot, "catalog", "assets", "controls", "ghost.json"),
		[]byte(`{"kind":"catalog-asset","asset":{"id":"controls.ghost","kind":"component"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(scenarioRoot, "catalog", "config.json"),
		[]byte(`{"gates":[{"id":"rtl","rung":"verified","blocking":true,"attribution":"attributable","appliesTo":["component"]}]}`), 0o644))

	key := func(rows []GateEvidence) map[string]string {
		out := map[string]string{}
		for _, row := range rows {
			out[row.AssetID+"\x00"+row.Gate] = row.Result
		}
		return out
	}

	revisions, err := BuildRevisionIndex(root)
	require.NoError(t, err)
	cold, err := recomputeEvidenceWithSkip(root, nil, nil, revisions)
	require.NoError(t, err)
	require.NotEmpty(t, cold)

	definitions, err := LoadGateDefinitions(filepath.Join(scenarioRoot, "catalog", "config.json"))
	require.NoError(t, err)
	stale := staleAssetsByGate(root, cold, definitions, revisions)
	require.Empty(t, stale["rtl"], "every asset was just measured, so nothing is stale")

	warm, err := recomputeEvidenceWithSkip(root, nil, stale, revisions)
	require.NoError(t, err)
	require.Empty(t, warm, "a fully warm pass recomputes nothing")

	// The served result is the warm recompute merged with the persisted rows.
	merged := append([]GateEvidence{}, warm...)
	for _, row := range cold {
		if revisions[row.AssetID] == row.SourceRevision {
			merged = append(merged, row)
		}
	}
	require.Equal(t, key(cold), key(merged), "warm and cold must agree on every asset and gate")
}

// The evidence mapper binds a finding to an asset by exact id, so a finding
// with no AssetID matches nothing and leaves every asset on the default "pass".
// That is why an unattributable gate failure has to travel as a RunnerError,
// which the mapper fails closed on, rather than as a corpus finding.
func TestCorpusFindingMatchesNoAssetSoRunnerErrorCarriesTheFailure(t *testing.T) {
	corpus := []gates.Finding{{Code: "catalog.types_failed", AssetID: "", Message: "catalog conformance failed"}}
	require.False(t, hasFinding(corpus, "controls.button", "Button", "types"),
		"a corpus finding must not be mistaken for evidence about an individual asset")

	attributed := []gates.Finding{{Code: "catalog.types_failed", AssetID: "IconButton", Message: "failed"}}
	require.True(t, hasFinding(attributed, "controls.icon-button", "IconButton", "types"),
		"an attributed finding matches by implementation name")
	require.False(t, hasFinding(attributed, "controls.button", "Button", "types"),
		"and only the named asset")
}
