package audio_admin

import (
	"context"
	"testing"

	"web-console/internal/audioports"

	"connectrpc.com/connect"
	audioadminv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_admin"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type fakeAdminPorts struct{}

func (fakeAdminPorts) GetStreamConfig(context.Context) (audioports.StreamConfig, error) {
	return audioports.StreamConfig{FlushIntervalMs: 1}, nil
}

func (fakeAdminPorts) UpdateStreamConfig(context.Context, audioports.FieldMask, audioports.StreamConfig) (audioports.StreamConfig, error) {
	return audioports.StreamConfig{FlushIntervalMs: 2}, nil
}

func (fakeAdminPorts) GetWakeWordConfig(context.Context) (audioports.WakeWordConfig, error) {
	return audioports.WakeWordConfig{Configured: true}, nil
}

func (fakeAdminPorts) UpdateWakeWordTemplate(context.Context, audioports.WakeWordTemplate) (audioports.WakeWordConfig, error) {
	return audioports.WakeWordConfig{Configured: true}, nil
}

func (fakeAdminPorts) DeleteWakeWordTemplate(context.Context) (audioports.WakeWordConfig, error) {
	return audioports.WakeWordConfig{}, nil
}

func (fakeAdminPorts) GetTTSConfig(context.Context) (audioports.TTSConfig, error) {
	return audioports.TTSConfig{AutoEnabled: true}, nil
}

func (fakeAdminPorts) UpdateTTSConfig(context.Context, audioports.FieldMask, audioports.TTSConfig) (audioports.TTSConfig, error) {
	return audioports.TTSConfig{AutoEnabled: true}, nil
}

func (fakeAdminPorts) GetSummarizeConfig(context.Context) (audioports.SummarizeConfig, error) {
	return audioports.SummarizeConfig{Enabled: true}, nil
}

func (fakeAdminPorts) UpdateSummarizeConfig(context.Context, audioports.FieldMask, audioports.SummarizeConfig) (audioports.SummarizeConfig, error) {
	return audioports.SummarizeConfig{Enabled: true}, nil
}

func (fakeAdminPorts) ListSummarizeModels(context.Context) ([]audioports.SummarizeModel, error) {
	return []audioports.SummarizeModel{{ID: "m"}}, nil
}

func TestConnectHandlerAdminConfigOperations(t *testing.T) {
	f := fakeAdminPorts{}
	h := NewConnectHandler(Deps{StreamConfig: f, WakeWord: f, TTSConfig: f, SummarizeConfig: f})
	ctx := context.Background()
	if r, err := h.GetStreamConfig(ctx, connect.NewRequest(&audioadminv1.GetStreamConfigRequest{})); err != nil || r.Msg.Config.FlushIntervalMs != 1 {
		t.Fatal(err)
	}
	mask := &fieldmaskpb.FieldMask{Paths: []string{"enabled"}}
	if r, err := h.UpdateStreamConfig(ctx, connect.NewRequest(&audioadminv1.UpdateStreamConfigRequest{UpdateMask: mask})); err != nil || r.Msg.Config.FlushIntervalMs != 2 {
		t.Fatal(err)
	}
	if r, err := h.GetWakeWordConfig(ctx, connect.NewRequest(&audioadminv1.GetWakeWordConfigRequest{})); err != nil || !r.Msg.Config.Configured {
		t.Fatal(err)
	}
	if _, err := h.UpdateWakeWordTemplate(ctx, connect.NewRequest(&audioadminv1.UpdateWakeWordTemplateRequest{})); err != nil {
		t.Fatal(err)
	}
	if _, err := h.DeleteWakeWordTemplate(ctx, connect.NewRequest(&audioadminv1.DeleteWakeWordTemplateRequest{})); err != nil {
		t.Fatal(err)
	}
	if r, err := h.GetTTSConfig(ctx, connect.NewRequest(&audioadminv1.GetTTSConfigRequest{})); err != nil || !r.Msg.Config.AutoEnabled {
		t.Fatal(err)
	}
	if _, err := h.UpdateTTSConfig(ctx, connect.NewRequest(&audioadminv1.UpdateTTSConfigRequest{UpdateMask: mask})); err != nil {
		t.Fatal(err)
	}
	if r, err := h.GetSummarizeConfig(ctx, connect.NewRequest(&audioadminv1.GetSummarizeConfigRequest{})); err != nil || !r.Msg.Config.Enabled {
		t.Fatal(err)
	}
	if _, err := h.UpdateSummarizeConfig(ctx, connect.NewRequest(&audioadminv1.UpdateSummarizeConfigRequest{UpdateMask: mask})); err != nil {
		t.Fatal(err)
	}
	if r, err := h.ListSummarizeModels(ctx, connect.NewRequest(&audioadminv1.ListSummarizeModelsRequest{})); err != nil || len(r.Msg.Models) != 1 {
		t.Fatal(err)
	}
}

func TestConnectHandlerSpeakerProfileOperations(t *testing.T) {
	f := &fakeSpeakerAdmin{
		cfg:      audioports.SpeakerConfig{Enabled: true},
		profiles: []audioports.SpeakerProfile{{ID: "p1", DisplayName: "Primary"}},
	}
	h := NewConnectHandler(Deps{Speaker: f})
	ctx := context.Background()
	if r, err := h.ListSpeakerProfiles(ctx, connect.NewRequest(&audioadminv1.ListSpeakerProfilesRequest{})); err != nil || len(r.Msg.Profiles) != 1 {
		t.Fatal(err)
	}
	if _, err := h.ClearSpeakerProfileBinding(ctx, connect.NewRequest(&audioadminv1.ClearSpeakerProfileBindingRequest{})); err != nil {
		t.Fatal(err)
	}
	if _, err := h.DeleteSpeakerProfile(ctx, connect.NewRequest(&audioadminv1.DeleteSpeakerProfileRequest{ProfileId: "p1"})); err != nil {
		t.Fatal(err)
	}
	if f.lastDeleteID != "p1" {
		t.Fatalf("delete id=%q", f.lastDeleteID)
	}
}

func TestConnectHandlerAdminNilAndEmptyMasks(t *testing.T) {
	h := NewConnectHandler(Deps{})
	for _, call := range []func() error{
		func() error {
			_, e := h.GetStreamConfig(context.Background(), connect.NewRequest(&audioadminv1.GetStreamConfigRequest{}))
			return e
		},
		func() error {
			_, e := h.GetWakeWordConfig(context.Background(), connect.NewRequest(&audioadminv1.GetWakeWordConfigRequest{}))
			return e
		},
		func() error {
			_, e := h.GetTTSConfig(context.Background(), connect.NewRequest(&audioadminv1.GetTTSConfigRequest{}))
			return e
		},
		func() error {
			_, e := h.GetSummarizeConfig(context.Background(), connect.NewRequest(&audioadminv1.GetSummarizeConfigRequest{}))
			return e
		},
	} {
		if err := call(); err == nil {
			t.Fatal("expected unavailable")
		}
	}
	f := fakeAdminPorts{}
	h = NewConnectHandler(Deps{StreamConfig: f, TTSConfig: f, SummarizeConfig: f})
	if _, err := h.UpdateStreamConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateStreamConfigRequest{})); err == nil {
		t.Fatal("expected stream mask validation")
	}
	if _, err := h.UpdateTTSConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateTTSConfigRequest{})); err == nil {
		t.Fatal("expected tts mask validation")
	}
	if _, err := h.UpdateSummarizeConfig(context.Background(), connect.NewRequest(&audioadminv1.UpdateSummarizeConfigRequest{})); err == nil {
		t.Fatal("expected summary mask validation")
	}
}
