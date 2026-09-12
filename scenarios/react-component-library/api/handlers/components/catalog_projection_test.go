package components

import (
	"github.com/stretchr/testify/require"
	"react-component-library/internal/assetgraph"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
	"testing"
)

func TestCatalogKindFilterUsesSemanticKindBeforeLimit(t *testing.T) {
	index, err := assetgraph.Build([]catalogcoverage.Asset{
		{ID: "hooks.viewport", Kind: "runtime-hook"},
		{ID: "services.renderer", Kind: "runtime-service"},
		{ID: "controls.button", Kind: "primitive"},
	})
	require.NoError(t, err)
	rows := []components.Component{
		{CatalogID: "hooks.viewport", AssetKind: components.AssetKindComponent},
		{CatalogID: "services.renderer", AssetKind: components.AssetKindComponent},
		{CatalogID: "controls.button", AssetKind: components.AssetKindComponent},
		{LibraryID: "unclassified", AssetKind: components.AssetKindComponent},
	}
	h := &connectHandler{}
	for _, tc := range []struct{ kind, id string }{
		{"primitive", "controls.button"}, {"runtime-hook", "hooks.viewport"}, {"runtime-service", "services.renderer"}, {"", ""},
	} {
		result := h.projectCatalogComponents(index, rows, []string{tc.kind}, 1)
		require.Len(t, result, 1, tc.kind)
		require.Equal(t, tc.id, result[0].CatalogID)
		require.Equal(t, tc.kind, result[0].CatalogKind)
	}
	require.Len(t, h.projectCatalogComponents(index, rows, nil, 2), 2)
}
