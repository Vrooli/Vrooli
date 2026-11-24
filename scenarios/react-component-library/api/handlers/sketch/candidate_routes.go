package sketch

import (
	"context"
	"errors"
	"fmt"

	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"

	"connectrpc.com/connect"

	"react-component-library/internal/preview"
	internal "react-component-library/internal/sketch"

	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
)

func intentFromProto(i *sharedv1.DesignIntent) internal.DesignIntent {
	out := internal.DesignIntent{Intent: i.GetIntent(), Users: i.GetUsers(), PrimaryTasks: i.GetPrimaryTasks(), Target: i.GetTarget(), Kit: i.GetKit(), RequiredCapabilities: i.GetRequiredCapabilities(), Viewports: i.GetViewports()}
	if c := i.GetConstraints(); c != nil {
		out.Constraints = &internal.DesignConstraints{DesignSource: c.GetDesignSource(), DesignSourceHash: c.GetDesignSourceHash(), PreserveRoutes: c.PreserveRoutes, PreserveBusinessBehavior: c.PreserveBusinessBehavior}
	}
	return out
}

func (h *connectHandler) ProposeSketch(ctx context.Context, req *connect.Request[sharedv1.ProposeSketchRequest]) (*connect.Response[sharedv1.ProposeSketchResponse], error) {
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
	if h.deps.ImportSearch == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("proposal catalog unavailable"))
	}
	search := h.deps.ImportSearch
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
	candidates, err := internal.Propose(ctx, snapshot, intent, int(req.Msg.GetCandidateLimit()), search)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := &sharedv1.ProposeSketchResponse{ContentHash: snapshot.ContentHash, RetrievalMode: "lexical"}
	if source != nil {
		out.DesignSource = &sharedv1.DesignSourceSnapshot{Path: source.Path, ContentHash: source.ContentHash, Content: source.Content}
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
		out.Candidates = append(out.Candidates, &sharedv1.DesignProposal{Title: candidate.Title, Sketch: toProto(candidate.Document), Obligations: candidate.Obligations})
	}
	if len(candidates) == 0 {
		out.Diagnostics = append(out.Diagnostics, "No published template matched the intent and declared compatibility constraints.")
	}
	return connect.NewResponse(out), nil
}

func candidateResponse(scenario string, c internal.Candidate) (*connect.Response[sharedv1.CandidateResponse], error) {
	snapshot, err := c.Snapshot()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var refinement *sharedv1.CandidateRefinement
	if r := c.Refinement; r != nil {
		refinement = &sharedv1.CandidateRefinement{Round: int32(r.Round), Budget: int32(r.Budget), Reason: r.Reason, RequestedRegions: r.RequestedRegions, ChangedRegions: r.ChangedRegions, BroaderChangeReason: r.BroaderChangeReason}
	}
	return connect.NewResponse(&sharedv1.CandidateResponse{Refinement: refinement, Candidate: &sharedv1.CandidateReference{Scenario: scenario, DesignId: c.DesignID, Hash: c.Hash}, Page: c.Page, BaseHash: c.BaseHash, ParentHash: c.ParentHash, Sketch: toProto(snapshot.Document), DeclaredRegions: toProto(internal.Document{Regions: snapshot.DeclaredRegions}).Regions}), nil
}

func (h *connectHandler) SaveCandidate(ctx context.Context, req *connect.Request[sketchv1.SaveCandidateRequest]) (*connect.Response[sharedv1.CandidateResponse], error) {
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

func (h *connectHandler) candidate(ctx context.Context, ref *sharedv1.CandidateReference) (*connectHandler, internal.Candidate, error) {
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

func (h *connectHandler) GetCandidate(ctx context.Context, req *connect.Request[sharedv1.CandidateReference]) (*connect.Response[sharedv1.CandidateResponse], error) {
	_, c, err := h.candidate(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return candidateResponse(req.Msg.GetScenario(), c)
}

func (h *connectHandler) RenderCandidate(ctx context.Context, req *connect.Request[sharedv1.RenderCandidateRequest]) (*connect.Response[sharedv1.RenderCompositionResponse], error) {
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

func (h *connectHandler) MapCandidateRegions(ctx context.Context, req *connect.Request[sketchv1.MapCandidateRegionsRequest]) (*connect.Response[sharedv1.CandidateResponse], error) {
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

func (h *connectHandler) PlaceCandidateAsset(ctx context.Context, req *connect.Request[sketchv1.PlaceCandidateAssetRequest]) (*connect.Response[sharedv1.CandidateResponse], error) {
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

func (h *connectHandler) ListCandidates(ctx context.Context, req *connect.Request[sharedv1.SketchTarget]) (*connect.Response[sketchv1.ListCandidatesResponse], error) {
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
		row := &sketchv1.CandidateSummary{Candidate: &sharedv1.CandidateReference{Scenario: target.scenario, DesignId: c.DesignID, Hash: c.Hash}, ParentHash: c.ParentHash, BaseHash: c.BaseHash}
		if template := snapshot.Document.Template; template != nil {
			row.TemplateAsset = template.Asset
			row.TemplateVersion = template.Version
		}
		out.Candidates = append(out.Candidates, row)
	}
	return connect.NewResponse(out), nil
}
