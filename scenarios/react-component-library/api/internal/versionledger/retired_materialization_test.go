package versionledger

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransitionRetiredVersionVerifiesAndReclaimsMaterialization(t *testing.T) {
	f := newRetentionFixture(t)
	f.release("Card", "1.0.0", "retired")
	f.release("Card", "2.0.0", "released")
	f.publish()

	repository := NewRepository(f.db, f.libraryRoot)
	_, err := repository.Transition(context.Background(), "Card", "1.0.0", "retired", true)
	require.NoError(t, err)

	original := filepath.Join(f.libraryRoot, "components", "Card", "versions", "1.0.0")
	require.NoDirExists(t, original)
	backup := repository.retiredSourcePath("react-component-library:Card", "1.0.0")
	require.FileExists(t, filepath.Join(backup, "Card.tsx"))
	require.FileExists(t, filepath.Join(backup, "dependencies.json"))

	manifest, err := os.ReadFile(filepath.Join(f.libraryRoot, "components", "Card", "component.json"))
	require.NoError(t, err)
	var document map[string]any
	require.NoError(t, json.Unmarshal(manifest, &document))
	require.Equal(t, []any{"1.0.0"}, document["evictedVersions"])
	var status, presence, lifecycle string
	require.NoError(t, f.db.QueryRow(`SELECT status, presence FROM component_versions WHERE component_id='Card' AND version='1.0.0'`).Scan(&status, &presence))
	require.Equal(t, "retired", status)
	require.Equal(t, "evicted", presence)
	require.NoError(t, f.db.QueryRow(`SELECT lifecycle_state FROM version_ledger WHERE library_id='react-component-library:Card' AND version='1.0.0'`).Scan(&lifecycle))
	require.Equal(t, "retired", lifecycle)
}

func TestTransitionRetiredVersionRejectsMismatchedMaterialization(t *testing.T) {
	f := newRetentionFixture(t)
	f.release("Card", "1.0.0", "retired")
	f.release("Card", "2.0.0", "released")
	f.publish()
	versionDir := filepath.Join(f.libraryRoot, "components", "Card", "versions", "1.0.0")
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "Card.tsx"), []byte("tampered source\n"), 0o644))

	repository := NewRepository(f.db, f.libraryRoot)
	_, err := repository.Transition(context.Background(), "Card", "1.0.0", "retired", true)
	require.ErrorContains(t, err, "mirror mismatch at Card.tsx")
	require.DirExists(t, versionDir)
	require.NoDirExists(t, repository.retiredSourcePath("react-component-library:Card", "1.0.0"))
}
