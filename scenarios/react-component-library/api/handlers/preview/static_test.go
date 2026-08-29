package preview

import (
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/components"
	internaldeps "react-component-library/internal/deps"
	internalpreview "react-component-library/internal/preview"
)

func TestPreviewDesignSystemCSSComposesBaseBeforeKitOverrides(t *testing.T) {
	repo := t.TempDir()
	base := filepath.Join(repo, "templates", "design", "_base")
	adapter := filepath.Join(repo, "templates", "design", "demo", "adapters", "react-vite-tailwind")
	require.NoError(t, os.MkdirAll(base, 0o755))
	require.NoError(t, os.MkdirAll(adapter, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(base, "tokens.css"), []byte(":root { --shared: base; --tone: base; }\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(adapter, "tokens.css"), []byte(":root { --tone: kit; }\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(adapter, "preview-utilities.css"), []byte(".utility { display: block; }\n"), 0o644))
	previewCSSCache = struct {
		sync.Mutex
		key      string
		modified string
		css      string
	}{}

	css, err := previewDesignSystemCSS(repo, "demo")
	require.NoError(t, err)
	require.Contains(t, css, "--shared: base")
	require.Less(t, strings.Index(css, "--tone: base"), strings.Index(css, "--tone: kit"))
	require.Contains(t, css, ".utility { display: block; }")
}

func TestPreviewDesignSystemCSSAllowsFoundationOnlyFixture(t *testing.T) {
	css, err := previewDesignSystemCSS(t.TempDir(), "none")
	require.NoError(t, err)
	require.Empty(t, css)
}

func TestExtractBaseStylesCSSReadsPublishedTemplateLiteral(t *testing.T) {
	css, err := extractBaseStylesCSS("export const baseStyles = `@layer rcl.tokens { :root { --space-md: 24px; } }`;\n")
	require.NoError(t, err)
	require.Contains(t, css, "--space-md: 24px")
}

func TestPreviewConsumerCSSReadsLiveTokenBridgeAndCompiledBundle(t *testing.T) {
	repo := t.TempDir()
	ui := filepath.Join(repo, "scenarios", "plain-consumer", "ui")
	require.NoError(t, os.MkdirAll(filepath.Join(ui, "src"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(ui, "dist", "assets"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ui, "src", "design-tokens.css"), []byte(":root{--color-surface:#fff}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(ui, "dist", "assets", "index.css"), []byte(".consumer-shell{display:grid}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(ui, "token-map.json"), []byte(`{"contrast_floor":4.5}`), 0o644))

	css, err := previewConsumerCSS(repo, "plain-consumer")
	require.NoError(t, err)
	require.Contains(t, css, "--color-surface:#fff")
	require.Contains(t, css, ".consumer-shell{display:grid}")
	floor, err := previewConsumerContrastFloor(repo, "plain-consumer")
	require.NoError(t, err)
	require.Equal(t, "4.5", floor)
}

func TestPreviewConsumerCSSRejectsPathTraversalAndMissingBuild(t *testing.T) {
	_, err := previewConsumerCSS(t.TempDir(), "../outside")
	require.ErrorContains(t, err, "invalid consumer")

	repo := t.TempDir()
	ui := filepath.Join(repo, "scenarios", "not-built", "ui", "src")
	require.NoError(t, os.MkdirAll(ui, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ui, "design-tokens.css"), []byte(":root{}"), 0o644))
	_, err = previewConsumerCSS(repo, "not-built")
	require.ErrorContains(t, err, "compiled CSS bundle is missing")
	_, err = previewConsumerContrastFloor(repo, "../outside")
	require.ErrorContains(t, err, "invalid consumer")
}

func TestPreviewStorySheetIDsIsBoundedAndDeduplicated(t *testing.T) {
	req := httptest.NewRequest("GET", "/preview/x/harness.html?stories=default,loading,default,error,extra", nil)
	got := previewStorySheetIDs(req)
	require.Equal(t, []string{"default", "loading", "error", "extra"}, got)
	query := url.Values{}
	query.Set("stories", "")
	req.URL.RawQuery = query.Encode()
	require.Empty(t, previewStorySheetIDs(req))
}

func TestStorySheetPublishesTheSharedReadinessMarker(t *testing.T) {
	require.Contains(t, storySheetJavaScript, `readiness.dataset.previewReadinessMarker`)
	require.Contains(t, storySheetJavaScript, `readiness.dataset.previewReady`)
	require.Contains(t, storySheetJavaScript, `setState("ready", "passed")`)
}

func TestHarnessPublishesConsumerContrastFloor(t *testing.T) {
	html := renderHarnessHTML("component", internalpreview.Bundle{}, harnessStory{}, "", false, "", "plain-consumer", "light", "4.5")
	require.Contains(t, html, `<meta name="consumer-contrast-floor" content="4.5" />`)
}

const testPreviewCSS = `:root { --color-primary: #2563eb; } .bg-app-primary { background: var(--color-primary); } .rounded-control { border-radius: var(--radius-control); }`

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
	require.NotContains(t, raw, `http://`)
	require.NotContains(t, raw, `https://`)
}

func TestResolvePreviewArchetypeKeepsPrimitiveWellButLetsCompositionsFillStage(t *testing.T) {
	require.Equal(t, "primitive", resolvePreviewArchetype(components.AssetKindComponent, "ui-primitive", nil))
	require.Equal(t, "pattern", resolvePreviewArchetype(components.AssetKindComponent, "ui-pattern", nil))
	require.Equal(t, "page", resolvePreviewArchetype(components.AssetKindComponent, "layout-nav", nil))
	require.Equal(t, "overlay", resolvePreviewArchetype(components.AssetKindComponent, "ui-pattern", &components.StoryFrame{Asset: "overlays.dialog"}))
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
	}, harnessStory{}, testPreviewCSS)
	require.Contains(t, html, `--color-primary`)
	require.Contains(t, html, `.bg-app-primary`)
	require.Contains(t, html, `.rounded-control`)
	require.NotContains(t, html, `background: #0b0d12`)
	require.Contains(t, html, `rcl-resolved-theme`)
	require.Contains(t, html, `document.documentElement.dataset.resolvedTheme`)
	require.Contains(t, html, `data-preview-sheet`)
	require.Contains(t, html, `[data-preview-sheet]:has([data-rcl-dialog])`)
	require.Contains(t, html, `Fixed overlays paint outside a tight content box`)
	require.Contains(t, html, `t: "HELLO", appId: "react-component-library"`)
	require.Contains(t, html, `"inspect"`)
	require.Contains(t, html, `queueMicrotask(ready)`)
	require.Contains(t, html, `[100, 500, 1500].forEach((delay) => setTimeout(ready, delay))`)
	require.Contains(t, html, `window.__vrooliBridgeChildInstalled = true`)
	require.Contains(t, html, `Route isolation: component navigation is part of the specimen`)
	require.Contains(t, html, `window.history.pushState =`)
	require.Contains(t, html, `document.addEventListener("click", (event) =>`)
	require.Less(t, strings.Index(html, `window.__vrooliBridgeChildInstalled = true`), strings.Index(html, `<script type="module">`), "the bridge handshake must not wait for preview module imports")
}

