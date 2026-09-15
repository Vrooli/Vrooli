package supervision

import (
	"context"
	"errors"
	"strings"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func validateRecoveryReferences(refs []string) error {
	if len(refs) == 0 || len(refs) > 30 {
		return errors.New("recovery requires 1-30 bounded owner evidence references")
	}
	seen := make(map[string]bool, len(refs))
	for _, ref := range refs {
		if strings.TrimSpace(ref) == "" || len(ref) > 2000 || seen[ref] {
			return errors.New("recovery references must be nonempty, bounded and distinct")
		}
		seen[ref] = true
	}
	return nil
}

func validateRecoveryExpectation(expectation *pb.EffortRecoveryExpectation) error {
	if expectation == nil {
		return nil
	}
	if strings.TrimSpace(expectation.ProgressCondition) == "" || len(expectation.ProgressCondition) > 2000 {
		return errors.New("recovery requires a bounded falsifiable progress condition")
	}
	return validateRecoveryReferences(expectation.BaselineEvidenceRefs)
}

// Verification is an authenticated comparison against the original expectation.
// These checks protect identity, chronology and evidence freshness; the verifier
// remains responsible for interpreting the referenced owner artifact's content.
func (s *EffortService) validateRecoveryVerification(ctx context.Context, e *pb.EffortEnrollment, d *pb.EffortDirective, v *pb.EffortRecoveryVerification, actor EffortActor) error {
	if !actor.supervises(e) {
		return errors.New("only the enrolled supervisor or operator may verify recovery")
	}
	if e.Withdrawn || e.TargetRevision != d.TargetRevision || e.AuthorizedBy == "" || e.AuthorityRef == "" || e.AuthorityExpiresAt == nil || !e.AuthorityExpiresAt.IsValid() || !e.AuthorityExpiresAt.AsTime().After(s.now()) {
		return errors.New("recovery verification requires the current unwithdrawn target and grant")
	}
	allowed, currentTarget := false, false
	for _, kind := range e.PermittedActions {
		allowed = allowed || kind == d.Kind
	}
	for _, subject := range e.Subjects {
		currentTarget = currentTarget || (subject.Role == "orchestrator" && subject.Owner == "agent-manager" && subject.RunId == recoveryTargetRunID(d))
	}
	if !allowed || !currentTarget {
		return errors.New("recovery target or action is no longer authorized by the current enrollment")
	}
	if d.SupersededBy != "" || d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED || d.DeliveredAt == nil || d.RecoveryExpectation == nil {
		return errors.New("recovery verification requires delivered original expectation")
	}
	if v.State != "progress-observed" && v.State != "owner-wait" && v.State != "failed" {
		return errors.New("verification state must be progress-observed, owner-wait or failed")
	}
	if strings.TrimSpace(v.Reason) == "" || len(v.Reason) > 2000 || len(v.NextOwnerCondition) > 2000 {
		return errors.New("recovery verification requires a bounded evidence comparison")
	}
	if v.State != "progress-observed" && strings.TrimSpace(v.NextOwnerCondition) == "" {
		return errors.New("unresolved recovery requires a next owner operation, assignment or reopening condition")
	}
	if err := validateRecoveryReferences(v.EvidenceRefs); err != nil {
		return err
	}
	if v.ObservedAt == nil || !v.ObservedAt.IsValid() || !v.ObservedAt.AsTime().After(d.DeliveredAt.AsTime()) || v.ObservedAt.AsTime().After(s.now()) {
		return errors.New("verification evidence must be observed after delivery and not in the future")
	}
	newEvidence := false
	baseline := make(map[string]bool, len(d.RecoveryExpectation.BaselineEvidenceRefs))
	for _, ref := range d.RecoveryExpectation.BaselineEvidenceRefs {
		baseline[ref] = true
	}
	for _, ref := range v.EvidenceRefs {
		if !baseline[ref] {
			newEvidence = true
		}
	}
	if !newEvidence {
		return errors.New("verification requires new owner evidence, not the unchanged baseline")
	}
	if v.State != "progress-observed" {
		return nil
	}
	if s.controller == nil {
		return errors.New("recovery target owner unavailable")
	}
	id, err := uuid.Parse(recoveryTargetRunID(d))
	if err != nil {
		return err
	}
	run, err := s.controller.GetRun(ctx, id)
	if err != nil || run == nil || run.ID != id {
		return errors.New("recovery target owner unavailable")
	}
	// Delivery can be reconciled after the recovered turn has already finished.
	// Compare its runtime to the immutable request baseline, not the later time
	// at which a lost response was reconciled. Evidence is still observed after
	// delivery above, and its content must satisfy the original progress condition.
	baselineTime := d.GetCreatedAt().AsTime()
	switch run.Status {
	case domain.RunStatusRunning:
		if run.EndedAt != nil || run.LastHeartbeat == nil || !run.LastHeartbeat.After(baselineTime) || run.LastHeartbeat.After(s.now()) || s.now().Sub(*run.LastHeartbeat) > s.config.StaleAfter {
			return errors.New("recovery runtime evidence is stale or lifecycle-inconsistent")
		}
	case domain.RunStatusComplete, domain.RunStatusNeedsReview:
		if run.EndedAt == nil || !run.EndedAt.After(baselineTime) || run.EndedAt.After(s.now()) {
			return errors.New("no post-delivery terminal owner evidence")
		}
	default:
		return errors.New("target has not resumed progress; record failed verification or an owner wait")
	}
	return nil
}
