package preview

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/mux"

	"react-component-library/internal/components"
	internaldeps "react-component-library/internal/deps"
	internalpreview "react-component-library/internal/preview"
	"react-component-library/internal/themes"
)

//go:embed assets/harness.js assets/story-evaluator.js assets/story-sheet.css assets/story-sheet.js
var previewAssets embed.FS

var (
	harnessJavaScript        = mustPreviewAsset("assets/harness.js")
	storyEvaluatorJavaScript = mustPreviewAsset("assets/story-evaluator.js")
	storySheetCSS            = mustPreviewAsset("assets/story-sheet.css")
	storySheetJavaScript     = mustPreviewAsset("assets/story-sheet.js")
)

func mustPreviewAsset(name string) string {
	data, err := previewAssets.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func base64Encode(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

// React/ReactDOM pin used in the importmap the harness HTML carries.
// Decision rationale and revisit triggers live in docs/RESEARCH.md.
const (
	defaultReactRuntimeVersion = "18.3.1"
)

var (
	reactRuntimeCandidates       = []string{"16.14.0", "17.0.2", "18.2.0", "18.3.1", "19.0.0", "19.1.0"}
	vendoredReactRuntimeVersions = map[string]bool{defaultReactRuntimeVersion: true}
	packageRuntimeCandidatesFor  = installedPackageVersionCandidates
)

// HarnessHandler serves the per-component HTML shell that the host UI
// loads into the live-preview iframe. Inlines the transpiled module so
// one request gives the browser everything it needs to render.
type HarnessHandler struct {
	service    internalpreview.Service
	components components.Service
	logger     *log.Logger
	repoRoot   string
}

func NewHarnessHandler(svc internalpreview.Service, logger *log.Logger) *HarnessHandler {
	return NewHarnessHandlerWithStories(svc, nil, logger)
}

func NewHarnessHandlerWithStories(svc internalpreview.Service, comp components.Service, logger *log.Logger) *HarnessHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &HarnessHandler{service: svc, components: comp, logger: logger, repoRoot: discoverRepoRoot("")}
}

func NewHarnessHandlerWithStoriesAtRoot(svc internalpreview.Service, comp components.Service, logger *log.Logger, repoRoot string) *HarnessHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &HarnessHandler{service: svc, components: comp, logger: logger, repoRoot: discoverRepoRoot(repoRoot)}
}

// ServeHTTP returns the iframe-friendly HTML shell. Status mapping
// mirrors the Connect handler: NotFound → 404, ErrBundle/PathEscape →
// 400, everything else → 500.
func (h *HarnessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(mux.Vars(r)["id"])
	if id == "" {
		http.Error(w, "missing component id", http.StatusBadRequest)
		return
	}
	componentID, err := h.resolveComponentID(r, id)
	if err != nil {
		writeHarnessError(w, h.logger, id, err)
		return
	}
	if stories := previewStorySheetIDs(r); len(stories) > 0 {
		h.serveStorySheet(w, r, componentID, stories)
		return
	}
	story, err := h.resolveStory(r, componentID)
	if err != nil {
		writeHarnessError(w, h.logger, id, err)
		return
	}
	var frame *components.StoryFrame
	var harness *components.StoryHarnessRef
	if story.Composition != nil {
		frame = story.Composition.Frame
		harness = story.Composition.Harness
	}
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("frame")), "off") {
		frame = nil
	}
	if override := queryFrameOverride(r); override != nil {
		frame = override
	}
	version := strings.TrimSpace(r.URL.Query().Get("version"))
	if version == "" {
		// resolveStory has already selected the catalog version that owns the
		// story contract. Carry that selection into bundling so live specimens
		// are loaded from the same immutable version as their metadata.
		version = story.Version
	}
	var bundle internalpreview.Bundle
	if composer, ok := h.service.(interface {
		GetBundleVersionWithFrameAndHarness(context.Context, string, string, *components.StoryFrame, *components.StoryHarnessRef) (internalpreview.Bundle, error)
	}); ok {
		bundle, err = composer.GetBundleVersionWithFrameAndHarness(r.Context(), componentID, version, frame, harness)
	} else {
		bundle, err = h.service.GetBundleVersionWithFrame(r.Context(), componentID, version, frame)
	}
	if err != nil {
		writeHarnessError(w, h.logger, id, err)
		return
	}
	kit := strings.TrimSpace(r.URL.Query().Get("kit"))
	if kit == "" {
		kit = defaultPreviewKit
	}
	css, err := previewDesignSystemCSS(h.repoRoot, kit)
	if err != nil {
		h.logger.Printf("preview.harness design kit %q: %v", kit, err)
		http.Error(w, "preview: requested design kit is unavailable", http.StatusInternalServerError)
		return
	}
	consumer := strings.TrimSpace(r.URL.Query().Get("consumer"))
	contrastFloor := ""
	if consumer != "" {
		consumerCSS, consumerErr := previewConsumerCSS(h.repoRoot, consumer)
		if consumerErr != nil {
			h.logger.Printf("preview.harness consumer %q: %v", consumer, consumerErr)
			http.Error(w, "preview: requested consumer context is unavailable: "+consumerErr.Error(), http.StatusUnprocessableEntity)
			return
		}
		contrastFloor, consumerErr = previewConsumerContrastFloor(h.repoRoot, consumer)
		if consumerErr != nil {
			h.logger.Printf("preview.harness consumer %q: %v", consumer, consumerErr)
			http.Error(w, "preview: requested consumer context is unavailable: "+consumerErr.Error(), http.StatusUnprocessableEntity)
			return
		}
		// Consumer CSS intentionally follows the neutral preview kit. The
		// composition contract asks the browser to resolve the asset through the
		// consumer's real cascade, including its root token overrides.
		css += "\n/* rcl:consumer-context:" + consumer + " */\n" + consumerCSS
	}
	direction := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("dir")))
	if direction != "rtl" && direction != "ltr" {
		direction = ""
	}
	theme := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("theme")))
	if theme != "light" && theme != "dark" {
		theme = ""
	}
	doc := renderHarnessHTML(id, bundle, story, css, strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("view")), "canvas"), direction, consumer, theme, contrastFloor)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	// Same-origin: the host iframe controls the `src`, and same-origin
	// is required for the future iframe-bridge inspect flow. CSP stays
	// permissive in this dev tool by design.
	if _, err := w.Write([]byte(doc)); err != nil {
		h.logger.Printf("preview.harness write %q: %v", id, err)
	}
}

