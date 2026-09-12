package plans

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"plan-manager/internal/planmodel"

	"github.com/google/uuid"
)

// CreateCandidate stores a whole-plan revision. It snapshots only authored
// fields from the supplied candidate; identity and computed fields remain owned
// by the canonical plan and are restored when the candidate is applied.
func (s *service) CreateCandidate(ctx context.Context, candidate CandidateRevision) (CandidateRevision, error) {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	if strings.TrimSpace(candidate.PlanID) == "" {
		return CandidateRevision{}, ErrInvalidPlan{Reason: "candidate plan_id is required"}
	}
	if strings.TrimSpace(candidate.ExpectedBaseContentHash) == "" {
		return CandidateRevision{}, ErrInvalidPlan{Reason: "candidate expected_base_content_hash is required"}
	}
	base, err := s.Get(ctx, candidate.PlanID, candidate.Workspace)
	if err != nil {
		return CandidateRevision{}, err
	}
	if candidate.ExpectedBaseContentHash != base.ContentHash {
		return CandidateRevision{}, ErrCandidateStaleBase{PlanID: base.ID, Expected: candidate.ExpectedBaseContentHash, Actual: base.ContentHash}
	}
	if strings.TrimSpace(candidate.ProposalProvenance) == "" {
		return CandidateRevision{}, ErrInvalidPlan{Reason: "candidate proposal_provenance is required"}
	}
	if strings.TrimSpace(candidate.CandidatePlan.Title) == "" {
		return CandidateRevision{}, ErrInvalidPlan{Reason: "candidate plan title is required"}
	}
	candidate.ID = uuid.NewString()
	candidate.PlanID = base.ID
	candidate.Workspace = WorkspaceScope{ID: base.WorkspaceID, Root: base.WorkspaceRoot}
	candidate.CandidatePlan = candidateAuthoredPlan(base, candidate.CandidatePlan)
	candidate.State = CandidateRevisionPending
	candidate.CreatedAt = s.now()
	candidate.UpdatedAt = candidate.CreatedAt
	if err := validateCandidateExpiry(candidate.ExpiresAt, s.clock.Now()); err != nil {
		return CandidateRevision{}, err
	}
	if err := s.repo.SaveCandidate(ctx, candidate); err != nil {
		return CandidateRevision{}, err
	}
	return candidate, nil
}

func (s *service) GetCandidate(ctx context.Context, id string) (CandidateRevision, error) {
	candidate, found, err := s.repo.GetCandidate(ctx, strings.TrimSpace(id))
	if err != nil {
		return CandidateRevision{}, err
	}
	if !found {
		return CandidateRevision{}, ErrCandidateNotFound{ID: id}
	}
	return s.expireCandidateIfNeeded(ctx, candidate)
}

func (s *service) PreviewCandidate(ctx context.Context, id string) (CandidateRevisionPreview, error) {
	candidate, err := s.GetCandidate(ctx, id)
	if err != nil {
		return CandidateRevisionPreview{}, err
	}
	return s.previewCandidate(ctx, candidate)
}

func (s *service) ValidateCandidate(ctx context.Context, id string) (CandidateRevisionPreview, error) {
	return s.PreviewCandidate(ctx, id)
}

func (s *service) ApplyCandidate(ctx context.Context, id, expectedBaseHash string, acknowledgeQualityImpact bool) (CandidateRevision, Plan, CandidateRevisionPreview, error) {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	candidate, err := s.GetCandidate(ctx, id)
	if err != nil {
		return CandidateRevision{}, Plan{}, CandidateRevisionPreview{}, err
	}
	if candidate.State != CandidateRevisionPending {
		if candidate.State == CandidateRevisionApplied {
			base, getErr := s.Get(ctx, candidate.PlanID, candidate.Workspace)
			if getErr != nil {
				return CandidateRevision{}, Plan{}, CandidateRevisionPreview{}, getErr
			}
			preview, previewErr := s.previewCandidate(ctx, candidate)
			return candidate, base, preview, previewErr
		}
		return CandidateRevision{}, Plan{}, CandidateRevisionPreview{}, ErrCandidateState{ID: candidate.ID, State: candidate.State}
	}
	if !acknowledgeQualityImpact {
		return CandidateRevision{}, Plan{}, CandidateRevisionPreview{}, ErrInvalidPlan{Reason: "candidate apply requires acknowledge_quality_impact"}
	}
	if strings.TrimSpace(expectedBaseHash) == "" || expectedBaseHash != candidate.ExpectedBaseContentHash {
		return CandidateRevision{}, Plan{}, CandidateRevisionPreview{}, ErrCandidateStaleBase{PlanID: candidate.PlanID, Expected: candidate.ExpectedBaseContentHash, Actual: expectedBaseHash}
	}
	base, err := s.Get(ctx, candidate.PlanID, candidate.Workspace)
	if err != nil {
		return CandidateRevision{}, Plan{}, CandidateRevisionPreview{}, err
	}
	if base.ContentHash != candidate.ExpectedBaseContentHash {
		return CandidateRevision{}, Plan{}, CandidateRevisionPreview{}, ErrCandidateStaleBase{PlanID: base.ID, Expected: candidate.ExpectedBaseContentHash, Actual: base.ContentHash}
	}
	preview, err := s.previewCandidate(ctx, candidate)
	if err != nil {
		return CandidateRevision{}, Plan{}, CandidateRevisionPreview{}, err
	}
	if preview.QualityStatus == planmodel.QualityStatusFail {
		return CandidateRevision{}, Plan{}, preview, ErrInvalidPlan{Reason: "candidate validation failed"}
	}
	active, err := s.activeExecutionIDs(ctx, candidate.PlanID)
	if err != nil {
		return CandidateRevision{}, Plan{}, preview, err
	}
	if len(active) > 0 {
		return CandidateRevision{}, Plan{}, preview, ErrCandidateExecutionActive{PlanID: candidate.PlanID, ExecutionIDs: active}
	}
	next := candidateAuthoredPlan(base, candidate.CandidatePlan)
	updated, err := s.saveRecomputed(ctx, next)
	if err != nil {
		return CandidateRevision{}, Plan{}, preview, err
	}
	candidate.State = CandidateRevisionApplied
	candidate.AppliedAt = s.now()
	candidate.AppliedContentHash = updated.ContentHash
	candidate.UpdatedAt = candidate.AppliedAt
	if err := s.repo.SaveCandidate(ctx, candidate); err != nil {
		return CandidateRevision{}, Plan{}, preview, err
	}
	return candidate, updated, preview, nil
}

