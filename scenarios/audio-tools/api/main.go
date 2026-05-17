package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"audio-tools/internal/ai/chains"
	"audio-tools/internal/ai/sttchain"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/byok"
	"audio-tools/internal/byokstore"
	"audio-tools/internal/store"
	"audio-tools/internal/usagereport"
	"sync/atomic"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/clock"
	"audio-tools/internal/modules"
	"audio-tools/internal/server"
	intsession "audio-tools/internal/session"
	sttpipeline "audio-tools/internal/stt/pipeline"
	intsumm "audio-tools/internal/summarize"
	inttts "audio-tools/internal/tts"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	audioH "audio-tools/handlers/audio"
	healthH "audio-tools/handlers/health"
	sessionH "audio-tools/handlers/session"
	settingsH "audio-tools/handlers/settings"
	sttH "audio-tools/handlers/stt"
	summarizeH "audio-tools/handlers/summarize"
	ttsH "audio-tools/handlers/tts"
	usageH "audio-tools/handlers/usage"
)

// sqliteDSN resolves the SQLite database file path and wraps it in a DSN
// with the canonical pragma string.
func sqliteDSN() (string, error) {
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_DB")); path != "" {
		return sqliteFileDSN(path)
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: "audio-tools"},
		storage.ClassData,
		"audio-tools.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve audio-tools db path: %w", err)
	}
	return sqliteFileDSN(path)
}

func sqliteFileDSN(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		path,
	), nil
}

