package preview

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	internaldeps "react-component-library/internal/deps"
	internalpreview "react-component-library/internal/preview"
)

func TestBuildImportMapJSONPinsDeclaredDeps(t *testing.T) {
	raw, warnings := buildImportMapJSON(internalpreview.Bundle{
		Dependencies: []internaldeps.Declaration{
			{DepName: "lucide-react", VersionRange: "^0.424.0"},
		},
	})
	require.Empty(t, warnings)
	require.Contains(t, raw, `"react"`)
	require.Contains(t, raw, `"lucide-react": "https://esm.sh/lucide-react@0.424.0?dev"`)
}

func TestRenderHarnessHTMLShowsImportMapDiagnostics(t *testing.T) {
	html := renderHarnessHTML("cmp-1", internalpreview.Bundle{
		JS:         "export default function Demo() { return null }",
		SourcePath: "components/Demo.tsx",
		SHA256:     "sha",
		Dependencies: []internaldeps.Declaration{
			{DepName: "some-lib", VersionRange: "*"},
		},
	})
	require.Contains(t, html, `id="preview-importmap-diagnostics"`)
	require.Contains(t, html, `cannot pin dependency`)
	require.False(t, strings.Contains(html, `some-lib@*`))
}
