package preview

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"

	"react-component-library/internal/components"
	internalpreview "react-component-library/internal/preview"
)

//go:embed assets/harness.js assets/story-evaluator.js assets/story-sheet.css assets/story-sheet.js assets/axe-evidence.js
var previewAssets embed.FS

var (
	harnessJavaScript        = mustPreviewAsset("assets/harness.js")
	storyEvaluatorJavaScript = mustPreviewAsset("assets/story-evaluator.js")
	storySheetCSS            = mustPreviewAsset("assets/story-sheet.css")
	storySheetJavaScript     = mustPreviewAsset("assets/story-sheet.js")
	axeEvidenceJavaScript    = mustPreviewAsset("assets/axe-evidence.js")
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
	service           internalpreview.Service
	components        components.Service
	logger            *log.Logger
	repoRoot          string
	axeCoreJavaScript string
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
	resolvedRoot := discoverRepoRoot(repoRoot)
	return &HarnessHandler{service: svc, components: comp, logger: logger, repoRoot: resolvedRoot, axeCoreJavaScript: loadAxeCoreJavaScript(resolvedRoot)}
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
	if sheetRequested(r) {
		stories, err := h.resolveStorySheetIDs(r, componentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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
	// Older preview sessions persisted the appearance mode "system" in the
	// kit slot. It is not a concrete server-backed design kit; preserve those
	// sessions by resolving the stale value to the canonical default kit.
	if kit == "system" {
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
	axeScript := ""
	if r.URL.Query().Get("runner") == "1" {
		axeScript = h.axeCoreJavaScript
	}
	doc := renderHarnessHTML(id, bundle, story, css, strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("view")), "canvas"), direction, consumer, theme, contrastFloor, axeScript)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	// Same-origin: the host iframe controls the `src`, and same-origin
	// is required for the future iframe-bridge inspect flow. CSP stays
	// permissive in this dev tool by design.
	if _, err := w.Write([]byte(doc)); err != nil {
		h.logger.Printf("preview.harness write %q: %v", id, err)
	}
}

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
	// Library-qualified preview URLs carry the manifest's public identifier.
	// Use the explicit library-id seam first so the browser route follows the
	// same lookup as the component transport, even when UUID/slug resolution
	// has different repository behavior.
	component, err := h.components.GetByLibraryID(r.Context(), id)
	if err != nil {
		var notFound components.ErrComponentNotFound
		if !errors.As(err, &notFound) {
			return "", err
		}
		// Retry the canonical service lookup for stable catalog IDs and UUIDs.
		// The suffix fallback also keeps unqualified authored references useful
		// when a caller accidentally carries the namespace in front of a slug.
		component, err = h.components.Get(r.Context(), id)
		if err != nil {
			if !errors.As(err, &notFound) {
				return "", err
			}
			if _, suffix, ok := strings.Cut(id, ":"); ok && strings.TrimSpace(suffix) != "" {
				component, err = h.components.Get(r.Context(), strings.TrimSpace(suffix))
			}
		}
	}
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
	Role                  string
	RendersNothing        bool
	RendersNothingReason  string
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
	if version == "" {
		// Story projections can outlive removed drafts. Default navigation must
		// follow the published manifest pointer, not version-row query ordering.
		version = strings.TrimSpace(component.LatestVersion)
		if version == "" {
			return harnessStory{}, components.StoryContractNotFoundError{ComponentID: id}
		}
	}
	stories, err := h.components.ListStories(r.Context(), components.StoryQuery{ComponentID: id, Version: version, Limit: 20})
	if err != nil {
		return harnessStory{}, err
	}
	if len(stories) == 0 {
		if contractJSON, materializedErr := materializedVersionFile(h.repoRoot, component, version, "story.json"); materializedErr == nil {
			stories = []components.ComponentStory{{ComponentID: id, Version: version, ContractJSON: string(contractJSON)}}
		} else {
			return harnessStory{}, components.StoryContractNotFoundError{ComponentID: id, Version: version}
		}
	}
	declaredIDs := []string{}
	for _, projected := range stories {
		contractJSON := []byte(projected.ContractJSON)
		materialized, materializedErr := materializedVersionFile(h.repoRoot, component, projected.Version, "story.json")
		if materializedErr == nil {
			contractJSON = materialized
		} else if !errors.Is(materializedErr, os.ErrNotExist) {
			return harnessStory{}, components.StoryContractParseError{ComponentID: id, Version: projected.Version, Detail: materializedErr.Error()}
		} else if reader, ok := h.components.(interface {
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
			resolved, err := components.ResolveStoryArgs(contract, storyID)
			if err != nil {
				return harnessStory{}, components.StoryContractParseError{ComponentID: id, Version: projected.Version, Detail: err.Error()}
			}
			args, err := json.Marshal(resolved)
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
				Kind: contract.Kind, Role: definition.Role, RendersNothing: definition.RendersNothing, RendersNothingReason: definition.RendersNothingReason,
				PropsJSON: string(args), ArgsJSON: projected.ArgsJSON,
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
