package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"ai-gateway/internal/providers"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// TypedRunner is the typed-value path (Service.Run) a decision role falls back
// to: schema gate, provider walk, local validation and repair included.
type TypedRunner func(context.Context, ProviderRequest) *inferencev1.RunResponse

// DecisionRepository walks a decision role's candidates.
type DecisionRepository interface {
	Decide(context.Context, DecisionRequest, TypedRunner) *inferencev1.DecideResponse
}

// candidateReasons are the stable walk codes a provider failure maps onto; any
// other failure is provider_failed.
var candidateReasons = map[string]struct{}{
	providers.CodeUnconfigured: {}, providers.CodeUnreachable: {}, providers.CodeRateLimited: {},
	providers.CodeProviderOverloaded: {}, providers.CodeContextOverflow: {}, providers.CodeProviderFailed: {},
	"timeout": {}, "missing_binary": {},
}

const (
	reasonLocalityForbidden = "locality_forbidden"
	reasonUnsupportedTypes  = "unsupported_question_types"
	reasonInvalidAnswers    = "invalid_answers"
)

// Decide walks the role's candidates in catalog order and returns the first
// complete, validated answer set. Every candidate gets a row: the ones that
// could not serve here (credential absent, model not pulled, locality the
// profile forbids) are skipped and say why; nothing about an absent hosted key
// is an error.
func (r *ResourceRepository) Decide(ctx context.Context, request DecisionRequest, classify TypedRunner) *inferencev1.DecideResponse {
	response := &inferencev1.DecideResponse{Usage: &inferencev1.Usage{}}
	definition, ok := r.catalog.Roles[strings.TrimSpace(request.Role)]
	if !ok {
		return decideError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, fmt.Sprintf("inference role %q is not declared", request.Role), "role")
	}
	if !definition.IsDecision() {
		return decideError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, fmt.Sprintf("inference role %q is a typed-value role; use Run", request.Role), "role")
	}
	body, err := systemOneBody(request)
	if err != nil {
		return decideError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, err.Error(), "questions")
	}
	for _, candidate := range definition.Candidates {
		row := &inferencev1.DecisionCandidate{Provider: candidate.Provider, ResourceRole: candidate.ResourceRole}
		response.Candidates = append(response.Candidates, row)
		switch {
		case response.Method != inferencev1.DecisionMethod_DECISION_METHOD_UNSPECIFIED:
			row.Status = inferencev1.DecisionCandidateStatus_DECISION_CANDIDATE_STATUS_NOT_ATTEMPTED
		case candidate.Provider == ProviderGateway:
			r.decideByEnum(ctx, request, candidate, classify, row, response)
		default:
			r.decideByModel(ctx, definition, request, candidate, body, row, response)
		}
	}
	if response.Method != inferencev1.DecisionMethod_DECISION_METHOD_UNSPECIFIED {
		finalizeUsage(response.Usage)
		return response
	}
	return decideError(response, terminalDecisionCode(response.Candidates),
		fmt.Sprintf("no decision candidate answered for role %q (%s)", request.Role, summarizeCandidates(response.Candidates)), "")
}

func (r *ResourceRepository) decideByModel(ctx context.Context, definition InferenceRole, request DecisionRequest, candidate Candidate, body string, row *inferencev1.DecisionCandidate, response *inferencev1.DecideResponse) {
	adapter, ok := r.adapters[strings.ToLower(strings.TrimSpace(candidate.Provider))]
	if !ok {
		skip(row, providers.CodeUnconfigured, "adapter not configured")
		return
	}
	resolved, err := adapter.ResolveRole(ctx, candidate.ResourceRole)
	if err != nil {
		skip(row, candidateReason(err), err.Error())
		return
	}
	row.Locality = resolved.Locality
	if !profileAdmits(request.Profile, resolved.Locality) {
		skip(row, reasonLocalityForbidden, fmt.Sprintf("profile %s forbids a %s candidate", request.Profile, resolved.Locality))
		return
	}
	started := time.Now()
	execution, err := adapter.Execute(ctx, providers.ExecutionRequest{
		Kind: sharedv1.RequestKind_REQUEST_KIND_DECISION, Role: candidate.ResourceRole,
		Profile: request.Profile, InputText: body, Timeout: definition.Timeout(),
	})
	row.LatencyMs = time.Since(started).Milliseconds()
	if err != nil {
		fail(row, candidateReason(err), err.Error())
		return
	}
	var output decisionOutput
	if err := json.Unmarshal([]byte(execution.OutputText), &output); err != nil {
		fail(row, reasonInvalidAnswers, "decode decision output: "+err.Error())
		return
	}
	answers, err := mapAnswers(request.Questions, output.Answers)
	if err != nil {
		fail(row, reasonInvalidAnswers, err.Error())
		return
	}
	row.Status = inferencev1.DecisionCandidateStatus_DECISION_CANDIDATE_STATUS_ANSWERED
	response.Answers, response.Method = answers, inferencev1.DecisionMethod_DECISION_METHOD_DECISION_MODEL
	response.Provider, response.Model = resolved.Provider, firstNonBlank(output.Model, resolved.Model)
	response.ResourceRole, response.Locality, response.LatencyMs = candidate.ResourceRole, resolved.Locality, row.LatencyMs
	addUsage(response.Usage, decisionUsage(output))
}

