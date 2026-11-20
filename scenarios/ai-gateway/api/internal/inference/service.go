package inference

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"

	"ai-gateway/internal/providers"

	profilepolicy "github.com/vrooli/ai-provider-go/routing"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"google.golang.org/protobuf/proto"
)

var ErrUnavailable = errors.New("typed inference is unavailable")
var ErrNoEligibleRoute = errors.New("no eligible source route exists before provider dispatch")
var ErrOutcomeUncertain = errors.New("provider outcome is uncertain; do not dispatch a fallback")

const (
	MaxBatchSize = 256

	// MaxBatchConcurrency bounds in-flight provider executions for one batch.
	// It is deliberately modest: the first candidate for every role is a local
	// model whose throughput degrades rather than scales under parallel load.
	MaxBatchConcurrency = 4

	// MaxValidationAttempts is the total number of provider calls one request
	// may make. A second attempt carries the local validator's rejection so the
	// provider can repair its own output. Further attempts rarely converge and
	// would make cost unpredictable for callers.
	MaxValidationAttempts = 2
	// MaxProviderCalls is the largest generic request-level provider execution
	// ceiling accepted by typed inference. Callers may choose a lower bound.
	MaxProviderCalls = 8
)

type Service struct {
	repository       Repository
	gate             SchemaGate
	codeRoleContract func(string) (CodeRoleContract, bool)
}

func NewService(repository Repository) *Service {
	if repository == nil {
		repository = StaticRepository{}
	}
	service := &Service{repository: repository}
	if lookup, ok := repository.(interface {
		LookupCodeRole(string) (CodeRoleContract, bool)
	}); ok {
		service.codeRoleContract = lookup.LookupCodeRole
	}
	return service
}

func (s *Service) Run(ctx context.Context, request ProviderRequest) *inferencev1.RunResponse {
	response := &inferencev1.RunResponse{Usage: &inferencev1.Usage{}}
	if request.RequestDigest == "" {
		request.RequestDigest = canonicalRequestDigest(request)
	}
	schema, err := s.gate.Parse(request.SchemaJSON)
	if err != nil {
		return withError(response, schemaErrorCode(err), err.Error(), schemaConstruct(err))
	}
	return s.runWithSchema(ctx, request, schema, response)
}

