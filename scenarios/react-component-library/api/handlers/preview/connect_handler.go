package preview

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"connectrpc.com/connect"

	"react-component-library/internal/components"
	"react-component-library/internal/preview"

	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"
)

// Deps wires the seams the Connect preview handler needs.
type Deps struct {
	RepoRoot string
	Service  preview.Service
	Logger   *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetPreviewBundle(ctx context.Context, req *connect.Request[previewv1.GetPreviewBundleRequest]) (*connect.Response[previewv1.GetPreviewBundleResponse], error) {
	bundle, err := h.deps.Service.GetBundle(ctx, req.Msg.Id)
	if err != nil {
		connectErr := toConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("preview.GetPreviewBundle(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&previewv1.GetPreviewBundleResponse{
		Js:         bundle.JS,
		SourcePath: bundle.SourcePath,
		Sha256:     bundle.SHA256,
		Warnings:   append([]string(nil), bundle.Warnings...),
	}), nil
}

// toConnectError translates preview-domain failures into Connect codes.
// Components-domain errors are forwarded to that package's mapper so
// the wire codes stay consistent across both handlers.
func toConnectError(err error) error {
	if err == nil {
		return nil
	}
	var bundleErr preview.ErrBundle
	if errors.As(err, &bundleErr) {
		return connect.NewError(connect.CodeInvalidArgument, bundleErr)
	}
	// Anything else is a components-domain pass-through (NotFound,
	// PathEscape, …) — delegate to the canonical mapper.
	return components.ToConnectError(err)
}

func (h *connectHandler) GetCompositionBundle(ctx context.Context, req *connect.Request[previewv1.GetCompositionBundleRequest]) (*connect.Response[previewv1.GetCompositionBundleResponse], error) {
	service, ok := h.deps.Service.(preview.CompositionService)
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("composition bundling is unavailable"))
	}
	input := compositionInput(req.Msg)
	bundle, err := service.GetCompositionBundle(ctx, input)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := &previewv1.GetCompositionBundleResponse{Revision: bundle.Revision, Source: bundle.Source, Js: bundle.JS, Sha256: bundle.SHA256, Warnings: bundle.Warnings}
	for _, gap := range bundle.Gaps {
		out.Gaps = append(out.Gaps, &previewv1.CompositionGap{Region: gap.Region, Code: gap.Code, Message: gap.Message, Required: gap.Required})
	}
	return connect.NewResponse(out), nil
}

func compositionInput(message *previewv1.GetCompositionBundleRequest) preview.Composition {
	ref := func(asset *previewv1.CompositionAsset) preview.CompositionAsset {
		return preview.CompositionAsset{CatalogID: asset.GetCatalogId(), Version: asset.GetVersion(), Export: asset.GetExport()}
	}
	input := preview.Composition{Revision: message.GetRevision(), Template: ref(message.GetTemplate())}
	for _, region := range message.GetRegions() {
		item := preview.CompositionRegion{ID: region.GetId(), Parent: region.GetParent(), Slot: region.GetSlot(), Required: region.GetRequired()}
		if region.Asset != nil {
			asset := ref(region.Asset)
			item.Asset = &asset
		}
		input.Regions = append(input.Regions, item)
	}

	return input
}

func (h *connectHandler) RenderComposition(ctx context.Context, req *connect.Request[previewv1.RenderCompositionRequest]) (*connect.Response[previewv1.RenderCompositionResponse], error) {
	input := compositionInput(req.Msg.GetComposition())
	var fixtures []preview.CompositionFixture
	for _, f := range req.Msg.GetFixtures() {
		fixtures = append(fixtures, preview.CompositionFixture{Target: f.GetTarget(), Asset: f.GetAsset(), Version: f.GetVersion(), State: f.GetState(), Field: f.GetField(), Prop: f.GetProp()})
	}
	prepared, err := preview.PrepareComposition(input, req.Msg.GetBindings().AsMap(), fixtures)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return h.RenderPrepared(ctx, prepared, req.Msg)
}

