package supervision

import (
	"context"
	"database/sql"
	"errors"
	"math"

	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (r *Repository) SaveEffortAssessment(ctx context.Context, a *pb.EffortAssessment, key, digest string) error {
	b, err := effortJSON.Marshal(a)
	if err != nil {
		return err
	}
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO supervision_effort_assessments(assessment_id,shared_operation_ref,assessment_json,recorded_at) VALUES(?,?,?,?) ON CONFLICT(shared_operation_ref) DO NOTHING`, a.AssessmentId, a.SharedOperationRef, string(b), formatTime(r.now().UTC()))
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	for _, ref := range a.EffortRefs {
		if _, err = tx.ExecContext(ctx, `INSERT INTO supervision_effort_assessment_subjects(assessment_id,effort_ref) VALUES(?,?)`, a.AssessmentId, ref); err != nil {
			return err
		}
	}
	if err = r.recordEffortTransition(ctx, tx, key, digest, b); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) LatestEffortAssessment(ctx context.Context, ref string) (*pb.EffortAssessment, error) {
	var b string
	err := r.db.QueryRowContext(ctx, `SELECT a.assessment_json FROM supervision_effort_assessments a JOIN supervision_effort_assessment_subjects s ON a.assessment_id=s.assessment_id WHERE s.effort_ref=? ORDER BY a.recorded_at DESC,a.assessment_id DESC LIMIT 1`, ref).Scan(&b)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a := &pb.EffortAssessment{}
	err = protojson.Unmarshal([]byte(b), a)
	return a, err
}

func (s *EffortService) RecordAssessment(ctx context.Context, req *pb.RecordEffortAssessmentRequest, actor EffortActor) (*pb.EffortAssessment, error) {
	if req == nil || req.Assessment == nil || actor.ID == "" || proto.Size(req) > 64*1024 {
		return nil, errors.New("bounded authenticated assessment required")
	}
	a := proto.Clone(req.Assessment).(*pb.EffortAssessment)
	if a.IdempotencyKey == "" || a.SharedOperationRef == "" || a.Rationale == "" || len(a.EffortRefs) > 100 || len(a.EvidenceRefs) == 0 {
		return nil, errors.New("assessment requires idempotency/shared operation identities, rationale, evidence and at most 100 efforts")
	}
	switch a.Disposition {
	case "quiet", "sample", "investigate", "steer", "unknown":
	default:
		return nil, errors.New("invalid assessment disposition")
	}
	switch a.Benefit {
	case "", "unknown":
		a.Benefit = "unknown"
	case "supported", "contradicted":
		if a.Hypothesis == "" || a.Comparison == "" {
			return nil, errors.New("benefit needs a falsifiable hypothesis and comparison")
		}
	default:
		return nil, errors.New("invalid benefit assessment")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := "assessment:" + a.IdempotencyKey
	digest := effortDigest(req) + actor.ID
	replay := &pb.EffortAssessment{}
	if ok, err := s.repo.ReplayEffortOperation(ctx, key, digest, replay); ok || err != nil {
		return replay, err
	}
	if len(a.EffortRefs) == 0 && !actor.Operator {
		return nil, errors.New("standing assessment requires operator authority")
	}
	seen := map[string]bool{}
	// Resolve the authenticated actor once from the existing run owner. This
	// closes the create-to-discovery interval without minting a grant or scanning
	// every run. Persisted work references remain attributable declarations.
	actorReferences := map[string]string{}
	if !actor.Operator && s.controller != nil {
		if id, parseErr := uuid.Parse(actor.ID); parseErr == nil {
			run, readErr := s.controller.GetRun(ctx, id)
			if readErr == nil && run != nil && run.ID == id {
				for _, ref := range run.WorkReferences {
					if publicActiveEffortReference(ref) && ref.GetRelationship() == "supervisor" {
						actorReferences[ref.GetId()] = ref.GetRevision()
					}
				}
			}
		}
	}
	for _, ref := range a.EffortRefs {
		if seen[ref] {
			return nil, errors.New("duplicate benefiting effort")
		}
		seen[ref] = true
		e, _, err := s.repo.GetEffort(ctx, ref)
		if err != nil {
			return nil, err
		}
		if e.Withdrawn && !actor.Operator {
			return nil, errors.New("effort withdrawn; no new supervisor assessment admitted")
		}
		observedRevision, ok := a.TargetRevisions[ref]
		if !ok || observedRevision != e.TargetRevision {
			return nil, ErrConflict
		}
		permitted := actor.supervises(e)
		if revision, ok := actorReferences[ref]; ok && revision == e.TargetRevision {
			permitted = true
		}
		for _, subject := range e.Subjects {
			if subject.Role == "supervisor" && subject.RunId == actor.ID {
				permitted = true
			}
		}
		if !permitted {
			return nil, errors.New("assessment caller is not an observed supervisor for every benefiting effort")
		}
	}
	a.AssessmentId = uuid.NewString()
	a.SupervisorRunId = actor.ID
	if a.ObservedAt == nil {
		a.ObservedAt = timestamppb.New(s.now().UTC())
	}
	if !a.ObservedAt.IsValid() || a.ObservedAt.AsTime().After(s.now()) {
		return nil, errors.New("assessment observation time must not be in the future")
	}
	if a.ObservedUsage == nil {
		a.ObservedUsage = &pb.EffortUsage{Partial: true, Source: "supervisor assessment", Limitations: []string{"full observation, judgment, disruption and shared idle costs unmeasured"}}
	}
	if err := validateEffortUsage(a.ObservedUsage); err != nil {
		return nil, err
	}
	// Allocation is not guessed from the number of efforts. Until an owner reports
	// an allocation, retain the full observed amount once in the shared boundary.
	if a.UnallocatedUsage == nil {
		a.UnallocatedUsage = proto.Clone(a.ObservedUsage).(*pb.EffortUsage)
	}
	if err := validateEffortUsage(a.UnallocatedUsage); err != nil {
		return nil, err
	}
	if !sameUsageAmounts(a.ObservedUsage, a.UnallocatedUsage) {
		return nil, errors.New("allocated shares are not supplied by this contract; retain the observed total as unallocated")
	}
	a.Limitations = append(a.Limitations, "shared operation retained once; costs are unallocated until owner allocation evidence is supplied")
	if a.AllowanceRef == "" {
		a.Limitations = append(a.Limitations, "standing/shared owner allowance is unknown; this record does not authorize spend")
	}
	if err := s.repo.SaveEffortAssessment(ctx, a, key, digest); err != nil {
		return nil, err
	}
	return a, nil
}
func sameUsageAmounts(a, b *pb.EffortUsage) bool {
	return (a.Tokens == nil) == (b.Tokens == nil) && a.GetTokens() == b.GetTokens() && (a.ReportedCostUsd == nil) == (b.ReportedCostUsd == nil) && math.Abs(a.GetReportedCostUsd()-b.GetReportedCostUsd()) < 1e-9 && (a.AgentSeconds == nil) == (b.AgentSeconds == nil) && math.Abs(a.GetAgentSeconds()-b.GetAgentSeconds()) < 1e-9
}