func (s *Service) runWithSchema(ctx context.Context, request ProviderRequest, schema map[string]any, response *inferencev1.RunResponse) *inferencev1.RunResponse {
	if err := validatePrivacyProvenance(request.PrivacyProvenance); err != nil {
		return withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, err.Error(), "privacy_provenance")
	}
	if request.MaxProviderCalls < 0 || request.MaxProviderCalls > MaxProviderCalls {
		return withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST,
			fmt.Sprintf("max_provider_calls must be between 0 and %d", MaxProviderCalls), "max_provider_calls")
	}
	if strings.TrimSpace(request.Source) == "" && len(request.Turns) == 0 && len(request.Attachments) == 0 {
		return withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "source is required", "source")
	}
	if strings.TrimSpace(request.Role) == "" {
		return withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "role is required", "role")
	}
	if strings.HasPrefix(strings.TrimSpace(request.Role), "browser.") && !validRootOperationID(request.RootOperationID) {
		return withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "browser inference requires a caller-generated root_operation_id for exact reservation reconciliation", "root_operation_id")
	}
	if strings.HasPrefix(strings.TrimSpace(request.Role), "browser.") && !validBrowserSessionBinding(request.BrowserSession) {
		return withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "browser inference requires its authenticated run, grant and BAS session binding", "browser_session")
	}
	if s.codeRoleContract != nil {
		contract, declared := s.codeRoleContract(request.Role)
		if declared {
			value, err := evaluateDeclaredCodeRole(contract, schema, request.Source)
			if err != nil {
				if errors.Is(err, ErrCodeRoleCallerSchema) {
					return withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED, err.Error(), "schema_json")
				}
				return withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, err.Error(), "source")
			}
			response.ValueJson = value
			response.Provider = "vrooli-code"
			response.Model = contract.HandlerID
			response.Validated = true
			finalizeUsage(response.Usage)
			return response
		}
	}
	var validationErr error
	for attempt := 0; attempt < MaxValidationAttempts; attempt++ {
		remaining := request.MaxProviderCalls
		if remaining > 0 {
			remaining -= response.ProviderCallsMade
			if remaining <= 0 {
				break
			}
		}
		attemptRequest := request
		attemptRequest.ValidationAttempt = attempt
		attemptRequest.MaxProviderCalls = remaining
		if validationErr != nil {
			attemptRequest.Instruction = repairInstruction(request.Instruction, validationErr)
		}
		result, err := s.repository.Run(ctx, attemptRequest)
		response.ProviderCallsMade += result.ProviderCallsMade
		if result.RightsContextSHA256 != "" {
			request.PriorRightsProvider = result.Provider
			request.PriorRightsModel = result.Model
			request.PriorRightsContextSHA256 = result.RightsContextSHA256
		}
		if result.Recovery != nil {
			response.Recovery = result.Recovery
			response.ValueJson = ""
			response.Provider = ""
			response.Model = ""
			response.Validated = false
			response.Applied = nil
			response.SpendAuthorization = nil
			response.Usage = &inferencev1.Usage{
				AccountingStatus: result.Recovery.GetAccountingStatus(),
			}
			if result.Recovery.ChargedCredits != nil {
				response.Usage.CreditsCharged = proto.Int64(result.Recovery.GetChargedCredits())
			}
			response.Error = &inferencev1.InferenceError{
				Code:    inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_OUTCOME_UNCERTAIN,
				Message: "hosted inference output is unavailable; reuse root_operation_id to resolve its accounting receipt",
			}
			return response
		}
		response.Provider, response.Model = result.Provider, result.Model
		// Applied is set before the error branch: a refused or truncated call is
		// exactly when a caller needs to know what the gateway would have sent.
		response.Applied = appliedSettings(result.Applied)
		response.SpendAuthorization = result.SpendAuthorization
		// Usage accumulates across attempts so a caller is billed for the work
		// actually performed, not only for the attempt that happened to land.
		addUsage(response.Usage, usage(result))
		if err != nil {
			// Provider failures are not retried here: the repository already
			// walks every declared candidate before reporting one.
			code, construct := samplingErrorCode(err)
			message := err.Error()
			if errors.Is(err, ErrOutcomeUncertain) {
				code = inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_OUTCOME_UNCERTAIN
				construct = "root_operation_id"
				message = fmt.Sprintf("provider outcome is uncertain; reuse root_operation_id %q for reconciliation and do not switch funding source", request.RootOperationID)
			} else if errors.Is(err, ErrNoEligibleRoute) {
				code = inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_NO_ELIGIBLE_ROUTE
				construct = "source_or_route_eligibility"
				message = "no eligible route is configured for this browser source and profile; select an available source or ask the account owner to enable an eligible route. No model inference was dispatched and funding was not changed"
			}
			if errors.Is(err, ErrRootConflict) {
				code, construct = inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "root_operation_id"
			}
			if errors.Is(err, ErrRootRequired) {
				code, construct = inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "root_operation_id"
			}
			return withError(response, code, message, construct)
		}
		response.ValueJson = result.ValueJSON
		response.BillingOperationId = result.BillingOperationID
		if IsVisualLocateRole(request.Role) {
			value, err := NormalizeLocateVisualJSON(result.ValueJSON, result.CoordinateConvention, requestAttachments(request))
			if err != nil {
				return withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED, err.Error(), "bounds")
			}
			response.ValueJson = value
		}
		if err := ValidateJSON(schema, []byte(response.ValueJson)); err != nil {
			validationErr = err
			continue
		}
		response.Validated = true
		finalizeUsage(response.Usage)
		return response
	}
	return withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED,
		fmt.Sprintf("provider value failed local validation within the provider-call budget: %v", validationErr), "")
}

func validBrowserSessionBinding(binding *inferencev1.BrowserSessionBinding) bool {
	if binding == nil || strings.TrimSpace(binding.RunId) == "" || strings.TrimSpace(binding.SessionId) == "" || strings.TrimSpace(binding.GrantReceipt) == "" {
		return false
	}
	if strings.TrimSpace(binding.RunId) != binding.RunId || strings.TrimSpace(binding.SessionId) != binding.SessionId || strings.TrimSpace(binding.GrantReceipt) != binding.GrantReceipt {
		return false
	}
	if len(binding.RunId) > 64 || len(binding.SessionId) > 128 || len(binding.GrantReceipt) > 512 || strings.ContainsAny(binding.RunId+binding.SessionId+binding.GrantReceipt, "\r\n") {
		return false
	}
	return true
}

