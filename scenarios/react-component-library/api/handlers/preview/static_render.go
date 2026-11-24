package preview

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"react-component-library/internal/components"
	internalpreview "react-component-library/internal/preview"
)

func materializedVersionFile(repoRoot string, asset components.Component, version, name string) ([]byte, error) {
	if strings.TrimSpace(repoRoot) == "" || strings.TrimSpace(asset.SourcePath) == "" || strings.TrimSpace(version) == "" {
		return nil, os.ErrNotExist
	}
	source := filepath.Join(repoRoot, "scenarios", "react-component-library", "library", filepath.FromSlash(filepath.Clean(asset.SourcePath)))
	assetDir := filepath.Dir(filepath.Dir(filepath.Dir(source)))
	return os.ReadFile(filepath.Join(assetDir, "versions", strings.TrimSpace(version), name))
}

type previewArchetype string

const (
	previewArchetypePrimitive previewArchetype = "primitive"
	previewArchetypePattern   previewArchetype = "pattern"
	previewArchetypePage      previewArchetype = "page"
	previewArchetypeOverlay   previewArchetype = "overlay"
)

func resolvePreviewArchetype(kind components.AssetKind, slot string, frame *components.StoryFrame) string {
	if frame != nil {
		asset := strings.ToLower(frame.Asset)
		if strings.Contains(asset, "dialog") || strings.Contains(asset, "modal") || strings.Contains(asset, "popover") || strings.Contains(asset, "overlay") {
			return string(previewArchetypeOverlay)
		}
		return string(previewArchetypePage)
	}
	if kind == components.AssetKindHook || kind == components.AssetKindFoundation {
		return string(previewArchetypePrimitive)
	}
	switch strings.ToLower(strings.TrimSpace(slot)) {
	case "ui-primitive":
		return string(previewArchetypePrimitive)
	case "ui-pattern":
		return string(previewArchetypePattern)
	case "layout-nav":
		return string(previewArchetypePage)
	default:
		return string(previewArchetypePattern)
	}
}

