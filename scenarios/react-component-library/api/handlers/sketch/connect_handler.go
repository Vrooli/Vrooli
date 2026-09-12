package sketch

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	"database/sql"
	"fmt"
	apidb "github.com/vrooli/api-core/database"
	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"
	"react-component-library/internal/availability"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/catalogsearch"
	"react-component-library/internal/components"
	"react-component-library/internal/designcapture"
	"react-component-library/internal/designcritique"
	"react-component-library/internal/designinference"
	"react-component-library/internal/preview"
	"react-component-library/internal/reconcile"
	internal "react-component-library/internal/sketch"
	"strings"

	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
)

type CompositionRenderer interface {
	RenderPrepared(context.Context, preview.PreparedComposition, *previewv1.RenderCompositionRequest) (*connect.Response[previewv1.RenderCompositionResponse], error)
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
	ImportSearch      func(context.Context) (*catalogsearch.Index, error)
	Availability      func(context.Context) (*availability.Snapshot, error)
	Store             internal.Repository
	StoreFor          func(context.Context) (internal.Repository, error)
	RecordWrite       func(context.Context)
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
	search, err := h.deps.ImportSearch(ctx)
	if err != nil {
		return nil, h.connectError("ImportPage", err)
	}
	result, err := internal.Import(ctx, snapshot, search)
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

func (h *connectHandler) BuildBrief(context.Context, *connect.Request[sketchv1.BuildBriefRequest]) (*connect.Response[sketchv1.BuildBriefResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("sketch briefs are not implemented"))
}

type target struct{ scenario, page string }

func requestTarget(value *sketchv1.SketchTarget) (target, error) {
	if value == nil || value.GetScenario() == "" || value.GetPage() == "" {
		return target{}, connect.NewError(connect.CodeInvalidArgument, errors.New("target scenario and page are required"))
	}
	return target{scenario: value.GetScenario(), page: value.GetPage()}, nil
}

func (h *connectHandler) connectError(operation string, err error) error {
	if errors.Is(err, designinference.ErrConflict) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var locked *internal.LockedRegionError
	if errors.As(err, &locked) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	var conflict *internal.ConflictError
	if errors.As(err, &conflict) {
		out := connect.NewError(connect.CodeAborted, err)
		if detail, detailErr := connect.NewErrorDetail(&sketchv1.RevisionConflict{ExpectedContentHash: conflict.Expected, CurrentContentHash: conflict.Current}); detailErr == nil {
			out.AddDetail(detail)
		}
		return out
	}
	if errors.Is(err, internal.ErrExpectedRevision) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}

	if errors.Is(err, os.ErrNotExist) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	h.deps.Logger.Printf("sketch.%s: %v", operation, err)
	return connect.NewError(connect.CodeInvalidArgument, err)
}

func (h *connectHandler) save(ctx context.Context, target target, expected string, doc internal.Document) (*connect.Response[sketchv1.PutSketchResponse], error) {
	snapshot, err := h.deps.Store.Save(target.scenario, target.page, expected, doc)
	if err != nil {
		return nil, h.connectError("Save", err)
	}
	if snapshot.Changed && h.deps.RecordWrite != nil {
		h.deps.RecordWrite(ctx)
	}
	path, _ := h.deps.Store.Path(target.scenario, target.page)
	return connect.NewResponse(&sketchv1.PutSketchResponse{Sketch: toProto(snapshot.Document), DocumentPath: path, Changed: snapshot.Changed, ContentHash: snapshot.ContentHash}), nil
}

