package supervision

import (
	"context"
	"errors"
	"fmt"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EffortRunRegistry is the canonical run owner, not a private workload index.
// Its bounded pages are observations; absence never implies effort acceptance.
type EffortRunRegistry interface {
	List(context.Context, repository.RunListFilter) ([]*domain.Run, error)
}

func (s *EffortService) SetRunRegistry(r EffortRunRegistry) { s.runRegistry = r }

func publicActiveEffortReference(ref *eventpb.WorkReference) bool {
	return ref.GetKind() == "effort" && ref.GetId() != "" && ref.GetVerified() && ref.GetVisibility() == eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC && ref.GetState() == eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE
}
func (s *EffortService) reconcileRunRegistry(ctx context.Context, d *pb.EffortDiscovery) error {
	if s.runRegistry == nil {
		d.Findings = append(d.Findings, &pb.EffortDiscoveryFinding{Source: "agent-manager:runs", Code: "registry_unavailable", Reason: "canonical run registry is not configured"})
		d.Partial = true
		return nil
	}
	offset, err := s.repo.EffortRegistryOffset(ctx)
	if err != nil {
		return err
	}
	runs, err := s.runRegistry.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: 101, Offset: offset}})
	if err != nil {
		return err
	}
	more := len(runs) > 100
	if more {
		runs = runs[:100]
		d.Partial = true
	}
	for _, run := range runs {
		if run == nil {
			continue
		}
		for _, ref := range run.WorkReferences {
			// Private, unverified and unavailable references cannot become public enrollment.
			if !publicActiveEffortReference(ref) {
				continue
			}
			e, o, getErr := s.repo.GetEffort(ctx, ref.Id)
			if getErr != nil && !errors.Is(getErr, ErrNotFound) {
				return getErr
			}
			expected := uint64(0)
			if e == nil {
				e = &pb.EffortEnrollment{EffortRef: ref.Id, DisplayName: ref.Id, SourceRevision: ref.Revision}
				o = &pb.EffortBoardRow{Freshness: pb.EffortFreshness_EFFORT_FRESHNESS_FRESH, OutcomeStanding: &pb.EffortOutcomeStanding{State: "unknown", Attribution: "canonical AM work reference; acceptance not supplied"}, Limitations: []string{"accepted destination and target revision unavailable from runtime reference"}}
			} else {
				expected = e.Revision
			}
			if e.Withdrawn {
				continue
			}
			// A recovery child inherits work references. Its reference is not a
			// second grant: reconcile the retained owner admission before generic
			// joining can invalidate the original grant or duplicate its leader.
			if handled, recoveryErr := s.reconcileRegistryRecovery(ctx, e, run, ref.Relationship); handled || recoveryErr != nil {
				if recoveryErr != nil {
					d.Partial = true
					d.Findings = append(d.Findings, &pb.EffortDiscoveryFinding{Source: "agent-manager:runs", Code: "recovery_reconciliation_pending", Reason: recoveryErr.Error()})
				}
				continue
			}
			exists := false
			for _, subject := range e.Subjects {
				if subject.RunId == run.ID.String() {
					exists = true
				}
			}
			if exists {
				continue
			}
			if len(e.Subjects) >= 100 {
				d.Partial = true
				d.Findings = append(d.Findings, &pb.EffortDiscoveryFinding{Source: "agent-manager:runs", Code: "subject_limit", Reason: "effort subject sample capped at 100; use canonical run owner for complete accounting"})
				continue
			}
			role := ref.Relationship
			switch role {
			case "orchestrator", "worker", "supervisor", "reviewer", "investigator", "repair":
			default:
				role = "unknown"
				o.Limitations = append(o.Limitations, "runtime reference has no recognized assignment role; product runtime attribution remains unknown")
			}
			// New runtime subjects do not expand an existing mutation grant.
			e.Subjects = append(e.Subjects, &pb.EffortSubject{Owner: "agent-manager", Kind: "run", Reference: run.ID.String(), RunId: run.ID.String(), Role: role})
			if e.AuthorizedBy != "" && role == "orchestrator" {
				e.AuthorizedBy = ""
				e.PermittedActions = nil
				e.AuthorityExpiresAt = nil
				o.Limitations = append(o.Limitations, "runtime subject set changed; steering grant requires owner reconciliation")
			}
			e.Revision = expected + 1
			e.UpdatedAt = timestamppb.New(s.now().UTC())
			if e.Workspace == "" {
				o.ObservedAt = e.UpdatedAt
			}
			// Keep the workspace source cut independent of derived run joins. Otherwise
			// the next unchanged file scan drops joins and manufactures a change.
			if err = s.repo.SaveEffort(ctx, e, o, expected, fmt.Sprintf("registry:%s:%d", e.EffortRef, e.Revision), effortDigest(e)); err != nil {
				return err
			}
		}
	}
	next := 0
	if more {
		next = offset + len(runs)
	}
	return s.repo.SaveEffortRegistryOffset(ctx, next)
}
