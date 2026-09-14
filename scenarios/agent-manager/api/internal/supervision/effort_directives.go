package supervision

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *EffortService) RequestDirective(ctx context.Context, req *pb.RequestEffortDirectiveRequest, actor EffortActor) (*pb.EffortDirective, error) {
	if req == nil || req.Directive == nil || actor.ID == "" {
		return nil, errors.New("authenticated directive request required")
	}
	input := req.Directive
	if err := validateRecoveryExpectation(input.RecoveryExpectation); err != nil {
		return nil, err
	}
	if input.IdempotencyKey == "" || input.EffortRef == "" || input.TargetRevision == "" || input.TargetRunId == "" || input.Scope == "" || input.Adjustment == "" || input.ExpectedResult == "" || len(input.EvidenceRefs) == 0 || len(input.Adjustment) > 3500 || len(input.EvidenceRefs) > 100 {
		return nil, errors.New("directive needs bounded adjustment, scope, evidence, expected result, exact target and idempotency key")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := "directive:" + input.IdempotencyKey
	digest := effortDigest(req) + actor.ID
	replay := &pb.EffortDirective{}
	if ok, err := s.repo.ReplayEffortOperation(ctx, key, digest, replay); ok || err != nil {
		if err != nil {
			return nil, err
		}
		return s.repo.GetEffortDirective(ctx, replay.DirectiveId)
	}
	if input.ExpiresAt == nil || !input.ExpiresAt.IsValid() || !input.ExpiresAt.AsTime().After(s.now()) {
		return nil, errors.New("directive expiry must be in the future")
	}
	e, o, err := s.repo.GetEffort(ctx, input.EffortRef)
	if err != nil {
		return nil, err
	}
	if e.Revision != req.ExpectedEnrollmentRevision {
		return nil, ErrConflict
	}
	// Construct the immutable request explicitly; ignore caller-written lifecycle fields.
	d := &pb.EffortDirective{DirectiveId: uuid.NewString(), EffortRef: input.EffortRef, TargetRevision: input.TargetRevision, Issuer: actor.ID, TargetRunId: input.TargetRunId, Kind: input.Kind, Scope: input.Scope, EvidenceRefs: input.EvidenceRefs, Adjustment: input.Adjustment, ExpectedResult: input.ExpectedResult, ExpiresAt: input.ExpiresAt, IdempotencyKey: input.IdempotencyKey, Delivery: pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_PENDING, Revision: 1, CreatedAt: timestamppb.New(s.now().UTC()), Hypothesis: input.Hypothesis, Comparison: input.Comparison, Assessment: "unknown"}
	d.AuthorityBinding = bindDirectiveAuthority(actor, e)
	if input.RecoveryExpectation != nil {
		d.RecoveryExpectation = proto.Clone(input.RecoveryExpectation).(*pb.EffortRecoveryExpectation)
		d.RecoveryVerification = &pb.EffortRecoveryVerification{State: "pending"}
	}
	if !actor.supervises(e) {
		return nil, errors.New("requester is not the enrolled supervisor")
	}
	if reason := s.directiveAuthority(e, o, d); reason != "" {
		d.Delivery = pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED
		d.DeliveryReason = reason
	}
	discovery, err := s.repo.GetEffortDiscovery(ctx)
	if err != nil {
		return nil, err
	}
	d.SourceSnapshot = s.projectEffort(ctx, e, o, discovery)
	d.SupervisionUsage = &pb.EffortUsage{Partial: true, Source: "supervision request", Limitations: []string{"observation, judgment and shared idle costs are unmeasured"}}
	if err = s.repo.SaveEffortDirective(ctx, d, 0, key, digest); err != nil {
		return nil, err
	}
	if d.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_PENDING {
		return s.deliverDirective(ctx, d)
	}
	return d, nil
}
func (s *EffortService) directiveAuthority(e *pb.EffortEnrollment, o *pb.EffortBoardRow, d *pb.EffortDirective) string {
	if e.Withdrawn {
		return "effort withdrawn"
	}
	if e.TargetRevision != d.TargetRevision {
		return "target revision changed"
	}
	if e.Workspace != "" && (o.Freshness == pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE) {
		return "workspace evidence unavailable"
	}
	if e.Workspace != "" && (o.GetObservedAt() == nil || s.now().Sub(o.ObservedAt.AsTime()) > s.config.StaleAfter) {
		return "workspace evidence stale; obtain current owner observation before steering"
	}
	if e.AuthorizedBy == "" || e.AuthorityRef == "" || e.AuthorityExpiresAt == nil || !e.AuthorityExpiresAt.IsValid() || !e.AuthorityExpiresAt.AsTime().After(s.now()) {
		return "actual owner grant unavailable or expired"
	}
	allowed := e.AutonomousSupervision
	for _, kind := range e.PermittedActions {
		if kind == d.Kind {
			allowed = true
		}
	}
	if !allowed {
		return "action outside authorized scope"
	}
	for _, sub := range e.Subjects {
		if sub.Owner == "agent-manager" && sub.Kind == "run" && sub.RunId == d.TargetRunId && (e.AutonomousSupervision || sub.Role == "orchestrator") {
			return ""
		}
	}
	return "only the enrolled assigning orchestrator may receive steering"
}
func (s *EffortService) transitionDirective(ctx context.Context, d *pb.EffortDirective, state pb.EffortDirectiveDelivery, reason string) (*pb.EffortDirective, error) {
	next := proto.Clone(d).(*pb.EffortDirective)
	next.Revision++
	next.Delivery = state
	next.DeliveryReason = reason
	if state == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED {
		next.DeliveredAt = timestamppb.New(s.now().UTC())
	}
	err := s.repo.SaveEffortDirective(ctx, next, d.Revision, fmt.Sprintf("delivery:%s:%d", d.DirectiveId, next.Revision), effortDigest(next))
	return next, err
}
func (s *EffortService) deliverDirective(ctx context.Context, d *pb.EffortDirective) (*pb.EffortDirective, error) {
	current, readErr := s.repo.GetEffortDirective(ctx, d.DirectiveId)
	if readErr != nil {
		return d, readErr
	}
	d = current
	if d.Kind == pb.WatchActionKind_WATCH_ACTION_KIND_RECOVER_FRESH && d.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED {
		return d, s.bindRecoveredOrchestrator(ctx, d)
	}
	if d.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN {
		return s.reconcileDirectiveReceipt(ctx, d)
	}
	if d.SupersededBy != "" {
		return d, nil
	} // uncertain prior effect is retained; no new send
	if d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_PENDING && d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN {
		return d, nil
	}
	e, o, err := s.repo.GetEffort(ctx, d.EffortRef)
	if err != nil {
		return d, err
	}
	if d.ExpiresAt == nil || !s.now().Before(d.ExpiresAt.AsTime()) {
		return s.transitionDirective(ctx, d, pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_EXPIRED, "directive expired; any uncertain prior effect remains in transition history")
	}
	if reason := s.directiveAuthority(e, o, d); reason != "" {
		return s.transitionDirective(ctx, d, pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED, reason)
	}
	if reason := s.originalDirectiveAuthority(e, d); reason != "" {
		return s.transitionDirective(ctx, d, pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED, reason)
	}
	if s.policies == nil {
		return d, errors.New("supervision policy unavailable")
	}
	disabled, _, err := s.policies.Disabled(ctx)
	if err != nil {
		return d, err
	}
	if disabled {
		return s.transitionDirective(ctx, d, pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED, "supervision disabled")
	}
	if s.controller == nil {
		return d, errors.New("run controller unavailable")
	}
	id, err := uuid.Parse(d.TargetRunId)
	if err != nil {
		return d, err
	}
	run, err := s.controller.GetRun(ctx, id)
	if err != nil {
		return d, err
	}
	if run == nil {
		return d, errors.New("target run unavailable")
	}
	// A prior dispatch may already have resumed the run. Never declare delivery from
	// its activity; reconcile using the same owner idempotency identity at rest.
	if run.Status.IsActive() || run.Status == domain.RunStatusParked {
		return d, nil
	}
	if d.Kind == pb.WatchActionKind_WATCH_ACTION_KIND_RECOVER_FRESH {
		return s.deliverFreshRecovery(ctx, d, run)
	}
	if run.Status == domain.RunStatusFailed && d.Kind == pb.WatchActionKind_WATCH_ACTION_KIND_CONTINUE {
		if d.RecoveryExpectation == nil {
			return s.transitionDirective(ctx, d, pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED, "failed-run recovery requires an immutable progress condition and baseline before dispatch")
		}
		if strings.TrimSpace(d.Hypothesis) == "" || strings.TrimSpace(d.Comparison) == "" {
			return s.transitionDirective(ctx, d, pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED, "failed-run recovery requires a resolution hypothesis and comparison")
		}
		if allowed, reason := domain.CanContinueRun(run); !allowed {
			return s.transitionDirective(ctx, d, pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED, reason+"; use qualified owner fresh-run recovery when authorized")
		}
	} else if run.Status != domain.RunStatusNeedsReview {
		return s.transitionDirective(ctx, d, pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED, "target is not at a resumable cooperative boundary")
	}
	if d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN {
		d, err = s.transitionDirective(ctx, d, pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN, "dispatch reserved; reconcile with same owner identity on retry")
		if err != nil {
			return nil, err
		}
	}
	if err = s.controller.ContinueRun(ctx, id, effortDirectiveMessage(d), "effort-directive:"+d.DirectiveId); err != nil {
		if domain.IsPreEffectRefusal(err) {
			d.RefusalBeforeEffects = true
			return s.transitionDirective(ctx, d, pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED, fmt.Sprintf("owner refused continuation before effects (%s); inspect the owner preflight, and use a qualified fresh-session recovery only under its separate authority", domain.GetErrorCode(err)))
		}
		return d, err
	}
	return s.transitionDirective(ctx, d, pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED, "owner continuation accepted; acknowledgment and outcome are separate")
}

// One immutable payload is shared by dispatch and exact owner receipt verification.
func effortDirectiveMessage(d *pb.EffortDirective) string {
	message := fmt.Sprintf("[effort directive=%s effort=%s target_revision=%s supervisor=%s]\nScope: %s\n%s\nExpected: %s\nEvidence: %s\nAcknowledge through UpdateEffortDirective; accepted, deferred with owner wait, or challenged with evidence.", d.DirectiveId, d.EffortRef, d.TargetRevision, d.Issuer, d.Scope, d.Adjustment, d.ExpectedResult, strings.Join(d.EvidenceRefs, ", "))
	if expectation := d.RecoveryExpectation; expectation != nil {
		message += fmt.Sprintf("\nRecovery progress condition: %s\nBaseline: %s\nRetain new owner evidence against this condition. A resumed process alone does not verify recovery.", expectation.ProgressCondition, strings.Join(expectation.BaselineEvidenceRefs, ", "))
	}
	return message
}
func (s *EffortService) ListDirectives(ctx context.Context, req *pb.ListEffortDirectivesRequest) (*pb.ListEffortDirectivesResponse, error) {
	if req == nil {
		req = &pb.ListEffortDirectivesRequest{}
	}
	n := effortPageSize(req.PageSize)
	rows, err := s.repo.ListEffortDirectives(ctx, req.EffortRef, req.PageToken, n+1)
	if err != nil {
		return nil, err
	}
	r := &pb.ListEffortDirectivesResponse{Directives: rows}
	if len(rows) > n {
		r.Directives = rows[:n]
		r.NextPageToken = rows[n-1].DirectiveId
	}
	return r, nil
}
func (s *EffortService) UpdateDirective(ctx context.Context, req *pb.UpdateEffortDirectiveRequest, actor EffortActor) (*pb.EffortDirective, error) {
	if req == nil || req.IdempotencyKey == "" || actor.ID == "" {
		return nil, errors.New("authenticated update and idempotency key required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := "directive-update:" + req.IdempotencyKey
	digest := effortDigest(req) + actor.ID
	replay := &pb.EffortDirective{}
	if ok, err := s.repo.ReplayEffortOperation(ctx, key, digest, replay); ok || err != nil {
		return replay, err
	}
	d, err := s.repo.GetEffortDirective(ctx, req.DirectiveId)
	if err != nil {
		return nil, err
	}
	e, _, err := s.repo.GetEffort(ctx, d.EffortRef)
	if err != nil {
		return nil, err
	}
	if d.Revision != req.ExpectedRevision {
		return nil, ErrConflict
	}
	if req.ReconcileDelivery {
		if !actor.supervises(e) {
			return nil, errors.New("only supervisor or operator may reconcile historical delivery")
		}
		if req.Acknowledgment != 0 || req.ActionRef != "" || req.Assessment != "" || req.SupersededBy != "" || req.SupervisionUsage != nil || req.RecoveryVerification != nil {
			return nil, errors.New("receipt-only reconciliation cannot be combined with another directive update")
		}
		if d.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED {
			return nil, errors.New("delivery already reconciled")
		}
		if d.RefusalBeforeEffects {
			return nil, errors.New("owner proved refusal before effects; there is no uncertain delivery to reopen")
		}
		hadUncertainty, err := s.repo.HadUncertainEffortDelivery(ctx, d.DirectiveId)
		if err != nil {
			return nil, err
		}
		if !hadUncertainty {
			return nil, errors.New("no retained uncertain dispatch to reconcile; no effect is authorized")
		}
		next, err := s.inspectDirectiveReceipt(ctx, d)
		if err != nil {
			return nil, err
		}
		next.Revision = d.Revision + 1
		if err = s.repo.SaveEffortDirective(ctx, next, d.Revision, key, digest); err != nil {
			return nil, err
		}
		return next, nil
	}
	next := proto.Clone(d).(*pb.EffortDirective)
	if req.RecoveryVerification != nil {
		if err := s.validateRecoveryVerification(ctx, e, d, req.RecoveryVerification, actor); err != nil {
			return nil, err
		}
		next.RecoveryVerification = proto.Clone(req.RecoveryVerification).(*pb.EffortRecoveryVerification)
		next.RecoveryVerification.Verifier = actor.ID
	}
	if req.Acknowledgment != pb.EffortDirectiveAcknowledgment_EFFORT_DIRECTIVE_ACKNOWLEDGMENT_UNSPECIFIED || req.ActionRef != "" {
		if actor.ID != recoveryTargetRunID(d) {
			return nil, errors.New("only the target run may acknowledge or report action")
		}
		if d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED {
			return nil, errors.New("directive has not been delivered")
		}
		switch req.Acknowledgment {
		case pb.EffortDirectiveAcknowledgment_EFFORT_DIRECTIVE_ACKNOWLEDGMENT_UNSPECIFIED, pb.EffortDirectiveAcknowledgment_EFFORT_DIRECTIVE_ACKNOWLEDGMENT_ACCEPTED:
		case pb.EffortDirectiveAcknowledgment_EFFORT_DIRECTIVE_ACKNOWLEDGMENT_DEFERRED:
			if req.OwnerWaitRef == "" || req.Reason == "" {
				return nil, errors.New("defer requires a named owner wait and reason")
			}
		case pb.EffortDirectiveAcknowledgment_EFFORT_DIRECTIVE_ACKNOWLEDGMENT_CHALLENGED:
			if len(req.EvidenceRefs) == 0 || req.Reason == "" {
				return nil, errors.New("challenge requires reason and evidence")
			}
		default:
			return nil, errors.New("unsupported acknowledgment")
		}
		if req.Acknowledgment != 0 {
			next.Acknowledgment = req.Acknowledgment
			next.AcknowledgmentReason = req.Reason
			next.OwnerWaitRef = req.OwnerWaitRef
		}
		if req.ActionRef != "" {
			next.ActionRef = req.ActionRef
		}
	}
	if req.Assessment != "" || req.SupersededBy != "" || req.SupervisionUsage != nil {
		if !actor.supervises(e) {
			return nil, errors.New("only supervisor or operator may assess or supersede")
		}
		if req.Assessment != "" {
			if req.Assessment != "unknown" && req.Assessment != "supported" && req.Assessment != "contradicted" {
				return nil, errors.New("assessment must be unknown, supported or contradicted")
			}
			if req.Assessment != "unknown" && (len(req.EvidenceRefs) == 0 || d.Hypothesis == "" || d.Comparison == "") {
				return nil, errors.New("assessed benefit requires hypothesis, comparison and evidence")
			}
			next.Assessment = req.Assessment
			next.AssessmentEvidenceRefs = req.EvidenceRefs
		}
		if req.SupersededBy != "" {
			other, getErr := s.repo.GetEffortDirective(ctx, req.SupersededBy)
			if getErr != nil {
				return nil, getErr
			}
			if other.DirectiveId == d.DirectiveId || other.EffortRef != d.EffortRef {
				return nil, errors.New("supersession must name another directive for the same effort")
			}
			next.SupersededBy = other.DirectiveId
			if next.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_PENDING || next.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN {
				if next.Delivery == pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN {
					next.DeliveryReason = "superseded; prior dispatch outcome remains uncertain and requires read-only owner reconciliation"
				} else {
					next.Delivery = pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_SUPERSEDED
				}
			}
		}
		if req.SupervisionUsage != nil {
			if err := validateEffortUsage(req.SupervisionUsage); err != nil {
				return nil, err
			}
			next.SupervisionUsage = req.SupervisionUsage
		}
	}
	if proto.Equal(d, next) {
		return nil, errors.New("update contains no change")
	}
	next.Revision++
	if err = s.repo.SaveEffortDirective(ctx, next, d.Revision, key, digest); err != nil {
		return nil, err
	}
	return next, nil
}
func validateEffortUsage(u *pb.EffortUsage) error {
	if u.GetTokens() < 0 || u.GetReportedCostUsd() < 0 || u.GetAgentSeconds() < 0 {
		return errors.New("usage measurements cannot be negative")
	}
	if (u.Tokens != nil || u.ReportedCostUsd != nil || u.AgentSeconds != nil) && u.Source == "" {
		return errors.New("usage measurements require a source")
	}
	return nil
}
