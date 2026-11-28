package inference

import (
	"context"
	"errors"
	"fmt"
	"strings"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"
)

func recoveryReceipt(record AttemptRecord, state inferencev1.RecoveryReceiptState) *inferencev1.RecoveryReceipt {
	return &inferencev1.RecoveryReceipt{
		AttemptReference: attemptReference(record.Key), State: state,
		AccountingStatus: record.AccountingStatus, ChargedCredits: record.ChargedCredits,
	}
}

func recoveryStateForRecord(state AttemptState) inferencev1.RecoveryReceiptState {
	switch state {
	case AttemptPending:
		return inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_PENDING
	case AttemptSettled:
		return inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_SETTLED
	case AttemptReleased:
		return inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_RELEASED
	case AttemptOutputObserved:
		return inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_OUTPUT_UNAVAILABLE
	default:
		return inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_UNCERTAIN
	}
}

func ledgerState(receipt *inferencev1.RecoveryReceipt) AttemptState {
	switch receipt.GetState() {
	case inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_PENDING:
		return AttemptPending
	case inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_SETTLED:
		return AttemptSettled
	case inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_RELEASED:
		return AttemptReleased
	default:
		return AttemptUncertain
	}
}

func (r *ResourceRepository) hostedAdapter(provider string) (providers.Adapter, bool) {
	adapter, ok := r.adapters[strings.ToLower(strings.TrimSpace(provider))]
	return adapter, ok && adapter.Door != nil
}

func (r *ResourceRepository) recoverLatest(ctx context.Context, snapshot *LedgerSnapshot, request ProviderRequest) (ProviderResult, bool, error) {
	if snapshot == nil || snapshot.Attempt == nil || snapshot.Attempt.ValidationAttempt < request.ValidationAttempt {
		return ProviderResult{}, false, nil
	}
	record := *snapshot.Attempt
	switch record.State {
	case AttemptPreDispatchFailed:
		return ProviderResult{}, false, nil
	case AttemptOutputObserved:
		return r.recoverOutputObserved(ctx, snapshot, record)
	case AttemptSettled, AttemptReleased:
		return ProviderResult{Recovery: recoveryReceipt(record, recoveryStateForRecord(record.State))}, true, nil
	case AttemptDispatching, AttemptUncertain, AttemptPending:
		return r.recoverInFlight(ctx, snapshot, request, record)
	default:
		return ProviderResult{}, false, nil
	}
}

func (r *ResourceRepository) recoverOutputObserved(ctx context.Context, snapshot *LedgerSnapshot, record AttemptRecord) (ProviderResult, bool, error) {
	adapter, ok := r.hostedAdapter(record.Provider)
	if !ok {
		return ProviderResult{}, true, errors.New("recorded HostedDoor adapter is unavailable; recovery fails closed")
	}
	execution, err := adapter.Execute(ctx, providers.ExecutionRequest{IdempotencyKey: record.Key, RecoveryOnly: true, ExpectedKeyGeneration: snapshot.Generation})
	if errors.Is(err, providers.ErrDeviceKeyGenerationChanged) {
		return ProviderResult{}, true, ErrRootConflict
	}
	if err == nil && execution.Recovery != nil {
		receipt := execution.Recovery
		receipt.State = inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_OUTPUT_UNAVAILABLE
		return ProviderResult{Recovery: receipt}, true, nil
	}
	return ProviderResult{Recovery: recoveryReceipt(record, inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_OUTPUT_UNAVAILABLE)}, true, nil
}

func (r *ResourceRepository) recoverInFlight(ctx context.Context, snapshot *LedgerSnapshot, request ProviderRequest, record AttemptRecord) (ProviderResult, bool, error) {
	adapter, ok := r.hostedAdapter(record.Provider)
	if !ok {
		return ProviderResult{}, true, errors.New("recorded HostedDoor adapter is unavailable; recovery fails closed")
	}
	generation, err := adapter.Door.DeviceKeyGeneration(ctx)
	if err != nil {
		return ProviderResult{}, true, err
	}
	if generation != snapshot.Generation {
		return ProviderResult{}, true, ErrRootConflict
	}
	execution, err := adapter.Execute(ctx, providers.ExecutionRequest{IdempotencyKey: record.Key, RecoveryOnly: true, ExpectedKeyGeneration: snapshot.Generation})
	if err != nil {
		return ProviderResult{}, true, err
	}
	receipt := execution.Recovery
	if receipt == nil {
		receipt = recoveryReceipt(record, inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_UNCERTAIN)
	}
	_ = r.ledger.Set(ctx, request.RootOperationID, record.Ordinal, ledgerState(receipt), receipt.GetAccountingStatus(), receipt.ChargedCredits)
	return ProviderResult{Recovery: receipt}, true, nil
}