func envBool(key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch v {
	case "":
		return def
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return def
}

func envOr(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func envDuration(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func main() {
	if preflight.Run(preflight.Config{ScenarioName: "audio-tools"}) {
		return
	}

	dsn, err := sqliteDSN()
	if err != nil {
		log.Fatalf("sqlite configuration failed: %v", err)
	}

	db, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db, modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	logger := log.Default()

	// Build the capability domain singletons. These wrap the ported web-console
	// services and feed the Local* providers in the three provider chains.
	capsRegistry := capabilities.NewRegistry(nil, nil, 30*time.Second)
	skipVerifyCount := &atomic.Int64{}
	voiceSvc := sttpipeline.NewService(
		sttpipeline.Config{},
		"", // configPath — defaults
		nil,
		"", // wake-word path — defaults
		sttpipeline.SpeakerConfig{},
		"", // speaker path
		nil,
		capsRegistry,
		skipVerifyCount,
		envOr("AUDIO_WHISPER_URL", "http://localhost:8090"),
		nil,
		nil,
	)
	ttsCache := inttts.NewCache(64 * 1024 * 1024) // 64 MiB content-addressable cache
	kokoroURL := envOr("AUDIO_KOKORO_URL", "http://localhost:8880")
	kokoroSynth := &inttts.KokoroSynthesizer{BaseURL: kokoroURL}
	ttsCfg := inttts.DefaultConfig()
	ttsSummCfg := intsumm.DefaultSummarizeConfig()
	ttsSvc := inttts.NewService(inttts.Deps{
		Logger:        logger,
		GetConfig:     func() inttts.Config { return ttsCfg },
		SetConfig:     func(c inttts.Config) { ttsCfg = c },
		PersistConfig: func(inttts.Config) error { return nil },
		KokoroCapability: func(ctx context.Context) (string, string) {
			return "available", "Kokoro (Local)"
		},
		SynthesizeAudio: func(ctx context.Context, in inttts.SynthesizeInput) (io.ReadCloser, string, error) {
			return kokoroSynth.Synthesize(ctx, inttts.SynthesizeRequest{
				Input:          in.Input,
				Voice:          in.Voice,
				ResponseFormat: in.ResponseFormat,
				Speed:          in.Speed,
			})
		},
		GetCache: func(key inttts.CacheKey) (inttts.SynthesizeResult, bool) {
			out, ok := ttsCache.Get(key)
			if !ok {
				return inttts.SynthesizeResult{}, false
			}
			return inttts.SynthesizeResult{Audio: out.Audio, ContentType: out.ContentType}, true
		},
		PutCache: func(key inttts.CacheKey, audio []byte, ct string) {
			ttsCache.Put(key, audio, ct)
		},
	})
	ollamaSummarizer := intsumm.NewSummarizer(envOr("AUDIO_OLLAMA_URL", "http://localhost:11434"))

	enableBYOK := envBool("AUDIO_AI_ENABLE_BYOK", true)
	enableVrooli := envBool("AUDIO_AI_ENABLE_VROOLI", false)
	enableLocal := envBool("AUDIO_AI_ENABLE_LOCAL", true)
	ttlByOK := envDuration("AUDIO_AVAIL_TTL_BYOK", 5*time.Minute)
	ttlVrooli := envDuration("AUDIO_AVAIL_TTL_VROOLI", 30*time.Second)

	byokRegistries := byok.NewRegistries()

	sttChain := sttchain.NewChain(sttchain.Options{
		Local:          sttchain.NewLocalProvider(voiceSvc),
		BYOK:           sttchain.NewBYOKProvider(byokRegistries.STT),
		EnableLocal:    enableLocal,
		EnableBYOK:     enableBYOK,
		EnableVrooli:   enableVrooli,
		AvailTTLByOK:   ttlByOK,
		AvailTTLVrooli: ttlVrooli,
	})
	ttsChain := ttschain.NewChain(ttschain.Options{
		Local:          ttschain.NewLocalProvider(ttsSvc),
		BYOK:           ttschain.NewBYOKProvider(byokRegistries.TTS),
		EnableLocal:    enableLocal,
		EnableBYOK:     enableBYOK,
		EnableVrooli:   enableVrooli,
		AvailTTLByOK:   ttlByOK,
		AvailTTLVrooli: ttlVrooli,
	})
	summChain := summarizechain.NewChain(summarizechain.Options{
		Local:          summarizechain.NewLocalProvider(ollamaSummarizer, envOr("AUDIO_SUMMARIZE_DEFAULT_MODEL", "qwen3:4b")),
		BYOK:           summarizechain.NewBYOKProvider(byokRegistries.Summarize),
		EnableLocal:    enableLocal,
		EnableBYOK:     enableBYOK,
		EnableVrooli:   enableVrooli,
		AvailTTLByOK:   ttlByOK,
		AvailTTLVrooli: ttlVrooli,
	})

	sessionRegistry := intsession.NewRegistry()

	// --- persistence-backed stores ---
	providerStore := store.NewProviderConfigStore(db, store.ProviderConfig{
		BYOKEnabled:         enableBYOK,
		VrooliEnabled:       enableVrooli,
		LocalEnabled:        enableLocal,
		WhisperURL:          envOr("AUDIO_WHISPER_URL", "http://localhost:8090"),
		KokoroURL:           kokoroURL,
		OllamaURL:           envOr("AUDIO_OLLAMA_URL", "http://localhost:11434"),
		LPBSBaseURL:         envOr("AUDIO_LPBS_BASE_URL", ""),
		LPBSAppBundleKey:    envOr("AUDIO_LPBS_APP_BUNDLE_KEY", ""),
		AvailTTLBYOKSeconds: int32(ttlByOK / time.Second),
		AvailTTLVrooliSecs:  int32(ttlVrooli / time.Second),
	})
	voiceOverrideStore := store.NewVoiceOverrideStore(db)
	byokRepo := store.NewBYOKStore(db)
	usageStore := store.NewUsageStore(db)
	wakewordStore := store.NewWakeWordStore(db)
	speakerStore := store.NewSpeakerStore(db)
	sttStreamStore := store.NewSTTStreamConfigStore(db)
	ttsConfigStore := store.NewTTSConfigStore(db)
	playbackStore := store.NewPlaybackStore(db)

	keyPath := envOr("AUDIO_TOOLS_DB_KEY_PATH", "")
	if keyPath == "" {
		keyPath = filepath.Join(os.TempDir(), "audio-tools-key")
	}
	key, err := byokstore.LoadOrCreateKey(keyPath)
	if err != nil {
		log.Fatalf("byok key init failed: %v", err)
	}
	encryptor, err := byokstore.NewEncryptor(key)
	if err != nil {
		log.Fatalf("byok encryptor: %v", err)
	}
	byokStore := byokstore.New(encryptor, byokRepo)

	coordinator := &chains.Coordinator{STT: sttChain, TTS: ttsChain, Summarize: summChain}

	// Async usage reporter — chains enqueue rows after each call.
	usageRecorder := usagereport.New(usageStore, logger)


	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: logger},
		healthH.Module(db, "audio-tools-api", "1.0.0"),
		audioH.Module(logger),
		sessionH.Module(sessionRegistry, logger),
		settingsH.Module(settingsH.Deps{
			Logger:         logger,
			ProviderConfig: providerStore,
			BYOK:           byokStore,
			VoiceOverrides: voiceOverrideStore,
			Coordinator:    coordinator,
		}),
		sttH.Module(sttH.Deps{
			Chain:        sttChain,
			Selector:     sttpkg.NewSelector(sttChain),
			Voice:        voiceSvc,
			Logger:       logger,
			StreamConfig: sttStreamStore,
			Wakeword:     wakewordStore,
			Speaker:      speakerStore,
		}),
		summarizeH.Module(
			summChain,
			func() intsumm.SummarizeConfig { return ttsSummCfg },
			func(c intsumm.SummarizeConfig) { ttsSummCfg = c },
			logger,
			usageRecorder,
		),
		ttsH.Module(ttsH.Deps{
			Chain:          ttsChain,
			SummarizeChain: summChain,
			TTSService:     ttsSvc,
			Logger:         logger,
			Cache:          ttsCache,
			ConfigStore:    ttsConfigStore,
			Playback:       playbackStore,
		}),
		usageH.Module(usageH.Deps{Logger: logger, Store: usageStore}),
	)

	if err := apiserver.Run(apiserver.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
