package adoptions

import (
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"

	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
)

func TestListScenarios_ReturnsSortedDirectories(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "zeta"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "alpha"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("not a scenario"), 0o644))

	handler := NewConnectHandler(Deps{ScenariosRoot: root})
	response, err := handler.ListScenarios(t.Context(), connect.NewRequest(&adoptionsv1.ListScenariosRequest{}))
	require.NoError(t, err)
	require.Len(t, response.Msg.Scenarios, 2)
	require.Equal(t, "alpha", response.Msg.Scenarios[0].GetName())
	require.Equal(t, "alpha", response.Msg.Scenarios[0].GetDisplayName())
	require.Equal(t, "zeta", response.Msg.Scenarios[1].GetName())
	require.Equal(t, "zeta", response.Msg.Scenarios[1].GetDisplayName())
}

func TestListScenarios_ReturnsEmptyCollectionForEmptyRoot(t *testing.T) {
	handler := NewConnectHandler(Deps{ScenariosRoot: t.TempDir()})
	response, err := handler.ListScenarios(t.Context(), connect.NewRequest(&adoptionsv1.ListScenariosRequest{}))
	require.NoError(t, err)
	require.Empty(t, response.Msg.Scenarios)
}

func TestSuggestionMatch_UsesComponentIdentityNotGenericTags(t *testing.T) {
	component := components.Component{Slug: "DataTable", DisplayName: "Data Table", Tags: []string{"data", "surface"}}
	require.Equal(t, "", suggestionMatch(component, "Data Renderers", "ui/DataRenderers.tsx"))
	require.Equal(t, "DataTable", suggestionMatch(component, "DataTable", "ui/DataTable.tsx"))
}

func TestSuggestionMatch_UsesShellSurfaceNameWithoutBroadTagMatching(t *testing.T) {
	component := components.Component{Slug: "DrawerShell", DisplayName: "Drawer Shell", Tags: []string{"overlay", "layout"}}
	require.Equal(t, "drawer", suggestionMatch(component, "Async operation drawer", "ui/components/AsyncOperationDrawer.tsx"))
	require.Equal(t, "", suggestionMatch(component, "Overlay layout", "ui/components/OverlayLayout.tsx"))
}

func TestActiveSuggestionComponentsExcludesDeprecatedCatalogIdentity(t *testing.T) {
	candidates := []components.Component{
		{ID: "old", CatalogID: "navigation.top-bar", DisplayName: "Renamed toolbar"},
		{ID: "active", CatalogID: "navigation.app-shell", DisplayName: "AppShell"},
		{ID: "different", CatalogID: "product.top-bar", DisplayName: "TopBar"},
	}
	assets := []catalogcoverage.Asset{
		{ID: "navigation.top-bar", Maturity: "deprecated"},
		{ID: "navigation.app-shell", Maturity: "production-ready"},
		{ID: "product.top-bar", Maturity: "production-ready"},
	}
	result := activeSuggestionComponents(candidates, assets)
	require.Len(t, result, 2)
	require.Equal(t, "active", result[0].ID)
	require.Equal(t, "different", result[1].ID)
}

func TestActiveSuggestionComponentsRetiredPageTierIsNotRecommended(t *testing.T) {
	assets, err := catalogcoverage.LoadCatalogContext(t.Context(), filepath.Join("..", "..", "..", "catalog"))
	require.NoError(t, err)
	retired := []string{"navigation.top-bar", "react-component-library:PageFrame", "templates.dashboard-page", "templates.detail-page", "templates.collection-page"}
	candidates := []components.Component{{ID: "shell", CatalogID: "navigation.app-shell"}, {ID: "page", CatalogID: "navigation.page"}}
	for _, id := range retired {
		candidates = append(candidates, components.Component{ID: id, CatalogID: id})
	}
	result := activeSuggestionComponents(candidates, assets)
	require.Equal(t, []components.Component{{ID: "shell", CatalogID: "navigation.app-shell"}, {ID: "page", CatalogID: "navigation.page"}}, result)
}