func (r *ResourceRepository) Run(ctx context.Context, request ProviderRequest) (ProviderResult, error) {
	definition, result, done, err := r.prepareRun(ctx, &request)
	if done || err != nil {
		return result, err
	}
	state := candidateRunState{}
	// The source arrives as an HTTP header and is not trusted to be normalized.
	// Normalize once so a case or whitespace difference cannot turn a valid
	// route into the "unsupported AI source" refusal.
	source := strings.ToLower(strings.TrimSpace(providers.Source(ctx)))
	for candidateIndex, candidate := range orderedCandidates(definition.Candidates, r.adapters, request.Profile) {
		disposition, result, err := r.processCandidate(ctx, request, definition, source, candidateIndex, candidate, &state)
		if err != nil || disposition == candidateFinished {
			return result, err
		}
	}
	return finishCandidateRun(request, state.failures, state.calls, state.samplingRefused, state.rootRequired)
}

type candidateRunState struct {
	failures        []string
	calls           int32
	usageIncomplete bool
	samplingRefused bool
	rootRequired    bool
}

type candidateDisposition uint8

const (
	candidateSkipped candidateDisposition = iota
	candidateFinished
	candidateReady
)

func (r *ResourceRepository) processCandidate(ctx context.Context, request ProviderRequest, definition rolecatalog.InferenceRole, source string, index int, candidate rolecatalog.Candidate, state *candidateRunState) (candidateDisposition, ProviderResult, error) {
	disposition, err := candidateSourceDisposition(source, candidate.Provider)
	if err != nil || disposition == candidateSkipped {
		return disposition, ProviderResult{}, err
	}
	plan, skip, err := r.prepareCandidate(ctx, request, definition, candidate, state.calls, &state.failures, &state.samplingRefused)
	if err != nil {
		return candidateFinished, ProviderResult{ProviderCallsMade: state.calls}, err
	}
	if skip {
		return candidateSkipped, ProviderResult{}, nil
	}
	disposition, result, start := r.startCandidateForRun(ctx, request, index, candidate, plan.adapter, state)
	if disposition != candidateReady {
		return disposition, result, start.err
	}
	continueCandidate, result, err := r.dispatchCandidate(ctx, request, definition, candidate, plan, start, &state.calls, &state.usageIncomplete, &state.failures)
	if err != nil || !continueCandidate {
		return candidateFinished, result, err
	}
	return candidateSkipped, ProviderResult{}, nil
}

func (r *ResourceRepository) startCandidateForRun(ctx context.Context, request ProviderRequest, index int, candidate rolecatalog.Candidate, adapter providers.Adapter, state *candidateRunState) (candidateDisposition, ProviderResult, attemptStart) {
	start := r.beginCandidateAttempt(ctx, request, index, candidate, adapter, state.calls, &state.failures)
	state.rootRequired = state.rootRequired || start.rootRequired
	if start.done || start.err != nil {
		return candidateFinished, start.result, start
	}
	if start.skip {
		return candidateSkipped, ProviderResult{}, start
	}
	return candidateReady, ProviderResult{}, start
}

func candidateSourceDisposition(source, provider string) (candidateDisposition, error) {
	allowed, err := sourceAllowsCandidate(source, provider)
	if err != nil {
		return candidateFinished, err
	}
	if !allowed {
		return candidateSkipped, nil
	}
	return candidateReady, nil
}

type attemptStart struct {
	attempt      AttemptRecord
	generation   string
	result       ProviderResult
	err          error
	skip         bool
	done         bool
	rootRequired bool
}

