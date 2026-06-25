package audioports

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"swarm-manager/integrations/audiotools"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

// -----------------------------------------------------------------------------
// RemoteStreamConfigAdmin
// -----------------------------------------------------------------------------

func TestRemoteStreamConfigAdmin(t *testing.T) {
	ctx := context.Background()

	t.Run("get nil client", func(t *testing.T) {
		r := &RemoteStreamConfigAdmin{}
		if _, err := r.GetStreamConfig(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
	})

	t.Run("get happy", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{getStreamConfig: func() (*sttv1.GetStreamConfigResponse, error) {
			return &sttv1.GetStreamConfigResponse{Config: &sttv1.StreamConfig{FlushIntervalMs: 77}}, nil
		}}
		r := &RemoteStreamConfigAdmin{remoteBase: remoteBase{Client: c}}
		got, err := r.GetStreamConfig(ctx)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got.FlushIntervalMs != 77 {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("get empty response", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{getStreamConfig: func() (*sttv1.GetStreamConfigResponse, error) { return nil, nil }}
		r := &RemoteStreamConfigAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.GetStreamConfig(ctx); err == nil || err.Error() != "audiotools: empty get_stream_config response" {
			t.Fatalf("want empty err, got %v", err)
		}
	})

	t.Run("get transport failure", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{getStreamConfig: func() (*sttv1.GetStreamConfigResponse, error) { return nil, unavailableErr() }}
		r := &RemoteStreamConfigAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.GetStreamConfig(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
		if c.Resolved() {
			t.Error("expected re-resolve")
		}
	})

	t.Run("update empty mask rejected before RPC", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{updateStreamConfig: func() (*sttv1.UpdateStreamConfigResponse, error) {
			t.Fatal("RPC must not be called with empty mask")
			return nil, nil
		}}
		r := &RemoteStreamConfigAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.UpdateStreamConfig(ctx, FieldMask{}, StreamConfig{}); !errors.Is(err, audiotools.ErrInvalidArgument) {
			t.Fatalf("want ErrInvalidArgument, got %v", err)
		}
	})

	t.Run("update happy", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{updateStreamConfig: func() (*sttv1.UpdateStreamConfigResponse, error) {
			return &sttv1.UpdateStreamConfigResponse{Config: &sttv1.StreamConfig{MinDeltaBytes: 5}}, nil
		}}
		r := &RemoteStreamConfigAdmin{remoteBase: remoteBase{Client: c}}
		got, err := r.UpdateStreamConfig(ctx, FieldMask{Paths: []string{"min_delta_bytes"}}, StreamConfig{MinDeltaBytes: 5})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got.MinDeltaBytes != 5 {
			t.Errorf("got %+v", got)
		}
	})
}

// -----------------------------------------------------------------------------
// RemoteWakeWordAdmin
// -----------------------------------------------------------------------------

