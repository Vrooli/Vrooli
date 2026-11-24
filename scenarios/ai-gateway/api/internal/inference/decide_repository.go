package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"

	profilepolicy "github.com/vrooli/ai-provider-go/routing"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// TypedRunner is the typed-value path (Service.Run) a decision role falls back
// to: schema gate, provider walk, local validation and repair included.
type TypedRunner func(context.Context, ProviderRequest) *inferencev1.RunResponse

// DecisionRepository walks a decision role's candidates for a batch of states.
type DecisionRepository interface {
	DecideBatch(context.Context, []DecisionItem, TypedRunner) []*inferencev1.DecideResponse
}

var (
	_ DecisionRepository = (*ResourceRepository)(nil)
	_ DecisionRepository = UnavailableRepository{}
)

// DecisionItem is one state of a decision batch. Every item of one batch shares
// the role, questions, profile, privacy and pin; only the state differs. Ctx is
// the context the item's route event is traced in.
type DecisionItem struct {
	Ctx     context.Context
	Request DecisionRequest
}

// candidateReasons are the stable walk codes a provider failure maps onto; any
// other failure is provider_failed.
var candidateReasons = map[string]struct{}{
	providers.CodeUnconfigured: {}, providers.CodeUnreachable: {}, providers.CodeRateLimited: {},
	providers.CodeProviderOverloaded: {}, providers.CodeContextOverflow: {}, providers.CodeProviderFailed: {},
	providers.CodeInsufficientVRAM: {}, providers.CodeInsufficientMemory: {}, providers.CodeLimitsExceeded: {},
	"timeout": {}, "missing_binary": {},
}

const (
	reasonLocalityForbidden = "locality_forbidden"
	reasonUnsupportedTypes  = "unsupported_question_types"
	constructPinnedRole     = "pin_resource_role"
	reasonInvalidAnswers    = "invalid_answers"
	reasonSourceNotSelected = "source_not_selected"
	reasonNoEligibleRoute   = "no_eligible_route"
)

// Decide answers one request: a batch of one, so a single decision and a batch
// walk the same path.
func (r *ResourceRepository) Decide(ctx context.Context, request DecisionRequest, classify TypedRunner) *inferencev1.DecideResponse {
	return r.DecideBatch(ctx, []DecisionItem{{Ctx: ctx, Request: request}}, classify)[0]
}

// DecideBatch walks the role's candidates in catalog order for every state and
// returns, per state, the first complete, validated answer set. Every
// candidate gets a row on every state: the ones that could not serve here
// (credential absent, model not pulled, locality the profile forbids) are
// skipped and say why; nothing about an absent hosted key is an error.
//
// A candidate is dispatched once for all the states still unanswered when the
// walk reaches it, so a batch of N states costs one resource command per
// candidate tried, not N. A state the candidate could not answer falls through
// to the next candidate on its own.
func (r *ResourceRepository) DecideBatch(ctx context.Context, items []DecisionItem, classify TypedRunner) []*inferencev1.DecideResponse {
	responses := make([]*inferencev1.DecideResponse, len(items))
	for index := range responses {
		responses[index] = &inferencev1.DecideResponse{Usage: &inferencev1.Usage{}}
	}
	if len(items) == 0 {
		return responses
	}
	refuseAll := func(code inferencev1.InferenceErrorCode, message, construct string) []*inferencev1.DecideResponse {
		for index := range responses {
			responses[index] = decideError(responses[index], code, message, construct)
		}
		return responses
	}
	template := items[0].Request
	definition, ok := r.catalog.Roles[strings.TrimSpace(template.Role)]
	if !ok {
		return refuseAll(inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, fmt.Sprintf("inference role %q is not declared", template.Role), "role")
	}
	if !definition.IsDecision() {
		return refuseAll(inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, fmt.Sprintf("inference role %q is a typed-value role; use Run", template.Role), "role")
	}
	candidates, err := walkCandidates(definition, template, r.adapters)
	if err != nil {
		return refuseAll(inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, err.Error(), constructPinnedRole)
	}
	walk := make([]int, 0, len(items))
	for index, item := range items {
		if _, err := systemOneBody(item.Request); err != nil {
			responses[index] = decideError(responses[index], inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, err.Error(), "questions")
			continue
		}
		walk = append(walk, index)
	}
	batch := decisionBatch{items: items, responses: responses, walk: walk}
	for _, candidate := range candidates {
		r.decideCandidate(ctx, definition, template, candidate, classify, &batch)
	}
	for _, index := range walk {
		responses[index] = finishDecision(items[index].Request, responses[index])
	}
	return responses
}

