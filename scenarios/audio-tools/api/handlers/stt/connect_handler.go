// Package stt hosts the STTService Connect-RPC handler.
package stt

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	"audio-tools/internal/ai/sttchain"
	intvoice "audio-tools/internal/voice"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

type Deps struct {
	Chain   *sttchain.Chain
	Voice   *intvoice.Service
	Logger  *log.Logger
}

type connectHandler struct {
	sttconnect.UnimplementedSTTServiceHandler
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
	chainReq := sttchain.Request{
		Audio:                   req.Msg.Audio,
		Format:                  req.Msg.Format,
		Language:                req.Msg.Language,
		SkipSpeakerVerification: req.Msg.SkipSpeakerVerification,
		InitialPrompt:           req.Msg.InitialPrompt,
		BYOKProvider:            req.Header().Get("X-Audio-BYOK-Provider"),
		BYOKKey:                 req.Header().Get("X-Audio-BYOK-Key"),
		LPBSToken:               req.Header().Get("X-Audio-LPBS-Token"),
		UserIdentity:            req.Header().Get("X-Audio-User-Identity"),
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
	}
	return connect.NewError(connect.CodeInternal, err)
}