func toProto(doc internal.Document) *sketchv1.Sketch {
	out := &sketchv1.Sketch{Viewport: doc.Viewport}
	if i := doc.Intent; i != nil {
		out.Intent = &sketchv1.DesignIntent{Intent: i.Intent, Users: i.Users, PrimaryTasks: i.PrimaryTasks, Target: i.Target, Kit: i.Kit, RequiredCapabilities: i.RequiredCapabilities, Viewports: i.Viewports}
		if c := i.Constraints; c != nil {
			out.Intent.Constraints = &sketchv1.DesignConstraints{DesignSource: c.DesignSource, DesignSourceHash: c.DesignSourceHash, PreserveRoutes: c.PreserveRoutes, PreserveBusinessBehavior: c.PreserveBusinessBehavior}
		}
	}
	if doc.Render != nil {
		bindings, _ := structpb.NewStruct(doc.Render.Bindings)
		out.Render = &sketchv1.RenderSettings{TemplateExport: doc.Render.TemplateExport, Bindings: bindings}
		for _, r := range doc.Render.Regions {
			out.Render.Regions = append(out.Render.Regions, &sketchv1.RenderRegion{Id: r.ID, Parent: r.Parent, TemplateRegion: r.TemplateRegion, Story: r.Story, Export: r.Export, Slot: r.Slot, Optional: r.Optional})
		}
		for _, f := range doc.Render.Fixtures {
			out.Render.Fixtures = append(out.Render.Fixtures, &sketchv1.RenderFixture{Target: f.Target, Asset: f.Asset, Version: f.Version, State: f.State, Field: f.Field, Prop: f.Prop})
		}
	}
	if doc.Template != nil {
		out.Template = &sketchv1.AssetReference{Asset: doc.Template.Asset, Version: doc.Template.Version}
	}
	for _, placement := range doc.Placements {
		out.Placements = append(out.Placements, &sketchv1.Placement{Region: placement.Region, Fills: &sketchv1.Fill{Asset: placement.Fills.Asset, Version: placement.Fills.Version, Placeholder: placement.Fills.Placeholder, Intent: placement.Fills.Intent}, State: placement.State, Note: placement.Note, Intent: placement.Intent})
	}
	for _, item := range doc.Unplaced {
		out.Unplaced = append(out.Unplaced, &sketchv1.UnplacedItem{Was: item.Was, Reason: item.Reason, Region: item.Region, Placement: placementToProto(item.Placement)})
	}
	for _, note := range doc.Notes {
		out.Notes = append(out.Notes, &sketchv1.SketchNote{Scope: note.Scope, Text: note.Text})
	}
	for _, region := range doc.Regions {
		item := &sketchv1.SketchRegion{Id: region.ID, Origin: region.Origin, Locked: region.Locked, Note: region.Note, Elements: region.Elements}
		if region.Grid != nil {
			item.Grid = &sketchv1.Grid{X: int32(region.Grid.X), Y: int32(region.Grid.Y), W: int32(region.Grid.W), H: int32(region.Grid.H)}
		}
		out.Regions = append(out.Regions, item)
	}
	return out
}

func fromProto(value *sketchv1.Sketch) internal.Document {
	if value == nil {
		return internal.Document{}
	}
	out := internal.Document{Viewport: value.GetViewport()}
	if value.GetIntent() != nil {
		i := intentFromProto(value.GetIntent())
		out.Intent = &i
	}
	if r := value.GetRender(); r != nil {
		out.Render = &internal.RenderSettings{TemplateExport: r.GetTemplateExport(), Bindings: r.GetBindings().AsMap()}
		for _, p := range r.GetRegions() {
			out.Render.Regions = append(out.Render.Regions, internal.RenderRegion{ID: p.GetId(), Parent: p.GetParent(), TemplateRegion: p.GetTemplateRegion(), Story: p.GetStory(), Export: p.GetExport(), Slot: p.GetSlot(), Optional: p.GetOptional()})
		}
		for _, f := range r.GetFixtures() {
			out.Render.Fixtures = append(out.Render.Fixtures, internal.RenderFixture{Target: f.GetTarget(), Asset: f.GetAsset(), Version: f.GetVersion(), State: f.GetState(), Field: f.GetField(), Prop: f.GetProp()})
		}
	}
	if value.GetTemplate() != nil {
		out.Template = &internal.AssetRef{Asset: value.GetTemplate().GetAsset(), Version: value.GetTemplate().GetVersion()}
	}
	for _, placement := range value.GetPlacements() {
		if placement == nil {
			continue
		}
		fill := placement.GetFills()
		item := internal.Placement{Region: placement.GetRegion(), State: placement.GetState(), Note: placement.GetNote(), Intent: placement.GetIntent()}
		if fill != nil {
			item.Fills = internal.Fill{Asset: fill.GetAsset(), Version: fill.GetVersion(), Placeholder: fill.GetPlaceholder(), Intent: fill.GetIntent()}
		}
		out.Placements = append(out.Placements, item)
	}
	for _, item := range value.GetUnplaced() {
		if item != nil {
			out.Unplaced = append(out.Unplaced, internal.Unplaced{Was: item.GetWas(), Reason: item.GetReason(), Region: item.GetRegion(), Placement: placementFromProto(item.GetPlacement())})
		}
	}
	for _, note := range value.GetNotes() {
		if note != nil {
			out.Notes = append(out.Notes, internal.Note{Scope: note.GetScope(), Text: note.GetText()})
		}
	}
	for _, region := range value.GetRegions() {
		if region == nil {
			continue
		}
		item := internal.Region{ID: region.GetId(), Origin: region.GetOrigin(), Locked: region.GetLocked(), Note: region.GetNote(), Elements: region.GetElements()}
		if region.GetGrid() != nil {
			item.Grid = &internal.Grid{X: int(region.GetGrid().GetX()), Y: int(region.GetGrid().GetY()), W: int(region.GetGrid().GetW()), H: int(region.GetGrid().GetH())}
		}
		out.Regions = append(out.Regions, item)
	}
	return out
}

