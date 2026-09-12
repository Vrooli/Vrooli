package audio_runtime

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"web-console/integrations/audiotools"
	"web-console/internal/audioports"

	audiocommonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_common"
	audioruntimev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_runtime"
)

type fakeSTT struct {
	lastOpts audioports.STTOptions
	result   audioports.STTResult
	err      error
}

func (f *fakeSTT) Transcribe(_ context.Context, _ []byte, opts audioports.STTOptions) (audioports.STTResult, error) {
	f.lastOpts = opts
	return f.result, f.err
}

type fakePlayback struct{ last audioports.PlaybackEvent }

func (f *fakePlayback) RecordPlaybackEvent(_ context.Context, ev audioports.PlaybackEvent) error {
	f.last = ev
	return nil
}

type fakeTTS struct {
	last   audioports.TTSRequest
	result audioports.TTSResult
	voices []audioports.Voice
	hit    bool
	err    error
}

func (f *fakeTTS) Synthesize(_ context.Context, req audioports.TTSRequest) (audioports.TTSResult, error) {
	f.last = req
	return f.result, f.err
}
func (f *fakeTTS) ListVoices(context.Context) ([]audioports.Voice, error) { return f.voices, f.err }
func (f *fakeTTS) GetCached(context.Context, audioports.CacheLookup) (audioports.TTSResult, bool) {
	return f.result, f.hit
}

type fakeSumm struct {
	last   audioports.SummarizeInput
	result audioports.SummarizeOutput
	err    error
}

func (f *fakeSumm) Summarize(_ context.Context, in audioports.SummarizeInput) (audioports.SummarizeOutput, error) {
	f.last = in
	return f.result, f.err
}

