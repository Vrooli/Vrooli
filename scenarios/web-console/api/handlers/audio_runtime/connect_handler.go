// Package audio_runtime is web-console's per-utterance audio surface
// (transcribe, synthesize, list voices, fetch cache, summarize, playback
// events). Same-origin from the UI; delegates to audioports.* for
// inter-scenario calls.
//
// Wire proto: packages/proto/schemas/web-console/v1/audio_runtime/audio_runtime.proto.
package audio_runtime

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	"web-console/integrations/audiotools"
	"web-console/internal/audioports"

	audioruntimev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_runtime"
)

// Deps is the seam the handler depends on.
type Deps struct {
	STT      audioports.SpeechToText
	TTS      audioports.TextToSpeech
	Playback audioports.PlaybackEventRecorder
	Summ     audioports.Summarizer
	Logger   *log.Logger
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

func mapErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, audiotools.ErrTimeout):
		return connect.NewError(connect.CodeDeadlineExceeded, err)
	case errors.Is(err, audiotools.ErrUnavailable):
		return connect.NewError(connect.CodeUnavailable, err)
	case errors.Is(err, audiotools.ErrFailedPrecondition):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, audiotools.ErrInsufficientCredits):
		return connect.NewError(connect.CodeResourceExhausted, err)
	case errors.Is(err, audiotools.ErrInvalidArgument):
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return connectErr
	}
	return connect.NewError(connect.CodeInternal, err)
}

// -----------------------------------------------------------------------------
// Transcribe
// -----------------------------------------------------------------------------

func (h *connectHandler) Transcribe(ctx context.Context, req *connect.Request[audioruntimev1.TranscribeRequest]) (*connect.Response[audioruntimev1.TranscribeResponse], error) {
	if h.deps.STT == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	if len(req.Msg.Audio) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("audio is required"))
	}
	out, err := h.deps.STT.Transcribe(ctx, req.Msg.Audio, audioports.STTOptions{
		Language:                req.Msg.Language,
		SkipSpeakerVerification: req.Msg.SkipSpeakerVerification,
		InitialPrompt:           req.Msg.InitialPrompt,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioruntimev1.TranscribeResponse{Text: out.Text}), nil
}

// -----------------------------------------------------------------------------
// Synthesize / Voices / Cache / Playback
// -----------------------------------------------------------------------------

// responseFormatStr maps the web-console-owned ResponseFormat enum to the
// legacy string ("mp3"/"wav"/...) accepted by audioports.TTSRequest.
func responseFormatStr(f int32) string {
	switch f {
	case 1:
		return "mp3"
	case 2:
		return "wav"
	case 3:
		return "opus"
	case 4:
		return "flac"
	default:
		return ""
	}
}

func (h *connectHandler) Synthesize(ctx context.Context, req *connect.Request[audioruntimev1.SynthesizeRequest]) (*connect.Response[audioruntimev1.SynthesizeResponse], error) {
	if h.deps.TTS == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	out, err := h.deps.TTS.Synthesize(ctx, audioports.TTSRequest{
		Input:          req.Msg.Text,
		Voice:          req.Msg.Voice,
		Speed:          req.Msg.Speed,
		ResponseFormat: responseFormatStr(int32(req.Msg.ResponseFormat)),
		EventID:        req.Msg.EventId,
		Version:        req.Msg.Version,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioruntimev1.SynthesizeResponse{
		Audio:       out.Audio,
		ContentType: out.ContentType,
	}), nil
}

func (h *connectHandler) ListVoices(ctx context.Context, _ *connect.Request[audioruntimev1.ListVoicesRequest]) (*connect.Response[audioruntimev1.ListVoicesResponse], error) {
	if h.deps.TTS == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	voices, err := h.deps.TTS.ListVoices(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*audioruntimev1.Voice, 0, len(voices))
	for _, v := range voices {
		out = append(out, &audioruntimev1.Voice{Id: v.ID, Name: v.Name})
	}
	return connect.NewResponse(&audioruntimev1.ListVoicesResponse{Voices: out}), nil
}

func (h *connectHandler) GetTTSCache(ctx context.Context, req *connect.Request[audioruntimev1.GetTTSCacheRequest]) (*connect.Response[audioruntimev1.GetTTSCacheResponse], error) {
	if h.deps.TTS == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	out, hit := h.deps.TTS.GetCached(ctx, audioports.CacheLookup{
		EventID: req.Msg.EventId,
		Voice:   req.Msg.Voice,
		Speed:   req.Msg.Speed,
		Version: req.Msg.Version,
	})
	if !hit {
		return connect.NewResponse(&audioruntimev1.GetTTSCacheResponse{Hit: false}), nil
	}
	return connect.NewResponse(&audioruntimev1.GetTTSCacheResponse{
		Audio:       out.Audio,
		ContentType: out.ContentType,
		Hit:         true,
	}), nil
}

func (h *connectHandler) RecordPlaybackEvent(ctx context.Context, req *connect.Request[audioruntimev1.RecordPlaybackEventRequest]) (*connect.Response[audioruntimev1.RecordPlaybackEventResponse], error) {
	if h.deps.Playback == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	if req.Msg.Event == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("event is required"))
	}
	err := h.deps.Playback.RecordPlaybackEvent(ctx, audioports.PlaybackEvent{
		Source:    req.Msg.Event.Source,
		Stage:     req.Msg.Event.Stage,
		Backend:   req.Msg.Event.Backend,
		SessionID: req.Msg.Event.SessionId,
		Message:   req.Msg.Event.Message,
		EventID:   req.Msg.Event.EventId,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioruntimev1.RecordPlaybackEventResponse{Status: "ok"}), nil
}

// -----------------------------------------------------------------------------
// Summarize
// -----------------------------------------------------------------------------

func summarizeLevelToString(l int32) string {
	switch l {
	case 1:
		return "light"
	case 2:
		return "moderate"
	case 3:
		return "heavy"
	default:
		return ""
	}
}

func (h *connectHandler) Summarize(ctx context.Context, req *connect.Request[audioruntimev1.SummarizeRequest]) (*connect.Response[audioruntimev1.SummarizeResponse], error) {
	if h.deps.Summ == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	if req.Msg.Text == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("text is required"))
	}
	out, err := h.deps.Summ.Summarize(ctx, audioports.SummarizeInput{
		Text:           req.Msg.Text,
		Level:          summarizeLevelToString(int32(req.Msg.Level)),
		Model:          req.Msg.Model,
		TimeoutSeconds: int(req.Msg.TimeoutSeconds),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioruntimev1.SummarizeResponse{
		Text:         out.Text,
		PromptTokens: int32(out.PromptTokens),
		OutputTokens: int32(out.OutputTokens),
	}), nil
}
