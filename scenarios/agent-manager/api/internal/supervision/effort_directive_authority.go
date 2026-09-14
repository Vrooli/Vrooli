package supervision

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// The existing enrollment store also retains purpose-bound dispatcher grants.
// A grant-only record must not turn the standing service into its own work item.
// Before issuance, the authenticated owner/scope binding is the typed purpose
// marker; after issuance, server authorization is the stronger marker. Neither
// path uses an effort name or file-supplied tag.
func dispatchOnlyEnrollment(e *pb.EffortEnrollment) bool {
	return e != nil && e.DispatchAuthorization != nil && e.Workspace == "" && e.DestinationRef == "" && len(e.Subjects) == 0 && len(e.PermittedActions) == 0
}

// authorizationAnchorEnrollment extends the strict source predicate for a
// retained row that already received supervisor attribution before the source
// guard was installed. Only supervisor observations are tolerated; a business
// subject makes the row an ordinary effort. This remains generic and never
// relies on an effort name or display hint.
func authorizationAnchorEnrollment(e *pb.EffortEnrollment) bool {
	if e == nil || e.Workspace != "" || e.DestinationRef != "" || len(e.PermittedActions) != 0 {
		return false
	}
	// Before IssueDispatch, the authenticated owner binding is the only typed
	// purpose signal. After issuance, the server authorization remains the
	// stronger anchor. Both shapes are generic and tolerate supervisor
	// attribution added by discovery, without naming a particular effort.
	hasOwnerAnchor := e.AuthorizedBy != "" && e.SupervisorOwnerSubject != "" && e.SupervisorScope != "" && e.AuthorizedBy == e.SupervisorOwnerSubject
	if e.DispatchAuthorization == nil && !hasOwnerAnchor {
		return false
	}
	if len(e.Subjects) == 0 {
		return true
	}
	for _, subject := range e.Subjects {
		if subject == nil || subject.Role != "supervisor" {
			return false
		}
	}
	return true
}

func bindDirectiveAuthority(actor EffortActor, e *pb.EffortEnrollment) *pb.EffortDirectiveAuthorityBinding {
	if actor.Operator {
		return &pb.EffortDirectiveAuthorityBinding{Mode: "operator", Subject: actor.ID}
	}
	if e.SupervisorOwnerSubject != "" && actor.OwnerSubject == e.SupervisorOwnerSubject {
		for _, scope := range actor.Scopes {
			if scope == e.SupervisorScope {
				return &pb.EffortDirectiveAuthorityBinding{Mode: "delegated", Subject: actor.OwnerSubject, Scope: scope}
			}
		}
	}
	if actor.ID != "" && actor.ID == e.SupervisorRunId {
		return &pb.EffortDirectiveAuthorityBinding{Mode: "run", Subject: actor.ID}
	}
	return nil
}

// Only new dispatch uses this original grant fence. Observations may change the
// enrollment revision without changing the exact authorization route.
func (s *EffortService) originalDirectiveAuthority(current *pb.EffortEnrollment, d *pb.EffortDirective) string {
	original := d.GetSourceSnapshot().GetEnrollment()
	if original == nil || original.AuthorizedBy == "" || original.AuthorityRef == "" || original.AuthorizedBy != current.AuthorizedBy || original.AuthorityRef != current.AuthorityRef || original.TargetRevision != current.TargetRevision {
		return "original directive grant changed or is unavailable; issue a new authorized directive"
	}
	if original.AuthorityExpiresAt == nil || !original.AuthorityExpiresAt.IsValid() || !original.AuthorityExpiresAt.AsTime().After(s.now()) {
		return "original directive grant expired; renewal does not carry old instructions forward"
	}
	allowed, exactTarget := original.AutonomousSupervision, false
	for _, action := range original.PermittedActions {
		allowed = allowed || action == d.Kind
	}
	for _, subject := range original.Subjects {
		exactTarget = exactTarget || subject.Owner == "agent-manager" && subject.Kind == "run" && subject.RunId == d.TargetRunId && (original.AutonomousSupervision || subject.Role == "orchestrator")
	}
	if !allowed || !exactTarget {
		return "original grant does not authorize this action and exact target"
	}
	binding := d.AuthorityBinding
	if binding == nil {
		return "original directive authorization route unavailable; no new dispatch"
	}
	switch binding.Mode {
	case "operator":
		if binding.Subject == d.Issuer && binding.Subject != "" {
			return ""
		}
	case "run":
		if binding.Subject == d.Issuer && binding.Subject == original.SupervisorRunId && binding.Subject == current.SupervisorRunId {
			return ""
		}
	case "delegated":
		if binding.Subject != "" && binding.Scope != "" && binding.Subject == original.SupervisorOwnerSubject && binding.Subject == current.SupervisorOwnerSubject && binding.Scope == original.SupervisorScope && binding.Scope == current.SupervisorScope {
			return ""
		}
	}
	return "original directive supervisor binding changed; no new dispatch"
}

// Reconciliation cannot send effects and does not depend on a still-current
// effect grant. Withdrawal/supersession and the original snapshot remain intact.
func (s *EffortService) reconcileDirectiveReceipt(ctx context.Context, d *pb.EffortDirective) (*pb.EffortDirective, error) {
	next, err := s.inspectDirectiveReceipt(ctx, d)
	if err != nil || proto.Equal(next, d) {
		return d, err
	}
	next.Revision = d.Revision + 1
	if err = s.repo.SaveEffortDirective(ctx, next, d.Revision, "delivery:"+d.DirectiveId+":"+fmt.Sprint(next.Revision), effortDigest(next)); err != nil {
		return d, err
	}
	if next.Kind == pb.WatchActionKind_WATCH_ACTION_KIND_RECOVER_FRESH && next.RecoveredRunId != "" {
		return next, s.bindRecoveredOrchestrator(ctx, next)
	}
	return next, nil
}

// Inspection is read-only. Callers persist under either the delivery identity or
// the explicit operator request identity, never by issuing a second effect.
func (s *EffortService) inspectDirectiveReceipt(ctx context.Context, d *pb.EffortDirective) (*pb.EffortDirective, error) {
	if d.RefusalBeforeEffects {
		return d, errors.New("owner proved refusal before effects")
	}
	if d.Kind == pb.WatchActionKind_WATCH_ACTION_KIND_RECOVER_FRESH {
		return s.inspectFreshRecoveryReceipt(ctx, d)
	}
	reader, ok := s.controller.(interface {
		ContinuationAccepted(context.Context, uuid.UUID, string, string) (bool, error)
	})
	if !ok {
		return d, errors.New("uncertain continuation requires the original owner's admission receipt; no resend")
	}
	id, err := uuid.Parse(d.TargetRunId)
	if err != nil {
		return d, err
	}
	accepted, err := reader.ContinuationAccepted(ctx, id, effortDirectiveMessage(d), "effort-directive:"+d.DirectiveId)
	if err != nil {
		return d, err
	}
	if accepted {
		next := proto.Clone(d).(*pb.EffortDirective)
		next.Delivery = pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED
		next.DeliveryReason = "original owner admission reconciled without resending; original grant and supersession retained; progress still requires verification"
		next.DeliveredAt = timestamppb.New(s.now().UTC())
		return next, nil
	}
	if d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN {
		next := proto.Clone(d).(*pb.EffortDirective)
		next.Delivery = pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN
		next.DeliveryReason = "historical dispatch remains uncertain; no resend or authority renewal"
		return next, nil
	}
	return d, nil
}