func placementToProto(value *internal.Placement) *sketchv1.Placement {
	if value == nil {
		return nil
	}
	return toProto(internal.Document{Placements: []internal.Placement{*value}}).Placements[0]
}
func placementFromProto(value *sketchv1.Placement) *internal.Placement {
	if value == nil {
		return nil
	}
	doc := fromProto(&sketchv1.Sketch{Placements: []*sketchv1.Placement{value}})
	return &doc.Placements[0]
}

func (h *connectHandler) loadForMutation(target target, expected string) (internal.Document, error) {
	if expected == "" {
		return internal.Document{}, internal.ErrExpectedRevision
	}
	snapshot, err := h.deps.Store.Read(target.scenario, target.page)
	if err != nil {
		return internal.Document{}, err
	}
	if snapshot.ContentHash != expected {
		return internal.Document{}, &internal.ConflictError{Expected: expected, Current: snapshot.ContentHash}
	}
	return snapshot.Document, nil
}

func (h *connectHandler) GetHistory(ctx context.Context, req *connect.Request[sketchv1.GetHistoryRequest]) (*connect.Response[sketchv1.GetHistoryResponse], error) {
	h, routeErr := h.forRequest(ctx)
	if routeErr != nil {
		return nil, routeErr
	}
	target, err := requestTarget(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	history, err := h.deps.Store.History(target.scenario, target.page)
	if err != nil {
		return nil, h.connectError("GetHistory", err)
	}
	response := &sketchv1.GetHistoryResponse{}
	for _, revision := range history {
		if revision.Current {
			response.CurrentContentHash = revision.ContentHash
		}
		var envelope struct {
			Sketch  internal.Document `json:"sketch"`
			Regions []struct {
				ID, Purpose string
				Elements    []string
			} `json:"regions"`
		}
		if err := json.Unmarshal(revision.Page, &envelope); err != nil {
			return nil, h.connectError("GetHistory", err)
		}
		created := ""
		if !revision.CreatedAt.IsZero() {
			created = revision.CreatedAt.Format(time.RFC3339Nano)
		}
		var declared []internal.Region
		for _, r := range envelope.Regions {
			declared = append(declared, internal.Region{ID: r.ID, Note: r.Purpose, Elements: r.Elements})
		}
		response.Revisions = append(response.Revisions, &sketchv1.SketchRevision{DeclaredRegions: toProto(internal.Document{Regions: declared}).Regions, ContentHash: revision.ContentHash, CreatedFromHash: revision.ParentHash, CreatedAt: created, Sketch: toProto(envelope.Sketch), Current: revision.Current})
	}
	return connect.NewResponse(response), nil
}
func (h *connectHandler) RecoverSketch(ctx context.Context, req *connect.Request[sketchv1.RecoverSketchRequest]) (*connect.Response[sketchv1.PutSketchResponse], error) {
	h, routeErr := h.forRequest(ctx)
	if routeErr != nil {
		return nil, routeErr
	}
	target, err := requestTarget(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	if _, err := h.loadForMutation(target, req.Msg.GetExpectedContentHash()); err != nil {
		return nil, h.connectError("RecoverSketch", err)
	}
	snapshot, err := h.deps.Store.Recover(target.scenario, target.page)
	if err != nil {
		return nil, h.connectError("RecoverSketch", err)
	}
	if snapshot.Changed && h.deps.RecordWrite != nil {
		h.deps.RecordWrite(ctx)
	}
	path, _ := h.deps.Store.Path(target.scenario, target.page)
	return connect.NewResponse(&sketchv1.PutSketchResponse{Sketch: toProto(snapshot.Document), ContentHash: snapshot.ContentHash, Changed: snapshot.Changed, DocumentPath: path}), nil
}

func (h *connectHandler) forRequest(ctx context.Context) (*connectHandler, error) {
	if h.deps.StoreFor == nil {
		return h, nil
	}
	store, err := h.deps.StoreFor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	request := *h
	request.deps.Store = store
	request.deps.StoreFor = nil
	if h.deps.ReconcilerFor != nil {
		resolver, err := h.deps.ReconcilerFor(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		request.deps.Reconciler = resolver
		request.deps.ReconcilerFor = nil
	}
	return &request, nil
}

func (h *connectHandler) ListDesignPages(ctx context.Context, req *connect.Request[sketchv1.ListDesignPagesRequest]) (*connect.Response[sketchv1.ListDesignPagesResponse], error) {
	h, err := h.forRequest(ctx)
	if err != nil {
		return nil, err
	}
	reader, ok := h.deps.Store.(internal.WorkspaceReader)
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("design workspace inventory unavailable"))
	}
	inventory, err := reader.ListDesignPages(ctx, req.Msg.GetScenario())
	if err != nil {
		return nil, h.connectError("ListDesignPages", err)
	}
	out := &sketchv1.ListDesignPagesResponse{Issues: inventory.Issues}
	for _, scenario := range inventory.Scenarios {
		out.Scenarios = append(out.Scenarios, &sketchv1.DesignScenario{Scenario: scenario.Scenario, PageCount: int32(scenario.PageCount), Issue: scenario.Issue})
	}
	for _, page := range inventory.Pages {
		out.Pages = append(out.Pages, &sketchv1.DesignPage{Page: page.Page, Title: page.Title, Route: page.Route, Status: page.Status, ContentHash: page.ContentHash, RegionCount: int32(page.RegionCount), Issue: page.Issue, Registered: page.Registered, Routes: page.Routes})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) RenderSketch(ctx context.Context, req *connect.Request[sketchv1.RenderSketchRequest]) (*connect.Response[previewv1.RenderCompositionResponse], error) {
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
		return nil, h.connectError("RenderSketch", err)
	}
	if req.Msg.GetExpectedContentHash() == "" {
		return nil, h.connectError("RenderSketch", internal.ErrExpectedRevision)
	}
	if snapshot.ContentHash != req.Msg.GetExpectedContentHash() {
		return nil, h.connectError("RenderSketch", &internal.ConflictError{Expected: req.Msg.GetExpectedContentHash(), Current: snapshot.ContentHash})
	}
	if h.deps.Renderer == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("composition renderer unavailable"))
	}
	prepared, err := internal.PrepareRender(snapshot, req.Msg.GetMissingLabel(), req.Msg.GetFailedLabel())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	result, err := h.deps.Renderer.RenderPrepared(ctx, prepared, &previewv1.RenderCompositionRequest{Kit: req.Msg.GetKit(), Theme: req.Msg.GetTheme(), Direction: req.Msg.GetDirection()})
	if err != nil {
		return nil, err
	}
	// A slow bundle must not return apparently current evidence after an edit.
	current, err := h.deps.Store.Read(target.scenario, target.page)
	if err != nil {
		return nil, h.connectError("RenderSketch", err)
	}
	if current.ContentHash != snapshot.ContentHash {
		return nil, h.connectError("RenderSketch", &internal.ConflictError{Expected: snapshot.ContentHash, Current: current.ContentHash})
	}
	return result, nil
}

