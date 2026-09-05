package stt

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/logx"
	"audio-tools/internal/store"
	intstt "audio-tools/internal/stt"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/schedule"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

func newSTTClient(t *testing.T, d Deps) sttconnect.STTAdminServiceClient {
	t.Helper()
	if d.Logger == nil {
		d.Logger = logx.Std{}
	}
	if d.Clock == nil {
		d.Clock = schedule.System()
	}
	mod := Module(d)
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return sttconnect.NewSTTAdminServiceClient(http.DefaultClient, srv.URL)
}

func newSTTRuntimeClient(t *testing.T, d Deps) sttconnect.STTServiceClient {
	t.Helper()
	if d.Logger == nil {
		d.Logger = logx.Std{}
	}
	if d.Clock == nil {
		d.Clock = schedule.System()
	}
	mod := Module(d)
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return sttconnect.NewSTTServiceClient(http.DefaultClient, srv.URL)
}

func newSpeakerStoreT(t *testing.T) *store.SpeakerStore {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(intstt.Schema)))
	return store.NewSpeakerStore(apidb.NewFromPrimary(d))
}

func newWakeWordStoreT(t *testing.T) *store.WakeWordStore {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(intstt.Schema)))
	return store.NewWakeWordStore(apidb.NewFromPrimary(d))
}

func newStreamCfgStoreT(t *testing.T) *store.STTStreamConfigStore {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(intstt.Schema)))
	return store.NewSTTStreamConfigStore(apidb.NewFromPrimary(d))
}

func newSpeakerCfgStoreT(t *testing.T) *store.STTSpeakerConfigStore {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(intstt.Schema)))
	return store.NewSTTSpeakerConfigStore(apidb.NewFromPrimary(d))
}

// Reset the in-process speaker-config cell between tests that mutate it.
func resetSpeakerCfg() {
	speakerCfgMu.Lock()
	speakerCfg = defaultSpeakerCfg()
	speakerCfgMu.Unlock()
}

// ----- mappers / pure helpers --------------------------------------

func TestSpeakerCfgDoc_ToProto_RoundTrip(t *testing.T) {
	d := speakerCfgDoc{
		Enabled: true, ProfileIDs: []string{"a", "b"}, Threshold: 0.8,
		Mode: "filter", RejectBehavior: "drop",
		FallbackWithoutVerification: true, ExtractionEnabled: true,
	}
	p := d.toProto()
	require.True(t, p.GetEnabled())
	require.Equal(t, []string{"a", "b"}, p.GetProfileIds())
	require.Equal(t, 0.8, p.GetThreshold())
	require.Equal(t, sttv1.SpeakerMode_SPEAKER_MODE_FILTER, p.GetMode())
	require.Equal(t, sttv1.RejectBehavior_REJECT_BEHAVIOR_DROP, p.GetRejectBehavior())
	require.True(t, p.GetFallbackWithoutVerification())
	require.True(t, p.GetExtractionEnabled())
}

// TestSpeakerConfig_PersistsAcrossRestart is the B2 contract: speaker
// mode/threshold/profile bindings must survive a process restart. Previously
// they lived only in the in-process cell and were lost on every restart while
// profiles persisted — so "only my voice" silently disabled itself.
func TestSpeakerConfig_PersistsAcrossRestart(t *testing.T) {
	scs := newSpeakerCfgStoreT(t)
	resetSpeakerCfg()

	// Session 1: enable filter mode with a bound profile + custom threshold.
	c1 := newSTTClient(t, Deps{SpeakerConfig: scs})
	_, err := c1.UpdateSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.UpdateSpeakerConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"enabled", "mode", "threshold", "profile_ids"}},
		Config: &sttv1.SpeakerConfig{
			Enabled:    true,
			Mode:       sttv1.SpeakerMode_SPEAKER_MODE_FILTER,
			Threshold:  0.82,
			ProfileIds: []string{"my-voice"},
		},
	}))
	require.NoError(t, err)

	// Simulate a restart: blow away the in-process cell, then construct a fresh
	// handler against the SAME store. NewConnectHandler must rehydrate the cell.
	resetSpeakerCfg()
	c2 := newSTTClient(t, Deps{SpeakerConfig: scs})
	got, err := c2.GetSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.GetSpeakerConfigRequest{}))
	require.NoError(t, err)
	cfg := got.Msg.GetConfig()
	require.True(t, cfg.GetEnabled(), "enabled must survive restart")
	require.Equal(t, sttv1.SpeakerMode_SPEAKER_MODE_FILTER, cfg.GetMode())
	require.Equal(t, 0.82, cfg.GetThreshold())
	require.Equal(t, []string{"my-voice"}, cfg.GetProfileIds())
}

