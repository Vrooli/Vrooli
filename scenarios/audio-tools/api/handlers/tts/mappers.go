// Pure proto<->domain mappers and error mapping for the tts package.
// These helpers hold no I/O and depend on no handler state — safe to
// call from any goroutine.
package tts

import (
	"errors"

	"connectrpc.com/connect"

	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/protomap"
	inttts "audio-tools/internal/tts"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

func configToProto(c inttts.Config) *ttsv1.Config {
	return &ttsv1.Config{
		AutoEnabled:           c.AutoEnabled,
		DefaultVoice:          c.KokoroVoice,
		DefaultSpeed:          c.KokoroSpeed,
		DefaultResponseFormat: protomap.ResponseFormatToProto(c.DefaultResponseFormat),
	}
}

func mapChainError(err error) error {
	// errors.Is, not ==, so a provider that wraps a sentinel keeps its
	// actionable code instead of flattening to Internal.
	switch {
	case errors.Is(err, inttts.ErrInvalidArgument):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, ttschain.ErrInsufficientCredits):
		return connect.NewError(connect.CodeResourceExhausted, err)
	case errors.Is(err, ttschain.ErrUnknownBYOKProvider), errors.Is(err, ttschain.ErrMissingBYOKProvider):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, ttschain.ErrAllProvidersFailed):
		return connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
