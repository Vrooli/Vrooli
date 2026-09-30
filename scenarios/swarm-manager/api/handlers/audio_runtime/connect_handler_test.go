package audio_runtime

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"swarm-manager/integrations/audiotools"
	"swarm-manager/internal/audioports"

	audioruntimev1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/audio_runtime"
)

// -----------------------------------------------------------------------------
// Fakes
// -----------------------------------------------------------------------------

type fakeSTT struct {
	result audioports.STTResult
	err    error

	gotAudio []byte
	gotOpts  audioports.STTOptions
}

func (f *fakeSTT) Transcribe(_ context.Context, audio []byte, opts audioports.STTOptions) (audioports.STTResult, error) {
	f.gotAudio = audio
	f.gotOpts = opts
	return f.result, f.err
}

type fakeTTS struct {
	synthResult audioports.TTSResult
	synthErr    error
	gotReq      audioports.TTSRequest

	voices    []audioports.Voice
	voicesErr error

	cacheResult audioports.TTSResult
	cacheHit    bool
	gotLookup   audioports.CacheLookup
}

func (f *fakeTTS) Synthesize(_ context.Context, req audioports.TTSRequest) (audioports.TTSResult, error) {
	f.gotReq = req
	return f.synthResult, f.synthErr
}

func (f *fakeTTS) ListVoices(_ context.Context) ([]audioports.Voice, error) {
	return f.voices, f.voicesErr
}

func (f *fakeTTS) GetCached(_ context.Context, key audioports.CacheLookup) (audioports.TTSResult, bool) {
	f.gotLookup = key
	return f.cacheResult, f.cacheHit
}

type fakePlayback struct {
	err   error
	gotEv audioports.PlaybackEvent
}

func (f *fakePlayback) RecordPlaybackEvent(_ context.Context, ev audioports.PlaybackEvent) error {
	f.gotEv = ev
	return f.err
}

type fakeSummarizer struct {
	result audioports.SummarizeOutput
	err    error
	gotIn  audioports.SummarizeInput
}

