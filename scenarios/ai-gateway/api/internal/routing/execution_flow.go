package routing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ai-gateway/internal/providers"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type executionState struct {
	failures                    []string
	lastFailureClass            FailureClass
	lastCapacityEval            CapacityEvaluation
	capacityRejection           *CapacityEvaluation
	resolvedProfileRejection    bool
	providerExecutionDispatched bool
}

type candidateAttempt struct {
	result           *providers.ExecutionResult
	role             providers.ResolvedRole
	temperature      *float64
	capacity         CapacityEvaluation
	capacitySeen     bool
	capacityRejected bool
	failureReason    string
	failureClass     FailureClass
	profileRejected  bool
	dispatched       bool
	stop             bool
}

func (s *Service) execute(ctx context.Context, req *sharedv1.GatewayRequest, input string) (*routingv1.ExecuteRouteResponse, error) {
	started := s.now()
	validated, issues := s.prepareExecution(ctx, req, input)
	if len(issues) > 0 {
		return &routingv1.ExecuteRouteResponse{Valid: false, Issues: issues}, nil
	}
	req = validated
	plan := s.plan(ctx, req)
	if plan.selectedProvider == "" {
		return s.persistBlockedExecution(ctx, req, plan, started)
	}
	state := executionState{}
	for i, candidate := range s.executionAttempts(plan) {
		attempt := s.tryCandidate(ctx, req, input, plan, candidate, i)
		state.absorb(attempt)
		if attempt.result != nil {
			return s.persistSuccessfulExecution(ctx, req, plan, candidate, attempt, state, started, i > 0)
		}
		if attempt.stop {
			break
		}
	}
	return s.persistFailedExecution(ctx, req, plan, state, started)
}

func (s *Service) prepareExecution(ctx context.Context, req *sharedv1.GatewayRequest, input string) (*sharedv1.GatewayRequest, []*sharedv1.ValidationIssue) {
	issues := s.validator.Validate(req)
	if strings.TrimSpace(input) == "" {
		issues = append(issues, &sharedv1.ValidationIssue{Field: "input_text", Code: "required", Message: "input_text is required for execution"})
	}
	if len(issues) > 0 {
		return nil, issues
	}
	resolved, err := s.resolveAttachments(ctx, req)
	if err != nil {
		return nil, []*sharedv1.ValidationIssue{{Field: "attachments", Code: "unresolved", Message: err.Error()}}
	}
	return resolved, s.validator.Validate(resolved)
}

func (s *Service) persistBlockedExecution(ctx context.Context, req *sharedv1.GatewayRequest, plan routePlan, started time.Time) (*routingv1.ExecuteRouteResponse, error) {
	ev := s.evidence(req, plan, "blocked", s.now().Sub(started), false, []string{failureReasonNoProvider})
	ev.RejectionReason, ev.CapacityVerdict = blockedReason(plan), blockedCapacityVerdict(plan)
	if err := s.repo.Create(ctx, ev); err != nil {
		return nil, fmt.Errorf("persist blocked route evidence: %w", err)
	}
	return &routingv1.ExecuteRouteResponse{Valid: true, Evidence: ev, PolicyReasons: plan.policyReasons, Applied: appliedSettings(ev, req)}, nil
}

func (s *Service) tryCandidate(ctx context.Context, req *sharedv1.GatewayRequest, input string, plan routePlan, candidate *routingv1.RouteCandidate, index int) candidateAttempt {
	adapter, ok := s.adapters[candidate.GetProvider()]
	if !ok {
		return candidateAttempt{failureReason: failureReasonAdapterMissing, failureClass: FailurePolicyError}
	}
	ctxExec, cancel := executionContext(ctx, req)
	defer cancel()
	role, err := adapter.ResolveRole(ctxExec, candidate.GetRole())
	if err != nil {
		failure := ClassifyProviderFailure(err)
		s.recordOutcome(ctx, candidate.GetProvider(), candidate.GetRole(), req.GetKind(), failure, false)
		return candidateAttempt{failureReason: failureReasonRoleResolve, failureClass: failure.Class, stop: stopAfterPrimary(index, plan)}
	}
	admission := s.admitResolvedCandidate(ctxExec, req, plan, candidate, role)
	if admission.failureReason != "" {
		admission.stop = stopAfterPrimary(index, plan)
		return admission
	}
	capacity := s.admitExecutionCapacity(ctxExec, req, candidate)
	if capacity.failureReason != "" {
		capacity.stop = stopAfterPrimary(index, plan)
		return capacity
	}
	attempt := s.dispatchCandidate(ctxExec, req, input, plan, candidate, role, admission.temperature, capacity.capacity)
	attempt.capacitySeen = capacity.capacitySeen
	return attempt
}