func (r *ResourceRepository) dispatchCandidate(ctx context.Context, request ProviderRequest, definition rolecatalog.InferenceRole, candidate rolecatalog.Candidate, plan candidatePlan, start attemptStart, calls *int32, usageIncomplete *bool, failures *[]string) (bool, ProviderResult, error) {
	*calls++
	meteredRootOperationID := browserRootOperationID(request)
	execution, err := executeTraced(ctx, plan.adapter, plan.resolved, candidateExecutionRequest(request, definition, candidate, plan, start, meteredRootOperationID))
	if plan.adapter.Door != nil && err == nil && execution.Recovery != nil {
		receipt := execution.Recovery
		_ = r.ledger.Set(ctx, request.RootOperationID, start.attempt.Ordinal, ledgerState(receipt), receipt.GetAccountingStatus(), receipt.ChargedCredits)
		return false, ProviderResult{Recovery: receipt, ProviderCallsMade: *calls}, nil
	}
	if err != nil {
		return r.handleDispatchError(ctx, request, candidate, plan.adapter, start.attempt, err, calls, usageIncomplete, failures)
	}
	if plan.adapter.Door != nil {
		if err := r.ledger.Set(ctx, request.RootOperationID, start.attempt.Ordinal, AttemptOutputObserved, "", nil); err != nil {
			return false, ProviderResult{Recovery: recoveryReceipt(start.attempt, inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_UNCERTAIN), ProviderCallsMade: *calls}, nil
		}
	}
	if request.SpendLimit != nil && !spendAuthorizationMatches(request, execution.SpendAuthorization) {
		if plan.adapter.Door != nil {
			return false, ProviderResult{Recovery: recoveryReceipt(start.attempt, inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_OUTPUT_UNAVAILABLE), ProviderCallsMade: *calls}, nil
		}
		*usageIncomplete = true
		*failures = append(*failures, candidate.Provider+": provider owner did not return matching spend-cap authorization")
		return true, ProviderResult{}, nil
	}
	return r.decodeExecution(request, candidate, plan, start.attempt, execution, meteredRootOperationID, calls, usageIncomplete, failures)
}

func browserRootOperationID(request ProviderRequest) string {
	if strings.HasPrefix(strings.TrimSpace(request.Role), "browser.") {
		return request.RootOperationID
	}
	return ""
}

func candidateExecutionRequest(request ProviderRequest, definition rolecatalog.InferenceRole, candidate rolecatalog.Candidate, plan candidatePlan, start attemptStart, meteredRootOperationID string) providers.ExecutionRequest {
	role := plan.executionRole
	return providers.ExecutionRequest{
		RootOperationID: meteredRootOperationID, Kind: sharedv1.RequestKind_REQUEST_KIND_STRUCTURED_EXTRACTION,
		Role: candidate.ResourceRole, ExpectedModel: role.Model, ExpectedArtifactDigest: plan.expectedArtifactDigest,
		ExpectedRightsDecisionSHA256: role.RightsDecisionSHA256, ExpectedRightsEvaluatedAt: role.RightsEvaluatedAt,
		ExpectedRightsReceiptSHA256: role.RightsReceiptSHA256, ExpectedPolicyFileSHA256: role.PolicyFileSHA256,
		Profile: request.Profile, BrowserSession: request.BrowserSession, InputText: plan.prompt, Timeout: definition.Timeout(),
		MaxOutputTokens: request.MaxOutputTokens, Temperature: plan.temperature, SchemaJSON: providerSchema(request),
		Attachments: requestAttachments(request), SpendLimit: request.SpendLimit,
		IdempotencyKey: start.attempt.Key, ExpectedKeyGeneration: start.generation,
	}
}

func (r *ResourceRepository) handleDispatchError(ctx context.Context, request ProviderRequest, candidate rolecatalog.Candidate, adapter providers.Adapter, attempt AttemptRecord, err error, calls *int32, usageIncomplete *bool, failures *[]string) (bool, ProviderResult, error) {
	var commandErr *providers.CommandError
	if errors.As(err, &commandErr) && commandErr.Code == providers.CodeOutcomeUncertain {
		return false, ProviderResult{ProviderCallsMade: *calls}, fmt.Errorf("%w: %v", ErrOutcomeUncertain, err)
	}
	if adapter.Door != nil {
		if errors.Is(err, providers.ErrDeviceKeyGenerationChanged) {
			return false, ProviderResult{ProviderCallsMade: *calls}, ErrRootConflict
		}
		_ = r.ledger.Set(ctx, request.RootOperationID, attempt.Ordinal, AttemptPreDispatchFailed, "", nil)
	}
	*usageIncomplete = true
	*failures = append(*failures, candidate.Provider+": "+err.Error())
	return true, ProviderResult{}, nil
}

