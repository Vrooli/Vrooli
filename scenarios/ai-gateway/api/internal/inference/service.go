package inference

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
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
		// Usage accumulates across attempts so a caller is billed for the work
		// actually performed, not only for the attempt that happened to land.
		addUsage(response.Usage, usage(result))
		if err != nil {
			// Provider failures are not retried here: the repository already
			// walks every declared candidate before reporting one.
			code := inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_PROVIDER_FAILED
			if errors.Is(err, ErrUnavailable) {
				code = inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE
			}
			return withError(response, code, err.Error(), "")
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