func executionContext(ctx context.Context, req *sharedv1.GatewayRequest) (context.Context, context.CancelFunc) {
	if req.GetTimeoutMs() > 0 {
		return context.WithTimeout(ctx, time.Duration(req.GetTimeoutMs())*time.Millisecond)
	}
	return ctx, func() {}
}
func stopAfterPrimary(index int, plan routePlan) bool { return index == 0 && !plan.fallbackAllowed }

func (s *Service) admitResolvedCandidate(ctx context.Context, req *sharedv1.GatewayRequest, plan routePlan, candidate *routingv1.RouteCandidate, role providers.ResolvedRole) candidateAttempt {
	if !s.admitsRoute(ctx, req, candidate.GetProvider(), role.Locality) {
		candidate.RejectionReason = "locality_forbidden"
		candidate.Reasons = append(candidate.Reasons, "resolved role rejected by current locality, source-consent, or profile policy")
		return candidateAttempt{failureReason: failureReasonProfile, failureClass: FailurePolicyError, profileRejected: true}
	}
	if candidateNeedsPlannedRights(candidate, role, plan) && !s.plannedLocalRightsAdmit(ctx, candidate, role, plan) {
		return candidateAttempt{failureReason: failureReasonRights, failureClass: FailurePolicyError}
	}
	temperature, err := requestedTemperature(req, role)
	if err != nil {
		return candidateAttempt{failureReason: failureReasonSampling, failureClass: FailureUnsupportedSampling}
	}
	return candidateAttempt{temperature: temperature, role: role}
}

func candidateNeedsPlannedRights(candidate *routingv1.RouteCandidate, role providers.ResolvedRole, plan routePlan) bool {
	_, planned := plan.rights[candidateKey(candidate.GetProvider(), candidate.GetRole())]
	knownOwner := candidate.GetProvider() == providers.ProviderOllama || candidate.GetProvider() == providers.ProviderOllaya || candidate.GetProvider() == providers.ProviderPiiDetector
	localModel := strings.EqualFold(role.Locality, "local") && (strings.TrimSpace(role.Model) != "" || role.Rights != nil)
	return (strings.EqualFold(candidate.GetLocality(), "local") || strings.EqualFold(role.Locality, "local")) && (planned || knownOwner || localModel)
}

func (s *Service) admitExecutionCapacity(ctx context.Context, req *sharedv1.GatewayRequest, candidate *routingv1.RouteCandidate) candidateAttempt {
	capacity := s.acquireCapacity(ctx, req, candidate)
	if !strings.EqualFold(candidate.GetLocality(), "local") || requestCapacityRequirementBytes(req) <= 0 {
		return candidateAttempt{capacity: capacity}
	}
	if capacityExecutionAdmitted(capacity, requestCapacityRequirementBytes(req), allowReclaim(req)) {
		return candidateAttempt{capacity: capacity, capacitySeen: true}
	}
	if capacity.Verdict == CapacityFit {
		capacity.Verdict = CapacityUnknown
	}
	candidate.CapacityVerdict, candidate.RejectionReason = string(capacity.Verdict), "insufficient_capacity"
	s.releaseCapacity(ctx, capacity)
	return candidateAttempt{capacity: capacity, capacitySeen: true, capacityRejected: true, failureReason: failureReasonCapacity, failureClass: FailurePolicyError}
}

func (s *Service) dispatchCandidate(ctx context.Context, req *sharedv1.GatewayRequest, input string, plan routePlan, candidate *routingv1.RouteCandidate, role providers.ResolvedRole, temperature *float64, capacity CapacityEvaluation) candidateAttempt {
	key := candidateKey(candidate.GetProvider(), candidate.GetRole())
	rights := plan.rights[key]
	result, err := s.adapters[candidate.GetProvider()].Execute(ctx, providers.ExecutionRequest{
		Kind: req.GetKind(), Role: req.GetRole(), ExpectedModel: role.Model, ExpectedArtifactDigest: rights.runtimeDigest,
		ExpectedRightsDecisionSHA256: rights.decision.DecisionSHA256, ExpectedRightsEvaluatedAt: rights.decision.EvaluatedAt,
		ExpectedRightsReceiptSHA256: rights.receiptSHA256, ExpectedPolicyFileSHA256: rights.policyFileSHA256,
		Profile: req.GetProfile(), InputText: input, MaxOutputTokens: req.GetMaxOutputTokens(), Timeout: timeoutDuration(req.GetTimeoutMs()),
		Temperature: temperature, Attachments: req.GetAttachments(), ExpectedWorkerDigest: strings.TrimSpace(req.GetMetadata()["expected_worker_digest"]),
	})
	s.releaseCapacity(ctx, capacity)
	if err != nil {
		failure := ClassifyProviderFailure(err)
		s.recordOutcome(ctx, candidate.GetProvider(), candidate.GetRole(), req.GetKind(), failure, false)
		return candidateAttempt{capacity: capacity, capacitySeen: capacity.RequiredBytes > 0, failureReason: failureReasonExecute, failureClass: failure.Class, dispatched: true}
	}
	s.recordOutcome(ctx, candidate.GetProvider(), candidate.GetRole(), req.GetKind(), ProviderFailure{}, true)
	return candidateAttempt{result: &result, role: role, temperature: temperature, capacity: capacity, capacitySeen: capacity.RequiredBytes > 0, dispatched: true}
}

