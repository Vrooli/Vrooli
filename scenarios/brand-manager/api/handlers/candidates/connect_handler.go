package candidates

import (
	"context"
	"errors"
	"log"

	"brand-manager/internal/candidates"

	"connectrpc.com/connect"

	candsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/candidates"
	candsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/candidates/candidates_v1connect"
)

// Deps wires the handler's seams.
type Deps struct {
	Service *candidates.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the CandidatesService handler.
func NewConnectHandler(d Deps) *connectHandler { return &connectHandler{deps: d} }

var _ candsconnect.CandidatesServiceHandler = (*connectHandler)(nil)

func (h *connectHandler) ExploreCandidates(ctx context.Context, req *connect.Request[candsv1.ExploreCandidatesRequest]) (*connect.Response[candsv1.ExploreCandidatesResponse], error) {
	m := req.Msg
	list, warnings, err := h.deps.Service.Explore(ctx, candidates.ExploreInput{
		BrandID:        m.GetBrandId(),
		Brief:          m.GetBrief(),
		Concepts:       m.GetConcepts(),
		Variations:     int(m.GetVariations()),
		PreferVector:   m.GetPreferVector(),
		QualityPolicy:  m.GetQualityPolicy(),
		FallbackPolicy: m.GetFallbackPolicy(),
		AllowBYOK:      m.GetAllowByok(),
		Role:           m.GetRole(),
		// Style references resolve before any generation starts, so a bad
		// reference fails the call before anything is paid for.
		StyleReferenceBrand:   m.GetStyleReferenceBrand(),
		StyleReferenceAssetID: m.GetStyleReferenceAssetId(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &candsv1.ExploreCandidatesResponse{Warnings: warnings}
	for _, c := range list {
		resp.Candidates = append(resp.Candidates, candidateToProto(c))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ImportCandidate(ctx context.Context, req *connect.Request[candsv1.ImportCandidateRequest]) (*connect.Response[candsv1.ImportCandidateResponse], error) {
	m := req.Msg
	c, err := h.deps.Service.Import(ctx, m.GetBrandId(), m.GetAssetId(), m.GetConcept(), m.GetParentId(), m.GetNote())
	if err != nil {
		return nil, translate(err)
	}
	return connect.NewResponse(&candsv1.ImportCandidateResponse{Candidate: candidateToProto(c)}), nil
}

func (h *connectHandler) ListCandidates(ctx context.Context, req *connect.Request[candsv1.ListCandidatesRequest]) (*connect.Response[candsv1.ListCandidatesResponse], error) {
	list, err := h.deps.Service.List(ctx, req.Msg.GetBrandId(), statusFromProto(req.Msg.GetStatus()), int(req.Msg.GetLimit()), int(req.Msg.GetOffset()))
	if err != nil {
		return nil, translate(err)
	}
	resp := &candsv1.ListCandidatesResponse{}
	for _, c := range list {
		resp.Candidates = append(resp.Candidates, candidateToProto(c))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetCandidate(ctx context.Context, req *connect.Request[candsv1.GetCandidateRequest]) (*connect.Response[candsv1.GetCandidateResponse], error) {
	c, err := h.deps.Service.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, translate(err)
	}
	return connect.NewResponse(&candsv1.GetCandidateResponse{Candidate: candidateToProto(c)}), nil
}

func (h *connectHandler) RefineCandidate(ctx context.Context, req *connect.Request[candsv1.RefineCandidateRequest]) (*connect.Response[candsv1.RefineCandidateResponse], error) {
	m := req.Msg
	action := candidates.RefineAction{}
	switch a := m.GetAction().(type) {
	case *candsv1.RefineCandidateRequest_Instruction:
		action.Instruction = a.Instruction
	case *candsv1.RefineCandidateRequest_MaskAssetId:
		action.MaskAssetID = a.MaskAssetId
	case *candsv1.RefineCandidateRequest_RemoveBackground:
		action.RemoveBackground = a.RemoveBackground
	case *candsv1.RefineCandidateRequest_Vectorize:
		action.Vectorize = vectorizeFromProto(a.Vectorize)
	}
	c, err := h.deps.Service.Refine(ctx, m.GetCandidateId(), action)
	if err != nil {
		return nil, translate(err)
	}
	return connect.NewResponse(&candsv1.RefineCandidateResponse{Candidate: candidateToProto(c)}), nil
}

func (h *connectHandler) PickCandidate(ctx context.Context, req *connect.Request[candsv1.PickCandidateRequest]) (*connect.Response[candsv1.PickCandidateResponse], error) {
	c, markID, vectorized, err := h.deps.Service.Pick(ctx, req.Msg.GetCandidateId())
	if err != nil {
		return nil, translate(err)
	}
	return connect.NewResponse(&candsv1.PickCandidateResponse{Candidate: candidateToProto(c), MarkAssetId: markID, Vectorized: vectorized}), nil
}

func (h *connectHandler) RejectCandidate(ctx context.Context, req *connect.Request[candsv1.RejectCandidateRequest]) (*connect.Response[candsv1.RejectCandidateResponse], error) {
	c, err := h.deps.Service.Reject(ctx, req.Msg.GetCandidateId(), req.Msg.GetNote())
	if err != nil {
		return nil, translate(err)
	}
	return connect.NewResponse(&candsv1.RejectCandidateResponse{Candidate: candidateToProto(c)}), nil
}

func (h *connectHandler) RestoreCandidate(ctx context.Context, req *connect.Request[candsv1.RestoreCandidateRequest]) (*connect.Response[candsv1.RestoreCandidateResponse], error) {
	c, err := h.deps.Service.Restore(ctx, req.Msg.GetCandidateId())
	if err != nil {
		return nil, translate(err)
	}
	return connect.NewResponse(&candsv1.RestoreCandidateResponse{Candidate: candidateToProto(c)}), nil
}

func translate(err error) error {
	var nf candidates.ErrNotFound
	if errors.As(err, &nf) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
