// Pure proto<->domain mappers and error mapping for the summarize package.
// These functions hold no I/O and depend on no handler state — safe to call
// from any goroutine.
package summarize

import (
	"errors"

	"connectrpc.com/connect"

	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/protomap"
	intsumm "audio-tools/internal/summarize"

	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
)

func toProtoSummarizeConfig(c intsumm.SummarizeConfig) *summv1.SummarizeConfig {
	return &summv1.SummarizeConfig{
		Enabled:        c.Enabled,
		CharThreshold:  int32(c.CharThreshold),
		Level:          protomap.SummarizeLevelToProto(c.Level),
		Model:          c.Model,
		TimeoutSeconds: int32(c.TimeoutSeconds),
	}
}

func toProtoSummarizeModel(m intsumm.SummarizeModelInfo) *summv1.SummarizeModel {
	return &summv1.SummarizeModel{
		Id:              m.ID,
		DisplayName:     m.DisplayName,
		Installed:       m.Installed,
		Recommended:     m.Recommended,
		DefaultEligible: m.DefaultEligible,
		Reasoning:       m.Reasoning,
		StatusLabel:     m.StatusLabel,
		PullCommand:     m.PullCommand,
		SizeBytes:       m.SizeBytes,
		ParameterSize:   m.ParameterSize,
		SourceUrl:       m.SourceURL,
		Notes:           m.Notes,
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
	if errors.Is(err, intsumm.ErrSummarizeModelNotInstalled) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
