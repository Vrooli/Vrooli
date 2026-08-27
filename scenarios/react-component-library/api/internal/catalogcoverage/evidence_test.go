package catalogcoverage

import (
	"context"
	"os"
	"path/filepath"
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

func TestFreshTypesEvidenceCompleteSkipsOnlyFullyWarmCorpus(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "controls")
	componentDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button")
	versionDir := filepath.Join(componentDir, "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.MkdirAll(versionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "button.json"), []byte(`{"kind":"catalog-asset","asset":{"id":"controls.button","kind":"component"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(`{"catalogId":"controls.button","latest":"1.0.0"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "Button.tsx"), []byte("export const Button = () => null;"), 0o644))
	revision, err := CurrentRevision(root, "controls.button")
	require.NoError(t, err)

	require.False(t, freshTypesEvidenceComplete(root, nil), "cold evidence must retain the toolchain path")
	require.True(t, freshTypesEvidenceComplete(root, []GateEvidence{{AssetID: "controls.button", Gate: "types", Result: "pass", SourceRevision: revision}}))
	require.False(t, freshTypesEvidenceComplete(root, []GateEvidence{{AssetID: "controls.button", Gate: "types", Result: "pass", SourceRevision: "stale"}}))
}

func TestFreshAttributableGatesRequireEveryBuiltAssetAtCurrentRevision(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "controls")
	for _, name := range []string{"button", "card"} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, "scenarios", "react-component-library", "library", "components", name, "versions", "1.0.0"), 0o755))
		require.NoError(t, os.MkdirAll(assetDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(assetDir, name+".json"), []byte(`{"kind":"catalog-asset","asset":{"id":"controls.`+name+`","kind":"component"}}`), 0o644))
		libraryName := map[string]string{"button": "Button", "card": "Card"}[name]
		componentDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", name)
		require.NoError(t, os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(`{"catalogId":"controls.`+name+`","latest":"1.0.0"}`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(componentDir, "versions", "1.0.0", libraryName+".tsx"), []byte("export const "+libraryName+" = () => null;"), 0o644))
	}
	buttonRevision, err := CurrentRevision(root, "controls.button")
	require.NoError(t, err)
	cardRevision, err := CurrentRevision(root, "controls.card")
	require.NoError(t, err)
	definitions := []GateDefinition{{ID: "types", Attribution: "attributable", AppliesTo: []string{"component"}}, {ID: "visual", Attribution: "attributable", AppliesTo: []string{"component"}}}
	persisted := []GateEvidence{
		{AssetID: "controls.button", Gate: "types", SourceRevision: buttonRevision, RecordedAt: "2026-01-02T00:00:00Z"},
		{AssetID: "controls.card", Gate: "types", SourceRevision: cardRevision, RecordedAt: "2026-01-02T00:00:00Z"},
		{AssetID: "controls.button", Gate: "visual", SourceRevision: buttonRevision, RecordedAt: "2026-01-02T00:00:00Z"},
	}
	fresh := freshAttributableGates(root, persisted, definitions)
	require.True(t, fresh["types"])
	require.False(t, fresh["visual"], "a gate is warm only when every applicable built asset has current evidence")
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
