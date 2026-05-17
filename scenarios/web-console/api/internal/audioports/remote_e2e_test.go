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

// startFakeAudioTools spins up an httptest server hosting the fake
// STT + TTS Connect services. Returns the base URL and a cleanup func.
func startFakeAudioTools(t *testing.T, stt *fakeSTT, tts *fakeTTS) (string, func()) {
	t.Helper()
	mux := http.NewServeMux()
	sttPath, sttHandler := sttconnect.NewSTTServiceHandler(stt)
	mux.Handle(sttPath, sttHandler)
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