func (r *ResourceRepository) decodeExecution(request ProviderRequest, candidate rolecatalog.Candidate, plan candidatePlan, attempt AttemptRecord, execution providers.ExecutionResult, meteredRootOperationID string, calls *int32, usageIncomplete *bool, failures *[]string) (bool, ProviderResult, error) {
	value, usage, err := decodeProviderResponse(candidate.Provider, execution.OutputText, plan.prompt)
	if err != nil {
		return r.handleDecodeError(candidate, plan.adapter, attempt, meteredRootOperationID, err, calls, usageIncomplete, failures)
	}
	if candidate.Provider == providers.ProviderMetered && meteredRootOperationID != "" && usage.rootOperationID != meteredRootOperationID {
		return false, ProviderResult{ProviderCallsMade: *calls}, fmt.Errorf("%w: metered response root_operation_id did not match %q", ErrOutcomeUncertain, meteredRootOperationID)
	}
	if *usageIncomplete {
		usage.inputProvenance = inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
		usage.outputProvenance = inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
		usage.costProvenance = inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
	}
	resolved, role := plan.resolved, plan.executionRole
	return false, ProviderResult{
		ValueJSON: value, BillingOperationID: execution.BillingOperationID, Provider: resolved.Provider, Model: resolved.Model,
		CoordinateConvention: resolved.CoordinateConvention,
		InputTokens:          usage.input, OutputTokens: usage.output, CostMicros: usage.cost,
		CreditsCharged: usage.creditsCharged, AccountingStatus: usage.accountingStatus,
		InputTokensProvenance: usage.inputProvenance, OutputTokensProvenance: usage.outputProvenance, CostProvenance: usage.costProvenance,
		Applied: plan.applied, SpendAuthorization: execution.SpendAuthorization,
		ContextWindow: resolved.ContextWindow, ProviderCallsMade: *calls,
		RightsReceiptSHA256: role.RightsReceiptSHA256, RightsContextSHA256: role.RightsContextSHA256,
	}, nil
}

func (r *ResourceRepository) handleDecodeError(candidate rolecatalog.Candidate, adapter providers.Adapter, attempt AttemptRecord, rootID string, err error, calls *int32, usageIncomplete *bool, failures *[]string) (bool, ProviderResult, error) {
	if candidate.Provider == providers.ProviderMetered && rootID != "" {
		return false, ProviderResult{ProviderCallsMade: *calls}, fmt.Errorf("%w: metered response could not be reconciled to root_operation_id %q: %v", ErrOutcomeUncertain, rootID, err)
	}
	if adapter.Door != nil {
		return false, ProviderResult{Recovery: recoveryReceipt(attempt, inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_OUTPUT_UNAVAILABLE), ProviderCallsMade: *calls}, nil
	}
	*usageIncomplete = true
	*failures = append(*failures, candidate.Provider+": "+err.Error())
	return true, ProviderResult{}, nil
}

func (r *ResourceRepository) beginCandidateAttempt(ctx context.Context, request ProviderRequest, index int, candidate rolecatalog.Candidate, adapter providers.Adapter, calls int32, failures *[]string) attemptStart {
	if adapter.Door == nil {
		return attemptStart{}
	}
	if !validRootOperationID(request.RootOperationID) {
		*failures = append(*failures, candidate.Provider+": root_operation_id is required before hosted dispatch")
		return attemptStart{skip: true, rootRequired: true}
	}
	if r.ledger == nil {
		*failures = append(*failures, candidate.Provider+": hosted inference recovery ledger is unavailable")
		return attemptStart{skip: true}
	}
	generation, err := adapter.Door.DeviceKeyGeneration(ctx)
	if err != nil {
		*failures = append(*failures, candidate.Provider+": "+err.Error())
		return attemptStart{skip: true}
	}
	attempt, err := r.ledger.Begin(ctx, request.RootOperationID, request.RequestDigest, generation,
		request.ValidationAttempt, index, candidate.Provider, candidate.ResourceRole)
	if err != nil {
		return attemptStart{result: ProviderResult{ProviderCallsMade: calls}, err: err, done: true}
	}
	if !attempt.Created {
		return r.recoverCandidateAttempt(ctx, request, attempt, generation, candidate, calls, failures)
	}
	return attemptStart{attempt: attempt, generation: generation}
}

func (r *ResourceRepository) recoverCandidateAttempt(ctx context.Context, request ProviderRequest, attempt AttemptRecord, generation string, candidate rolecatalog.Candidate, calls int32, failures *[]string) attemptStart {
	if attempt.State == AttemptPreDispatchFailed {
		*failures = append(*failures, candidate.Provider+": prior attempt failed before dispatch")
		return attemptStart{skip: true}
	}
	result, handled, err := r.recoverLatest(ctx, &LedgerSnapshot{Generation: generation, Attempt: &attempt}, request)
	if err != nil || handled {
		return attemptStart{result: result, err: err, done: true}
	}
	return attemptStart{result: ProviderResult{Recovery: recoveryReceipt(attempt, inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_UNCERTAIN), ProviderCallsMade: calls}, done: true}
}