func intentFromProto(i *sketchv1.DesignIntent) internal.DesignIntent {
	out := internal.DesignIntent{Intent: i.GetIntent(), Users: i.GetUsers(), PrimaryTasks: i.GetPrimaryTasks(), Target: i.GetTarget(), Kit: i.GetKit(), RequiredCapabilities: i.GetRequiredCapabilities(), Viewports: i.GetViewports()}
	if c := i.GetConstraints(); c != nil {
		out.Constraints = &internal.DesignConstraints{DesignSource: c.GetDesignSource(), DesignSourceHash: c.GetDesignSourceHash(), PreserveRoutes: c.PreserveRoutes, PreserveBusinessBehavior: c.PreserveBusinessBehavior}
	}
	return out
}
func (h *connectHandler) ProposeSketch(ctx context.Context, req *connect.Request[sketchv1.ProposeSketchRequest]) (*connect.Response[sketchv1.ProposeSketchResponse], error) {
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
		return nil, h.connectError("ProposeSketch", err)
	}
	if req.Msg.GetExpectedContentHash() == "" {
		return nil, h.connectError("ProposeSketch", internal.ErrExpectedRevision)
	}
	if snapshot.ContentHash != req.Msg.GetExpectedContentHash() {
		return nil, h.connectError("ProposeSketch", &internal.ConflictError{Expected: req.Msg.GetExpectedContentHash(), Current: snapshot.ContentHash})
	}
	if h.deps.Catalog == nil || h.deps.ImportSearch == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("proposal catalog unavailable"))
	}
	catalog, err := h.deps.Catalog(ctx)
	if err != nil {
		return nil, h.connectError("ProposeSketch", err)
	}
	search, err := h.deps.ImportSearch(ctx)
	if err != nil {
		return nil, h.connectError("ProposeSketch", err)
	}
	intent := intentFromProto(req.Msg.GetIntent())
	var source *internal.DesignSourceSnapshot
	if intent.Constraints != nil {
		source, err = h.deps.Store.ReadDesignSource(target.scenario, intent.Constraints.DesignSource)
		if err != nil {
			return nil, h.connectError("ProposeSketch", err)
		}
		if intent.Constraints.DesignSourceHash != "" && (source == nil || source.ContentHash != intent.Constraints.DesignSourceHash) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("design source changed; read the current source before proposing"))
		}
		intent.Constraints.DesignSourceHash = ""
		if source != nil {
			intent.Constraints.DesignSource = source.Path
			intent.Constraints.DesignSourceHash = source.ContentHash
		}
	}
	candidates, err := internal.Propose(ctx, snapshot, intent, int(req.Msg.GetCandidateLimit()), search, catalog)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := &sketchv1.ProposeSketchResponse{ContentHash: snapshot.ContentHash, RetrievalMode: "lexical"}
	if source != nil {
		out.DesignSource = &sketchv1.DesignSourceSnapshot{Path: source.Path, ContentHash: source.ContentHash, Content: source.Content}
	}
	for _, candidate := range candidates {
		if h.deps.Assets != nil {
			configured, obligations, err := internal.ConfigureTemplateRender(ctx, candidate.Document, h.deps.Assets)
			if err != nil {
				candidate.Obligations = append(candidate.Obligations, err.Error())
			} else {
				candidate.Document = configured
				candidate.Obligations = append(candidate.Obligations[1:], obligations...)
			}
		}
		out.Candidates = append(out.Candidates, &sketchv1.DesignProposal{Title: candidate.Title, Sketch: toProto(candidate.Document), Obligations: candidate.Obligations})
	}
	if len(candidates) == 0 {
		out.Diagnostics = append(out.Diagnostics, "No published template matched the intent and declared compatibility constraints.")
	}
	return connect.NewResponse(out), nil
}

