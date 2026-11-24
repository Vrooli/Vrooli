package sketch

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"

	"connectrpc.com/connect"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"

	"react-component-library/internal/availability"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
	"react-component-library/internal/designcapture"
	"react-component-library/internal/designcritique"
	"react-component-library/internal/designinference"
	"react-component-library/internal/preview"
	"react-component-library/internal/reconcile"
	internalsearch "react-component-library/internal/search"
	internal "react-component-library/internal/sketch"

	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
)

type CompositionRenderer interface {
	RenderPrepared(context.Context, preview.PreparedComposition, *previewv1.RenderCompositionRequest) (*connect.Response[sharedv1.RenderCompositionResponse], error)
}
type Deps struct {
	InferenceFor      func(context.Context) (designinference.Repository, error)
	InferenceClient   designinference.Gateway
	CritiquesFor      func(context.Context) (designcritique.Repository, error)
	CapturesFor       func(context.Context) (designcapture.Repository, error)
	CaptureDispatcher designcapture.Dispatcher
	Assets            components.DependencyReader
	Renderer          CompositionRenderer
	Catalog           func(context.Context) ([]catalogcoverage.Asset, error)
	ImportSearch      internalsearch.CatalogSearch
	Availability      func(context.Context) (*availability.Snapshot, error)
	Store             internal.Repository
	StoreFor          func(context.Context) (internal.Repository, error)
	RecordWrite       func(context.Context)
	AdoptExperience   func(context.Context, string, string, []byte) (*workspacev1.Workspace, string, string, string, string, bool, error)
	ReconcilerFor     func(context.Context) (*reconcile.Resolver, error)
	Reconciler        *reconcile.Resolver
	Logger            *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(deps Deps) *connectHandler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &connectHandler{deps: deps}
}

func (h *connectHandler) GetSketch(ctx context.Context, req *connect.Request[sketchv1.GetSketchRequest]) (*connect.Response[sketchv1.GetSketchResponse], error) {
	h, routeErr := h.forRequest(ctx)
	if routeErr != nil {
		return nil, routeErr
	}
	target, err := requestTarget(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	snapshot, err := h.deps.Store.Read(target.scenario, target.page)
	if err != nil {
		return nil, h.connectError("GetSketch", err)
	}
	path, _ := h.deps.Store.Path(target.scenario, target.page)
	return connect.NewResponse(&sketchv1.GetSketchResponse{Sketch: toProto(snapshot.Document), DocumentPath: path, ContentHash: snapshot.ContentHash, DeclaredRegions: toProto(internal.Document{Regions: snapshot.DeclaredRegions}).Regions}), nil
}

func (h *connectHandler) PutSketch(ctx context.Context, req *connect.Request[sketchv1.PutSketchRequest]) (*connect.Response[sketchv1.PutSketchResponse], error) {
	h, routeErr := h.forRequest(ctx)
	if routeErr != nil {
		return nil, routeErr
	}
	target, err := requestTarget(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	if req.Msg.GetSketch() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("sketch is required"))
	}
	return h.save(ctx, target, req.Msg.GetExpectedContentHash(), fromProto(req.Msg.GetSketch()))
}

