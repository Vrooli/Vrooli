package stt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/audioformat"
	sttpipeline "audio-tools/internal/stt/pipeline"
	sttspeaker "audio-tools/internal/stt/speaker"
)

// fakeExtractResource serves /v1/extract returning `cleaned` as the body plus
// the score/matched headers the Go client parses. It records the uploaded bytes
// so a test can assert the window was WAV-wrapped before sending.
func fakeExtractResource(t *testing.T, cleaned []byte, matched bool) (*sttpipeline.SpeakerClient, *[]byte) {
	t.Helper()
	var gotUpload []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/extract" && r.Method == http.MethodPost {
			_ = r.ParseMultipartForm(1 << 20)
			if f, _, err := r.FormFile("audio"); err == nil {
				buf := make([]byte, 0, 256)
				tmp := make([]byte, 256)
				for {
					n, rerr := f.Read(tmp)
					buf = append(buf, tmp[:n]...)
					if rerr != nil {
						break
					}
				}
				_ = f.Close()
				gotUpload = buf
			}
			w.Header().Set("X-Speaker-Score", "0.91")
			if matched {
				w.Header().Set("X-Speaker-Matched", "true")
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(cleaned)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return &sttpipeline.SpeakerClient{BaseURL: srv.URL, Doer: http.DefaultClient}, &gotUpload
}

// TestCurrentSpeakerExtraction_NilUnlessEnabled proves the ingress extractor is
// omitted unless extraction is explicitly enabled with a bound profile.
func TestCurrentSpeakerExtraction_NilUnlessEnabled(t *testing.T) {
	resetSpeakerCfg()
	t.Cleanup(resetSpeakerCfg)

	client, _ := fakeExtractResource(t, []byte("x"), true)

	// Default config (disabled) -> nil.
	require.Nil(t, currentSpeakerExtraction(Deps{SpeakerResource: client}))

	// Enabled + mode set but extraction OFF -> still nil.
	speakerCfgMu.Lock()
	speakerCfg = speakerCfgDoc{Enabled: true, Mode: "filter", ProfileIDs: []string{"p1"}}
	speakerCfgMu.Unlock()
	require.Nil(t, currentSpeakerExtraction(Deps{SpeakerResource: client}))

	// Extraction enabled but no profile -> nil.
	speakerCfgMu.Lock()
	speakerCfg = speakerCfgDoc{Enabled: true, Mode: "filter", ExtractionEnabled: true}
	speakerCfgMu.Unlock()
	require.Nil(t, currentSpeakerExtraction(Deps{SpeakerResource: client}))

	// All conditions met -> built.
	speakerCfgMu.Lock()
	speakerCfg = speakerCfgDoc{Enabled: true, Mode: "filter", ExtractionEnabled: true, ProfileIDs: []string{"p1"}}
	speakerCfgMu.Unlock()
	require.NotNil(t, currentSpeakerExtraction(Deps{SpeakerResource: client}))

	// No resource client -> nil even when fully configured.
	require.Nil(t, currentSpeakerExtraction(Deps{}))
}

// TestSpeakerExtraction_ReturnsCleanedOnMatch proves the adapter WAV-wraps the
// PCM window for the resource and returns the cleaned canonical PCM on a match.
func TestSpeakerExtraction_ReturnsCleanedOnMatch(t *testing.T) {
	cleaned := []byte{9, 9, 9, 9}
	client, upload := fakeExtractResource(t, cleaned, true)
	ext := sttspeaker.NewExtraction(sttpipeline.SpeakerConfig{Enabled: true, ExtractionEnabled: true, ProfileIDs: []string{"p1"}, Threshold: 0.35, Mode: "filter"}, client)
	out, err := ext.Extract(context.Background(), []byte{1, 2, 3, 4})
	require.NoError(t, err)
	require.Equal(t, cleaned, out, "matched profile returns the resource's cleaned audio")

	// The resource received WAV-wrapped audio, not the raw PCM window.
	require.GreaterOrEqual(t, len(*upload), 4)
	require.Equal(t, "RIFF", string((*upload)[:4]), "PCM window is WAV-wrapped before extraction")
	// Sanity: a freestanding WAV-wrap of the same PCM matches what was sent.
	require.Equal(t, audioformat.WAVFromCanonicalPCM([]byte{1, 2, 3, 4}), *upload)
}

// TestSpeakerExtraction_PassthroughWhenNothingToDo proves the adapter degrades
// to passthrough (returns the input) when it has nothing to isolate against.
func TestSpeakerExtraction_PassthroughWhenNothingToDo(t *testing.T) {
	require.Nil(t, sttspeaker.NewExtraction(sttpipeline.SpeakerConfig{}, nil))
}