func candidateResponse(scenario string, c internal.Candidate) (*connect.Response[sketchv1.CandidateResponse], error) {
	snapshot, err := c.Snapshot()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var refinement *sketchv1.CandidateRefinement
	if r := c.Refinement; r != nil {
		refinement = &sketchv1.CandidateRefinement{Round: int32(r.Round), Budget: int32(r.Budget), Reason: r.Reason, RequestedRegions: r.RequestedRegions, ChangedRegions: r.ChangedRegions, BroaderChangeReason: r.BroaderChangeReason}
	}
	return connect.NewResponse(&sketchv1.CandidateResponse{Refinement: refinement, Candidate: &sketchv1.CandidateReference{Scenario: scenario, DesignId: c.DesignID, Hash: c.Hash}, Page: c.Page, BaseHash: c.BaseHash, ParentHash: c.ParentHash, Sketch: toProto(snapshot.Document), DeclaredRegions: toProto(internal.Document{Regions: snapshot.DeclaredRegions}).Regions}), nil
}
func (h *connectHandler) SaveCandidate(ctx context.Context, req *connect.Request[sketchv1.SaveCandidateRequest]) (*connect.Response[sketchv1.CandidateResponse], error) {
	h, err := h.forRequest(ctx)
	if err != nil {
		return nil, err
	}
	target, err := requestTarget(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	if req.Msg.GetSketch() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("candidate sketch is required"))
	}
	store, ok := h.deps.Store.(internal.CandidateRepository)
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("candidate repository unavailable"))
	}
	c, err := store.SaveCandidate(target.scenario, target.page, req.Msg.GetDesignId(), req.Msg.GetExpectedContentHash(), fromProto(req.Msg.GetSketch()))
	if err != nil {
		return nil, h.connectError("SaveCandidate", err)
	}
	if h.deps.RecordWrite != nil {
		h.deps.RecordWrite(ctx)
	}
	return candidateResponse(target.scenario, c)
}
func (h *connectHandler) candidate(ctx context.Context, ref *sketchv1.CandidateReference) (*connectHandler, internal.Candidate, error) {
	h, err := h.forRequest(ctx)
	if err != nil {
		return nil, internal.Candidate{}, err
	}
	store, ok := h.deps.Store.(internal.CandidateRepository)
	if !ok {
		return nil, internal.Candidate{}, connect.NewError(connect.CodeUnavailable, errors.New("candidate repository unavailable"))
	}
	c, err := store.ReadCandidate(ref.GetScenario(), ref.GetDesignId(), ref.GetHash())
	if err != nil {
		return nil, c, h.connectError("GetCandidate", err)
	}
	return h, c, nil
}
func (h *connectHandler) GetCandidate(ctx context.Context, req *connect.Request[sketchv1.CandidateReference]) (*connect.Response[sketchv1.CandidateResponse], error) {
	_, c, err := h.candidate(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return candidateResponse(req.Msg.GetScenario(), c)
}
func (h *connectHandler) RenderCandidate(ctx context.Context, req *connect.Request[sketchv1.RenderCandidateRequest]) (*connect.Response[previewv1.RenderCompositionResponse], error) {
	h, c, err := h.candidate(ctx, req.Msg.GetCandidate())
	if err != nil {
		return nil, err
	}
	if h.deps.Renderer == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("composition renderer unavailable"))
	}
	snapshot, err := c.Snapshot()
	if err != nil {
		return nil, h.connectError("RenderCandidate", err)
	}
	prepared, err := internal.PrepareRender(snapshot, req.Msg.GetMissingLabel(), req.Msg.GetFailedLabel())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	prepared, err = preview.SelectCompositionPreviewState(prepared, req.Msg.GetPreviewState())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return h.deps.Renderer.RenderPrepared(ctx, prepared, &previewv1.RenderCompositionRequest{Kit: req.Msg.GetKit(), Theme: req.Msg.GetTheme(), Direction: req.Msg.GetDirection()})
}

