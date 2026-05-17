// Pure proto<->domain mappers and error mapping for the summarize package.
// These functions hold no I/O and depend on no handler state — safe to call
// from any goroutine.
package summarize

import (
	"connectrpc.com/connect"

	"audio-tools/internal/ai/summarizechain"
	intsumm "audio-tools/internal/summarize"

	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
)

func toProtoSummarizeConfig(c intsumm.SummarizeConfig) *summv1.SummarizeConfig {
	return &summv1.SummarizeConfig{
		Enabled:        c.Enabled,
		CharThreshold:  int32(c.CharThreshold),
		Level:          c.Level,
		Model:          c.Model,
		TimeoutSeconds: int32(c.TimeoutSeconds),
	}
}

func mapChainError(err error) error {
	switch err {
	case summarizechain.ErrInsufficientCredits:
		return connect.NewError(connect.CodeResourceExhausted, err)
	case summarizechain.ErrUnknownBYOKProvider, summarizechain.ErrMissingBYOKProvider:
		return connect.NewError(connect.CodeInvalidArgument, err)
	case summarizechain.ErrAllProvidersFailed:
		return connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
