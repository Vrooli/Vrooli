// Package audiotools — cross-scenario integration tests.
//
// Stands up a fake audio-tools Connect server in-process and exercises the
// integrations adapter + Remote* audioports adapters against it. This is the
// canonical Phase I test surface: web-console adoption is validated without
// a real audio-tools binary running on the network.
package audioports_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"

	"web-console/integrations/audiotools"
	"web-console/internal/audioports"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	summconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
)

// fakeSTT implements the generated STTServiceHandler.
type fakeSTT struct {
	sttconnect.UnimplementedSTTServiceHandler
	mu          sync.Mutex
	transcripts []string
	lastReq     *sttv1.TranscribeRequest
}

func (f *fakeSTT) Transcribe(ctx context.Context, req *connect.Request[sttv1.TranscribeRequest]) (*connect.Response[sttv1.TranscribeResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastReq = req.Msg
	text := "fake-transcript"
	if len(f.transcripts) > 0 {
		text = f.transcripts[0]
		f.transcripts = f.transcripts[1:]
	}
	return connect.NewResponse(&sttv1.TranscribeResponse{
		Text:         text,
		ProviderTier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL,
		ProviderId:   "fake",
		ModelId:      "fake-model",
	}), nil
}

// fakeSTTAdmin returns successful empty responses. The adapter tests below
// intentionally exercise the complete admin surface without coupling this
// integration test to audio-tools' persistence implementation.
type fakeSTTAdmin struct {
	sttconnect.UnimplementedSTTAdminServiceHandler
}

func (f *fakeSTTAdmin) GetStreamConfig(context.Context, *connect.Request[sttv1.GetStreamConfigRequest]) (*connect.Response[sttv1.GetStreamConfigResponse], error) {
	return connect.NewResponse(&sttv1.GetStreamConfigResponse{}), nil
}

func (f *fakeSTTAdmin) UpdateStreamConfig(context.Context, *connect.Request[sttv1.UpdateStreamConfigRequest]) (*connect.Response[sttv1.UpdateStreamConfigResponse], error) {
	return connect.NewResponse(&sttv1.UpdateStreamConfigResponse{}), nil
}

func (f *fakeSTTAdmin) GetWakeWordConfig(context.Context, *connect.Request[sttv1.GetWakeWordConfigRequest]) (*connect.Response[sttv1.GetWakeWordConfigResponse], error) {
	return connect.NewResponse(&sttv1.GetWakeWordConfigResponse{}), nil
}

func (f *fakeSTTAdmin) UpdateWakeWordTemplate(context.Context, *connect.Request[sttv1.UpdateWakeWordTemplateRequest]) (*connect.Response[sttv1.UpdateWakeWordTemplateResponse], error) {
	return connect.NewResponse(&sttv1.UpdateWakeWordTemplateResponse{}), nil
}

func (f *fakeSTTAdmin) DeleteWakeWordTemplate(context.Context, *connect.Request[sttv1.DeleteWakeWordTemplateRequest]) (*connect.Response[sttv1.DeleteWakeWordTemplateResponse], error) {
	return connect.NewResponse(&sttv1.DeleteWakeWordTemplateResponse{}), nil
}

func (f *fakeSTTAdmin) GetSpeakerConfig(context.Context, *connect.Request[sttv1.GetSpeakerConfigRequest]) (*connect.Response[sttv1.GetSpeakerConfigResponse], error) {
	return connect.NewResponse(&sttv1.GetSpeakerConfigResponse{}), nil
}

func (f *fakeSTTAdmin) UpdateSpeakerConfig(context.Context, *connect.Request[sttv1.UpdateSpeakerConfigRequest]) (*connect.Response[sttv1.UpdateSpeakerConfigResponse], error) {
	return connect.NewResponse(&sttv1.UpdateSpeakerConfigResponse{}), nil
}

func (f *fakeSTTAdmin) GetSpeakerStatus(context.Context, *connect.Request[sttv1.GetSpeakerStatusRequest]) (*connect.Response[sttv1.GetSpeakerStatusResponse], error) {
	return connect.NewResponse(&sttv1.GetSpeakerStatusResponse{}), nil
}

func (f *fakeSTTAdmin) ListSpeakerProfiles(context.Context, *connect.Request[sttv1.ListSpeakerProfilesRequest]) (*connect.Response[sttv1.ListSpeakerProfilesResponse], error) {
	return connect.NewResponse(&sttv1.ListSpeakerProfilesResponse{}), nil
}

func (f *fakeSTTAdmin) EnrollSpeakerProfile(context.Context, *connect.Request[sttv1.EnrollSpeakerProfileRequest]) (*connect.Response[sttv1.EnrollSpeakerProfileResponse], error) {
	return connect.NewResponse(&sttv1.EnrollSpeakerProfileResponse{}), nil
}

func (f *fakeSTTAdmin) ClearSpeakerProfileBinding(context.Context, *connect.Request[sttv1.ClearSpeakerProfileBindingRequest]) (*connect.Response[sttv1.ClearSpeakerProfileBindingResponse], error) {
	return connect.NewResponse(&sttv1.ClearSpeakerProfileBindingResponse{}), nil
}

func (f *fakeSTTAdmin) UnbindSpeakerProfile(context.Context, *connect.Request[sttv1.UnbindSpeakerProfileRequest]) (*connect.Response[sttv1.UnbindSpeakerProfileResponse], error) {
	return connect.NewResponse(&sttv1.UnbindSpeakerProfileResponse{}), nil
}

func (f *fakeSTTAdmin) DeleteSpeakerProfile(context.Context, *connect.Request[sttv1.DeleteSpeakerProfileRequest]) (*connect.Response[sttv1.DeleteSpeakerProfileResponse], error) {
	return connect.NewResponse(&sttv1.DeleteSpeakerProfileResponse{}), nil
}

type fakeSummarize struct {
	summconnect.UnimplementedSummarizeServiceHandler
}

func (f *fakeSummarize) Summarize(context.Context, *connect.Request[summv1.SummarizeRequest]) (*connect.Response[summv1.SummarizeResponse], error) {
	return connect.NewResponse(&summv1.SummarizeResponse{}), nil
}

func (f *fakeSummarize) GetSummarizeConfig(context.Context, *connect.Request[summv1.GetSummarizeConfigRequest]) (*connect.Response[summv1.GetSummarizeConfigResponse], error) {
	return connect.NewResponse(&summv1.GetSummarizeConfigResponse{}), nil
}

func (f *fakeSummarize) UpdateSummarizeConfig(context.Context, *connect.Request[summv1.UpdateSummarizeConfigRequest]) (*connect.Response[summv1.UpdateSummarizeConfigResponse], error) {
	return connect.NewResponse(&summv1.UpdateSummarizeConfigResponse{}), nil
}

func (f *fakeSummarize) ListSummarizeModels(context.Context, *connect.Request[summv1.ListSummarizeModelsRequest]) (*connect.Response[summv1.ListSummarizeModelsResponse], error) {
	return connect.NewResponse(&summv1.ListSummarizeModelsResponse{}), nil
}

// fakeTTS implements the generated TTSServiceHandler.
type fakeTTS struct {
	ttsconnect.UnimplementedTTSServiceHandler
	mu         sync.Mutex
	audio      []byte
	lastReq    *ttsv1.SynthesizeRequest
	normCalls  int
	splitCalls int
}

func (f *fakeTTS) Synthesize(ctx context.Context, req *connect.Request[ttsv1.SynthesizeRequest]) (*connect.Response[ttsv1.SynthesizeResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastReq = req.Msg
	audio := f.audio
	if len(audio) == 0 {
		audio = []byte("fake-audio")
	}
	return connect.NewResponse(&ttsv1.SynthesizeResponse{
		Audio:        audio,
		ContentType:  "audio/mpeg",
		ProviderTier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL,
		ProviderId:   "fake",
		ModelId:      "fake-tts",
		VoiceUsed:    req.Msg.Voice,
	}), nil
}

func (f *fakeTTS) NormalizeForSpeech(ctx context.Context, req *connect.Request[ttsv1.NormalizeForSpeechRequest]) (*connect.Response[ttsv1.NormalizeForSpeechResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.normCalls++
	return connect.NewResponse(&ttsv1.NormalizeForSpeechResponse{Text: "normalized: " + req.Msg.Text}), nil
}

func (f *fakeTTS) SplitParagraphs(ctx context.Context, req *connect.Request[ttsv1.SplitParagraphsRequest]) (*connect.Response[ttsv1.SplitParagraphsResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.splitCalls++
	return connect.NewResponse(&ttsv1.SplitParagraphsResponse{Paragraphs: []string{req.Msg.Text}}), nil
}

func (f *fakeTTS) ListVoices(context.Context, *connect.Request[ttsv1.ListVoicesRequest]) (*connect.Response[ttsv1.ListVoicesResponse], error) {
	return connect.NewResponse(&ttsv1.ListVoicesResponse{}), nil
}

func (f *fakeTTS) GetCache(context.Context, *connect.Request[ttsv1.GetCacheRequest]) (*connect.Response[ttsv1.GetCacheResponse], error) {
	return connect.NewResponse(&ttsv1.GetCacheResponse{}), nil
}

func (f *fakeTTS) GetConfig(context.Context, *connect.Request[ttsv1.GetConfigRequest]) (*connect.Response[ttsv1.GetConfigResponse], error) {
	return connect.NewResponse(&ttsv1.GetConfigResponse{}), nil
}

func (f *fakeTTS) UpdateConfig(context.Context, *connect.Request[ttsv1.UpdateConfigRequest]) (*connect.Response[ttsv1.UpdateConfigResponse], error) {
	return connect.NewResponse(&ttsv1.UpdateConfigResponse{}), nil
}

// startFakeAudioTools spins up an httptest server hosting the fake
// STT + TTS Connect services. Returns the base URL and a cleanup func.
func startFakeAudioTools(t *testing.T, stt *fakeSTT, tts *fakeTTS) (string, func()) {
	t.Helper()
	mux := http.NewServeMux()
	sttPath, sttHandler := sttconnect.NewSTTServiceHandler(stt)
	mux.Handle(sttPath, sttHandler)
	adminPath, adminHandler := sttconnect.NewSTTAdminServiceHandler(&fakeSTTAdmin{})
	mux.Handle(adminPath, adminHandler)
	summarizePath, summarizeHandler := summconnect.NewSummarizeServiceHandler(&fakeSummarize{})
	mux.Handle(summarizePath, summarizeHandler)
	ttsPath, ttsHandler := ttsconnect.NewTTSServiceHandler(tts)
	mux.Handle(ttsPath, ttsHandler)
	srv := httptest.NewServer(mux)
	return srv.URL, srv.Close
}

type staticResolver struct{ url string }

func (s staticResolver) Resolve() (string, error) { return s.url, nil }

func TestRemoteSTT_RoundTrip(t *testing.T) {
	stt := &fakeSTT{transcripts: []string{"hello world"}}
	tts := &fakeTTS{}
	baseURL, cleanup := startFakeAudioTools(t, stt, tts)
	defer cleanup()

	client, err := audiotools.New(staticResolver{url: baseURL}, audiotools.Policy{Required: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	adapter := &audioports.RemoteSpeechToText{Client: client}

	res, err := adapter.Transcribe(context.Background(), []byte("audio-bytes"), audioports.STTOptions{Language: "en"})
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if res.Text != "hello world" {
		t.Errorf("text = %q; want %q", res.Text, "hello world")
	}
	if stt.lastReq == nil || stt.lastReq.Language != "en" {
		t.Errorf("language not propagated: %+v", stt.lastReq)
	}
}

func TestRemoteTTS_RoundTrip(t *testing.T) {
	stt := &fakeSTT{}
	tts := &fakeTTS{audio: []byte{0x01, 0x02, 0x03}}
	baseURL, cleanup := startFakeAudioTools(t, stt, tts)
	defer cleanup()

	client, err := audiotools.New(staticResolver{url: baseURL}, audiotools.Policy{Required: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	adapter := &audioports.RemoteTextToSpeech{Client: client}

	res, err := adapter.Synthesize(context.Background(), audioports.TTSRequest{
		Input:          "hello",
		Voice:          "voice.feminine.warm",
		ResponseFormat: "mp3",
		Speed:          1.0,
	})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if len(res.Audio) != 3 {
		t.Errorf("audio length = %d; want 3", len(res.Audio))
	}
	if res.ContentType != "audio/mpeg" {
		t.Errorf("content_type = %q; want audio/mpeg", res.ContentType)
	}
	if tts.lastReq == nil || tts.lastReq.Voice != "voice.feminine.warm" {
		t.Errorf("voice not propagated: %+v", tts.lastReq)
	}
}

func TestRemoteAdminAndAuxiliaryPorts_RoundTrip(t *testing.T) {
	baseURL, cleanup := startFakeAudioTools(t, &fakeSTT{}, &fakeTTS{})
	defer cleanup()
	client, err := audiotools.New(staticResolver{url: baseURL}, audiotools.Policy{Required: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	stream := &audioports.RemoteStreamConfigAdmin{Client: client}
	if _, err := stream.GetStreamConfig(ctx); err != nil {
		t.Fatalf("GetStreamConfig: %v", err)
	}
	if _, err := stream.UpdateStreamConfig(ctx, audioports.FieldMask{Paths: []string{"vad_silence_ms"}}, audioports.StreamConfig{}); err != nil {
		t.Fatalf("UpdateStreamConfig: %v", err)
	}

	wake := &audioports.RemoteWakeWordAdmin{Client: client}
	if _, err := wake.GetWakeWordConfig(ctx); err != nil {
		t.Fatalf("GetWakeWordConfig: %v", err)
	}
	if _, err := wake.UpdateWakeWordTemplate(ctx, audioports.WakeWordTemplate{Label: "hey", Threshold: 0.7}); err != nil {
		t.Fatalf("UpdateWakeWordTemplate: %v", err)
	}
	if _, err := wake.DeleteWakeWordTemplate(ctx); err != nil {
		t.Fatalf("DeleteWakeWordTemplate: %v", err)
	}

	speaker := &audioports.RemoteSpeakerAdmin{Client: client}
	if _, err := speaker.GetSpeakerConfig(ctx); err != nil {
		t.Fatalf("GetSpeakerConfig: %v", err)
	}
	if _, err := speaker.UpdateSpeakerConfig(ctx, audioports.FieldMask{Paths: []string{"enabled"}}, audioports.SpeakerConfig{}); err != nil {
		t.Fatalf("UpdateSpeakerConfig: %v", err)
	}
	if _, err := speaker.GetSpeakerStatus(ctx); err != nil {
		t.Fatalf("GetSpeakerStatus: %v", err)
	}
	if _, err := speaker.ListSpeakerProfiles(ctx); err != nil {
		t.Fatalf("ListSpeakerProfiles: %v", err)
	}
	if _, err := speaker.EnrollSpeakerProfile(ctx, audioports.EnrollSpeakerInput{Audio: []byte("audio"), Format: audioports.AudioFormatWAV}); err != nil {
		t.Fatalf("EnrollSpeakerProfile: %v", err)
	}
	if _, err := speaker.ClearSpeakerProfileBinding(ctx); err != nil {
		t.Fatalf("ClearSpeakerProfileBinding: %v", err)
	}
	if _, err := speaker.UnbindSpeakerProfile(ctx, "profile"); err != nil {
		t.Fatalf("UnbindSpeakerProfile: %v", err)
	}
	if _, err := speaker.DeleteSpeakerProfile(ctx, "profile"); err != nil {
		t.Fatalf("DeleteSpeakerProfile: %v", err)
	}

	tts := &audioports.RemoteTextToSpeech{Client: client}
	if _, err := tts.ListVoices(ctx); err != nil {
		t.Fatalf("ListVoices: %v", err)
	}
	if _, hit := tts.GetCached(ctx, audioports.CacheLookup{EventID: "event"}); hit {
		t.Fatal("empty cache response unexpectedly hit")
	}
	ttsConfig := &audioports.RemoteTTSConfigAdmin{Client: client}
	if _, err := ttsConfig.GetTTSConfig(ctx); err != nil {
		t.Fatalf("GetTTSConfig: %v", err)
	}
	if _, err := ttsConfig.UpdateTTSConfig(ctx, audioports.FieldMask{Paths: []string{"auto_enabled"}}, audioports.TTSConfig{}); err != nil {
		t.Fatalf("UpdateTTSConfig: %v", err)
	}

	summarizeConfig := &audioports.RemoteSummarizeConfigAdmin{Client: client}
	if _, err := summarizeConfig.GetSummarizeConfig(ctx); err != nil {
		t.Fatalf("GetSummarizeConfig: %v", err)
	}
	if _, err := summarizeConfig.UpdateSummarizeConfig(ctx, audioports.FieldMask{Paths: []string{"enabled"}}, audioports.SummarizeConfig{}); err != nil {
		t.Fatalf("UpdateSummarizeConfig: %v", err)
	}
	if _, err := summarizeConfig.ListSummarizeModels(ctx); err != nil {
		t.Fatalf("ListSummarizeModels: %v", err)
	}
	if _, err := (&audioports.RemoteSummarizer{Client: client}).Summarize(ctx, audioports.SummarizeInput{Text: "long text", Level: "moderate", TimeoutSeconds: 5}); err != nil {
		t.Fatalf("Summarize: %v", err)
	}

	if got := (&audioports.RemoteSpeechTextProcessor{Client: client}).SplitIntoParagraphs("hello"); len(got) != 1 || got[0] != "hello" {
		t.Fatalf("SplitIntoParagraphs = %#v", got)
	}
}

func TestRemoteSpeechTextProcessor_CachesNormalize(t *testing.T) {
	stt := &fakeSTT{}
	tts := &fakeTTS{}
	baseURL, cleanup := startFakeAudioTools(t, stt, tts)
	defer cleanup()

	client, err := audiotools.New(staticResolver{url: baseURL}, audiotools.Policy{Required: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	proc := &audioports.RemoteSpeechTextProcessor{Client: client}

	out1 := proc.NormalizeForSpeech("hello")
	out2 := proc.NormalizeForSpeech("hello") // should hit cache
	if out1 != "normalized: hello" || out2 != out1 {
		t.Errorf("got (%q, %q); want both %q", out1, out2, "normalized: hello")
	}
	tts.mu.Lock()
	if tts.normCalls != 1 {
		t.Errorf("normCalls = %d; want 1 (second call should have been cached)", tts.normCalls)
	}
	tts.mu.Unlock()
}

func TestNormalizeError_MapsConnectCodes(t *testing.T) {
	cases := []struct {
		code     connect.Code
		expected error
	}{
		{connect.CodeUnavailable, audiotools.ErrUnavailable},
		{connect.CodeDeadlineExceeded, audiotools.ErrTimeout},
		{connect.CodeFailedPrecondition, audiotools.ErrFailedPrecondition},
		{connect.CodeResourceExhausted, audiotools.ErrInsufficientCredits},
		{connect.CodeInvalidArgument, audiotools.ErrInvalidArgument},
	}
	for _, tc := range cases {
		raw := connect.NewError(tc.code, nil)
		got := audiotools.NormalizeError(raw)
		if got != tc.expected {
			t.Errorf("code %s: got %v; want %v", tc.code, got, tc.expected)
		}
	}
}

func TestClient_HandlesTransportFailure_Reresolves(t *testing.T) {
	stt := &fakeSTT{}
	tts := &fakeTTS{}
	baseURL, cleanup := startFakeAudioTools(t, stt, tts)
	defer cleanup()

	c, err := audiotools.New(staticResolver{url: baseURL}, audiotools.Policy{Required: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if !c.Resolved() {
		t.Fatal("client not resolved after New")
	}
	c.HandleTransportFailure()
	if c.Resolved() {
		t.Error("resolved flag still set after HandleTransportFailure")
	}
	if err := c.Ensure(); err != nil {
		t.Fatalf("Ensure after invalidate: %v", err)
	}
	if !c.Resolved() {
		t.Error("Ensure did not re-resolve")
	}
}
