// Package stt hosts the STTService Connect-RPC handler.
package stt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"audio-tools/internal/ai/chains/tiered"
	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
	"audio-tools/internal/byok/envelope"
	"audio-tools/internal/capabilities"
	"audio-tools/internal/logx"
	"audio-tools/internal/protomap"
	"audio-tools/internal/store"
	sttpkg "audio-tools/internal/stt"
	sttpipeline "audio-tools/internal/stt/pipeline"
	"audio-tools/internal/stt/quality"
	"audio-tools/internal/stt/session"
	"audio-tools/internal/sttcapacity"
	"audio-tools/internal/sttengine"

	"github.com/vrooli/api-core/schedule"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

type Deps struct {
	Chain    *sttchain.Chain
	Selector *sttpkg.Selector
	Registry *sttengine.Registry
	// SpeakerResource is the speaker-verification resource client used for
	// enrollment + streaming verification. nil means the resource is not
	// wired (enrollment fails FailedPrecondition; isolation falls back).
	SpeakerResource *sttpipeline.SpeakerClient
	Engine          *audioformat.Engine
	Logger          logx.Logger
	Clock           schedule.Clock
	Usage           UsageRecorder
	StreamConfig    STTStreamConfigRepository
	SpeakerConfig   SpeakerConfigRepository
	Wakeword        WakewordRepository
	Speaker         SpeakerRepository
	// Capacity reports streaming transcription activity to the platform capacity
	// broker so the backing local resource (whisper/kyutai-stt) is marked active
	// (protected from idle reclaim) while a session is live. nil = no reporting.
	Capacity          sttcapacity.Reporter
	DefaultCredential DefaultCredentialResolver
	EngineResolver    *sttengine.LiveResolver
	// Sessions is the server-owned replay ledger registry for Connect streams.
	Sessions *session.Registry
	// StreamIdleTimeout bounds periods with no inbound audio after the stream
	// handshake. Zero selects the conservative default.
	StreamIdleTimeout time.Duration
	// StreamIdleTimerFactory is an injectable timer seam for stream lifecycle tests.
	StreamIdleTimerFactory func(time.Duration) StreamIdleTimer
}

// isNilDependency reports whether an optional dependency is absent, including a
// typed nil pointer stored in an interface. Optional repositories are commonly
// wired as (*store.T)(nil); the interface comparison alone would be non-nil and
// the first method call would panic.
func isNilDependency(dep any) bool {
	if dep == nil {
		return true
	}
	value := reflect.ValueOf(dep)
	switch value.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return value.IsNil()
	default:
		return false
	}
}

