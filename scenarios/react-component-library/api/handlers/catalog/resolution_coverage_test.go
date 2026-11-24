package catalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"react-component-library/internal/components"
	componentmocks "react-component-library/internal/components/mocks"
)

func TestResolveAssetSupportsLibraryAndCatalogIdentities(t *testing.T) {
	ctx := context.Background()
	repo := componentmocks.NewFakeRepository()
	_, err := repo.Upsert(ctx, components.UpsertInput{
		CatalogID: "controls.button", LibraryID: "react-component-library:Button", DisplayName: "Button",
	})
	require.NoError(t, err)
	assets := components.NewService(repo)

	byLibrary, err := resolveAsset(ctx, assets, "react-component-library:Button")
	require.NoError(t, err)
	require.Equal(t, "controls.button", byLibrary.CatalogID)
	byCatalog, err := resolveAsset(ctx, assets, "controls.button")
	require.NoError(t, err)
	require.Equal(t, "react-component-library:Button", byCatalog.LibraryID)
	_, err = resolveAsset(ctx, assets, "controls.missing")
	require.ErrorContains(t, err, `asset "controls.missing" is not indexed`)
}

func TestResolveRevisionLibraryIDFallsBackToAuthoredImplementation(t *testing.T) {
	ctx := context.Background()
	repo := componentmocks.NewFakeRepository()
	assets := components.NewService(repo)
	root := t.TempDir()
	manifest := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button", "component.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(manifest), 0o700))
	require.NoError(t, os.WriteFile(manifest, []byte(`{"catalogId":"controls.button","libraryId":"react-component-library:Button","latest":"1.0.0"}`), 0o600))

	resolved := resolveRevisionLibraryID(ctx, assets, root, "controls.button")
	require.Equal(t, "react-component-library:Button", resolved)
	unknown := resolveRevisionLibraryID(ctx, assets, root, "controls.missing")
	require.Equal(t, "controls.missing", unknown)
}
