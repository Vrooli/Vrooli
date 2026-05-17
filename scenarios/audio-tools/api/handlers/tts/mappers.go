// Pure proto<->domain mappers and error mapping for the tts package.
// These helpers hold no I/O and depend on no handler state — safe to
// call from any goroutine.
package tts

import (
	"connectrpc.com/connect"

	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/protomap"
	intsumm "audio-tools/internal/summarize"
	inttts "audio-tools/internal/tts"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

func configToProto(c inttts.Config, s intsumm.SummarizeConfig) *ttsv1.Config {
	return &ttsv1.Config{
		AutoEnabled:             c.AutoEnabled,
		DefaultVoice:            c.KokoroVoice,
		DefaultSpeed:            c.KokoroSpeed,
		DefaultResponseFormat:   protomap.ResponseFormatToProto(c.Backend),
		SummarizeEnabled:        s.Enabled,
		SummarizeCharThreshold:  int32(s.CharThreshold),
		SummarizeLevel:          protomap.SummarizeLevelToProto(s.Level),
		SummarizeModel:          s.Model,
		SummarizeTimeoutSeconds: int32(s.TimeoutSeconds),
	}
}

func mapChainError(err error) error {
	switch {
	case err == ttschain.ErrInsufficientCredits:
		return connect.NewError(connect.CodeResourceExhausted, err)
	case err == ttschain.ErrUnknownBYOKProvider, err == ttschain.ErrMissingBYOKProvider:
		return connect.NewError(connect.CodeInvalidArgument, err)
	case err == ttschain.ErrAllProvidersFailed:
		return connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
