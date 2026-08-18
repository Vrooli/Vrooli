package inference

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"ai-gateway/internal/providers"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"google.golang.org/protobuf/proto"
)

var ErrUnavailable = errors.New("typed inference is unavailable")

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
)

type Service struct {
	repository Repository
	gate       SchemaGate
}

func NewService(repository Repository) *Service {
	if repository == nil {
		repository = StaticRepository{}
	}
	return &Service{repository: repository}
}

func (s *Service) Run(ctx context.Context, request ProviderRequest) *inferencev1.RunResponse {
	response := &inferencev1.RunResponse{Usage: &inferencev1.Usage{}}
	schema, err := s.gate.Parse(request.SchemaJSON)
	if err != nil {
		return withError(response, schemaErrorCode(err), err.Error(), schemaConstruct(err))
	}
	return s.runWithSchema(ctx, request, schema, response)
}

func (s *Service) runWithSchema(ctx context.Context, request ProviderRequest, schema map[string]any, response *inferencev1.RunResponse) *inferencev1.RunResponse {
	if strings.TrimSpace(request.Source) == "" && len(request.Turns) == 0 && len(request.Attachments) == 0 {
		return withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "source is required", "source")
	}
	if strings.TrimSpace(request.Role) == "" {
		return withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "role is required", "role")
	}
	var validationErr error
	for attempt := 0; attempt < MaxValidationAttempts; attempt++ {
		attemptRequest := request
		if validationErr != nil {
			attemptRequest.Instruction = repairInstruction(request.Instruction, validationErr)
		}
		result, err := s.repository.Run(ctx, attemptRequest)
		response.Provider, response.Model = result.Provider, result.Model
		// Applied is set before the error branch: a refused or truncated call is
		// exactly when a caller needs to know what the gateway would have sent.
		response.Applied = appliedSettings(result.Applied)
		// Usage accumulates across attempts so a caller is billed for the work
		// actually performed, not only for the attempt that happened to land.
		addUsage(response.Usage, usage(result))
		if err != nil {
			// Provider failures are not retried here: the repository already
			// walks every declared candidate before reporting one.
			code, construct := samplingErrorCode(err)
			return withError(response, code, err.Error(), construct)
		}
		response.ValueJson = result.ValueJSON
		if strings.TrimSpace(request.Role) == "locate.visual" {
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
		return response
	}
	return withError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED,
		fmt.Sprintf("provider value failed local validation after %d attempts: %v", MaxValidationAttempts, validationErr), "")
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
	response.Results = results
	return response
}

func (s *Service) Embed(ctx context.Context, role string, texts []string, sampling bool) *inferencev1.EmbedResponse {
	response := &inferencev1.EmbedResponse{Usage: &inferencev1.Usage{}}
	if sampling {
		return embedError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "embedding roles are deterministic and reject sampling controls", "sampling")
	}
	if strings.TrimSpace(role) == "" {
		return embedError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "role is required", "role")
	}
	if len(texts) == 0 {
		return embedError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "texts are required", "texts")
	}
	repository, ok := s.repository.(EmbeddingRepository)
	if !ok {
		return embedError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, "embedding repository is unavailable", "embedding")
	}
	result, err := repository.Embed(ctx, role, texts)
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
	if result.Dimension <= 0 || len(result.Vectors) != len(texts) {
		return embedError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED, "embedding repository returned an incomplete vector batch", "vectors")
	}
	response.Provider, response.Model, response.Dimension = result.Provider, result.Model, int32(result.Dimension)
	response.Usage.InputTokens, response.Usage.CostMicros = result.InputTokens, result.CostMicros
	for _, vector := range result.Vectors {
		if len(vector) != result.Dimension {
			return embedError(response, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED, "embedding_dimension_mismatch", "dimension")
		}
		response.Vectors = append(response.Vectors, &inferencev1.EmbeddingVector{Values: vector})
	}
	return response
}

func embedError(response *inferencev1.EmbedResponse, code inferencev1.InferenceErrorCode, message, construct string) *inferencev1.EmbedResponse {
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
}

func usage(result ProviderResult) *inferencev1.Usage {
	return &inferencev1.Usage{InputTokens: result.InputTokens, OutputTokens: result.OutputTokens, CostMicros: result.CostMicros}
}

func withError(response *inferencev1.RunResponse, code inferencev1.InferenceErrorCode, message, construct string) *inferencev1.RunResponse {
	response.Validated = false
	response.Error = &inferencev1.InferenceError{Code: code, Message: message, Construct: construct}
	return response
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
