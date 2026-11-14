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
// isTransportFailure
// -----------------------------------------------------------------------------

func TestIsTransportFailure(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"unavailable", connect.NewError(connect.CodeUnavailable, errors.New("x")), true},
		{"deadline", connect.NewError(connect.CodeDeadlineExceeded, errors.New("x")), true},
		{"internal", connect.NewError(connect.CodeInternal, errors.New("x")), false},
		{"plain", errors.New("not connect"), false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		if got := isTransportFailure(c.err); got != c.want {
			t.Errorf("%s: isTransportFailure = %v, want %v", c.name, got, c.want)
		}
	}
}

// -----------------------------------------------------------------------------
// RemoteSpeechToText.Transcribe
// -----------------------------------------------------------------------------

func TestRemoteSTTTranscribe(t *testing.T) {
	ctx := context.Background()

	t.Run("nil client", func(t *testing.T) {
		r := &RemoteSpeechToText{}
		_, err := r.Transcribe(ctx, nil, STTOptions{})
		if !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
	})

	t.Run("happy path", func(t *testing.T) {
		c := newTestClient(t)
		c.STT = &fakeSTT{resp: &sttv1.TranscribeResponse{Text: "hello world"}}
		r := &RemoteSpeechToText{Client: c}
		got, err := r.Transcribe(ctx, []byte{1}, STTOptions{Language: "en"})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if got.Text != "hello world" {
			t.Errorf("Text = %q", got.Text)
		}
	})

	t.Run("empty response", func(t *testing.T) {
		c := newTestClient(t)
		c.STT = &fakeSTT{resp: nil} // returns nil,nil -> empty response branch
		r := &RemoteSpeechToText{Client: c}
		_, err := r.Transcribe(ctx, nil, STTOptions{})
		if err == nil || err.Error() != "audiotools: empty transcribe response" {
			t.Fatalf("want empty response error, got %v", err)
		}
	})

	t.Run("transport failure normalizes + re-resolves", func(t *testing.T) {
		c := newTestClient(t)
		c.STT = &fakeSTT{err: unavailableErr()}
		r := &RemoteSpeechToText{Client: c}
		_, err := r.Transcribe(ctx, nil, STTOptions{})
		if !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want normalized ErrUnavailable, got %v", err)
		}
		if c.Resolved() {
			t.Error("expected HandleTransportFailure to mark client unresolved")
		}
	})

	t.Run("non-transport error does not re-resolve", func(t *testing.T) {
		c := newTestClient(t)
		c.STT = &fakeSTT{err: connect.NewError(connect.CodeInvalidArgument, errors.New("bad"))}
		r := &RemoteSpeechToText{Client: c}
		_, err := r.Transcribe(ctx, nil, STTOptions{})
		if !errors.Is(err, audiotools.ErrInvalidArgument) {
			t.Fatalf("want ErrInvalidArgument, got %v", err)
		}
		if !c.Resolved() {
			t.Error("non-transport error must not unresolve the client")
		}
	})
}

// -----------------------------------------------------------------------------
// RemoteTextToSpeech
// -----------------------------------------------------------------------------

func TestRemoteTTSSynthesize(t *testing.T) {
	ctx := context.Background()

	t.Run("nil client", func(t *testing.T) {
		r := &RemoteTextToSpeech{}
		_, err := r.Synthesize(ctx, TTSRequest{})
		if !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
	})

	t.Run("happy path", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{synthesize: func() (*ttsv1.SynthesizeResponse, error) {
			return &ttsv1.SynthesizeResponse{Audio: []byte{9, 9}, ContentType: "audio/mpeg"}, nil
		}}
		r := &RemoteTextToSpeech{Client: c}
		got, err := r.Synthesize(ctx, TTSRequest{Input: "hi", ResponseFormat: "mp3"})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if string(got.Audio) != "\x09\x09" || got.ContentType != "audio/mpeg" {
			t.Errorf("result mismatch: %+v", got)
		}
	})

	t.Run("empty response", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{synthesize: func() (*ttsv1.SynthesizeResponse, error) { return nil, nil }}
		r := &RemoteTextToSpeech{Client: c}
		_, err := r.Synthesize(ctx, TTSRequest{})
		if err == nil || err.Error() != "audiotools: empty synthesize response" {
			t.Fatalf("want empty synthesize error, got %v", err)
		}
	})

	t.Run("transport failure", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{synthesize: func() (*ttsv1.SynthesizeResponse, error) { return nil, unavailableErr() }}
		r := &RemoteTextToSpeech{Client: c}
		_, err := r.Synthesize(ctx, TTSRequest{})
		if !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
		if c.Resolved() {
			t.Error("expected re-resolve scheduled")
		}
	})
}

