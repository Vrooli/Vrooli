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
