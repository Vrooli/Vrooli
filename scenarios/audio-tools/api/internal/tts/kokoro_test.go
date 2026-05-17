package tts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKokoroVoiceLister_ListAsArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"af_bella","name":"Bella"}]`))
	}))
	defer srv.Close()
	k := &KokoroVoiceLister{BaseURL: srv.URL}
	voices, err := k.ListVoices(context.Background())
	require.NoError(t, err)
	require.Len(t, voices, 1)
	require.Equal(t, "af_bella", voices[0].ID)
}

func TestKokoroVoiceLister_ListAsStrings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`["af_bella","af_sarah"]`))
	}))
	defer srv.Close()
	k := &KokoroVoiceLister{BaseURL: srv.URL}
	voices, err := k.ListVoices(context.Background())
	require.NoError(t, err)
	require.Len(t, voices, 2)
}

func TestKokoroVoiceLister_ListAsMap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"af_bella":{}}`))
	}))
	defer srv.Close()
	k := &KokoroVoiceLister{BaseURL: srv.URL}
	voices, err := k.ListVoices(context.Background())
	require.NoError(t, err)
	require.Len(t, voices, 1)
}

func TestKokoroVoiceLister_Unrecognized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`123`))
	}))
	defer srv.Close()
	k := &KokoroVoiceLister{BaseURL: srv.URL}
	_, err := k.ListVoices(context.Background())
	require.Error(t, err)
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
	k := &KokoroSynthesizer{BaseURL: srv.URL}
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
	k := &KokoroSynthesizer{BaseURL: srv.URL}
	_, _, err := k.Synthesize(context.Background(), SynthesizeRequest{Input: "hi", Voice: "v", ResponseFormat: "wav"})
	require.Error(t, err)
}
