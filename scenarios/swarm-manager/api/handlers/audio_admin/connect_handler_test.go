package audio_admin

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"swarm-manager/integrations/audiotools"
	"swarm-manager/internal/audioports"

	audioadminv1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/audio_admin"
	audiocommonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/audio_common"
)

// -----------------------------------------------------------------------------
// Fakes
// -----------------------------------------------------------------------------

type fakeStreamConfig struct {
	cfg     audioports.StreamConfig
	err     error
	gotMask audioports.FieldMask
	gotCfg  audioports.StreamConfig
}

func (f *fakeStreamConfig) GetStreamConfig(_ context.Context) (audioports.StreamConfig, error) {
	return f.cfg, f.err
}

func (f *fakeStreamConfig) UpdateStreamConfig(_ context.Context, mask audioports.FieldMask, cfg audioports.StreamConfig) (audioports.StreamConfig, error) {
	f.gotMask = mask
	f.gotCfg = cfg
	return f.cfg, f.err
}

type fakeWakeWord struct {
	cfg      audioports.WakeWordConfig
	err      error
	gotTmpl  audioports.WakeWordTemplate
	deleteOK bool
}

func (f *fakeWakeWord) GetWakeWordConfig(_ context.Context) (audioports.WakeWordConfig, error) {
	return f.cfg, f.err
}

func (f *fakeWakeWord) UpdateWakeWordTemplate(_ context.Context, t audioports.WakeWordTemplate) (audioports.WakeWordConfig, error) {
	f.gotTmpl = t
	return f.cfg, f.err
}

func (f *fakeWakeWord) DeleteWakeWordTemplate(_ context.Context) (audioports.WakeWordConfig, error) {
	f.deleteOK = true
	return f.cfg, f.err
}

type fakeSpeaker struct {
	cfg    audioports.SpeakerConfig
	status audioports.SpeakerStatus

	profiles    []audioports.SpeakerProfile
	enrollOut   audioports.SpeakerEnrollResult
	err         error
	gotMask     audioports.FieldMask
	gotCfg      audioports.SpeakerConfig
	gotEnroll   audioports.EnrollSpeakerInput
	gotUnbindID string
	gotDeleteID string
}

func (f *fakeSpeaker) GetSpeakerConfig(_ context.Context) (audioports.SpeakerConfig, error) {
	return f.cfg, f.err
}

func (f *fakeSpeaker) UpdateSpeakerConfig(_ context.Context, mask audioports.FieldMask, cfg audioports.SpeakerConfig) (audioports.SpeakerConfig, error) {
	f.gotMask = mask
	f.gotCfg = cfg
	return f.cfg, f.err
}

func (f *fakeSpeaker) GetSpeakerStatus(_ context.Context) (audioports.SpeakerStatus, error) {
	return f.status, f.err
}

func (f *fakeSpeaker) ListSpeakerProfiles(_ context.Context) ([]audioports.SpeakerProfile, error) {
	return f.profiles, f.err
}

func (f *fakeSpeaker) EnrollSpeakerProfile(_ context.Context, in audioports.EnrollSpeakerInput) (audioports.SpeakerEnrollResult, error) {
	f.gotEnroll = in
	return f.enrollOut, f.err
}

func (f *fakeSpeaker) ClearSpeakerProfileBinding(_ context.Context) (audioports.SpeakerConfig, error) {
	return f.cfg, f.err
}

func (f *fakeSpeaker) UnbindSpeakerProfile(_ context.Context, id string) (audioports.SpeakerConfig, error) {
	f.gotUnbindID = id
	return f.cfg, f.err
}

func (f *fakeSpeaker) DeleteSpeakerProfile(_ context.Context, id string) (audioports.SpeakerConfig, error) {
	f.gotDeleteID = id
	return f.cfg, f.err
}

type fakeTTSConfig struct {
	cfg     audioports.TTSConfig
	err     error
	gotMask audioports.FieldMask
	gotCfg  audioports.TTSConfig
}

func (f *fakeTTSConfig) GetTTSConfig(_ context.Context) (audioports.TTSConfig, error) {
	return f.cfg, f.err
}

