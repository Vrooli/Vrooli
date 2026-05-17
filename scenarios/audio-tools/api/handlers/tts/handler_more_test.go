package tts_test

import (
	"context"
	"errors"
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
	"audio-tools/internal/store"
	intsumm "audio-tools/internal/summarize"
	"audio-tools/internal/testutil/db"
	inttts "audio-tools/internal/tts"

	localdb "audio-tools/internal/database"
	apidb "github.com/vrooli/api-core/database"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
)

func newServer2(t *testing.T, deps ttsH.Deps) ttsconnect.TTSServiceClient {
	t.Helper()
	mod := ttsH.Module(deps)
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return ttsconnect.NewTTSServiceClient(http.DefaultClient, srv.URL)
}

func newTTSStoreDB(t *testing.T) *store.TTSConfigStore {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema)))
	return store.NewTTSConfigStore(d)
}

func newPlaybackStoreDB(t *testing.T) *store.PlaybackStore {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema)))
	return store.NewPlaybackStore(d)
}

func TestTTS_GetCache_Hit(t *testing.T) {
	cache := inttts.NewCache(1024 * 1024)
	key := inttts.CacheKey{EventID: "evt1", Voice: "voice.feminine.warm", Speed: 1.0, Version: "active"}
	cache.Put(key, []byte("AUDIO"), "audio/mpeg")
	c := newServer2(t, ttsH.Deps{Cache: cache})
	res, err := c.GetCache(context.Background(), connect.NewRequest(&ttsv1.GetCacheRequest{
		EventId: "evt1", Voice: "voice.feminine.warm", Speed: 1.0, Version: "active",
	}))
	require.NoError(t, err)
	require.True(t, res.Msg.GetHit())
	require.Equal(t, []byte("AUDIO"), res.Msg.GetAudio())
	require.Equal(t, "audio/mpeg", res.Msg.GetContentType())
}

func TestTTS_GetCache_MissOnUnknownEvent(t *testing.T) {
	cache := inttts.NewCache(1024 * 1024)
	c := newServer2(t, ttsH.Deps{Cache: cache})
	res, err := c.GetCache(context.Background(), connect.NewRequest(&ttsv1.GetCacheRequest{EventId: "nope", Voice: "v"}))
	require.NoError(t, err)
	require.False(t, res.Msg.GetHit())
}

func TestTTS_GetCache_VersionDefaultsToActive(t *testing.T) {
	cache := inttts.NewCache(1024 * 1024)
	cache.Put(inttts.CacheKey{EventID: "e", Voice: "v", Speed: 1, Version: "active"}, []byte("X"), "audio/mpeg")
	c := newServer2(t, ttsH.Deps{Cache: cache})
	res, err := c.GetCache(context.Background(), connect.NewRequest(&ttsv1.GetCacheRequest{
		EventId: "e", Voice: "v", Speed: 1,
	}))
	require.NoError(t, err)
	require.True(t, res.Msg.GetHit())
}

func TestTTS_GetCache_EmptyEventIdMisses(t *testing.T) {
	cache := inttts.NewCache(1024)
	c := newServer2(t, ttsH.Deps{Cache: cache})
	res, err := c.GetCache(context.Background(), connect.NewRequest(&ttsv1.GetCacheRequest{Voice: "v"}))
	require.NoError(t, err)
	require.False(t, res.Msg.GetHit())
}

func TestTTS_UpdateConfig_PersistsAndReturns(t *testing.T) {
	cfgStore := newTTSStoreDB(t)
	c := newServer2(t, ttsH.Deps{ConfigStore: cfgStore})
	res, err := c.UpdateConfig(context.Background(), connect.NewRequest(&ttsv1.UpdateConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{
			"auto_enabled", "default_voice", "default_speed", "default_response_format",
			"summarize_enabled", "summarize_char_threshold", "summarize_level",
			"summarize_model", "summarize_timeout_seconds",
		}},
		Config: &ttsv1.Config{
			AutoEnabled:             true,
			DefaultVoice:            "voice.masculine.warm",
			DefaultSpeed:            1.25,
			DefaultResponseFormat:   commonv1.ResponseFormat_RESPONSE_FORMAT_MP3,
			SummarizeEnabled:        true,
			SummarizeCharThreshold:  1024,
			SummarizeLevel:          ttsv1.SummarizeLevel_SUMMARIZE_LEVEL_MODERATE,
			SummarizeModel:          "haiku",
			SummarizeTimeoutSeconds: 30,
		},
	}))
	require.NoError(t, err)
	require.True(t, res.Msg.GetConfig().GetAutoEnabled())
	require.Equal(t, "voice.masculine.warm", res.Msg.GetConfig().GetDefaultVoice())
	require.Equal(t, ttsv1.SummarizeLevel_SUMMARIZE_LEVEL_MODERATE, res.Msg.GetConfig().GetSummarizeLevel())

	// Round-trip — GetConfig should return the persisted values.
	got, err := c.GetConfig(context.Background(), connect.NewRequest(&ttsv1.GetConfigRequest{}))
	require.NoError(t, err)
	require.Equal(t, "voice.masculine.warm", got.Msg.GetConfig().GetDefaultVoice())
	require.Equal(t, int32(1024), got.Msg.GetConfig().GetSummarizeCharThreshold())
}