func renderHarnessHTML(id string, b internalpreview.Bundle, ex harnessStory, designSystemCSS string, galleryMode ...any) string {
	var sb strings.Builder
	bodyClass := ""
	gallery := false
	direction := ""
	if len(galleryMode) > 0 {
		gallery, _ = galleryMode[0].(bool)
	}
	if len(galleryMode) > 1 {
		direction, _ = galleryMode[1].(string)
	}
	consumer := ""
	if len(galleryMode) > 2 {
		consumer, _ = galleryMode[2].(string)
	}
	theme := ""
	if len(galleryMode) > 3 {
		theme, _ = galleryMode[3].(string)
	}
	contrastFloor := ""
	if len(galleryMode) > 4 {
		contrastFloor, _ = galleryMode[4].(string)
	}
	axeCoreScript := ""
	if len(galleryMode) > 5 {
		axeCoreScript, _ = galleryMode[5].(string)
	}
	if direction != "rtl" && direction != "ltr" {
		direction = ""
	}
	if gallery {
		bodyClass = ` class="rcl-preview-gallery"`
	}
	sb.WriteString(`<!doctype html>
<html lang="en"`)
	if direction != "" {
		sb.WriteString(` dir="`)
		sb.WriteString(direction)
		sb.WriteString(`"`)
	}
	if theme == "light" || theme == "dark" {
		sb.WriteString(` data-resolved-theme="`)
		sb.WriteString(theme)
		sb.WriteString(`"`)
	}
	if theme == "dark" {
		sb.WriteString(` class="dark"`)
	}
	sb.WriteString(`>
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>preview: `)
	sb.WriteString(html.EscapeString(b.SourcePath))
	sb.WriteString(`</title>
<link rel="icon" href="data:," />
<meta name="component-id" content="`)
	sb.WriteString(html.EscapeString(id))
	sb.WriteString(`" />
<meta name="bundle-sha256" content="`)
	sb.WriteString(html.EscapeString(b.SHA256))
	sb.WriteString(`" />
<meta name="story-id" content="`)
	sb.WriteString(html.EscapeString(ex.Name))
	sb.WriteString(`" />
<meta name="consumer-context" content="`)
	sb.WriteString(html.EscapeString(consumer))
	sb.WriteString(`" />
<meta name="consumer-contrast-floor" content="`)
	sb.WriteString(html.EscapeString(contrastFloor))
	sb.WriteString(`" />
<style>
`)
	sb.WriteString(designSystemCSS)
	sb.WriteString(`
  html, body { margin: 0; padding: 0; min-height: 100vh; background: var(--color-background); color: var(--color-foreground); font-family: var(--font-sans, ui-sans-serif), system-ui, sans-serif; }
  #root { min-height: 100vh; box-sizing: border-box; background: var(--color-background); }
  html[data-rcl-capture="deterministic"] *,
  html[data-rcl-capture="deterministic"] *::before,
  html[data-rcl-capture="deterministic"] *::after {
    animation-duration: 0s !important;
    animation-delay: 0s !important;
    transition-duration: 0s !important;
    transition-delay: 0s !important;
    caret-color: transparent !important;
  }
  .rcl-preview-gallery,
  .rcl-preview-gallery #root { min-height: 100%; height: 100%; overflow: auto; }
  .rcl-preview-gallery .rcl-preview-specimen { min-height: 100%; height: 100%; padding: 24px; }
  .rcl-preview-specimen {
    min-height: 100vh;
    box-sizing: border-box;
    display: grid;
    place-items: center;
    padding: clamp(24px, 6vw, 80px);
    background:
      radial-gradient(circle at 50% 0%, color-mix(in srgb, var(--color-primary) 10%, transparent), transparent 42%),
      linear-gradient(135deg, color-mix(in srgb, var(--color-surface) 54%, transparent), transparent 58%),
      var(--color-background);
  }
  .rcl-preview-stage {
    min-height: 100vh;
    box-sizing: border-box;
    width: 100%;
    display: grid;
    align-content: start;
    background: var(--color-background);
  }
  .rcl-preview-stage--pattern { padding: clamp(24px, 4vw, 64px); }
  .rcl-preview-stage--page,
  .rcl-preview-stage--overlay { min-height: 100vh; padding: 0; }
  /* Stable boundary for component-sheet screenshot evidence. */
  [data-preview-sheet] { min-height: 100%; box-sizing: border-box; }
  /* Isolated BAS/component-test evidence should describe the rendered
     specimen, not a full browser viewport with a small control stranded in
     the middle. Overlay and frame stories opt back into viewport geometry
     below because their backdrop/focus context is part of the subject. */
  html[data-rcl-capture-mode="isolated"] #root,
  html[data-rcl-capture-mode="isolated"] [data-preview-sheet]:not(:has([role="dialog"])):not(:has([data-rcl-dialog])) {
    min-height: 0;
  }
  html[data-rcl-capture-mode="isolated"] .rcl-preview-specimen,
  html[data-rcl-capture-mode="isolated"] .rcl-preview-stage {
    min-height: 0;
  }
  /* A fixed/absolute specimen has no in-flow height. Preserve the viewport
     capture context only after the mounted boundary confirms that geometry. */
  html[data-rcl-capture-mode="isolated"] [data-preview-sheet][data-preview-positioned="true"] {
    min-height: 100vh;
  }
  [data-preview-sheet]:has([data-preview-harness-density="compact"]) {
    width: min(100%, 640px);
    margin-inline: auto;
  }
  /* Fixed overlays paint outside a tight content box. Keep the host-owned
     capture boundary viewport-sized whenever a story contains an overlay so
     the authoritative screenshot includes the surface, backdrop, and focus
     context instead of clipping to the trigger's bounds. */
  [data-preview-sheet]:has([data-rcl-dialog]),
  [data-preview-sheet]:has([role="dialog"]) {
    min-height: 100vh;
    width: 100%;
  }
  .rcl-preview-well {
    width: min(100%, 760px);
    box-sizing: border-box;
    padding: clamp(24px, 5vw, 56px);
    border: var(--border-hairline) solid color-mix(in srgb, var(--color-border) 72%, transparent);
    border-radius: var(--radius-sheet);
    background: color-mix(in srgb, var(--color-surface) 88%, transparent);
    box-shadow: var(--elev-overlay);
    /*
      Keep the specimen shell from becoming a containing block. CSS
      backdrop-filter changes the containing block for fixed descendants,
      which would trap modal/command-palette previews inside this card rather
      than showing their real viewport-sized behavior.
    */
  }
  .rcl-preview-well__meta {
    margin: 0 0 var(--space-md);
    color: var(--color-muted-foreground);
    font-size: var(--text-label-size);
    font-weight: 700;
    letter-spacing: .08em;
    line-height: var(--text-label-line);
    text-transform: uppercase;
  }
  .rcl-preview-well__content { display: grid; place-items: center; min-height: 96px; }
  .rcl-preview-well__content > * { max-width: 100%; }
  .rcl-preview-fixture {
    display: grid;
    gap: var(--space-sm);
    min-height: 260px;
    box-sizing: border-box;
    padding: var(--space-lg);
    border: var(--border-hairline) solid var(--color-border);
    border-radius: var(--radius-panel);
    background: var(--color-surface-raised);
    color: var(--color-foreground);
  }
  .rcl-preview-fixture__heading { color: var(--color-muted-foreground); font-size: var(--text-label-size); font-weight: 700; letter-spacing: .08em; text-transform: uppercase; }
  .rcl-preview-fixture__rows { display: grid; align-content: start; gap: var(--space-2xs); }
  .rcl-preview-fixture__row { block-size: var(--space-sm); border-radius: var(--radius-control); background: var(--color-surface-muted); }
  .rcl-preview-fixture__row:nth-child(1) { inline-size: 88%; }
  .rcl-preview-fixture__row:nth-child(2) { inline-size: 66%; }
  .rcl-preview-fixture__row:nth-child(3) { inline-size: 76%; }
  [data-frame-asset="navigation.page"] { min-height: 100vh; }
  @media (max-width: 520px) {
    .rcl-preview-specimen { padding: 24px 16px; }
  }
  #preview-importmap-diagnostics,
  #preview-error { padding: 16px; font-family: ui-monospace, SFMono-Regular, monospace; color: #ff8c8c; white-space: pre-wrap; }
</style>
<script type="importmap">
`)
	importMap, importWarnings := buildImportMapJSON(b)
	sb.WriteString(strings.ReplaceAll(importMap, "</script", "<\\/script"))
	sb.WriteString(`
</script>
<script>
// The browser profile owns locale, while the harness owns the semantic
// writing direction. Locale-driven fallback keeps the RTL matrix observable
// even when a capture producer cannot add a dir query parameter.
(() => {
  const requested = new URLSearchParams(window.location.search).get("dir");
  const language = String(navigator.language || "").toLowerCase();
  const direction = requested === "rtl" || requested === "ltr"
    ? requested
    : /^(ar|fa|he|ur)(-|$)/.test(language) ? "rtl" : "ltr";
  document.documentElement.dir = direction;
  document.documentElement.dataset.rclDirection = direction;
})();
// Signal the standard iframe bridge before the preview's module graph loads.
// The component bundle can be cold and take longer than the host's readiness
// budget, but the harness itself is ready to receive bridge/inspect messages.
(() => {
  if (window.parent === window) return;
  window.__vrooliBridgeChildInstalled = true;
  const post = (payload) => { try { window.parent.postMessage(payload, "*"); } catch (e) {} };
  post({ v: 1, t: "HELLO", appId: "react-component-library", title: document.title, caps: ["history", "hash", "title", "deeplink", "screenshot", "shortcuts", "inspect"] });
  // Re-announce READY while the host completes iframe attachment. This keeps
  // cold component imports from turning a valid harness into a timing race.
  const ready = () => post({ v: 1, t: "READY" });
  queueMicrotask(ready);
  [100, 500, 1500].forEach((delay) => setTimeout(ready, delay));
})();
</script>
</head>
<body`)
	sb.WriteString(bodyClass)
	sb.WriteString(`>
<div id="root" data-testid="component-harness-root" data-experience-surface="component-harness" data-experience-state="loading"></div>
`)
	if len(importWarnings) > 0 {
		sb.WriteString(`<div id="preview-importmap-diagnostics">`)
		for _, warning := range importWarnings {
			sb.WriteString(html.EscapeString(warning))
			sb.WriteString("\n")
		}
		sb.WriteString(`</div>
`)
	}
	sb.WriteString(`
<div id="preview-error" hidden></div>
<pre id="rcl-story-result" hidden></pre>
`)
	if axeCoreScript != "" {
		sb.WriteString(`<script>`)
		sb.WriteString(strings.ReplaceAll(axeCoreScript, `</script`, `<\/script`))
		sb.WriteString(`</script>`)
	}
	sb.WriteString(`<script>`)
	sb.WriteString(axeEvidenceJavaScript)
	sb.WriteString(`</script>
<script type="module">`)
	sb.WriteString(renderHarnessJavaScript(id, b, ex))
	sb.WriteString(`</script>
</body>
</html>
`)
	return sb.String()
}

