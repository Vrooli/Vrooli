package tts

import (
	"context"
	"errors"
	"log"
	"strings"

	"connectrpc.com/connect"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/tts"
)

// Deps wires the seams the Connect TTS handler needs.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC handler implementing
// TTSServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// Sentinel errors mapped via classify(). The legacy handlers returned the
// catalog codes "not_configured", "tts_unavailable", "tts_input_required",
// "tts_input_too_long", "tts_invalid_format", "tts_synthesis_failed",
// "tts_voice_list_failed", "tts_cache_missing_id", and "invalid_body".
// Those collapse to a small set of Connect codes:
var (
	ErrNotFound          = errors.New("not found")
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrFailedPrecondition = errors.New("failed precondition")
	ErrUnavailable       = errors.New("tts unavailable")
	ErrInternal          = errors.New("internal error")
)

func (h *connectHandler) GetConfig(ctx context.Context, _ *connect.Request[ttsv1.GetConfigRequest]) (*connect.Response[ttsv1.GetConfigResponse], error) {
	cfg, err := h.deps.Service.GetConfig(ctx)
	if err != nil {
		return nil, h.classify(err, "tts.GetConfig")
	}
	return connect.NewResponse(&ttsv1.GetConfigResponse{Config: configToProto(cfg)}), nil
}

func (h *connectHandler) UpdateConfig(ctx context.Context, req *connect.Request[ttsv1.UpdateConfigRequest]) (*connect.Response[ttsv1.UpdateConfigResponse], error) {
	patch := ConfigPatch{}
	if req.Msg.GetHasAutoEnabled() {
		v := req.Msg.GetAutoEnabled()
		patch.AutoEnabled = &v
	}
	if req.Msg.GetHasBackend() {
		v := req.Msg.GetBackend()
		patch.Backend = &v
	}
	if req.Msg.GetHasKokoroVoice() {
		v := req.Msg.GetKokoroVoice()
		patch.KokoroVoice = &v
	}
	if req.Msg.GetHasKokoroSpeed() {
		v := req.Msg.GetKokoroSpeed()
		patch.KokoroSpeed = &v
	}
	cfg, err := h.deps.Service.UpdateConfig(ctx, patch)
	if err != nil {
		return nil, h.classify(err, "tts.UpdateConfig")
	}
	return connect.NewResponse(&ttsv1.UpdateConfigResponse{Config: configToProto(cfg)}), nil
}

func (h *connectHandler) GetStatus(ctx context.Context, _ *connect.Request[ttsv1.GetStatusRequest]) (*connect.Response[ttsv1.GetStatusResponse], error) {
	st, err := h.deps.Service.GetStatus(ctx)
	if err != nil {
		return nil, h.classify(err, "tts.GetStatus")
	}
	return connect.NewResponse(&ttsv1.GetStatusResponse{Status: statusToProto(st)}), nil
}

func (h *connectHandler) RecordPlaybackEvent(ctx context.Context, req *connect.Request[ttsv1.RecordPlaybackEventRequest]) (*connect.Response[ttsv1.RecordPlaybackEventResponse], error) {
	pe := req.Msg.GetEvent()
	if pe == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("event is required"))
	}
	src := strings.TrimSpace(pe.GetSource())
	stage := strings.TrimSpace(pe.GetStage())
	if src == "" || stage == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("source and stage are required"))
	}
	ev := PlaybackEvent{
		Source:    src,
		Stage:     stage,
		Backend:   pe.GetBackend(),
		SessionID: pe.GetSessionId(),
		Message:   pe.GetMessage(),
	}
	if err := h.deps.Service.RecordPlaybackEvent(ctx, ev); err != nil {
		return nil, h.classify(err, "tts.RecordPlaybackEvent")
	}
	return connect.NewResponse(&ttsv1.RecordPlaybackEventResponse{Status: "ok"}), nil
}

