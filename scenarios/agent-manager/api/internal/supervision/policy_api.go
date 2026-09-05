package supervision

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) GetPolicy(ctx context.Context, req *domainpb.GetSupervisionPolicyRequest) (*domainpb.SupervisionPolicyRecord, error) {
	if s.policies == nil {
		return nil, errors.New("supervision policy service is unavailable")
	}
	var record PolicyRecord
	var err error
	if req == nil || strings.TrimSpace(req.GetVersion()) == "" {
		record, err = s.policies.Active(ctx)
	} else {
		record, err = s.policies.Get(ctx, req.GetVersion())
	}
	if err != nil {
		return nil, err
	}
	result := policyRecordProto(record)
	report := &domainpb.SupervisionReplayReport{Version: record.Policy.Version}
	err = s.policies.db.QueryRowContext(ctx, `SELECT sample_count,false_positives,false_negatives,safety_violations,completion_impact,rollout_samples,replay_passed,rollout_passed FROM supervision_policy_gates WHERE version=?`, record.Policy.Version).Scan(&report.SampleCount, &report.FalsePositives, &report.FalseNegatives, &report.SafetyViolations, &report.CompletionImpact, &report.RolloutSamples, &report.ReplayPassed, &report.RolloutPassed)
	if err == nil {
		result.Evaluation = report
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	err = s.policies.db.QueryRowContext(ctx, `SELECT identity_digest FROM supervision_inference_identity WHERE version=?`, record.Policy.Version).Scan(&result.InferenceIdentityDigest)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return result, nil
}

func (s *Service) CreatePolicyCandidate(ctx context.Context, req *domainpb.CreateSupervisionPolicyCandidateRequest) (*domainpb.SupervisionPolicyRecord, error) {
	if s.policies == nil || req == nil || req.GetPolicy() == nil {
		return nil, errors.New("policy service and definition are required")
	}
	record, err := s.policies.CreateCandidate(ctx, policyFromProto(req.GetPolicy()), req.GetSupersedes(), req.GetCreatedBy())
	if err != nil {
		return nil, err
	}
	return policyRecordProto(record), nil
}

func (s *Service) RecordPolicyOutcome(ctx context.Context, req *domainpb.RecordSupervisionOutcomeRequest) (*domainpb.RecordSupervisionOutcomeResponse, error) {
	if s.policies == nil || req == nil || req.GetOutcome() == nil {
		return nil, errors.New("policy service and outcome are required")
	}
	result, err := s.policies.RecordOutcome(ctx, outcomeFromProto(req.GetOutcome()))
	if err != nil {
		return nil, err
	}
	response := &domainpb.RecordSupervisionOutcomeResponse{Outcome: outcomeProto(result.Outcome), IdempotentReplay: result.Reused, SourceLedgerId: result.LedgerID, SourceLedgerSynced: result.LedgerError == nil && result.LedgerID != ""}
	if result.LedgerError != nil {
		response.DegradationReason = result.LedgerError.Error()
	}
	return response, nil
}

func (s *Service) EvaluatePolicy(ctx context.Context, req *domainpb.EvaluateSupervisionPolicyRequest) (*domainpb.SupervisionReplayReport, error) {
	if s.policies == nil || req == nil {
		return nil, errors.New("policy service and request are required")
	}
	report, err := s.policies.EvaluateCandidate(ctx, req.GetVersion(), int(req.GetRolloutSamples()), ReplayThresholds{MinSamples: int(req.GetMinSamples()), MaxFalsePositiveRate: req.GetMaxFalsePositiveRate(), MaxFalseNegativeRate: req.GetMaxFalseNegativeRate(), MinRolloutSamples: int(req.GetMinRolloutSamples())})
	if err != nil {
		return nil, err
	}
	return &domainpb.SupervisionReplayReport{Version: report.Version, SampleCount: int32(report.SampleCount), FalsePositives: int32(report.FalsePositives), FalseNegatives: int32(report.FalseNegatives), SafetyViolations: int32(report.SafetyViolations), CompletionImpact: report.CompletionImpact, RolloutSamples: int32(report.RolloutSamples), ReplayPassed: report.ReplayPassed, RolloutPassed: report.RolloutPassed}, nil
}

func (s *Service) PromotePolicy(ctx context.Context, req *domainpb.PromoteSupervisionPolicyRequest) (*domainpb.SupervisionPolicyRecord, error) {
	record, err := s.policies.Promote(ctx, req.GetVersion(), req.GetReviewedBy())
	return policyRecordProto(record), err
}

func (s *Service) RejectPolicy(ctx context.Context, req *domainpb.RejectSupervisionPolicyRequest) (*domainpb.SupervisionPolicyRecord, error) {
	if err := s.policies.Reject(ctx, req.GetVersion(), req.GetReviewedBy(), req.GetReason()); err != nil {
		return nil, err
	}
	record, err := s.policies.Get(ctx, req.GetVersion())
	return policyRecordProto(record), err
}

func (s *Service) RollbackPolicy(ctx context.Context, req *domainpb.RollbackSupervisionPolicyRequest) (*domainpb.SupervisionPolicyRecord, error) {
	record, err := s.policies.Rollback(ctx, req.GetActiveVersion(), req.GetReviewedBy())
	return policyRecordProto(record), err
}

func (s *Service) SetPolicyDisabled(ctx context.Context, req *domainpb.SetSupervisionPolicyDisabledRequest) (*domainpb.SupervisionPolicyControl, error) {
	if err := s.policies.SetDisabled(ctx, req.GetDisabled(), req.GetReason(), req.GetActor()); err != nil {
		return nil, err
	}
	disabled, reason, err := s.policies.Disabled(ctx)
	return &domainpb.SupervisionPolicyControl{Disabled: disabled, Reason: reason}, err
}

func (s *Service) ListPolicyOutcomes(ctx context.Context, req *domainpb.ListSupervisionOutcomesRequest) (*domainpb.ListSupervisionOutcomesResponse, error) {
	outcomes, err := s.policies.ListOutcomes(ctx, req.GetPolicyVersion(), int(req.GetLimit()), req.GetWatchId())
	if err != nil {
		return nil, err
	}
	response := &domainpb.ListSupervisionOutcomesResponse{Outcomes: make([]*domainpb.SupervisionOutcomeRecord, 0, len(outcomes))}
	for _, outcome := range outcomes {
		response.Outcomes = append(response.Outcomes, outcomeProto(outcome))
	}
	return response, nil
}

func policyFromProto(value *domainpb.SupervisionPolicyDefinition) SupervisionPolicy {
	return SupervisionPolicy{Version: value.GetVersion(), EventCount: value.GetEventCount(), QuietSeconds: value.GetQuietSeconds(), FrictionThreshold: value.GetFrictionThreshold(), Terminal: value.GetTerminal(), AllowedActions: append([]string(nil), value.GetAllowedActions()...), ClassifierRevision: value.GetClassifierRevision(), EvaluatorDigest: value.GetEvaluatorDigest()}
}

func policyRecordProto(value PolicyRecord) *domainpb.SupervisionPolicyRecord {
	return &domainpb.SupervisionPolicyRecord{Policy: &domainpb.SupervisionPolicyDefinition{Version: value.Policy.Version, EventCount: value.Policy.EventCount, QuietSeconds: value.Policy.QuietSeconds, FrictionThreshold: value.Policy.FrictionThreshold, Terminal: value.Policy.Terminal, AllowedActions: append([]string(nil), value.Policy.AllowedActions...), ClassifierRevision: value.Policy.ClassifierRevision, EvaluatorDigest: value.Policy.EvaluatorDigest}, State: policyStateProto(value.State), Digest: value.Digest, Supersedes: value.Supersedes, CreatedBy: value.CreatedBy, ReviewedBy: value.ReviewedBy, RejectionReason: value.RejectionReason}
}

func outcomeFromProto(value *domainpb.SupervisionOutcomeRecord) SupervisionOutcome {
	outcome := SupervisionOutcome{ID: value.GetOutcomeId(), IdempotencyKey: value.GetIdempotencyKey(), PolicyVersion: value.GetPolicyVersion(), FamilyExecutionID: value.GetFamilyExecutionId(), WatchID: value.GetWatchId(), DecisionID: value.GetDecisionId(), ActionID: value.GetActionId(), ChildRunID: value.GetChildRunId(), EvidenceIDs: append([]string(nil), value.GetEvidenceIds()...), PredictedClass: value.GetPredictedClass(), ObservedClass: value.GetObservedClass(), Overridden: value.GetOverridden(), Counterexample: value.GetCounterexample(), SafetyViolation: value.GetSafetyViolation(), CompletionImpact: value.GetCompletionImpact(), CompletionImpactObserved: value.GetCompletionImpactObserved(), Supersedes: value.GetSupersedesOutcomeId()}
	if value.GetCreatedAt().IsValid() {
		outcome.CreatedAt = value.GetCreatedAt().AsTime()
	}
	if value.GetExpiresAt().IsValid() {
		outcome.ExpiresAt = value.GetExpiresAt().AsTime()
	}
	return outcome
}

func outcomeProto(value SupervisionOutcome) *domainpb.SupervisionOutcomeRecord {
	result := &domainpb.SupervisionOutcomeRecord{OutcomeId: value.ID, IdempotencyKey: value.IdempotencyKey, PolicyVersion: value.PolicyVersion, FamilyExecutionId: value.FamilyExecutionID, WatchId: value.WatchID, DecisionId: value.DecisionID, ActionId: value.ActionID, ChildRunId: value.ChildRunID, EvidenceIds: append([]string(nil), value.EvidenceIDs...), PredictedClass: value.PredictedClass, ObservedClass: value.ObservedClass, Overridden: value.Overridden, Counterexample: value.Counterexample, SafetyViolation: value.SafetyViolation, CompletionImpact: value.CompletionImpact, CompletionImpactObserved: value.CompletionImpactObserved, SupersedesOutcomeId: value.Supersedes}
	if !value.CreatedAt.IsZero() {
		result.CreatedAt = timestamppb.New(value.CreatedAt)
	}
	if !value.ExpiresAt.IsZero() {
		result.ExpiresAt = timestamppb.New(value.ExpiresAt)
	}
	return result
}

func policyStateProto(value string) domainpb.SupervisionPolicyState {
	return map[string]domainpb.SupervisionPolicyState{"candidate": domainpb.SupervisionPolicyState_SUPERVISION_POLICY_STATE_CANDIDATE, "active": domainpb.SupervisionPolicyState_SUPERVISION_POLICY_STATE_ACTIVE, "retired": domainpb.SupervisionPolicyState_SUPERVISION_POLICY_STATE_RETIRED, "rejected": domainpb.SupervisionPolicyState_SUPERVISION_POLICY_STATE_REJECTED, "rolled_back": domainpb.SupervisionPolicyState_SUPERVISION_POLICY_STATE_ROLLED_BACK}[value]
}
