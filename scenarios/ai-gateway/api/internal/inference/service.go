package inference

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"ai-gateway/internal/rolecatalog"
	"ai-gateway/internal/schemagate"

	profilepolicy "github.com/vrooli/ai-provider-go/routing"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"google.golang.org/protobuf/proto"
)

var (
	ErrUnavailable      = errors.New("typed inference is unavailable")
	ErrNoEligibleRoute  = errors.New("no eligible source route exists before provider dispatch")
	ErrOutcomeUncertain = errors.New("provider outcome is uncertain; do not dispatch a fallback")
)

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
	gate             schemagate.SchemaGate
	codeRoleContract func(string) (rolecatalog.CodeRoleContract, bool)
}

func NewService(repository Repository) *Service {
	if repository == nil {
		repository = StaticRepository{}
	}
	service := &Service{repository: repository}
	if lookup, ok := repository.(interface {
		LookupCodeRole(string) (rolecatalog.CodeRoleContract, bool)
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
	if validation := validateRunRequest(request); validation != nil {
		return withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, validation.message, validation.construct)
	}
	if handled := s.runCodeRole(request, schema, response); handled {
		return response
	}
	return s.runProviderAttempts(ctx, request, schema, response)
}

func (s *Service) runCodeRole(request ProviderRequest, schema map[string]any, response *inferencev1.RunResponse) bool {
	if s.codeRoleContract != nil {
		contract, declared := s.codeRoleContract(request.Role)
		if declared {
			value, err := evaluateDeclaredCodeRole(contract, schema, request.Source)
			if err != nil {
				if errors.Is(err, ErrCodeRoleCallerSchema) {
					withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED, err.Error(), "schema_json")
					return true
				}
				withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, err.Error(), "source")
				return true
			}
			response.ValueJson = value
			response.Provider = "vrooli-code"
			response.Model = contract.HandlerID
			response.Validated = true
			finalizeUsage(response.Usage)
			return true
		}
	}
	return false
}

func (s *Service) runProviderAttempts(ctx context.Context, request ProviderRequest, schema map[string]any, response *inferencev1.RunResponse) *inferencev1.RunResponse {
	var validationErr error
	for attempt := 0; attempt < MaxValidationAttempts; attempt++ {
		attemptRequest, allowed := providerAttemptRequest(request, response.ProviderCallsMade, attempt, validationErr)
		if !allowed {
			break
		}
		result, err := s.repository.Run(ctx, attemptRequest)
		recordProviderAttempt(response, result, &request)
		if applyRecoveryResult(response, result) {
			return response
		}
		applyProviderResultMetadata(response, result)
		if err != nil {
			return applyProviderError(response, request, err)
		}
		var terminal bool
		terminal, validationErr = validateProviderValue(response, request, result, schema)
		if terminal {
			return response
		}
		if validationErr != nil {
			continue
		}
		return response
	}
	return validationFailure(response, validationErr)
}

type requestValidation struct{ message, construct string }

func validateRunRequest(request ProviderRequest) *requestValidation {
	if validation := validateBasicRunRequest(request); validation != nil {
		return validation
	}
	return validateBrowserRunRequest(request)
}

func validateBasicRunRequest(request ProviderRequest) *requestValidation {
	if err := validatePrivacyProvenance(request.PrivacyProvenance); err != nil {
		return &requestValidation{err.Error(), "privacy_provenance"}
	}
	if request.MaxProviderCalls < 0 || request.MaxProviderCalls > MaxProviderCalls {
		return &requestValidation{fmt.Sprintf("max_provider_calls must be between 0 and %d", MaxProviderCalls), "max_provider_calls"}
	}
	if strings.TrimSpace(request.Source) == "" && len(request.Turns) == 0 && len(request.Attachments) == 0 {
		return &requestValidation{"source is required", "source"}
	}
	if strings.TrimSpace(request.Role) == "" {
		return &requestValidation{"role is required", "role"}
	}
	return nil
}

func validateBrowserRunRequest(request ProviderRequest) *requestValidation {
	if !strings.HasPrefix(strings.TrimSpace(request.Role), "browser.") {
		return nil
	}
	if !validRootOperationID(request.RootOperationID) {
		return &requestValidation{"browser inference requires a caller-generated root_operation_id for exact reservation reconciliation", "root_operation_id"}
	}
	if !validBrowserSessionBinding(request.BrowserSession) {
		return &requestValidation{"browser inference requires its authenticated run, grant and BAS session binding", "browser_session"}
	}
	return nil
}

func providerAttemptRequest(request ProviderRequest, completedCalls int32, attempt int, validationErr error) (ProviderRequest, bool) {
	remaining := request.MaxProviderCalls
	if remaining > 0 {
		remaining -= completedCalls
		if remaining <= 0 {
			return ProviderRequest{}, false
		}
	}
	attemptRequest := request
	attemptRequest.ValidationAttempt = attempt
	attemptRequest.MaxProviderCalls = remaining
	if validationErr != nil {
		attemptRequest.Instruction = repairInstruction(request.Instruction, validationErr)
	}
	return attemptRequest, true
}