func (h *connectHandler) GetSummarizeConfig(ctx context.Context, _ *connect.Request[ttsv1.GetSummarizeConfigRequest]) (*connect.Response[ttsv1.GetSummarizeConfigResponse], error) {
	cfg, err := h.deps.Service.GetSummarizeConfig(ctx)
	if err != nil {
		return nil, h.classify(err, "tts.GetSummarizeConfig")
	}
	return connect.NewResponse(&ttsv1.GetSummarizeConfigResponse{Config: summarizeToProto(cfg)}), nil
}

func (h *connectHandler) UpdateSummarizeConfig(ctx context.Context, req *connect.Request[ttsv1.UpdateSummarizeConfigRequest]) (*connect.Response[ttsv1.UpdateSummarizeConfigResponse], error) {
	patch := SummarizeConfigPatch{}
	if req.Msg.GetHasEnabled() {
		v := req.Msg.GetEnabled()
		patch.Enabled = &v
	}
	if req.Msg.GetHasCharThreshold() {
		v := int(req.Msg.GetCharThreshold())
		patch.CharThreshold = &v
	}
	if req.Msg.GetHasLevel() {
		v := req.Msg.GetLevel()
		patch.Level = &v
	}
	if req.Msg.GetHasModel() {
		v := req.Msg.GetModel()
		patch.Model = &v
	}
	if req.Msg.GetHasTimeoutSeconds() {
		v := int(req.Msg.GetTimeoutSeconds())
		patch.TimeoutSeconds = &v
	}
	cfg, err := h.deps.Service.UpdateSummarizeConfig(ctx, patch)
	if err != nil {
		return nil, h.classify(err, "tts.UpdateSummarizeConfig")
	}
	return connect.NewResponse(&ttsv1.UpdateSummarizeConfigResponse{Config: summarizeToProto(cfg)}), nil
}

func (h *connectHandler) Synthesize(ctx context.Context, req *connect.Request[ttsv1.SynthesizeRequest]) (*connect.Response[ttsv1.SynthesizeResponse], error) {
	in := SynthesizeInput{
		Input:          req.Msg.GetInput(),
		Voice:          req.Msg.GetVoice(),
		ResponseFormat: req.Msg.GetResponseFormat(),
		Speed:          req.Msg.GetSpeed(),
		EventID:        req.Msg.GetEventId(),
		Version:        req.Msg.GetVersion(),
	}
	res, err := h.deps.Service.Synthesize(ctx, in)
	if err != nil {
		return nil, h.classify(err, "tts.Synthesize")
	}
	return connect.NewResponse(&ttsv1.SynthesizeResponse{Audio: res.Audio, ContentType: res.ContentType}), nil
}

func (h *connectHandler) GetCache(ctx context.Context, req *connect.Request[ttsv1.GetCacheRequest]) (*connect.Response[ttsv1.GetCacheResponse], error) {
	id := strings.TrimSpace(req.Msg.GetEventId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("event_id is required"))
	}
	res, err := h.deps.Service.GetCache(ctx, CacheLookup{
		EventID: id,
		Voice:   req.Msg.GetVoice(),
		Speed:   req.Msg.GetSpeed(),
		Version: req.Msg.GetVersion(),
	})
	if err != nil {
		return nil, h.classify(err, "tts.GetCache")
	}
	return connect.NewResponse(&ttsv1.GetCacheResponse{Audio: res.Audio, ContentType: res.ContentType}), nil
}

func (h *connectHandler) ListVoices(ctx context.Context, _ *connect.Request[ttsv1.ListVoicesRequest]) (*connect.Response[ttsv1.ListVoicesResponse], error) {
	out, err := h.deps.Service.ListVoices(ctx)
	if err != nil {
		return nil, h.classify(err, "tts.ListVoices")
	}
	voices := make([]*ttsv1.Voice, 0, len(out))
	for _, v := range out {
		voices = append(voices, &ttsv1.Voice{Id: v.ID, Name: v.Name})
	}
	return connect.NewResponse(&ttsv1.ListVoicesResponse{Voices: voices}), nil
}