func TestRemoteWakeWordAdmin(t *testing.T) {
	ctx := context.Background()

	t.Run("get nil client", func(t *testing.T) {
		r := &RemoteWakeWordAdmin{}
		if _, err := r.GetWakeWordConfig(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
	})

	t.Run("get happy", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{getWakeWordConfig: func() (*sttv1.GetWakeWordConfigResponse, error) {
			return &sttv1.GetWakeWordConfigResponse{Config: &sttv1.WakeWordConfig{Configured: true}}, nil
		}}
		r := &RemoteWakeWordAdmin{remoteBase: remoteBase{Client: c}}
		got, err := r.GetWakeWordConfig(ctx)
		if err != nil || !got.Configured {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	t.Run("update happy", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{updateWakeWord: func() (*sttv1.UpdateWakeWordTemplateResponse, error) {
			return &sttv1.UpdateWakeWordTemplateResponse{Config: &sttv1.WakeWordConfig{Configured: true}}, nil
		}}
		r := &RemoteWakeWordAdmin{remoteBase: remoteBase{Client: c}}
		got, err := r.UpdateWakeWordTemplate(ctx, WakeWordTemplate{Label: "hi"})
		if err != nil || !got.Configured {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	t.Run("delete empty response", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{deleteWakeWord: func() (*sttv1.DeleteWakeWordTemplateResponse, error) { return nil, nil }}
		r := &RemoteWakeWordAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.DeleteWakeWordTemplate(ctx); err == nil ||
			err.Error() != "audiotools: empty delete_wake_word_template response" {
			t.Fatalf("want empty err, got %v", err)
		}
	})

	t.Run("delete transport failure", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{deleteWakeWord: func() (*sttv1.DeleteWakeWordTemplateResponse, error) { return nil, unavailableErr() }}
		r := &RemoteWakeWordAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.DeleteWakeWordTemplate(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
		if c.Resolved() {
			t.Error("expected re-resolve")
		}
	})
}

// -----------------------------------------------------------------------------
// RemoteSpeakerAdmin
// -----------------------------------------------------------------------------

func TestRemoteSpeakerAdmin(t *testing.T) {
	ctx := context.Background()

	t.Run("get config nil client", func(t *testing.T) {
		r := &RemoteSpeakerAdmin{}
		if _, err := r.GetSpeakerConfig(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
	})

	t.Run("get config happy", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{getSpeakerConfig: func() (*sttv1.GetSpeakerConfigResponse, error) {
			return &sttv1.GetSpeakerConfigResponse{Config: &sttv1.SpeakerConfig{Enabled: true, Threshold: 0.5}}, nil
		}}
		r := &RemoteSpeakerAdmin{remoteBase: remoteBase{Client: c}}
		got, err := r.GetSpeakerConfig(ctx)
		if err != nil || !got.Enabled || got.Threshold != 0.5 {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	t.Run("update config empty mask", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{}
		r := &RemoteSpeakerAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.UpdateSpeakerConfig(ctx, FieldMask{}, SpeakerConfig{}); !errors.Is(err, audiotools.ErrInvalidArgument) {
			t.Fatalf("want ErrInvalidArgument, got %v", err)
		}
	})

	t.Run("update config happy", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{updateSpeakerConfig: func() (*sttv1.UpdateSpeakerConfigResponse, error) {
			return &sttv1.UpdateSpeakerConfigResponse{Config: &sttv1.SpeakerConfig{Enabled: true}}, nil
		}}
		r := &RemoteSpeakerAdmin{remoteBase: remoteBase{Client: c}}
		got, err := r.UpdateSpeakerConfig(ctx, FieldMask{Paths: []string{"enabled"}}, SpeakerConfig{Enabled: true})
		if err != nil || !got.Enabled {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	t.Run("get status happy", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{getSpeakerStatus: func() (*sttv1.GetSpeakerStatusResponse, error) {
			return &sttv1.GetSpeakerStatusResponse{Status: &sttv1.SpeakerStatus{Capability: "available", ProfileCount: 1}}, nil
		}}
		r := &RemoteSpeakerAdmin{remoteBase: remoteBase{Client: c}}
		got, err := r.GetSpeakerStatus(ctx)
		if err != nil || got.Capability != SpeakerCapabilityAvailable || got.ProfileCount != 1 {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	t.Run("list profiles happy", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{listSpeakerProfiles: func() (*sttv1.ListSpeakerProfilesResponse, error) {
			return &sttv1.ListSpeakerProfilesResponse{Profiles: []*sttv1.SpeakerProfile{{Id: "a"}, {Id: "b"}}}, nil
		}}
		r := &RemoteSpeakerAdmin{remoteBase: remoteBase{Client: c}}
		got, err := r.ListSpeakerProfiles(ctx)
		if err != nil || len(got) != 2 || got[0].ID != "a" || got[1].ID != "b" {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	t.Run("list profiles nil response -> nil slice", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{listSpeakerProfiles: func() (*sttv1.ListSpeakerProfilesResponse, error) { return nil, nil }}
		r := &RemoteSpeakerAdmin{remoteBase: remoteBase{Client: c}}
		got, err := r.ListSpeakerProfiles(ctx)
		if err != nil || got != nil {
			t.Fatalf("want nil,nil got %v,%v", got, err)
		}
	})

	t.Run("enroll forwards tri-state + maps result", func(t *testing.T) {
		c := newTestClient(t)
		yes := true
		var seen *sttv1.EnrollSpeakerProfileRequest
		c.STTAdmin = &fakeSTTAdmin{enrollSpeaker: func(req *sttv1.EnrollSpeakerProfileRequest) (*sttv1.EnrollSpeakerProfileResponse, error) {
			seen = req
			return &sttv1.EnrollSpeakerProfileResponse{
				Enrollment: &sttv1.SpeakerEnrollment{ProfileId: "p9"},
				Config:     &sttv1.SpeakerConfig{Enabled: true},
			}, nil
		}}
		r := &RemoteSpeakerAdmin{remoteBase: remoteBase{Client: c}}
		got, err := r.EnrollSpeakerProfile(ctx, EnrollSpeakerInput{
			ProfileID: "p9", Format: AudioFormatWAV, AddToActive: &yes, Enable: &yes,
		})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got.Enrollment.ProfileID != "p9" || !got.Config.Enabled {
			t.Errorf("result mismatch: %+v", got)
		}
		if seen == nil || seen.AddToActive == nil || !*seen.AddToActive || seen.Enable == nil || !*seen.Enable {
			t.Errorf("tri-state not forwarded: %+v", seen)
		}
	})

	t.Run("unbind empty id rejected", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{}
		r := &RemoteSpeakerAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.UnbindSpeakerProfile(ctx, ""); !errors.Is(err, audiotools.ErrInvalidArgument) {
			t.Fatalf("want ErrInvalidArgument, got %v", err)
		}
	})

	t.Run("unbind happy", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{unbindSpeaker: func() (*sttv1.UnbindSpeakerProfileResponse, error) {
			return &sttv1.UnbindSpeakerProfileResponse{Config: &sttv1.SpeakerConfig{}}, nil
		}}
		r := &RemoteSpeakerAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.UnbindSpeakerProfile(ctx, "p1"); err != nil {
			t.Fatalf("err: %v", err)
		}
	})

	t.Run("delete empty id rejected", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{}
		r := &RemoteSpeakerAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.DeleteSpeakerProfile(ctx, ""); !errors.Is(err, audiotools.ErrInvalidArgument) {
			t.Fatalf("want ErrInvalidArgument, got %v", err)
		}
	})

	t.Run("delete happy", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{deleteSpeaker: func() (*sttv1.DeleteSpeakerProfileResponse, error) {
			return &sttv1.DeleteSpeakerProfileResponse{Config: &sttv1.SpeakerConfig{}}, nil
		}}
		r := &RemoteSpeakerAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.DeleteSpeakerProfile(ctx, "p1"); err != nil {
			t.Fatalf("err: %v", err)
		}
	})

	t.Run("clear binding happy", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{clearSpeakerBinding: func() (*sttv1.ClearSpeakerProfileBindingResponse, error) {
			return &sttv1.ClearSpeakerProfileBindingResponse{Config: &sttv1.SpeakerConfig{}}, nil
		}}
		r := &RemoteSpeakerAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.ClearSpeakerProfileBinding(ctx); err != nil {
			t.Fatalf("err: %v", err)
		}
	})

	t.Run("clear binding transport failure", func(t *testing.T) {
		c := newTestClient(t)
		c.STTAdmin = &fakeSTTAdmin{clearSpeakerBinding: func() (*sttv1.ClearSpeakerProfileBindingResponse, error) { return nil, unavailableErr() }}
		r := &RemoteSpeakerAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.ClearSpeakerProfileBinding(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
		if c.Resolved() {
			t.Error("expected re-resolve")
		}
	})

	// Credentials attach path: verify creds-bearing request reaches the RPC
	// and the header is set (exercises attach()).
	t.Run("credentials attach", func(t *testing.T) {
		c := newTestClient(t)
		var hdr string
		c.STTAdmin = &fakeSTTAdminWithHeader{
			fn: func(req *connect.Request[sttv1.GetSpeakerConfigRequest]) {
				hdr = req.Header().Get("X-Audio-BYOK-Key")
			},
			config: &sttv1.SpeakerConfig{},
		}
		r := &RemoteSpeakerAdmin{
			remoteBase: remoteBase{
				Client: c,
				Credentials: func(context.Context) audiotools.Credentials {
					return audiotools.Credentials{BYOKKey: "secret", BYOKProvider: "openai"}
				},
			},
		}
		if _, err := r.GetSpeakerConfig(ctx); err != nil {
			t.Fatalf("err: %v", err)
		}
		if hdr != "secret" {
			t.Errorf("BYOK header not attached, got %q", hdr)
		}
	})
}

// fakeSTTAdminWithHeader is a minimal admin fake that inspects the request
// header on GetSpeakerConfig (to exercise the credential attach path).
type fakeSTTAdminWithHeader struct {
	fakeSTTAdmin
	fn     func(*connect.Request[sttv1.GetSpeakerConfigRequest])
	config *sttv1.SpeakerConfig
}

func (f *fakeSTTAdminWithHeader) GetSpeakerConfig(_ context.Context, req *connect.Request[sttv1.GetSpeakerConfigRequest]) (*connect.Response[sttv1.GetSpeakerConfigResponse], error) {
	f.fn(req)
	return connect.NewResponse(&sttv1.GetSpeakerConfigResponse{Config: f.config}), nil
}

// -----------------------------------------------------------------------------
// RemoteSummarizeConfigAdmin
// -----------------------------------------------------------------------------

func TestRemoteSummarizeConfigAdmin(t *testing.T) {
	ctx := context.Background()

	t.Run("get nil client", func(t *testing.T) {
		r := &RemoteSummarizeConfigAdmin{}
		if _, err := r.GetSummarizeConfig(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
	})

	t.Run("get happy", func(t *testing.T) {
		c := newTestClient(t)
		c.Summarize = &fakeSummarize{getConfig: func() (*summv1.GetSummarizeConfigResponse, error) {
			return &summv1.GetSummarizeConfigResponse{Config: &summv1.SummarizeConfig{Enabled: true, CharThreshold: 500}}, nil
		}}
		r := &RemoteSummarizeConfigAdmin{remoteBase: remoteBase{Client: c}}
		got, err := r.GetSummarizeConfig(ctx)
		if err != nil || !got.Enabled || got.CharThreshold != 500 {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	t.Run("update empty mask", func(t *testing.T) {
		c := newTestClient(t)
		c.Summarize = &fakeSummarize{}
		r := &RemoteSummarizeConfigAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.UpdateSummarizeConfig(ctx, FieldMask{}, SummarizeConfig{}); !errors.Is(err, audiotools.ErrInvalidArgument) {
			t.Fatalf("want ErrInvalidArgument, got %v", err)
		}
	})

	t.Run("update happy", func(t *testing.T) {
		c := newTestClient(t)
		c.Summarize = &fakeSummarize{updateConfig: func() (*summv1.UpdateSummarizeConfigResponse, error) {
			return &summv1.UpdateSummarizeConfigResponse{Config: &summv1.SummarizeConfig{Model: "gemma"}}, nil
		}}
		r := &RemoteSummarizeConfigAdmin{remoteBase: remoteBase{Client: c}}
		got, err := r.UpdateSummarizeConfig(ctx, FieldMask{Paths: []string{"model"}}, SummarizeConfig{Model: "gemma"})
		if err != nil || got.Model != "gemma" {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	t.Run("list models happy", func(t *testing.T) {
		c := newTestClient(t)
		c.Summarize = &fakeSummarize{listModels: func() (*summv1.ListSummarizeModelsResponse, error) {
			return &summv1.ListSummarizeModelsResponse{Models: []*summv1.SummarizeModel{{Id: "m1"}, {Id: "m2"}}}, nil
		}}
		r := &RemoteSummarizeConfigAdmin{remoteBase: remoteBase{Client: c}}
		got, err := r.ListSummarizeModels(ctx)
		if err != nil || len(got) != 2 || got[0].ID != "m1" {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	t.Run("list models empty response is an error", func(t *testing.T) {
		c := newTestClient(t)
		c.Summarize = &fakeSummarize{listModels: func() (*summv1.ListSummarizeModelsResponse, error) { return nil, nil }}
		r := &RemoteSummarizeConfigAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.ListSummarizeModels(ctx); err == nil ||
			err.Error() != "audiotools: empty list_summarize_models response" {
			t.Fatalf("want empty err, got %v", err)
		}
	})

	t.Run("list models transport failure", func(t *testing.T) {
		c := newTestClient(t)
		c.Summarize = &fakeSummarize{listModels: func() (*summv1.ListSummarizeModelsResponse, error) { return nil, unavailableErr() }}
		r := &RemoteSummarizeConfigAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.ListSummarizeModels(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
		if c.Resolved() {
			t.Error("expected re-resolve")
		}
	})
}

// -----------------------------------------------------------------------------
// RemoteTTSConfigAdmin
// -----------------------------------------------------------------------------

func TestRemoteTTSConfigAdmin(t *testing.T) {
	ctx := context.Background()

	t.Run("get nil client", func(t *testing.T) {
		r := &RemoteTTSConfigAdmin{}
		if _, err := r.GetTTSConfig(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
	})

	t.Run("get happy", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{getConfig: func() (*ttsv1.GetConfigResponse, error) {
			return &ttsv1.GetConfigResponse{Config: &ttsv1.Config{AutoEnabled: true, DefaultVoice: "af_heart"}}, nil
		}}
		r := &RemoteTTSConfigAdmin{remoteBase: remoteBase{Client: c}}
		got, err := r.GetTTSConfig(ctx)
		if err != nil || !got.AutoEnabled || got.DefaultVoice != "af_heart" {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	t.Run("update empty mask", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{}
		r := &RemoteTTSConfigAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.UpdateTTSConfig(ctx, FieldMask{}, TTSConfig{}); !errors.Is(err, audiotools.ErrInvalidArgument) {
			t.Fatalf("want ErrInvalidArgument, got %v", err)
		}
	})

	t.Run("update happy", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{updateConfig: func() (*ttsv1.UpdateConfigResponse, error) {
			return &ttsv1.UpdateConfigResponse{Config: &ttsv1.Config{DefaultVoice: "x"}}, nil
		}}
		r := &RemoteTTSConfigAdmin{remoteBase: remoteBase{Client: c}}
		got, err := r.UpdateTTSConfig(ctx, FieldMask{Paths: []string{"default_voice"}}, TTSConfig{DefaultVoice: "x"})
		if err != nil || got.DefaultVoice != "x" {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})

	t.Run("update empty response", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{updateConfig: func() (*ttsv1.UpdateConfigResponse, error) { return nil, nil }}
		r := &RemoteTTSConfigAdmin{remoteBase: remoteBase{Client: c}}
		if _, err := r.UpdateTTSConfig(ctx, FieldMask{Paths: []string{"x"}}, TTSConfig{}); err == nil ||
			err.Error() != "audiotools: empty update_tts_config response" {
			t.Fatalf("want empty err, got %v", err)
		}
	})
}
