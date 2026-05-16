// Package tts hosts the TTSService Connect-RPC handler.
package tts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"connectrpc.com/connect"

	"audio-tools/internal/ai/ttschain"
	intsumm "audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/store"
	inttts "audio-tools/internal/tts"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

// Deps wires the seams the TTS handler needs.
type Deps struct {
	Chain          *ttschain.Chain
	SummarizeChain *intsumm.Chain
	TTSService     *inttts.Service
	Logger         *log.Logger
	Cache          *inttts.Cache
	ConfigStore    *store.TTSConfigStore
	Playback       *store.PlaybackStore
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs a Connect handler. The Chain is required for
// Synthesize; admin methods (config/status/cache/playback) bind to their
// respective stores via the corresponding Deps fields.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// SynthesizeStream routes through the TTS provider chain's streaming
// path. When no tier declares streaming, the chain falls back to the
// unary Execute() and emits a single is_final=true frame — the wire
// shape is identical so consumers code against a stream uniformly.
func (h *connectHandler) SynthesizeStream(ctx context.Context, req *connect.Request[ttsv1.SynthesizeRequest], stream *connect.ServerStream[ttsv1.AudioFrame]) error {
	if h.deps.Chain == nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("tts chain not configured"))
	}
	creds := extractCreds(req.Header())
	chainReq := ttschain.Request{
		Text:           req.Msg.Text,
		Voice:          req.Msg.Voice,
		VoiceOverrides: req.Msg.VoiceOverrides,
		Speed:          req.Msg.Speed,
		ResponseFormat: req.Msg.ResponseFormat,
		BYOKProvider:   creds.byokProvider,
		BYOKKey:        creds.byokKey,
		LPBSToken:      creds.lpbsToken,
		UserIdentity:   creds.userIdentity,
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
			msg.ProviderTier = string(frame.Tier)
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
	creds := extractCreds(req.Header())
	chainReq := ttschain.Request{
		Text:           req.Msg.Text,
		Voice:          req.Msg.Voice,
		VoiceOverrides: req.Msg.VoiceOverrides,
		Speed:          req.Msg.Speed,
		ResponseFormat: req.Msg.ResponseFormat,
		BYOKProvider:   creds.byokProvider,
		BYOKKey:        creds.byokKey,
		LPBSToken:      creds.lpbsToken,
		UserIdentity:   creds.userIdentity,
		EventID:        req.Msg.EventId,
		Version:        req.Msg.Version,
	}
	res, err := h.deps.Chain.Execute(ctx, chainReq)
	if err != nil {
		return nil, mapChainError(err)
	}
	return connect.NewResponse(&ttsv1.SynthesizeResponse{
		Audio:        res.Audio,
		ContentType:  res.ContentType,
		ContentHash:  res.ContentHash,
		ProviderTier: string(res.Tier),
		ProviderId:   res.ProviderID,
		ModelId:      res.ModelID,
		VoiceUsed:    res.VoiceUsed,
		LatencyMs:    float64(res.Latency.Milliseconds()),
	}), nil
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
	out := inttts.NormalizeTextForSpeech(req.Msg.Text)
	return connect.NewResponse(&ttsv1.NormalizeForSpeechResponse{Text: out}), nil
}

// SplitParagraphs runs the speech-paragraph splitter. Pure function.
// MaxChars is currently advisory (the splitter uses internal heuristics);
// future expansion threads it through internal/tts.
func (h *connectHandler) SplitParagraphs(ctx context.Context, req *connect.Request[ttsv1.SplitParagraphsRequest]) (*connect.Response[ttsv1.SplitParagraphsResponse], error) {
	paragraphs := inttts.SplitIntoSpeechParagraphs(req.Msg.Text)
	return connect.NewResponse(&ttsv1.SplitParagraphsResponse{Paragraphs: paragraphs}), nil
}

// credentialSet bundles per-request audio-tools credentials extracted from
// metadata headers.
type credentialSet struct {
	byokProvider string
	byokKey      string
	lpbsToken    string
	userIdentity string
}

func extractCreds(h headerGetter) credentialSet {
	return credentialSet{
		byokProvider: h.Get("X-Audio-BYOK-Provider"),
		byokKey:      h.Get("X-Audio-BYOK-Key"),
		lpbsToken:    h.Get("X-Audio-LPBS-Token"),
		userIdentity: h.Get("X-Audio-User-Identity"),
	}
}

// headerGetter is the http.Header subset used by extractCreds; abstracted so
// tests can fake without an http.Header instance.
type headerGetter interface {
	Get(string) string
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

// stableContentHash is a helper used by future cache implementations to
// derive the content-addressable cache key.
func stableContentHash(voice string, speed float64, format string, text string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%f|%s|%s", voice, speed, format, text)
	return hex.EncodeToString(h.Sum(nil))
}