func recordProviderAttempt(response *inferencev1.RunResponse, result ProviderResult, request *ProviderRequest) {
	response.ProviderCallsMade += result.ProviderCallsMade
	if result.RightsContextSHA256 == "" {
		return
	}
	request.PriorRightsProvider, request.PriorRightsModel = result.Provider, result.Model
	request.PriorRightsContextSHA256 = result.RightsContextSHA256
}

func applyRecoveryResult(response *inferencev1.RunResponse, result ProviderResult) bool {
	if result.Recovery == nil {
		return false
	}
	response.Recovery = result.Recovery
	response.ValueJson, response.Provider, response.Model = "", "", ""
	response.Validated, response.Applied, response.SpendAuthorization = false, nil, nil
	response.Usage = &inferencev1.Usage{AccountingStatus: result.Recovery.GetAccountingStatus()}
	if result.Recovery.ChargedCredits != nil {
		response.Usage.CreditsCharged = proto.Int64(result.Recovery.GetChargedCredits())
	}
	response.Error = &inferencev1.InferenceError{
		Code:    inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_OUTCOME_UNCERTAIN,
		Message: "hosted inference output is unavailable; reuse root_operation_id to resolve its accounting receipt",
	}
	return true
}

func applyProviderResultMetadata(response *inferencev1.RunResponse, result ProviderResult) {
	response.Provider, response.Model = result.Provider, result.Model
	response.Applied = appliedSettings(result.Applied)
	response.SpendAuthorization = result.SpendAuthorization
	addUsage(response.Usage, usage(result))
}

func applyProviderError(response *inferencev1.RunResponse, request ProviderRequest, err error) *inferencev1.RunResponse {
	code, construct := samplingErrorCode(err)
	message := err.Error()
	if errors.Is(err, ErrOutcomeUncertain) {
		code, construct = inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_OUTCOME_UNCERTAIN, "root_operation_id"
		message = fmt.Sprintf("provider outcome is uncertain; reuse root_operation_id %q for reconciliation and do not switch funding source", request.RootOperationID)
	} else if errors.Is(err, ErrNoEligibleRoute) {
		code, construct = inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_NO_ELIGIBLE_ROUTE, "source_or_route_eligibility"
		message = "no eligible route is configured for this browser source and profile; select an available source or ask the account owner to enable an eligible route. No model inference was dispatched and funding was not changed"
	}
	if errors.Is(err, ErrRootConflict) || errors.Is(err, ErrRootRequired) {
		code, construct = inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "root_operation_id"
	}
	return withError(response, code, message, construct)
}

func validateProviderValue(response *inferencev1.RunResponse, request ProviderRequest, result ProviderResult, schema map[string]any) (bool, error) {
	response.ValueJson, response.BillingOperationId = result.ValueJSON, result.BillingOperationID
	if IsVisualLocateRole(request.Role) {
		value, err := NormalizeLocateVisualJSON(result.ValueJSON, result.CoordinateConvention, requestAttachments(request))
		if err != nil {
			withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED, err.Error(), "bounds")
			return true, nil
		}
		response.ValueJson = value
	}
	if err := schemagate.ValidateJSON(schema, []byte(response.ValueJson)); err != nil {
		return false, err
	}
	response.Validated = true
	finalizeUsage(response.Usage)
	return true, nil
}

func validationFailure(response *inferencev1.RunResponse, validationErr error) *inferencev1.RunResponse {
	return withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED,
		fmt.Sprintf("provider value failed local validation within the provider-call budget: %v", validationErr), "")
}

func validBrowserSessionBinding(binding *inferencev1.BrowserSessionBinding) bool {
	if binding == nil {
		return false
	}
	return validBrowserBindingValue(binding.RunId, 64) && validBrowserBindingValue(binding.SessionId, 128) && validBrowserBindingValue(binding.GrantReceipt, 512)
}

func validBrowserBindingValue(value string, maxBytes int) bool {
	if strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value {
		return false
	}
	if len(value) > maxBytes || strings.ContainsAny(value, "\r\n") {
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
	if validation := validateEmbeddingRequest(request); validation != nil {
		return embedError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, validation.message, validation.construct)
	}
	repository, ok := s.repository.(EmbeddingRepository)
	if !ok {
		return embedError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, "embedding repository is unavailable", "embedding")
	}
	result, err := repository.Embed(ctx, request)
	return finishEmbeddingResponse(response, request, result, err)
}

func validateEmbeddingRequest(request EmbeddingRequest) *requestValidation {
	if request.Sampling {
		return &requestValidation{"embedding roles are deterministic and reject sampling controls", "sampling"}
	}
	if strings.TrimSpace(request.Role) == "" {
		return &requestValidation{"role is required", "role"}
	}
	if len(request.Texts) == 0 {
		return &requestValidation{"texts are required", "texts"}
	}
	if request.Profile != sharedv1.Profile_PROFILE_LOCAL_ONLY {
		return &requestValidation{"typed embeddings require an explicit local-only profile", "profile"}
	}
	if err := validateEmbeddingPrivacy(request.PrivacyClass, request.PrivacyProvenance); err != nil {
		return &requestValidation{err.Error(), "privacy_provenance"}
	}
	return nil
}

func finishEmbeddingResponse(response *inferencev1.EmbedResponse, request EmbeddingRequest, result EmbeddingResult, err error) *inferencev1.EmbedResponse {
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