func (f *fakeTTSConfig) UpdateTTSConfig(_ context.Context, mask audioports.FieldMask, cfg audioports.TTSConfig) (audioports.TTSConfig, error) {
	f.gotMask = mask
	f.gotCfg = cfg
	return f.cfg, f.err
}

type fakeSummarizeConfig struct {
	cfg     audioports.SummarizeConfig
	models  []audioports.SummarizeModel
	err     error
	gotMask audioports.FieldMask
	gotCfg  audioports.SummarizeConfig
}

func (f *fakeSummarizeConfig) GetSummarizeConfig(_ context.Context) (audioports.SummarizeConfig, error) {
	return f.cfg, f.err
}

func (f *fakeSummarizeConfig) UpdateSummarizeConfig(_ context.Context, mask audioports.FieldMask, cfg audioports.SummarizeConfig) (audioports.SummarizeConfig, error) {
	f.gotMask = mask
	f.gotCfg = cfg
	return f.cfg, f.err
}

func (f *fakeSummarizeConfig) ListSummarizeModels(_ context.Context) ([]audioports.SummarizeModel, error) {
	return f.models, f.err
}

func assertCode(t *testing.T, err error, want connect.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %v, got nil", want)
	}
	if got := connect.CodeOf(err); got != want {
		t.Fatalf("expected code %v, got %v (err=%v)", want, got, err)
	}
}

func mask(paths ...string) *fieldmaskpb.FieldMask { return &fieldmaskpb.FieldMask{Paths: paths} }

// -----------------------------------------------------------------------------
// Stream config
// -----------------------------------------------------------------------------