// RenderPrepared lets the authored sketch adapter reuse this exact harness.
func (h *connectHandler) RenderPrepared(ctx context.Context, prepared preview.PreparedComposition, options *previewv1.RenderCompositionRequest) (*connect.Response[previewv1.RenderCompositionResponse], error) {
	service, ok := h.deps.Service.(preview.CompositionService)
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("composition rendering unavailable"))
	}
	input := prepared.Composition
	bundle, err := service.GetCompositionBundle(ctx, prepared.Composition)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	// Missing fixture evidence supersedes the generic unselected-asset gap.
	missing := map[string]bool{}
	for _, gap := range prepared.Gaps {
		missing[gap.Region] = true
	}
	filtered := prepared.Gaps
	for _, gap := range bundle.Gaps {
		if !missing[gap.Region] {
			filtered = append(filtered, gap)
		}
	}
	bundle.Gaps = filtered
	kit := options.GetKit()
	if kit == "" {
		kit = defaultPreviewKit
	}
	css, err := previewDesignSystemCSS(h.deps.RepoRoot, kit)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	theme := options.GetTheme()
	if theme == "" {
		theme = "light"
	}
	if theme != "" && theme != "light" && theme != "dark" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("theme must be light or dark"))
	}
	direction := options.GetDirection()
	if direction == "" {
		direction = "ltr"
	}
	if direction != "" && direction != "ltr" && direction != "rtl" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("direction must be ltr or rtl"))
	}
	args, err := json.Marshal(map[string]any{"bindings": prepared.Bindings})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	importMap, _ := buildImportMapJSON(bundle.Bundle)
	identity, err := json.Marshal(struct {
		Composition                           preview.Composition
		Bindings                              map[string]any
		Fixtures                              []preview.DeterministicFixturePayload
		Bundle                                preview.Bundle
		Kit, Theme, Direction, CSS, ImportMap string
	}{input, prepared.Bindings, prepared.Fixtures, bundle.Bundle, kit, theme, direction, css, importMap})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	inputDigest := sha256.Sum256(identity)
	story := harnessStory{Name: "composition", Version: input.Revision, Kind: components.StoryKindComponent, Mode: components.StoryModePinned, ArgsJSON: string(args), PropsJSON: string(args), Archetype: "page", Slot: "page-template"}
	documentFor := func(id string) string {
		document := renderHarnessHTML(id, bundle.Bundle, story, css, true, direction, "", theme)
		return strings.Replace(document, "<head>", `<head>`+preview.LocalSubmitPolicyMarker+`<meta http-equiv="Content-Security-Policy" content="connect-src 'none'; frame-src 'none'; object-src 'none'; form-action 'none'">`, 1)
	}
	// Hash the actual generated harness with a fixed identity placeholder.
	// This includes embedded runtime code and Go-generated markup without a
	// self-referential hash. A harness change must stale earlier capture targets.
	neutralDocument := documentFor("composition-render-identity")
	// Retain independently comparable contributors for exact-target diagnostics.
	parts := map[string]any{"composition": input, "bindings": prepared.Bindings, "fixtures": prepared.Fixtures, "bundle": bundle.Bundle, "bundle_identity": bundle.SHA256, "javascript": bundle.JS, "dependencies": bundle.Dependencies, "css": css, "import_map": importMap, "appearance": []string{kit, theme, direction}, "harness": neutralDocument}
	inputHashes := make(map[string]string, len(parts))
	for name, value := range parts {
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		sum := sha256.Sum256(raw)
		inputHashes[name] = hex.EncodeToString(sum[:])
	}

	renderIdentity, err := json.Marshal(struct{ InputsSHA256, Document string }{hex.EncodeToString(inputDigest[:]), neutralDocument})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	digest := sha256.Sum256(renderIdentity)
	renderHash := hex.EncodeToString(digest[:])
	bundle.SHA256 = renderHash
	document := documentFor("composition-" + renderHash)
	htmlDigest := sha256.Sum256([]byte(document))
	out := &previewv1.RenderCompositionResponse{Html: document, RenderHash: renderHash,
		Target: &previewv1.CompositionRenderTarget{InputHashes: inputHashes, Revision: input.Revision, RenderHash: renderHash, HtmlSha256: hex.EncodeToString(htmlDigest[:]), InputsSha256: hex.EncodeToString(inputDigest[:]), Kind: "preview", Kit: kit, Theme: theme, Direction: direction},
		Bundle: &previewv1.GetCompositionBundleResponse{Revision: bundle.Revision, Source: bundle.Source, Js: bundle.JS, Sha256: renderHash, Warnings: bundle.Warnings}}
	for _, gap := range bundle.Gaps {
		out.Bundle.Gaps = append(out.Bundle.Gaps, &previewv1.CompositionGap{Region: gap.Region, Code: gap.Code, Message: gap.Message, Required: gap.Required})
	}
	return connect.NewResponse(out), nil
}