func TestRenderHarnessHTMLGalleryModeBoundsTheEmbeddedSurface(t *testing.T) {
	html := renderHarnessHTML("cmp-1", internalpreview.Bundle{
		JS:         "export default function Demo() { return null }",
		SourcePath: "components/Demo.tsx",
		SHA256:     "sha",
	}, harnessStory{}, testPreviewCSS, true)
	require.Contains(t, html, `<body class="rcl-preview-gallery">`)
	require.Contains(t, html, `.rcl-preview-gallery #root`)
	require.Contains(t, html, `.rcl-preview-gallery .rcl-preview-specimen`)
	require.Contains(t, html, `height: 100%; overflow: auto;`)
}

func TestRenderHarnessHTMLAppliesExplicitThemeToDocument(t *testing.T) {
	darkHTML := renderHarnessHTML("cmp-1", internalpreview.Bundle{
		JS:         "export default function Demo() { return null }",
		SourcePath: "components/Demo.tsx",
		SHA256:     "sha",
	}, harnessStory{}, testPreviewCSS, false, "", "web-console", "dark")
	require.Contains(t, darkHTML, `<html lang="en" data-resolved-theme="dark" class="dark">`)

	lightHTML := renderHarnessHTML("cmp-1", internalpreview.Bundle{
		JS:         "export default function Demo() { return null }",
		SourcePath: "components/Demo.tsx",
		SHA256:     "sha",
	}, harnessStory{}, testPreviewCSS, false, "", "web-console", "light")
	require.Contains(t, lightHTML, `<html lang="en" data-resolved-theme="light">`)
	require.NotContains(t, lightHTML, `data-resolved-theme="light" class="dark"`)
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
	}, harnessStory{}, testPreviewCSS)
	require.Contains(t, html, `id="preview-importmap-diagnostics"`)
	require.Contains(t, html, `cannot be resolved from declared range`)
	require.Contains(t, html, `scenario-dependency-analyzer deps install npm/some-lib@*`)
	require.False(t, strings.Contains(html, `/preview/runtime/npm/some-lib@*`))
}