type candidatePlan struct {
	adapter                providers.Adapter
	resolved               providers.ResolvedRole
	executionRole          providers.ResolvedRole
	expectedArtifactDigest string
	temperature            *float64
	support                providers.SamplingSupport
	maxOutputTokens        int32
	capSource              OutputCapSource
	prompt                 string
	applied                AppliedSettings
}

func (r *ResourceRepository) prepareCandidate(ctx context.Context, request ProviderRequest, definition rolecatalog.InferenceRole, candidate rolecatalog.Candidate, calls int32, failures *[]string, samplingRefused *bool) (candidatePlan, bool, error) {
	adapter, resolved, skip := r.resolveCandidateRole(ctx, request, candidate, failures)
	if skip {
		return candidatePlan{}, true, nil
	}
	plan := candidatePlan{adapter: adapter, resolved: resolved, executionRole: resolved}
	plan.support = resolved.TemperatureSupport()
	var err error
	plan.temperature, err = resolveTemperature(request, definition, plan.support)
	if err != nil {
		if errors.Is(err, ErrRoleForbidsSampling) {
			return candidatePlan{}, false, fmt.Errorf("%w: inference role %q declares no overridable sampling", ErrRoleForbidsSampling, request.Role)
		}
		*samplingRefused = true
		*failures = append(*failures, fmt.Sprintf("%s: temperature not honored by resolved role %s (%s)", candidate.Provider, candidate.ResourceRole, plan.support))
		return candidatePlan{}, true, nil
	}
	return r.applyCandidatePolicies(ctx, request, candidate, calls, failures, plan)
}

func (r *ResourceRepository) resolveCandidateRole(ctx context.Context, request ProviderRequest, candidate rolecatalog.Candidate, failures *[]string) (providers.Adapter, providers.ResolvedRole, bool) {
	adapter, ok := r.adapters[strings.ToLower(strings.TrimSpace(candidate.Provider))]
	if ok && !r.admitsRequest(ctx, request, candidate.Provider, adapter.Locality) {
		return providers.Adapter{}, providers.ResolvedRole{}, true
	}
	if !ok {
		*failures = append(*failures, candidate.Provider+": adapter unavailable")
		return providers.Adapter{}, providers.ResolvedRole{}, true
	}
	resolved, err := adapter.ResolveRole(ctx, candidate.ResourceRole)
	if err != nil {
		*failures = append(*failures, candidate.Provider+": "+err.Error())
		return providers.Adapter{}, providers.ResolvedRole{}, true
	}
	if !r.admitsRequest(ctx, request, candidate.Provider, resolved.Locality) {
		*failures = append(*failures, fmt.Sprintf("%s: resolved locality %q violates %s", candidate.Provider, resolved.Locality, request.Profile))
		return providers.Adapter{}, providers.ResolvedRole{}, true
	}
	return adapter, resolved, false
}

func (r *ResourceRepository) applyCandidatePolicies(ctx context.Context, request ProviderRequest, candidate rolecatalog.Candidate, calls int32, failures *[]string, plan candidatePlan) (candidatePlan, bool, error) {
	resolved := plan.resolved
	plan.maxOutputTokens, plan.capSource = resolveOutputCap(request, resolved)
	plan.prompt = providerPrompt(request, resolved.CoordinateConvention)
	if resolved.ContextWindow > 0 && estimateTokens(plan.prompt)+int64(plan.maxOutputTokens) > int64(resolved.ContextWindow) {
		return candidatePlan{}, false, fmt.Errorf("%w: assembled input and requested output cap do not fit the resolved model policy", ErrContextOverflow)
	}
	if request.MaxProviderCalls > 0 && calls >= request.MaxProviderCalls {
		return candidatePlan{}, false, fmt.Errorf("%w: request limit of %d provider calls exhausted", ErrUnavailable, request.MaxProviderCalls)
	}
	if err := r.authorizeCandidate(ctx, request, candidate, &plan, failures); err != nil {
		return candidatePlan{}, true, nil
	}
	plan.applied = AppliedSettings{TemperatureSent: plan.temperature, TemperatureSupport: plan.support, MaxOutputTokens: plan.maxOutputTokens, MaxOutputTokensSource: plan.capSource}
	return plan, false, nil
}