func (h *connectHandler) applyDefaultCredential(ctx context.Context, capability, provider, key string) (string, string) {
	if key != "" || h.deps.DefaultCredential == nil {
		return provider, key
	}
	defaultProvider, defaultKey, ok := h.deps.DefaultCredential(ctx, capability)
	if ok {
		return defaultProvider, defaultKey
	}
	return provider, key
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
	// A nil *store assigned to the interface is a non-nil interface holding a
	// nil pointer (typed nil). Treat it as unconfigured rather than letting the
	// first method call panic during construction.
	if !isNilDependency(d.SpeakerConfig) {
		loadPersistedSpeakerCfg(context.Background(), d.SpeakerConfig, d.Logger)
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Transcribe(ctx context.Context, req *connect.Request[sttv1.TranscribeRequest]) (*connect.Response[sttv1.TranscribeResponse], error) {
	if h.deps.Chain == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("stt chain not configured"))
	}
	if len(req.Msg.Audio) > sttpipeline.MaxAudioSize {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("audio exceeds maximum size of %d bytes", sttpipeline.MaxAudioSize))
	}
	env := envelope.FromConnectRequest(req)
	modelID := strings.TrimSpace(req.Header().Get("X-Audio-Model-ID"))
	if modelID == "" {
		env.Provider, env.Key = h.applyDefaultCredential(ctx, "stt", env.Provider, env.Key)
	}
	cfg := h.resolveStreamPipelineConfig(ctx)
	chainReq := sttchain.Request{
		Audio:                   req.Msg.Audio,
		Format:                  protomap.AudioFormatFromProto(req.Msg.GetFormat()),
		Language:                req.Msg.Language,
		SkipSpeakerVerification: req.Msg.SkipSpeakerVerification,
		InitialPrompt:           req.Msg.InitialPrompt,
		Model:                   modelID,
		VADFilter:               cfg.VADFilterEnabled,
		BYOKProvider:            env.Provider,
		BYOKKey:                 env.Key,
		LPBSToken:               env.LPBSToken,
		UserIdentity:            env.UserIdentity,
	}
	var speakerBatch *batchSpeakerOutcome
	if !req.Msg.SkipSpeakerVerification {
		var policyErr error
		chainReq.Audio, speakerBatch, policyErr = h.applyBatchSpeakerPolicy(ctx, chainReq.Audio, speakerCodec(chainReq.Format))
		if policyErr != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, policyErr)
		}
		if speakerBatch != nil {
			chainReq.Format = audioformat.CodecPCMS16LE.String()
		}
		if speakerBatch != nil && !speakerBatch.Allowed {
			response := connect.NewResponse(&sttv1.TranscribeResponse{Filtered: true, FilterReason: speakerBatch.Reason})
			h.addBatchSpeakerPolicy(response.Msg, speakerBatch)
			return response, nil
		}
	}
	opID := req.Header().Get("X-Audio-Operation-ID")
	if opID == "" {
		opID = uuid.NewString()
	}
	start := h.deps.Clock.Now()
	resp := connect.NewResponse(&sttv1.TranscribeResponse{})
	fallbackReason := ""
	ctx = tiered.WithOnFallback(ctx, func(ev tiered.FallbackEvent) {
		fallbackReason = fmt.Sprintf("%s_to_%s:%s", ev.From.String(), ev.To.String(), ev.Reason)
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
	row.FallbackReason = fallbackReason
	if h.deps.Usage != nil {
		h.deps.Usage.Enqueue(row)
	}
	resp.Msg = h.responseFromResult(ctx, res, chainReq.Audio, cfg, speakerBatch == nil)
	if speakerBatch != nil {
		h.addBatchSpeakerPolicy(resp.Msg, speakerBatch)
	}
	return resp, nil
}

func speakerCodec(format string) audioformat.Codec {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "pcm", "pcm_s16le":
		return audioformat.CodecPCMS16LE
	case "wav":
		return audioformat.CodecWAV
	case "mp3":
		return audioformat.CodecMP3
	case "flac":
		return audioformat.CodecFLAC
	case "ogg":
		return audioformat.CodecOGG
	case "webm":
		return audioformat.CodecWebM
	case "opus":
		return audioformat.CodecOpus
	case "aac":
		return audioformat.CodecAAC
	default:
		return audioformat.CodecUnknown
	}
}

// batchSpeakerOutcome is the batch equivalent of the per-segment streaming
// speaker gate. Keeping this result explicit prevents batch transcription from
// silently bypassing an enabled policy and gives clients an honest applied vs
// degraded state.
type batchSpeakerOutcome struct {
	Allowed      bool
	Applied      bool
	FallbackUsed bool
	Extracted    bool
	Reason       string
	Score        float64
	Threshold    float64
}

func (h *connectHandler) applyBatchSpeakerPolicy(ctx context.Context, audio []byte, format audioformat.Codec) ([]byte, *batchSpeakerOutcome, error) {
	isolation := currentSpeakerIsolation(h.deps)
	extractor := currentSpeakerExtraction(h.deps)
	if isolation == nil && extractor == nil {
		return audio, nil, nil
	}
	engine := h.deps.Engine
	if engine == nil {
		engine = audioformat.New()
	}
	pcm, err := engine.Normalize(ctx, format, audio)
	if err != nil {
		return nil, nil, fmt.Errorf("normalize audio for speaker policy: %w", err)
	}
	out := &batchSpeakerOutcome{Allowed: true}
	if extractor != nil {
		cleaned, extractErr := extractor.Extract(ctx, pcm)
		if extractErr != nil {
			// Extraction is a degraded-but-recoverable path; verification below
			// still decides whether required policy can emit text.
			out.Reason = "speaker extraction degraded: " + extractErr.Error()
		} else if len(cleaned) > 0 {
			out.Extracted = !bytes.Equal(cleaned, pcm)
			pcm = cleaned
		}
	}
	if isolation != nil {
		verdict := isolation.Evaluate(ctx, pcm)
		out.Allowed = verdict.Allowed
		out.Applied = verdict.Applied
		out.FallbackUsed = verdict.FallbackUsed
		out.Reason = verdict.Reason
		out.Score = verdict.Score
		out.Threshold = verdict.Threshold
	}
	// Once the batch has been normalized, keep the canonical PCM fast path for
	// recognition. This is also the format the extraction and verification
	// adapters measure, avoiding a second decode or a container mismatch.
	return pcm, out, nil
}

