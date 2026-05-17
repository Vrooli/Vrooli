package bootstrap

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"audio-tools/internal/byokstore"
	"audio-tools/internal/store"
)

// Stores bundles every persistence-backed store the API handlers need.
type Stores struct {
	ProviderConfig *store.ProviderConfigStore
	VoiceOverrides *store.VoiceOverrideStore
	BYOKRepo       *store.BYOKStore
	Usage          *store.UsageStore
	Wakeword       *store.WakeWordStore
	Speaker        *store.SpeakerStore
	STTStream      *store.STTStreamConfigStore
	TTSConfig      *store.TTSConfigStore
	Playback       *store.PlaybackStore
	BYOK           *byokstore.Store
}

// BuildStores constructs every store plus the BYOK encryptor-backed store.
func BuildStores(db *sql.DB, env Env) (Stores, error) {
	providerStore := store.NewProviderConfigStore(db, store.ProviderConfig{
		BYOKEnabled:         env.EnableBYOK,
		VrooliEnabled:       env.EnableVrooli,
		LocalEnabled:        env.EnableLocal,
		WhisperURL:          env.WhisperURL,
		KokoroURL:           env.KokoroURL,
		OllamaURL:           env.OllamaURL,
		LPBSBaseURL:         env.LPBSBaseURL,
		LPBSAppBundleKey:    env.LPBSAppBundleKey,
		AvailTTLBYOKSeconds: int32(env.AvailTTLBYOK / time.Second),
		AvailTTLVrooliSecs:  int32(env.AvailTTLVrooli / time.Second),
	})

	byokRepo := store.NewBYOKStore(db)

	keyPath := env.DBKeyPath
	if keyPath == "" {
		keyPath = filepath.Join(os.TempDir(), "audio-tools-key")
	}
	key, err := byokstore.LoadOrCreateKey(keyPath)
	if err != nil {
		return Stores{}, fmt.Errorf("byok key init: %w", err)
	}
	encryptor, err := byokstore.NewEncryptor(key)
	if err != nil {
		return Stores{}, fmt.Errorf("byok encryptor: %w", err)
	}
	byokStore := byokstore.New(encryptor, byokRepo)

	return Stores{
		ProviderConfig: providerStore,
		VoiceOverrides: store.NewVoiceOverrideStore(db),
		BYOKRepo:       byokRepo,
		Usage:          store.NewUsageStore(db),
		Wakeword:       store.NewWakeWordStore(db),
		Speaker:        store.NewSpeakerStore(db),
		STTStream:      store.NewSTTStreamConfigStore(db),
		TTSConfig:      store.NewTTSConfigStore(db),
		Playback:       store.NewPlaybackStore(db),
		BYOK:           byokStore,
	}, nil
}