// repairInstruction re-asks with the local validator's rejection attached. The
// schema is unchanged; only the caller-intent instruction grows, which keeps
// the schema gate the single description of what is enforceable.
func repairInstruction(original string, validationErr error) string {
	instruction := strings.TrimSpace(original)
	if instruction == "" {
		instruction = "Produce the value described by the schema for the supplied source."
	}
	return instruction +
		"\n\nA previous response was rejected by local schema validation: " + validationErr.Error() +
		"\nReturn one corrected JSON value that satisfies the schema exactly."
}

func (s *Service) RunBatch(ctx context.Context, requests []ProviderRequest) *inferencev1.RunBatchResponse {
	response := &inferencev1.RunBatchResponse{Usage: &inferencev1.Usage{}}
	if len(requests) == 0 {
		return response
	}
	if len(requests) > MaxBatchSize {
		response.Results = []*inferencev1.RunResponse{{Usage: &inferencev1.Usage{}, Error: &inferencev1.InferenceError{
			Code:    inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST,
			Message: fmt.Sprintf("batch contains %d items; maximum is %d", len(requests), MaxBatchSize), Construct: "items",
		}}}
		return response
	}
	schema, err := s.gate.Parse(requests[0].SchemaJSON)
	if err != nil {
		failure := &inferencev1.RunResponse{Usage: &inferencev1.Usage{}}
		failure.Error = &inferencev1.InferenceError{Code: schemaErrorCode(err), Message: err.Error(), Construct: schemaConstruct(err)}
		response.Results = []*inferencev1.RunResponse{failure}
		return response
	}
	// Results are placed by index so ordering is independent of completion
	// order, which the proto guarantees. schema is read-only from here on and
	// is therefore safe to share across workers.
	results := make([]*inferencev1.RunResponse, len(requests))
	slots := make(chan struct{}, MaxBatchConcurrency)
	var wait sync.WaitGroup
	for index, request := range requests {
		wait.Add(1)
		go func(index int, request ProviderRequest) {
			defer wait.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			results[index] = s.runWithSchema(ctx, request, schema, &inferencev1.RunResponse{Usage: &inferencev1.Usage{}})
		}(index, request)
	}
	wait.Wait()
	for _, result := range results {
		addUsage(response.Usage, result.Usage)
	}
	response.Usage.CreditsCharged = aggregateCredits(results)
	if len(results) != 1 {
		response.Usage.AccountingStatus = ""
	}
	response.Results = results
	return response
}

// Decide answers one decision request. Validation happens here, once, so every
// candidate sees the same well-formed request; the repository owns the walk and
// reaches the typed classify path through s.Run for its enum fallback.
func (s *Service) Decide(ctx context.Context, request DecisionRequest) *inferencev1.DecideResponse {
	if err := validatePrivacyProvenance(request.PrivacyProvenance); err != nil {
		return decideError(&inferencev1.DecideResponse{Usage: &inferencev1.Usage{}}, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, err.Error(), "privacy_provenance")
	}
	if err := ValidateQuestions(request.Questions); err != nil {
		return decisionRefusal(err)
	}
	return s.decideValidated(ctx, request)
}

func (s *Service) decideValidated(ctx context.Context, request DecisionRequest) *inferencev1.DecideResponse {
	if strings.TrimSpace(request.Role) == "" {
		return decideError(&inferencev1.DecideResponse{Usage: &inferencev1.Usage{}}, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "role is required", "role")
	}
	if strings.HasPrefix(strings.TrimSpace(request.Role), "browser.") {
		// DecideRequest currently has no Gateway-verified AM/BAS browser binding.
		// Do not make a browser decision dispatch based on user-account auth or
		// infer authority from question/state fields. Keep the catalog route for
		// source-policy inspection, but fail closed until an owner supplies a
		// verified run/grant/session context.
		return decideError(&inferencev1.DecideResponse{Usage: &inferencev1.Usage{}}, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_NO_ELIGIBLE_ROUTE,
			"browser decision has no supported verified run, grant and BAS session binding path; no inference was dispatched", "browser_session")
	}
	if err := ValidateState(request.State, request.StateJSON); err != nil {
		return decisionRefusal(err)
	}
	repository, ok := s.repository.(DecisionRepository)
	if !ok {
		return decideError(&inferencev1.DecideResponse{Usage: &inferencev1.Usage{}}, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, "decision repository is unavailable", "decision")
	}
	return repository.Decide(ctx, request, s.Run)
}