func TestRenderHarnessHTMLCanShowRuntimeImportFailure(t *testing.T) {
	html := renderHarnessHTML("cmp-1", internalpreview.Bundle{
		JS:         "export default function Demo() { return null }",
		SourcePath: "components/Demo.tsx",
		SHA256:     "sha",
	}, harnessStory{}, testPreviewCSS)
	require.NotContains(t, html, `import { createRoot } from "react-dom/client";`)
	require.Contains(t, html, `try {`)
	require.Contains(t, html, `import("react-dom/client")`)
	require.Contains(t, html, `import(componentModuleURL)`)
	require.Contains(t, html, `preview: render failed`)
	require.Contains(t, html, `errEl.hidden = false`)
	require.Contains(t, html, `type: "preview-error"`)
}

func TestRenderHarnessHTMLSupportsScopedTemporaryPropsOverrides(t *testing.T) {
	html := renderHarnessHTML("cmp-1", internalpreview.Bundle{
		JS:         "export default function Demo() { return null }",
		SourcePath: "components/Demo.tsx",
		SHA256:     "sha",
	}, harnessStory{Name: "default", Version: "1.0.0", PropsJSON: `{"title":"Indexed"}`}, testPreviewCSS)
	require.Contains(t, html, `const root = createRoot(document.getElementById("root"))`)
	require.Contains(t, html, `const renderPreview = (override, environment = previewStory.environment)`)
	require.Contains(t, html, `const validateEnvironment = (environment)`)
	require.Contains(t, html, `fixture option is not declared.`)
	require.Contains(t, html, `data.environment || previewStory.environment`)
	require.Contains(t, html, `const mergeStoryProps = (base, override)`)
	require.Contains(t, html, `valueAtPath(merged, field.path)`)
	require.NotContains(t, html, `field.path.includes(".")`)
	require.Contains(t, html, `rcl-preview-props-override`)
	require.Contains(t, html, `rcl-preview-props-reset`)
	require.Contains(t, html, `rcl-preview-props-applied`)
	require.Contains(t, html, `rcl-preview-props-error`)
	require.Contains(t, html, `rcl-story-result`)
	require.Contains(t, html, `id="rcl-story-result"`)
	require.Contains(t, html, `const reportStoryResult = (passed, failures, skipped = []) =>`)
	require.Contains(t, html, `export async function runStory(previewStory, modules, env = browserEnv)`)
	require.Contains(t, html, `void runStory(previewStory, { document, window }`)
	require.Contains(t, html, `const initialTarget = previewStory.interactions?.[0]?.target || previewStory.expect?.[0]`)
	require.Contains(t, html, `different realm implementations of accessible-name lookup`)
	require.NotContains(t, html, `setTimeout(() => {`)
	require.NotContains(t, html, `const expectationFailure = (expectation)`)
	require.Contains(t, html, `@testing-library/dom`)
	require.Contains(t, html, `data.componentId !== "cmp-1"`)
	require.NotContains(t, html, `eval(`)
}