func (h *connectHandler) MapCandidateRegions(ctx context.Context, req *connect.Request[sketchv1.MapCandidateRegionsRequest]) (*connect.Response[sketchv1.CandidateResponse], error) {
	h, c, err := h.candidate(ctx, req.Msg.GetCandidate())
	if err != nil {
		return nil, err
	}
	source, err := c.Snapshot()
	if err != nil {
		return nil, h.connectError("MapCandidateRegions", err)
	}
	var mappings []internal.PortMapping
	for _, m := range req.Msg.GetMappings() {
		mappings = append(mappings, internal.PortMapping{Region: m.GetRegion(), TemplateRegion: m.GetTemplateRegion()})
	}
	doc, err := internal.MapRegionPorts(source, mappings)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	store, ok := h.deps.Store.(interface {
		DeriveCandidate(string, string, string, internal.Document) (internal.Candidate, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("candidate branching unavailable"))
	}
	next, err := store.DeriveCandidate(req.Msg.GetCandidate().GetScenario(), c.DesignID, c.Hash, doc)
	if err != nil {
		return nil, h.connectError("MapCandidateRegions", err)
	}
	if h.deps.RecordWrite != nil {
		h.deps.RecordWrite(ctx)
	}
	return candidateResponse(req.Msg.GetCandidate().GetScenario(), next)
}

func (h *connectHandler) PlaceCandidateAsset(ctx context.Context, req *connect.Request[sketchv1.PlaceCandidateAssetRequest]) (*connect.Response[sketchv1.CandidateResponse], error) {
	h, c, err := h.candidate(ctx, req.Msg.GetCandidate())
	if err != nil {
		return nil, err
	}
	if h.deps.Availability == nil || h.deps.Catalog == nil || h.deps.Assets == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("published asset resolution unavailable"))
	}
	ref := internal.AssetRef{Asset: req.Msg.GetAsset().GetAsset(), Version: req.Msg.GetAsset().GetVersion()}
	if ref.Version == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("exact asset version is required"))
	}
	availability, err := h.deps.Availability(ctx)
	if err != nil {
		return nil, h.connectError("PlaceCandidateAsset", err)
	}
	resolved := availability.Resolve(ctx, ref.Asset, ref.Version)
	if !resolved.IsBuilt() || resolved.Version != ref.Version {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("selected asset is not available at %s: %s", ref.Version, resolved.Reason))
	}
	catalog, err := h.deps.Catalog(ctx)
	if err != nil {
		return nil, h.connectError("PlaceCandidateAsset", err)
	}
	source, err := c.Snapshot()
	if err != nil {
		return nil, h.connectError("PlaceCandidateAsset", err)
	}
	doc, err := internal.SelectRegionAsset(ctx, source, req.Msg.GetRegion(), ref, req.Msg.GetStory(), h.deps.Assets, catalog)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	store, ok := h.deps.Store.(interface {
		DeriveCandidate(string, string, string, internal.Document) (internal.Candidate, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("candidate branching unavailable"))
	}
	next, err := store.DeriveCandidate(req.Msg.GetCandidate().GetScenario(), c.DesignID, c.Hash, doc)
	if err != nil {
		return nil, h.connectError("PlaceCandidateAsset", err)
	}
	if h.deps.RecordWrite != nil {
		h.deps.RecordWrite(ctx)
	}
	return candidateResponse(req.Msg.GetCandidate().GetScenario(), next)
}