// DecideBatch asks one question set about many states. Results are placed by
// index, so ordering is independent of completion order; a bad item fails
// alone while a bad question set fails the batch once.
func (s *Service) DecideBatch(ctx context.Context, template DecisionRequest, items []DecisionRequest) *inferencev1.DecideBatchResponse {
	response := &inferencev1.DecideBatchResponse{Usage: &inferencev1.Usage{}}
	if err := validatePrivacyProvenance(template.PrivacyProvenance); err != nil {
		response.Results = []*inferencev1.DecideResponse{decideError(&inferencev1.DecideResponse{Usage: &inferencev1.Usage{}}, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, err.Error(), "privacy_provenance")}
		return response
	}
	if len(items) == 0 {
		return response
	}
	if len(items) > MaxBatchSize {
		response.Results = []*inferencev1.DecideResponse{decideError(&inferencev1.DecideResponse{Usage: &inferencev1.Usage{}},
			inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, fmt.Sprintf("batch contains %d items; maximum is %d", len(items), MaxBatchSize), "items")}
		return response
	}
	if err := ValidateQuestions(template.Questions); err != nil {
		response.Results = []*inferencev1.DecideResponse{decisionRefusal(err)}
		return response
	}
	results := make([]*inferencev1.DecideResponse, len(items))
	slots := make(chan struct{}, MaxBatchConcurrency)
	var wait sync.WaitGroup
	for index, item := range items {
		wait.Add(1)
		go func(index int, item DecisionRequest) {
			defer wait.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			request := template
			request.State, request.StateJSON = item.State, item.StateJSON
			results[index] = s.decideValidated(ctx, request)
		}(index, item)
	}
	wait.Wait()
	for _, result := range results {
		addUsage(response.Usage, result.GetUsage())
	}
	finalizeUsage(response.Usage)
	response.Results = results
	return response
}

func validatePrivacyProvenance(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("privacy provenance is required and must name the non-sensitive source and basis for the caller's classification")
	}
	if !profilepolicy.ValidPrivacyProvenance(value) {
		return errors.New("privacy provenance must be a short non-sensitive identifier containing only letters, digits, '.', '_', '-', ':' or '/'")
	}
	return nil
}

// validateEmbeddingPrivacy requires every embedding request to carry the
// caller's recognized privacy class and non-sensitive source basis. Embed has
// no remote route, even for PUBLIC data.
func validateEmbeddingPrivacy(class sharedv1.PrivacyClass, provenance string) error {
	switch class {
	case sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC,
		sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET:
		return validatePrivacyProvenance(provenance)
	case sharedv1.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED:
		return errors.New("privacy class is required for typed embedding")
	default:
		return errors.New("embedding privacy class is unknown")
	}
}

func decisionRefusal(err error) *inferencev1.DecideResponse {
	construct := ""
	var decisionErr *DecisionError
	if errors.As(err, &decisionErr) {
		construct = decisionErr.Construct
	}
	return decideError(&inferencev1.DecideResponse{Usage: &inferencev1.Usage{}}, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, err.Error(), construct)
}

