package adoptions

import (
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/components"
)

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
