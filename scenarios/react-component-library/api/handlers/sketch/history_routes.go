package sketch

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	experiencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/experience"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"

	"connectrpc.com/connect"

	internal "react-component-library/internal/sketch"

	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
)

func (h *connectHandler) AdoptExperiencePage(ctx context.Context, req *connect.Request[experiencev1.AdoptExperiencePageRequest]) (*connect.Response[experiencev1.AdoptExperiencePageResponse], error) {
	h, err := h.forRequest(ctx)
	if err != nil {
		return nil, err
	}
	scenario, page := req.Msg.GetScenario(), req.Msg.GetPage()
	if scenario == "" || page == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario and page are required"))
	}
	if h.deps.AdoptExperience == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("experience adoption is unavailable"))
	}
	sourceReader, ok := h.deps.Store.(interface {
		ReadPageSource(string, string) ([]byte, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("experience source reader is unavailable"))
	}
	source, err := sourceReader.ReadPageSource(scenario, page)
	if err != nil {
		return nil, h.connectError("AdoptExperiencePage", err)
	}
	workspace, receiptID, sourceHash, targetHash, receiptJSON, alreadyAdopted, err := h.deps.AdoptExperience(ctx, scenario, page, source)
	if err != nil {
		return nil, h.connectError("AdoptExperiencePage", err)
	}
	if workspace == nil || workspace.GetId() == "" || receiptID == "" {
		return nil, connect.NewError(connect.CodeInternal, errors.New("adoption did not return workspace and recovery receipt"))
	}
	return connect.NewResponse(&experiencev1.AdoptExperiencePageResponse{
		Workspace: workspace, ReceiptId: receiptID, SourceHash: sourceHash,
		TargetHash: targetHash, AlreadyAdopted: alreadyAdopted, ReceiptJson: receiptJSON,
	}), nil
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

func (h *connectHandler) RenderSketch(ctx context.Context, req *connect.Request[sketchv1.RenderSketchRequest]) (*connect.Response[sharedv1.RenderCompositionResponse], error) {
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