func TestTranscribe_HappyPath(t *testing.T) {
	stt := &fakeSTT{result: audioports.STTResult{Text: "hello world"}}
	h := NewConnectHandler(Deps{STT: stt})
	resp, err := h.Transcribe(context.Background(), connect.NewRequest(&audioruntimev1.TranscribeRequest{
		Audio:                   []byte{0x01},
		Format:                  audiocommonv1.AudioFormat_AUDIO_FORMAT_WEBM,
		SkipSpeakerVerification: true,
		Language:                "en",
	}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Msg.Text != "hello world" {
		t.Errorf("text: got %q, want %q", resp.Msg.Text, "hello world")
	}
	if !stt.lastOpts.SkipSpeakerVerification || stt.lastOpts.Language != "en" {
		t.Errorf("opts forwarded incorrectly: %+v", stt.lastOpts)
	}
}

func TestTranscribe_EmptyAudio_InvalidArgument(t *testing.T) {
	h := NewConnectHandler(Deps{STT: &fakeSTT{}})
	_, err := h.Transcribe(context.Background(), connect.NewRequest(&audioruntimev1.TranscribeRequest{}))
	if err == nil {
		t.Fatal("expected error")
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeInvalidArgument {
		t.Errorf("err: got %v, want CodeInvalidArgument", err)
	}
}

func TestRecordPlaybackEvent_PassesFields(t *testing.T) {
	pb := &fakePlayback{}
	h := NewConnectHandler(Deps{Playback: pb})
	_, err := h.RecordPlaybackEvent(context.Background(), connect.NewRequest(&audioruntimev1.RecordPlaybackEventRequest{
		Event: &audioruntimev1.PlaybackEvent{
			Source:    "ui",
			Stage:     "start",
			Backend:   "kokoro",
			SessionId: "sess-1",
			EventId:   "evt-1",
		},
	}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if pb.last.Source != "ui" || pb.last.Stage != "start" || pb.last.EventID != "evt-1" {
		t.Errorf("playback event forwarded incorrectly: %+v", pb.last)
	}
}

func TestRuntimeTTSCacheAndSummarize(t *testing.T) {
	tts := &fakeTTS{result: audioports.TTSResult{Audio: []byte{1, 2}, ContentType: "audio/wav"}, voices: []audioports.Voice{{ID: "v1", Name: "Voice"}}, hit: true}
	summ := &fakeSumm{result: audioports.SummarizeOutput{Text: "short", PromptTokens: 2, OutputTokens: 3}}
	h := NewConnectHandler(Deps{TTS: tts, Summ: summ})
	ctx := context.Background()
	resp, err := h.Synthesize(ctx, connect.NewRequest(&audioruntimev1.SynthesizeRequest{Text: "hello", Voice: "v1", Speed: 1.1, ResponseFormat: audiocommonv1.ResponseFormat_RESPONSE_FORMAT_WAV, EventId: "e", Version: "1", ChunkIndex: 2}))
	if err != nil || string(resp.Msg.Audio) != string([]byte{1, 2}) || tts.last.ResponseFormat != "wav" || tts.last.ChunkIndex != 2 {
		t.Fatalf("synthesize: %#v %v req=%+v", resp, err, tts.last)
	}
	voices, err := h.ListVoices(ctx, connect.NewRequest(&audioruntimev1.ListVoicesRequest{}))
	if err != nil || len(voices.Msg.Voices) != 1 || voices.Msg.Voices[0].Id != "v1" {
		t.Fatalf("voices: %#v %v", voices, err)
	}
	cache, err := h.GetTTSCache(ctx, connect.NewRequest(&audioruntimev1.GetTTSCacheRequest{EventId: "e"}))
	if err != nil || !cache.Msg.Hit {
		t.Fatalf("cache hit: %#v %v", cache, err)
	}
	tts.hit = false
	cache, err = h.GetTTSCache(ctx, connect.NewRequest(&audioruntimev1.GetTTSCacheRequest{}))
	if err != nil || cache.Msg.Hit {
		t.Fatalf("cache miss: %#v %v", cache, err)
	}
	summary, err := h.Summarize(ctx, connect.NewRequest(&audioruntimev1.SummarizeRequest{Text: "long", Level: audiocommonv1.SummarizeLevel_SUMMARIZE_LEVEL_HEAVY, Model: "m", TimeoutSeconds: 9}))
	if err != nil || summary.Msg.Text != "short" || summ.last.Level != "heavy" || summ.last.TimeoutSeconds != 9 {
		t.Fatalf("summarize: %#v %v input=%+v", summary, err, summ.last)
	}
}

func TestRuntimeUnavailableAndMappedErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		deps Deps
		call func(*connectHandler) error
		want connect.Code
	}{
		{"stt nil", Deps{}, func(h *connectHandler) error {
			_, e := h.Transcribe(context.Background(), connect.NewRequest(&audioruntimev1.TranscribeRequest{Audio: []byte{1}}))
			return e
		}, connect.CodeUnavailable},
		{"tts nil", Deps{}, func(h *connectHandler) error {
			_, e := h.Synthesize(context.Background(), connect.NewRequest(&audioruntimev1.SynthesizeRequest{}))
			return e
		}, connect.CodeUnavailable},
		{"voices nil", Deps{}, func(h *connectHandler) error {
			_, e := h.ListVoices(context.Background(), connect.NewRequest(&audioruntimev1.ListVoicesRequest{}))
			return e
		}, connect.CodeUnavailable},
		{"cache nil", Deps{}, func(h *connectHandler) error {
			_, e := h.GetTTSCache(context.Background(), connect.NewRequest(&audioruntimev1.GetTTSCacheRequest{}))
			return e
		}, connect.CodeUnavailable},
		{"playback nil", Deps{}, func(h *connectHandler) error {
			_, e := h.RecordPlaybackEvent(context.Background(), connect.NewRequest(&audioruntimev1.RecordPlaybackEventRequest{Event: &audioruntimev1.PlaybackEvent{}}))
			return e
		}, connect.CodeUnavailable},
		{"summ nil", Deps{}, func(h *connectHandler) error {
			_, e := h.Summarize(context.Background(), connect.NewRequest(&audioruntimev1.SummarizeRequest{Text: "x"}))
			return e
		}, connect.CodeUnavailable},
		{"playback missing event", Deps{Playback: &fakePlayback{}}, func(h *connectHandler) error {
			_, e := h.RecordPlaybackEvent(context.Background(), connect.NewRequest(&audioruntimev1.RecordPlaybackEventRequest{}))
			return e
		}, connect.CodeInvalidArgument},
		{"summ missing text", Deps{Summ: &fakeSumm{}}, func(h *connectHandler) error {
			_, e := h.Summarize(context.Background(), connect.NewRequest(&audioruntimev1.SummarizeRequest{}))
			return e
		}, connect.CodeInvalidArgument},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var ce *connect.Error
			if err := tc.call(NewConnectHandler(tc.deps)); !errors.As(err, &ce) || ce.Code() != tc.want {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
	for _, errValue := range []error{audiotools.ErrTimeout, audiotools.ErrUnavailable, audiotools.ErrFailedPrecondition, audiotools.ErrInsufficientCredits, audiotools.ErrInvalidArgument, errors.New("other")} {
		var ce *connect.Error
		if err := mapErr(errValue); !errors.As(err, &ce) {
			t.Fatalf("mapErr(%v) did not return connect error", errValue)
		}
	}
	for i, want := range []string{"", "mp3", "wav", "opus", "flac", ""} {
		if got := responseFormatStr(int32(i)); got != want {
			t.Errorf("format %d: got %q want %q", i, got, want)
		}
	}
	for i, want := range []string{"", "light", "moderate", "heavy", ""} {
		if got := summarizeLevelToString(int32(i)); got != want {
			t.Errorf("level %d: got %q want %q", i, got, want)
		}
	}
}
