package stt

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

func TestTranscribeExplicitBatchEngineReachesConfiguredProvider(t *testing.T) {
	var transcribed bool
	var credentialLookup bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/v1/models":
			_, _ = io.WriteString(w, `{"data":[{"id":"Reson8/Rumble-2-Flash"}]}`)
		case "/v1/chat/completions":
			transcribed = true
			_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"language English<asr_text>Product route reached Rumble."}}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	provider := sttchain.NewBatchProvider("rumble-stt", sttchain.NewRumbleVLLMBackend(srv.URL, srv.Client()), nil)
	chain := sttchain.NewChain(sttchain.Options{
		Local:       sttchain.NewLocalProvider(nil),
		BatchModels: map[string]sttchain.Provider{"rumble-2-flash": provider},
		EnableLocal: true,
	})
	client := newSTTRuntimeClient(t, Deps{
		Chain: chain,
		DefaultCredential: func(context.Context, string) (string, string, bool) {
			credentialLookup = true
			return "openai", "fixture-key", true
		},
	})
	req := connect.NewRequest(&sttv1.TranscribeRequest{
		Audio:                   []byte{1, 0, 2, 0},
		Format:                  commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE,
		Language:                "en",
		SkipSpeakerVerification: true,
	})
	req.Header().Set("X-Audio-Model-ID", "rumble-2-flash")
	res, err := client.Transcribe(context.Background(), req)
	require.NoError(t, err)
	require.True(t, transcribed)
	require.False(t, credentialLookup, "explicit model routing must not resolve unrelated BYOK credentials")
	require.Equal(t, "Product route reached Rumble.", res.Msg.GetText())
	require.Equal(t, "rumble-stt", res.Msg.GetProviderId())
	require.Equal(t, "Reson8/Rumble-2-Flash", res.Msg.GetModelId())
}
