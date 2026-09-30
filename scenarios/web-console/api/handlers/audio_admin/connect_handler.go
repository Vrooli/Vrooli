package audio_admin

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	"web-console/integrations/audiotools"
	"web-console/internal/audioports"

	audioadminv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_admin"
)

// Deps is the seam the handler depends on. Each port is one of the
// audioports admin interfaces — the handler does no I/O of its own.
type Deps struct {
	StreamConfig    audioports.StreamConfigAdmin
	WakeWord        audioports.WakeWordAdmin
	Speaker         audioports.SpeakerAdmin
	TTSConfig       audioports.TTSConfigAdmin
	SummarizeConfig audioports.SummarizeConfigAdmin
	Logger          *log.Logger
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

// mapErr translates an audioports error to a typed *connect.Error.
// audio-tools error envelopes have already been normalized in
// audiotools.NormalizeError before reaching here.
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
	// Last-resort: pass through *connect.Error untouched, otherwise Internal.
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return connectErr
	}
	return connect.NewError(connect.CodeInternal, err)
}

// -----------------------------------------------------------------------------
// Stream config
// -----------------------------------------------------------------------------

func (h *connectHandler) GetStreamConfig(ctx context.Context, _ *connect.Request[audioadminv1.GetStreamConfigRequest]) (*connect.Response[audioadminv1.GetStreamConfigResponse], error) {
	if h.deps.StreamConfig == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	cfg, err := h.deps.StreamConfig.GetStreamConfig(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioadminv1.GetStreamConfigResponse{Config: streamConfigToProto(cfg)}), nil
}

func (h *connectHandler) UpdateStreamConfig(ctx context.Context, req *connect.Request[audioadminv1.UpdateStreamConfigRequest]) (*connect.Response[audioadminv1.UpdateStreamConfigResponse], error) {
	if h.deps.StreamConfig == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	mask := audioports.FieldMask{}
	if req.Msg.UpdateMask != nil {
		mask.Paths = append(mask.Paths, req.Msg.UpdateMask.Paths...)
	}
	if len(mask.Paths) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask is required"))
	}
	cfg, err := h.deps.StreamConfig.UpdateStreamConfig(ctx, mask, streamConfigFromProto(req.Msg.Config))
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioadminv1.UpdateStreamConfigResponse{Config: streamConfigToProto(cfg)}), nil
}

// -----------------------------------------------------------------------------
// Wake word
// -----------------------------------------------------------------------------

func (h *connectHandler) GetWakeWordConfig(ctx context.Context, _ *connect.Request[audioadminv1.GetWakeWordConfigRequest]) (*connect.Response[audioadminv1.GetWakeWordConfigResponse], error) {
	if h.deps.WakeWord == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	cfg, err := h.deps.WakeWord.GetWakeWordConfig(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioadminv1.GetWakeWordConfigResponse{Config: wakeWordConfigToProto(cfg)}), nil
}

func (h *connectHandler) UpdateWakeWordTemplate(ctx context.Context, req *connect.Request[audioadminv1.UpdateWakeWordTemplateRequest]) (*connect.Response[audioadminv1.UpdateWakeWordTemplateResponse], error) {
	if h.deps.WakeWord == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	tmpl := wakeWordTemplateFromProto(req.Msg.Template)
	cfg, err := h.deps.WakeWord.UpdateWakeWordTemplate(ctx, tmpl)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioadminv1.UpdateWakeWordTemplateResponse{Config: wakeWordConfigToProto(cfg)}), nil
}

func (h *connectHandler) DeleteWakeWordTemplate(ctx context.Context, _ *connect.Request[audioadminv1.DeleteWakeWordTemplateRequest]) (*connect.Response[audioadminv1.DeleteWakeWordTemplateResponse], error) {
	if h.deps.WakeWord == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	cfg, err := h.deps.WakeWord.DeleteWakeWordTemplate(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioadminv1.DeleteWakeWordTemplateResponse{Config: wakeWordConfigToProto(cfg)}), nil
}

// -----------------------------------------------------------------------------
// Speaker
// -----------------------------------------------------------------------------

