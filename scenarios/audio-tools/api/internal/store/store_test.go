package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	localdb "audio-tools/internal/database"
	"audio-tools/internal/store"
	"audio-tools/internal/testutil/db"
)

func TestProviderConfigStore_RoundTrip(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema)))
	s := store.NewProviderConfigStore(d, store.ProviderConfig{
		BYOKEnabled: true, LocalEnabled: true,
		WhisperURL: "http://w", KokoroURL: "http://k", OllamaURL: "http://o",
		AvailTTLBYOKSeconds: 300, AvailTTLVrooliSecs: 30,
	})

	got, err := s.Get(context.Background())
	require.NoError(t, err)
	require.True(t, got.BYOKEnabled)
	require.Equal(t, "http://w", got.WhisperURL)

	want := false
	url := "http://w2"
	got2, err := s.Update(context.Background(), store.ProviderConfigPatch{BYOKEnabled: &want, WhisperURL: &url})
	require.NoError(t, err)
	require.False(t, got2.BYOKEnabled)
	require.Equal(t, "http://w2", got2.WhisperURL)
}

func TestBYOKStore_RoundTrip(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema)))
	s := store.NewBYOKStore(d)
	ctx := context.Background()

	require.NoError(t, s.Upsert(ctx, store.BYOKCredential{
		ProviderID: "openai-whisper", Capability: "stt",
		Cipher: []byte("ciphertext"), Fingerprint: "sk-***abcd",
	}))
	list, err := s.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "openai-whisper", list[0].ProviderID)

	deleted, err := s.Delete(ctx, "openai-whisper", "stt")
	require.NoError(t, err)
	require.True(t, deleted)

	list, err = s.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 0)
}

func TestUsageStore_InsertListSummary(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema)))
	s := store.NewUsageStore(d)
	ctx := context.Background()
	now := time.Now().UTC()
	require.NoError(t, s.Insert(ctx, store.UsageRow{
		OperationID: "op-1", EmittedAt: now.Add(-1 * time.Minute),
		Capability: "stt", Operation: "transcribe",
		ProviderTier: "local", ProviderID: "whisper", LatencyMs: 120, CreditsCharged: 0,
	}))
	require.NoError(t, s.Insert(ctx, store.UsageRow{
		OperationID: "op-2", EmittedAt: now.Add(-30 * time.Second),
		Capability: "tts", Operation: "synthesize",
		ProviderTier: "local", ProviderID: "kokoro", LatencyMs: 200, CreditsCharged: 1,
	}))
	// Idempotent on op id
	require.NoError(t, s.Insert(ctx, store.UsageRow{OperationID: "op-1", Capability: "stt", Operation: "transcribe", ProviderTier: "local", ProviderID: "whisper"}))

	rows, err := s.ListRecent(ctx, now.Add(-1*time.Hour), 10, "", "")
	require.NoError(t, err)
	require.Len(t, rows, 2)

	sum, err := s.Summary(ctx, now.Add(-1*time.Hour), "")
	require.NoError(t, err)
	require.EqualValues(t, 2, sum.OperationsTotal)
	require.EqualValues(t, 1, sum.CreditsTotal)
	require.Len(t, sum.Distribution, 2)
}

func TestVoiceOverridesStore(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema)))
	s := store.NewVoiceOverrideStore(d)
	ctx := context.Background()
	require.NoError(t, s.Set(ctx, store.VoiceOverride{CanonicalVoice: "voice.feminine.warm", TierProvider: "byok:elevenlabs", AdapterVoice: "Rachel"}))
	list, err := s.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	// Empty adapter = delete
	require.NoError(t, s.Set(ctx, store.VoiceOverride{CanonicalVoice: "voice.feminine.warm", TierProvider: "byok:elevenlabs"}))
	list, err = s.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 0)
}

func TestJSONSingletonStores(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema)))
	ctx := context.Background()

	stts := store.NewSTTStreamConfigStore(d)
	_, ok, err := stts.Get(ctx)
	require.NoError(t, err)
	require.False(t, ok)
	require.NoError(t, stts.Set(ctx, `{"sample_rate":16000}`))
	v, ok, err := stts.Get(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `{"sample_rate":16000}`, v)

	ttss := store.NewTTSConfigStore(d)
	require.NoError(t, ttss.Set(ctx, `{"voice":"warm"}`, `{"level":"moderate"}`))
	cfg, summ, ok, err := ttss.Get(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `{"voice":"warm"}`, cfg)
	require.Equal(t, `{"level":"moderate"}`, summ)
}

func TestSpeakerAndWakeword(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema)))
	ctx := context.Background()

	sp := store.NewSpeakerStore(d)
	require.NoError(t, sp.Upsert(ctx, store.SpeakerProfile{ID: "sp1", Name: "alice", Embedding: []byte{1, 2, 3}, BoundUserIdentity: "user-1"}))
	got, ok, err := sp.Get(ctx, "sp1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "user-1", got.BoundUserIdentity)
	require.NoError(t, sp.ClearBinding(ctx, "sp1"))
	got, _, _ = sp.Get(ctx, "sp1")
	require.Empty(t, got.BoundUserIdentity)

	ww := store.NewWakeWordStore(d)
	require.NoError(t, ww.Upsert(ctx, store.WakeWordTemplate{ID: "hello", Phrase: "hello vrooli", Embedding: []byte{4, 5}}))
	tmpl, ok, err := ww.Get(ctx, "hello")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "hello vrooli", tmpl.Phrase)
}

func TestPlaybackStore(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema)))
	ctx := context.Background()
	ps := store.NewPlaybackStore(d)
	require.NoError(t, ps.Insert(ctx, store.PlaybackEvent{EventID: "e1", Kind: "start", Voice: "warm", ProviderTier: "local", ProviderID: "kokoro"}))
	require.NoError(t, ps.Insert(ctx, store.PlaybackEvent{EventID: "e1", Kind: "start"})) // idempotent
	require.NoError(t, ps.Insert(ctx, store.PlaybackEvent{EventID: "e2", Kind: "finish"}))
	rows, err := ps.List(ctx, 10)
	require.NoError(t, err)
	require.Len(t, rows, 2)
}