func loadAxeCoreJavaScript(repoRoot string) string {
	path := filepath.Join(repoRoot, "scenarios", "react-component-library", "ui", "node_modules", "axe-core", "axe.min.js")
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 || len(data) > 1024*1024 {
		return ""
	}
	return string(data)
}

func renderHarnessJavaScript(id string, b internalpreview.Bundle, ex harnessStory) string {
	replacements := map[string]string{
		"__COMPONENT_MODULE_URL__":         jsString("data:text/javascript;base64," + base64Encode(b.JS)),
		"__STORY_HARNESS_MODULE_URL__":     jsString(storyHarnessModuleURL(b.HarnessJS, b.CompositionHarnessJS)),
		"__FRAME_MODULE_URL__":             jsString("data:text/javascript;base64," + base64Encode(b.FrameJS)),
		"__STORY_NAME__":                   jsString(ex.Name),
		"__STORY_VERSION__":                jsString(ex.Version),
		"__STORY_DISPLAY_NAME__":           jsString(ex.DisplayName),
		"__STORY_KIND__":                   jsString(string(ex.Kind)),
		"__STORY_ROLE__":                   jsString(ex.Role),
		"__STORY_RENDERS_NOTHING__":        strconv.FormatBool(ex.RendersNothing),
		"__STORY_RENDERS_NOTHING_REASON__": jsString(ex.RendersNothingReason),
		"__STORY_PROPS__":                  jsonObjectLiteral(ex.PropsJSON),
		"__STORY_ARGS__":                   jsonObjectLiteral(ex.ArgsJSON),
		"__STORY_ENVIRONMENT__":            jsonObjectLiteral(ex.EnvironmentJSON),
		"__STORY_ENVIRONMENT_SCHEMA__":     jsonObjectLiteral(ex.EnvironmentSchemaJSON),
		"__STORY_INTERACTIONS__":           jsonArrayLiteral(ex.InteractionsJSON),
		"__STORY_EXPECT__":                 jsonArrayLiteral(ex.ExpectJSON),
		"__STORY_COMPOSITION__":            compositionJSON(ex.Composition),
		"__STORY_GEOMETRY__":               geometryJSON(ex.Geometry),
		"__STORY_MODE__":                   jsString(string(ex.Mode)),
		"__STORY_SLOT__":                   jsString(ex.Slot),
		"__STORY_ASSET_KIND__":             jsString(string(ex.AssetKind)),
		"__STORY_ARCHETYPE__":              jsString(ex.Archetype),
		"__STORY_FIXTURE__":                fixtureJSON(ex.Fixture, b.FixtureJSON),
		"__PREVIEW_ID__":                   jsString(id),
		"__BUNDLE_SHA256__":                jsString(b.SHA256),
	}
	// The evaluator is embedded into the same module as the harness so the
	// browser and jsdom paths execute one implementation.
	script := storyEvaluatorJavaScript + "\n" + harnessJavaScript
	for placeholder, replacement := range replacements {
		script = strings.ReplaceAll(script, placeholder, replacement)
	}
	return script
}

