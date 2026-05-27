// Package tts hosts the TTSService Connect-RPC handler.
package tts

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"audio-tools/internal/ai/chains/tiered"
	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/audioformat"
	"audio-tools/internal/byok/envelope"
	"audio-tools/internal/protomap"
	"audio-tools/internal/store"
	"audio-tools/internal/text/normalizer"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs a Connect handler. The Chain is required for
// Synthesize; admin methods (config/status/cache/playback) bind to their
// respective stores via the corresponding Deps fields. Deps.Logger and
// Deps.Clock are required seams (no fallback); nil values panic.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		panic("tts.NewConnectHandler requires Deps.Logger")
	}
	if d.Clock == nil {
		panic("tts.NewConnectHandler requires Deps.Clock")
	}
	return &connectHandler{deps: d}
}

// GetSupportedFormats reports the TTS egress capability matrix from the
// audioformat substrate. The emitted-format set is the full output
// vocabulary (the synthesis engine encodes to these itself, so it does not
// depend on the substrate's own ffmpeg); the ffmpeg flag is informational.
func (h *connectHandler) GetSupportedFormats(_ context.Context, _ *connect.Request[ttsv1.GetSupportedFormatsRequest]) (*connect.Response[ttsv1.GetSupportedFormatsResponse], error) {
	eng := h.deps.Engine
	if eng == nil {
		eng = audioformat.New()
	}
	outs := audioformat.AllOutputFormats()
	formats := make([]commonv1.ResponseFormat, 0, len(outs))
	for _, f := range outs {
		formats = append(formats, audioformat.OutputFormatToProto(f))
	}
	return connect.NewResponse(&ttsv1.GetSupportedFormatsResponse{
		EmittedFormats:  formats,
		FfmpegAvailable: eng.HasFfmpeg(),
	}), nil
}

// voiceOverridesFromProto translates the typed AdapterMapping list to
// the legacy "tier:provider-id" -> backend voice id map that the TTS
// chain still consumes internally.
func voiceOverridesFromProto(in []*ttsv1.AdapterMapping) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for _, m := range in {
		tier := protomap.ProviderTierFromProto(m.GetTier())
		if tier == "" || m.GetProviderId() == "" {
			continue
		}
		out[tier+":"+m.GetProviderId()] = m.GetBackendVoiceId()
	}
	return out
}

// SynthesizeStream routes through the TTS provider chain's streaming
// path. When no tier declares streaming, the chain falls back to the
// unary Execute() and emits a single is_final=true frame — the wire
// shape is identical so consumers code against a stream uniformly.
func (h *connectHandler) SynthesizeStream(ctx context.Context, req *connect.Request[ttsv1.SynthesizeRequest], stream *connect.ServerStream[ttsv1.AudioFrame]) error {
	if h.deps.Chain == nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("tts chain not configured"))
	}
	env := envelope.FromConnectRequest(req)
	chainReq := ttschain.Request{
		Text:           req.Msg.Text,
		Voice:          req.Msg.Voice,
		VoiceOverrides: voiceOverridesFromProto(req.Msg.GetVoiceOverrides()),
		Speed:          req.Msg.Speed,
		ResponseFormat: protomap.ResponseFormatFromProto(req.Msg.GetResponseFormat()),
		BYOKProvider:   env.Provider,
		BYOKKey:        env.Key,
		LPBSToken:      env.LPBSToken,
		UserIdentity:   env.UserIdentity,
		EventID:        req.Msg.EventId,
		Version:        req.Msg.Version,
	}
	frames, err := h.deps.Chain.Stream(ctx, chainReq)
	if err != nil {
		return mapChainError(err)
	}
	for frame := range frames {
		if frame.Err != nil {
			return mapChainError(frame.Err)
		}
		msg := &ttsv1.AudioFrame{
			Audio:       frame.Audio,
			ContentType: frame.ContentType,
			IsFinal:     frame.IsFinal,
		}
		if frame.IsFinal {
			msg.ProviderTier = protomap.ProviderTierToProto(string(frame.Tier))
			msg.ProviderId = frame.ProviderID
			msg.ModelId = frame.ModelID
			msg.VoiceUsed = frame.VoiceUsed
			msg.LatencyMs = float64(frame.Latency.Milliseconds())
			msg.ContentHash = frame.ContentHash
		}
		if err := stream.Send(msg); err != nil {
			return err
		}
	}
	return nil
}