// TestStreamConfig_DenoiseRoundTrip is the Phase C config-plumbing contract:
// denoise_enabled must survive the proto → streamCfgDoc → sqlite → proto path
// and default off when never set.
func TestStreamConfig_DenoiseRoundTrip(t *testing.T) {
	scs := newStreamCfgStoreT(t)
	c := newSTTClient(t, Deps{StreamConfig: scs})

	// Default: off.
	res, err := c.GetStreamConfig(context.Background(), connect.NewRequest(&sttv1.GetStreamConfigRequest{}))
	require.NoError(t, err)
	require.False(t, res.Msg.GetConfig().GetDenoiseEnabled(), "denoise must default off")

	// Enable + round-trip.
	_, err = c.UpdateStreamConfig(context.Background(), connect.NewRequest(&sttv1.UpdateStreamConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"denoise_enabled"}},
		Config:     &sttv1.StreamConfig{DenoiseEnabled: true},
	}))
	require.NoError(t, err)
	got, err := c.GetStreamConfig(context.Background(), connect.NewRequest(&sttv1.GetStreamConfigRequest{}))
	require.NoError(t, err)
	require.True(t, got.Msg.GetConfig().GetDenoiseEnabled(), "denoise_enabled must persist + round-trip")
}

// TestStreamConfig_StallRejectsRoundTrip is the stall-fallback config
// contract: overlap_max_stall_rejects must default to 0, and — because 0
// is a MEANINGFUL "disabled" value, not "unset" — an explicitly-set 0 must
// survive the proto → streamCfgDoc → sqlite → backfill → proto round trip
// rather than backfilling to the default. This is the presence-tracking
// guarantee (a plain int32 with zero-as-default backfill would silently
// re-enable a disabled fallback on the next load).
func TestStreamConfig_StallRejectsRoundTrip(t *testing.T) {
	scs := newStreamCfgStoreT(t)
	c := newSTTClient(t, Deps{StreamConfig: scs})

	// Default: 3 (operator default applied at the config layer).
	res, err := c.GetStreamConfig(context.Background(), connect.NewRequest(&sttv1.GetStreamConfigRequest{}))
	require.NoError(t, err)
	require.Equal(t, int32(0), res.Msg.GetConfig().GetOverlapMaxStallRejects(), "stall-rejects must default to 0")

	// Set an explicit non-default value and round-trip.
	_, err = c.UpdateStreamConfig(context.Background(), connect.NewRequest(&sttv1.UpdateStreamConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"overlap_max_stall_rejects"}},
		Config:     &sttv1.StreamConfig{OverlapMaxStallRejects: 5},
	}))
	require.NoError(t, err)
	got, err := c.GetStreamConfig(context.Background(), connect.NewRequest(&sttv1.GetStreamConfigRequest{}))
	require.NoError(t, err)
	require.Equal(t, int32(5), got.Msg.GetConfig().GetOverlapMaxStallRejects(), "explicit value must persist")

	// Set 0 (disabled) and round-trip — must STAY 0, not backfill to 3.
	_, err = c.UpdateStreamConfig(context.Background(), connect.NewRequest(&sttv1.UpdateStreamConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"overlap_max_stall_rejects"}},
		Config:     &sttv1.StreamConfig{OverlapMaxStallRejects: 0},
	}))
	require.NoError(t, err)
	got, err = c.GetStreamConfig(context.Background(), connect.NewRequest(&sttv1.GetStreamConfigRequest{}))
	require.NoError(t, err)
	require.Equal(t, int32(0), got.Msg.GetConfig().GetOverlapMaxStallRejects(),
		"explicit 0 (disabled) must survive — presence tracking, not zero-as-default backfill")
}

func TestStreamCfgDoc_ToProtoAndDefaults(t *testing.T) {
	d := defaultStreamCfg()
	require.Equal(t, "auto", d.StreamingMode)
	require.Equal(t, "auto", d.StrategyPreference)
	p := d.toProto()
	require.Equal(t, int32(250), p.GetFlushIntervalMs())
	require.Equal(t, sttv1.StreamingMode_STREAMING_MODE_AUTO, p.GetStreamingMode())
	require.Equal(t, int32(0), p.GetOverlapMaxStallRejects(), "stall-rejects default is 0")
	// The server-side VAD silence default drives where Whisper sees segment
	// boundaries; 1200ms is the SSOT for both the mic-button ring countdown
	// (via useHydrateVoiceConfig) and the actual segment cut.
	require.Equal(t, int32(1200), p.GetVadSilenceMs())
}