func (s *Service) Embed(ctx context.Context, request EmbeddingRequest) *inferencev1.EmbedResponse {
	response := &inferencev1.EmbedResponse{Usage: &inferencev1.Usage{}}
	if request.Sampling {
		return embedError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "embedding roles are deterministic and reject sampling controls", "sampling")
	}
	if strings.TrimSpace(request.Role) == "" {
		return embedError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "role is required", "role")
	}
	if len(request.Texts) == 0 {
		return embedError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "texts are required", "texts")
	}
	if request.Profile != sharedv1.Profile_PROFILE_LOCAL_ONLY {
		return embedError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "typed embeddings require an explicit local-only profile", "profile")
	}
	if err := validateEmbeddingPrivacy(request.PrivacyClass, request.PrivacyProvenance); err != nil {
		return embedError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, err.Error(), "privacy_provenance")
	}
	repository, ok := s.repository.(EmbeddingRepository)
	if !ok {
		return embedError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, "embedding repository is unavailable", "embedding")
	}
	result, err := repository.Embed(ctx, request)
	if err != nil {
		code := inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_PROVIDER_FAILED
		if errors.Is(err, ErrUnavailable) {
			code = inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE
		}
		if strings.Contains(err.Error(), "dimension_mismatch") {
			code = inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED
		}
		return embedError(response, code, err.Error(), "embedding")
	}
	if result.Dimension <= 0 || len(result.Vectors) != len(request.Texts) {
		return embedError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED, "embedding repository returned an incomplete vector batch", "vectors")
	}
	response.Provider, response.Model, response.Dimension = result.Provider, result.Model, int32(result.Dimension)
	response.Usage.InputTokens, response.Usage.CostMicros = result.InputTokens, result.CostMicros
	response.Usage.InputTokensProvenance = nonUnknownProvenance(result.InputTokensProvenance)
	response.Usage.OutputTokensProvenance = inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
	response.Usage.CostProvenance = nonUnknownProvenance(result.CostProvenance)
	for _, vector := range result.Vectors {
		if len(vector) != result.Dimension {
			return embedError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED, "embedding_dimension_mismatch", "dimension")
		}
		response.Vectors = append(response.Vectors, &inferencev1.EmbeddingVector{Values: vector})
	}
	return response
}

func embedError(response *inferencev1.EmbedResponse, code inferencev1.InferenceErrorCode, message, construct string) *inferencev1.EmbedResponse {
	finalizeUsage(response.Usage)
	response.Error = &inferencev1.InferenceError{Code: code, Message: message, Construct: construct}
	return response
}

func addUsage(total, item *inferencev1.Usage) {
	if total == nil || item == nil {
		return
	}
	total.InputTokens += item.InputTokens
	total.OutputTokens += item.OutputTokens
	total.CostMicros += item.CostMicros
	total.InputTokensProvenance = combineProvenance(total.InputTokensProvenance, item.InputTokensProvenance)
	total.OutputTokensProvenance = combineProvenance(total.OutputTokensProvenance, item.OutputTokensProvenance)
	total.CostProvenance = combineProvenance(total.CostProvenance, item.CostProvenance)
	if item.CreditsCharged == nil {
		total.CreditsCharged = nil
	} else {
		charged := *item.CreditsCharged
		total.CreditsCharged = &charged
	}
	total.AccountingStatus = item.AccountingStatus
}

func aggregateCredits(results []*inferencev1.RunResponse) *int64 {
	if len(results) == 0 {
		return nil
	}
	var total int64
	for _, result := range results {
		if result == nil || result.Usage == nil || result.Usage.CreditsCharged == nil || *result.Usage.CreditsCharged < 0 || math.MaxInt64-total < *result.Usage.CreditsCharged {
			return nil
		}
		total += *result.Usage.CreditsCharged
	}
	return &total
}

func usage(result ProviderResult) *inferencev1.Usage {
	value := &inferencev1.Usage{
		InputTokens: result.InputTokens, OutputTokens: result.OutputTokens, CostMicros: result.CostMicros,
		InputTokensProvenance:  nonUnknownProvenance(result.InputTokensProvenance),
		OutputTokensProvenance: nonUnknownProvenance(result.OutputTokensProvenance),
		CostProvenance:         nonUnknownProvenance(result.CostProvenance),
		AccountingStatus:       result.AccountingStatus,
	}
	if result.CreditsCharged != nil && *result.CreditsCharged >= 0 {
		charged := *result.CreditsCharged
		value.CreditsCharged = &charged
	}
	return value
}

func withError(response *inferencev1.RunResponse, code inferencev1.InferenceErrorCode, message, construct string) *inferencev1.RunResponse {
	response.Validated = false
	finalizeUsage(response.Usage)
	response.Error = &inferencev1.InferenceError{Code: code, Message: message, Construct: construct}
	return response
}

func nonUnknownProvenance(value inferencev1.UsageProvenance) inferencev1.UsageProvenance {
	if value == inferencev1.UsageProvenance_USAGE_PROVENANCE_UNSPECIFIED {
		return inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
	}
	return value
}

func combineProvenance(current, next inferencev1.UsageProvenance) inferencev1.UsageProvenance {
	if current == inferencev1.UsageProvenance_USAGE_PROVENANCE_UNSPECIFIED {
		return nonUnknownProvenance(next)
	}
	next = nonUnknownProvenance(next)
	if current == inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN || next == inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN {
		return inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
	}
	if current == next {
		return current
	}
	return inferencev1.UsageProvenance_USAGE_PROVENANCE_MIXED
}

