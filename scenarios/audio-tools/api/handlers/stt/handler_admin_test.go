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
	localdb "audio-tools/internal/database"
	"audio-tools/internal/store"
	"audio-tools/internal/testutil/db"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

// stubVrooliSTT lets the chain take the vrooli branch deterministically.
type stubVrooliSTT struct {
	available bool
	res       *sttchain.Result
	err       error
}

func (s *stubVrooliSTT) Transcribe(_ context.Context, _, _ string, _ sttchain.Request) (*sttchain.Result, error) {
	return s.res, s.err
}
func (s *stubVrooliSTT) IsAvailable(context.Context) bool { return s.available }
func (s *stubVrooliSTT) Model() string                    { return "stub-stt" }

func newSTTClient(t *testing.T, d Deps) sttconnect.STTServiceClient {
	t.Helper()
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
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema)))
	return store.NewSpeakerStore(d)
}

func newWakeWordStoreT(t *testing.T) *store.WakeWordStore {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema)))
	return store.NewWakeWordStore(d)
}

func newStreamCfgStoreT(t *testing.T) *store.STTStreamConfigStore {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema)))
	return store.NewSTTStreamConfigStore(d)
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
		Mode: "verify", RejectBehavior: "warn",
		FallbackWithoutVerification: true, ExtractionEnabled: true,
	}
	p := d.toProto()
	require.True(t, p.GetEnabled())
	require.Equal(t, []string{"a", "b"}, p.GetProfileIds())
	require.Equal(t, 0.8, p.GetThreshold())
	require.Equal(t, "verify", p.GetMode())
	require.Equal(t, "warn", p.GetRejectBehavior())
	require.True(t, p.GetFallbackWithoutVerification())
	require.True(t, p.GetExtractionEnabled())
}

func TestStreamCfgDoc_ToProtoAndDefaults(t *testing.T) {
	d := defaultStreamCfg()
	require.Equal(t, "auto", d.StreamingMode)
	require.Equal(t, "auto", d.StrategyPreference)
	p := d.toProto()
	require.Equal(t, int32(250), p.GetFlushIntervalMs())
	require.Equal(t, "auto", p.GetStreamingMode())
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
	c := newSTTClient(t, Deps{})
	_, err := c.Transcribe(context.Background(), connect.NewRequest(&sttv1.TranscribeRequest{Audio: []byte("X")}))
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
		HasEnabled: true, Enabled: true,
		HasProfileIds: true, ProfileIds: []string{"sp-1"},
		HasThreshold: true, Threshold: 0.9,
		HasMode: true, Mode: "verify",
		HasRejectBehavior: true, RejectBehavior: "warn",
		HasFallbackWithoutVerification: true, FallbackWithoutVerification: true,
		HasExtractionEnabled: true, ExtractionEnabled: true,
	}))
	require.NoError(t, err)
	require.True(t, upd.Msg.GetConfig().GetEnabled())
	require.Equal(t, []string{"sp-1"}, upd.Msg.GetConfig().GetProfileIds())
	require.Equal(t, "verify", upd.Msg.GetConfig().GetMode())
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
	c := newSTTClient(t, Deps{Speaker: sp})
	res, err := c.EnrollSpeakerProfile(context.Background(), connect.NewRequest(&sttv1.EnrollSpeakerProfileRequest{
		ProfileId:      "sp-Z",
		DisplayName:    "Zoe",
		Audio:          []byte("RAW-AUDIO-BYTES"),
		HasAddToActive: true, AddToActive: true,
		HasEnable: true, Enable: true,
	}))
	require.NoError(t, err)
	require.Equal(t, "sp-Z", res.Msg.GetEnrollment().GetProfileId())
	require.Contains(t, res.Msg.GetConfig().GetProfileIds(), "sp-Z")
	require.True(t, res.Msg.GetConfig().GetEnabled())
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

