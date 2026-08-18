package inference

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type samplingErrRepo struct{ err error }

func (r samplingErrRepo) Run(context.Context, ProviderRequest) (ProviderResult, error) {
	return ProviderResult{}, r.err
}

// The two sampling failures must surface as different codes on the wire, and
// the response must still report what the gateway would have done.
func TestServiceMapsSamplingErrorsAndAlwaysReportsApplied(t *testing.T) {
	for _, tc := range []struct {
		name      string
		err       error
		want      inferencev1.InferenceErrorCode
		construct string
	}{
		{"role forbids override", ErrRoleForbidsSampling, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, "sampling.temperature"},
		{"no candidate honors", ErrUnsupportedSampling, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNSUPPORTED_SAMPLING, "sampling.temperature"},
		{"context overflow", ErrContextOverflow, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_CONTEXT_OVERFLOW, "context"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := NewService(samplingErrRepo{err: tc.err})
			response := service.Run(context.Background(), ProviderRequest{
				Source: "s", SchemaJSON: `{"type":"object"}`, Role: "write.default",
			})
			require.Equal(t, tc.want, response.GetError().GetCode())
			require.Equal(t, tc.construct, response.GetError().GetConstruct())
			require.NotNil(t, response.GetApplied(), "applied is populated on error paths too")
			require.Equal(t, sharedv1.SamplingSupport_SAMPLING_SUPPORT_UNSPECIFIED,
				response.GetApplied().GetTemperatureSupport(),
				"no candidate resolved, so there is no declaration to report")
		})
	}
}
