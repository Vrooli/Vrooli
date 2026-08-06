package catalogcoverage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	"react-component-library/internal/testutil/db"
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
