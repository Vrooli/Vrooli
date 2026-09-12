package sketch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/protobuf/encoding/protojson"
	"os"
	internal "react-component-library/internal/sketch"
	"strings"

	"connectrpc.com/connect"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
	"react-component-library/internal/designcapture"
	"react-component-library/internal/designcritique"
)

// CheckCandidateAcceptance resolves current facts itself. In particular, clients
// cannot submit an assessment, a calibrated flag, or a behavioral pass boolean.
// This preflight deliberately does not publish or imply an acceptance receipt.
func (h *connectHandler) CheckCandidateAcceptance(ctx context.Context, req *connect.Request[sketchv1.CheckCandidateAcceptanceRequest]) (*connect.Response[sketchv1.CandidateAcceptanceCheck], error) {
	render := req.Msg.GetRender()
	if render == nil || strings.TrimSpace(req.Msg.GetExpectedRenderHash()) == "" || len(req.Msg.GetCritiqueIds()) > 16 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("render, expected render hash, and at most sixteen critique IDs are required"))
	}
	h, c, err := h.candidate(ctx, render.GetCandidate())
	if err != nil {
		return nil, err
	}
	output := &sketchv1.CandidateAcceptanceCheck{Candidate: render.Candidate, PolicyVersion: internal.AcceptancePolicy}
	row := func(code, status, detail string, evidence ...string) {
		output.Requirements = append(output.Requirements, &sketchv1.AcceptanceRequirement{Code: code, Status: status, Detail: detail, EvidenceIds: evidence})
	}
	current, err := h.deps.Store.Read(render.Candidate.Scenario, c.Page)
	if err != nil {
		return nil, h.connectError("CheckCandidateAcceptance", err)
	}
	if current.ContentHash != c.BaseHash {
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
		if output.RenderHash != req.Msg.ExpectedRenderHash {
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
	if len(req.Msg.CritiqueIds) == 0 {
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
			target := designcritique.Target{Scenario: render.Candidate.Scenario, DesignID: c.DesignID, Revision: c.Hash, RenderHash: output.RenderHash}
			seen := map[string]bool{}
			for _, id := range req.Msg.CritiqueIds {
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
	// These are explicit unimplemented proof sources, not waived gates. Keeping
	// them visible prevents the existing visual assessment from becoming a false
	// acceptance path while the remaining producer integrations are completed.
	row("behavior_evidence", "unavailable", "Acceptance still requires current producer-owned deterministic behavior and journey evidence.")
	row("rubric_calibration", "unavailable", "The visual rubric has no versioned human-reviewed calibration corpus yet.")
	row("independent_review", "unavailable", "Final demonstration acceptance requires an attributable independent review.")
	return connect.NewResponse(output), nil
}

// AcceptCandidate records a terminal decision, including unmet requirements.
// It does not select or apply a design; those operations require an accepted
// receipt and their own current-input checks.
func (h *connectHandler) AcceptCandidate(ctx context.Context, req *connect.Request[sketchv1.AcceptCandidateRequest]) (*connect.Response[sketchv1.AcceptanceDecision], error) {
	check := req.Msg.GetCheck()
	if check == nil || check.GetRender() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("acceptance check inputs are required"))
	}
	h, _, err := h.candidate(ctx, check.Render.GetCandidate())
	if err != nil {
		return nil, err
	}
	store, ok := h.deps.Store.(interface {
		RecordAcceptance(context.Context, string, string, internal.AcceptanceIntent, func(context.Context) (internal.AcceptanceFacts, error)) (internal.AcceptanceDecision, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("acceptance persistence unavailable"))
	}
	r := check.Render
	intent := internal.AcceptanceIntent{DesignID: r.Candidate.DesignId, CandidateHash: r.Candidate.Hash, ExpectedRenderHash: check.ExpectedRenderHash, Actor: req.Msg.Actor, Kit: r.Kit, Theme: r.Theme, Direction: r.Direction, PreviewState: r.PreviewState, MissingLabel: r.MissingLabel, FailedLabel: r.FailedLabel, CritiqueIDs: append([]string{}, check.CritiqueIds...)}
	evaluated := false
	decision, err := store.RecordAcceptance(ctx, r.Candidate.Scenario, req.Msg.IdempotencyKey, intent, func(ctx context.Context) (internal.AcceptanceFacts, error) {
		evaluated = true
		result, e := h.CheckCandidateAcceptance(ctx, connect.NewRequest(check))
		if e != nil {
			return internal.AcceptanceFacts{}, e
		}
		facts := internal.AcceptanceFacts{PolicyVersion: result.Msg.PolicyVersion, CandidateHash: r.Candidate.Hash, RenderHash: result.Msg.RenderHash, InputHashes: result.Msg.InputHashes}
		for _, row := range result.Msg.Requirements {
			facts.Requirements = append(facts.Requirements, internal.AcceptanceRequirement{Code: row.Code, Status: row.Status, Detail: row.Detail, EvidenceIDs: row.EvidenceIds})
		}
		return facts, nil
	})
	if errors.Is(err, internal.ErrAcceptanceConflict) {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	if err != nil {
		return nil, h.connectError("AcceptCandidate", err)
	}
	if evaluated && h.deps.RecordWrite != nil {
		h.deps.RecordWrite(ctx)
	}
	return acceptanceResponse(decision)
}
func (h *connectHandler) GetAcceptance(ctx context.Context, req *connect.Request[sketchv1.GetAcceptanceRequest]) (*connect.Response[sketchv1.AcceptanceDecision], error) {
	h, err := h.forRequest(ctx)
	if err != nil {
		return nil, err
	}
	store, ok := h.deps.Store.(interface {
		ReadAcceptance(string, string, string) (internal.AcceptanceDecision, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("acceptance persistence unavailable"))
	}
	d, err := store.ReadAcceptance(req.Msg.Scenario, req.Msg.DesignId, req.Msg.Id)
	if errors.Is(err, os.ErrNotExist) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, h.connectError("GetAcceptance", err)
	}
	return acceptanceResponse(d)
}
func acceptanceResponse(d internal.AcceptanceDecision) (*connect.Response[sketchv1.AcceptanceDecision], error) {
	raw, err := json.Marshal(d)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var response sketchv1.AcceptanceDecision
	if err = protojson.Unmarshal(raw, &response); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&response), nil
}
