package routing

import (
	"context"
	"fmt"
	"log"
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
	preparationDurationNS       int64
	routePlanDurationNS         int64
	roleResolutionDurationNS    int64
	eligibilityDurationNS       int64
	capacityAdmissionDurationNS int64
	providerExecutionDurationNS int64
	providerTiming              providers.ProviderTiming
}

type candidateAttempt struct {
	result                      *providers.ExecutionResult
	role                        providers.ResolvedRole
	temperature                 *float64
	capacity                    CapacityEvaluation
	capacitySeen                bool
	capacityRejected            bool
	failureReason               string
	failureClass                FailureClass
	profileRejected             bool
	dispatched                  bool
	stop                        bool
	roleResolutionDurationNS    int64
	eligibilityDurationNS       int64
	capacityAdmissionDurationNS int64
	providerExecutionDurationNS int64
	providerTiming              providers.ProviderTiming
}

func (s *Service) execute(ctx context.Context, req *sharedv1.GatewayRequest, input string) (*routingv1.ExecuteRouteResponse, error) {
	started := s.now()
	state := executionState{}
	phaseStarted := s.now()
	validated, issues := s.prepareExecution(ctx, req, input)
	state.preparationDurationNS = durationNS(phaseStarted, s.now())
	if len(issues) > 0 {
		return &routingv1.ExecuteRouteResponse{Valid: false, Issues: issues}, nil
	}
	req = validated
	phaseStarted = s.now()
	plan := s.plan(ctx, req)
	state.routePlanDurationNS = durationNS(phaseStarted, s.now())
	if plan.selectedProvider == "" {
		return s.persistBlockedExecution(ctx, req, plan, started, state)
	}
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

func (s *Service) persistBlockedExecution(ctx context.Context, req *sharedv1.GatewayRequest, plan routePlan, started time.Time, state executionState) (*routingv1.ExecuteRouteResponse, error) {
	ev := s.evidence(req, plan, "blocked", s.now().Sub(started), false, []string{failureReasonNoProvider})
	applyExecutionTimings(ev, state)
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
	attempt := candidateAttempt{}
	phaseStarted := s.now()
	role, err := adapter.ResolveRole(ctxExec, candidate.GetRole())
	attempt.roleResolutionDurationNS = durationNS(phaseStarted, s.now())
	if err != nil {
		failure := ClassifyProviderFailure(err)
		s.recordOutcome(ctx, candidate.GetProvider(), candidate.GetRole(), req.GetKind(), failure, false)
		attempt.failureReason, attempt.failureClass, attempt.stop = failureReasonRoleResolve, failure.Class, stopAfterPrimary(index, plan)
		return attempt
	}
	phaseStarted = s.now()
	admission := s.admitResolvedCandidate(ctxExec, req, plan, candidate, role)
	attempt.eligibilityDurationNS = durationNS(phaseStarted, s.now())
	if admission.failureReason != "" {
		admission.stop = stopAfterPrimary(index, plan)
		admission.roleResolutionDurationNS = attempt.roleResolutionDurationNS
		admission.eligibilityDurationNS = attempt.eligibilityDurationNS
		return admission
	}
	phaseStarted = s.now()
	capacity := s.admitExecutionCapacity(ctxExec, req, candidate)
	capacity.capacityAdmissionDurationNS = durationNS(phaseStarted, s.now())
	if capacity.failureReason != "" {
		capacity.stop = stopAfterPrimary(index, plan)
		capacity.roleResolutionDurationNS = attempt.roleResolutionDurationNS
		capacity.eligibilityDurationNS = attempt.eligibilityDurationNS
		return capacity
	}
	dispatched := s.dispatchCandidate(ctxExec, req, input, plan, candidate, role, admission.temperature, capacity.capacity)
	dispatched.roleResolutionDurationNS = attempt.roleResolutionDurationNS
	dispatched.eligibilityDurationNS = attempt.eligibilityDurationNS
	dispatched.capacityAdmissionDurationNS = capacity.capacityAdmissionDurationNS
	dispatched.capacitySeen = capacity.capacitySeen
	return dispatched
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
	adapter := s.adapters[candidate.GetProvider()]
	started := s.now()
	result, err := adapter.Execute(ctx, providers.ExecutionRequest{
		Kind: req.GetKind(), Role: req.GetRole(), ExpectedModel: role.Model, ExpectedArtifactDigest: rights.runtimeDigest,
		ExpectedRightsDecisionSHA256: rights.decision.DecisionSHA256, ExpectedRightsEvaluatedAt: rights.decision.EvaluatedAt,
		ExpectedRightsReceiptSHA256: rights.receiptSHA256, ExpectedPolicyFileSHA256: rights.policyFileSHA256,
		Profile: req.GetProfile(), InputText: input, MaxOutputTokens: req.GetMaxOutputTokens(), Timeout: timeoutDuration(req.GetTimeoutMs()),
		Temperature: temperature, Attachments: req.GetAttachments(), ExpectedWorkerDigest: strings.TrimSpace(req.GetMetadata()["expected_worker_digest"]),
	})
	providerDurationNS := durationNS(started, s.now())
	s.releaseCapacity(ctx, capacity)
	if err != nil {
		failure := ClassifyProviderFailure(err)
		// The resource may have refused evidence planned from a fact that has
		// since changed; the next request plans from fresh facts.
		adapter.InvalidateFacts(providers.AnyFact)
		logExecutionFailure(req, candidate, failure, err)
		s.recordOutcome(ctx, candidate.GetProvider(), candidate.GetRole(), req.GetKind(), failure, false)
		return candidateAttempt{capacity: capacity, capacitySeen: capacity.RequiredBytes > 0, failureReason: failureReasonExecute, failureClass: failure.Class, dispatched: true, providerExecutionDurationNS: providerDurationNS}
	}
	s.recordOutcome(ctx, candidate.GetProvider(), candidate.GetRole(), req.GetKind(), ProviderFailure{}, true)
	return candidateAttempt{result: &result, role: role, temperature: temperature, capacity: capacity, capacitySeen: capacity.RequiredBytes > 0, dispatched: true, providerExecutionDurationNS: providerDurationNS, providerTiming: result.Timing}
}

func (s *Service) persistSuccessfulExecution(ctx context.Context, req *sharedv1.GatewayRequest, plan routePlan, candidate *routingv1.RouteCandidate, attempt candidateAttempt, state executionState, started time.Time, fallbackUsed bool) (*routingv1.ExecuteRouteResponse, error) {
	ev := s.evidence(req, plan, "succeeded", s.now().Sub(started), fallbackUsed, state.failures)
	applyExecutionTimings(ev, state)
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
	applyExecutionTimings(ev, state)
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
	s.roleResolutionDurationNS += attempt.roleResolutionDurationNS
	s.eligibilityDurationNS += attempt.eligibilityDurationNS
	s.capacityAdmissionDurationNS += attempt.capacityAdmissionDurationNS
	s.providerExecutionDurationNS += attempt.providerExecutionDurationNS
	s.providerTiming.AdmissionWaitNS += attempt.providerTiming.AdmissionWaitNS
	s.providerTiming.RuntimeNS += attempt.providerTiming.RuntimeNS
}

func applyExecutionTimings(ev *routingv1.RouteEvidence, state executionState) {
	ev.PreparationDurationNs = state.preparationDurationNS
	ev.RoutePlanDurationNs = state.routePlanDurationNS
	ev.RoleResolutionDurationNs = state.roleResolutionDurationNS
	ev.EligibilityDurationNs = state.eligibilityDurationNS
	ev.CapacityAdmissionDurationNs = state.capacityAdmissionDurationNS
	ev.ProviderExecutionDurationNs = state.providerExecutionDurationNS
	ev.ProviderRuntimeDurationNs = state.providerTiming.RuntimeNS
	ev.ProviderAdmissionWaitNs = state.providerTiming.AdmissionWaitNS
	ev.OverheadDurationNs = overheadDurationNS(ev.GetTotalDurationNs(), state)
}

// overheadDurationNS is the route time that was neither provider work nor a
// wait for provider admission. A resource that reports its provider runtime
// lets its own start-up and verification count as overhead; otherwise the
// whole resource command is counted as provider time.
func overheadDurationNS(totalNS int64, state executionState) int64 {
	provider := state.providerExecutionDurationNS
	if state.providerTiming.RuntimeNS > 0 {
		provider = state.providerTiming.RuntimeNS + state.providerTiming.AdmissionWaitNS
	}
	return max(totalNS-provider, 0)
}

// logExecutionFailure keeps the resource's own reason for a failed dispatch.
// Route evidence stores only the stable failure class; without this line the
// cause of a provider failure is lost.
func logExecutionFailure(req *sharedv1.GatewayRequest, candidate *routingv1.RouteCandidate, failure ProviderFailure, err error) {
	log.Printf("route execution failed scenario=%s provider=%s role=%s kind=%s class=%s: %s",
		firstNonEmpty(req.GetScenario(), "unknown"), candidate.GetProvider(), candidate.GetRole(), req.GetKind(), failure.Class, providers.FailureDetail(err))
}

func durationNS(started, ended time.Time) int64 {
	duration := ended.Sub(started)
	if duration < 0 {
		return 0
	}
	return int64(duration)
}
