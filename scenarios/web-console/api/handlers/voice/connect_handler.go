package voice

import (
	"context"
	"errors"
	"log"
	"strings"

	"connectrpc.com/connect"

	voicev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/voice"
)

// Deps wires the seams the Connect Voice handler needs.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC handler implementing
// VoiceServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Transcribe(ctx context.Context, req *connect.Request[voicev1.TranscribeRequest]) (*connect.Response[voicev1.TranscribeResponse], error) {
	in := TranscribeInput{
		Audio:                   req.Msg.GetAudio(),
		ContentType:             req.Msg.GetContentType(),
		Language:                req.Msg.GetLanguage(),
		SkipSpeakerVerification: req.Msg.GetSkipSpeakerVerification(),
	}
	text, err := h.deps.Service.Transcribe(ctx, in)
	if err != nil {
		return nil, h.classify(err, "voice.Transcribe")
	}
	return connect.NewResponse(&voicev1.TranscribeResponse{Text: text}), nil
}

func (h *connectHandler) GetStreamConfig(ctx context.Context, _ *connect.Request[voicev1.GetStreamConfigRequest]) (*connect.Response[voicev1.GetStreamConfigResponse], error) {
	cfg, err := h.deps.Service.GetStreamConfig(ctx)
	if err != nil {
		return nil, h.classify(err, "voice.GetStreamConfig")
	}
	return connect.NewResponse(&voicev1.GetStreamConfigResponse{Config: streamToProto(cfg)}), nil
}

func (h *connectHandler) UpdateStreamConfig(ctx context.Context, req *connect.Request[voicev1.UpdateStreamConfigRequest]) (*connect.Response[voicev1.UpdateStreamConfigResponse], error) {
	patch := StreamConfigPatch{}
	if req.Msg.GetHasFlushIntervalMs() {
		v := int(req.Msg.GetFlushIntervalMs())
		patch.FlushIntervalMs = &v
	}
	if req.Msg.GetHasMinDeltaBytes() {
		v := int(req.Msg.GetMinDeltaBytes())
		patch.MinDeltaBytes = &v
	}
	if req.Msg.GetHasOverlapBytes() {
		v := int(req.Msg.GetOverlapBytes())
		patch.OverlapBytes = &v
	}
	if req.Msg.GetHasPersistentMode() {
		v := req.Msg.GetPersistentMode()
		patch.PersistentMode = &v
	}
	if req.Msg.GetHasWakeWordEnabled() {
		v := req.Msg.GetWakeWordEnabled()
		patch.WakeWordEnabled = &v
	}
	if req.Msg.GetHasWakeWordThreshold() {
		v := req.Msg.GetWakeWordThreshold()
		patch.WakeWordThreshold = &v
	}
	if req.Msg.GetHasSegmentSilenceMs() {
		v := int(req.Msg.GetSegmentSilenceMs())
		patch.SegmentSilenceMs = &v
	}
	cfg, err := h.deps.Service.UpdateStreamConfig(ctx, patch)
	if err != nil {
		return nil, h.classify(err, "voice.UpdateStreamConfig")
	}
	return connect.NewResponse(&voicev1.UpdateStreamConfigResponse{Config: streamToProto(cfg)}), nil
}

func (h *connectHandler) GetWakeWordConfig(ctx context.Context, _ *connect.Request[voicev1.GetWakeWordConfigRequest]) (*connect.Response[voicev1.GetWakeWordConfigResponse], error) {
	cfg, err := h.deps.Service.GetWakeWordConfig(ctx)
	if err != nil {
		return nil, h.classify(err, "voice.GetWakeWordConfig")
	}
	return connect.NewResponse(&voicev1.GetWakeWordConfigResponse{Config: wakeWordToProto(cfg)}), nil
}