func TestMinInt(t *testing.T) {
	require.Equal(t, 1, minInt(1, 5))
	require.Equal(t, 2, minInt(7, 2))
	require.Equal(t, 3, minInt(3, 3))
}

func TestValidateStreamingLevers(t *testing.T) {
	require.NoError(t, validateStreamingLevers(streamCfgDoc{}))
	require.NoError(t, validateStreamingLevers(streamCfgDoc{
		StreamingMode: "auto", StrategyPreference: "vad", VadSilenceMs: 700,
		OverlapWindowMs: 2000, OverlapCommitRuns: 2,
	}))
	require.Error(t, validateStreamingLevers(streamCfgDoc{StreamingMode: "bogus"}))
	require.Error(t, validateStreamingLevers(streamCfgDoc{StrategyPreference: "bogus"}))
	require.Error(t, validateStreamingLevers(streamCfgDoc{VadSilenceMs: 100}))
	require.Error(t, validateStreamingLevers(streamCfgDoc{OverlapWindowMs: 6000}))
	require.Error(t, validateStreamingLevers(streamCfgDoc{OverlapCommitRuns: 9}))
	// Stall-rejects: nil (absent) ok; 0 (disabled) ok; in-range ok; >10 error.
	require.NoError(t, validateStreamingLevers(streamCfgDoc{OverlapMaxStallRejects: int32Ptr(0)}))
	require.NoError(t, validateStreamingLevers(streamCfgDoc{OverlapMaxStallRejects: int32Ptr(10)}))
	require.Error(t, validateStreamingLevers(streamCfgDoc{OverlapMaxStallRejects: int32Ptr(11)}))
	require.Error(t, validateStreamingLevers(streamCfgDoc{OverlapMaxStallRejects: int32Ptr(-1)}))
}

// ----- mapChainError ----------------------------------------------

func TestMapChainError_Codes(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want connect.Code
	}{
		{"insufficient_credits", sttchain.ErrInsufficientCredits, connect.CodeResourceExhausted},
		{"unknown_byok", sttchain.ErrUnknownBYOKProvider, connect.CodeInvalidArgument},
		{"missing_byok", sttchain.ErrMissingBYOKProvider, connect.CodeInvalidArgument},
		{"all_failed", sttchain.ErrAllProvidersFailed, connect.CodeUnavailable},
		{"generic", errors.New("other"), connect.CodeInternal},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := mapChainError(tc.in)
			require.Equal(t, tc.want, connect.CodeOf(err))
		})
	}
}

// ----- Transcribe (no chain) -------------------------------------

func TestTranscribe_NoChainMapsToUnavailable(t *testing.T) {
	c := newSTTRuntimeClient(t, Deps{})
	_, err := c.Transcribe(context.Background(), connect.NewRequest(&sttv1.TranscribeRequest{Audio: []byte("X"), Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

// ----- speaker config / profiles ----------------------------------

func TestSpeakerConfig_GetUpdateRoundTrip(t *testing.T) {
	resetSpeakerCfg()
	t.Cleanup(resetSpeakerCfg)
	c := newSTTClient(t, Deps{})

	get, err := c.GetSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.GetSpeakerConfigRequest{}))
	require.NoError(t, err)
	require.False(t, get.Msg.GetConfig().GetEnabled())

	upd, err := c.UpdateSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.UpdateSpeakerConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{
			"enabled", "profile_ids", "threshold", "mode", "reject_behavior",
			"fallback_without_verification", "extraction_enabled",
		}},
		Config: &sttv1.SpeakerConfig{
			Enabled:                     true,
			ProfileIds:                  []string{"sp-1"},
			Threshold:                   0.9,
			Mode:                        sttv1.SpeakerMode_SPEAKER_MODE_FILTER,
			RejectBehavior:              sttv1.RejectBehavior_REJECT_BEHAVIOR_DROP,
			FallbackWithoutVerification: true,
			ExtractionEnabled:           true,
		},
	}))
	require.NoError(t, err)
	require.True(t, upd.Msg.GetConfig().GetEnabled())
	require.Equal(t, []string{"sp-1"}, upd.Msg.GetConfig().GetProfileIds())
	require.Equal(t, sttv1.SpeakerMode_SPEAKER_MODE_FILTER, upd.Msg.GetConfig().GetMode())
}