// Synthesize routes through the TTS provider chain.
func (h *connectHandler) Synthesize(ctx context.Context, req *connect.Request[ttsv1.SynthesizeRequest]) (*connect.Response[ttsv1.SynthesizeResponse], error) {
	if h.deps.Chain == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("tts chain not configured"))
	}
	env := envelope.FromConnectRequest(req)
	chainReq := ttschain.Request{
		Text:           req.Msg.Text,
		Voice:          req.Msg.Voice,
		VoiceOverrides: voiceOverridesFromProto(req.Msg.GetVoiceOverrides()),
		Speed:          req.Msg.Speed,
		ResponseFormat: protomap.ResponseFormatFromProto(req.Msg.GetResponseFormat()),
		BYOKProvider:   env.Provider,
		BYOKKey:        env.Key,
		LPBSToken:      env.LPBSToken,
		UserIdentity:   env.UserIdentity,
		EventID:        req.Msg.EventId,
		Version:        req.Msg.Version,
	}
	opID := req.Header().Get("X-Audio-Operation-ID")
	if opID == "" {
		opID = uuid.NewString()
	}
	start := h.deps.Clock.Now()
	resp := connect.NewResponse(&ttsv1.SynthesizeResponse{})
	ctx = tiered.WithOnFallback(ctx, func(ev tiered.FallbackEvent) {
		resp.Header().Set("x-audio-tools-fallback",
			fmt.Sprintf("from=%s;to=%s;reason=%s", ev.From.String(), ev.To.String(), ev.Reason))
	})
	res, err := h.deps.Chain.Execute(ctx, chainReq)
	row := store.UsageRow{
		OperationID:  opID,
		EmittedAt:    h.deps.Clock.Now().UTC(),
		Capability:   "tts",
		Operation:    "synthesize",
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
	if h.deps.Usage != nil {
		h.deps.Usage.Enqueue(row)
	}
	resp.Msg = &ttsv1.SynthesizeResponse{
		Audio:        res.Audio,
		ContentType:  res.ContentType,
		ContentHash:  res.ContentHash,
		ProviderTier: protomap.ProviderTierToProto(string(res.Tier)),
		ProviderId:   res.ProviderID,
		ModelId:      res.ModelID,
		VoiceUsed:    res.VoiceUsed,
		LatencyMs:    float64(res.Latency.Milliseconds()),
	}
	return resp, nil
}

// ListVoices returns the canonical voice catalog. Until internal/tts/voice_catalog.go
// lands, we ship a static list of canonical IDs so consumers can enumerate
// the wire-stable set.
func (h *connectHandler) ListVoices(ctx context.Context, req *connect.Request[ttsv1.ListVoicesRequest]) (*connect.Response[ttsv1.ListVoicesResponse], error) {
	voices := []*ttsv1.Voice{
		{Id: "voice.feminine.warm", Name: "Feminine — Warm"},
		{Id: "voice.feminine.neutral", Name: "Feminine — Neutral"},
		{Id: "voice.masculine.warm", Name: "Masculine — Warm"},
		{Id: "voice.masculine.neutral", Name: "Masculine — Neutral"},
		{Id: "voice.neutral.default", Name: "Neutral — Default"},
	}
	return connect.NewResponse(&ttsv1.ListVoicesResponse{Voices: voices}), nil
}

// NormalizeForSpeech runs the TTS-pipeline-specific text normalizer. Pure
// function; no provider chain.
func (h *connectHandler) NormalizeForSpeech(ctx context.Context, req *connect.Request[ttsv1.NormalizeForSpeechRequest]) (*connect.Response[ttsv1.NormalizeForSpeechResponse], error) {
	out := normalizer.NormalizeTextForSpeech(req.Msg.Text)
	return connect.NewResponse(&ttsv1.NormalizeForSpeechResponse{Text: out}), nil
}

// SplitParagraphs runs the speech-paragraph splitter. Pure function.
// MaxChars is currently advisory (the splitter uses internal heuristics);
// future expansion threads it through internal/tts.
func (h *connectHandler) SplitParagraphs(ctx context.Context, req *connect.Request[ttsv1.SplitParagraphsRequest]) (*connect.Response[ttsv1.SplitParagraphsResponse], error) {
	paragraphs := normalizer.SplitIntoSpeechParagraphs(req.Msg.Text)
	return connect.NewResponse(&ttsv1.SplitParagraphsResponse{Paragraphs: paragraphs}), nil
}
