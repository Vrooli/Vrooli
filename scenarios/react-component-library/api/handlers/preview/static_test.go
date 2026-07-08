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

func TestBuildImportMapJSONHonorsDeclaredReactRange(t *testing.T) {
	raw, warnings := buildImportMapJSON(internalpreview.Bundle{
		Dependencies: []internaldeps.Declaration{
			{DepName: "react", VersionRange: "^17.0.0"},
			{DepName: "react-dom", VersionRange: "^17.0.0"},
		},
	})
	require.Empty(t, warnings)
	require.Contains(t, raw, `"react": "https://esm.sh/react@17.0.2?dev"`)
	require.Contains(t, raw, `"react/jsx-runtime": "https://esm.sh/react@17.0.2/jsx-runtime?dev"`)
	require.Contains(t, raw, `"react-dom/client": "https://esm.sh/react-dom@17.0.2/client?dev"`)
	require.NotContains(t, raw, `react@18.3.1`)
}

func TestBuildImportMapJSONWarnsOnUnresolvableReactRange(t *testing.T) {
	raw, warnings := buildImportMapJSON(internalpreview.Bundle{
		Dependencies: []internaldeps.Declaration{
			{DepName: "react", VersionRange: "workspace:*"},
		},
	})
	require.NotEmpty(t, warnings)
	require.Contains(t, warnings[0], `cannot pin dependency "react"`)
	require.Contains(t, raw, `"react": "https://esm.sh/react@18.3.1?dev"`)
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

func TestRenderHarnessHTMLCanShowRuntimeImportFailure(t *testing.T) {
	html := renderHarnessHTML("cmp-1", internalpreview.Bundle{
		JS:         "export default function Demo() { return null }",
		SourcePath: "components/Demo.tsx",
		SHA256:     "sha",
	})
	require.NotContains(t, html, `import { createRoot } from "react-dom/client";`)
	require.Contains(t, html, `try {`)
	require.Contains(t, html, `import("react-dom/client")`)
	require.Contains(t, html, `import(componentModuleURL)`)
	require.Contains(t, html, `preview: render failed`)
	require.Contains(t, html, `errEl.hidden = false`)
}