func (r *ResourceRepository) authorizeCandidate(ctx context.Context, request ProviderRequest, candidate rolecatalog.Candidate, plan *candidatePlan, failures *[]string) error {
	if !localModelRequiresRights(candidate.Provider, plan.resolved) {
		return nil
	}
	executionRole, artifactDigest, _, err := r.authorizeLocalCandidate(ctx, plan.adapter, candidate.ResourceRole, plan.resolved)
	if err != nil {
		*failures = append(*failures, candidate.Provider+": "+err.Error())
		return err
	}
	plan.executionRole, plan.expectedArtifactDigest = executionRole, artifactDigest
	if strings.EqualFold(request.PriorRightsProvider, executionRole.Provider) && request.PriorRightsModel == executionRole.Model &&
		request.PriorRightsContextSHA256 != "" && executionRole.RightsContextSHA256 != request.PriorRightsContextSHA256 {
		*failures = append(*failures, candidate.Provider+": local model rights changed between schema-repair attempts")
		return errors.New("local model rights changed between schema-repair attempts")
	}
	return nil
}

func finishCandidateRun(request ProviderRequest, failures []string, calls int32, samplingRefused, rootRequired bool) (ProviderResult, error) {
	result := ProviderResult{ProviderCallsMade: calls}
	if samplingRefused {
		return result, fmt.Errorf("%w: no candidate for inference role %q honors an explicit temperature (%s)", ErrUnsupportedSampling, request.Role, strings.Join(failures, "; "))
	}
	if rootRequired {
		return result, ErrRootRequired
	}
	// Browser callers distinguish an attempted route from a source/profile pair
	// that had no usable candidate; provider failures never invite downgrade.
	if calls == 0 && strings.HasPrefix(strings.TrimSpace(request.Role), "browser.") {
		return result, ErrNoEligibleRoute
	}
	return result, fmt.Errorf("%w: no inference candidate completed for role %q (%s)", ErrUnavailable, request.Role, strings.Join(failures, "; "))
}

func sourceAllowsCandidate(source, provider string) (bool, error) {
	switch source {
	case "local":
		return provider == providers.ProviderOllama, nil
	case "own-key":
		return provider != providers.ProviderMetered && provider != providers.ProviderOllama, nil
	case "vrooli-credits":
		return provider == providers.ProviderMetered, nil
	case "":
		return true, nil
	default:
		return false, fmt.Errorf("unsupported AI source")
	}
}

func (r *ResourceRepository) prepareRun(ctx context.Context, request *ProviderRequest) (rolecatalog.InferenceRole, ProviderResult, bool, error) {
	definition, ok := r.catalog.Roles[strings.TrimSpace(request.Role)]
	if !ok {
		return definition, ProviderResult{}, true, fmt.Errorf("%w: inference role %q is not declared", ErrUnavailable, request.Role)
	}
	if definition.IsDecision() {
		return definition, ProviderResult{}, true, fmt.Errorf("%w: inference role %q is a decision role; use Decide", ErrUnavailable, request.Role)
	}
	if err := validateGeometrySpendLimit(*request); err != nil {
		return definition, ProviderResult{}, true, err
	}
	if request.RequestDigest == "" {
		request.RequestDigest = canonicalRequestDigest(*request)
	}
	result, done, err := r.recoverRun(ctx, *request)
	return definition, result, done, err
}

func (r *ResourceRepository) recoverRun(ctx context.Context, request ProviderRequest) (ProviderResult, bool, error) {
	if !validRootOperationID(request.RootOperationID) || r.ledger == nil {
		return ProviderResult{}, false, nil
	}
	snapshot, err := r.ledger.Lookup(ctx, request.RootOperationID, request.RequestDigest)
	if err != nil || snapshot == nil || snapshot.Attempt == nil {
		return ProviderResult{}, false, err
	}
	adapter, ok := r.hostedAdapter(snapshot.Attempt.Provider)
	if !ok {
		return ProviderResult{}, true, errors.New("recorded HostedDoor adapter is unavailable; recovery fails closed")
	}
	generation, err := adapter.Door.DeviceKeyGeneration(ctx)
	if err != nil {
		return ProviderResult{}, true, err
	}
	if generation != snapshot.Generation {
		return ProviderResult{}, true, ErrRootConflict
	}
	return r.recoverLatest(ctx, snapshot, request)
}
