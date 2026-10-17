package stt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

func hallucinatingChain(seen *sttchain.Request, text string) *sttchain.Chain {
	return sttchain.NewChain(sttchain.Options{
		EnableVrooli: true,
		Vrooli: sttchain.NewVrooliProvider(&sttmocks.FakeVrooliClient{
			Available: true,
			TranscribeFn: func(_ context.Context, _, _ string, req sttchain.Request) (*sttchain.Result, error) {
				*seen = req
				return &sttchain.Result{
					Text:             text,
					DetectedLanguage: "en",
					Tier:             sttchain.TierVrooli,
					ProviderID:       "lpbs",
					ModelID:          "fake",
					Latency:          time.Millisecond,
					Confidence:       &sttchain.Confidence{NoSpeechProb: 0.99, AvgLogProb: -2.5},
				}, nil
			},
		}),
	})
}

func TestConnectTranscribe_FiltersKnownHallucinationAndSetsVADFilter(t *testing.T) {
	var seen sttchain.Request
	chain := hallucinatingChain(&seen, "Thanks for watching!")
	c := newSTTRuntimeClient(t, Deps{Chain: chain})
	req := connect.NewRequest(&sttv1.TranscribeRequest{Audio: []byte("x")})
	req.Header().Set("X-Audio-LPBS-Token", "tok")
	resp, err := c.Transcribe(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "", resp.Msg.GetText())
	require.True(t, resp.Msg.GetFiltered())
	require.NotEmpty(t, resp.Msg.GetFilterReason())
	require.True(t, seen.VADFilter)
}

func TestMultipartTranscribe_FiltersKnownHallucination(t *testing.T) {
	var seen sttchain.Request
	chain := hallucinatingChain(&seen, "Thanks for watching!")
	h := MultipartTranscribeHandler(Deps{Chain: chain})
	body, ct := buildSTTMultipart(t, []byte("RAW"), map[string]string{"language": "en", "format": "wav"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/transcribe", strings.NewReader(body))
	req.Header.Set("Content-Type", ct)
	req.Header.Set("X-Audio-LPBS-Token", "tok")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var out sttv1.TranscribeResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.Equal(t, "", out.GetText())
	require.True(t, out.GetFiltered())
	require.True(t, seen.VADFilter)
}

func TestConnectTranscribe_PreservesLegitimateSpeech(t *testing.T) {
	chain := sttchain.NewChain(sttchain.Options{
		EnableVrooli: true,
		Vrooli: sttchain.NewVrooliProvider(&sttmocks.FakeVrooliClient{
			Available: true,
			Result: &sttchain.Result{
				Text: "hello world", Tier: sttchain.TierVrooli, ProviderID: "lpbs", Latency: time.Millisecond,
			},
		}),
	})
	c := newSTTRuntimeClient(t, Deps{Chain: chain})
	req := connect.NewRequest(&sttv1.TranscribeRequest{Audio: []byte("x")})
	req.Header().Set("X-Audio-LPBS-Token", "tok")
	resp, err := c.Transcribe(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "hello world", resp.Msg.GetText())
	require.False(t, resp.Msg.GetFiltered())
}