func (r *ResourceRepository) decideByEnum(ctx context.Context, request DecisionRequest, candidate Candidate, classify TypedRunner, row *inferencev1.DecisionCandidate, response *inferencev1.DecideResponse) {
	fallback, ok := enumFallback(request)
	if !ok {
		skip(row, reasonUnsupportedTypes, "the LLM enum fallback answers choice questions only")
		return
	}
	if classify == nil {
		skip(row, providers.CodeUnconfigured, "typed classify path is not wired")
		return
	}
	fallback.Role = candidate.ResourceRole
	started := time.Now()
	run := classify(ctx, fallback)
	row.LatencyMs = time.Since(started).Milliseconds()
	// Work performed is billed whether or not it produced a usable value.
	addUsage(response.Usage, run.GetUsage())
	if failure := run.GetError(); failure != nil || !run.GetValidated() {
		fail(row, providers.CodeProviderFailed, strings.TrimSpace(failure.GetCode().String()+": "+failure.GetMessage()))
		return
	}
	answers, err := enumAnswers(request.Questions, run.GetValueJson())
	if err != nil {
		fail(row, reasonInvalidAnswers, err.Error())
		return
	}
	row.Status = inferencev1.DecisionCandidateStatus_DECISION_CANDIDATE_STATUS_ANSWERED
	response.Answers, response.Method = answers, inferencev1.DecisionMethod_DECISION_METHOD_LLM_ENUM
	response.Provider, response.Model = run.GetProvider(), run.GetModel()
	response.ResourceRole, response.LatencyMs = candidate.ResourceRole, row.LatencyMs
}

func skip(row *inferencev1.DecisionCandidate, reason, detail string) {
	row.Status, row.Reason, row.Detail = inferencev1.DecisionCandidateStatus_DECISION_CANDIDATE_STATUS_SKIPPED, reason, detail
}

func fail(row *inferencev1.DecisionCandidate, reason, detail string) {
	row.Status, row.Reason, row.Detail = inferencev1.DecisionCandidateStatus_DECISION_CANDIDATE_STATUS_FAILED, reason, detail
}

func candidateReason(err error) string {
	var cmdErr *providers.CommandError
	if errors.As(err, &cmdErr) {
		if _, ok := candidateReasons[cmdErr.Code]; ok {
			return cmdErr.Code
		}
	}
	return providers.CodeProviderFailed
}

// terminalDecisionCode reports a context overflow only when every dispatched
// candidate refused for that reason; anything else is unavailability.
func terminalDecisionCode(rows []*inferencev1.DecisionCandidate) inferencev1.InferenceErrorCode {
	overflow, dispatched := false, false
	for _, row := range rows {
		if row.GetStatus() != inferencev1.DecisionCandidateStatus_DECISION_CANDIDATE_STATUS_FAILED {
			continue
		}
		dispatched = true
		if row.GetReason() != providers.CodeContextOverflow {
			return inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE
		}
		overflow = true
	}
	if dispatched && overflow {
		return inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_CONTEXT_OVERFLOW
	}
	return inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE
}

func summarizeCandidates(rows []*inferencev1.DecisionCandidate) string {
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		parts = append(parts, row.GetProvider()+"/"+row.GetResourceRole()+": "+row.GetReason())
	}
	return strings.Join(parts, "; ")
}

func decisionUsage(output decisionOutput) *inferencev1.Usage {
	usage := &inferencev1.Usage{
		InputTokensProvenance:  inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN,
		OutputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN,
		CostProvenance:         inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN,
	}
	if value := output.Usage.InputTokens; value != nil && *value >= 0 {
		usage.InputTokens, usage.InputTokensProvenance = *value, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED
	}
	if value := output.Usage.OutputTokens; value != nil && *value >= 0 {
		usage.OutputTokens, usage.OutputTokensProvenance = *value, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED
	}
	if cost := output.CostUSD; cost != nil && !math.IsNaN(*cost) && !math.IsInf(*cost, 0) && *cost >= 0 {
		usage.CostMicros, usage.CostProvenance = int64(math.Round(*cost*1_000_000)), inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED
	}
	return usage
}

func decideError(response *inferencev1.DecideResponse, code inferencev1.InferenceErrorCode, message, construct string) *inferencev1.DecideResponse {
	finalizeUsage(response.Usage)
	response.Error = &inferencev1.InferenceError{Code: code, Message: message, Construct: construct}
	return response
}

// profileAdmits applies the caller's locality stance to one candidate's
// declared locality: LOCAL_ONLY never leaves the host and REMOTE_ONLY never
// runs locally. Other profiles rank candidates by catalog order.
func profileAdmits(profile sharedv1.Profile, locality string) bool {
	switch profile {
	case sharedv1.Profile_PROFILE_LOCAL_ONLY:
		return strings.EqualFold(locality, "local")
	case sharedv1.Profile_PROFILE_REMOTE_ONLY:
		return strings.EqualFold(locality, "remote")
	default:
		return true
	}
}