func (h *connectHandler) addBatchSpeakerPolicy(resp *sttv1.TranscribeResponse, outcome *batchSpeakerOutcome) {
	if resp == nil || outcome == nil {
		return
	}
	if resp.PolicyDetails == nil {
		resp.PolicyDetails = map[string]string{}
	}
	state := "applied"
	if outcome.FallbackUsed || (outcome.Reason != "" && !outcome.Applied) {
		state = "degraded"
	}
	if !outcome.Allowed && outcome.Reason != "" {
		state = "rejected"
	}
	resp.PolicyDetails["speaker_policy"] = state
	resp.PolicyDetails["speaker_verification_applied"] = fmt.Sprintf("%t", outcome.Applied)
	resp.PolicyDetails["speaker_extraction_applied"] = fmt.Sprintf("%t", outcome.Extracted)
	resp.PolicyDetails["speaker_fallback_used"] = fmt.Sprintf("%t", outcome.FallbackUsed)
	resp.PolicyDetails["speaker_score"] = fmt.Sprintf("%g", outcome.Score)
	resp.PolicyDetails["speaker_threshold"] = fmt.Sprintf("%g", outcome.Threshold)
	if outcome.Reason != "" {
		resp.PolicyDetails["speaker_policy_reason"] = outcome.Reason
	}
}

func (h *connectHandler) responseFromResult(ctx context.Context, res *sttchain.Result, audio []byte, cfg sttpkg.StreamConfig, sourceCoordinatesKnown bool) *sttv1.TranscribeResponse {
	decision := quality.New(cfg, h.deps.Registry, nil).ApplyResult(ctx, res, audio)
	resp := &sttv1.TranscribeResponse{
		Text:             decision.Text,
		DetectedLanguage: res.DetectedLanguage,
		DurationSeconds:  res.DurationSeconds,
		ProviderTier:     protomap.ProviderTierToProto(string(res.Tier)),
		ProviderId:       capabilities.CanonicalProviderID(res.ProviderID),
		ModelId:          res.ModelID,
		LatencyMs:        float64(res.Latency) / float64(time.Millisecond),
		Filtered:         decision.Filtered,
		FilterReason:     decision.FilterReason,
	}
	if !decision.Filtered && sourceCoordinatesKnown {
		for _, word := range res.Words {
			item := &sttv1.WordTimestamp{Word: word.Word, StartSeconds: word.Start, EndSeconds: word.End}
			if res.Tier == sttchain.TierLocal && res.Confidence != nil {
				probability := word.Prob
				item.Probability = &probability
			}
			resp.WordTimestamps = append(resp.WordTimestamps, item)
		}
	}
	if len(decision.Stages) > 0 || decision.Filtered || (cfg.EngineID != "" && cfg.EngineID != res.ProviderID) {
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
		if cfg.EngineID != "" && cfg.EngineID != res.ProviderID {
			resp.PolicyDetails["selected_engine"] = cfg.EngineID
			resp.PolicyDetails["served_by_engine"] = res.ProviderID
			resp.PolicyDetails["engine_substitution"] = "true"
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
	// errors.Is, not ==, so a provider that wraps a sentinel keeps its
	// actionable code instead of flattening to Internal.
	switch {
	case errors.Is(err, sttchain.ErrInsufficientCredits):
		return connect.NewError(connect.CodeResourceExhausted, err)
	case errors.Is(err, sttchain.ErrUnknownBYOKProvider), errors.Is(err, sttchain.ErrMissingBYOKProvider):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, sttchain.ErrUnaryModelUnconfigured):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, sttchain.ErrUnaryModelUnavailable), errors.Is(err, sttchain.ErrLocalModelDisabled):
		return connect.NewError(connect.CodeUnavailable, err)
	case errors.Is(err, sttchain.ErrAllProvidersFailed):
		return connect.NewError(connect.CodeUnavailable, err)
	case errors.Is(err, sttpkg.ErrIncompatibleStrategyProvider), errors.Is(err, sttpkg.ErrStreamingDisabled):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, sttpkg.ErrNoEligibleProvider):
		return connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