func TestRemoteTTSListVoices(t *testing.T) {
	ctx := context.Background()

	t.Run("nil client", func(t *testing.T) {
		r := &RemoteTextToSpeech{}
		_, err := r.ListVoices(ctx)
		if !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
	})

	t.Run("happy path", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{listVoices: func() (*ttsv1.ListVoicesResponse, error) {
			return &ttsv1.ListVoicesResponse{Voices: []*ttsv1.Voice{
				{Id: "af_heart", Name: "Heart"},
				{Id: "am_adam", Name: "Adam"},
			}}, nil
		}}
		r := &RemoteTextToSpeech{Client: c}
		got, err := r.ListVoices(ctx)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(got) != 2 || got[0].ID != "af_heart" || got[0].Name != "Heart" || got[1].ID != "am_adam" {
			t.Errorf("voices mismatch: %+v", got)
		}
	})

	t.Run("nil response yields nil slice", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{listVoices: func() (*ttsv1.ListVoicesResponse, error) { return nil, nil }}
		r := &RemoteTextToSpeech{Client: c}
		got, err := r.ListVoices(ctx)
		if err != nil || got != nil {
			t.Fatalf("want nil,nil got %v,%v", got, err)
		}
	})

	t.Run("transport failure", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{listVoices: func() (*ttsv1.ListVoicesResponse, error) { return nil, unavailableErr() }}
		r := &RemoteTextToSpeech{Client: c}
		_, err := r.ListVoices(ctx)
		if !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
		if c.Resolved() {
			t.Error("expected re-resolve")
		}
	})
}

func TestRemoteTTSGetCached(t *testing.T) {
	ctx := context.Background()

	t.Run("nil client", func(t *testing.T) {
		r := &RemoteTextToSpeech{}
		_, ok := r.GetCached(ctx, CacheLookup{})
		if ok {
			t.Fatal("want miss on nil client")
		}
	})

	t.Run("hit", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{getCache: func() (*ttsv1.GetCacheResponse, error) {
			return &ttsv1.GetCacheResponse{Hit: true, Audio: []byte{7}, ContentType: "audio/wav"}, nil
		}}
		r := &RemoteTextToSpeech{Client: c}
		got, ok := r.GetCached(ctx, CacheLookup{EventID: "e1"})
		if !ok {
			t.Fatal("want hit")
		}
		if string(got.Audio) != "\x07" || got.ContentType != "audio/wav" {
			t.Errorf("result mismatch: %+v", got)
		}
	})

	t.Run("miss when Hit=false", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{getCache: func() (*ttsv1.GetCacheResponse, error) {
			return &ttsv1.GetCacheResponse{Hit: false}, nil
		}}
		r := &RemoteTextToSpeech{Client: c}
		if _, ok := r.GetCached(ctx, CacheLookup{}); ok {
			t.Fatal("want miss when Hit=false")
		}
	})

	t.Run("miss on error", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{getCache: func() (*ttsv1.GetCacheResponse, error) { return nil, unavailableErr() }}
		r := &RemoteTextToSpeech{Client: c}
		if _, ok := r.GetCached(ctx, CacheLookup{}); ok {
			t.Fatal("want miss on error")
		}
	})
}

// -----------------------------------------------------------------------------
// RemotePlaybackEventRecorder
// -----------------------------------------------------------------------------

func TestRemotePlaybackEventRecorder(t *testing.T) {
	ctx := context.Background()

	t.Run("nil client", func(t *testing.T) {
		r := &RemotePlaybackEventRecorder{}
		if err := r.RecordPlaybackEvent(ctx, PlaybackEvent{}); !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
	})

	t.Run("happy path", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{recordEvent: func() (*ttsv1.RecordPlaybackEventResponse, error) {
			return &ttsv1.RecordPlaybackEventResponse{}, nil
		}}
		r := &RemotePlaybackEventRecorder{Client: c}
		if err := r.RecordPlaybackEvent(ctx, PlaybackEvent{Source: "ui", Stage: "start"}); err != nil {
			t.Fatalf("err: %v", err)
		}
	})

	t.Run("transport failure", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{recordEvent: func() (*ttsv1.RecordPlaybackEventResponse, error) { return nil, unavailableErr() }}
		r := &RemotePlaybackEventRecorder{Client: c}
		if err := r.RecordPlaybackEvent(ctx, PlaybackEvent{}); !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
		if c.Resolved() {
			t.Error("expected re-resolve")
		}
	})
}

// -----------------------------------------------------------------------------
// RemoteSpeechTextProcessor (caching + fallback-to-input)
// -----------------------------------------------------------------------------

