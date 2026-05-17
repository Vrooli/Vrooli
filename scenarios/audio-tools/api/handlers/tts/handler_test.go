package tts_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	ttsH "audio-tools/handlers/tts"
	"audio-tools/internal/ai/ttschain"
	ttsmocks "audio-tools/internal/ai/ttschain/mocks"
	"audio-tools/internal/byok/envelope"
	"audio-tools/internal/clock"
	"audio-tools/internal/logx"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
)

func newServer(t *testing.T, deps ttsH.Deps) ttsconnect.TTSServiceClient {
	t.Helper()
	if deps.Logger == nil {
		deps.Logger = logx.Std{}
	}
	if deps.Clock == nil {
		deps.Clock = clock.System{}
	}
	mod := ttsH.Module(deps)
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return ttsconnect.NewTTSServiceClient(http.DefaultClient, srv.URL)
}

func TestTTS_Synthesize_HappyPathViaVrooli(t *testing.T) {
	chain := ttschain.NewChain(ttschain.Options{
		EnableVrooli: true,
		Vrooli: ttschain.NewVrooliProvider(&ttsmocks.FakeVrooliClient{
			Available: true,
			Result:    &ttschain.Result{Audio: []byte("PCM"), ContentType: "audio/mpeg"},
		}),
	})
	c := newServer(t, ttsH.Deps{Chain: chain})
	req := connect.NewRequest(&ttsv1.SynthesizeRequest{Text: "hello", Voice: "voice.feminine.warm"})
	req.Header().Set(envelope.HeaderLPBSToken, "tok")
	res, err := c.Synthesize(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, []byte("PCM"), res.Msg.GetAudio())
	require.Equal(t, commonv1.ProviderTier_PROVIDER_TIER_VROOLI, res.Msg.GetProviderTier())
}

func TestTTS_Synthesize_NoChainReturnsUnavailable(t *testing.T) {
	c := newServer(t, ttsH.Deps{})
	_, err := c.Synthesize(context.Background(), connect.NewRequest(&ttsv1.SynthesizeRequest{Text: "x"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestTTS_Synthesize_InsufficientCreditsMapsToResourceExhausted(t *testing.T) {
	chain := ttschain.NewChain(ttschain.Options{
		EnableVrooli: true,
		Vrooli: ttschain.NewVrooliProvider(&ttsmocks.FakeVrooliClient{
			Available: true,
			Err:       ttschain.ErrInsufficientCredits,
		}),
	})
	c := newServer(t, ttsH.Deps{Chain: chain})
	req := connect.NewRequest(&ttsv1.SynthesizeRequest{Text: "x"})
	req.Header().Set(envelope.HeaderLPBSToken, "tok")
	_, err := c.Synthesize(context.Background(), req)
	require.Error(t, err)
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
}

func TestTTS_ListVoices_ReturnsCanonicalSet(t *testing.T) {
	c := newServer(t, ttsH.Deps{})
	res, err := c.ListVoices(context.Background(), connect.NewRequest(&ttsv1.ListVoicesRequest{}))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(res.Msg.GetVoices()), 5)
	require.Equal(t, "voice.feminine.warm", res.Msg.GetVoices()[0].GetId())
}

func TestTTS_NormalizeForSpeech_ReturnsString(t *testing.T) {
	c := newServer(t, ttsH.Deps{})
	res, err := c.NormalizeForSpeech(context.Background(), connect.NewRequest(&ttsv1.NormalizeForSpeechRequest{Text: "Dr. Smith"}))
	require.NoError(t, err)
	require.NotEmpty(t, res.Msg.GetText())
}

func TestTTS_SplitParagraphs_NonEmptyForMultilineInput(t *testing.T) {
	c := newServer(t, ttsH.Deps{})
	res, err := c.SplitParagraphs(context.Background(), connect.NewRequest(&ttsv1.SplitParagraphsRequest{Text: "First para.\n\nSecond para."}))
	require.NoError(t, err)
	require.NotEmpty(t, res.Msg.GetParagraphs())
}

func TestTTS_GetCache_MissWithoutCache(t *testing.T) {
	c := newServer(t, ttsH.Deps{})
	res, err := c.GetCache(context.Background(), connect.NewRequest(&ttsv1.GetCacheRequest{EventId: "ev-1", Voice: "voice.feminine.warm"}))
	require.NoError(t, err)
	require.False(t, res.Msg.GetHit())
}

func TestTTS_GetConfig_DefaultsWhenNoStore(t *testing.T) {
	c := newServer(t, ttsH.Deps{})
	res, err := c.GetConfig(context.Background(), connect.NewRequest(&ttsv1.GetConfigRequest{}))
	require.NoError(t, err)
	require.NotNil(t, res.Msg.GetConfig())
}

func TestTTS_GetStatus_UnavailableCapabilityWithoutChain(t *testing.T) {
	c := newServer(t, ttsH.Deps{})
	res, err := c.GetStatus(context.Background(), connect.NewRequest(&ttsv1.GetStatusRequest{}))
	require.NoError(t, err)
	require.Equal(t, "unavailable", res.Msg.GetStatus().GetCapability())
}

func TestTTS_RecordPlaybackEvent_NoopWithoutPlaybackStore(t *testing.T) {
	c := newServer(t, ttsH.Deps{})
	res, err := c.RecordPlaybackEvent(context.Background(), connect.NewRequest(&ttsv1.RecordPlaybackEventRequest{
		Event: &ttsv1.PlaybackEvent{EventId: "ev-1", Stage: "started", Backend: "kokoro"},
	}))
	require.NoError(t, err)
	require.Equal(t, "noop", res.Msg.GetStatus())
}
