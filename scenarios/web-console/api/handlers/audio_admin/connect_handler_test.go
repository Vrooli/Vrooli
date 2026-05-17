package audio_admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	"web-console/integrations/audiotools"
	"web-console/internal/audioports"

	audioadminv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_admin"
	audiocommonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_common"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// fakeSpeakerAdmin captures calls and returns canned data.
type fakeSpeakerAdmin struct {
	cfg              audioports.SpeakerConfig
	status           audioports.SpeakerStatus
	profiles         []audioports.SpeakerProfile
	enroll           audioports.SpeakerEnrollResult
	lastUpdateMask   audioports.FieldMask
	lastUpdateCfg    audioports.SpeakerConfig
	lastUnbindID     string
	lastDeleteID     string
	lastEnrollInput  audioports.EnrollSpeakerInput
	getConfigErr     error
}

func (f *fakeSpeakerAdmin) GetSpeakerConfig(_ context.Context) (audioports.SpeakerConfig, error) {
	if f.getConfigErr != nil {
		return audioports.SpeakerConfig{}, f.getConfigErr
	}
	return f.cfg, nil
}
func (f *fakeSpeakerAdmin) UpdateSpeakerConfig(_ context.Context, mask audioports.FieldMask, cfg audioports.SpeakerConfig) (audioports.SpeakerConfig, error) {
	f.lastUpdateMask = mask
	f.lastUpdateCfg = cfg
	return cfg, nil
}
func (f *fakeSpeakerAdmin) GetSpeakerStatus(_ context.Context) (audioports.SpeakerStatus, error) {
	return f.status, nil
}
func (f *fakeSpeakerAdmin) ListSpeakerProfiles(_ context.Context) ([]audioports.SpeakerProfile, error) {
	return f.profiles, nil
}
func (f *fakeSpeakerAdmin) EnrollSpeakerProfile(_ context.Context, in audioports.EnrollSpeakerInput) (audioports.SpeakerEnrollResult, error) {
	f.lastEnrollInput = in
	return f.enroll, nil
}
func (f *fakeSpeakerAdmin) ClearSpeakerProfileBinding(_ context.Context) (audioports.SpeakerConfig, error) {
	return f.cfg, nil
}
func (f *fakeSpeakerAdmin) UnbindSpeakerProfile(_ context.Context, id string) (audioports.SpeakerConfig, error) {
	f.lastUnbindID = id
	return f.cfg, nil
}
func (f *fakeSpeakerAdmin) DeleteSpeakerProfile(_ context.Context, id string) (audioports.SpeakerConfig, error) {
	f.lastDeleteID = id
	return f.cfg, nil
}

func newTestHandler(speaker audioports.SpeakerAdmin) *connectHandler {
	return NewConnectHandler(Deps{Speaker: speaker})
}

