package sketch

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review"
	"react-component-library/internal/designcapture"
	"react-component-library/internal/designcritique"
	internal "react-component-library/internal/sketch"
)

// ValidateAcceptanceCandidate resolves an immutable candidate before the Review
// service prepares an idempotent acceptance receipt.
func (h *connectHandler) ValidateAcceptanceCandidate(ctx context.Context, req *reviewv1.CheckCandidateAcceptanceRequest) error {
	if req == nil || req.GetRender() == nil || req.GetRender().GetCandidate() == nil {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("candidate render is required"))
	}
	_, _, err := h.candidate(ctx, req.GetRender().GetCandidate())
	return err
}

// EvaluateAcceptance resolves current facts from the candidate page, renderer,
// critique records and producer-owned capture evidence. The Review handler calls
// this only inside RecordAcceptance's transactional evaluation callback.
func (h *connectHandler) EvaluateAcceptance(ctx context.Context, req *reviewv1.CheckCandidateAcceptanceRequest) (*reviewv1.CandidateAcceptanceCheck, error) {
	render := req.GetRender()
	if render == nil || strings.TrimSpace(req.GetExpectedRenderHash()) == "" || len(req.GetCritiqueIds()) > 16 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("render, expected render hash, and at most sixteen critique IDs are required"))
	}
	h, candidate, err := h.candidate(ctx, render.GetCandidate())
	if err != nil {
		return nil, err
	}
	output := &reviewv1.CandidateAcceptanceCheck{Candidate: render.GetCandidate(), PolicyVersion: internal.AcceptancePolicy}
	row := func(code, status, detail string, evidence ...string) {
		output.Requirements = append(output.Requirements, &reviewv1.AcceptanceRequirement{Code: code, Status: status, Detail: detail, EvidenceIds: evidence})
	}
	current, err := h.deps.Store.Read(render.GetCandidate().GetScenario(), candidate.Page)
	if err != nil {
		return nil, h.connectError("CheckCandidateAcceptance", err)
	}
	if current.ContentHash != candidate.BaseHash {
		row("page_freshness", "blocked", "The authored page changed after this candidate was created. Rebase and review the candidate.")
	} else {
		row("page_freshness", "passed", "The candidate still derives from the current authored page.", current.ContentHash)
	}
	rendered, err := h.RenderCandidate(ctx, connect.NewRequest(render))
	if err != nil {
		row("render_completeness", "blocked", err.Error())
		row("render_freshness", "unavailable", "A current render could not be resolved.")
	} else {
		output.RenderHash = rendered.Msg.GetRenderHash()
		output.InputHashes = rendered.Msg.GetTarget().GetInputHashes()
		if output.RenderHash != req.GetExpectedRenderHash() {
			row("render_freshness", "blocked", "Rendering inputs changed. Review a new capture of the current render.", output.RenderHash)
		} else {
			row("render_freshness", "passed", "Current rendering inputs match the reviewed render.", output.RenderHash)
		}
		missing := []string{}
		for _, gap := range rendered.Msg.GetBundle().GetGaps() {
			if gap.GetRequired() {
				missing = append(missing, gap.GetRegion()+": "+gap.GetCode())
			}
		}
		if rendered.Msg.GetBundle() == nil {
			row("render_completeness", "unavailable", "The renderer did not supply a region-resolution report.")
		} else if len(missing) > 0 {
			row("render_completeness", "blocked", strings.Join(missing, "; "))
		} else {
			row("render_completeness", "passed", "No required region is unresolved in the current generated render.", output.RenderHash)
		}
	}
	if len(req.GetCritiqueIds()) == 0 {
		row("visual_review", "blocked", "Record an evidence-linked visual review of this exact candidate and render.")
	} else {
		resolver, ok := h.deps.CaptureDispatcher.(designcapture.ScreenshotResolver)
		if !ok || h.deps.CritiquesFor == nil || h.deps.CapturesFor == nil {
			row("visual_review", "unavailable", "Visual review and capture evidence services are unavailable.")
		} else {
			repo, e := h.deps.CritiquesFor(ctx)
			if e != nil {
				return nil, connect.NewError(connect.CodeUnavailable, e)
			}
			captures, e := h.deps.CapturesFor(ctx)
			if e != nil {
				return nil, connect.NewError(connect.CodeUnavailable, e)
			}
			target := designcritique.Target{Scenario: render.GetCandidate().GetScenario(), DesignID: candidate.DesignID, Revision: candidate.Hash, RenderHash: output.RenderHash}
			seen := map[string]bool{}
			for _, id := range req.GetCritiqueIds() {
				if id == "" || seen[id] {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("critique IDs must be nonempty and unique"))
				}
				seen[id] = true
				review, e := repo.Get(ctx, id)
				if e != nil {
					row("visual_review", "blocked", fmt.Sprintf("Review %s is unavailable.", id), id)
					continue
				}
				if review.Review.Target != target {
					row("visual_review", "blocked", "The visual review targets a different candidate or render.", id)
					continue
				}
				assessment, e := designcritique.Evaluate(ctx, review.Review, designcritique.CaptureEvidenceVerifier{Captures: captures, Screenshots: resolver})
				if e != nil {
					row("visual_review", "blocked", e.Error(), id)
					continue
				}
				if !assessment.VisualFloorMet {
					row("visual_review", "blocked", "At least one dimension is below 3 or a critical/major finding remains.", id)
				} else {
					row("visual_review", "passed", "Every dimension meets the visual floor and no critical/major finding remains; image evidence was revalidated.", id, review.Hash)
				}
			}
		}
	}
	// These proof sources remain explicitly unavailable, never waived.
	row("behavior_evidence", "unavailable", "Acceptance still requires current producer-owned deterministic behavior and journey evidence.")
	row("rubric_calibration", "unavailable", "The visual rubric has no versioned human-reviewed calibration corpus yet.")
	row("independent_review", "unavailable", "Final demonstration acceptance requires an attributable independent review.")
	return output, nil
}