func (h *connectHandler) GetSpeakerConfig(ctx context.Context, _ *connect.Request[audioadminv1.GetSpeakerConfigRequest]) (*connect.Response[audioadminv1.GetSpeakerConfigResponse], error) {
	if h.deps.Speaker == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	cfg, err := h.deps.Speaker.GetSpeakerConfig(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioadminv1.GetSpeakerConfigResponse{Config: speakerConfigToProto(cfg)}), nil
}

func (h *connectHandler) UpdateSpeakerConfig(ctx context.Context, req *connect.Request[audioadminv1.UpdateSpeakerConfigRequest]) (*connect.Response[audioadminv1.UpdateSpeakerConfigResponse], error) {
	if h.deps.Speaker == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	mask := audioports.FieldMask{}
	if req.Msg.UpdateMask != nil {
		mask.Paths = append(mask.Paths, req.Msg.UpdateMask.Paths...)
	}
	if len(mask.Paths) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask is required"))
	}
	cfg, err := h.deps.Speaker.UpdateSpeakerConfig(ctx, mask, speakerConfigFromProto(req.Msg.Config))
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioadminv1.UpdateSpeakerConfigResponse{Config: speakerConfigToProto(cfg)}), nil
}

func (h *connectHandler) GetSpeakerStatus(ctx context.Context, _ *connect.Request[audioadminv1.GetSpeakerStatusRequest]) (*connect.Response[audioadminv1.GetSpeakerStatusResponse], error) {
	if h.deps.Speaker == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	s, err := h.deps.Speaker.GetSpeakerStatus(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioadminv1.GetSpeakerStatusResponse{Status: speakerStatusToProto(s)}), nil
}

func (h *connectHandler) ListSpeakerProfiles(ctx context.Context, _ *connect.Request[audioadminv1.ListSpeakerProfilesRequest]) (*connect.Response[audioadminv1.ListSpeakerProfilesResponse], error) {
	if h.deps.Speaker == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	profiles, err := h.deps.Speaker.ListSpeakerProfiles(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*audioadminv1.SpeakerProfile, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, speakerProfileToProto(p))
	}
	return connect.NewResponse(&audioadminv1.ListSpeakerProfilesResponse{
		Profiles: out,
		Count:    int32(len(out)),
	}), nil
}

func (h *connectHandler) EnrollSpeakerProfile(ctx context.Context, req *connect.Request[audioadminv1.EnrollSpeakerProfileRequest]) (*connect.Response[audioadminv1.EnrollSpeakerProfileResponse], error) {
	if h.deps.Speaker == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	in := audioports.EnrollSpeakerInput{
		Audio:       req.Msg.Audio,
		Format:      audioports.AudioFormat(req.Msg.Format),
		ProfileID:   req.Msg.ProfileId,
		DisplayName: req.Msg.DisplayName,
		Notes:       req.Msg.Notes,
	}
	if req.Msg.AddToActive != nil {
		v := *req.Msg.AddToActive
		in.AddToActive = &v
	}
	if req.Msg.Enable != nil {
		v := *req.Msg.Enable
		in.Enable = &v
	}
	out, err := h.deps.Speaker.EnrollSpeakerProfile(ctx, in)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioadminv1.EnrollSpeakerProfileResponse{
		Enrollment: speakerEnrollmentToProto(out.Enrollment),
		Config:     speakerConfigToProto(out.Config),
	}), nil
}

func (h *connectHandler) ClearSpeakerProfileBinding(ctx context.Context, _ *connect.Request[audioadminv1.ClearSpeakerProfileBindingRequest]) (*connect.Response[audioadminv1.ClearSpeakerProfileBindingResponse], error) {
	if h.deps.Speaker == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	cfg, err := h.deps.Speaker.ClearSpeakerProfileBinding(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioadminv1.ClearSpeakerProfileBindingResponse{Config: speakerConfigToProto(cfg)}), nil
}

func (h *connectHandler) UnbindSpeakerProfile(ctx context.Context, req *connect.Request[audioadminv1.UnbindSpeakerProfileRequest]) (*connect.Response[audioadminv1.UnbindSpeakerProfileResponse], error) {
	if h.deps.Speaker == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	if req.Msg.ProfileId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("profile_id is required"))
	}
	cfg, err := h.deps.Speaker.UnbindSpeakerProfile(ctx, req.Msg.ProfileId)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioadminv1.UnbindSpeakerProfileResponse{Config: speakerConfigToProto(cfg)}), nil
}