func TestRemoveSpeakerProfile(t *testing.T) {
	resetSpeakerCfg()
	t.Cleanup(resetSpeakerCfg)
	speakerCfgMu.Lock()
	speakerCfg.ProfileIDs = []string{"a", "b", "c"}
	speakerCfgMu.Unlock()
	c := newSTTClient(t, Deps{})
	res, err := c.RemoveSpeakerProfile(context.Background(), connect.NewRequest(&sttv1.RemoveSpeakerProfileRequest{ProfileId: "b"}))
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

	_, err = c.UpdateWakeWordTemplate(context.Background(), connect.NewRequest(&sttv1.UpdateWakeWordTemplateRequest{TemplateJson: "hey-vrooli"}))
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

	_, err = c.UpdateWakeWordTemplate(context.Background(), connect.NewRequest(&sttv1.UpdateWakeWordTemplateRequest{TemplateJson: "x"}))
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

func TestStreamConfig_UpdateRoundTrip(t *testing.T) {
	scs := newStreamCfgStoreT(t)
	c := newSTTClient(t, Deps{StreamConfig: scs})
	_, err := c.UpdateStreamConfig(context.Background(), connect.NewRequest(&sttv1.UpdateStreamConfigRequest{
		HasFlushIntervalMs: true, FlushIntervalMs: 300,
		HasMinDeltaBytes: true, MinDeltaBytes: 2048,
		HasOverlapBytes: true, OverlapBytes: 1024,
		HasPersistentMode: true, PersistentMode: true,
		HasWakeWordEnabled: true, WakeWordEnabled: true,
		HasWakeWordThreshold: true, WakeWordThreshold: 0.7,
		HasSegmentSilenceMs: true, SegmentSilenceMs: 900,
		HasStreamingMode: true, StreamingMode: "auto",
		HasStrategyPreference: true, StrategyPreference: "vad",
		HasVadSilenceMs: true, VadSilenceMs: 700,
		HasOverlapWindowMs: true, OverlapWindowMs: 2000,
		HasOverlapCommitRuns: true, OverlapCommitRuns: 2,
	}))
	require.NoError(t, err)

	get, err := c.GetStreamConfig(context.Background(), connect.NewRequest(&sttv1.GetStreamConfigRequest{}))
	require.NoError(t, err)
	require.Equal(t, int32(300), get.Msg.GetConfig().GetFlushIntervalMs())
	require.Equal(t, "vad", get.Msg.GetConfig().GetStrategyPreference())
}

func TestStreamConfig_UpdateRejectsInvalidStreamingMode(t *testing.T) {
	scs := newStreamCfgStoreT(t)
	c := newSTTClient(t, Deps{StreamConfig: scs})
	_, err := c.UpdateStreamConfig(context.Background(), connect.NewRequest(&sttv1.UpdateStreamConfigRequest{
		HasStreamingMode: true, StreamingMode: "bogus",
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
	h := MultipartTranscribeHandler(nil)
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
		Vrooli:       sttchain.NewVrooliProvider(&stubVrooliSTT{available: true, res: &sttchain.Result{Text: "hello"}}),
	})
	h := MultipartTranscribeHandler(chain)
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
	h := MultipartTranscribeHandler(chain)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/transcribe", strings.NewReader("not-multipart"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=bogus")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMultipartTranscribe_HappyPath(t *testing.T) {
	chain := sttchain.NewChain(sttchain.Options{
		EnableVrooli: true,
		Vrooli: sttchain.NewVrooliProvider(&stubVrooliSTT{
			available: true,
			res:       &sttchain.Result{Text: "hello", DetectedLanguage: "en"},
		}),
	})
	h := MultipartTranscribeHandler(chain)
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
		Vrooli:       sttchain.NewVrooliProvider(&stubVrooliSTT{available: true, err: sttchain.ErrInsufficientCredits}),
	})
	h := MultipartTranscribeHandler(chain)
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