func (h *connectHandler) ListCandidates(ctx context.Context, req *connect.Request[sketchv1.SketchTarget]) (*connect.Response[sketchv1.ListCandidatesResponse], error) {
	h, err := h.forRequest(ctx)
	if err != nil {
		return nil, err
	}
	target, err := requestTarget(req.Msg)
	if err != nil {
		return nil, err
	}
	store, ok := h.deps.Store.(interface {
		ListCandidates(string, string) ([]internal.Candidate, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("candidate inventory unavailable"))
	}
	candidates, err := store.ListCandidates(target.scenario, target.page)
	if err != nil {
		return nil, h.connectError("ListCandidates", err)
	}
	out := &sketchv1.ListCandidatesResponse{}
	for _, c := range candidates {
		snapshot, err := c.Snapshot()
		if err != nil {
			return nil, h.connectError("ListCandidates", err)
		}
		row := &sketchv1.CandidateSummary{Candidate: &sketchv1.CandidateReference{Scenario: target.scenario, DesignId: c.DesignID, Hash: c.Hash}, ParentHash: c.ParentHash, BaseHash: c.BaseHash}
		if template := snapshot.Document.Template; template != nil {
			row.TemplateAsset = template.Asset
			row.TemplateVersion = template.Version
		}
		out.Candidates = append(out.Candidates, row)
	}
	return connect.NewResponse(out), nil
}

func captureOperation(op designcapture.Operation) *sketchv1.CaptureOperation {
	t := op.Request.Target
	out := &sketchv1.CaptureOperation{Id: op.ID, State: string(op.State), ProducerId: op.ProducerID,
		Candidate: &sketchv1.CandidateReference{Scenario: t.Scenario, DesignId: t.DesignID, Hash: t.Revision},
		Target:    &previewv1.CompositionRenderTarget{Revision: t.Revision, RenderHash: t.RenderHash, HtmlSha256: t.HTMLSHA256, InputsSha256: t.InputsSHA256, Kind: t.Kind, Kit: t.Kit, Theme: t.Theme, Direction: t.Direction},
		Width:     int32(op.Request.Width), Height: int32(op.Request.Height), Detail: op.Detail, Version: op.Version, PreviousId: op.Request.PreviousID}
	for _, a := range op.Artifacts {
		item := &sketchv1.CaptureArtifact{Kind: a.Kind, Reference: a.Reference}
		if e := a.Evidence; e != nil {
			item.Evidence = &sketchv1.CapturedTargetEvidence{RenderHash: e.RenderHash, Width: e.Width, Height: e.Height}
			for _, r := range e.Regions {
				item.Evidence.Regions = append(item.Evidence.Regions, &sketchv1.CapturedRegionGeometry{Region: r.Region, X: r.X, Y: r.Y, Width: r.Width, Height: r.Height})
			}
		}
		out.Artifacts = append(out.Artifacts, item)
	}
	return out
}
func (h *connectHandler) CaptureCandidate(ctx context.Context, req *connect.Request[sketchv1.CaptureCandidateRequest]) (*connect.Response[sketchv1.CaptureOperation], error) {
	if h.deps.CapturesFor == nil || h.deps.CaptureDispatcher == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("capture service unavailable"))
	}
	// BAS target navigation currently carries no routed-storage lease. Refuse
	// test dispatch until that producer boundary propagates the same route.
	if apidb.IsTestMode(ctx) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("browser capture requires a propagated storage lease"))
	}
	if req.Msg.GetRender() == nil || req.Msg.GetExpectedRenderHash() == "" || (req.Msg.GetIdempotencyKey() == "" || len(req.Msg.GetIdempotencyKey()) > 200) || req.Msg.GetWidth() < 100 || req.Msg.GetWidth() > 4000 || req.Msg.GetHeight() < 100 || req.Msg.GetHeight() > 4000 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("exact render target, idempotency key, and bounded viewport are required"))
	}
	repo, err := h.deps.CapturesFor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	rendered, err := h.RenderCandidate(ctx, connect.NewRequest(req.Msg.GetRender()))
	if err != nil {
		return nil, err
	}
	t := rendered.Msg.GetTarget()
	if t == nil || t.GetRenderHash() != req.Msg.GetExpectedRenderHash() || t.GetRevision() != req.Msg.GetRender().GetCandidate().GetHash() {
		conflict := connect.NewError(connect.CodeAborted, fmt.Errorf("capture render target is stale: expected %s at %s; rendered %s at %s (inputs %s)", req.Msg.GetExpectedRenderHash(), req.Msg.GetRender().GetCandidate().GetHash(), t.GetRenderHash(), t.GetRevision(), t.GetInputsSha256()))
		if t != nil {
			if detail, err := connect.NewErrorDetail(t); err == nil {
				conflict.AddDetail(detail)
			}
		}
		return nil, conflict
	}
	ref := req.Msg.GetRender().GetCandidate()
	input := designcapture.Request{Target: designcapture.Target{Scenario: ref.GetScenario(), DesignID: ref.GetDesignId(), Revision: t.GetRevision(), RenderHash: t.GetRenderHash(), HTMLSHA256: t.GetHtmlSha256(), InputsSHA256: t.GetInputsSha256(), Kind: t.GetKind(), Kit: t.GetKit(), Theme: t.GetTheme(), Direction: t.GetDirection()}, HTML: rendered.Msg.GetHtml(), Width: int(req.Msg.GetWidth()), Height: int(req.Msg.GetHeight())}
	op, err := (designcapture.Service{Repository: repo, Dispatcher: h.deps.CaptureDispatcher}).Start(ctx, req.Msg.GetIdempotencyKey(), input)
	if errors.Is(err, designcapture.ErrConflict) {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(captureOperation(op)), nil
}
func (h *connectHandler) GetCapture(ctx context.Context, req *connect.Request[sketchv1.GetCaptureRequest]) (*connect.Response[sketchv1.CaptureOperation], error) {
	if h.deps.CapturesFor == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("capture repository unavailable"))
	}
	repo, err := h.deps.CapturesFor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	op, err := repo.Get(ctx, req.Msg.GetId())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(captureOperation(op)), nil
}