func (h *connectHandler) UpdateWakeWordTemplate(ctx context.Context, req *connect.Request[voicev1.UpdateWakeWordTemplateRequest]) (*connect.Response[voicev1.UpdateWakeWordTemplateResponse], error) {
	tj := strings.TrimSpace(req.Msg.GetTemplateJson())
	if tj == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("template_json is required"))
	}
	cfg, err := h.deps.Service.UpdateWakeWordTemplate(ctx, tj)
	if err != nil {
		return nil, h.classify(err, "voice.UpdateWakeWordTemplate")
	}
	return connect.NewResponse(&voicev1.UpdateWakeWordTemplateResponse{Config: wakeWordToProto(cfg)}), nil
}

func (h *connectHandler) DeleteWakeWordTemplate(ctx context.Context, _ *connect.Request[voicev1.DeleteWakeWordTemplateRequest]) (*connect.Response[voicev1.DeleteWakeWordTemplateResponse], error) {
	cfg, err := h.deps.Service.DeleteWakeWordTemplate(ctx)
	if err != nil {
		return nil, h.classify(err, "voice.DeleteWakeWordTemplate")
	}
	return connect.NewResponse(&voicev1.DeleteWakeWordTemplateResponse{Config: wakeWordToProto(cfg)}), nil
}

func (h *connectHandler) GetSpeakerConfig(ctx context.Context, _ *connect.Request[voicev1.GetSpeakerConfigRequest]) (*connect.Response[voicev1.GetSpeakerConfigResponse], error) {
	cfg, err := h.deps.Service.GetSpeakerConfig(ctx)
	if err != nil {
		return nil, h.classify(err, "voice.GetSpeakerConfig")
	}
	return connect.NewResponse(&voicev1.GetSpeakerConfigResponse{Config: speakerConfigToProto(cfg)}), nil
}

func (h *connectHandler) UpdateSpeakerConfig(ctx context.Context, req *connect.Request[voicev1.UpdateSpeakerConfigRequest]) (*connect.Response[voicev1.UpdateSpeakerConfigResponse], error) {
	patch := SpeakerConfigPatch{}
	if req.Msg.GetHasEnabled() {
		v := req.Msg.GetEnabled()
		patch.Enabled = &v
	}
	if req.Msg.GetHasProfileIds() {
		v := append([]string(nil), req.Msg.GetProfileIds()...)
		patch.ProfileIDs = &v
	}
	if req.Msg.GetHasThreshold() {
		v := req.Msg.GetThreshold()
		patch.Threshold = &v
	}
	if req.Msg.GetHasMode() {
		v := req.Msg.GetMode()
		patch.Mode = &v
	}
	if req.Msg.GetHasRejectBehavior() {
		v := req.Msg.GetRejectBehavior()
		patch.RejectBehavior = &v
	}
	if req.Msg.GetHasFallbackWithoutVerification() {
		v := req.Msg.GetFallbackWithoutVerification()
		patch.FallbackWithoutVerification = &v
	}
	if req.Msg.GetHasExtractionEnabled() {
		v := req.Msg.GetExtractionEnabled()
		patch.ExtractionEnabled = &v
	}
	cfg, err := h.deps.Service.UpdateSpeakerConfig(ctx, patch)
	if err != nil {
		return nil, h.classify(err, "voice.UpdateSpeakerConfig")
	}
	return connect.NewResponse(&voicev1.UpdateSpeakerConfigResponse{Config: speakerConfigToProto(cfg)}), nil
}

func (h *connectHandler) GetSpeakerStatus(ctx context.Context, _ *connect.Request[voicev1.GetSpeakerStatusRequest]) (*connect.Response[voicev1.GetSpeakerStatusResponse], error) {
	st, err := h.deps.Service.GetSpeakerStatus(ctx)
	if err != nil {
		return nil, h.classify(err, "voice.GetSpeakerStatus")
	}
	return connect.NewResponse(&voicev1.GetSpeakerStatusResponse{Status: speakerStatusToProto(st)}), nil
}