func (s *Service) persistSuccessfulExecution(ctx context.Context, req *sharedv1.GatewayRequest, plan routePlan, candidate *routingv1.RouteCandidate, attempt candidateAttempt, state executionState, started time.Time, fallbackUsed bool) (*routingv1.ExecuteRouteResponse, error) {
	ev := s.evidence(req, plan, "succeeded", s.now().Sub(started), fallbackUsed, state.failures)
	ev.SelectedProvider, ev.SelectedLocality, ev.SelectedModel = candidate.GetProvider(), candidate.GetLocality(), attempt.role.Model
	ev.TaskRole = strings.TrimSpace(req.GetTaskRole())
	if ev.TaskRole == "" {
		ev.TaskRole = strings.TrimSpace(req.GetRole())
	}
	ev.SelectedResourceRole = strings.TrimSpace(attempt.role.Role)
	if fallbackUsed {
		ev.FailureClass = string(state.lastFailureClass)
	}
	ev.BreakerState = candidate.GetBreakerState()
	applySamplingEvidence(ev, attempt.temperature, attempt.role.TemperatureSupport())
	evidenceCapacity := attempt.capacity
	if evidenceCapacity.RequiredBytes == 0 && state.capacityRejection != nil {
		evidenceCapacity = *state.capacityRejection
	}
	applyCapacityEvidence(ev, candidate, evidenceCapacity)
	if err := s.repo.Create(ctx, ev); err != nil {
		return nil, fmt.Errorf("persist successful route evidence: %w", err)
	}
	return &routingv1.ExecuteRouteResponse{Valid: true, Evidence: ev, OutputText: attempt.result.OutputText, PolicyReasons: plan.policyReasons, Applied: appliedSettings(ev, req)}, nil
}

func (s *Service) persistFailedExecution(ctx context.Context, req *sharedv1.GatewayRequest, plan routePlan, state executionState, started time.Time) (*routingv1.ExecuteRouteResponse, error) {
	status := "failed"
	if state.resolvedProfileRejection && !state.providerExecutionDispatched {
		status = "blocked"
	}
	ev := s.evidence(req, plan, status, s.now().Sub(started), len(state.failures) > 1, state.failures)
	ev.FailureClass = string(state.lastFailureClass)
	if state.resolvedProfileRejection {
		ev.RejectionReason = "locality_forbidden"
	}
	if state.lastCapacityEval.RequiredBytes > 0 {
		applyCapacityToSelectedCandidate(ev, plan, state.lastCapacityEval)
	}
	if err := s.repo.Create(ctx, ev); err != nil {
		return nil, fmt.Errorf("persist failed route evidence: %w", err)
	}
	return &routingv1.ExecuteRouteResponse{Valid: true, Evidence: ev, PolicyReasons: plan.policyReasons, Applied: appliedSettings(ev, req)}, nil
}

func applyCapacityToSelectedCandidate(ev *routingv1.RouteEvidence, plan routePlan, capacity CapacityEvaluation) {
	for _, candidate := range plan.candidates {
		if candidate.GetSelected() {
			applyCapacityEvidence(ev, candidate, capacity)
			return
		}
	}
}

func (s *executionState) absorb(attempt candidateAttempt) {
	if attempt.failureReason != "" {
		s.failures = append(s.failures, attempt.failureReason)
	}
	if attempt.failureClass != "" {
		s.lastFailureClass = attempt.failureClass
	}
	if attempt.capacitySeen {
		s.lastCapacityEval = attempt.capacity
	}
	if attempt.capacityRejected {
		rejected := attempt.capacity
		s.capacityRejection = &rejected
	}
	s.resolvedProfileRejection = s.resolvedProfileRejection || attempt.profileRejected
	s.providerExecutionDispatched = s.providerExecutionDispatched || attempt.dispatched
}
