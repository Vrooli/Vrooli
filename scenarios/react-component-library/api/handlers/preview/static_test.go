package preview

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	internaldeps "react-component-library/internal/deps"
	internalpreview "react-component-library/internal/preview"
)

func TestBuildImportMapJSONPinsDeclaredDeps(t *testing.T) {
	withPackageRuntimeCandidates(t, func(name string) []string {
		if name == "lucide-react" {
			return []string{"0.424.0"}
		}
		return nil
	})
	raw, warnings := buildImportMapJSON(internalpreview.Bundle{
		Dependencies: []internaldeps.Declaration{
			{DepName: "lucide-react", VersionRange: "^0.424.0"},
		},
	})
	require.Empty(t, warnings)
	require.Contains(t, raw, `"react"`)
	require.Contains(t, raw, `"lucide-react": "/preview/runtime/npm/lucide-react@0.424.0/index.js"`)
	require.Contains(t, raw, `"/preview/runtime/react@18.3.1/index.js"`)
	require.NotContains(t, raw, `esm.sh`)
}

func TestBuildImportMapJSONUsesResolvedDependencyVersionPerBundle(t *testing.T) {
	withPackageRuntimeCandidates(t, func(name string) []string {
		if name == "date-fns" {
			return []string{"2.30.0", "3.6.0"}
		}
		return nil
	})
	v2, warnings := buildImportMapJSON(internalpreview.Bundle{
		Dependencies: []internaldeps.Declaration{{DepName: "date-fns", VersionRange: "^2.0.0"}},
	})
	require.Empty(t, warnings)
	require.Contains(t, v2, `"date-fns": "/preview/runtime/npm/date-fns@2.30.0/index.js"`)

	v3, warnings := buildImportMapJSON(internalpreview.Bundle{
		Dependencies: []internaldeps.Declaration{{DepName: "date-fns", VersionRange: "^3.0.0"}},
	})
	require.Empty(t, warnings)
	require.Contains(t, v3, `"date-fns": "/preview/runtime/npm/date-fns@3.6.0/index.js"`)
	require.NotEqual(t, v2, v3)
}

func TestBuildImportMapJSONPreservesScopedPackageRuntimePath(t *testing.T) {
	withPackageRuntimeCandidates(t, func(name string) []string {
		if name == "@radix-ui/react-slot" {
			return []string{"1.1.0"}
		}
		return nil
	})
	raw, warnings := buildImportMapJSON(internalpreview.Bundle{
		Dependencies: []internaldeps.Declaration{{DepName: "@radix-ui/react-slot", VersionRange: "^1.1.0"}},
	})
	require.Empty(t, warnings)
	require.Contains(t, raw, `"@radix-ui/react-slot": "/preview/runtime/npm/@radix-ui/react-slot@1.1.0/index.js"`)
}

func TestBuildImportMapJSONWarnsAndFallsBackWhenDeclaredReactRangeIsNotVendored(t *testing.T) {
	raw, warnings := buildImportMapJSON(internalpreview.Bundle{
		Dependencies: []internaldeps.Declaration{
			{DepName: "react", VersionRange: "^17.0.0"},
			{DepName: "react-dom", VersionRange: "^17.0.0"},
		},
	})
	require.NotEmpty(t, warnings)
	require.Contains(t, warnings[0], `runtime react@17.0.2 is not vendored`)
	require.Contains(t, raw, `"react": "/preview/runtime/react@18.3.1/index.js"`)
	require.Contains(t, raw, `"react/jsx-runtime": "/preview/runtime/react@18.3.1/jsx-runtime.js"`)
	require.Contains(t, raw, `"react-dom/client": "/preview/runtime/react-dom@18.3.1/client.js"`)
	require.NotContains(t, raw, `esm.sh/react`)
}

func TestBuildImportMapJSONWarnsOnUnresolvableReactRange(t *testing.T) {
	raw, warnings := buildImportMapJSON(internalpreview.Bundle{
		Dependencies: []internaldeps.Declaration{
			{DepName: "react", VersionRange: "workspace:*"},
		},
	})
	require.NotEmpty(t, warnings)
	require.Contains(t, warnings[0], `cannot pin dependency "react"`)
	require.Contains(t, raw, `"react": "/preview/runtime/react@18.3.1/index.js"`)
}

func TestRenderHarnessHTMLInjectsDesignSystemCSS(t *testing.T) {
	html := renderHarnessHTML("cmp-1", internalpreview.Bundle{
		JS:         "export default function Demo() { return null }",
		SourcePath: "components/Demo.tsx",
		SHA256:     "sha",
	}, harnessExample{})
	require.Contains(t, html, `--color-primary`)
	require.Contains(t, html, `.bg-app-primary`)
	require.Contains(t, html, `.rounded-control`)
}

func TestRenderHarnessHTMLShowsImportMapDiagnostics(t *testing.T) {
	withPackageRuntimeCandidates(t, func(string) []string { return nil })
	html := renderHarnessHTML("cmp-1", internalpreview.Bundle{
		JS:         "export default function Demo() { return null }",
		SourcePath: "components/Demo.tsx",
		SHA256:     "sha",
		Dependencies: []internaldeps.Declaration{
			{DepName: "some-lib", VersionRange: "*"},
		},
	}, harnessExample{})
	require.Contains(t, html, `id="preview-importmap-diagnostics"`)
	require.Contains(t, html, `cannot pin dependency`)
	require.False(t, strings.Contains(html, `some-lib@*`))
}

func TestRenderHarnessHTMLCanShowRuntimeImportFailure(t *testing.T) {
	html := renderHarnessHTML("cmp-1", internalpreview.Bundle{
		JS:         "export default function Demo() { return null }",
		SourcePath: "components/Demo.tsx",
		SHA256:     "sha",
	}, harnessExample{})
	require.NotContains(t, html, `import { createRoot } from "react-dom/client";`)
	require.Contains(t, html, `try {`)
	require.Contains(t, html, `import("react-dom/client")`)
	require.Contains(t, html, `import(componentModuleURL)`)
	require.Contains(t, html, `preview: render failed`)
	require.Contains(t, html, `errEl.hidden = false`)
	require.Contains(t, html, `type: "preview-error"`)
}

func withPackageRuntimeCandidates(t *testing.T, fn func(string) []string) {
	t.Helper()
	prev := packageRuntimeCandidatesFor
	packageRuntimeCandidatesFor = fn
	t.Cleanup(func() {
		packageRuntimeCandidatesFor = prev
	})
}