func (h *connectHandler) ListSpeakerProfiles(ctx context.Context, _ *connect.Request[voicev1.ListSpeakerProfilesRequest]) (*connect.Response[voicev1.ListSpeakerProfilesResponse], error) {
	profiles, count, err := h.deps.Service.ListSpeakerProfiles(ctx)
	if err != nil {
		return nil, h.classify(err, "voice.ListSpeakerProfiles")
	}
	out := make([]*voicev1.SpeakerProfile, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, profileToProto(p))
	}
	return connect.NewResponse(&voicev1.ListSpeakerProfilesResponse{Profiles: out, Count: int32(count)}), nil
}

func (h *connectHandler) EnrollSpeakerProfile(ctx context.Context, req *connect.Request[voicev1.EnrollSpeakerProfileRequest]) (*connect.Response[voicev1.EnrollSpeakerProfileResponse], error) {
	if len(req.Msg.GetAudio()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("audio is required"))
	}
	in := EnrollInput{
		Audio:       req.Msg.GetAudio(),
		ContentType: req.Msg.GetContentType(),
		ProfileID:   strings.TrimSpace(req.Msg.GetProfileId()),
		DisplayName: strings.TrimSpace(req.Msg.GetDisplayName()),
		Notes:       strings.TrimSpace(req.Msg.GetNotes()),
	}
	if req.Msg.GetHasAddToActive() {
		v := req.Msg.GetAddToActive()
		in.AddToActive = &v
	}
	if req.Msg.GetHasEnable() {
		v := req.Msg.GetEnable()
		in.Enable = &v
	}
	enrollment, cfg, err := h.deps.Service.EnrollSpeakerProfile(ctx, in)
	if err != nil {
		return nil, h.classify(err, "voice.EnrollSpeakerProfile")
	}
	return connect.NewResponse(&voicev1.EnrollSpeakerProfileResponse{
		Enrollment: enrollmentToProto(enrollment),
		Config:     speakerConfigToProto(cfg),
	}), nil
}

func (h *connectHandler) ClearSpeakerProfileBinding(ctx context.Context, _ *connect.Request[voicev1.ClearSpeakerProfileBindingRequest]) (*connect.Response[voicev1.ClearSpeakerProfileBindingResponse], error) {
	cfg, err := h.deps.Service.ClearSpeakerProfileBinding(ctx)
	if err != nil {
		return nil, h.classify(err, "voice.ClearSpeakerProfileBinding")
	}
	return connect.NewResponse(&voicev1.ClearSpeakerProfileBindingResponse{Config: speakerConfigToProto(cfg)}), nil
}

func (h *connectHandler) RemoveSpeakerProfile(ctx context.Context, req *connect.Request[voicev1.RemoveSpeakerProfileRequest]) (*connect.Response[voicev1.RemoveSpeakerProfileResponse], error) {
	id := strings.TrimSpace(req.Msg.GetProfileId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("profile_id is required"))
	}
	cfg, err := h.deps.Service.RemoveSpeakerProfile(ctx, id)
	if err != nil {
		return nil, h.classify(err, "voice.RemoveSpeakerProfile")
	}
	return connect.NewResponse(&voicev1.RemoveSpeakerProfileResponse{Config: speakerConfigToProto(cfg)}), nil
}

func (h *connectHandler) DeleteSpeakerProfile(ctx context.Context, req *connect.Request[voicev1.DeleteSpeakerProfileRequest]) (*connect.Response[voicev1.DeleteSpeakerProfileResponse], error) {
	id := strings.TrimSpace(req.Msg.GetProfileId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("profile_id is required"))
	}
	cfg, err := h.deps.Service.DeleteSpeakerProfile(ctx, id)
	if err != nil {
		return nil, h.classify(err, "voice.DeleteSpeakerProfile")
	}
	return connect.NewResponse(&voicev1.DeleteSpeakerProfileResponse{Config: speakerConfigToProto(cfg)}), nil
}