func previewStorySheetIDs(r *http.Request) []string {
	raw := strings.TrimSpace(r.URL.Query().Get("stories"))
	if raw == "" {
		return nil
	}
	seen := map[string]bool{}
	ids := make([]string, 0, 4)
	for _, value := range strings.Split(raw, ",") {
		id := strings.TrimSpace(value)
		if id == "" || seen[id] || len(ids) >= 4 {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

// serveStorySheet is the generic composite-review surface. It deliberately
// contains only isolated story documents and labels their source contracts;
// BAS captures the outer data-preview-sheet as one review accelerator while
// each iframe remains independently capturable through the normal route.
func (h *HarnessHandler) serveStorySheet(w http.ResponseWriter, r *http.Request, componentID string, storyIDs []string) {
	type tile struct{ ID, Label, Source string }
	tiles := make([]tile, 0, len(storyIDs))
	componentLabel := componentID
	if h.components != nil {
		if component, err := h.components.Get(r.Context(), componentID); err == nil && strings.TrimSpace(component.DisplayName) != "" {
			componentLabel = component.DisplayName
		}
	}
	version := strings.TrimSpace(r.URL.Query().Get("version"))
	kit := strings.TrimSpace(r.URL.Query().Get("kit"))
	if kit == "" {
		kit = defaultPreviewKit
	}
	for _, storyID := range storyIDs {
		story, err := h.resolveStory(r, componentID)
		if err != nil || story.Name != storyID {
			// resolveStory reads the requested story query, so validate each tile
			// with an explicit query clone rather than trusting client labels.
			clone := r.Clone(r.Context())
			query := clone.URL.Query()
			query.Set("story", storyID)
			clone.URL.RawQuery = query.Encode()
			story, err = h.resolveStory(clone, componentID)
			if err != nil {
				writeHarnessError(w, h.logger, componentID, err)
				return
			}
		}
		if version == "" {
			version = story.Version
		}
		query := url.Values{"story": []string{storyID}, "version": []string{version}, "kit": []string{kit}, "view": []string{"canvas"}}
		tiles = append(tiles, tile{ID: storyID, Label: story.DisplayName, Source: "/preview/" + url.PathEscape(componentID) + "/harness.html?" + query.Encode()})
	}
	var body strings.Builder
	body.WriteString(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>`)
	body.WriteString(html.EscapeString(componentLabel + " story review"))
	body.WriteString(`</title><style>`)
	body.WriteString(storySheetCSS)
	body.WriteString(`</style></head><body><main class="story-sheet"`)
	body.WriteString(` data-preview-sheet="story-gallery" data-preview-capture-boundary="component-sheet"`)
	body.WriteString(` data-experience-surface="component-harness" data-experience-state="loading"`)
	body.WriteString(` data-rcl-story-status="pending"><header class="story-sheet__header"><h1>`)
	body.WriteString(html.EscapeString(componentLabel + " stories"))
	body.WriteString(`</h1><p data-story-sheet-summary aria-live="polite">Validating `)
	body.WriteString(fmt.Sprint(len(tiles)))
	body.WriteString(` labeled story specimens · `)
	body.WriteString(html.EscapeString(version))
	body.WriteString(` · `)
	body.WriteString(html.EscapeString(kit))
	body.WriteString(`</p></header><section class="story-sheet__grid" aria-label="Story specimens">`)
	for _, tile := range tiles {
		body.WriteString(`<article class="story-tile" data-story-id="`)
		body.WriteString(html.EscapeString(tile.ID))
		body.WriteString(`"><h2>`)
		body.WriteString(html.EscapeString(tile.Label))
		body.WriteString(`</h2><p>`)
		body.WriteString(html.EscapeString(tile.ID))
		body.WriteString(`</p><iframe title="`)
		body.WriteString(html.EscapeString(tile.Label))
		body.WriteString(` story" src="`)
		body.WriteString(html.EscapeString(tile.Source))
		body.WriteString(`" loading="eager"></iframe></article>`)
	}
	body.WriteString(`</section><p class="story-sheet__footer">Source: version-pinned RCL story contracts.`)
	body.WriteString(` Individual story captures remain authoritative.</p>`)
	body.WriteString(`<pre id="rcl-story-result" hidden></pre></main><script>`)
	body.WriteString(storySheetJavaScript)
	body.WriteString(`</script></body></html>`)
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(body.String()))
}

// queryFrameOverride is intentionally transient. It is only accepted when a
// complete immutable reference is present; persistence belongs to story.json
// authoring and never happens as a side effect of Preview rendering.
func queryFrameOverride(r *http.Request) *components.StoryFrame {
	q := r.URL.Query()
	asset := strings.TrimSpace(q.Get("frameAsset"))
	version := strings.TrimSpace(q.Get("frameVersion"))
	region := strings.TrimSpace(q.Get("frameRegion"))
	if asset == "" || version == "" || region == "" {
		return nil
	}
	return &components.StoryFrame{
		Asset:      asset,
		Version:    version,
		Region:     region,
		Capability: strings.TrimSpace(q.Get("frameCapability")),
		Fixture:    strings.TrimSpace(q.Get("frameFixture")),
	}
}

func (h *HarnessHandler) resolveComponentID(r *http.Request, id string) (string, error) {
	if !strings.Contains(id, ":") || h.components == nil {
		return id, nil
	}
	component, err := h.components.GetByLibraryID(r.Context(), id)
	if err != nil {
		return "", err
	}
	return component.ID, nil
}

func writeHarnessError(w http.ResponseWriter, logger *log.Logger, id string, err error) {
	var (
		notFound         components.ErrComponentNotFound
		pathEscape       components.ErrPathEscape
		bundleErr        internalpreview.ErrBundle
		storyNotFound    components.StoryNotFoundError
		contractNotFound components.StoryContractNotFoundError
		contractParse    components.StoryContractParseError
		storyEncode      components.StoryEncodeError
	)
	switch {
	case errors.As(err, &storyNotFound):
		http.Error(w, storyNotFound.Error(), http.StatusNotFound)
	case errors.As(err, &contractNotFound):
		http.Error(w, contractNotFound.Error(), http.StatusNotFound)
	case errors.As(err, &contractParse):
		http.Error(w, contractParse.Error(), http.StatusUnprocessableEntity)
	case errors.As(err, &storyEncode):
		http.Error(w, storyEncode.Error(), http.StatusUnprocessableEntity)
	case errors.As(err, &notFound):
		http.Error(w, fmt.Sprintf("preview: component %q not found", id), http.StatusNotFound)
	case errors.As(err, &pathEscape):
		http.Error(w, fmt.Sprintf("preview: source_path escapes root for %q", id), http.StatusBadRequest)
	case errors.As(err, &bundleErr):
		// Render the diagnostic inside the iframe so the user sees the
		// error in-place rather than a blank pane. Keep the HTTP status
		// truthful so automation cannot mistake this document for a ready
		// preview.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(renderBundleErrorHTML(bundleErr)))
	default:
		logger.Printf("preview.harness internal %q: %v", id, err)
		http.Error(w, "preview: internal error", http.StatusInternalServerError)
	}
}

type harnessStory struct {
	Name                  string
	Version               string
	DisplayName           string
	Kind                  components.StoryKind
	PropsJSON             string
	ArgsJSON              string
	EnvironmentJSON       string
	EnvironmentSchemaJSON string
	InteractionsJSON      string
	ExpectJSON            string
	Composition           *components.StoryComposition
	Fixture               *components.StoryFixtureRef
	Frame                 *components.StoryFrame
	Geometry              *components.StoryGeometry
	Mode                  components.StoryMode
	Slot                  string
	AssetKind             components.AssetKind
	Archetype             string
}

// resolveStory uses the indexed story contract as the only harness baseline.
func (h *HarnessHandler) resolveStory(r *http.Request, id string) (harnessStory, error) {
	storyID := strings.TrimSpace(r.URL.Query().Get("story"))
	if storyID == "" || h.components == nil {
		return harnessStory{}, nil
	}
	version := strings.TrimSpace(r.URL.Query().Get("version"))
	component, err := h.components.Get(r.Context(), id)
	if err != nil {
		return harnessStory{}, err
	}
	stories, err := h.components.ListStories(r.Context(), components.StoryQuery{ComponentID: id, Version: version, Limit: 20})
	if err != nil {
		return harnessStory{}, err
	}
	if len(stories) == 0 {
		return harnessStory{}, components.StoryContractNotFoundError{ComponentID: id, Version: version}
	}
	declaredIDs := []string{}
	for _, projected := range stories {
		contractJSON := []byte(projected.ContractJSON)
		if reader, ok := h.components.(interface {
			GetVersionContentAt(context.Context, string, string, string) (components.Content, error)
		}); ok {
			content, contentErr := reader.GetVersionContentAt(r.Context(), id, projected.Version, "story.json")
			if contentErr != nil {
				if errors.Is(contentErr, os.ErrNotExist) {
					return harnessStory{}, components.StoryContractNotFoundError{ComponentID: id, Version: projected.Version}
				}
				return harnessStory{}, components.StoryContractParseError{ComponentID: id, Version: projected.Version, Detail: contentErr.Error()}
			}
			contractJSON = []byte(content.Body)
		}
		contract, diagnostics := components.ParseStoryContract(contractJSON)
		if contract == nil || len(components.StoryContractErrors(diagnostics)) > 0 {
			messages := make([]string, 0, len(diagnostics))
			for _, diagnostic := range diagnostics {
				messages = append(messages, diagnostic.Error())
			}
			return harnessStory{}, components.StoryContractParseError{ComponentID: id, Version: projected.Version, Detail: strings.Join(messages, "; ")}
		}
		declaredIDs = make([]string, 0, len(contract.Stories))
		for _, definition := range contract.Stories {
			declaredIDs = append(declaredIDs, definition.ID)
			if definition.ID != storyID {
				continue
			}
			args, err := json.Marshal(definition.Args)
			if err != nil {
				return harnessStory{}, components.StoryEncodeError{ComponentID: id, StoryID: storyID, Field: "args", Cause: err}
			}
			expect, err := json.Marshal(definition.Expect)
			if err != nil {
				return harnessStory{}, components.StoryEncodeError{ComponentID: id, StoryID: storyID, Field: "expectations", Cause: err}
			}
			interactions, err := json.Marshal(definition.Interactions)
			if err != nil {
				return harnessStory{}, components.StoryEncodeError{ComponentID: id, StoryID: storyID, Field: "interactions", Cause: err}
			}
			environment, err := json.Marshal(definition.Environment)
			if err != nil {
				return harnessStory{}, components.StoryEncodeError{ComponentID: id, StoryID: storyID, Field: "environment", Cause: err}
			}
			composition := definition.Composition
			var fixture *components.StoryFixtureRef
			if composition != nil {
				fixture = composition.Fixture
			}
			var frame *components.StoryFrame
			if composition != nil {
				frame = composition.Frame
			}
			return harnessStory{
				Name: definition.ID, Version: projected.Version, DisplayName: definition.Name,
				Kind: contract.Kind, PropsJSON: string(args), ArgsJSON: projected.ArgsJSON,
				EnvironmentJSON: string(environment), EnvironmentSchemaJSON: projected.EnvironmentJSON,
				InteractionsJSON: string(interactions), ExpectJSON: string(expect),
				Composition: composition, Fixture: fixture, Geometry: definition.Geometry,
				Mode: definition.Mode, Slot: component.Slot, AssetKind: component.AssetKind,
				Archetype: resolvePreviewArchetype(component.AssetKind, component.Slot, frame),
			}, nil
		}
	}
	return harnessStory{}, components.StoryNotFoundError{ComponentID: id, StoryID: storyID, DeclaredIDs: declaredIDs}
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
<script type="module">`)
	sb.WriteString(renderHarnessJavaScript(id, b, ex))
	sb.WriteString(`</script>
</body>
</html>
`)
	return sb.String()
}

func renderHarnessJavaScript(id string, b internalpreview.Bundle, ex harnessStory) string {
	replacements := map[string]string{
		"__COMPONENT_MODULE_URL__":     jsString("data:text/javascript;base64," + base64Encode(b.JS)),
		"__STORY_HARNESS_MODULE_URL__": jsString("data:text/javascript;base64," + base64Encode(b.HarnessJS+"\n"+b.CompositionHarnessJS)),
		"__FRAME_MODULE_URL__":         jsString("data:text/javascript;base64," + base64Encode(b.FrameJS)),
		"__STORY_NAME__":               jsString(ex.Name),
		"__STORY_VERSION__":            jsString(ex.Version),
		"__STORY_DISPLAY_NAME__":       jsString(ex.DisplayName),
		"__STORY_KIND__":               jsString(string(ex.Kind)),
		"__STORY_PROPS__":              jsonObjectLiteral(ex.PropsJSON),
		"__STORY_ARGS__":               jsonObjectLiteral(ex.ArgsJSON),
		"__STORY_ENVIRONMENT__":        jsonObjectLiteral(ex.EnvironmentJSON),
		"__STORY_ENVIRONMENT_SCHEMA__": jsonObjectLiteral(ex.EnvironmentSchemaJSON),
		"__STORY_INTERACTIONS__":       jsonArrayLiteral(ex.InteractionsJSON),
		"__STORY_EXPECT__":             jsonArrayLiteral(ex.ExpectJSON),
		"__STORY_COMPOSITION__":        compositionJSON(ex.Composition),
		"__STORY_GEOMETRY__":           geometryJSON(ex.Geometry),
		"__STORY_MODE__":               jsString(string(ex.Mode)),
		"__STORY_SLOT__":               jsString(ex.Slot),
		"__STORY_ASSET_KIND__":         jsString(string(ex.AssetKind)),
		"__STORY_ARCHETYPE__":          jsString(ex.Archetype),
		"__STORY_FIXTURE__":            fixtureJSON(ex.Fixture, b.FixtureJSON),
		"__PREVIEW_ID__":               jsString(id),
		"__BUNDLE_SHA256__":            jsString(b.SHA256),
	}
	// The evaluator is embedded into the same module as the harness so the
	// browser and jsdom paths execute one implementation.
	script := storyEvaluatorJavaScript + "\n" + harnessJavaScript
	for placeholder, replacement := range replacements {
		script = strings.ReplaceAll(script, placeholder, replacement)
	}
	return script
}

func buildImportMapJSON(b internalpreview.Bundle) (string, []string) {
	reactVersion, reactDOMVersion, warnings := resolveReactRuntimeVersions(b.Dependencies)
	imports := map[string]string{
		"react":                 runtimeURL("react", reactVersion, "", &warnings),
		"react/jsx-runtime":     runtimeURL("react", reactVersion, "jsx-runtime", &warnings),
		"react/jsx-dev-runtime": runtimeURL("react", reactVersion, "jsx-dev-runtime", &warnings),
		"react-dom":             runtimeURL("react-dom", reactDOMVersion, "", &warnings),
		"react-dom/client":      runtimeURL("react-dom", reactDOMVersion, "client", &warnings),
	}
	if version, ok := internaldeps.ResolveRangeToLatest("10.4.1", packageRuntimeCandidatesFor("@testing-library/dom")); ok {
		imports["@testing-library/dom"] = packageRuntimeURL("@testing-library/dom", version, "", &warnings)
	} else {
		warnings = append(warnings, "preview: dependency \"@testing-library/dom\" cannot be resolved from the governed preview runtime store")
	}
	for _, d := range b.Dependencies {
		name := strings.TrimSpace(d.DepName)
		if name == "" || isReactRuntimeDep(name) {
			continue
		}
		version, ok := internaldeps.ResolveRangeToLatest(d.VersionRange, packageRuntimeCandidatesFor(name))
		if !ok {
			warnings = append(warnings, fmt.Sprintf(
				"preview: dependency %q cannot be resolved from declared range %q; "+
					"package is absent from the governed preview runtime store. Populate with: %s",
				name, d.VersionRange, previewDependencyPopulateCommand(name, d.VersionRange),
			))
			continue
		}
		imports[name] = packageRuntimeURL(name, version, "", &warnings)
	}
	if _, ok := imports["lucide-react"]; !ok {
		if version, resolved := internaldeps.ResolveRangeToLatest("^0.424.0", packageRuntimeCandidatesFor("lucide-react")); resolved {
			imports["lucide-react"] = packageRuntimeURL("lucide-react", version, "", &warnings)
		}
	}
	raw, err := json.MarshalIndent(map[string]map[string]string{"imports": imports}, "", "  ")
	if err != nil {
		return `{"imports":{}}`, append(warnings, "preview: failed to encode importmap: "+err.Error())
	}
	return string(raw), warnings
}

func resolveReactRuntimeVersions(declarations []internaldeps.Declaration) (string, string, []string) {
	reactVersion := defaultReactRuntimeVersion
	reactDOMVersion := ""
	var warnings []string
	for _, d := range declarations {
		name := strings.TrimSpace(d.DepName)
		switch name {
		case "react":
			version, ok := internaldeps.ResolveRangeToLatest(d.VersionRange, reactRuntimeCandidates)
			if !ok {
				warnings = append(warnings, fmt.Sprintf("preview: cannot pin dependency %q from range %q", name, d.VersionRange))
				continue
			}
			reactVersion = version
		case "react-dom":
			version, ok := internaldeps.ResolveRangeToLatest(d.VersionRange, reactRuntimeCandidates)
			if !ok {
				warnings = append(warnings, fmt.Sprintf("preview: cannot pin dependency %q from range %q", name, d.VersionRange))
				continue
			}
			reactDOMVersion = version
		}
	}
	if reactDOMVersion == "" {
		reactDOMVersion = reactVersion
	}
	return reactVersion, reactDOMVersion, warnings
}

func isReactRuntimeDep(name string) bool {
	return name == "react" || name == "react-dom" || strings.HasPrefix(name, "react/")
}

func runtimeURL(name, version, subpath string, warnings *[]string) string {
	resolvedVersion := strings.TrimSpace(version)
	if !vendoredReactRuntimeVersions[resolvedVersion] {
		*warnings = append(*warnings, fmt.Sprintf("preview: runtime %s@%s is not vendored; using %s", name, resolvedVersion, defaultReactRuntimeVersion))
		resolvedVersion = defaultReactRuntimeVersion
	}
	if subpath != "" {
		return "/preview/runtime/" + name + "@" + resolvedVersion + "/" + subpath + ".js"
	}
	return "/preview/runtime/" + name + "@" + resolvedVersion + "/index.js"
}

func packageRuntimeURL(name, version, subpath string, warnings *[]string) string {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" || version == "" {
		*warnings = append(*warnings, "preview: dependency runtime URL missing package name or version")
		return ""
	}
	if subpath != "" {
		return "/preview/runtime/npm/" + name + "@" + version + "/" + strings.TrimPrefix(subpath, "/")
	}
	return "/preview/runtime/npm/" + name + "@" + version + "/index.js"
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
func jsString(s string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			sb.WriteString(`\\`)
		case '"':
			sb.WriteString(`\"`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		default:
			if r < 0x20 || r == '<' || r == '>' || r == '&' {
				fmt.Fprintf(&sb, `\u%04x`, r)
			} else {
				sb.WriteRune(r)
			}
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

func jsonObjectLiteral(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "{}"
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return "{}"
	}
	normalized, err := json.Marshal(obj)
	if err != nil {
		return "{}"
	}
	return string(normalized)
}

func compositionJSON(composition *components.StoryComposition) string {
	if composition == nil {
		return "null"
	}
	raw, err := json.Marshal(composition)
	if err != nil {
		return "null"
	}
	return string(raw)
}

func fixtureJSON(fixture *components.StoryFixtureRef, base string) string {
	baseLiteral := jsonObjectLiteral(base)
	if fixture == nil {
		return baseLiteral
	}
	raw, err := json.Marshal(fixture)
	if err != nil {
		return baseLiteral
	}
	payload, payloadErr := internalpreview.ResolveDeterministicFixture(fixture.Asset, fixture.Version, fixture.State)
	if payloadErr != nil {
		failure, _ := json.Marshal(map[string]string{"fixtureError": payloadErr.Error()})
		return "Object.assign(" + baseLiteral + "," + string(raw) + "," + string(failure) + ")"
	}
	payloadJSON, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return "Object.assign(" + baseLiteral + "," + string(raw) + ")"
	}
	return "Object.assign(" + baseLiteral + "," + string(raw) + "," + string(payloadJSON) + ")"
}

func geometryJSON(geometry *components.StoryGeometry) string {
	if geometry == nil {
		return "null"
	}
	raw, err := json.Marshal(geometry)
	if err != nil {
		return "null"
	}
	return string(raw)
}

func jsonArrayLiteral(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "[]"
	}
	var arr []any
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return "[]"
	}
	normalized, err := json.Marshal(arr)
	if err != nil {
		return "[]"
	}
	return string(normalized)
}

