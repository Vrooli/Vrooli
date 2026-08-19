package bootstrap

import (
	"fmt"
	"path/filepath"
	"time"

	"audio-tools/internal/byokstore"
	"audio-tools/internal/protoint"
	"audio-tools/internal/store"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
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
	STTSpeaker     *store.STTSpeakerConfigStore
	TTSConfig      *store.TTSConfigStore
	Playback       *store.PlaybackStore
	BYOK           *byokstore.Store
}

// BuildStores constructs every store plus the BYOK encryptor-backed store.
func BuildStores(db *database.RoutedDB, env Env) (Stores, error) {
	providerStore := store.NewProviderConfigStore(db, store.ProviderConfig{
		BYOKEnabled:         env.EnableBYOK,
		VrooliEnabled:       env.EnableVrooli,
		LocalEnabled:        env.EnableLocal,
		WhisperURL:          env.WhisperURL,
		KokoroURL:           env.SherpaURL,
		OllamaURL:           env.OllamaURL,
		LPBSBaseURL:         env.LPBSBaseURL,
		LPBSAppBundleKey:    env.LPBSAppBundleKey,
		AvailTTLBYOKSeconds: protoint.FromInt64(int64(env.AvailTTLBYOK / time.Second)),
		AvailTTLVrooliSecs:  protoint.FromInt64(int64(env.AvailTTLVrooli / time.Second)),
	})

	byokRepo := store.NewBYOKStore(db)

	keyPath := env.DBKeyPath
	if keyPath == "" {
		namespace, err := storage.ScenarioNamespace("audio-tools")
		if err != nil {
			return Stores{}, fmt.Errorf("resolve BYOK key namespace: %w", err)
		}
		resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
		if err != nil {
			return Stores{}, fmt.Errorf("create BYOK key storage resolver: %w", err)
		}
		paths, err := resolver.Resolve(storage.Options{ScenarioID: namespace})
		if err != nil {
			return Stores{}, fmt.Errorf("resolve BYOK key storage: %w", err)
		}
		keyPath = filepath.Join(paths.DataDir, "byok", "encryption.key")
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
		STTSpeaker:     store.NewSTTSpeakerConfigStore(db),
		TTSConfig:      store.NewTTSConfigStore(db),
		Playback:       store.NewPlaybackStore(db),
		BYOK:           byokStore,
	}, nil
}