// decisionBatch is the walk's state: the items, their responses, and which
// items are walking (an item refused before the walk never gets rows).
type decisionBatch struct {
	items     []DecisionItem
	responses []*inferencev1.DecideResponse
	walk      []int
}

// open adds candidate's row to every walking item and returns the items still
// unanswered with their rows. An answered item records the candidate as not
// attempted.
func (b *decisionBatch) open(candidate rolecatalog.Candidate) ([]int, []*inferencev1.DecisionCandidate) {
	var pending []int
	var rows []*inferencev1.DecisionCandidate
	for _, index := range b.walk {
		row := &inferencev1.DecisionCandidate{Provider: candidate.Provider, ResourceRole: candidate.ResourceRole}
		response := b.responses[index]
		response.Candidates = append(response.Candidates, row)
		if response.Method != inferencev1.DecisionMethod_DECISION_METHOD_UNSPECIFIED {
			row.Status = inferencev1.DecisionCandidateStatus_DECISION_CANDIDATE_STATUS_NOT_ATTEMPTED
			continue
		}
		pending, rows = append(pending, index), append(rows, row)
	}
	return pending, rows
}

func skipAll(rows []*inferencev1.DecisionCandidate, reason, detail string) {
	for _, row := range rows {
		skip(row, reason, detail)
	}
}

func (r *ResourceRepository) decideCandidate(ctx context.Context, definition rolecatalog.InferenceRole, template DecisionRequest, candidate rolecatalog.Candidate, classify TypedRunner, batch *decisionBatch) {
	pending, rows := batch.open(candidate)
	if len(pending) == 0 {
		return
	}
	if strings.HasPrefix(strings.TrimSpace(template.Role), "browser.") && !browserCandidateMatchesSource(providers.Source(ctx), candidate.Provider, "") {
		skipAll(rows, reasonSourceNotSelected, "candidate is outside the caller-selected AI source")
		return
	}
	if candidate.Provider == rolecatalog.ProviderGateway {
		// The enum fallback is a typed run per state through the gateway's own
		// walk; it has no batch form, so states run with the batch's bound.
		slots := make(chan struct{}, MaxBatchConcurrency)
		var wait sync.WaitGroup
		for position, index := range pending {
			wait.Add(1)
			slots <- struct{}{}
			go func(row *inferencev1.DecisionCandidate, item DecisionItem, response *inferencev1.DecideResponse) {
				defer wait.Done()
				defer func() { <-slots }()
				r.decideByEnum(item.Ctx, item.Request, candidate, classify, row, response)
			}(rows[position], batch.items[index], batch.responses[index])
		}
		wait.Wait()
		return
	}
	r.decideByModel(ctx, definition, template, candidate, pending, rows, batch)
}

