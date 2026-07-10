package authoring

import (
	"context"
	"fmt"
	"strings"

	planmodel "plan-manager/internal/planmodel"
)

func (s *service) PreviewPlan(ctx context.Context, sessionID string) (string, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return "", GuidedStep{}, err
	}
	if s.renderer == nil {
		return "", GuidedStep{}, ErrInvalidSession{Reason: "render preview unavailable: no renderer configured"}
	}
	draft, err := sessionToPlan(sess)
	if err != nil {
		return "", GuidedStep{}, err
	}
	// Apply the same posture derivation finalize/Create uses so the preview render
	// agrees with the persisted render (greenfield default OR brownfield from
	// scenario maturity) instead of always showing the default greenfield block.
	if s.posture != nil {
		draft = s.posture.PreparePosture(ctx, draft)
	}
	if renderer, ok := s.renderer.(DraftPlanRenderer); ok {
		return renderer.RenderDraft(draft, sess.ID), stepForReview(sess), nil
	}
	return s.renderer.Render(draft), stepForReview(sess), nil
}

// FinalizeOptions carries finalize-time inputs that are not session state.
type FinalizeOptions struct {
	// WorkspaceRoot stamps the produced plan's canonical workspace (usually
	// the caller's repo root) so workspace-scoped reads (`plans get`/`list`
	// run from that root) resolve the finalized plan. Empty leaves the plan
	// unscoped (legacy behavior).
	WorkspaceRoot string
}

// FinalizeResult is finalize's honest persistence report: not just the plan,
// but WHERE it was written (store path, workspace) and what the mirror publish
// REALLY did (computed at write time, never the read-model default).
type FinalizeResult struct {
	Plan planmodel.Plan
	// Mirror is the computed mirror publish result threaded out of the Create
	// call for a fresh finalize, or the stored state for an idempotent re-run.
	// Never default/unknown for a plan this call just created.
	Mirror planmodel.RenderedPlanMirror
	// AlreadyFinalized marks the idempotent short-circuit: the session was
	// finalized earlier and no new plan row was written by this call.
	AlreadyFinalized bool
	// FinalizedAt is the session timestamp of the finalize that persisted the
	// plan (the current one, or the earlier one when AlreadyFinalized).
	FinalizedAt string
	// StorePath is the resolved physical SQLite path of the plans store, or
	// empty when the process was wired without one (tests).
	StorePath string
}

func (s *service) Finalize(ctx context.Context, sessionID string, opts FinalizeOptions) (FinalizeResult, GuidedStep, error) {
	var result FinalizeResult
	sess, err := s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		if sess.Finalized && strings.TrimSpace(sess.PlanID) != "" {
			existing, err := s.readFinalizedPlan(ctx, planmodel.Plan{ID: sess.PlanID})
			if err != nil {
				return false, err
			}
			result = FinalizeResult{
				Plan:             existing,
				Mirror:           existing.Mirror,
				AlreadyFinalized: true,
				FinalizedAt:      sess.UpdatedAt,
				StorePath:        s.storePath,
			}
			return false, nil
		}
		if _, err := sessionToPlan(*sess); err != nil {
			return false, err
		}
		if violations := s.readinessViolations(ctx, *sess); len(violations) > 0 {
			return false, ErrStructureGate{Violations: violations}
		}
		if violations := s.commandViolationsForSections(ctx, sess.Sections); len(violations) > 0 {
			return false, ErrStructureGate{Violations: violations}
		}
		draft, err := sessionToPlan(*sess)
		if err != nil {
			return false, err
		}
		if root := strings.TrimSpace(opts.WorkspaceRoot); root != "" {
			draft.WorkspaceRoot = root
		}
		created, err := s.writer.CreatePlan(ctx, draft)
		if err != nil {
			return false, err
		}
		verified, err := s.readFinalizedPlan(ctx, created)
		if err != nil {
			return false, err
		}
		result = FinalizeResult{
			Plan:      verified,
			Mirror:    computedMirror(created),
			StorePath: s.storePath,
		}
		sess.Finalized = true
		sess.PlanID = verified.ID
		return true, nil
	})
	if err != nil {
		return FinalizeResult{}, GuidedStep{}, err
	}
	if result.FinalizedAt == "" {
		result.FinalizedAt = sess.UpdatedAt
	}
	return result, stepForFinalizedPlan(sess, result.Plan.ID, result.Plan.Slug), nil
}

// computedMirror normalizes the mirror publish result threaded out of Create.
// A plan this process just created must never report the default/unknown
// mirror state: an unreported status means no mirror file was written (e.g.
// the plans service has no mirror store), which is surfaced honestly as
// write_failed with the reason — loud, never silent.
func computedMirror(created planmodel.Plan) planmodel.RenderedPlanMirror {
	m := created.Mirror
	switch m.Status {
	case planmodel.RenderedMirrorStatusFresh,
		planmodel.RenderedMirrorStatusWriteFailed,
		planmodel.RenderedMirrorStatusMissing,
		planmodel.RenderedMirrorStatusStale:
		return m
	}
	m.Status = planmodel.RenderedMirrorStatusWriteFailed
	if strings.TrimSpace(m.LastError) == "" {
		m.LastError = "mirror publish result not reported by the plans store — no mirror file was written; run `plan-manager plans reconcile --repair-mirrors`"
	}
	return m
}

func (s *service) readFinalizedPlan(ctx context.Context, fallback planmodel.Plan) (planmodel.Plan, error) {
	idOrSlug := fallback.ID
	if s.reader == nil {
		return fallback, nil
	}
	plan, err := s.reader.GetPlan(ctx, idOrSlug, fallback.WorkspaceRoot)
	if err != nil {
		return planmodel.Plan{}, ErrFinalizeReadback{PlanID: idOrSlug, Cause: err}
	}
	if strings.TrimSpace(plan.ID) == "" {
		return planmodel.Plan{}, ErrFinalizeReadback{PlanID: idOrSlug, Cause: fmt.Errorf("resolved plan has empty id")}
	}
	if _, err := s.reader.RenderPlan(ctx, plan.ID, fallback.WorkspaceRoot); err != nil {
		return planmodel.Plan{}, ErrFinalizeReadback{PlanID: plan.ID, Cause: err}
	}
	return plan, nil
}