func TestSpeakerStatus_NoStore(t *testing.T) {
	resetSpeakerCfg()
	t.Cleanup(resetSpeakerCfg)
	c := newSTTClient(t, Deps{})
	res, err := c.GetSpeakerStatus(context.Background(), connect.NewRequest(&sttv1.GetSpeakerStatusRequest{}))
	require.NoError(t, err)
	require.NotNil(t, res.Msg.GetStatus())
	require.False(t, res.Msg.GetStatus().GetProfileExists())
}

func TestSpeakerStatus_WithStore(t *testing.T) {
	resetSpeakerCfg()
	t.Cleanup(resetSpeakerCfg)
	sp := newSpeakerStoreT(t)
	require.NoError(t, sp.Upsert(context.Background(), store.SpeakerProfile{ID: "sp-X", Name: "X", Embedding: []byte{1, 2}}))
	c := newSTTClient(t, Deps{Speaker: sp})
	res, err := c.GetSpeakerStatus(context.Background(), connect.NewRequest(&sttv1.GetSpeakerStatusRequest{}))
	require.NoError(t, err)
	require.True(t, res.Msg.GetStatus().GetProfileExists())
	require.Equal(t, int32(1), res.Msg.GetStatus().GetProfileCount())
}

func TestListSpeakerProfiles_NoStoreReturnsEmpty(t *testing.T) {
	c := newSTTClient(t, Deps{})
	res, err := c.ListSpeakerProfiles(context.Background(), connect.NewRequest(&sttv1.ListSpeakerProfilesRequest{}))
	require.NoError(t, err)
	require.Empty(t, res.Msg.GetProfiles())
}

func TestListSpeakerProfiles_FromStore(t *testing.T) {
	sp := newSpeakerStoreT(t)
	require.NoError(t, sp.Upsert(context.Background(), store.SpeakerProfile{ID: "p-1", Name: "Alice", Embedding: []byte{1, 2, 3}}))
	c := newSTTClient(t, Deps{Speaker: sp})
	res, err := c.ListSpeakerProfiles(context.Background(), connect.NewRequest(&sttv1.ListSpeakerProfilesRequest{}))
	require.NoError(t, err)
	require.Len(t, res.Msg.GetProfiles(), 1)
	require.Equal(t, "p-1", res.Msg.GetProfiles()[0].GetId())
	require.Equal(t, int32(1), res.Msg.GetCount())
}

