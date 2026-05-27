// Package stt hosts the STTService Connect-RPC handler.
package stt

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"audio-tools/internal/ai/chains/tiered"
	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
	"audio-tools/internal/byok/envelope"
	"audio-tools/internal/clock"
	"audio-tools/internal/logx"
	"audio-tools/internal/protomap"
	"audio-tools/internal/store"
	sttpkg "audio-tools/internal/stt"
	sttpipeline "audio-tools/internal/stt/pipeline"
	"audio-tools/internal/usagereport"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

type Deps struct {
	Chain        *sttchain.Chain
	Selector     *sttpkg.Selector
	Voice        *sttpipeline.Service
	Engine       *audioformat.Engine
	Logger       logx.Logger
	Clock        clock.Clock
	Usage        usagereport.Recorder
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
	opID := req.Header().Get("X-Audio-Operation-ID")
	if opID == "" {
		opID = uuid.NewString()
	}
	start := h.deps.Clock.Now()
	resp := connect.NewResponse(&sttv1.TranscribeResponse{})
	ctx = tiered.WithOnFallback(ctx, func(ev tiered.FallbackEvent) {
		resp.Header().Set("x-audio-tools-fallback",
			fmt.Sprintf("from=%s;to=%s;reason=%s", ev.From.String(), ev.To.String(), ev.Reason))
	})
	res, err := h.deps.Chain.Execute(ctx, chainReq)
	row := store.UsageRow{
		OperationID:  opID,
		EmittedAt:    h.deps.Clock.Now().UTC(),
		Capability:   "stt",
		Operation:    "transcribe",
		LatencyMs:    float64(h.deps.Clock.Now().Sub(start).Milliseconds()),
		UserIdentity: chainReq.UserIdentity,
	}
	if err != nil {
		row.Error = err.Error()
		if h.deps.Usage != nil {
			h.deps.Usage.Enqueue(row)
		}
		return nil, mapChainError(err)
	}
	row.ProviderTier = string(res.Tier)
	row.ProviderID = res.ProviderID
	row.ModelID = res.ModelID
	row.AudioDurationSeconds = res.DurationSeconds
	if h.deps.Usage != nil {
		h.deps.Usage.Enqueue(row)
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