func (h *connectHandler) classify(err error, op string) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrInvalidArgument):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, ErrFailedPrecondition):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, ErrUnavailable):
		return connect.NewError(connect.CodeUnavailable, err)
	case errors.Is(err, ErrInternal):
		h.deps.Logger.Printf("%s: %v", op, err)
		return connect.NewError(connect.CodeInternal, err)
	default:
		h.deps.Logger.Printf("%s: %v", op, err)
		return connect.NewError(connect.CodeInternal, err)
	}
}

// -----------------------------------------------------------------------------
// proto helpers
// -----------------------------------------------------------------------------

func configToProto(c Config) *ttsv1.Config {
	return &ttsv1.Config{
		AutoEnabled: c.AutoEnabled,
		Backend:     c.Backend,
		KokoroVoice: c.KokoroVoice,
		KokoroSpeed: c.KokoroSpeed,
	}
}

func summarizeToProto(c SummarizeConfig) *ttsv1.SummarizeConfig {
	return &ttsv1.SummarizeConfig{
		Enabled:        c.Enabled,
		CharThreshold:  int32(c.CharThreshold),
		Level:          c.Level,
		Model:          c.Model,
		TimeoutSeconds: int32(c.TimeoutSeconds),
	}
}

func appendResultToProto(a *AppendResult) *ttsv1.AppendResult {
	if a == nil {
		return nil
	}
	return &ttsv1.AppendResult{
		Appended:  a.Appended,
		Code:      a.Code,
		Reason:    a.Reason,
		Source:    a.Source,
		SessionId: a.SessionID,
		EventId:   a.EventID,
		Sequence:  a.Sequence,
		Duplicate: a.Duplicate,
	}
}

func ackToProto(a *ClientAck) *ttsv1.ClientAck {
	if a == nil {
		return nil
	}
	return &ttsv1.ClientAck{
		EventId:   a.EventID,
		Source:    a.Source,
		SessionId: a.SessionID,
		Stage:     a.Stage,
		Backend:   a.Backend,
		Message:   a.Message,
	}
}

func playbackToProto(p *PlaybackEvent) *ttsv1.PlaybackEvent {
	if p == nil {
		return nil
	}
	return &ttsv1.PlaybackEvent{
		Source:    p.Source,
		Stage:     p.Stage,
		Backend:   p.Backend,
		SessionId: p.SessionID,
		Message:   p.Message,
	}
}

func statusToProto(s Status) *ttsv1.Status {
	return &ttsv1.Status{
		Config:                configToProto(s.Config),
		HookRegistered:        s.HookRegistered,
		HookCode:              s.HookCode,
		HookReason:            s.HookReason,
		HookSettingsPath:      s.HookSettingsPath,
		LastRouting:           appendResultToProto(s.LastRouting),
		LastRoutingAt:         s.LastRoutingAt,
		LastHookRouting:       appendResultToProto(s.LastHookRouting),
		LastHookRoutingAt:     s.LastHookRoutingAt,
		LastTailerRouting:     appendResultToProto(s.LastTailerRouting),
		LastTailerRoutingAt:   s.LastTailerRoutingAt,
		LastAck:               ackToProto(s.LastAck),
		LastAckAt:             s.LastAckAt,
		LastHookAck:           ackToProto(s.LastHookAck),
		LastHookAckAt:         s.LastHookAckAt,
		LastTailerAck:         ackToProto(s.LastTailerAck),
		LastTailerAckAt:       s.LastTailerAckAt,
		LastPlaybackEvent:     playbackToProto(s.LastPlaybackEvent),
		LastPlaybackAt:        s.LastPlaybackAt,
		KokoroCapability:      s.KokoroCapability,
		KokoroCapabilityLabel: s.KokoroCapabilityLabel,
	}
}