func (s *service) DiscardCandidate(ctx context.Context, id, reason string) (CandidateRevision, error) {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	candidate, err := s.GetCandidate(ctx, id)
	if err != nil {
		return CandidateRevision{}, err
	}
	if candidate.State != CandidateRevisionPending {
		return CandidateRevision{}, ErrCandidateState{ID: candidate.ID, State: candidate.State}
	}
	candidate.State = CandidateRevisionDiscarded
	candidate.DiscardReason = strings.TrimSpace(reason)
	candidate.UpdatedAt = s.now()
	if err := s.repo.SaveCandidate(ctx, candidate); err != nil {
		return CandidateRevision{}, err
	}
	return candidate, nil
}

func (s *service) previewCandidate(ctx context.Context, candidate CandidateRevision) (CandidateRevisionPreview, error) {
	base, err := s.Get(ctx, candidate.PlanID, candidate.Workspace)
	if err != nil {
		return CandidateRevisionPreview{}, err
	}
	next := candidateAuthoredPlan(base, candidate.CandidatePlan)
	report := planmodel.AssessPlanQuality(next, "")
	impact, _ := assessMutationImpactFromReport(base.Status, planmodel.AssessPlanQuality(base, ""), next, true)
	diagnostics := make([]CandidateValidationDiagnostic, 0, len(report.Findings))
	for _, finding := range report.Findings {
		diagnostics = append(diagnostics, CandidateValidationDiagnostic{Severity: string(finding.Severity), Code: finding.Code, Location: finding.Location, Message: finding.Message, Guidance: finding.Guidance})
	}
	return CandidateRevisionPreview{
		Candidate: candidate, BasePlan: base, Diff: diffCandidatePlans(base, next), Impact: impact,
		Rendered: RenderMarkdown(next), QualityStatus: report.Status, Diagnostics: diagnostics,
	}, nil
}

func candidateAuthoredPlan(base, incoming Plan) Plan {
	planmodel.PreserveNonAuthoredPlanFields(&incoming, &base)
	incoming.ID = base.ID
	incoming.Slug = base.Slug
	incoming.WorkspaceID = base.WorkspaceID
	incoming.WorkspaceRoot = base.WorkspaceRoot
	incoming.CreatedAt = base.CreatedAt
	incoming.UpdatedAt = base.UpdatedAt
	incoming.Status = base.Status
	incoming.ContentHash = base.ContentHash
	return incoming
}

func diffCandidatePlans(base, candidate Plan) CandidateRevisionDiff {
	baseValue := reflect.ValueOf(base)
	candidateValue := reflect.ValueOf(candidate)
	typ := baseValue.Type()
	changes := make([]CandidateFieldChange, 0)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if planmodel.PlanFieldClasses[field.Name] != planmodel.FieldClassAuthored || reflect.DeepEqual(baseValue.Field(i).Interface(), candidateValue.Field(i).Interface()) {
			continue
		}
		before, _ := json.Marshal(baseValue.Field(i).Interface())
		after, _ := json.Marshal(candidateValue.Field(i).Interface())
		changes = append(changes, CandidateFieldChange{Field: field.Name, BeforeJSON: string(before), AfterJSON: string(after)})
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Field < changes[j].Field })
	return CandidateRevisionDiff{Changes: changes}
}

func validateCandidateExpiry(value string, now time.Time) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return ErrInvalidPlan{Reason: "candidate expires_at must be RFC3339"}
	}
	if !expiresAt.After(now) {
		return ErrInvalidPlan{Reason: "candidate expires_at must be in the future"}
	}
	return nil
}

func (s *service) expireCandidateIfNeeded(ctx context.Context, candidate CandidateRevision) (CandidateRevision, error) {
	if candidate.State != CandidateRevisionPending || strings.TrimSpace(candidate.ExpiresAt) == "" {
		return candidate, nil
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, candidate.ExpiresAt)
	if err != nil || expiresAt.After(s.clock.Now()) {
		return candidate, nil
	}
	candidate.State = CandidateRevisionExpired
	candidate.UpdatedAt = s.now()
	if err := s.repo.SaveCandidate(ctx, candidate); err != nil {
		return CandidateRevision{}, err
	}
	return candidate, nil
}

type candidateExecutionReader interface {
	ActiveExecutionIDs(ctx context.Context, planID string) ([]string, error)
}

func (s *service) activeExecutionIDs(ctx context.Context, planID string) ([]string, error) {
	reader, ok := s.repo.(candidateExecutionReader)
	if !ok {
		return nil, fmt.Errorf("candidate execution guard is unavailable")
	}
	return reader.ActiveExecutionIDs(ctx, planID)
}