func TestEnrollSpeakerProfile_NoStoreIsFailedPrecondition(t *testing.T) {
	c := newSTTClient(t, Deps{})
	_, err := c.EnrollSpeakerProfile(context.Background(), connect.NewRequest(&sttv1.EnrollSpeakerProfileRequest{Audio: []byte("X")}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestEnrollSpeakerProfile_HappyPath(t *testing.T) {
	resetSpeakerCfg()
	t.Cleanup(resetSpeakerCfg)
	sp := newSpeakerStoreT(t)
	c := newSTTClient(t, Deps{Speaker: sp, SpeakerResource: fakeSpeakerResource(t)})
	addToActive := true
	enable := true
	res, err := c.EnrollSpeakerProfile(context.Background(), connect.NewRequest(&sttv1.EnrollSpeakerProfileRequest{
		ProfileId:   "sp-Z",
		DisplayName: "Zoe",
		Audio:       []byte("RAW-AUDIO-BYTES"),
		AddToActive: &addToActive,
		Enable:      &enable,
	}))
	require.NoError(t, err)
	require.Equal(t, "sp-Z", res.Msg.GetEnrollment().GetProfileId())
	require.Contains(t, res.Msg.GetConfig().GetProfileIds(), "sp-Z")
	require.True(t, res.Msg.GetConfig().GetEnabled())
	// Enabling from the inert default (mode=off) lifts to advisory so the
	// enrolled voice actually takes effect instead of staying a no-op.
	require.Equal(t, sttv1.SpeakerMode_SPEAKER_MODE_ADVISORY, res.Msg.GetConfig().GetMode())
}

// TestEnrollSpeakerProfile_AutoAdvisoryPersistsAndPreservesExplicitMode covers
// the enroll-side config contract: (1) enabling from the inert mode=off default
// lifts to advisory and PERSISTS (survives a restart, like UpdateSpeakerConfig),
// and (2) an explicit filter/advisory choice is never silently downgraded by a
// later re-enroll.
func TestEnrollSpeakerProfile_AutoAdvisoryPersistsAndPreservesExplicitMode(t *testing.T) {
	scs := newSpeakerCfgStoreT(t)
	resetSpeakerCfg()
	t.Cleanup(resetSpeakerCfg)
	sp := newSpeakerStoreT(t)

	enable := true
	addToActive := true
	c1 := newSTTClient(t, Deps{Speaker: sp, SpeakerResource: fakeSpeakerResource(t), SpeakerConfig: scs})
	res, err := c1.EnrollSpeakerProfile(context.Background(), connect.NewRequest(&sttv1.EnrollSpeakerProfileRequest{
		ProfileId:   "sp-A",
		DisplayName: "Ann",
		Audio:       []byte("RAW-AUDIO-BYTES"),
		AddToActive: &addToActive,
		Enable:      &enable,
	}))
	require.NoError(t, err)
	require.Equal(t, sttv1.SpeakerMode_SPEAKER_MODE_ADVISORY, res.Msg.GetConfig().GetMode())

	// Restart: blow away the cell, rehydrate from the same store. advisory + the
	// binding + enabled must all have been persisted by the enroll handler.
	resetSpeakerCfg()
	c2 := newSTTClient(t, Deps{Speaker: sp, SpeakerResource: fakeSpeakerResource(t), SpeakerConfig: scs})
	got, err := c2.GetSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.GetSpeakerConfigRequest{}))
	require.NoError(t, err)
	require.True(t, got.Msg.GetConfig().GetEnabled(), "enabled must persist across restart")
	require.Equal(t, sttv1.SpeakerMode_SPEAKER_MODE_ADVISORY, got.Msg.GetConfig().GetMode())
	require.Contains(t, got.Msg.GetConfig().GetProfileIds(), "sp-A")

	// Operator deliberately switches to filter; a later re-enroll must NOT
	// downgrade that explicit choice back to advisory.
	_, err = c2.UpdateSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.UpdateSpeakerConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"mode"}},
		Config:     &sttv1.SpeakerConfig{Mode: sttv1.SpeakerMode_SPEAKER_MODE_FILTER},
	}))
	require.NoError(t, err)
	res2, err := c2.EnrollSpeakerProfile(context.Background(), connect.NewRequest(&sttv1.EnrollSpeakerProfileRequest{
		ProfileId:   "sp-B",
		DisplayName: "Bea",
		Audio:       []byte("RAW-AUDIO-BYTES"),
		AddToActive: &addToActive,
		Enable:      &enable,
	}))
	require.NoError(t, err)
	require.Equal(t, sttv1.SpeakerMode_SPEAKER_MODE_FILTER, res2.Msg.GetConfig().GetMode(), "explicit filter must not be downgraded")
}