// decideByModel qualifies a decision-model candidate once for the batch (the
// checks depend on the role and question set, never on a state), then asks it
// about every pending state in one resource command.
func (r *ResourceRepository) decideByModel(ctx context.Context, definition rolecatalog.InferenceRole, template DecisionRequest, candidate rolecatalog.Candidate, pending []int, rows []*inferencev1.DecisionCandidate, batch *decisionBatch) {
	adapter, ok := r.adapters[strings.ToLower(strings.TrimSpace(candidate.Provider))]
	if !ok {
		skipAll(rows, providers.CodeUnconfigured, "adapter not configured")
		return
	}
	resolved, err := adapter.ResolveRole(ctx, candidate.ResourceRole)
	if err != nil {
		skipAll(rows, candidateReason(err), err.Error())
		return
	}
	for _, row := range rows {
		row.Locality = resolved.Locality
	}
	if strings.HasPrefix(strings.TrimSpace(template.Role), "browser.") && !browserCandidateMatchesSource(providers.Source(ctx), candidate.Provider, resolved.Locality) {
		skipAll(rows, reasonSourceNotSelected, "resolved candidate is outside the caller-selected AI source")
		return
	}
	providerRequest := ProviderRequest{Profile: template.Profile, PrivacyClass: template.PrivacyClass}
	if !r.admitsRequest(ctx, providerRequest, candidate.Provider, resolved.Locality) {
		skipAll(rows, reasonLocalityForbidden, fmt.Sprintf("profile %s forbids a %s candidate", template.Profile, resolved.Locality))
		return
	}
	// The model's declared bounds are data the resource owns; a request outside
	// them is skipped here instead of being dispatched to be refused.
	if violation := limitsViolation(resolved.Limits, template.Questions); violation != "" {
		skipAll(rows, providers.CodeLimitsExceeded, violation)
		return
	}
	executionRole, expectedArtifactDigest := resolved, ""
	if localModelRequiresRights(candidate.Provider, resolved) {
		executionRole, expectedArtifactDigest, _, err = r.authorizeLocalCandidate(ctx, adapter, candidate.ResourceRole, resolved)
		if err != nil {
			skipAll(rows, providers.CodeUnconfigured, "local model rights or exact runtime identity could not be qualified: "+err.Error())
			return
		}
	}
	states := make([]json.RawMessage, len(pending))
	for position, index := range pending {
		states[position] = systemOneState(batch.items[index].Request)
	}
	started := time.Now()
	results, err := adapter.ExecuteDecisionBatch(ctx, providers.ExecutionRequest{
		Kind:                         sharedv1.RequestKind_REQUEST_KIND_DECISION,
		Role:                         candidate.ResourceRole,
		ExpectedModel:                executionRole.Model,
		ExpectedArtifactDigest:       expectedArtifactDigest,
		ExpectedRightsDecisionSHA256: executionRole.RightsDecisionSHA256,
		ExpectedRightsEvaluatedAt:    executionRole.RightsEvaluatedAt,
		ExpectedRightsReceiptSHA256:  executionRole.RightsReceiptSHA256,
		ExpectedPolicyFileSHA256:     executionRole.PolicyFileSHA256,
		Profile:                      template.Profile,
		Timeout:                      definition.Timeout(),
	}, systemOneQuestions(template.Questions), states)
	took := time.Since(started)
	// Each state's route event carries its share of the one command and, when
	// the resource reported it, the state's own round trip to the model.
	share := took / time.Duration(len(pending))
	for position, index := range pending {
		item, row := batch.items[index], rows[position]
		if err != nil {
			traceAttempt(item.Ctx, adapter.Provider, candidate.ResourceRole, resolved, share, providers.ProviderTiming{})
			row.LatencyMs = took.Milliseconds()
			fail(row, candidateReason(err), err.Error())
			continue
		}
		result := results[position]
		traceAttempt(item.Ctx, adapter.Provider, candidate.ResourceRole, resolved, share, providers.ProviderTiming{RuntimeNS: result.LatencyMS * int64(time.Millisecond)})
		row.LatencyMs = result.LatencyMS
		if result.Err != nil {
			fail(row, candidateReason(result.Err), result.Err.Error())
			continue
		}
		r.acceptDecisionOutput(item.Request, candidate, resolved, row, batch.responses[index], result.OutputText)
	}
}

// walkCandidates is the candidates a request may be answered by. An ordinary
// request walks the role's declared candidates in profile order. A pinned
// request walks exactly the one it names, so nothing falls back: the pin must
// be a candidate the role declares, because the catalog, not the caller, owns
// which models a role may reach.
func walkCandidates(definition rolecatalog.InferenceRole, request DecisionRequest, adapters map[string]providers.Adapter) ([]rolecatalog.Candidate, error) {
	pin := strings.TrimSpace(request.PinResourceRole)
	if pin == "" {
		return orderedCandidates(definition.Candidates, adapters, request.Profile), nil
	}
	declared := make([]string, 0, len(definition.Candidates))
	for _, candidate := range definition.Candidates {
		if candidate.ResourceRole == pin {
			return []rolecatalog.Candidate{candidate}, nil
		}
		declared = append(declared, candidate.ResourceRole)
	}
	return nil, fmt.Errorf("pin_resource_role %q is not a candidate of role %q (declared: %s)", pin, request.Role, strings.Join(declared, ", "))
}

func finishDecision(request DecisionRequest, response *inferencev1.DecideResponse) *inferencev1.DecideResponse {
	if response.Method != inferencev1.DecisionMethod_DECISION_METHOD_UNSPECIFIED {
		finalizeUsage(response.Usage)
		return response
	}
	code := terminalDecisionCode(response.Candidates)
	message := fmt.Sprintf("no decision candidate answered for role %q (%s)", request.Role, summarizeCandidates(response.Candidates))
	construct := ""
	if pin := strings.TrimSpace(request.PinResourceRole); pin != "" {
		message = fmt.Sprintf("pinned candidate %q of role %q did not answer and no other candidate was tried (%s)", pin, request.Role, summarizeCandidates(response.Candidates))
		construct = constructPinnedRole
	}
	if strings.HasPrefix(strings.TrimSpace(request.Role), "browser.") && browserNoRouteOnly(response.Candidates) {
		code = inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_NO_ELIGIBLE_ROUTE
		construct = "source_or_route_eligibility"
		message = "no eligible route is configured for this browser source and profile; select an available source or ask the account owner to enable or upgrade an eligible route. No model inference was dispatched and funding was not changed"
	}
	return decideError(response, code, message, construct)
}

