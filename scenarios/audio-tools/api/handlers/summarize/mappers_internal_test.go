package summarize

import (
	"fmt"
	"testing"

	"connectrpc.com/connect"

	"audio-tools/internal/ai/summarizechain"
)

// TestMapChainError_WrappedSentinels pins errors.Is semantics: a provider
// that wraps a sentinel must keep the actionable code, not flatten to
// Internal.
func TestMapChainError_WrappedSentinels(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want connect.Code
	}{
		{"credits", fmt.Errorf("gateway: %w", summarizechain.ErrInsufficientCredits), connect.CodeResourceExhausted},
		{"unknown_byok", fmt.Errorf("adapter: %w", summarizechain.ErrUnknownBYOKProvider), connect.CodeInvalidArgument},
		{"missing_byok", fmt.Errorf("adapter: %w", summarizechain.ErrMissingBYOKProvider), connect.CodeInvalidArgument},
		{"all_failed", fmt.Errorf("chain: %w", summarizechain.ErrAllProvidersFailed), connect.CodeUnavailable},
	}
	for _, tc := range cases {
		if got := connect.CodeOf(mapChainError(tc.in)); got != tc.want {
			t.Errorf("%s: code = %v, want %v", tc.name, got, tc.want)
		}
	}
}