func TestEnrollSpeakerProfile_AudioRequired(t *testing.T) {
	sp := newSpeakerStoreT(t)
	c := newSTTClient(t, Deps{Speaker: sp})
	_, err := c.EnrollSpeakerProfile(context.Background(), connect.NewRequest(&sttv1.EnrollSpeakerProfileRequest{ProfileId: "x"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestClearSpeakerProfileBinding(t *testing.T) {
	resetSpeakerCfg()
	t.Cleanup(resetSpeakerCfg)
	speakerCfgMu.Lock()
	speakerCfg.ProfileIDs = []string{"a", "b"}
	speakerCfgMu.Unlock()
	c := newSTTClient(t, Deps{})
	res, err := c.ClearSpeakerProfileBinding(context.Background(), connect.NewRequest(&sttv1.ClearSpeakerProfileBindingRequest{}))
	require.NoError(t, err)
	require.Empty(t, res.Msg.GetConfig().GetProfileIds())
}

func TestUnbindSpeakerProfile(t *testing.T) {
	resetSpeakerCfg()
	t.Cleanup(resetSpeakerCfg)
	speakerCfgMu.Lock()
	speakerCfg.ProfileIDs = []string{"a", "b", "c"}
	speakerCfgMu.Unlock()
	c := newSTTClient(t, Deps{})
	res, err := c.UnbindSpeakerProfile(context.Background(), connect.NewRequest(&sttv1.UnbindSpeakerProfileRequest{ProfileId: "b"}))
	require.NoError(t, err)
	require.NotContains(t, res.Msg.GetConfig().GetProfileIds(), "b")
}

func TestDeleteSpeakerProfile_NoStoreIsFailedPrecondition(t *testing.T) {
	c := newSTTClient(t, Deps{})
	_, err := c.DeleteSpeakerProfile(context.Background(), connect.NewRequest(&sttv1.DeleteSpeakerProfileRequest{ProfileId: "x"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestDeleteSpeakerProfile_EmptyIDIsInvalidArgument(t *testing.T) {
	sp := newSpeakerStoreT(t)
	c := newSTTClient(t, Deps{Speaker: sp})
	_, err := c.DeleteSpeakerProfile(context.Background(), connect.NewRequest(&sttv1.DeleteSpeakerProfileRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestDeleteSpeakerProfile_HappyPath(t *testing.T) {
	resetSpeakerCfg()
	t.Cleanup(resetSpeakerCfg)
	sp := newSpeakerStoreT(t)
	require.NoError(t, sp.Upsert(context.Background(), store.SpeakerProfile{ID: "del-me", Name: "x", Embedding: []byte{1}}))
	c := newSTTClient(t, Deps{Speaker: sp})
	_, err := c.DeleteSpeakerProfile(context.Background(), connect.NewRequest(&sttv1.DeleteSpeakerProfileRequest{ProfileId: "del-me"}))
	require.NoError(t, err)
}

// ----- wakeword ---------------------------------------------------

func TestWakeword_GetUpsertDeleteRoundTrip(t *testing.T) {
	ww := newWakeWordStoreT(t)
	c := newSTTClient(t, Deps{Wakeword: ww})

	get, err := c.GetWakeWordConfig(context.Background(), connect.NewRequest(&sttv1.GetWakeWordConfigRequest{}))
	require.NoError(t, err)
	require.False(t, get.Msg.GetConfig().GetConfigured())

	_, err = c.UpdateWakeWordTemplate(context.Background(), connect.NewRequest(&sttv1.UpdateWakeWordTemplateRequest{
		Template: &sttv1.WakeWordTemplate{Label: "hey-vrooli", Threshold: 0.6},
	}))
	require.NoError(t, err)

	get, err = c.GetWakeWordConfig(context.Background(), connect.NewRequest(&sttv1.GetWakeWordConfigRequest{}))
	require.NoError(t, err)
	require.True(t, get.Msg.GetConfig().GetConfigured())

	_, err = c.DeleteWakeWordTemplate(context.Background(), connect.NewRequest(&sttv1.DeleteWakeWordTemplateRequest{}))
	require.NoError(t, err)
}

func TestWakeword_NoStoreBehaviour(t *testing.T) {
	c := newSTTClient(t, Deps{})
	res, err := c.GetWakeWordConfig(context.Background(), connect.NewRequest(&sttv1.GetWakeWordConfigRequest{}))
	require.NoError(t, err)
	require.False(t, res.Msg.GetConfig().GetConfigured())

	_, err = c.UpdateWakeWordTemplate(context.Background(), connect.NewRequest(&sttv1.UpdateWakeWordTemplateRequest{
		Template: &sttv1.WakeWordTemplate{Label: "x"},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))

	_, err = c.DeleteWakeWordTemplate(context.Background(), connect.NewRequest(&sttv1.DeleteWakeWordTemplateRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

// ----- stream_config ---------------------------------------------

func TestStreamConfig_GetDefaultsWithoutStore(t *testing.T) {
	c := newSTTClient(t, Deps{})
	res, err := c.GetStreamConfig(context.Background(), connect.NewRequest(&sttv1.GetStreamConfigRequest{}))
	require.NoError(t, err)
	require.Equal(t, int32(250), res.Msg.GetConfig().GetFlushIntervalMs())
}

func TestResolveStreamPipelineConfigProjectsPersistedLevers(t *testing.T) {
	repo := staticStreamConfig{raw: `{"streaming_mode":"off","strategy_preference":"overlap","engine_id":"kyutai","vad_silence_ms":800,"overlap_window_ms":2400,"overlap_commit_runs":3,"overlap_max_stall_rejects":0,"hallucination_filter_enabled":false,"vad_filter_enabled":false,"no_speech_threshold":0.4,"logprob_threshold":-2.5,"denoise_enabled":true}`}
	cfg := ResolveStreamPipelineConfig(context.Background(), repo)
	require.Equal(t, intstt.StreamingMode("off"), cfg.Mode)
	require.Equal(t, intstt.StrategyPreference("overlap"), cfg.StrategyPreference)
	require.Equal(t, "kyutai", cfg.EngineID)
	require.Equal(t, 800, cfg.VADSilenceMs)
	require.Equal(t, 2400, cfg.OverlapWindowMs)
	require.Equal(t, 3, cfg.OverlapCommitRuns)
	require.Equal(t, 0, cfg.OverlapMaxStallRejects)
	require.False(t, cfg.HallucinationFilterEnabled)
	require.False(t, cfg.VADFilterEnabled)
	require.Equal(t, 0.4, cfg.NoSpeechThreshold)
	require.Equal(t, -2.5, cfg.LogProbThreshold)
	require.True(t, cfg.DenoiseEnabled)
}

func TestStreamConfig_UpdateRoundTrip(t *testing.T) {
	scs := newStreamCfgStoreT(t)
	c := newSTTClient(t, Deps{StreamConfig: scs})
	_, err := c.UpdateStreamConfig(context.Background(), connect.NewRequest(&sttv1.UpdateStreamConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{
			"flush_interval_ms", "min_delta_bytes", "overlap_bytes", "persistent_mode",
			"wake_word_enabled", "wake_word_threshold", "segment_silence_ms",
			"streaming_mode", "strategy_preference", "vad_silence_ms",
			"overlap_window_ms", "overlap_commit_runs",
		}},
		Config: &sttv1.StreamConfig{
			FlushIntervalMs:    300,
			MinDeltaBytes:      2048,
			OverlapBytes:       1024,
			PersistentMode:     true,
			WakeWordEnabled:    true,
			WakeWordThreshold:  0.7,
			SegmentSilenceMs:   900,
			StreamingMode:      sttv1.StreamingMode_STREAMING_MODE_AUTO,
			StrategyPreference: sttv1.StrategyPreference_STRATEGY_PREFERENCE_VAD,
			VadSilenceMs:       700,
			OverlapWindowMs:    2000,
			OverlapCommitRuns:  2,
		},
	}))
	require.NoError(t, err)

	get, err := c.GetStreamConfig(context.Background(), connect.NewRequest(&sttv1.GetStreamConfigRequest{}))
	require.NoError(t, err)
	cfg := get.Msg.GetConfig()
	require.Equal(t, int32(300), cfg.GetFlushIntervalMs())
	// The five advanced fields are the single source of truth for client-side
	// streaming VAD timing — see scenarios/audio-tools/docs/domains/stt/
	// streaming-pipeline.md. Round-tripping them through SQLite must be lossless.
	require.Equal(t, sttv1.StreamingMode_STREAMING_MODE_AUTO, cfg.GetStreamingMode())
	require.Equal(t, sttv1.StrategyPreference_STRATEGY_PREFERENCE_VAD, cfg.GetStrategyPreference())
	require.Equal(t, int32(700), cfg.GetVadSilenceMs())
	require.Equal(t, int32(2000), cfg.GetOverlapWindowMs())
	require.Equal(t, int32(2), cfg.GetOverlapCommitRuns())
}

// Regression: legacy persisted docs lacking newer fields (vad_silence_ms,
// segment_silence_ms, etc.) used to surface zero to the client via
// GetStreamConfig while the server-side resolveStreamPipelineConfig fell
// back to sttpkg.Defaults() — the two paths then disagreed on VAD timing,
// and the mic-button ring would fill to ~58% before the server cut.
// Fix: loadStreamCfg backfills zero fields from defaultStreamCfg().
func TestStreamConfig_LegacyDocBackfillsDefaults(t *testing.T) {
	scs := newStreamCfgStoreT(t)
	// Simulate a doc persisted before vad_silence_ms / segment_silence_ms
	// / overlap_* fields existed: only flush_interval_ms is present.
	require.NoError(t, scs.Set(context.Background(), `{"flush_interval_ms":250}`))

	c := newSTTClient(t, Deps{StreamConfig: scs})
	res, err := c.GetStreamConfig(context.Background(), connect.NewRequest(&sttv1.GetStreamConfigRequest{}))
	require.NoError(t, err)
	cfg := res.Msg.GetConfig()

	def := defaultStreamCfg()
	require.Equal(t, def.VadSilenceMs, cfg.GetVadSilenceMs(),
		"legacy doc must surface default vad_silence_ms so server VAD and client ring agree")
	require.Equal(t, def.SegmentSilenceMs, cfg.GetSegmentSilenceMs())
	require.Equal(t, def.OverlapWindowMs, cfg.GetOverlapWindowMs())
	require.Equal(t, def.OverlapCommitRuns, cfg.GetOverlapCommitRuns())
	require.Equal(t, def.OverlapBytes, cfg.GetOverlapBytes())
	require.Equal(t, def.MinDeltaBytes, cfg.GetMinDeltaBytes())
	require.Equal(t, sttv1.StreamingMode_STREAMING_MODE_AUTO, cfg.GetStreamingMode())
	require.Equal(t, sttv1.StrategyPreference_STRATEGY_PREFERENCE_AUTO, cfg.GetStrategyPreference())
}

func TestStreamConfig_UpdateRejectsUnknownMaskPath(t *testing.T) {
	scs := newStreamCfgStoreT(t)
	c := newSTTClient(t, Deps{StreamConfig: scs})
	_, err := c.UpdateStreamConfig(context.Background(), connect.NewRequest(&sttv1.UpdateStreamConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"not_a_real_field"}},
		Config:     &sttv1.StreamConfig{},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// ----- multipart -------------------------------------------------

func buildSTTMultipart(t *testing.T, audio []byte, fields map[string]string) (string, string) {
	t.Helper()
	var sb strings.Builder
	mw := newMultipart(&sb)
	require.NoError(t, mw.WriteFile("audio", "in.wav", audio))
	for k, v := range fields {
		require.NoError(t, mw.WriteField(k, v))
	}
	require.NoError(t, mw.Close())
	return sb.String(), mw.ContentType()
}

func TestMultipartTranscribe_ChainNotConfigured(t *testing.T) {
	h := MultipartTranscribeHandler(Deps{})
	body, ct := buildSTTMultipart(t, []byte("X"), nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/transcribe", strings.NewReader(body))
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestMultipartTranscribe_MissingFile(t *testing.T) {
	chain := sttchain.NewChain(sttchain.Options{
		EnableVrooli: true,
		Vrooli:       sttchain.NewVrooliProvider(&sttmocks.FakeVrooliClient{Available: true, Result: &sttchain.Result{Text: "hello"}}),
	})
	h := MultipartTranscribeHandler(Deps{Chain: chain})
	var sb strings.Builder
	mw := newMultipart(&sb)
	require.NoError(t, mw.WriteField("format", "wav"))
	require.NoError(t, mw.Close())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/transcribe", strings.NewReader(sb.String()))
	req.Header.Set("Content-Type", mw.ContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMultipartTranscribe_BadMultipart(t *testing.T) {
	chain := sttchain.NewChain(sttchain.Options{})
	h := MultipartTranscribeHandler(Deps{Chain: chain})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/transcribe", strings.NewReader("not-multipart"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=bogus")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMultipartTranscribe_HappyPath(t *testing.T) {
	chain := sttchain.NewChain(sttchain.Options{
		EnableVrooli: true,
		Vrooli: sttchain.NewVrooliProvider(&sttmocks.FakeVrooliClient{
			Available: true,
			Result:    &sttchain.Result{Text: "hello", DetectedLanguage: "en"},
		}),
	})
	h := MultipartTranscribeHandler(Deps{Chain: chain})
	body, ct := buildSTTMultipart(t, []byte("RAW"), map[string]string{"language": "en", "format": "wav"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/transcribe", strings.NewReader(body))
	req.Header.Set("Content-Type", ct)
	// Set LPBS token header so the chain takes the vrooli branch.
	req.Header.Set("X-Audio-LPBS-Token", "tok")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"text":"hello"`)
}

func TestMultipartTranscribe_ChainErrorMapsHTTP(t *testing.T) {
	chain := sttchain.NewChain(sttchain.Options{
		EnableVrooli: true,
		Vrooli:       sttchain.NewVrooliProvider(&sttmocks.FakeVrooliClient{Available: true, Err: sttchain.ErrInsufficientCredits}),
	})
	h := MultipartTranscribeHandler(Deps{Chain: chain})
	body, ct := buildSTTMultipart(t, []byte("RAW"), nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/transcribe", strings.NewReader(body))
	req.Header.Set("Content-Type", ct)
	req.Header.Set("X-Audio-LPBS-Token", "tok")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.Equal(t, http.StatusPaymentRequired, w.Code)
}

func TestModuleSchemaIsEmpty(t *testing.T) {
	require.Equal(t, "", Schema())
}

// --- small multipart writer wrapper kept local to tests ----------

type multipartWriter struct{ mw *multipart.Writer }

func newMultipart(w io.Writer) *multipartWriter { return &multipartWriter{mw: multipart.NewWriter(w)} }

func (m *multipartWriter) WriteFile(field, filename string, body []byte) error {
	fw, err := m.mw.CreateFormFile(field, filename)
	if err != nil {
		return err
	}
	_, err = fw.Write(body)
	return err
}

func (m *multipartWriter) WriteField(k, v string) error { return m.mw.WriteField(k, v) }
func (m *multipartWriter) Close() error                 { return m.mw.Close() }
func (m *multipartWriter) ContentType() string          { return m.mw.FormDataContentType() }
