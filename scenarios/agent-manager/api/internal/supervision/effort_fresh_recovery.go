package supervision

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type freshRecoveryController interface {
	RecoverMissingSession(context.Context, uuid.UUID, string) (*domain.Run, error)
	FreshRecoveryAccepted(context.Context, uuid.UUID, string) (*domain.Run, error)
}

// Registry observations cannot authorize a replacement. Only the original
// directive and exact owner receipt may finish its already-approved handoff.
// Ambiguous receipts leave the existing authority untouched and visibly pending.
func (s *EffortService) reconcileRegistryRecovery(ctx context.Context, e *pb.EffortEnrollment, run *domain.Run, role string) (bool, error) {
	if role != "orchestrator" || len(run.SourceRunIDs) == 0 {
		return false, nil
	}
	directives, err := s.repo.ListEffortDirectives(ctx, e.EffortRef, "", 1001)
	if err != nil {
		return true, err
	}
	if len(directives) > 1000 {
		return true, errors.New("recovery receipt history exceeds the bounded registry reconciliation; authority unchanged")
	}
	for _, d := range directives {
		if d.Kind != pb.WatchActionKind_WATCH_ACTION_KIND_RECOVER_FRESH || (d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN && d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED) {
			continue
		}
		sourceID, parseErr := uuid.Parse(d.TargetRunId)
		if parseErr != nil || !slices.Contains(run.SourceRunIDs, sourceID) || (d.RecoveredRunId != "" && d.RecoveredRunId != run.ID.String()) {
			continue
		}
		next, reconcileErr := s.deliverDirective(ctx, d) // receipt read/binding only; never admission
		if reconcileErr != nil {
			return true, reconcileErr
		}
		if next.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED || next.RecoveredRunId != run.ID.String() {
			return true, errors.New("recovery child observed without its exact accepted owner receipt; authority unchanged")
		}
		return true, nil
	}
	return false, nil
}

func recoveryTargetRunID(d *pb.EffortDirective) string {
	if d.Kind == pb.WatchActionKind_WATCH_ACTION_KIND_RECOVER_FRESH {
		return d.RecoveredRunId
	}
	return d.TargetRunId
}

func (s *EffortService) deliverFreshRecovery(ctx context.Context, d *pb.EffortDirective, run *domain.Run) (*pb.EffortDirective, error) {
	refuse := func(reason string) (*pb.EffortDirective, error) {
		return s.transitionDirective(ctx, d, pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED, reason)
	}
	if run.Status != domain.RunStatusFailed || run.CancelRequestedAt != nil {
		return refuse("fresh recovery requires a failed, uncancelled owner run")
	}
	if d.RecoveryExpectation == nil || strings.TrimSpace(d.Hypothesis) == "" || strings.TrimSpace(d.Comparison) == "" {
		return refuse("fresh recovery requires an immutable progress expectation, baseline, hypothesis and comparison")
	}
	controller, ok := s.controller.(freshRecoveryController)
	if !ok {
		return refuse("qualified missing-session recovery owner unavailable")
	}
	var err error
	d, err = s.transitionDirective(ctx, d, pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN, "fresh recovery reserved; reconcile the exact owner's original admission, never resend")
	if err != nil {
		return nil, err
	}
	replacement, err := controller.RecoverMissingSession(ctx, run.ID, effortDirectiveMessage(d))
	if err != nil {
		if domain.IsPreEffectRefusal(err) {
			d.RefusalBeforeEffects = true
			return refuse(fmt.Sprintf("owner refused fresh recovery before effects (%s)", domain.GetErrorCode(err)))
		}
		return d, err
	}
	next, err := freshRecoveryResult(s, d, replacement)
	if err != nil {
		return d, err
	}
	next.Revision++
	if err = s.repo.SaveEffortDirective(ctx, next, d.Revision, fmt.Sprintf("delivery:%s:%d", d.DirectiveId, next.Revision), effortDigest(next)); err != nil {
		return d, err
	}
	return next, s.bindRecoveredOrchestrator(ctx, next)
}

func freshRecoveryResult(s *EffortService, d *pb.EffortDirective, replacement *domain.Run) (*pb.EffortDirective, error) {
	sourceID, err := uuid.Parse(d.TargetRunId)
	if err != nil {
		return d, err
	}
	if replacement == nil || replacement.ID == uuid.Nil || replacement.ID == sourceID || !slices.Contains(replacement.SourceRunIDs, sourceID) {
		return d, errors.New("fresh recovery receipt lacks the exact replacement/predecessor lineage; delivery remains uncertain")
	}
	next := proto.Clone(d).(*pb.EffortDirective)
	next.RecoveredRunId = replacement.ID.String()
	next.Delivery = pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED
	next.DeliveredAt = timestamppb.New(s.now().UTC())
	next.DeliveryReason = "owner accepted exact missing-session replacement; current grant binding and useful progress are separately reconciled"
	return next, nil
}

func (s *EffortService) inspectFreshRecoveryReceipt(ctx context.Context, d *pb.EffortDirective) (*pb.EffortDirective, error) {
	controller, ok := s.controller.(freshRecoveryController)
	if !ok {
		return d, errors.New("fresh recovery receipt owner unavailable; no resend")
	}
	id, err := uuid.Parse(d.TargetRunId)
	if err != nil {
		return d, err
	}
	replacement, err := controller.FreshRecoveryAccepted(ctx, id, effortDirectiveMessage(d))
	if err != nil {
		return d, err
	}
	if replacement == nil {
		next := proto.Clone(d).(*pb.EffortDirective)
		next.Delivery = pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN
		next.DeliveryReason = "original fresh-recovery receipt missing; preserve uncertainty without replacement"
		return next, nil
	}
	return freshRecoveryResult(s, d, replacement)
}

// Complete the original explicitly granted replacement, not a new grant. The
// directive/owner admission precedes this CAS projection. A crash between them
// is reconciled from the retained delivery; no executor is started here.
func (s *EffortService) bindRecoveredOrchestrator(ctx context.Context, d *pb.EffortDirective) error {
	if d.RecoveredRunId == "" || d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED || d.SupersededBy != "" || d.ExpiresAt == nil || !d.ExpiresAt.AsTime().After(s.now()) {
		return nil
	}
	e, o, err := s.repo.GetEffort(ctx, d.EffortRef)
	if err != nil {
		return err
	}
	for _, sub := range e.Subjects {
		if sub.Owner == "agent-manager" && sub.Role == "orchestrator" && sub.RunId == d.RecoveredRunId {
			return nil
		}
	}
	if s.directiveAuthority(e, o, d) != "" || s.originalDirectiveAuthority(e, d) != "" {
		return nil
	}
	if len(e.Subjects) >= 100 {
		return errors.New("recovery binding requires subject history consolidation; existing authority was not expanded")
	}
	expected := e.Revision
	e = proto.Clone(e).(*pb.EffortEnrollment)
	for _, sub := range e.Subjects {
		if sub.Owner == "agent-manager" && sub.Role == "orchestrator" && sub.RunId == d.TargetRunId {
			sub.Role = "previous-orchestrator"
		}
	}
	e.Subjects = append(e.Subjects, &pb.EffortSubject{Owner: "agent-manager", Kind: "run", Reference: d.RecoveredRunId, RunId: d.RecoveredRunId, Role: "orchestrator"})
	e.Revision++
	e.UpdatedAt = timestamppb.New(s.now().UTC())
	return s.repo.SaveEffort(ctx, e, o, expected, "recovery-binding:"+d.DirectiveId, effortDigest(d))
}