func TestRenderHarnessHTMLWaitForInteractionWaitsForVisibleText(t *testing.T) {
	html := renderHarnessHTML("cmp-1", internalpreview.Bundle{
		JS:         "export default function Demo() { return null }",
		SourcePath: "components/Demo.tsx",
		SHA256:     "sha",
	}, harnessStory{
		Name:             "interactive",
		Version:          "1.0.0",
		InteractionsJSON: `[{"kind":"waitFor","text":"Copied"}]`,
	}, testPreviewCSS)
	require.Contains(t, html, `const waitText = String(interaction.text || "").trim()`)
	require.Contains(t, html, `const deadline = Date.now() + 2000`)
	require.Contains(t, html, `const visibleText = Array.from(document.querySelectorAll("body *"))`)
}

func TestRenderHarnessHTMLIncludesFrameAndFixtureComposition(t *testing.T) {
	html := renderHarnessHTML("cmp-sidebar", internalpreview.Bundle{
		JS:          "export default function Sidebar() { return null }",
		FrameJS:     "export function PageFrame() { return null }",
		FrameAsset:  "navigation.page",
		FrameRegion: "navigation",
		FixtureJSON: `{"asset":"fixtures.resource-collection"}`,
		SourcePath:  "components/SidebarShell.tsx",
		SHA256:      "sha-frame",
	}, harnessStory{Name: "persistent", Composition: &components.StoryComposition{Frame: &components.StoryFrame{Asset: "navigation.page", Region: "navigation", Fixture: "fixtures.resource-collection"}}}, testPreviewCSS)
	require.Contains(t, html, `const frameModuleURL`)
	require.Contains(t, html, `previewStory.composition`)
	require.Contains(t, html, `data-frame-region`)
	require.Contains(t, html, `const regions = { [previewStory.composition.frame.region]: subject, content: fixtureRegion }`)
	require.Contains(t, html, `data-fixture-asset`)
}

func TestRenderHarnessHTMLRecordsDeclarativeHandlersAndCustomHarnessEvents(t *testing.T) {
	html := renderHarnessHTML("cmp-1", internalpreview.Bundle{
		JS:         "export default function Demo() { return null }",
		HarnessJS:  "export function StatefulHarness() { return null }",
		SourcePath: "components/Demo.tsx",
		SHA256:     "sha",
	}, harnessStory{Name: "interactive", Version: "1.0.0", Composition: &components.StoryComposition{Specimen: &components.StorySpecimenRef{Module: "./story.tsx", Export: "StatefulHarness"}}}, testPreviewCSS)
	require.Contains(t, html, `const createNodeFactory = (React, Icons, log) =>`)
	require.Contains(t, html, `return (...args) => log(name, ...args);`)
	require.Contains(t, html, `type: "rcl-preview-event"`)
	require.Contains(t, html, `React.createElement(Specimen, { args: props, environment, fixtures: resolveFixtureContext(environment), log: postPreviewEvent })`)
	require.Less(t, strings.Index(html, `const postPreviewEvent =`), strings.Index(html, `const resolveProps = createNodeFactory`))
}

func withPackageRuntimeCandidates(t *testing.T, fn func(string) []string) {
	t.Helper()
	prev := packageRuntimeCandidatesFor
	packageRuntimeCandidatesFor = func(name string) []string {
		if name == "@testing-library/dom" {
			return []string{"10.4.1"}
		}
		return fn(name)
	}
	t.Cleanup(func() {
		packageRuntimeCandidatesFor = prev
	})
}
