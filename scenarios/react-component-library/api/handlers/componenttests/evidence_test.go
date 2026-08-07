package componenttests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"react-component-library/internal/catalogcoverage"
	domain "react-component-library/internal/componenttests"
	"react-component-library/internal/testutil/db"
)

func TestRecordContractEvidencePersistsOnlyContractGates(t *testing.T) {
	scenarioRoot := t.TempDir()
	libraryRoot := filepath.Join(scenarioRoot, "library")
	assetDir := filepath.Join(scenarioRoot, "catalog", "assets", "controls")
	componentDir := filepath.Join(libraryRoot, "components", "Button")
	versionDir := filepath.Join(componentDir, "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.MkdirAll(versionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "button.json"), []byte(`{"kind":"catalog-asset","asset":{"id":"controls.button","kind":"component","targets":["react-vite"]}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(`{"catalogId":"controls.button","latest":"1.0.0"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "Button.tsx"), []byte("export const Button = () => null;"), 0o644))

	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(catalogcoverage.Schema)))
	store := catalogcoverage.NewEvidenceStore(database)
	report := domain.Report{Results: []domain.Result{{Stage: domain.StageDeclared, AssetLibraryID: "react-component-library:Button", Version: "1.0.0", Verdict: domain.VerdictPassed}}}
	require.NoError(t, recordContractEvidence(context.Background(), store, libraryRoot, report))
	rows, err := store.List(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "controls.button", rows[0].AssetID)
	require.Equal(t, "interaction", rows[0].Gate)
	require.Equal(t, "pass", rows[0].Result)
	require.Equal(t, "unit", rows[1].Gate)
	require.Equal(t, "pass", rows[1].Result)
}