func (f *fakeSummarizer) Summarize(_ context.Context, in audioports.SummarizeInput) (audioports.SummarizeOutput, error) {
	f.gotIn = in
	return f.result, f.err
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

// -----------------------------------------------------------------------------
// Pure helpers
// -----------------------------------------------------------------------------

func TestResponseFormatStr(t *testing.T) {
	cases := map[int32]string{
		0: "",
		1: "mp3",
		2: "wav",
		3: "opus",
		4: "flac",
		5: "", // out of range -> default
	}
	for in, want := range cases {
		if got := responseFormatStr(in); got != want {
			t.Errorf("responseFormatStr(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestSummarizeLevelToString(t *testing.T) {
	cases := map[int32]string{
		0: "",
		1: "light",
		2: "moderate",
		3: "heavy",
		4: "", // out of range -> default
	}
	for in, want := range cases {
		if got := summarizeLevelToString(in); got != want {
			t.Errorf("summarizeLevelToString(%d) = %q, want %q", in, got, want)
		}
	}
}

// -----------------------------------------------------------------------------
// Transcribe
// -----------------------------------------------------------------------------

func TestTranscribe_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.Transcribe(context.Background(), connect.NewRequest(&audioruntimev1.TranscribeRequest{Audio: []byte("x")}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestTranscribe_EmptyAudio(t *testing.T) {
	h := NewConnectHandler(Deps{STT: &fakeSTT{}})
	_, err := h.Transcribe(context.Background(), connect.NewRequest(&audioruntimev1.TranscribeRequest{}))
	assertCode(t, err, connect.CodeInvalidArgument)
}

func TestTranscribe_HappyPath(t *testing.T) {
	stt := &fakeSTT{result: audioports.STTResult{Text: "hello world"}}
	h := NewConnectHandler(Deps{STT: stt})
	resp, err := h.Transcribe(context.Background(), connect.NewRequest(&audioruntimev1.TranscribeRequest{
		Audio:                   []byte("abc"),
		Language:                "en",
		SkipSpeakerVerification: true,
		InitialPrompt:           "prompt",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Text != "hello world" {
		t.Errorf("Text = %q, want %q", resp.Msg.Text, "hello world")
	}
	// Verify options were threaded through.
	if string(stt.gotAudio) != "abc" {
		t.Errorf("gotAudio = %q", stt.gotAudio)
	}
	if stt.gotOpts.Language != "en" || !stt.gotOpts.SkipSpeakerVerification || stt.gotOpts.InitialPrompt != "prompt" {
		t.Errorf("opts not threaded: %+v", stt.gotOpts)
	}
}

func TestTranscribe_ErrorMapping(t *testing.T) {
	h := NewConnectHandler(Deps{STT: &fakeSTT{err: audiotools.ErrTimeout}})
	_, err := h.Transcribe(context.Background(), connect.NewRequest(&audioruntimev1.TranscribeRequest{Audio: []byte("x")}))
	assertCode(t, err, connect.CodeDeadlineExceeded)
}

// -----------------------------------------------------------------------------
// Synthesize
// -----------------------------------------------------------------------------

func TestSynthesize_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.Synthesize(context.Background(), connect.NewRequest(&audioruntimev1.SynthesizeRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestSynthesize_HappyPath(t *testing.T) {
	tts := &fakeTTS{synthResult: audioports.TTSResult{Audio: []byte("pcm"), ContentType: "audio/mpeg"}}
	h := NewConnectHandler(Deps{TTS: tts})
	resp, err := h.Synthesize(context.Background(), connect.NewRequest(&audioruntimev1.SynthesizeRequest{
		Text:           "say this",
		Voice:          "af_heart",
		Speed:          1.5,
		ResponseFormat: 1, // mp3
		EventId:        "ev1",
		Version:        "v2",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(resp.Msg.Audio) != "pcm" || resp.Msg.ContentType != "audio/mpeg" {
		t.Errorf("response = %q/%q", resp.Msg.Audio, resp.Msg.ContentType)
	}
	if tts.gotReq.Input != "say this" || tts.gotReq.Voice != "af_heart" || tts.gotReq.Speed != 1.5 {
		t.Errorf("req not threaded: %+v", tts.gotReq)
	}
	if tts.gotReq.ResponseFormat != "mp3" {
		t.Errorf("ResponseFormat = %q, want mp3", tts.gotReq.ResponseFormat)
	}
	if tts.gotReq.EventID != "ev1" || tts.gotReq.Version != "v2" {
		t.Errorf("event/version not threaded: %+v", tts.gotReq)
	}
}

func TestSynthesize_ErrorMapping(t *testing.T) {
	h := NewConnectHandler(Deps{TTS: &fakeTTS{synthErr: audiotools.ErrInsufficientCredits}})
	_, err := h.Synthesize(context.Background(), connect.NewRequest(&audioruntimev1.SynthesizeRequest{}))
	assertCode(t, err, connect.CodeResourceExhausted)
}

// -----------------------------------------------------------------------------
// ListVoices
// -----------------------------------------------------------------------------

func TestListVoices_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.ListVoices(context.Background(), connect.NewRequest(&audioruntimev1.ListVoicesRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestListVoices_HappyPath(t *testing.T) {
	tts := &fakeTTS{voices: []audioports.Voice{{ID: "a", Name: "Alpha"}, {ID: "b", Name: "Beta"}}}
	h := NewConnectHandler(Deps{TTS: tts})
	resp, err := h.ListVoices(context.Background(), connect.NewRequest(&audioruntimev1.ListVoicesRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Voices) != 2 {
		t.Fatalf("got %d voices, want 2", len(resp.Msg.Voices))
	}
	if resp.Msg.Voices[0].Id != "a" || resp.Msg.Voices[0].Name != "Alpha" {
		t.Errorf("voice[0] = %+v", resp.Msg.Voices[0])
	}
	if resp.Msg.Voices[1].Id != "b" || resp.Msg.Voices[1].Name != "Beta" {
		t.Errorf("voice[1] = %+v", resp.Msg.Voices[1])
	}
}

func TestListVoices_ErrorMapping(t *testing.T) {
	h := NewConnectHandler(Deps{TTS: &fakeTTS{voicesErr: audiotools.ErrUnavailable}})
	_, err := h.ListVoices(context.Background(), connect.NewRequest(&audioruntimev1.ListVoicesRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

// -----------------------------------------------------------------------------
// GetTTSCache
// -----------------------------------------------------------------------------

func TestGetTTSCache_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.GetTTSCache(context.Background(), connect.NewRequest(&audioruntimev1.GetTTSCacheRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestGetTTSCache_Miss(t *testing.T) {
	tts := &fakeTTS{cacheHit: false}
	h := NewConnectHandler(Deps{TTS: tts})
	resp, err := h.GetTTSCache(context.Background(), connect.NewRequest(&audioruntimev1.GetTTSCacheRequest{
		EventId: "ev",
		Voice:   "v",
		Speed:   2.0,
		Version: "ver",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Hit {
		t.Errorf("expected miss, got Hit=true")
	}
	if len(resp.Msg.Audio) != 0 {
		t.Errorf("expected no audio on miss")
	}
	if tts.gotLookup.EventID != "ev" || tts.gotLookup.Voice != "v" || tts.gotLookup.Speed != 2.0 || tts.gotLookup.Version != "ver" {
		t.Errorf("lookup not threaded: %+v", tts.gotLookup)
	}
}

func TestGetTTSCache_Hit(t *testing.T) {
	tts := &fakeTTS{cacheHit: true, cacheResult: audioports.TTSResult{Audio: []byte("cached"), ContentType: "audio/wav"}}
	h := NewConnectHandler(Deps{TTS: tts})
	resp, err := h.GetTTSCache(context.Background(), connect.NewRequest(&audioruntimev1.GetTTSCacheRequest{EventId: "ev"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.Hit {
		t.Errorf("expected Hit=true")
	}
	if string(resp.Msg.Audio) != "cached" || resp.Msg.ContentType != "audio/wav" {
		t.Errorf("response = %q/%q", resp.Msg.Audio, resp.Msg.ContentType)
	}
}

// -----------------------------------------------------------------------------
// RecordPlaybackEvent
// -----------------------------------------------------------------------------

func TestRecordPlaybackEvent_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.RecordPlaybackEvent(context.Background(), connect.NewRequest(&audioruntimev1.RecordPlaybackEventRequest{
		Event: &audioruntimev1.PlaybackEvent{},
	}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestRecordPlaybackEvent_NilEvent(t *testing.T) {
	h := NewConnectHandler(Deps{Playback: &fakePlayback{}})
	_, err := h.RecordPlaybackEvent(context.Background(), connect.NewRequest(&audioruntimev1.RecordPlaybackEventRequest{}))
	assertCode(t, err, connect.CodeInvalidArgument)
}

func TestRecordPlaybackEvent_HappyPath(t *testing.T) {
	pb := &fakePlayback{}
	h := NewConnectHandler(Deps{Playback: pb})
	resp, err := h.RecordPlaybackEvent(context.Background(), connect.NewRequest(&audioruntimev1.RecordPlaybackEventRequest{
		Event: &audioruntimev1.PlaybackEvent{
			Source:    "ui",
			Stage:     "start",
			Backend:   "kokoro",
			SessionId: "s1",
			Message:   "msg",
			EventId:   "e1",
		},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Status != "ok" {
		t.Errorf("Status = %q, want ok", resp.Msg.Status)
	}
	if pb.gotEv.Source != "ui" || pb.gotEv.Stage != "start" || pb.gotEv.Backend != "kokoro" ||
		pb.gotEv.SessionID != "s1" || pb.gotEv.Message != "msg" || pb.gotEv.EventID != "e1" {
		t.Errorf("event not threaded: %+v", pb.gotEv)
	}
}

func TestRecordPlaybackEvent_ErrorMapping(t *testing.T) {
	h := NewConnectHandler(Deps{Playback: &fakePlayback{err: audiotools.ErrFailedPrecondition}})
	_, err := h.RecordPlaybackEvent(context.Background(), connect.NewRequest(&audioruntimev1.RecordPlaybackEventRequest{
		Event: &audioruntimev1.PlaybackEvent{},
	}))
	assertCode(t, err, connect.CodeFailedPrecondition)
}

// -----------------------------------------------------------------------------
// Summarize
// -----------------------------------------------------------------------------

func TestSummarize_NilDep(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.Summarize(context.Background(), connect.NewRequest(&audioruntimev1.SummarizeRequest{Text: "x"}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestSummarize_EmptyText(t *testing.T) {
	h := NewConnectHandler(Deps{Summ: &fakeSummarizer{}})
	_, err := h.Summarize(context.Background(), connect.NewRequest(&audioruntimev1.SummarizeRequest{}))
	assertCode(t, err, connect.CodeInvalidArgument)
}

func TestSummarize_HappyPath(t *testing.T) {
	sm := &fakeSummarizer{result: audioports.SummarizeOutput{Text: "short", PromptTokens: 100, OutputTokens: 20}}
	h := NewConnectHandler(Deps{Summ: sm})
	resp, err := h.Summarize(context.Background(), connect.NewRequest(&audioruntimev1.SummarizeRequest{
		Text:           "long text",
		Level:          2, // moderate
		Model:          "qwen",
		TimeoutSeconds: 30,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Text != "short" || resp.Msg.PromptTokens != 100 || resp.Msg.OutputTokens != 20 {
		t.Errorf("response = %+v", resp.Msg)
	}
	if sm.gotIn.Text != "long text" || sm.gotIn.Level != "moderate" || sm.gotIn.Model != "qwen" || sm.gotIn.TimeoutSeconds != 30 {
		t.Errorf("input not threaded: %+v", sm.gotIn)
	}
}

func TestSummarize_ErrorMapping(t *testing.T) {
	h := NewConnectHandler(Deps{Summ: &fakeSummarizer{err: audiotools.ErrInvalidArgument}})
	_, err := h.Summarize(context.Background(), connect.NewRequest(&audioruntimev1.SummarizeRequest{Text: "x"}))
	assertCode(t, err, connect.CodeInvalidArgument)
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

	// A pre-wrapped *connect.Error passes through unchanged.
	wrapped := connect.NewError(connect.CodeNotFound, errors.New("missing"))
	got := mapErr(wrapped)
	if connect.CodeOf(got) != connect.CodeNotFound {
		t.Errorf("pre-wrapped connect error not passed through: %v", got)
	}
}
