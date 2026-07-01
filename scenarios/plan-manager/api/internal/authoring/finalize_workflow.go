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
	return s.renderer.Render(draft), stepForReview(sess), nil
}

func (s *service) Finalize(ctx context.Context, sessionID string) (planmodel.Plan, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return planmodel.Plan{}, GuidedStep{}, err
	}
	unlock := s.lockSession(sess.ID)
	defer unlock()
	sess, err = s.load(ctx, sess.ID)
	if err != nil {
		return planmodel.Plan{}, GuidedStep{}, err
	}
	if sess.Finalized && strings.TrimSpace(sess.PlanID) != "" {
		plan, err := s.readFinalizedPlan(ctx, planmodel.Plan{ID: sess.PlanID})
		if err != nil {
			return planmodel.Plan{}, GuidedStep{}, err
		}
		return plan, stepForFinalizedPlan(sess, plan.ID, plan.Slug), nil
	}
	if violations := sessionViolations(sess); len(violations) > 0 {
		return planmodel.Plan{}, GuidedStep{}, ErrStructureGate{Violations: violations}
	}
	if violations := s.commandViolationsForSections(ctx, sess.Sections); len(violations) > 0 {
		return planmodel.Plan{}, GuidedStep{}, ErrStructureGate{Violations: violations}
	}
	draft, err := sessionToPlan(sess)
	if err != nil {
		return planmodel.Plan{}, GuidedStep{}, err
	}
	plan, err := s.writer.CreatePlan(ctx, draft)
	if err != nil {
		return planmodel.Plan{}, GuidedStep{}, err
	}
	verified, err := s.readFinalizedPlan(ctx, plan)
	if err != nil {
		return planmodel.Plan{}, GuidedStep{}, err
	}
	sess.Finalized = true
	sess.PlanID = verified.ID
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return planmodel.Plan{}, GuidedStep{}, err
	}
	return verified, stepForFinalizedPlan(sess, verified.ID, verified.Slug), nil
}

func (s *service) readFinalizedPlan(ctx context.Context, fallback planmodel.Plan) (planmodel.Plan, error) {
	idOrSlug := fallback.ID
	if s.reader == nil {
		return fallback, nil
	}
	plan, err := s.reader.GetPlan(ctx, idOrSlug)
	if err != nil {
		return planmodel.Plan{}, ErrFinalizeReadback{PlanID: idOrSlug, Cause: err}
	}
	if strings.TrimSpace(plan.ID) == "" {
		return planmodel.Plan{}, ErrFinalizeReadback{PlanID: idOrSlug, Cause: fmt.Errorf("resolved plan has empty id")}
	}
	if _, err := s.reader.RenderPlan(ctx, plan.ID); err != nil {
		return planmodel.Plan{}, ErrFinalizeReadback{PlanID: plan.ID, Cause: err}
	}
	return plan, nil
}
