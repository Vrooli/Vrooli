package inference

import (
	"errors"
	"math"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/schemagate"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"google.golang.org/protobuf/proto"
)

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
	var schemaErr *schemagate.SchemaError
	if errors.As(err, &schemaErr) && schemaErr.Construct != "" && schemaErr.Construct != "schema_json" && schemaErr.Construct != "schema" && schemaErr.Construct != "depth" {
		return inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNSUPPORTED_SCHEMA
	}
	return inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST
}

func schemaConstruct(err error) string {
	var schemaErr *schemagate.SchemaError
	if errors.As(err, &schemaErr) {
		return schemaErr.Construct
	}
	return ""
}

// samplingConstruct names the offending control on a sampling refusal, matching
// how UNSUPPORTED_SCHEMA names the offending keyword.
const samplingConstruct = "sampling.temperature"

// samplingErrorCode distinguishes caller errors from provider capability errors.
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

// appliedSettings projects the repository's account onto the wire.
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
