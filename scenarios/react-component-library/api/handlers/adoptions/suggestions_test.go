package adoptions

import (
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"

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