const defaultPreviewKit = "vrooli-default"

var previewCSSCache struct {
	sync.Mutex
	key      string
	modified string
	css      string
}

func discoverRepoRoot(explicit string) string {
	candidates := []string{explicit, ".", "../..", "../../..", "../../../..", "../../../../.."}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		path := filepath.Join(candidate, "templates", "design", defaultPreviewKit, "adapters", "react-vite-tailwind", "tokens.css")
		if _, err := os.Stat(path); err == nil {
			return candidate
		}
	}
	return explicit
}

func previewDesignSystemCSS(repoRoot, kit string) (string, error) {
	if kit == "none" {
		return "", nil
	}
	if kit == "" || strings.Contains(kit, ".") || strings.ContainsAny(kit, `/\\`) {
		return "", fmt.Errorf("invalid design kit %q", kit)
	}
	adapter := filepath.Join(repoRoot, "templates", "design", kit, "adapters", "react-vite-tailwind")
	tokensPath := filepath.Join(adapter, "tokens.css")
	baseTokensPath := filepath.Join(repoRoot, "templates", "design", "_base", "tokens.css")
	utilitiesPath := filepath.Join(adapter, "preview-utilities.css")
	baseTokensInfo, err := os.Stat(baseTokensPath)
	if err != nil {
		return "", fmt.Errorf("shared token vocabulary missing: %w", err)
	}
	tokensInfo, err := os.Stat(tokensPath)
	if err != nil {
		return "", fmt.Errorf("tokens stylesheet missing: %w", err)
	}
	utilitiesInfo, err := os.Stat(utilitiesPath)
	if err != nil {
		return "", fmt.Errorf("compiled utility stylesheet missing for %q: %w", kit, err)
	}
	key := kit
	modified := fmt.Sprintf("%d:%d:%d", baseTokensInfo.ModTime().UnixNano(), tokensInfo.ModTime().UnixNano(), utilitiesInfo.ModTime().UnixNano())
	previewCSSCache.Lock()
	defer previewCSSCache.Unlock()
	if previewCSSCache.key == key && previewCSSCache.modified == modified && previewCSSCache.css != "" {
		return previewCSSCache.css, nil
	}
	tokens, err := themes.ComposeKitCSS(repoRoot, kit)
	if err != nil {
		return "", fmt.Errorf("compose token stylesheets: %w", err)
	}
	utilities, err := os.ReadFile(utilitiesPath)
	if err != nil {
		return "", fmt.Errorf("read compiled utility stylesheet: %w", err)
	}
	if len(utilities) == 0 {
		return "", fmt.Errorf("compiled utility stylesheet for %q is empty", kit)
	}
	previewCSSCache.key = key
	previewCSSCache.modified = modified
	previewCSSCache.css = tokens + "\n" + string(utilities)
	return previewCSSCache.css, nil
}