func TestGetStreamConfig_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.GetStreamConfig(context.Background(), connect.NewRequest(&audioadminv1.GetStreamConfigRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestGetStreamConfig_HappyPath(t *testing.T) {
	sc := &fakeStreamConfig{cfg: audioports.StreamConfig{FlushIntervalMs: 250, WakeWordThreshold: 0.7}}
	h := NewConnectHandler(Deps{StreamConfig: sc})
	resp, err := h.GetStreamConfig(context.Background(), connect.NewRequest(&audioadminv1.GetStreamConfigRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Config.FlushIntervalMs != 250 || resp.Msg.Config.WakeWordThreshold != 0.7 {
		t.Errorf("config not mapped: %+v", resp.Msg.Config)
	}
}

func TestGetStreamConfig_ErrorMapping(t *testing.T) {
	h := NewConnectHandler(Deps{StreamConfig: &fakeStreamConfig{err: audiotools.ErrUnavailable}})
	_, err := h.GetStreamConfig(context.Background(), connect.NewRequest(&audioadminv1.GetStreamConfigRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestUpdateStreamConfig_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.UpdateStreamConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateStreamConfigRequest{UpdateMask: mask("flush_interval_ms")}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestUpdateStreamConfig_EmptyMask(t *testing.T) {
	h := NewConnectHandler(Deps{StreamConfig: &fakeStreamConfig{}})
	// nil mask
	_, err := h.UpdateStreamConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateStreamConfigRequest{}))
	assertCode(t, err, connect.CodeInvalidArgument)
	// empty-paths mask
	_, err = h.UpdateStreamConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateStreamConfigRequest{UpdateMask: mask()}))
	assertCode(t, err, connect.CodeInvalidArgument)
}

func TestUpdateStreamConfig_HappyPath(t *testing.T) {
	sc := &fakeStreamConfig{cfg: audioports.StreamConfig{FlushIntervalMs: 300}}
	h := NewConnectHandler(Deps{StreamConfig: sc})
	resp, err := h.UpdateStreamConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateStreamConfigRequest{
		UpdateMask: mask("flush_interval_ms"),
		Config:     &audioadminv1.StreamConfig{FlushIntervalMs: 300, PersistentMode: true},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Config.FlushIntervalMs != 300 {
		t.Errorf("config not mapped: %+v", resp.Msg.Config)
	}
	if len(sc.gotMask.Paths) != 1 || sc.gotMask.Paths[0] != "flush_interval_ms" {
		t.Errorf("mask not threaded: %+v", sc.gotMask)
	}
	if sc.gotCfg.FlushIntervalMs != 300 || !sc.gotCfg.PersistentMode {
		t.Errorf("incoming config not threaded: %+v", sc.gotCfg)
	}
}

// -----------------------------------------------------------------------------
// Wake word
// -----------------------------------------------------------------------------

func TestGetWakeWordConfig_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.GetWakeWordConfig(context.Background(), connect.NewRequest(&audioadminv1.GetWakeWordConfigRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestGetWakeWordConfig_HappyPath(t *testing.T) {
	ww := &fakeWakeWord{cfg: audioports.WakeWordConfig{
		Configured: true,
		Template:   &audioports.WakeWordTemplate{Label: "hey", Threshold: 0.5},
	}}
	h := NewConnectHandler(Deps{WakeWord: ww})
	resp, err := h.GetWakeWordConfig(context.Background(), connect.NewRequest(&audioadminv1.GetWakeWordConfigRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.Config.Configured || resp.Msg.Config.Template.Label != "hey" {
		t.Errorf("config not mapped: %+v", resp.Msg.Config)
	}
}

func TestUpdateWakeWordTemplate_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.UpdateWakeWordTemplate(context.Background(), connect.NewRequest(&audioadminv1.UpdateWakeWordTemplateRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestUpdateWakeWordTemplate_HappyPath(t *testing.T) {
	ww := &fakeWakeWord{cfg: audioports.WakeWordConfig{Configured: true}}
	h := NewConnectHandler(Deps{WakeWord: ww})
	resp, err := h.UpdateWakeWordTemplate(context.Background(), connect.NewRequest(&audioadminv1.UpdateWakeWordTemplateRequest{
		Template: &audioadminv1.WakeWordTemplate{
			Label:     "computer",
			Threshold: 0.6,
			Samples: []*audioadminv1.WakeWordSample{
				{Audio: []byte("a"), Format: audiocommonv1.AudioFormat_AUDIO_FORMAT_WAV, SampleRateHz: 16000},
			},
		},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.Config.Configured {
		t.Errorf("config not mapped: %+v", resp.Msg.Config)
	}
	if ww.gotTmpl.Label != "computer" || ww.gotTmpl.Threshold != 0.6 || len(ww.gotTmpl.Samples) != 1 {
		t.Errorf("template not threaded: %+v", ww.gotTmpl)
	}
	if ww.gotTmpl.Samples[0].SampleRateHz != 16000 || ww.gotTmpl.Samples[0].Format != audioports.AudioFormatWAV {
		t.Errorf("sample not threaded: %+v", ww.gotTmpl.Samples[0])
	}
}

func TestDeleteWakeWordTemplate_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.DeleteWakeWordTemplate(context.Background(), connect.NewRequest(&audioadminv1.DeleteWakeWordTemplateRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestDeleteWakeWordTemplate_HappyPath(t *testing.T) {
	ww := &fakeWakeWord{cfg: audioports.WakeWordConfig{Configured: false}}
	h := NewConnectHandler(Deps{WakeWord: ww})
	resp, err := h.DeleteWakeWordTemplate(context.Background(), connect.NewRequest(&audioadminv1.DeleteWakeWordTemplateRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ww.deleteOK {
		t.Errorf("delete not invoked")
	}
	if resp.Msg.Config.Configured {
		t.Errorf("expected unconfigured config")
	}
}

// -----------------------------------------------------------------------------
// Speaker
// -----------------------------------------------------------------------------

func TestGetSpeakerConfig_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.GetSpeakerConfig(context.Background(), connect.NewRequest(&audioadminv1.GetSpeakerConfigRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestGetSpeakerConfig_HappyPath(t *testing.T) {
	sp := &fakeSpeaker{cfg: audioports.SpeakerConfig{Enabled: true, ProfileIDs: []string{"p1"}, Threshold: 0.8}}
	h := NewConnectHandler(Deps{Speaker: sp})
	resp, err := h.GetSpeakerConfig(context.Background(), connect.NewRequest(&audioadminv1.GetSpeakerConfigRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.Config.Enabled || len(resp.Msg.Config.ProfileIds) != 1 || resp.Msg.Config.ProfileIds[0] != "p1" {
		t.Errorf("config not mapped: %+v", resp.Msg.Config)
	}
}

func TestUpdateSpeakerConfig_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.UpdateSpeakerConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateSpeakerConfigRequest{UpdateMask: mask("enabled")}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestUpdateSpeakerConfig_EmptyMask(t *testing.T) {
	h := NewConnectHandler(Deps{Speaker: &fakeSpeaker{}})
	_, err := h.UpdateSpeakerConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateSpeakerConfigRequest{}))
	assertCode(t, err, connect.CodeInvalidArgument)
}

func TestUpdateSpeakerConfig_HappyPath(t *testing.T) {
	sp := &fakeSpeaker{cfg: audioports.SpeakerConfig{Enabled: true}}
	h := NewConnectHandler(Deps{Speaker: sp})
	resp, err := h.UpdateSpeakerConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateSpeakerConfigRequest{
		UpdateMask: mask("enabled"),
		Config:     &audioadminv1.SpeakerConfig{Enabled: true, Threshold: 0.9},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.Config.Enabled {
		t.Errorf("config not mapped: %+v", resp.Msg.Config)
	}
	if len(sp.gotMask.Paths) != 1 || sp.gotCfg.Threshold != 0.9 {
		t.Errorf("mask/cfg not threaded: %+v / %+v", sp.gotMask, sp.gotCfg)
	}
}

func TestGetSpeakerStatus_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.GetSpeakerStatus(context.Background(), connect.NewRequest(&audioadminv1.GetSpeakerStatusRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestGetSpeakerStatus_HappyPath(t *testing.T) {
	sp := &fakeSpeaker{status: audioports.SpeakerStatus{
		Capability:      audioports.SpeakerCapabilityAvailable,
		CapabilityLabel: "available",
		ResourceReady:   true,
		ProfileCount:    2,
	}}
	h := NewConnectHandler(Deps{Speaker: sp})
	resp, err := h.GetSpeakerStatus(context.Background(), connect.NewRequest(&audioadminv1.GetSpeakerStatusRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.Status.ResourceReady || resp.Msg.Status.ProfileCount != 2 ||
		resp.Msg.Status.Capability != audiocommonv1.SpeakerCapability_SPEAKER_CAPABILITY_AVAILABLE {
		t.Errorf("status not mapped: %+v", resp.Msg.Status)
	}
}

func TestListSpeakerProfiles_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.ListSpeakerProfiles(context.Background(), connect.NewRequest(&audioadminv1.ListSpeakerProfilesRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestListSpeakerProfiles_HappyPath(t *testing.T) {
	sp := &fakeSpeaker{profiles: []audioports.SpeakerProfile{
		{ID: "p1", DisplayName: "Alice"},
		{ID: "p2", DisplayName: "Bob"},
	}}
	h := NewConnectHandler(Deps{Speaker: sp})
	resp, err := h.ListSpeakerProfiles(context.Background(), connect.NewRequest(&audioadminv1.ListSpeakerProfilesRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Count != 2 || len(resp.Msg.Profiles) != 2 {
		t.Fatalf("count = %d, profiles = %d", resp.Msg.Count, len(resp.Msg.Profiles))
	}
	if resp.Msg.Profiles[0].Id != "p1" || resp.Msg.Profiles[1].DisplayName != "Bob" {
		t.Errorf("profiles not mapped: %+v", resp.Msg.Profiles)
	}
}

func TestEnrollSpeakerProfile_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.EnrollSpeakerProfile(context.Background(), connect.NewRequest(&audioadminv1.EnrollSpeakerProfileRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestEnrollSpeakerProfile_HappyPath(t *testing.T) {
	sp := &fakeSpeaker{enrollOut: audioports.SpeakerEnrollResult{
		Enrollment: audioports.SpeakerEnrollment{ProfileID: "p9", DisplayName: "Carol"},
		Config:     audioports.SpeakerConfig{Enabled: true},
	}}
	h := NewConnectHandler(Deps{Speaker: sp})
	addActive := true
	enable := false
	resp, err := h.EnrollSpeakerProfile(context.Background(), connect.NewRequest(&audioadminv1.EnrollSpeakerProfileRequest{
		Audio:       []byte("wavbytes"),
		Format:      audiocommonv1.AudioFormat_AUDIO_FORMAT_WAV,
		ProfileId:   "p9",
		DisplayName: "Carol",
		Notes:       "test",
		AddToActive: &addActive,
		Enable:      &enable,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Enrollment.ProfileId != "p9" || !resp.Msg.Config.Enabled {
		t.Errorf("response not mapped: %+v", resp.Msg)
	}
	if string(sp.gotEnroll.Audio) != "wavbytes" || sp.gotEnroll.ProfileID != "p9" ||
		sp.gotEnroll.Format != audioports.AudioFormatWAV || sp.gotEnroll.Notes != "test" {
		t.Errorf("input not threaded: %+v", sp.gotEnroll)
	}
	if sp.gotEnroll.AddToActive == nil || !*sp.gotEnroll.AddToActive {
		t.Errorf("AddToActive not threaded: %+v", sp.gotEnroll.AddToActive)
	}
	if sp.gotEnroll.Enable == nil || *sp.gotEnroll.Enable {
		t.Errorf("Enable not threaded: %+v", sp.gotEnroll.Enable)
	}
}

func TestEnrollSpeakerProfile_TriStateNil(t *testing.T) {
	sp := &fakeSpeaker{}
	h := NewConnectHandler(Deps{Speaker: sp})
	_, err := h.EnrollSpeakerProfile(context.Background(), connect.NewRequest(&audioadminv1.EnrollSpeakerProfileRequest{
		Audio: []byte("x"),
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sp.gotEnroll.AddToActive != nil || sp.gotEnroll.Enable != nil {
		t.Errorf("unset tri-state should stay nil: %+v / %+v", sp.gotEnroll.AddToActive, sp.gotEnroll.Enable)
	}
}

func TestClearSpeakerProfileBinding_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.ClearSpeakerProfileBinding(context.Background(), connect.NewRequest(&audioadminv1.ClearSpeakerProfileBindingRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestClearSpeakerProfileBinding_HappyPath(t *testing.T) {
	sp := &fakeSpeaker{cfg: audioports.SpeakerConfig{Enabled: false}}
	h := NewConnectHandler(Deps{Speaker: sp})
	resp, err := h.ClearSpeakerProfileBinding(context.Background(), connect.NewRequest(&audioadminv1.ClearSpeakerProfileBindingRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Config == nil {
		t.Errorf("expected config in response")
	}
}

func TestUnbindSpeakerProfile_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.UnbindSpeakerProfile(context.Background(), connect.NewRequest(&audioadminv1.UnbindSpeakerProfileRequest{ProfileId: "p1"}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestUnbindSpeakerProfile_EmptyID(t *testing.T) {
	h := NewConnectHandler(Deps{Speaker: &fakeSpeaker{}})
	_, err := h.UnbindSpeakerProfile(context.Background(), connect.NewRequest(&audioadminv1.UnbindSpeakerProfileRequest{}))
	assertCode(t, err, connect.CodeInvalidArgument)
}

func TestUnbindSpeakerProfile_HappyPath(t *testing.T) {
	sp := &fakeSpeaker{cfg: audioports.SpeakerConfig{Enabled: true}}
	h := NewConnectHandler(Deps{Speaker: sp})
	_, err := h.UnbindSpeakerProfile(context.Background(), connect.NewRequest(&audioadminv1.UnbindSpeakerProfileRequest{ProfileId: "p7"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sp.gotUnbindID != "p7" {
		t.Errorf("profile id not threaded: %q", sp.gotUnbindID)
	}
}

func TestDeleteSpeakerProfile_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.DeleteSpeakerProfile(context.Background(), connect.NewRequest(&audioadminv1.DeleteSpeakerProfileRequest{ProfileId: "p1"}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestDeleteSpeakerProfile_EmptyID(t *testing.T) {
	h := NewConnectHandler(Deps{Speaker: &fakeSpeaker{}})
	_, err := h.DeleteSpeakerProfile(context.Background(), connect.NewRequest(&audioadminv1.DeleteSpeakerProfileRequest{}))
	assertCode(t, err, connect.CodeInvalidArgument)
}

func TestDeleteSpeakerProfile_HappyPath(t *testing.T) {
	sp := &fakeSpeaker{cfg: audioports.SpeakerConfig{}}
	h := NewConnectHandler(Deps{Speaker: sp})
	_, err := h.DeleteSpeakerProfile(context.Background(), connect.NewRequest(&audioadminv1.DeleteSpeakerProfileRequest{ProfileId: "p3"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sp.gotDeleteID != "p3" {
		t.Errorf("profile id not threaded: %q", sp.gotDeleteID)
	}
}

// -----------------------------------------------------------------------------
// TTS config
// -----------------------------------------------------------------------------

func TestGetTTSConfig_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.GetTTSConfig(context.Background(), connect.NewRequest(&audioadminv1.GetTTSConfigRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestGetTTSConfig_HappyPath(t *testing.T) {
	tc := &fakeTTSConfig{cfg: audioports.TTSConfig{AutoEnabled: true, DefaultVoice: "af_heart", DefaultSpeed: 1.2}}
	h := NewConnectHandler(Deps{TTSConfig: tc})
	resp, err := h.GetTTSConfig(context.Background(), connect.NewRequest(&audioadminv1.GetTTSConfigRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.Config.AutoEnabled || resp.Msg.Config.DefaultVoice != "af_heart" || resp.Msg.Config.DefaultSpeed != 1.2 {
		t.Errorf("config not mapped: %+v", resp.Msg.Config)
	}
}

func TestUpdateTTSConfig_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.UpdateTTSConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateTTSConfigRequest{UpdateMask: mask("auto_enabled")}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestUpdateTTSConfig_EmptyMask(t *testing.T) {
	h := NewConnectHandler(Deps{TTSConfig: &fakeTTSConfig{}})
	_, err := h.UpdateTTSConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateTTSConfigRequest{}))
	assertCode(t, err, connect.CodeInvalidArgument)
}

func TestUpdateTTSConfig_HappyPath(t *testing.T) {
	tc := &fakeTTSConfig{cfg: audioports.TTSConfig{AutoEnabled: true}}
	h := NewConnectHandler(Deps{TTSConfig: tc})
	resp, err := h.UpdateTTSConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateTTSConfigRequest{
		UpdateMask: mask("default_voice"),
		Config:     &audioadminv1.TTSConfig{DefaultVoice: "af_bella"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.Config.AutoEnabled {
		t.Errorf("config not mapped: %+v", resp.Msg.Config)
	}
	if tc.gotCfg.DefaultVoice != "af_bella" || len(tc.gotMask.Paths) != 1 {
		t.Errorf("cfg/mask not threaded: %+v / %+v", tc.gotCfg, tc.gotMask)
	}
}

// -----------------------------------------------------------------------------
// Summarize config
// -----------------------------------------------------------------------------

func TestGetSummarizeConfig_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.GetSummarizeConfig(context.Background(), connect.NewRequest(&audioadminv1.GetSummarizeConfigRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestGetSummarizeConfig_HappyPath(t *testing.T) {
	sm := &fakeSummarizeConfig{cfg: audioports.SummarizeConfig{Enabled: true, CharThreshold: 500, Model: "qwen"}}
	h := NewConnectHandler(Deps{SummarizeConfig: sm})
	resp, err := h.GetSummarizeConfig(context.Background(), connect.NewRequest(&audioadminv1.GetSummarizeConfigRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.Config.Enabled || resp.Msg.Config.CharThreshold != 500 || resp.Msg.Config.Model != "qwen" {
		t.Errorf("config not mapped: %+v", resp.Msg.Config)
	}
}

func TestUpdateSummarizeConfig_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.UpdateSummarizeConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateSummarizeConfigRequest{UpdateMask: mask("enabled")}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestUpdateSummarizeConfig_EmptyMask(t *testing.T) {
	h := NewConnectHandler(Deps{SummarizeConfig: &fakeSummarizeConfig{}})
	_, err := h.UpdateSummarizeConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateSummarizeConfigRequest{}))
	assertCode(t, err, connect.CodeInvalidArgument)
}

func TestUpdateSummarizeConfig_HappyPath(t *testing.T) {
	sm := &fakeSummarizeConfig{cfg: audioports.SummarizeConfig{Enabled: true}}
	h := NewConnectHandler(Deps{SummarizeConfig: sm})
	resp, err := h.UpdateSummarizeConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateSummarizeConfigRequest{
		UpdateMask: mask("model"),
		Config:     &audioadminv1.SummarizeConfig{Model: "llama"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.Config.Enabled {
		t.Errorf("config not mapped: %+v", resp.Msg.Config)
	}
	if sm.gotCfg.Model != "llama" || len(sm.gotMask.Paths) != 1 {
		t.Errorf("cfg/mask not threaded: %+v / %+v", sm.gotCfg, sm.gotMask)
	}
}

func TestListSummarizeModels_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.ListSummarizeModels(context.Background(), connect.NewRequest(&audioadminv1.ListSummarizeModelsRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestListSummarizeModels_HappyPath(t *testing.T) {
	sm := &fakeSummarizeConfig{models: []audioports.SummarizeModel{
		{ID: "m1", DisplayName: "Model One", Installed: true},
		{ID: "m2", DisplayName: "Model Two"},
	}}
	h := NewConnectHandler(Deps{SummarizeConfig: sm})
	resp, err := h.ListSummarizeModels(context.Background(), connect.NewRequest(&audioadminv1.ListSummarizeModelsRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Models) != 2 {
		t.Fatalf("got %d models, want 2", len(resp.Msg.Models))
	}
	if resp.Msg.Models[0].Id != "m1" || !resp.Msg.Models[0].Installed || resp.Msg.Models[1].DisplayName != "Model Two" {
		t.Errorf("models not mapped: %+v", resp.Msg.Models)
	}
}

func TestListSummarizeModels_ErrorMapping(t *testing.T) {
	h := NewConnectHandler(Deps{SummarizeConfig: &fakeSummarizeConfig{err: audiotools.ErrTimeout}})
	_, err := h.ListSummarizeModels(context.Background(), connect.NewRequest(&audioadminv1.ListSummarizeModelsRequest{}))
	assertCode(t, err, connect.CodeDeadlineExceeded)
}

// -----------------------------------------------------------------------------
// mapErr
// -----------------------------------------------------------------------------

func TestMapErr(t *testing.T) {
	if mapErr(nil) != nil {
		t.Errorf("mapErr(nil) should be nil")
	}
	cases := []struct {
		name string
		in   error
		want connect.Code
	}{
		{"timeout", audiotools.ErrTimeout, connect.CodeDeadlineExceeded},
		{"unavailable", audiotools.ErrUnavailable, connect.CodeUnavailable},
		{"failedprecondition", audiotools.ErrFailedPrecondition, connect.CodeFailedPrecondition},
		{"insufficientcredits", audiotools.ErrInsufficientCredits, connect.CodeResourceExhausted},
		{"invalidargument", audiotools.ErrInvalidArgument, connect.CodeInvalidArgument},
		{"plain", errors.New("boom"), connect.CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertCode(t, mapErr(tc.in), tc.want)
		})
	}
	wrapped := connect.NewError(connect.CodeAlreadyExists, errors.New("dup"))
	if connect.CodeOf(mapErr(wrapped)) != connect.CodeAlreadyExists {
		t.Errorf("pre-wrapped connect error not passed through")
	}
}
