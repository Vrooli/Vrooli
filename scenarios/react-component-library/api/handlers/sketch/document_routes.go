package sketch

import (
	"context"
	"database/sql"
	"errors"
	"os"

	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
	"google.golang.org/protobuf/types/known/structpb"

	"connectrpc.com/connect"

	"react-component-library/internal/designinference"
	internal "react-component-library/internal/sketch"
	"react-component-library/internal/workspaces"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
)

type target struct{ scenario, page string }

func requestTarget(value *sharedv1.SketchTarget) (target, error) {
	if value == nil || value.GetScenario() == "" || value.GetPage() == "" {
		return target{}, connect.NewError(connect.CodeInvalidArgument, errors.New("target scenario and page are required"))
	}
	return target{scenario: value.GetScenario(), page: value.GetPage()}, nil
}

func (h *connectHandler) connectError(operation string, err error) error {
	// A coded error already carries an intentional classification from the
	// call site (validation, unavailable, and so on). Preserve it instead of
	// reclassifying below.
	var coded *connect.Error
	if errors.As(err, &coded) {
		return coded
	}
	if errors.Is(err, designinference.ErrConflict) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	if errors.Is(err, workspaces.ErrConversionReceiptConflict) {
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
	// Everything else is a server-side failure: a disk/permission/parse fault
	// or an unavailable dependency. Labelling it InvalidArgument told the
	// client its request was malformed and made retry policies fire on the
	// wrong class.
	h.deps.Logger.Printf("sketch.%s: %v", operation, err)
	return connect.NewError(connect.CodeInternal, err)
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

func toProto(doc internal.Document) *sharedv1.Sketch {
	out := &sharedv1.Sketch{Viewport: doc.Viewport}
	if i := doc.Intent; i != nil {
		out.Intent = &sharedv1.DesignIntent{Intent: i.Intent, Users: i.Users, PrimaryTasks: i.PrimaryTasks, Target: i.Target, Kit: i.Kit, RequiredCapabilities: i.RequiredCapabilities, Viewports: i.Viewports}
		if c := i.Constraints; c != nil {
			out.Intent.Constraints = &sharedv1.DesignConstraints{DesignSource: c.DesignSource, DesignSourceHash: c.DesignSourceHash, PreserveRoutes: c.PreserveRoutes, PreserveBusinessBehavior: c.PreserveBusinessBehavior}
		}
	}
	if doc.Render != nil {
		bindings, _ := structpb.NewStruct(doc.Render.Bindings)
		out.Render = &sharedv1.RenderSettings{TemplateExport: doc.Render.TemplateExport, Bindings: bindings}
		for _, r := range doc.Render.Regions {
			out.Render.Regions = append(out.Render.Regions, &sharedv1.RenderRegion{Id: r.ID, Parent: r.Parent, TemplateRegion: r.TemplateRegion, Story: r.Story, Export: r.Export, Slot: r.Slot, Optional: r.Optional})
		}
		for _, f := range doc.Render.Fixtures {
			out.Render.Fixtures = append(out.Render.Fixtures, &sharedv1.RenderFixture{Target: f.Target, Asset: f.Asset, Version: f.Version, State: f.State, Field: f.Field, Prop: f.Prop})
		}
	}
	if doc.Template != nil {
		out.Template = &sharedv1.AssetReference{Asset: doc.Template.Asset, Version: doc.Template.Version}
	}
	for _, placement := range doc.Placements {
		out.Placements = append(out.Placements, &sharedv1.Placement{Region: placement.Region, Fills: &sharedv1.Fill{Asset: placement.Fills.Asset, Version: placement.Fills.Version, Placeholder: placement.Fills.Placeholder, Intent: placement.Fills.Intent}, State: placement.State, Note: placement.Note, Intent: placement.Intent})
	}
	for _, item := range doc.Unplaced {
		out.Unplaced = append(out.Unplaced, &sharedv1.UnplacedItem{Was: item.Was, Reason: item.Reason, Region: item.Region, Placement: placementToProto(item.Placement)})
	}
	for _, note := range doc.Notes {
		out.Notes = append(out.Notes, &sharedv1.SketchNote{Scope: note.Scope, Text: note.Text})
	}
	for _, region := range doc.Regions {
		item := &sharedv1.SketchRegion{Id: region.ID, Origin: region.Origin, Locked: region.Locked, Note: region.Note, Elements: region.Elements}
		if region.Grid != nil {
			item.Grid = &sharedv1.Grid{X: int32(region.Grid.X), Y: int32(region.Grid.Y), W: int32(region.Grid.W), H: int32(region.Grid.H)}
		}
		out.Regions = append(out.Regions, item)
	}
	return out
}

func fromProto(value *sharedv1.Sketch) internal.Document {
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

func placementToProto(value *internal.Placement) *sharedv1.Placement {
	if value == nil {
		return nil
	}
	return toProto(internal.Document{Placements: []internal.Placement{*value}}).Placements[0]
}

func placementFromProto(value *sharedv1.Placement) *internal.Placement {
	if value == nil {
		return nil
	}
	doc := fromProto(&sharedv1.Sketch{Placements: []*sharedv1.Placement{value}})
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