func (h *connectHandler) VerifySketch(ctx context.Context, req *connect.Request[sketchv1.VerifySketchRequest]) (*connect.Response[sketchv1.VerifySketchResponse], error) {
	h, routeErr := h.forRequest(ctx)
	if routeErr != nil {
		return nil, routeErr
	}
	target, err := requestTarget(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	if h.deps.Reconciler == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("sketch reconciler is not configured"))
	}
	joins, resolveErr := h.deps.Reconciler.Resolve(ctx, target.scenario, target.page)
	if resolveErr != nil {
		return nil, h.connectError("VerifySketch", resolveErr)
	}
	doc, loadErr := h.deps.Store.Load(target.scenario, target.page)
	if loadErr != nil {
		return nil, h.connectError("VerifySketch", loadErr)
	}
	fills := make(map[string]reconcile.Fill, len(doc.Placements))
	for _, placement := range doc.Placements {
		fills[placement.Region] = reconcile.Fill{Asset: placement.Fills.Asset, Version: placement.Fills.Version, Placeholder: placement.Fills.Placeholder}
	}
	available := map[string]availability.Result{}
	if h.deps.Availability != nil {
		snapshot, err := h.deps.Availability(ctx)
		if err != nil {
			return nil, h.connectError("VerifySketch", err)
		}
		for region, fill := range fills {
			if fill.Asset != "" {
				available[region] = snapshot.Resolve(ctx, fill.Asset, fill.Version)
			}
		}
	}
	verification := reconcile.Verify(h.deps.Reconciler.ScenariosRoot, target.scenario, target.page, joins, fills, available)
	out := &sketchv1.VerifySketchResponse{Passes: verification.Passes, Coverage: &sketchv1.Coverage{Built: int32(verification.Coverage.Built), Declared: int32(verification.Coverage.Declared), Invented: int32(verification.Coverage.Invented), BuiltPercent: verification.Coverage.BuiltPercent, Missing: int32(verification.Coverage.Missing), Unresolved: int32(verification.Coverage.Unresolved), Total: int32(verification.Coverage.Total), Status: verification.Coverage.Status, LibraryBacked: int32(verification.Coverage.LibraryBacked), Local: int32(verification.Coverage.Local), Resolved: int32(verification.Coverage.Resolved), ResolvedLocal: int32(verification.Coverage.ResolvedLocal)}}

	for _, item := range verification.Regions {
		finding := &sketchv1.Finding{CatalogId: item.Finding.CatalogID, Scope: string(item.Finding.Scope), Blocking: item.Finding.Blocking, Owner: item.Finding.Owner, SeverityClass: string(item.Finding.Severity), Message: item.Finding.Message}
		out.Regions = append(out.Regions, &sketchv1.RegionVerdict{Region: item.Region, LibraryAsset: item.LibraryAsset, LibraryVersion: item.LibraryVersion, LocalComponent: item.LocalComponent, Verdict: string(item.Verdict), JoinRule: item.JoinRule, Proven: item.Proven, FilePath: item.FilePath, ObservedState: string(item.Provenance), Reason: item.Reason, Finding: finding, SelectedAsset: item.SelectedAsset, SelectedVersion: item.SelectedVersion, ObservedAsset: item.ObservedAsset, ObservedVersion: item.ObservedVersion, ReasonCode: item.ReasonCode, EvidenceQuality: item.EvidenceQuality, Candidates: item.Candidates, AvailabilityState: string(item.Availability.State), AvailabilityReasonCode: item.Availability.ReasonCode, BuildHash: item.Availability.BuildHash, SourceHash: item.Availability.SourceHash})
		out.Findings = append(out.Findings, finding)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ImportPage(ctx context.Context, req *connect.Request[sketchv1.ImportPageRequest]) (*connect.Response[sketchv1.ImportPageResponse], error) {
	h, err := h.forRequest(ctx)
	if err != nil {
		return nil, err
	}
	target, err := requestTarget(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	snapshot, err := h.deps.Store.Read(target.scenario, target.page)
	if err != nil {
		return nil, h.connectError("ImportPage", err)
	}
	if req.Msg.GetWrite() && snapshot.ContentHash != req.Msg.GetExpectedContentHash() {
		return nil, h.connectError("ImportPage", &internal.ConflictError{Expected: req.Msg.GetExpectedContentHash(), Current: snapshot.ContentHash})
	}
	if h.deps.ImportSearch == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("import catalog is unavailable"))
	}
	result, err := internal.Import(ctx, snapshot, h.deps.ImportSearch)
	if err != nil {
		return nil, h.connectError("ImportPage", err)
	}
	out := &sketchv1.ImportPageResponse{Sketch: toProto(result.Document), ContentHash: snapshot.ContentHash}
	out.DocumentPath, _ = h.deps.Store.Path(target.scenario, target.page)
	for _, item := range result.Items {
		row := &sketchv1.DecompositionItem{Component: item.Region, Tier: int32(item.Tier), Match: item.Match, State: item.State}
		for _, check := range item.Checks {
			row.Checks = append(row.Checks, &sketchv1.TierCheck{Tier: int32(check.Tier), Ran: true, Matches: check.Matches})
		}
		out.Items = append(out.Items, row)
	}
	for _, candidate := range result.Templates {
		out.Templates = append(out.Templates, &sketchv1.TemplateCandidate{Asset: candidate.CatalogID, Score: candidate.Score, Implemented: candidate.Implemented, MatchingRegions: candidate.Regions})
	}
	if req.Msg.GetWrite() {
		saved, err := h.deps.Store.Save(target.scenario, target.page, req.Msg.GetExpectedContentHash(), result.Document)
		if err != nil {
			return nil, h.connectError("ImportPage", err)
		}
		out.ContentHash = saved.ContentHash
		out.Written = true
		if saved.Changed && h.deps.RecordWrite != nil {
			h.deps.RecordWrite(ctx)
		}
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) Place(ctx context.Context, req *connect.Request[sketchv1.PlaceRequest]) (*connect.Response[sketchv1.PutSketchResponse], error) {
	h, routeErr := h.forRequest(ctx)
	if routeErr != nil {
		return nil, routeErr
	}
	target, err := requestTarget(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	doc, err := h.loadForMutation(target, req.Msg.GetExpectedContentHash())
	if err != nil {
		return nil, h.connectError("Place", err)
	}
	state := "declared"
	if req.Msg.GetVersion() != "" {
		if h.deps.Availability == nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("exact published implementation resolver is unavailable"))
		}
		snapshot, err := h.deps.Availability(ctx)
		if err != nil {
			return nil, h.connectError("Place", err)
		}
		resolved := snapshot.Resolve(ctx, req.Msg.GetAsset(), req.Msg.GetVersion())
		if !resolved.IsBuilt() {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New(resolved.ReasonCode+": "+resolved.Reason))
		}
		state = "built"
	}
	doc, err = internal.Place(doc, internal.Placement{Region: req.Msg.GetRegion(), Fills: internal.Fill{Asset: req.Msg.GetAsset(), Version: req.Msg.GetVersion()}, State: state, Note: req.Msg.GetNote()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return h.save(ctx, target, req.Msg.GetExpectedContentHash(), doc)
}

func (h *connectHandler) Placeholder(ctx context.Context, req *connect.Request[sketchv1.PlaceholderRequest]) (*connect.Response[sketchv1.PutSketchResponse], error) {
	h, routeErr := h.forRequest(ctx)
	if routeErr != nil {
		return nil, routeErr
	}
	target, err := requestTarget(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	doc, err := h.loadForMutation(target, req.Msg.GetExpectedContentHash())
	if err != nil {
		return nil, h.connectError("Placeholder", err)
	}
	doc, err = internal.AddPlaceholder(doc, internal.Placement{Region: req.Msg.GetRegion(), Fills: internal.Fill{Placeholder: req.Msg.GetPlaceholder(), Intent: req.Msg.GetIntent()}, State: "invented", Note: req.Msg.GetNote(), Intent: req.Msg.GetIntent()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return h.save(ctx, target, req.Msg.GetExpectedContentHash(), doc)
}

func (h *connectHandler) AddNote(ctx context.Context, req *connect.Request[sketchv1.AddNoteRequest]) (*connect.Response[sketchv1.PutSketchResponse], error) {
	h, routeErr := h.forRequest(ctx)
	if routeErr != nil {
		return nil, routeErr
	}
	target, err := requestTarget(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	doc, err := h.loadForMutation(target, req.Msg.GetExpectedContentHash())
	if err != nil {
		return nil, h.connectError("AddNote", err)
	}
	scope := req.Msg.GetScope()
	valid := scope == "page" || scope == "region" || scope == "placement"
	if !valid {
		snapshot, err := h.deps.Store.Read(target.scenario, target.page)
		if err != nil {
			return nil, h.connectError("AddNote", err)
		}
		for _, r := range append(snapshot.DeclaredRegions, doc.Regions...) {
			if r.ID == scope {
				valid = true
			}
		}
		for _, p := range doc.Placements {
			if p.Region == scope {
				valid = true
			}
		}
	}
	if !valid || strings.TrimSpace(req.Msg.GetText()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("note requires text and a page or existing region scope"))
	}
	doc = internal.AddNote(doc, internal.Note{Scope: scope, Text: req.Msg.GetText()})
	return h.save(ctx, target, req.Msg.GetExpectedContentHash(), doc)
}

func (h *connectHandler) Unplace(ctx context.Context, req *connect.Request[sketchv1.UnplaceRequest]) (*connect.Response[sketchv1.PutSketchResponse], error) {
	h, routeErr := h.forRequest(ctx)
	if routeErr != nil {
		return nil, routeErr
	}
	target, err := requestTarget(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	doc, err := h.loadForMutation(target, req.Msg.GetExpectedContentHash())
	if err != nil {
		return nil, h.connectError("Unplace", err)
	}
	doc, err = internal.Unplace(doc, req.Msg.GetRegion(), req.Msg.GetReason())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return h.save(ctx, target, req.Msg.GetExpectedContentHash(), doc)
}

func (h *connectHandler) SetTemplate(ctx context.Context, req *connect.Request[sketchv1.SetTemplateRequest]) (*connect.Response[sketchv1.SetTemplateResponse], error) {
	h, err := h.forRequest(ctx)
	if err != nil {
		return nil, err
	}
	target, err := requestTarget(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	doc, err := h.loadForMutation(target, req.Msg.GetExpectedContentHash())
	if err != nil {
		return nil, h.connectError("SetTemplate", err)
	}
	if h.deps.Catalog == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("template catalog unavailable"))
	}
	assets, err := h.deps.Catalog(ctx)
	if err != nil {
		return nil, h.connectError("SetTemplate", err)
	}
	var selected *catalogcoverage.Asset
	for i := range assets {
		if assets[i].ID == req.Msg.GetAsset() {
			selected = &assets[i]
			break
		}
	}
	if selected == nil || selected.Kind != "page-template" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("select an exact catalog page-template identity"))
	}
	if req.Msg.GetVersion() != "" {
		if h.deps.Availability == nil {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("implementation resolver unavailable"))
		}
		available, err := h.deps.Availability(ctx)
		if err != nil {
			return nil, h.connectError("SetTemplate", err)
		}
		resolved := available.Resolve(ctx, selected.ID, req.Msg.GetVersion())
		if !resolved.IsBuilt() {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New(resolved.Reason))
		}
	}
	remaps := make([]internal.RegionRemap, 0, len(req.Msg.Remap))
	for _, r := range req.Msg.Remap {
		remaps = append(remaps, internal.RegionRemap{From: r.From, To: r.To})
	}
	// Apply the selected template's authored filler-domain constraints before
	// publishing any mapped occupant; unavailable identity cannot satisfy them.
	byID := map[string]catalogcoverage.Asset{}
	for _, asset := range assets {
		byID[asset.ID] = asset
	}
	for _, placement := range doc.Placements {
		destination := placement.Region
		for _, remap := range remaps {
			if remap.From == placement.Region {
				destination = remap.To
			}
		}
		accepts := selected.RegionAccepts[destination]
		if accepts != "" && placement.Fills.Asset != "" {
			asset, ok := byID[placement.Fills.Asset]
			if !ok || asset.Domain != accepts {
				return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("region %s accepts %s assets; %s does not satisfy that constraint", destination, accepts, placement.Fills.Asset))
			}
		}
	}
	result, err := internal.ChangeTemplate(doc, internal.AssetRef{Asset: selected.ID, Version: req.Msg.GetVersion()}, selected.Regions, remaps)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := &sketchv1.SetTemplateResponse{Sketch: toProto(result.Document), NewlyUnplaced: toProto(internal.Document{Unplaced: result.NewlyUnplaced}).Unplaced, ContentHash: req.Msg.GetExpectedContentHash(), RequiresConfirmation: len(result.NewlyUnplaced) > 0 && !req.Msg.GetConfirmUnplaced()}
	out.DocumentPath, _ = h.deps.Store.Path(target.scenario, target.page)
	if req.Msg.GetPreview() || out.RequiresConfirmation {
		return connect.NewResponse(out), nil
	}
	saved, err := h.deps.Store.Save(target.scenario, target.page, req.Msg.GetExpectedContentHash(), result.Document)
	if err != nil {
		return nil, h.connectError("SetTemplate", err)
	}
	out.ContentHash = saved.ContentHash
	out.Changed = saved.Changed
	if saved.Changed && h.deps.RecordWrite != nil {
		h.deps.RecordWrite(ctx)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) BuildBrief(ctx context.Context, req *connect.Request[sketchv1.BuildBriefRequest]) (*connect.Response[sketchv1.BuildBriefResponse], error) {
	h, err := h.forRequest(ctx)
	if err != nil {
		return nil, err
	}
	target, err := requestTarget(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	snapshot, err := h.deps.Store.Read(target.scenario, target.page)
	if err != nil {
		return nil, h.connectError("BuildBrief", err)
	}
	markdown, items, closing := internal.BuildBrief(target.scenario, target.page, snapshot.Document)
	response := &sketchv1.BuildBriefResponse{Markdown: markdown, ClosingGate: closing}
	for _, item := range items {
		response.Items = append(response.Items, &sketchv1.BriefItem{Order: item.Order, Tier: item.Tier, Region: item.Region, Action: item.Action, Asset: item.Asset, Constraint: item.Constraint})
	}
	return connect.NewResponse(response), nil
}
