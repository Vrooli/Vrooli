// Package stt hosts the STTService Connect-RPC handler.
package stt

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"audio-tools/internal/ai/chains/tiered"
	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/byok/envelope"
	"audio-tools/internal/clock"
	"audio-tools/internal/logx"
	"audio-tools/internal/protomap"
	sttpkg "audio-tools/internal/stt"
	sttpipeline "audio-tools/internal/stt/pipeline"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

type Deps struct {
	Chain        *sttchain.Chain
	Selector     *sttpkg.Selector
	Voice        *sttpipeline.Service
	Logger       logx.Logger
	Clock        clock.Clock
	StreamConfig STTStreamConfigRepository
	Wakeword     WakewordRepository
	Speaker      SpeakerRepository
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the Connect handler. Deps.Logger and
// Deps.Clock are required seams; nil values panic.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		panic("stt.NewConnectHandler requires Deps.Logger")
	}
	if d.Clock == nil {
		panic("stt.NewConnectHandler requires Deps.Clock")
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Transcribe(ctx context.Context, req *connect.Request[sttv1.TranscribeRequest]) (*connect.Response[sttv1.TranscribeResponse], error) {
	if h.deps.Chain == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("stt chain not configured"))
	}
	env := envelope.FromConnectRequest(req)
	chainReq := sttchain.Request{
		Audio:                   req.Msg.Audio,
		Format:                  protomap.AudioFormatFromProto(req.Msg.GetFormat()),
		Language:                req.Msg.Language,
		SkipSpeakerVerification: req.Msg.SkipSpeakerVerification,
		InitialPrompt:           req.Msg.InitialPrompt,
		BYOKProvider:            env.Provider,
		BYOKKey:                 env.Key,
		LPBSToken:               env.LPBSToken,
		UserIdentity:            env.UserIdentity,
	}
	resp := connect.NewResponse(&sttv1.TranscribeResponse{})
	ctx = tiered.WithOnFallback(ctx, func(ev tiered.FallbackEvent) {
		resp.Header().Set("x-audio-tools-fallback",
			fmt.Sprintf("from=%s;to=%s;reason=%s", ev.From.String(), ev.To.String(), ev.Reason))
	})
	res, err := h.deps.Chain.Execute(ctx, chainReq)
	if err != nil {
		return nil, mapChainError(err)
	}
	resp.Msg = &sttv1.TranscribeResponse{
		Text:             res.Text,
		DetectedLanguage: res.DetectedLanguage,
		DurationSeconds:  res.DurationSeconds,
		ProviderTier:     protomap.ProviderTierToProto(string(res.Tier)),
		ProviderId:       res.ProviderID,
		ModelId:          res.ModelID,
		LatencyMs:        float64(res.Latency.Milliseconds()),
	}
	return resp, nil
}

func mapChainError(err error) error {
	switch err {
	case sttchain.ErrInsufficientCredits:
		return connect.NewError(connect.CodeResourceExhausted, err)
	case sttchain.ErrUnknownBYOKProvider, sttchain.ErrMissingBYOKProvider:
		return connect.NewError(connect.CodeInvalidArgument, err)
	case sttchain.ErrAllProvidersFailed:
		return connect.NewError(connect.CodeUnavailable, err)
	case sttpkg.ErrIncompatibleStrategyProvider, sttpkg.ErrStreamingDisabled:
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case sttpkg.ErrNoEligibleProvider:
		return connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