func TestGetSpeakerStatus_HappyPath(t *testing.T) {
	fake := &fakeSpeakerAdmin{
		status: audioports.SpeakerStatus{
			Config:        audioports.SpeakerConfig{Enabled: true, Threshold: 0.7, Mode: audioports.SpeakerModeFilter},
			Capability:    audioports.SpeakerCapabilityAvailable,
			ResourceReady: true,
			ProfileCount:  2,
			CheckedAt:     time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC),
		},
	}
	h := newTestHandler(fake)
	resp, err := h.GetSpeakerStatus(context.Background(), connect.NewRequest(&audioadminv1.GetSpeakerStatusRequest{}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Msg.Status.Capability != audiocommonv1.SpeakerCapability_SPEAKER_CAPABILITY_AVAILABLE {
		t.Errorf("capability: got %v, want AVAILABLE", resp.Msg.Status.Capability)
	}
	if !resp.Msg.Status.ResourceReady {
		t.Errorf("resource_ready: got false, want true")
	}
	if resp.Msg.Status.Config.Mode != audiocommonv1.SpeakerMode_SPEAKER_MODE_FILTER {
		t.Errorf("mode: got %v, want FILTER", resp.Msg.Status.Config.Mode)
	}
}

func TestGetSpeakerStatus_PortNil_ReturnsUnavailable(t *testing.T) {
	h := newTestHandler(nil)
	_, err := h.GetSpeakerStatus(context.Background(), connect.NewRequest(&audioadminv1.GetSpeakerStatusRequest{}))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeUnavailable {
		t.Errorf("err code: got %v, want CodeUnavailable", err)
	}
}

func TestGetSpeakerStatus_AudioToolsUnavailable_MapsToConnectUnavailable(t *testing.T) {
	fake := &fakeSpeakerAdmin{getConfigErr: audiotools.ErrUnavailable}
	h := newTestHandler(fake)
	_, err := h.GetSpeakerConfig(context.Background(), connect.NewRequest(&audioadminv1.GetSpeakerConfigRequest{}))
	if err == nil {
		t.Fatal("expected error")
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeUnavailable {
		t.Errorf("err code: got %v, want CodeUnavailable", err)
	}
}

func TestUpdateSpeakerConfig_EmptyMask_InvalidArgument(t *testing.T) {
	fake := &fakeSpeakerAdmin{}
	h := newTestHandler(fake)
	_, err := h.UpdateSpeakerConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateSpeakerConfigRequest{
		Config: &audioadminv1.SpeakerConfig{Enabled: true},
	}))
	if err == nil {
		t.Fatal("expected error on empty mask")
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeInvalidArgument {
		t.Errorf("err code: got %v, want CodeInvalidArgument", err)
	}
}

func TestUpdateSpeakerConfig_PassesMaskAndConfig(t *testing.T) {
	fake := &fakeSpeakerAdmin{}
	h := newTestHandler(fake)
	_, err := h.UpdateSpeakerConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateSpeakerConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"enabled", "threshold"}},
		Config:     &audioadminv1.SpeakerConfig{Enabled: true, Threshold: 0.65},
	}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got := fake.lastUpdateMask.Paths; len(got) != 2 || got[0] != "enabled" || got[1] != "threshold" {
		t.Errorf("mask paths: got %v, want [enabled threshold]", got)
	}
	if !fake.lastUpdateCfg.Enabled || fake.lastUpdateCfg.Threshold != 0.65 {
		t.Errorf("config: got %+v, want Enabled=true Threshold=0.65", fake.lastUpdateCfg)
	}
}

func TestUnbindSpeakerProfile_RequiresProfileID(t *testing.T) {
	fake := &fakeSpeakerAdmin{}
	h := newTestHandler(fake)
	_, err := h.UnbindSpeakerProfile(context.Background(), connect.NewRequest(&audioadminv1.UnbindSpeakerProfileRequest{}))
	if err == nil {
		t.Fatal("expected error on empty profile_id")
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeInvalidArgument {
		t.Errorf("err code: got %v, want CodeInvalidArgument", err)
	}
}

func TestEnrollSpeakerProfile_PassesTriState(t *testing.T) {
	fake := &fakeSpeakerAdmin{}
	h := newTestHandler(fake)
	addToActive := true
	enable := false
	_, err := h.EnrollSpeakerProfile(context.Background(), connect.NewRequest(&audioadminv1.EnrollSpeakerProfileRequest{
		Audio:       []byte{0x01, 0x02},
		Format:      audiocommonv1.AudioFormat_AUDIO_FORMAT_WAV,
		ProfileId:   "p1",
		DisplayName: "Test",
		AddToActive: &addToActive,
		Enable:      &enable,
	}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if fake.lastEnrollInput.AddToActive == nil || !*fake.lastEnrollInput.AddToActive {
		t.Errorf("add_to_active: got %v, want true", fake.lastEnrollInput.AddToActive)
	}
	if fake.lastEnrollInput.Enable == nil || *fake.lastEnrollInput.Enable {
		t.Errorf("enable: got %v, want false", fake.lastEnrollInput.Enable)
	}
	if fake.lastEnrollInput.Format != audioports.AudioFormatWAV {
		t.Errorf("format: got %v, want WAV", fake.lastEnrollInput.Format)
	}
}