func finalizeUsage(value *inferencev1.Usage) {
	if value == nil {
		return
	}
	value.InputTokensProvenance = nonUnknownProvenance(value.InputTokensProvenance)
	value.OutputTokensProvenance = nonUnknownProvenance(value.OutputTokensProvenance)
	value.CostProvenance = nonUnknownProvenance(value.CostProvenance)
}

func schemaErrorCode(err error) inferencev1.InferenceErrorCode {
	var schemaErr *SchemaError
	if errors.As(err, &schemaErr) && schemaErr.Construct != "" && schemaErr.Construct != "schema_json" && schemaErr.Construct != "schema" && schemaErr.Construct != "depth" {
		return inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNSUPPORTED_SCHEMA
	}
	return inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST
}

func schemaConstruct(err error) string {
	var schemaErr *SchemaError
	if errors.As(err, &schemaErr) {
		return schemaErr.Construct
	}
	return ""
}

// samplingConstruct names the offending control on a sampling refusal, matching
// how UNSUPPORTED_SCHEMA names the offending keyword.
const samplingConstruct = "sampling.temperature"

// samplingErrorCode separates the two sampling failures, which look alike and
// are not. A role that forbids overrides is a request defect the caller can fix
// by not sending one; a provider that cannot honour the control is incapacity
// no request change would repair. Collapsing them would tell the caller to fix
// the wrong thing.
func samplingErrorCode(err error) (inferencev1.InferenceErrorCode, string) {
	switch {
	case errors.Is(err, ErrRoleForbidsSampling):
		return inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, samplingConstruct
	case errors.Is(err, ErrUnsupportedSampling):
		return inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNSUPPORTED_SAMPLING, samplingConstruct
	case errors.Is(err, ErrContextOverflow):
		return inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_CONTEXT_OVERFLOW, "context"
	case errors.Is(err, ErrUnavailable):
		return inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, ""
	default:
		return inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_PROVIDER_FAILED, ""
	}
}

// appliedSettings projects the repository's account onto the wire. A support
// state the gateway never resolved stays UNSPECIFIED rather than being reported
// as a declared "unknown" — the two mean different things to a caller comparing
// candidate sets.
func appliedSettings(applied AppliedSettings) *sharedv1.AppliedSettings {
	out := &sharedv1.AppliedSettings{
		TemperatureSupport:       samplingSupportProto(applied.TemperatureSupport),
		MaxOutputTokensEffective: applied.MaxOutputTokens,
		MaxOutputTokensSource:    outputCapSourceProto(applied.MaxOutputTokensSource),
	}
	if applied.TemperatureSent != nil {
		out.TemperatureSent = proto.Float64(*applied.TemperatureSent)
	}
	return out
}

func samplingSupportProto(support providers.SamplingSupport) sharedv1.SamplingSupport {
	switch support {
	case providers.SamplingHonored:
		return sharedv1.SamplingSupport_SAMPLING_SUPPORT_HONORED
	case providers.SamplingIgnored:
		return sharedv1.SamplingSupport_SAMPLING_SUPPORT_IGNORED
	case providers.SamplingRejected:
		return sharedv1.SamplingSupport_SAMPLING_SUPPORT_REJECTED
	case providers.SamplingUnknown:
		return sharedv1.SamplingSupport_SAMPLING_SUPPORT_UNKNOWN
	default:
		return sharedv1.SamplingSupport_SAMPLING_SUPPORT_UNSPECIFIED
	}
}

func outputCapSourceProto(source OutputCapSource) sharedv1.OutputCapSource {
	switch source {
	case OutputCapRequest:
		return sharedv1.OutputCapSource_OUTPUT_CAP_SOURCE_REQUEST
	case OutputCapRolePolicy:
		return sharedv1.OutputCapSource_OUTPUT_CAP_SOURCE_ROLE_POLICY
	case OutputCapNoneImposed:
		return sharedv1.OutputCapSource_OUTPUT_CAP_SOURCE_NONE_IMPOSED
	default:
		return sharedv1.OutputCapSource_OUTPUT_CAP_SOURCE_UNSPECIFIED
	}
}
