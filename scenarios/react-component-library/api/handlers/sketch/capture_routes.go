package sketch

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	apidb "github.com/vrooli/api-core/database"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"

	"connectrpc.com/connect"

	"react-component-library/internal/designcapture"
	internal "react-component-library/internal/sketch"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
	workspacecapturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacecapture"
)

func captureOperation(op designcapture.Operation) *workspacecapturev1.CaptureOperation {
	t := op.Request.Target
	out := &workspacecapturev1.CaptureOperation{
		Id: op.ID, State: string(op.State), ProducerId: op.ProducerID,
		Candidate: &sharedv1.CandidateReference{Scenario: t.Scenario, DesignId: t.DesignID, Hash: t.Revision},
		Target:    &sharedv1.CompositionRenderTarget{Revision: t.Revision, RenderHash: t.RenderHash, HtmlSha256: t.HTMLSHA256, InputsSha256: t.InputsSHA256, Kind: t.Kind, Kit: t.Kit, Theme: t.Theme, Direction: t.Direction},
		Width:     int32(op.Request.Width), Height: int32(op.Request.Height), Detail: op.Detail, Version: op.Version, PreviousId: op.Request.PreviousID,
	}
	for _, a := range op.Artifacts {
		item := &workspacecapturev1.CaptureArtifact{Kind: a.Kind, Reference: a.Reference}
		if e := a.Evidence; e != nil {
			item.Evidence = &workspacecapturev1.CapturedTargetEvidence{RenderHash: e.RenderHash, Width: e.Width, Height: e.Height}
			for _, r := range e.Regions {
				item.Evidence.Regions = append(item.Evidence.Regions, &workspacecapturev1.CapturedRegionGeometry{Region: r.Region, X: r.X, Y: r.Y, Width: r.Width, Height: r.Height})
			}
		}
		out.Artifacts = append(out.Artifacts, item)
	}
	return out
}

func (h *connectHandler) CaptureCandidate(ctx context.Context, req *connect.Request[workspacecapturev1.CaptureCandidateRequest]) (*connect.Response[workspacecapturev1.CaptureOperation], error) {
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

func (h *connectHandler) GetCapture(ctx context.Context, req *connect.Request[workspacecapturev1.GetCaptureRequest]) (*connect.Response[workspacecapturev1.CaptureOperation], error) {
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

func (h *connectHandler) AttachCapture(ctx context.Context, req *connect.Request[workspacecapturev1.GetCaptureRequest]) (*connect.Response[workspacecapturev1.CaptureOperation], error) {
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

func (h *connectHandler) CancelCapture(ctx context.Context, req *connect.Request[workspacecapturev1.GetCaptureRequest]) (*connect.Response[workspacecapturev1.CaptureOperation], error) {
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

func (h *connectHandler) GetCaptureScreenshot(ctx context.Context, req *connect.Request[workspacecapturev1.GetCaptureScreenshotRequest]) (*connect.Response[workspacecapturev1.CaptureScreenshot], error) {
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
	return connect.NewResponse(&workspacecapturev1.CaptureScreenshot{Reference: shot.Reference, Url: shot.URL, Width: int32(shot.Width), Height: int32(shot.Height), ContentType: shot.ContentType}), nil
}

func (h *connectHandler) RetryCapture(ctx context.Context, req *connect.Request[workspacecapturev1.RetryCaptureRequest]) (*connect.Response[workspacecapturev1.CaptureOperation], error) {
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