// storyHarnessModuleURL exposes the independently bundled story and composition
// harnesses through one module without merging their top-level declarations.
// Separately bundled esbuild outputs commonly declare helpers such as jsx; raw
// concatenation would redeclare those helpers in the same module scope.
func storyHarnessModuleURL(storyJS, compositionJS string) string {
	modules := make([]string, 0, 2)
	for _, source := range []string{storyJS, compositionJS} {
		if strings.TrimSpace(source) == "" {
			continue
		}
		modules = append(modules, jsString("data:text/javascript;base64,"+base64Encode(source)))
	}
	if len(modules) == 0 {
		return "data:text/javascript;base64," + base64Encode("")
	}
	var source strings.Builder
	for _, moduleURL := range modules {
		source.WriteString("export * from ")
		source.WriteString(moduleURL)
		source.WriteString(";\n")
	}
	return "data:text/javascript;base64," + base64Encode(source.String())
}

func renderBundleErrorHTML(err internalpreview.ErrBundle) string {
	var sb strings.Builder
	sb.WriteString(`<!doctype html>
<html lang="en"><head><meta charset="utf-8" /><title>preview: bundle error</title>
<style>body{margin:0;padding:16px;background:#0b0d12;color:#ff8c8c;font:13px/1.5 ui-monospace,SFMono-Regular,monospace;white-space:pre-wrap;}</style>
</head><body>`)
	sb.WriteString(html.EscapeString(fmt.Sprintf("bundle failed: %s\n\n", err.SourcePath)))
	for _, m := range err.Messages {
		sb.WriteString(html.EscapeString(m))
		sb.WriteString("\n")
	}
	sb.WriteString(`</body></html>`)
	return sb.String()
}

// jsString emits a JS-safe double-quoted string literal. Keeps the
// inline `<script>` literal-free of template-injection risk.