// ServeBaseStylesFixture renders the exact published foundation CSS without
// loading any design kit. It is a browser-measurable proof of both the
// canonical floor and unlayered scenario precedence.
func (h *HarnessHandler) ServeBaseStylesFixture(w http.ResponseWriter, r *http.Request) {
	version := strings.TrimSpace(r.URL.Query().Get("version"))
	if version == "" || filepath.Base(version) != version || strings.HasPrefix(version, ".") || strings.ContainsAny(version, `/\\`) {
		http.Error(w, "preview: a safe BaseStyles version is required", http.StatusBadRequest)
		return
	}
	path := filepath.Join(h.repoRoot, "scenarios", "react-component-library", "library", "foundations", "BaseStyles", "versions", version, "BaseStyles.ts")
	raw, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "preview: BaseStyles version is unavailable", http.StatusNotFound)
		return
	}
	css, err := extractBaseStylesCSS(string(raw))
	if err != nil {
		http.Error(w, "preview: BaseStyles source is malformed", http.StatusUnprocessableEntity)
		return
	}
	css = strings.ReplaceAll(css, "</style", "<\\/style")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><style>%s</style></head><body><main><div id="canonical" style="padding:var(--space-md);border-radius:var(--radius-panel)">Canonical layered default</div><div id="override" style="--space-md:7px;padding:var(--space-md);border-radius:var(--radius-panel)">Unlayered scenario override</div><output id="measurement" data-testid="base-styles-measurement"></output></main><script>const canonical=getComputedStyle(document.querySelector('#canonical'));const override=getComputedStyle(document.querySelector('#override'));const value='canonical padding='+canonical.paddingTop+', radius='+canonical.borderTopLeftRadius+'; override padding='+override.paddingTop;const output=document.querySelector('#measurement');output.textContent=value;output.dataset.canonicalPadding=canonical.paddingTop;output.dataset.canonicalRadius=canonical.borderTopLeftRadius;output.dataset.overridePadding=override.paddingTop;</script></body></html>`, css)
}

func extractBaseStylesCSS(source string) (string, error) {
	const marker = "export const baseStyles = `"
	start := strings.Index(source, marker)
	if start < 0 {
		return "", fmt.Errorf("baseStyles template literal not found")
	}
	rest := source[start+len(marker):]
	end := strings.Index(rest, "`;")
	if end < 0 {
		return "", fmt.Errorf("baseStyles template literal is unterminated")
	}
	return rest[:end], nil
}

