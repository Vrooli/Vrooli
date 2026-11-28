package tts

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func TestKokoroVoiceLister_ListAsArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"af_bella","name":"Bella"}]`))
	}))
	defer srv.Close()
	k := &KokoroVoiceLister{BaseURL: srv.URL, Doer: srv.Client()}
	voices, err := k.ListVoices(context.Background())
	require.NoError(t, err)
	require.Len(t, voices, 1)
	require.Equal(t, "af_bella", voices[0].ID)
}

func TestKaleVoiceLister_UnwrapsModelEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"kale-tts-1-nano","voices":[{"id":"s057b","name":"Kenzie","accent":"en-us"}]}`))
	}))
	defer srv.Close()

	voices, err := (&KokoroVoiceLister{BaseURL: srv.URL, Doer: srv.Client()}).ListVoices(context.Background())
	require.NoError(t, err)
	require.Equal(t, []Voice{{ID: "s057b", Name: "Kenzie"}}, voices)
}

func TestKokoroVoiceLister_ListAsStrings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`["af_bella","af_sarah"]`))
	}))
	defer srv.Close()
	k := &KokoroVoiceLister{BaseURL: srv.URL, Doer: srv.Client()}
	voices, err := k.ListVoices(context.Background())
	require.NoError(t, err)
	require.Len(t, voices, 2)
}

func TestVoiceListerRejectsUnavailableEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "model not loaded", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	_, err := (&KokoroVoiceLister{BaseURL: srv.URL, Doer: srv.Client()}).ListVoices(context.Background())
	require.ErrorContains(t, err, "status 503")
}

func TestKokoroVoiceLister_ListAsMap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"af_bella":{}}`))
	}))
	defer srv.Close()
	k := &KokoroVoiceLister{BaseURL: srv.URL, Doer: srv.Client()}
	voices, err := k.ListVoices(context.Background())
	require.NoError(t, err)
	require.Len(t, voices, 1)
}

func TestKokoroVoiceLister_Unrecognized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`123`))
	}))
	defer srv.Close()
	k := &KokoroVoiceLister{BaseURL: srv.URL, Doer: srv.Client()}
	_, err := k.ListVoices(context.Background())
	require.Error(t, err)
}

func TestKokoroVoiceLister_RejectsOversizedCatalog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]` + strings.Repeat(" ", maxVoiceCatalogResponseBytes)))
	}))
	defer srv.Close()

	k := &KokoroVoiceLister{BaseURL: srv.URL, Doer: srv.Client()}
	_, err := k.ListVoices(context.Background())
	require.ErrorIs(t, err, errResponseBodyTooLarge)
}

func TestResponseFormatContentType_All(t *testing.T) {
	for _, f := range []string{"wav", "opus", "mp3", "flac", "unknown"} {
		require.NotEmpty(t, responseFormatContentType(f))
	}
}

func TestKokoroSynthesizer_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write([]byte("audio"))
	}))
	defer srv.Close()
	k := &KokoroSynthesizer{BaseURL: srv.URL, Doer: srv.Client()}
	body, ct, err := k.Synthesize(context.Background(), SynthesizeRequest{Input: "hi", Voice: "af_bella", ResponseFormat: "wav"})
	require.NoError(t, err)
	require.Equal(t, "audio/wav", ct)
	defer body.Close()
}

func TestKokoroSynthesizer_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer srv.Close()
	k := &KokoroSynthesizer{BaseURL: srv.URL, Doer: srv.Client()}
	_, _, err := k.Synthesize(context.Background(), SynthesizeRequest{Input: "hi", Voice: "v", ResponseFormat: "wav"})
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrUnavailable, "internal synthesis errors must not trigger a backend fallback")
}

func TestSpeechSynthesizerMarksOnlyUnavailableStatusesForFallback(t *testing.T) {
	for _, tc := range []struct {
		status          int
		wantUnavailable bool
	}{{http.StatusServiceUnavailable, true}, {http.StatusBadGateway, true}, {http.StatusGatewayTimeout, true}, {http.StatusInternalServerError, false}, {http.StatusBadRequest, false}} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "failed", tc.status) }))
			defer srv.Close()
			synth := &SpeechSynthesizer{BaseURL: srv.URL, Model: "kale-tts-1-nano", Doer: srv.Client()}
			_, _, err := synth.Synthesize(context.Background(), SynthesizeRequest{Input: "hello", Voice: "kenzie", ResponseFormat: "wav"})
			require.Error(t, err)
			require.Equal(t, tc.wantUnavailable, errors.Is(err, ErrUnavailable))
			if tc.status == http.StatusBadRequest {
				require.ErrorIs(t, err, ErrInvalidArgument)
				require.Contains(t, err.Error(), "speech backend kale-tts-1-nano")
				require.NotContains(t, err.Error(), "kokoro returned")
			}
		})
	}
}

func TestSpeechSynthesizerPreservesRequestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	synth := &SpeechSynthesizer{BaseURL: "http://sherpa.test", Model: "kokoro", Doer: doerFunc(func(req *http.Request) (*http.Response, error) {
		close(started)
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	result := make(chan error, 1)
	go func() {
		_, _, err := synth.Synthesize(ctx, SynthesizeRequest{Input: "hello", Voice: "af_bella", ResponseFormat: "wav"})
		result <- err
	}()
	<-started
	cancel()
	err := <-result
	require.ErrorIs(t, err, context.Canceled)
	require.NotErrorIs(t, err, ErrUnavailable, "caller cancellation must not be treated as backend failover")
}
