package preview

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"

	"react-component-library/internal/components"
)

func previewStorySheetIDs(r *http.Request) []string {
	raw := strings.TrimSpace(r.URL.Query().Get("stories"))
	if raw == "" {
		return nil
	}
	seen := map[string]bool{}
	ids := make([]string, 0)
	for _, value := range strings.Split(raw, ",") {
		id := strings.TrimSpace(value)
		if id == "" || id == "all" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

func sheetRequested(r *http.Request) bool {
	q := r.URL.Query()
	return strings.TrimSpace(q.Get("stories")) != "" || strings.TrimSpace(q.Get("role")) != "" || strings.TrimSpace(q.Get("axis")) != "" || strings.TrimSpace(q.Get("viewport")) != ""
}

func (h *HarnessHandler) resolveStorySheetIDs(r *http.Request, componentID string) ([]string, error) {
	if h.components == nil {
		return nil, fmt.Errorf("story sheet for %s is unavailable without the component service", componentID)
	}
	component, err := h.components.Get(r.Context(), componentID)
	if err != nil {
		return nil, err
	}
	version := strings.TrimSpace(r.URL.Query().Get("version"))
	if version == "" {
		version = strings.TrimSpace(component.LatestVersion)
	}
	if version == "" {
		return nil, fmt.Errorf("asset %s has no released story version", componentID)
	}
	projections, err := h.components.ListStories(r.Context(), components.StoryQuery{ComponentID: componentID, Version: version, Limit: 1000})
	if err != nil {
		return nil, err
	}
	type storyInfo struct{ ID, Role, Axis string }
	known := make([]storyInfo, 0)
	for _, projection := range projections {
		contract, diagnostics := components.ParseStoryContract([]byte(projection.ContractJSON))
		if contract == nil || len(components.StoryContractErrors(diagnostics)) > 0 {
			continue
		}
		for _, story := range contract.Stories {
			known = append(known, storyInfo{ID: story.ID, Role: story.Role, Axis: story.Axis})
		}
		if version == "" {
			version = projection.Version
		}
	}
	if len(known) == 0 {
		return nil, fmt.Errorf("asset %s has no story contract", componentID)
	}
	requested := previewStorySheetIDs(r)
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("stories")), "all") {
		requested = make([]string, 0, len(known))
		for _, story := range known {
			requested = append(requested, story.ID)
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("role")); raw != "" {
		roles := map[string]bool{}
		for _, value := range strings.Split(raw, ",") {
			value = strings.TrimSpace(value)
			if value != "anatomy" && value != "axis" && value != "boundary" {
				return nil, fmt.Errorf("unknown story role %q for asset %s", value, componentID)
			}
			roles[value] = true
		}
		requested = requested[:0]
		for _, story := range known {
			if roles[story.Role] {
				requested = append(requested, story.ID)
			}
		}
	}
	if axis := strings.TrimSpace(r.URL.Query().Get("axis")); axis != "" {
		requested = requested[:0]
		anatomy := ""
		matchedAxis := false
		for _, story := range known {
			if story.Role == "anatomy" {
				anatomy = story.ID
			}
			if story.Role == "axis" && story.Axis == axis {
				requested = append(requested, story.ID)
				matchedAxis = true
			}
		}
		if !matchedAxis {
			return nil, fmt.Errorf("unknown axis %q for asset %s", axis, componentID)
		}
		if anatomy != "" {
			requested = append([]string{anatomy}, requested...)
		}
	}
	if len(requested) == 0 {
		if story := strings.TrimSpace(r.URL.Query().Get("story")); story != "" {
			requested = []string{story}
		} else {
			return nil, fmt.Errorf("story selection is empty for asset %s", componentID)
		}
	}
	knownIDs := map[string]bool{}
	for _, story := range known {
		knownIDs[story.ID] = true
	}
	for _, storyID := range requested {
		if !knownIDs[storyID] {
			return nil, fmt.Errorf("unknown story id %q for asset %s", storyID, componentID)
		}
	}
	viewports := strings.Split(strings.TrimSpace(r.URL.Query().Get("viewport")), ",")
	if strings.TrimSpace(r.URL.Query().Get("viewport")) != "" && len(viewports) > 1 && len(requested) > 1 {
		return nil, fmt.Errorf("viewport axis conflicts with multiple stories for asset %s", componentID)
	}
	return requested, nil
}

// serveStorySheet is the generic composite-review surface. It deliberately
// contains only isolated story documents and labels their source contracts;
// BAS captures the outer data-preview-sheet as one review accelerator while
// each iframe remains independently capturable through the normal route.
func (h *HarnessHandler) serveStorySheet(w http.ResponseWriter, r *http.Request, componentID string, storyIDs []string) {
	type tile struct{ ID, Label, Role, Viewport, Source string }
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
	viewports := []string{""}
	if raw := strings.TrimSpace(r.URL.Query().Get("viewport")); raw != "" {
		viewports = nil
		for _, value := range strings.Split(raw, ",") {
			value = strings.TrimSpace(value)
			if value != "" {
				viewports = append(viewports, value)
			}
		}
	}
	theme := strings.TrimSpace(r.URL.Query().Get("theme"))
	if theme != "light" && theme != "dark" {
		theme = ""
	}
	for _, storyID := range storyIDs {
		for _, viewport := range viewports {
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
			query := url.Values{"story": []string{storyID}, "version": []string{story.Version}, "kit": []string{kit}, "sheet": []string{"1"}}
			if theme != "" {
				query.Set("theme", theme)
			}
			if viewport != "" {
				query.Set("viewport", viewport)
			}
			tileID := storyID
			if viewport != "" {
				tileID += ":" + viewport
			}
			tiles = append(tiles, tile{ID: tileID, Label: story.DisplayName, Role: story.Role, Viewport: viewport, Source: "/preview/" + url.PathEscape(componentID) + "/harness.html?" + query.Encode()})
		}
	}
	var body strings.Builder
	body.WriteString(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>`)
	body.WriteString(html.EscapeString(componentLabel + " story review"))
	body.WriteString(`</title><style>`)
	body.WriteString(storySheetCSS)
	baseCSS, cssErr := previewDesignSystemCSS(h.repoRoot, kit)
	if cssErr != nil {
		http.Error(w, "preview: base styles unavailable: "+cssErr.Error(), http.StatusInternalServerError)
		return
	}
	body.WriteString(baseCSS)
	body.WriteString(storySheetCSS)
	body.WriteString(`</style></head><body><main class="story-sheet"`)
	if theme != "" {
		body.WriteString(` data-rcl-theme="`)
		body.WriteString(html.EscapeString(theme))
		body.WriteString(`"`)
	}
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
		body.WriteString(`</h2><p class="story-tile__meta" data-story-meta data-story-role="`)
		body.WriteString(html.EscapeString(tile.Role))
		body.WriteString(`"><span class="story-role story-role--`)
		body.WriteString(html.EscapeString(tile.Role))
		body.WriteString(`">`)
		body.WriteString(html.EscapeString(tile.Role))
		body.WriteString(`</span> · <span data-story-size>measuring</span>`)
		body.WriteString(`</p><div class="story-tile__viewport" data-story-viewport><iframe width="1280" height="320" title="`)
		body.WriteString(html.EscapeString(tile.Label))
		body.WriteString(` story" src="`)
		body.WriteString(html.EscapeString(tile.Source))
		body.WriteString(`" loading="eager"></iframe></div></article>`)
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