func TestRemoteProcessorNormalize(t *testing.T) {
	t.Run("nil client returns input", func(t *testing.T) {
		var r *RemoteSpeechTextProcessor
		if got := r.NormalizeForSpeech("abc"); got != "abc" {
			t.Errorf("nil receiver: got %q", got)
		}
		r2 := &RemoteSpeechTextProcessor{}
		if got := r2.NormalizeForSpeech("abc"); got != "abc" {
			t.Errorf("nil client: got %q", got)
		}
	})

	t.Run("happy path + cache", func(t *testing.T) {
		c := newTestClient(t)
		calls := 0
		c.TTS = &fakeTTS{normalize: func() (*ttsv1.NormalizeForSpeechResponse, error) {
			calls++
			return &ttsv1.NormalizeForSpeechResponse{Text: "normalized"}, nil
		}}
		r := &RemoteSpeechTextProcessor{Client: c}
		if got := r.NormalizeForSpeech("raw"); got != "normalized" {
			t.Errorf("got %q", got)
		}
		// second call must hit the cache, not the fake.
		if got := r.NormalizeForSpeech("raw"); got != "normalized" {
			t.Errorf("cached got %q", got)
		}
		if calls != 1 {
			t.Errorf("expected 1 RPC call (rest cached), got %d", calls)
		}
	})

	t.Run("error falls back to input", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{normalize: func() (*ttsv1.NormalizeForSpeechResponse, error) { return nil, unavailableErr() }}
		r := &RemoteSpeechTextProcessor{Client: c}
		if got := r.NormalizeForSpeech("raw"); got != "raw" {
			t.Errorf("fallback: got %q", got)
		}
	})
}

func TestRemoteProcessorSplit(t *testing.T) {
	t.Run("nil client returns single paragraph", func(t *testing.T) {
		r := &RemoteSpeechTextProcessor{}
		got := r.SplitIntoParagraphs("abc")
		if len(got) != 1 || got[0] != "abc" {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("happy path + cache", func(t *testing.T) {
		c := newTestClient(t)
		calls := 0
		c.TTS = &fakeTTS{split: func() (*ttsv1.SplitParagraphsResponse, error) {
			calls++
			return &ttsv1.SplitParagraphsResponse{Paragraphs: []string{"p1", "p2"}}, nil
		}}
		r := &RemoteSpeechTextProcessor{Client: c}
		got := r.SplitIntoParagraphs("raw")
		if len(got) != 2 || got[0] != "p1" || got[1] != "p2" {
			t.Errorf("got %+v", got)
		}
		_ = r.SplitIntoParagraphs("raw")
		if calls != 1 {
			t.Errorf("expected 1 RPC call (rest cached), got %d", calls)
		}
	})

	t.Run("error falls back to single paragraph", func(t *testing.T) {
		c := newTestClient(t)
		c.TTS = &fakeTTS{split: func() (*ttsv1.SplitParagraphsResponse, error) { return nil, unavailableErr() }}
		r := &RemoteSpeechTextProcessor{Client: c}
		got := r.SplitIntoParagraphs("raw")
		if len(got) != 1 || got[0] != "raw" {
			t.Errorf("got %+v", got)
		}
	})
}

// -----------------------------------------------------------------------------
// RemoteSummarizer
// -----------------------------------------------------------------------------

func TestRemoteSummarizer(t *testing.T) {
	ctx := context.Background()

	t.Run("nil client", func(t *testing.T) {
		r := &RemoteSummarizer{}
		if _, err := r.Summarize(ctx, SummarizeInput{}); !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
	})

	t.Run("happy path maps all fields", func(t *testing.T) {
		c := newTestClient(t)
		c.Summarize = &fakeSummarize{summarize: func() (*summv1.SummarizeResponse, error) {
			return &summv1.SummarizeResponse{
				Text:         "summary",
				PromptTokens: 10,
				OutputTokens: 5,
				ProviderTier: 2, // BYOK
				ProviderId:   "prov",
				ModelId:      "model",
				LatencyMs:    250,
			}, nil
		}}
		r := &RemoteSummarizer{Client: c}
		got, err := r.Summarize(ctx, SummarizeInput{Text: "long", Level: "heavy"})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got.Text != "summary" || got.PromptTokens != 10 || got.OutputTokens != 5 ||
			got.ProviderTier != "byok" || got.ProviderID != "prov" || got.ModelID != "model" {
			t.Errorf("mismatch: %+v", got)
		}
		if got.Latency.Milliseconds() != 250 {
			t.Errorf("latency = %v", got.Latency)
		}
	})

	t.Run("empty response -> ErrUnavailable", func(t *testing.T) {
		c := newTestClient(t)
		c.Summarize = &fakeSummarize{summarize: func() (*summv1.SummarizeResponse, error) { return nil, nil }}
		r := &RemoteSummarizer{Client: c}
		if _, err := r.Summarize(ctx, SummarizeInput{}); !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable on empty, got %v", err)
		}
	})

	t.Run("transport failure", func(t *testing.T) {
		c := newTestClient(t)
		c.Summarize = &fakeSummarize{summarize: func() (*summv1.SummarizeResponse, error) { return nil, unavailableErr() }}
		r := &RemoteSummarizer{Client: c}
		if _, err := r.Summarize(ctx, SummarizeInput{}); !errors.Is(err, audiotools.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
		if c.Resolved() {
			t.Error("expected re-resolve")
		}
	})
}