func (h *connectHandler) classify(err error, op string) error {
	switch {
	case errors.Is(err, ErrInvalidArgument):
		return connect.NewError(connect.CodeInvalidArgument, err)
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

func streamToProto(c StreamConfig) *voicev1.StreamConfig {
	return &voicev1.StreamConfig{
		FlushIntervalMs:   int32(c.FlushIntervalMs),
		MinDeltaBytes:     int32(c.MinDeltaBytes),
		OverlapBytes:      int32(c.OverlapBytes),
		PersistentMode:    c.PersistentMode,
		WakeWordEnabled:   c.WakeWordEnabled,
		WakeWordThreshold: c.WakeWordThreshold,
		SegmentSilenceMs:  int32(c.SegmentSilenceMs),
	}
}

func wakeWordToProto(c WakeWordConfig) *voicev1.WakeWordConfig {
	return &voicev1.WakeWordConfig{
		Configured:   c.Configured,
		TemplateJson: c.TemplateJSON,
	}
}

func speakerConfigToProto(c SpeakerConfig) *voicev1.SpeakerConfig {
	return &voicev1.SpeakerConfig{
		Enabled:                     c.Enabled,
		ProfileIds:                  append([]string(nil), c.ProfileIDs...),
		Threshold:                   c.Threshold,
		Mode:                        c.Mode,
		RejectBehavior:              c.RejectBehavior,
		FallbackWithoutVerification: c.FallbackWithoutVerification,
		ExtractionEnabled:           c.ExtractionEnabled,
	}
}

func profileToProto(p SpeakerProfile) *voicev1.SpeakerProfile {
	return &voicev1.SpeakerProfile{
		Id:                     p.ID,
		DisplayName:            p.DisplayName,
		CreatedAt:              p.CreatedAt,
		UpdatedAt:              p.UpdatedAt,
		ModelName:              p.ModelName,
		EmbeddingDim:           int32(p.EmbeddingDim),
		SampleRate:             int32(p.SampleRate),
		EnrollmentAudioSeconds: p.EnrollmentAudioSeconds,
		Notes:                  p.Notes,
	}
}

func infoToProto(i *SpeakerResourceInfo) *voicev1.SpeakerResourceInfo {
	if i == nil {
		return nil
	}
	return &voicev1.SpeakerResourceInfo{
		Backend:      i.Backend,
		Model:        i.Model,
		Device:       i.Device,
		SampleRate:   int32(i.SampleRate),
		Version:      i.Version,
		EmbeddingDim: int32(i.EmbeddingDim),
	}
}

func enrollmentToProto(e SpeakerEnrollment) *voicev1.SpeakerEnrollment {
	return &voicev1.SpeakerEnrollment{
		ProfileId:              e.ProfileID,
		DisplayName:            e.DisplayName,
		EmbeddingDim:           int32(e.EmbeddingDim),
		SampleRate:             int32(e.SampleRate),
		EnrollmentAudioSeconds: e.EnrollmentAudioSeconds,
		ModelName:              e.ModelName,
		CreatedAt:              e.CreatedAt,
	}
}

func speakerStatusToProto(s SpeakerStatus) *voicev1.SpeakerStatus {
	profiles := make([]*voicev1.SpeakerProfile, 0, len(s.Profiles))
	for _, p := range s.Profiles {
		profiles = append(profiles, profileToProto(p))
	}
	return &voicev1.SpeakerStatus{
		Config:            speakerConfigToProto(s.Config),
		Capability:        s.Capability,
		CapabilityLabel:   s.CapabilityLabel,
		ResourceReady:     s.ResourceReady,
		ProfileConfigured: s.ProfileConfigured,
		ProfileExists:     s.ProfileExists,
		ProfileCount:      int32(s.ProfileCount),
		Profiles:          profiles,
		Info:              infoToProto(s.Info),
		CheckedAt:         s.CheckedAt,
	}
}
