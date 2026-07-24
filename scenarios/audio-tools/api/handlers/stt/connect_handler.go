// Package stt hosts the STTService Connect-RPC handler.
package stt

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	"audio-tools/internal/stt/quality"
	"audio-tools/internal/stt/session"
	"audio-tools/internal/sttcapacity"
	"audio-tools/internal/sttengine"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

type Deps struct {
	Chain    *sttchain.Chain
	Selector *sttpkg.Selector
	Registry *sttengine.Registry
	Voice    *sttpipeline.Service
	// SpeakerResource is the speaker-verification resource client used for
	// enrollment + streaming verification. nil means the resource is not
	// wired (enrollment fails FailedPrecondition; isolation falls back).
	SpeakerResource *sttpipeline.SpeakerClient
	Engine          *audioformat.Engine
	Logger          logx.Logger
	Clock           clock.Clock
	Usage           UsageRecorder
	StreamConfig    STTStreamConfigRepository
	SpeakerConfig   SpeakerConfigRepository
	Wakeword        WakewordRepository
	Speaker         SpeakerRepository
	// Capacity reports streaming transcription activity to the platform capacity
	// broker so the backing local resource (whisper/kyutai-stt) is marked active
	// (protected from idle reclaim) while a session is live. nil = no reporting.
	Capacity sttcapacity.Reporter
	// Sessions is the server-owned replay ledger registry shared by Connect
	// and WebSocket transports. Module supplies a bounded default when unset.
	Sessions *session.Registry
	// TestIsolationActive reports whether the server currently has the complete
	// leased DB-and-file isolation session installed. Qualification faults are
	// accepted only while this is true and the individual request opts in.
	TestIsolationActive func() bool
}

// UsageRecorder is the handler-owned port for non-blocking usage submission.
// The usage domain supplies the concrete asynchronous implementation.
type UsageRecorder interface{ Enqueue(store.UsageRow) }

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
	if d.Sessions == nil {
		d.Sessions = session.NewRegistry(0)
	}
	// Hydrate the in-process speaker-config cell from its persisted row so the
	// mode/threshold/profile bindings survive a restart (profiles always
	// persisted; bindings used to be lost). Best-effort: a missing/corrupt row
	// leaves the documented defaults in place.
	if d.SpeakerConfig != nil {
		loadPersistedSpeakerCfg(context.Background(), d.SpeakerConfig, d.Logger)
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Transcribe(ctx context.Context, req *connect.Request[sttv1.TranscribeRequest]) (*connect.Response[sttv1.TranscribeResponse], error) {
	if h.deps.Chain == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("stt chain not configured"))
	}
	if audioExceedsLimit(len(req.Msg.Audio)) {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("audio exceeds maximum size of %d bytes", sttpipeline.MaxAudioSize))
	}
	env := envelope.FromConnectRequest(req)
	cfg := h.resolveStreamPipelineConfig(ctx)
	chainReq := sttchain.Request{
		Audio:                   req.Msg.Audio,
		Format:                  protomap.AudioFormatFromProto(req.Msg.GetFormat()),
		Language:                req.Msg.Language,
		SkipSpeakerVerification: req.Msg.SkipSpeakerVerification,
		InitialPrompt:           req.Msg.InitialPrompt,
		VADFilter:               cfg.VADFilterEnabled,
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
	resp.Msg = h.responseFromResult(ctx, res, req.Msg.Audio, cfg)
	return resp, nil
}

func (h *connectHandler) responseFromResult(ctx context.Context, res *sttchain.Result, audio []byte, cfg sttpkg.StreamConfig) *sttv1.TranscribeResponse {
	decision := quality.New(cfg, h.deps.Registry, nil).ApplyResult(ctx, res, audio)
	resp := &sttv1.TranscribeResponse{
		Text:             decision.Text,
		DetectedLanguage: res.DetectedLanguage,
		DurationSeconds:  res.DurationSeconds,
		ProviderTier:     protomap.ProviderTierToProto(string(res.Tier)),
		ProviderId:       res.ProviderID,
		ModelId:          res.ModelID,
		LatencyMs:        float64(res.Latency.Milliseconds()),
		Filtered:         decision.Filtered,
		FilterReason:     decision.FilterReason,
	}
	if len(decision.Stages) > 0 || decision.Filtered {
		resp.PolicyDetails = map[string]string{
			"stages":                   strings.Join(decision.Stages, ","),
			"hallucination_filter":     fmt.Sprintf("%t", cfg.HallucinationFilterEnabled),
			"vad_filter":               fmt.Sprintf("%t", cfg.VADFilterEnabled),
			"no_speech_threshold":      fmt.Sprintf("%g", cfg.NoSpeechThreshold),
			"logprob_threshold":        fmt.Sprintf("%g", cfg.LogProbThreshold),
			"raw_transcript_length":    fmt.Sprintf("%d", len(res.Text)),
			"final_transcript_length":  fmt.Sprintf("%d", len(decision.Text)),
			"post_recognition_policy":  "stt_egress",
			"provider_confidence_used": fmt.Sprintf("%t", res.Confidence != nil),
		}
	}
	return resp
}

// GetSupportedFormats reports the STT ingress capability matrix from the
// audioformat substrate. It is informational and provider-agnostic: the
// accepted-format vocabulary is the full set the API understands, and the
// ffmpeg flag tells operators whether live non-PCM streams normalize
// locally or degrade to buffered whole-file decode.
func (h *connectHandler) GetSupportedFormats(_ context.Context, _ *connect.Request[sttv1.GetSupportedFormatsRequest]) (*connect.Response[sttv1.GetSupportedFormatsResponse], error) {
	eng := h.deps.Engine
	if eng == nil {
		eng = audioformat.New()
	}
	codecs := audioformat.AllInputCodecs()
	formats := make([]commonv1.AudioFormat, 0, len(codecs))
	for _, c := range codecs {
		formats = append(formats, audioformat.ToProto(c))
	}
	return connect.NewResponse(&sttv1.GetSupportedFormatsResponse{
		AcceptedFormats:       formats,
		FfmpegAvailable:       eng.HasFfmpeg(),
		CanonicalSampleRateHz: audioformat.CanonicalSampleRate,
		CanonicalChannels:     audioformat.CanonicalChannels,
	}), nil
}

func mapChainError(err error) error {
	// Backend-down (whisper/kyutai-stt unreachable) maps to an honest,
	// user-actionable code (plan L2): a transient/starting backend is Unavailable
	// ("retry shortly"); one needing operator action is FailedPrecondition. The
	// typed *STTBackendError carries a user-safe message (no raw dial string).
	var backendErr *sttpipeline.STTBackendError
	if errors.As(err, &backendErr) {
		if backendErr.Transient {
			return connect.NewError(connect.CodeUnavailable, err)
		}
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
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