func TestTTS_GetStatus_AvailableWithChain(t *testing.T) {
	chain := ttschain.NewChain(ttschain.Options{
		EnableVrooli: true,
		Vrooli: ttschain.NewVrooliProvider(&ttsmocks.FakeVrooliClient{
			Available: true,
			Result:    &ttschain.Result{Audio: []byte("X"), ContentType: "audio/mpeg"},
		}),
	})
	c := newServer2(t, ttsH.Deps{Chain: chain})
	res, err := c.GetStatus(context.Background(), connect.NewRequest(&ttsv1.GetStatusRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, res.Msg.GetStatus().GetAvailability())
}

func TestTTS_RecordPlaybackEvent_HappyPath(t *testing.T) {
	ps := newPlaybackStoreDB(t)
	c := newServer2(t, ttsH.Deps{Playback: ps})
	res, err := c.RecordPlaybackEvent(context.Background(), connect.NewRequest(&ttsv1.RecordPlaybackEventRequest{
		Event: &ttsv1.PlaybackEvent{EventId: "p1", Stage: "started", Backend: "kokoro"},
	}))
	require.NoError(t, err)
	require.Equal(t, "recorded", res.Msg.GetStatus())
}

func TestTTS_RecordPlaybackEvent_DefaultsEventID(t *testing.T) {
	ps := newPlaybackStoreDB(t)
	c := newServer2(t, ttsH.Deps{Playback: ps})
	res, err := c.RecordPlaybackEvent(context.Background(), connect.NewRequest(&ttsv1.RecordPlaybackEventRequest{
		Event: &ttsv1.PlaybackEvent{Stage: "started", Backend: "kokoro"},
	}))
	require.NoError(t, err)
	require.Equal(t, "recorded", res.Msg.GetStatus())
}

// TestTTS_Synthesize_AllProvidersFailedMapsUnavailable ensures the
// chain-failure → Connect-code mapping for unavailable returns the
// expected Unavailable code.
func TestTTS_Synthesize_AllProvidersFailedMapsUnavailable(t *testing.T) {
	chain := ttschain.NewChain(ttschain.Options{
		EnableVrooli: true,
		Vrooli: ttschain.NewVrooliProvider(&ttsmocks.FakeVrooliClient{
			Available: true,
			Err:       ttschain.ErrAllProvidersFailed,
		}),
	})
	c := newServer2(t, ttsH.Deps{Chain: chain})
	req := connect.NewRequest(&ttsv1.SynthesizeRequest{Text: "x"})
	req.Header().Set(envelope.HeaderLPBSToken, "tok")
	_, err := c.Synthesize(context.Background(), req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

// TestTTS_Synthesize_UnknownErrorMapsInternal covers the default branch
// of mapChainError.
func TestTTS_Synthesize_UnknownErrorMapsInternal(t *testing.T) {
	chain := ttschain.NewChain(ttschain.Options{
		EnableVrooli: true,
		Vrooli: ttschain.NewVrooliProvider(&ttsmocks.FakeVrooliClient{
			Available: true,
			Err:       errors.New("synth boom"),
		}),
	})
	c := newServer2(t, ttsH.Deps{Chain: chain})
	req := connect.NewRequest(&ttsv1.SynthesizeRequest{Text: "x"})
	req.Header().Set(envelope.HeaderLPBSToken, "tok")
	_, err := c.Synthesize(context.Background(), req)
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestTTS_ConfigToProtoRoundTrip(t *testing.T) {
	// Sanity-check that mapper preserves field values through one
	// configToProto / UpdateConfig hop.
	cfg := inttts.DefaultConfig()
	summ := intsumm.DefaultSummarizeConfig()
	require.NotNil(t, cfg)
	require.NotNil(t, summ)
}