func (r *ResourceRepository) acceptDecisionOutput(request DecisionRequest, candidate rolecatalog.Candidate, resolved providers.ResolvedRole, row *inferencev1.DecisionCandidate, response *inferencev1.DecideResponse, outputText string) {
	var output decisionOutput
	if err := json.Unmarshal([]byte(outputText), &output); err != nil {
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

func (r *ResourceRepository) decideByEnum(ctx context.Context, request DecisionRequest, candidate rolecatalog.Candidate, classify TypedRunner, row *inferencev1.DecisionCandidate, response *inferencev1.DecideResponse) {
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
		if failure != nil && failure.GetCode() == inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_NO_ELIGIBLE_ROUTE {
			fail(row, reasonNoEligibleRoute, strings.TrimSpace(failure.GetCode().String()+": "+failure.GetMessage()))
			return
		}
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

// limitsViolation reports why a question set is outside a model's declared
// bounds, or "" when it fits or the model declares none. Only counts the
// gateway can know exactly are checked; the state's token length is the
// model's own refusal (context_overflow), because the gateway has no tokenizer
// and an estimate would skip candidates that could have answered.
func limitsViolation(limits *providers.DecisionLimits, questions []*inferencev1.DecisionQuestion) string {
	if limits == nil {
		return ""
	}
	if limits.MaxQuestions > 0 && len(questions) > int(limits.MaxQuestions) {
		return fmt.Sprintf("request has %d questions; the model accepts %d", len(questions), limits.MaxQuestions)
	}
	for _, question := range questions {
		if limits.MaxOptions > 0 && len(question.GetOptions()) > int(limits.MaxOptions) {
			return fmt.Sprintf("question %q has %d options; the model accepts %d", question.GetId(), len(question.GetOptions()), limits.MaxOptions)
		}
		if limits.ScoreLevels > 0 && len(question.GetLevels()) > 0 && len(question.GetLevels()) != int(limits.ScoreLevels) {
			return fmt.Sprintf("question %q has %d score levels; the model serves exactly %d", question.GetId(), len(question.GetLevels()), limits.ScoreLevels)
		}
	}
	return ""
}

func browserCandidateMatchesSource(source, provider, locality string) bool {
	source = strings.TrimSpace(source)
	provider = strings.ToLower(strings.TrimSpace(provider))
	if source == "" {
		return true
	}
	switch source {
	case "local":
		return browserLocalProvider(provider, locality)
	case "own-key":
		return provider == providers.ProviderOpenRouter || provider == providers.ProviderOrcaRouter || provider == rolecatalog.ProviderGateway
	case "vrooli-credits":
		return provider == providers.ProviderMetered || provider == rolecatalog.ProviderGateway
	default:
		return false
	}
}

func browserLocalProvider(provider, locality string) bool {
	if provider == providers.ProviderOllama || provider == rolecatalog.ProviderGateway {
		return true
	}
	return provider == providers.ProviderOllaya && (locality == "" || strings.EqualFold(strings.TrimSpace(locality), "local"))
}

func browserNoRouteOnly(rows []*inferencev1.DecisionCandidate) bool {
	if len(rows) == 0 {
		return true
	}
	for _, row := range rows {
		if row.GetStatus() == inferencev1.DecisionCandidateStatus_DECISION_CANDIDATE_STATUS_SKIPPED && row.GetReason() == reasonSourceNotSelected {
			continue
		}
		if row.GetStatus() == inferencev1.DecisionCandidateStatus_DECISION_CANDIDATE_STATUS_FAILED && row.GetReason() == reasonNoEligibleRoute {
			continue
		}
		return false
	}
	return true
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

func profileAdmitsRequest(profile sharedv1.Profile, privacyClass sharedv1.PrivacyClass, locality string) bool {
	allowed, _ := profilepolicy.Admits(profile.String(), privacyClass.String(), locality)
	return allowed
}