func (h *connectHandler) DeleteSpeakerProfile(ctx context.Context, req *connect.Request[audioadminv1.DeleteSpeakerProfileRequest]) (*connect.Response[audioadminv1.DeleteSpeakerProfileResponse], error) {
	if h.deps.Speaker == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	if req.Msg.ProfileId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("profile_id is required"))
	}
	cfg, err := h.deps.Speaker.DeleteSpeakerProfile(ctx, req.Msg.ProfileId)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioadminv1.DeleteSpeakerProfileResponse{Config: speakerConfigToProto(cfg)}), nil
}

// -----------------------------------------------------------------------------
// TTS config
// -----------------------------------------------------------------------------

func (h *connectHandler) GetTTSConfig(ctx context.Context, _ *connect.Request[audioadminv1.GetTTSConfigRequest]) (*connect.Response[audioadminv1.GetTTSConfigResponse], error) {
	if h.deps.TTSConfig == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	cfg, err := h.deps.TTSConfig.GetTTSConfig(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioadminv1.GetTTSConfigResponse{Config: ttsConfigToProto(cfg)}), nil
}

func (h *connectHandler) UpdateTTSConfig(ctx context.Context, req *connect.Request[audioadminv1.UpdateTTSConfigRequest]) (*connect.Response[audioadminv1.UpdateTTSConfigResponse], error) {
	if h.deps.TTSConfig == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	mask := audioports.FieldMask{}
	if req.Msg.UpdateMask != nil {
		mask.Paths = append(mask.Paths, req.Msg.UpdateMask.Paths...)
	}
	if len(mask.Paths) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask is required"))
	}
	cfg, err := h.deps.TTSConfig.UpdateTTSConfig(ctx, mask, ttsConfigFromProto(req.Msg.Config))
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioadminv1.UpdateTTSConfigResponse{Config: ttsConfigToProto(cfg)}), nil
}

// -----------------------------------------------------------------------------
// Summarize config
// -----------------------------------------------------------------------------

func (h *connectHandler) GetSummarizeConfig(ctx context.Context, _ *connect.Request[audioadminv1.GetSummarizeConfigRequest]) (*connect.Response[audioadminv1.GetSummarizeConfigResponse], error) {
	if h.deps.SummarizeConfig == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	cfg, err := h.deps.SummarizeConfig.GetSummarizeConfig(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioadminv1.GetSummarizeConfigResponse{Config: summarizeConfigToProto(cfg)}), nil
}

func (h *connectHandler) UpdateSummarizeConfig(ctx context.Context, req *connect.Request[audioadminv1.UpdateSummarizeConfigRequest]) (*connect.Response[audioadminv1.UpdateSummarizeConfigResponse], error) {
	if h.deps.SummarizeConfig == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	mask := audioports.FieldMask{}
	if req.Msg.UpdateMask != nil {
		mask.Paths = append(mask.Paths, req.Msg.UpdateMask.Paths...)
	}
	if len(mask.Paths) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask is required"))
	}
	cfg, err := h.deps.SummarizeConfig.UpdateSummarizeConfig(ctx, mask, summarizeConfigFromProto(req.Msg.Config))
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&audioadminv1.UpdateSummarizeConfigResponse{Config: summarizeConfigToProto(cfg)}), nil
}

func (h *connectHandler) ListSummarizeModels(ctx context.Context, _ *connect.Request[audioadminv1.ListSummarizeModelsRequest]) (*connect.Response[audioadminv1.ListSummarizeModelsResponse], error) {
	if h.deps.SummarizeConfig == nil {
		return nil, connect.NewError(connect.CodeUnavailable, audiotools.ErrUnavailable)
	}
	models, err := h.deps.SummarizeConfig.ListSummarizeModels(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*audioadminv1.SummarizeModel, 0, len(models))
	for _, model := range models {
		out = append(out, summarizeModelToProto(model))
	}
	return connect.NewResponse(&audioadminv1.ListSummarizeModelsResponse{Models: out}), nil
}
