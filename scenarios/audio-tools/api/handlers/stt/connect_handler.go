// Package stt hosts the STTService Connect-RPC handler.
package stt

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/byok/envelope"
	"audio-tools/internal/store"
	sttpkg "audio-tools/internal/stt"
	sttpipeline "audio-tools/internal/stt/pipeline"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

type Deps struct {
	Chain        *sttchain.Chain
	Selector     *sttpkg.Selector
	Voice        *sttpipeline.Service
	Logger       *log.Logger
	StreamConfig *store.STTStreamConfigStore
	Wakeword     *store.WakeWordStore
	Speaker      *store.SpeakerStore
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
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
		Format:                  req.Msg.Format,
		Language:                req.Msg.Language,
		SkipSpeakerVerification: req.Msg.SkipSpeakerVerification,
		InitialPrompt:           req.Msg.InitialPrompt,
		BYOKProvider:            env.Provider,
		BYOKKey:                 env.Key,
		LPBSToken:               env.LPBSToken,
		UserIdentity:            env.UserIdentity,
	}
	res, err := h.deps.Chain.Execute(ctx, chainReq)
	if err != nil {
		return nil, mapChainError(err)
	}
	return connect.NewResponse(&sttv1.TranscribeResponse{
		Text:             res.Text,
		DetectedLanguage: res.DetectedLanguage,
		DurationSeconds:  res.DurationSeconds,
		ProviderTier:     string(res.Tier),
		ProviderId:       res.ProviderID,
		ModelId:          res.ModelID,
		LatencyMs:        float64(res.Latency.Milliseconds()),
	}), nil
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