// previewConsumerCSS reads a scenario's current build output and source-owned
// token bridge directly. It never copies either artifact into RCL: the named
// scenario remains the sole source of truth for the cascade under test.
func previewConsumerCSS(repoRoot, consumer string) (string, error) {
	if consumer == "" || strings.Contains(consumer, ".") || strings.ContainsAny(consumer, `/\\`) {
		return "", fmt.Errorf("invalid consumer %q", consumer)
	}
	uiRoot := filepath.Join(repoRoot, "scenarios", consumer, "ui")
	tokensPath := filepath.Join(uiRoot, "src", "design-tokens.css")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		return "", fmt.Errorf("read source token bridge: %w", err)
	}
	bundlePaths, err := filepath.Glob(filepath.Join(uiRoot, "dist", "assets", "*.css"))
	if err != nil || len(bundlePaths) == 0 {
		return "", fmt.Errorf("compiled CSS bundle is missing; build scenarios/%s first", consumer)
	}
	var newestBundle int64
	var css strings.Builder
	for _, path := range bundlePaths {
		info, statErr := os.Stat(path)
		if statErr != nil {
			return "", fmt.Errorf("stat compiled CSS %s: %w", filepath.Base(path), statErr)
		}
		if info.ModTime().UnixNano() > newestBundle {
			newestBundle = info.ModTime().UnixNano()
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return "", fmt.Errorf("read compiled CSS %s: %w", filepath.Base(path), readErr)
		}
		css.Write(data)
		css.WriteByte('\n')
	}
	var newestSource int64
	err = filepath.Walk(filepath.Join(uiRoot, "src"), func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !info.IsDir() && info.ModTime().UnixNano() > newestSource {
			newestSource = info.ModTime().UnixNano()
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("inspect consumer sources: %w", err)
	}
	if newestSource > newestBundle {
		return "", fmt.Errorf("compiled CSS is older than consumer source; rebuild scenarios/%s", consumer)
	}
	return string(tokens) + "\n" + css.String(), nil
}

// previewConsumerContrastFloor keeps contrast acceptance coupled to the
// consumer-owned token contract. Browser workflows read the emitted meta value
// instead of duplicating a threshold that could drift from token-map.json.
func previewConsumerContrastFloor(repoRoot, consumer string) (string, error) {
	if consumer == "" || strings.Contains(consumer, ".") || strings.ContainsAny(consumer, `/\\`) {
		return "", fmt.Errorf("invalid consumer %q", consumer)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, "scenarios", consumer, "ui", "token-map.json"))
	if err != nil {
		return "", fmt.Errorf("read consumer token map: %w", err)
	}
	var tokenMap struct {
		ContrastFloor float64 `json:"contrast_floor"`
	}
	if err := json.Unmarshal(data, &tokenMap); err != nil {
		return "", fmt.Errorf("parse consumer token map: %w", err)
	}
	if tokenMap.ContrastFloor <= 0 {
		return "", fmt.Errorf("consumer token map has no positive contrast_floor")
	}
	return strconv.FormatFloat(tokenMap.ContrastFloor, 'f', -1, 64), nil
}