func (h *connectHandler) AttachCapture(ctx context.Context, req *connect.Request[sketchv1.GetCaptureRequest]) (*connect.Response[sketchv1.CaptureOperation], error) {
	observer, ok := h.deps.CaptureDispatcher.(designcapture.Observer)
	if h.deps.CapturesFor == nil || !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("capture attachment unavailable"))
	}
	repo, err := h.deps.CapturesFor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	observeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	op, err := (designcapture.Service{Repository: repo}).Attach(observeCtx, req.Msg.GetId(), observer)
	if errors.Is(err, designcapture.ErrConflict) {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(captureOperation(op)), nil
}

func (h *connectHandler) CancelCapture(ctx context.Context, req *connect.Request[sketchv1.GetCaptureRequest]) (*connect.Response[sketchv1.CaptureOperation], error) {
	canceller, ok := h.deps.CaptureDispatcher.(designcapture.Canceller)
	if h.deps.CapturesFor == nil || !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("capture cancellation unavailable"))
	}
	repo, err := h.deps.CapturesFor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	op, err := (designcapture.Service{Repository: repo}).Cancel(ctx, req.Msg.GetId(), canceller)
	if errors.Is(err, designcapture.ErrConflict) {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(captureOperation(op)), nil
}

func (h *connectHandler) GetCaptureScreenshot(ctx context.Context, req *connect.Request[sketchv1.GetCaptureScreenshotRequest]) (*connect.Response[sketchv1.CaptureScreenshot], error) {
	resolver, ok := h.deps.CaptureDispatcher.(designcapture.ScreenshotResolver)
	if h.deps.CapturesFor == nil || !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("capture screenshot retrieval unavailable"))
	}
	repo, err := h.deps.CapturesFor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	shot, err := (designcapture.Service{Repository: repo}).Screenshot(lookupCtx, req.Msg.GetId(), req.Msg.GetReference(), resolver)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&sketchv1.CaptureScreenshot{Reference: shot.Reference, Url: shot.URL, Width: int32(shot.Width), Height: int32(shot.Height), ContentType: shot.ContentType}), nil
}

func (h *connectHandler) RetryCapture(ctx context.Context, req *connect.Request[sketchv1.RetryCaptureRequest]) (*connect.Response[sketchv1.CaptureOperation], error) {
	if h.deps.CapturesFor == nil || h.deps.CaptureDispatcher == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("capture retry unavailable"))
	}
	if apidb.IsTestMode(ctx) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("browser capture does not yet propagate test storage leases"))
	}
	if len(req.Msg.GetIdempotencyKey()) < 1 || len(req.Msg.GetIdempotencyKey()) > 200 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("idempotency key must contain 1 to 200 bytes"))
	}
	repo, err := h.deps.CapturesFor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	op, err := (designcapture.Service{Repository: repo, Dispatcher: h.deps.CaptureDispatcher}).Retry(ctx, req.Msg.GetId(), req.Msg.GetIdempotencyKey())
	if errors.Is(err, designcapture.ErrConflict) {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(captureOperation(op)), nil
}

func (h *connectHandler) RefineCandidate(ctx context.Context, req *connect.Request[sketchv1.RefineCandidateRequest]) (*connect.Response[sketchv1.RefineCandidateResponse], error) {
	if req.Msg.GetSketch() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("candidate refinement sketch is required"))
	}
	h, _, err := h.candidate(ctx, req.Msg.GetCandidate())
	if err != nil {
		return nil, err
	}
	store, ok := h.deps.Store.(interface {
		RefineCandidate(string, string, string, internal.RefinementRequest) (internal.RefinementResult, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("candidate refinement unavailable"))
	}
	ref := req.Msg.GetCandidate()
	result, err := store.RefineCandidate(ref.GetScenario(), ref.GetDesignId(), ref.GetHash(), internal.RefinementRequest{Document: fromProto(req.Msg.GetSketch()), RequestedRegions: req.Msg.GetRequestedRegions(), Reason: req.Msg.GetReason(), BroaderChangeReason: req.Msg.GetBroaderChangeReason(), AdditionalRounds: int(req.Msg.GetAdditionalRounds())})
	if err != nil {
		return nil, h.connectError("RefineCandidate", err)
	}
	converted, err := candidateResponse(ref.GetScenario(), result.Candidate)
	if err != nil {
		return nil, err
	}
	if result.Status == "refined" && h.deps.RecordWrite != nil {
		h.deps.RecordWrite(ctx)
	}
	return connect.NewResponse(&sketchv1.RefineCandidateResponse{Result: converted.Msg, Status: result.Status, Round: int32(result.Round), Budget: int32(result.Budget)}), nil
}
