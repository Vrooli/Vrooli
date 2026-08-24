package components

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCatalogFrameAssetsFromDirBuildsStableFrameProjection(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "assets", "navigation")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "page.json"), []byte(`{
  "kind": "catalog-asset",
  "asset": {"id": "navigation.page", "kind": "navigation", "targets": ["react-vite"]},
  "regions": [
    {"id": "content", "accepts": "content"},
    {"id": "navigation", "accepts": "navigation"}
  ],
  "expects": [{"capability": "router-adapter", "typeArguments": ["Route"]}]
}`), 0o644))

	assets, err := CatalogFrameAssetsFromDir(root)
	require.NoError(t, err)
	require.Len(t, assets, 1)
	require.Equal(t, "navigation.page", assets[0].ID)
	require.Equal(t, []string{"content", "navigation"}, assets[0].Regions)
	require.Equal(t, map[string]string{"content": "content", "navigation": "navigation"}, assets[0].RegionCapabilities)
	require.Equal(t, []CatalogFramePort{{Capability: "router-adapter", TypeArguments: []string{"Route"}}}, assets[0].Expects)
}

func TestCatalogFrameRegistryIgnoresNonCatalogDocuments(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "assets", "misc")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "ignored.json"), []byte(`{"kind":"fixture"}`), 0o644))

	registry, err := CatalogFrameRegistryFromDir(root)
	require.NoError(t, err)
	_, ok := registry.LookupCatalogFrameAsset("ignored")
	require.False(t, ok)
}
