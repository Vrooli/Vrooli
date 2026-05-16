package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"web-console/session"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	aiH "web-console/handlers/ai"
	capabilitiesH "web-console/handlers/capabilities"
	conversationH "web-console/handlers/conversation"
	discoveryH "web-console/handlers/discovery"
	eventsH "web-console/handlers/events"
	hooksH "web-console/handlers/hooks"
	metricsH "web-console/handlers/metrics"

	sessionsH "web-console/handlers/sessions"
	settingsH "web-console/handlers/settings"
	shortcutsH "web-console/handlers/shortcuts"
	terminalH "web-console/handlers/terminal"
	workspaceH "web-console/handlers/workspace"
	intai "web-console/internal/ai"
	"web-console/internal/audioports"
	audiotoolsint "web-console/integrations/audiotools"
	"web-console/internal/backend"
	"web-console/internal/capabilities"
	"web-console/internal/config"
	"web-console/internal/events"
	"web-console/internal/metrics"
	intsessions "web-console/internal/sessions"
	"web-console/internal/sessionstore"
	intworkspace "web-console/internal/workspace"
)

// initSchema runs the idempotent schema and seed SQL against the database.
// SQL files are read from the initialization directory relative to the binary.
func initSchema(db *sql.DB) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	base := filepath.Dir(exe)

	for _, file := range []string{
		filepath.Join(base, "..", "initialization", "sqlite", "schema.sql"),
		filepath.Join(base, "..", "initialization", "sqlite", "seed.sql"),
	} {
		sql, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read %s: %w", filepath.Base(file), err)
		}
		if _, err := db.Exec(string(sql)); err != nil {
			return fmt.Errorf("exec %s: %w", filepath.Base(file), err)
		}
	}
	log.Println("Schema initialized successfully")

	// Migrations: add columns to existing tables. ALTER TABLE ADD COLUMN
	// errors if the column already exists, so we ignore that specific error.
	migrations := []string{
		`ALTER TABLE workspace_panes ADD COLUMN supports_messages_view INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE sessions ADD COLUMN backend TEXT NOT NULL DEFAULT 'standard'`,
		`ALTER TABLE sessions ADD COLUMN detached INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE sessions ADD COLUMN status TEXT NOT NULL DEFAULT 'live'`,
		`ALTER TABLE sessions ADD COLUMN agent_type TEXT NOT NULL DEFAULT 'none'`,
		`ALTER TABLE sessions ADD COLUMN launch_command TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sessions ADD COLUMN agent_session_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sessions ADD COLUMN cwd TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sessions ADD COLUMN last_rollout_path TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sessions ADD COLUMN last_activity_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sessions ADD COLUMN orphaned_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sessions ADD COLUMN recovered_into TEXT NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_agent ON sessions(agent_type, agent_session_id)`,
	}
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			// "duplicate column name" means the column already exists — safe to ignore.
			if !isDuplicateColumnError(err) {
				return fmt.Errorf("migration: %w", err)
			}
		}
	}

	return nil
}

// isDuplicateColumnError checks if a SQLite error is a "duplicate column name" error.
func isDuplicateColumnError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate column name")
}

// DOC: docs/concepts/ARCHITECTURE.md#system-layers
// Server wires the HTTP router, database connection, and session manager.
//
// Audio fields: speechProcessor, sttPort, ttsPort and summarizer are audio
// capability ports backed by the audio-tools scenario in production; tests
// substitute via the Set* methods. audio-tools is a required dependency
// declared in .vrooli/service.json — the lifecycle ensures it is running
// before web-console boots. All voice/TTS synthesis, summarization, voice
// listing, and cache logic lives in audio-tools. The web-console-side state
// is limited to: the small Claude-hook routing diagnostics
// (lastTTS* fields), the Claude-hook auto/backend/startMuted preference
// triple (ttsHookConfigState), and the auto-summarize policy cache
// (summarizeAutoPolicy*).
type Server struct {
	db                   *sql.DB
	router               *mux.Router
	sessions             *session.Manager
	fanouts              *ConversationFanoutRegistry
	events               *events.Logger
	metrics              *metrics.Metrics
	backendRegistry      *backend.Registry
	sessionStore         sessionstore.Store
	aiChain              *intai.Chain
	shortcuts            ShortcutStore
	aiConfig             intai.ConfigStore
	ai                   *intai.Service
	sweeper              *session.ExpirationSweeper
	idempotency          *intsessions.IdempotencyCache // replay-safe session creation
	capabilities         *capabilities.Registry
	workspace            intworkspace.Store
	hookAuthToken        string
	codexTailer          *CodexTailer
	codexCheckpointStore CodexCheckpointStore

	// Audio ports — all backed by audio-tools in production.
	speechProcessor audioports.SpeechTextProcessor
	sttPort         audioports.SpeechToText
	ttsPort         audioports.TextToSpeech
	summarizer      audioports.Summarizer

	// audioToolsResolver is the live audio-tools URL resolver, kept on
	// the server so the discovery handler can re-query it per request.
	audioToolsResolver audiotoolsint.URLResolver

	// Hook routing diagnostics + auto-config (web-console-internal).
	ttsHookConfigState    hookConfigState
	summarizeAutoPolicyMu sync.RWMutex
	summarizeAutoPolicy   SummarizeAutoPolicy

	conversations   *ConversationStore
	ttsStatusMu     sync.RWMutex
	lastTTSRouting  *ConversationAppendResult
	lastTTSAt       time.Time
	lastTTSBySource map[string]conversationAppendSnapshot
	lastTTSAck      *TTSClientAck
	lastTTSAckAt    time.Time
	lastTTSAckBySrc map[string]ttsAckSnapshot
	lastTTSPlayback *TTSPlaybackEvent
	lastTTSPlayAt   time.Time
	systemContext   *intai.SystemContext
	// nextWSGen is a monotonically increasing generation counter; each
	// new terminal WebSocket connection gets a fresh Gen that is echoed
	// to the client in session_ready. Clients use it as the wsGen write
	// barrier on pending-ack re-enqueue (see useStdinAck).
	nextWSGen atomic.Int64
}

// getSummarizeAutoPolicy returns the cached auto-summarize policy. The
// canonical config lives in audio-tools; web-console caches the subset the
// auto path needs (enabled + char threshold + level + timeout) so the
// router doesn't make a Connect call on every assistant event.
func (s *Server) getSummarizeAutoPolicy() SummarizeAutoPolicy {
	s.summarizeAutoPolicyMu.RLock()
	defer s.summarizeAutoPolicyMu.RUnlock()
	if s.summarizeAutoPolicy.CharThreshold == 0 && !s.summarizeAutoPolicy.Enabled {
		return defaultSummarizeAutoPolicy()
	}
	return s.summarizeAutoPolicy
}

// SetSummarizeAutoPolicy updates the cached policy. Production wiring polls
// audio-tools' GetSummarizeConfig on a slow schedule; tests inject directly.
func (s *Server) SetSummarizeAutoPolicy(p SummarizeAutoPolicy) {
	s.summarizeAutoPolicyMu.Lock()
	defer s.summarizeAutoPolicyMu.Unlock()
	s.summarizeAutoPolicy = p
}

type conversationAppendSnapshot struct {
	Result ConversationAppendResult
	At     time.Time
}

type ttsAckSnapshot struct {
	Result TTSClientAck
	At     time.Time
}

type TTSPlaybackEvent struct {
	Source    string `json:"source"`
	Stage     string `json:"stage"`
	Backend   string `json:"backend,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Message   string `json:"message,omitempty"`
}

// NewServer initializes database, session manager, and routes.
// It runs the schema initialization against the database and creates
// SQLite-backed stores for shortcuts and AI config.
func NewServer(db *sql.DB) *Server {
	if err := initSchema(db); err != nil {
		log.Fatalf("Schema initialization failed: %v", err)
	}

	// Load the small Claude-hook routing preference triple (auto/backend/
	// startMuted). Voice/speed/summarize knobs live in audio-tools — fetched
	// lazily by the UI via @audio-tools/embed, not loaded here.
	hookCfgPath := resolveTTSHookConfigPath()
	hookCfg, err := loadTTSHookConfig(hookCfgPath)
	if err != nil {
		log.Printf("tts-hook-config: using defaults: %v", err)
		hookCfg = DefaultTTSHookConfig()
	}
	log.Printf("tts-hook-config: loaded: autoEnabled=%v backend=%s startMuted=%v",
		hookCfg.AutoEnabled, hookCfg.Backend, hookCfg.StartMuted)

	// Generate or load hook auth token for Claude Stop hook validation.
	hookToken := loadOrCreateHookToken(resolveHookTokenPath())

	eventLog := events.NewLogger(1000)
	metrics := metrics.New()
	sessions := newSessionManager()
	fanouts := NewConversationFanoutRegistry().AttachToManager(sessions)

	// Initialize backend registry and session metadata store
	backendRegistry := InitDefaultRegistry()
	sessionStore := sessionstore.NewSQL(db)
	sessions.SetRegistry(backendRegistry)
	sessions.SetStore(sessionStore)
	sessions.SetMetrics(metrics)
	sessions.SetEvents(eventLog)

	// Resolve "auto" default backend now that the registry knows tmux availability.
	if sessions.GetConfig().DefaultBackend == "auto" {
		resolved := backendRegistry.ResolveAuto()
		sessions.SetConfigField(func(cfg *config.Config) { cfg.DefaultBackend = string(resolved) })
		log.Printf("default-backend: resolved 'auto' -> %q", resolved)
	}

	// Recover surviving tmux sessions from previous run
	report := sessions.Recover(sessionStore, backendRegistry)
	log.Printf("recovery: recovered=%d awaiting_recovery=%d orphaned_tmux=%d (awaiting_recovery rows preserved for explicit recovery via /api/v1/sessions/recoverable)",
		report.Recovered, report.AwaitingRecovery, report.OrphanedTmux)

	srv := &Server{
		db:                   db,
		router:               mux.NewRouter(),
		sessions:             sessions,
		fanouts:              fanouts,
		events:               eventLog,
		metrics:              metrics,
		backendRegistry:      backendRegistry,
		sessionStore:         sessionStore,
		aiChain:              intai.NewChain(intai.NewOllamaProvider(), intai.NewOpenRouterProvider()),
		shortcuts:            NewSQLShortcutStore(db),
		aiConfig:             intai.NewSQLConfigStore(db),
		sweeper:              session.NewExpirationSweeper(sessions, eventLog, metrics),
		idempotency:          intsessions.NewIdempotencyCache(),
		workspace:            intworkspace.NewSQLStore(db),
		hookAuthToken:        hookToken,
		ttsHookConfigState: hookConfigState{
			cfg:  hookCfg,
			path: hookCfgPath,
		},
		summarizeAutoPolicy:  defaultSummarizeAutoPolicy(),
		codexCheckpointStore: NewSQLCodexCheckpointStore(db),
		conversations:        NewConversationStoreWithRepository(NewSQLConversationRepository(db)),
		lastTTSBySource:      make(map[string]conversationAppendSnapshot),
		lastTTSAckBySrc:      make(map[string]ttsAckSnapshot),
		speechProcessor:      audioports.PassthroughSpeechTextProcessor{},
	}
	srv.systemContext = intai.DiscoverSystemContext(intai.DefaultLookPath)
	log.Printf("system-context: os=%s/%s shell=%s tools-found=%d",
		srv.systemContext.OS, srv.systemContext.Arch,
		srv.systemContext.Shell, intai.CountFoundTools(srv.systemContext.Tools))
	srv.ai = intai.NewService(srv.aiChain, srv.aiConfig, srv.systemContext, srv.events, &srv.metrics.AIGenerations, &srv.metrics.AISuggestions)

	ollamaURL := getEnvOrDefault("OLLAMA_URL", "http://localhost:11434")
	openrouterKey := os.Getenv("OPENROUTER_API_KEY")

	checkers := map[string]capabilities.Checker{
		"ollama": &capabilities.OllamaChecker{
			BaseURL: ollamaURL,
			Client:  &http.Client{Timeout: 5 * time.Second},
		},
		"openrouter": &capabilities.OpenRouterChecker{
			APIKey: openrouterKey,
			Client: &http.Client{Timeout: 5 * time.Second},
		},
		// Connected scenarios. Each DependencyScenario entry in
		// capabilities.Known needs a checker so the integrations UI can
		// render real status. These shell out to the Vrooli CLI rather than
		// calling another scenario's API directly — see
		// project_wrap_not_use_principle. audio-tools owns Whisper / Kokoro /
		// speaker-verification end-to-end.
		"audio-tools": &capabilities.ScenarioChecker{Slug: "audio-tools"},
	}
	srv.capabilities = capabilities.NewRegistry(capabilities.Known, checkers, 30*time.Second)
	srv.capabilities.SetLivenessCheckers(map[string]capabilities.Checker{
		"ollama": &capabilities.ResourceChecker{
			URL:    ollamaURL + "/api/tags",
			Client: &http.Client{Timeout: 5 * time.Second},
		},
		"openrouter": &capabilities.OpenRouterChecker{
			APIKey: openrouterKey,
		},
		"audio-tools": &capabilities.ScenarioChecker{Slug: "audio-tools"},
	})
	// Warm capability cache so the first /capabilities request returns instantly.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.capabilities.Resolve(ctx)
		log.Println("capabilities: initial check complete")
	}()

	// audio-tools is a required dependency (.vrooli/service.json). Resolve
	// it once at startup and wire the audioports.Remote* adapters; the
	// lifecycle ensures it is running before web-console boots.
	// AUDIO_TOOLS_URL pins to an explicit URL for dev/test override.
	var atResolver audiotoolsint.URLResolver
	if explicit := strings.TrimSpace(os.Getenv("AUDIO_TOOLS_URL")); explicit != "" {
		atResolver = audiotoolsint.EnvResolver{EnvVar: "AUDIO_TOOLS_URL", Default: explicit}
	} else {
		atResolver = &audiotoolsint.CachedResolver{
			Inner: audiotoolsint.ScenarioResolver{Slug: "audio-tools"},
			TTL:   30 * time.Second,
		}
	}
	atClient, err := audiotoolsint.New(atResolver, audiotoolsint.Policy{Required: true})
	if err != nil {
		log.Fatalf("audio-tools adoption: required dependency not reachable: %v", err)
	}
	srv.sttPort = &audioports.RemoteSpeechToText{Client: atClient}
	srv.ttsPort = &audioports.RemoteTextToSpeech{Client: atClient}
	srv.speechProcessor = &audioports.RemoteSpeechTextProcessor{Client: atClient}
	srv.summarizer = &audioports.RemoteSummarizer{Client: atClient}
	srv.audioToolsResolver = atResolver
	log.Printf("audio-tools adoption: STT/TTS/processor/summarize wired to %s", atClient.BaseURL())

	srv.sweeper.Start()
	sessions.StartReattachWatchdog()
	srv.setupRoutes()

	// Start Codex rollout tailer for auto-TTS.
	srv.codexTailer = NewCodexTailer(srv)
	srv.codexTailer.Start()
	log.Println("codex-tailer: started watching for per-session rollout files")

	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(requestIDMiddleware)
	s.router.Use(loggingMiddleware)

	// Health endpoints
	healthHandler := health.New().
		Version("1.0.0").
		Check(health.DB(s.db), health.Critical).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Sessions domain (CRUD, recovery, policy) — Connect-RPC.
	sessionsH.Module(&sessionsH.Adapter{
		Manager:          s.sessions,
		Store:            s.sessionStore,
		Idempotency:      s.idempotency,
		Events:           s.events,
		Metrics:          s.metrics,
		Conversations:    s.conversations,
		CodexCheckpoints: s.codexCheckpointStore,
		CopyCodexHome:    copyCodexHome,
	}, nil).Mount(s.router)

	// Terminal — Connect-RPC TerminalService (GetScreen, SendInput,
	// WaitIdle) plus REST exceptions for the WebSocket bridge
	// ([REQ:P0-002b]) and multipart image upload for path injection.
	terminalH.Module(&terminalH.Adapter{Manager: s.sessions}, terminalH.LegacyDeps{
		Upload: s.handleUpload,
		WS:     s.handleTerminalWS,
	}, nil).Mount(s.router)

	// Workspace domain (panes, groups, layout) — Connect-RPC.
	workspaceH.Module(&workspaceH.Adapter{
		Store:  s.workspace,
		Events: s.events,
	}, nil).Mount(s.router)

	// Conversation domain (history, cursor, summarize, file refs) — Connect-RPC.
	conversationH.Module(newConversationAdapter(s), nil).Mount(s.router)

	// Settings domain — Connect-RPC, mounted via Module.
	settingsH.Module(newSettingsAdapter(s), nil).Mount(s.router)

	// Shortcut profiles - [REQ:P1-002a] (Connect-RPC ShortcutsService)
	shortcutsH.Module(newShortcutsAdapter(s), nil).Mount(s.router)

	// AI domain - [REQ:P0-005a] [REQ:P1-003a] [REQ:P1-003b] (Connect-RPC AIService)
	aiH.Module(&aiH.Adapter{Backend: s.ai}, nil).Mount(s.router)

	// Metrics — Connect-RPC MetricsService [REQ:P1-004b]
	metricsH.Module(&metricsH.Adapter{Metrics: s.metrics}, nil).Mount(s.router)

	// Events — Connect-RPC EventsService [REQ:P1-004a]
	eventsH.Module(&eventsH.Adapter{Logger: s.events}, nil).Mount(s.router)

	// Capabilities — Connect-RPC.
	capabilitiesH.Module(&capabilitiesH.Adapter{
		Registry:        s.capabilities,
		BackendRegistry: s.backendRegistry,
		DefaultBackend:  func() string { return string(s.sessions.GetConfig().DefaultBackend) },
	}, nil).Mount(s.router)

	// Discovery — exposes resolved scenario endpoint URLs (currently
	// just audio-tools) to the browser so the UI never composes
	// scenario URLs client-side.
	discoveryH.Module(newAudioToolsResolverAdapter(s), nil).Mount(s.router)

	// Hooks — REST webhook receivers (Claude Code CLI dictates wire shape).
	hooksH.Module(hooksH.Deps{
		Stop:         s.handleHookStop,
		PromptSubmit: s.handleHookPromptSubmit,
	}).Mount(s.router)

	// TTS hook routing diagnostics + auto/backend/startMuted preference triple.
	// REST exception per RESTReasonHostHookGlue — this is web-console-internal
	// Claude-hook glue and never crosses scenario boundaries. All audio
	// synthesis flows through Connect against audio-tools (via @audio-tools/embed).
	s.registerTTSHookRoutes()
}

// Handler returns the router wrapped with CORS and panic-recovery middleware.
// CORS accepts both localhost and 127.0.0.1 on the UI port so that desktop
// bundles (where the UI and API run on separate ports) work without a proxy.
func (s *Server) Handler() http.Handler {
	uiPort := getEnvOrDefault("UI_PORT", "36233")
	allowedOrigins := []string{
		fmt.Sprintf("http://localhost:%s", uiPort),
		fmt.Sprintf("http://127.0.0.1:%s", uiPort),
	}
	cors := handlers.CORS(
		handlers.AllowedOrigins(allowedOrigins),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "X-Request-ID"}),
	)
	return handlers.RecoveryHandler()(cors(s.router))
}

// SetSpeechToText substitutes the SpeechToText port. Tests use this to inject
// fakes; production wires the audioports.RemoteSpeechToText backed by
// audio-tools in NewServer.
func (s *Server) SetSpeechToText(p audioports.SpeechToText) {
	s.sttPort = p
}

// SetTextToSpeech substitutes the TextToSpeech port. Tests use this to inject
// fakes; production wires the audioports.RemoteTextToSpeech backed by
// audio-tools in NewServer.
func (s *Server) SetTextToSpeech(p audioports.TextToSpeech) {
	s.ttsPort = p
}

type contextKey string

const requestIDKey contextKey = "request_id"

// requestIDMiddleware generates a short request ID and stores it in context.
// The ID is returned in the X-Request-ID response header so clients and logs
// can correlate errors to the same request.
func requestIDMiddleware(next http.Handler) http.Handler {
	var counter uint64
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := fmt.Sprintf("req-%d-%d", time.Now().UnixMilli()%100000, atomic.AddUint64(&counter, 1))
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getRequestID returns the request ID from context, or "" if not set.
func getRequestID(r *http.Request) string {
	if id, ok := r.Context().Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// loggingMiddleware prints structured request logs including request ID.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		reqID := getRequestID(r)
		if reqID != "" {
			log.Printf("[%s] %s %s [%s]", r.Method, r.RequestURI, time.Since(start), reqID)
		} else {
			log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
		}
	})
}

// getEnvOrDefault returns the value of the named environment variable, or
// fallback if the variable is empty or unset.
func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// resolveSQLiteDSN builds the SQLite DSN with performance pragmas.
// Path is resolved via api-core/storage for cross-platform portability.
func resolveSQLiteDSN() string {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		log.Fatalf("storage resolver: %v", err)
	}

	opts := storage.Options{ScenarioID: "web-console"}
	if _, err := storage.EnsureClassDir(resolver, opts, storage.ClassData, 0); err != nil {
		log.Fatalf("ensure data dir: %v", err)
	}

	dbPath, err := resolver.Path(opts, storage.ClassData, "web-console.db")
	if err != nil {
		log.Fatalf("resolve db path: %v", err)
	}

	log.Printf("SQLite database: %s", dbPath)
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		dbPath,
	)
}

func resolveHookTokenPath() string {
	return mustResolveScenarioStoragePath(storage.ClassState, "hook-token.txt")
}

func resolveClaudeProjectSettingsPath() string {
	if explicit := os.Getenv("CLAUDE_PROJECT_SETTINGS"); explicit != "" {
		return explicit
	}
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Clean(filepath.Join(filepath.Dir(exe), "..", "..", "..", ".claude", "settings.json"))
		if _, statErr := os.Stat(candidate); statErr == nil || !os.IsNotExist(statErr) {
			return candidate
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(cwd, ".claude", "settings.json")
		if _, statErr := os.Stat(candidate); statErr == nil || !os.IsNotExist(statErr) {
			return candidate
		}
	}
	return filepath.Join(".claude", "settings.json")
}

func (s *Server) expectedClaudeHookURL() string {
	apiPort := strings.TrimSpace(os.Getenv("API_PORT"))
	if apiPort == "" {
		return ""
	}
	return fmt.Sprintf("http://localhost:%s/api/v1/hooks/stop", apiPort)
}

func (s *Server) getClaudeHookStatus() (bool, string, string, string) {
	settingsPath := resolveClaudeProjectSettingsPath()
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "hook_missing_file", "Claude project settings file does not exist; the Stop hook has not been registered", settingsPath
		}
		return false, "hook_read_failed", "Claude project settings could not be read", settingsPath
	}

	var doc struct {
		Hooks map[string][]struct {
			Hooks []struct {
				ID      string            `json:"_id"`
				Type    string            `json:"type"`
				URL     string            `json:"url"`
				Command string            `json:"command"`
				Timeout int               `json:"timeout"`
				Headers map[string]string `json:"headers"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return false, "hook_invalid_json", "Claude project settings file is not valid JSON", settingsPath
	}

	expectedURL := s.expectedClaudeHookURL()
	for _, group := range doc.Hooks["Stop"] {
		for _, hook := range group.Hooks {
			if hook.ID == "web-console-tts" {
				switch hook.Type {
				case "http":
					if expectedURL != "" && hook.URL != expectedURL {
						return false, "hook_stale", "Claude Stop hook exists but points to a different API URL", settingsPath
					}
					if token := strings.TrimSpace(hook.Headers["X-Hook-Token"]); token == "" || token != s.hookAuthToken {
						return false, "hook_stale", "Claude Stop hook exists but has an outdated authentication token", settingsPath
					}
				case "command":
					if !strings.Contains(hook.Command, "claude-stop-hook.sh") {
						return false, "hook_stale", "Claude Stop hook exists but uses an unexpected command", settingsPath
					}
					if expectedURL != "" && !strings.Contains(hook.Command, expectedURL) {
						return false, "hook_stale", "Claude Stop hook exists but points to a different API URL", settingsPath
					}
					if !strings.Contains(hook.Command, s.hookAuthToken) {
						return false, "hook_stale", "Claude Stop hook exists but has an outdated authentication token", settingsPath
					}
				default:
					return false, "hook_stale", "Claude Stop hook exists but uses an unsupported hook type", settingsPath
				}
				return true, "hook_registered", "Claude Stop hook is registered", settingsPath
			}
		}
	}
	return false, "hook_missing", "Claude Stop hook is not registered in project settings", settingsPath
}

func (s *Server) recordLastTTSRouting(result ConversationAppendResult) {
	s.ttsStatusMu.Lock()
	defer s.ttsStatusMu.Unlock()
	if s.lastTTSBySource == nil {
		s.lastTTSBySource = make(map[string]conversationAppendSnapshot)
	}
	cp := result
	s.lastTTSRouting = &cp
	s.lastTTSAt = time.Now()
	s.lastTTSBySource[result.Source] = conversationAppendSnapshot{
		Result: cp,
		At:     s.lastTTSAt,
	}
	log.Printf("conversation-event: source=%s code=%s appended=%v session=%s event=%s reason=%s",
		result.Source, result.Code, result.Appended, sanitizeID(result.SessionID), sanitizeID(result.EventID), result.Reason)
}

func (s *Server) getLastTTSRouting() (*ConversationAppendResult, time.Time) {
	s.ttsStatusMu.RLock()
	defer s.ttsStatusMu.RUnlock()
	if s.lastTTSRouting == nil {
		return nil, time.Time{}
	}
	cp := *s.lastTTSRouting
	return &cp, s.lastTTSAt
}

func (s *Server) getLastTTSRoutingBySource(source string) (*ConversationAppendResult, time.Time) {
	s.ttsStatusMu.RLock()
	defer s.ttsStatusMu.RUnlock()
	snapshot, ok := s.lastTTSBySource[source]
	if !ok {
		return nil, time.Time{}
	}
	cp := snapshot.Result
	return &cp, snapshot.At
}

func (s *Server) recordTTSAck(event TTSClientAck) {
	s.ttsStatusMu.Lock()
	defer s.ttsStatusMu.Unlock()
	if s.lastTTSAckBySrc == nil {
		s.lastTTSAckBySrc = make(map[string]ttsAckSnapshot)
	}
	cp := event
	s.lastTTSAck = &cp
	s.lastTTSAckAt = time.Now()
	s.lastTTSAckBySrc[event.Source] = ttsAckSnapshot{
		Result: cp,
		At:     s.lastTTSAckAt,
	}
	if s.conversations != nil {
		s.conversations.RecordPlaybackStage(event.SessionID, event.EventID, event.Stage)
	}
	log.Printf("tts-ack: source=%s stage=%s backend=%s session=%s event=%s message=%s",
		event.Source, event.Stage, event.Backend, sanitizeID(event.SessionID), sanitizeID(event.EventID), strings.TrimSpace(event.Message))
}

func (s *Server) getLastTTSAck() (*TTSClientAck, time.Time) {
	s.ttsStatusMu.RLock()
	defer s.ttsStatusMu.RUnlock()
	if s.lastTTSAck == nil {
		return nil, time.Time{}
	}
	cp := *s.lastTTSAck
	return &cp, s.lastTTSAckAt
}

func (s *Server) getLastTTSAckBySource(source string) (*TTSClientAck, time.Time) {
	s.ttsStatusMu.RLock()
	defer s.ttsStatusMu.RUnlock()
	snapshot, ok := s.lastTTSAckBySrc[source]
	if !ok {
		return nil, time.Time{}
	}
	cp := snapshot.Result
	return &cp, snapshot.At
}

func (s *Server) recordTTSPlaybackEvent(event TTSPlaybackEvent) {
	s.ttsStatusMu.Lock()
	defer s.ttsStatusMu.Unlock()
	cp := event
	s.lastTTSPlayback = &cp
	s.lastTTSPlayAt = time.Now()
	log.Printf("tts-playback: source=%s stage=%s backend=%s session=%s message=%s",
		event.Source, event.Stage, event.Backend, sanitizeID(event.SessionID), strings.TrimSpace(event.Message))
}

func (s *Server) getLastTTSPlaybackEvent() (*TTSPlaybackEvent, time.Time) {
	s.ttsStatusMu.RLock()
	defer s.ttsStatusMu.RUnlock()
	if s.lastTTSPlayback == nil {
		return nil, time.Time{}
	}
	cp := *s.lastTTSPlayback
	return &cp, s.lastTTSPlayAt
}

// loadOrCreateHookToken reads a hook auth token from file, or generates a new
// one if the file doesn't exist. The token is 32 random bytes encoded as hex.
func loadOrCreateHookToken(path string) string {
	data, err := os.ReadFile(path)
	if err == nil {
		token := strings.TrimSpace(string(data))
		if token != "" {
			return token
		}
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("hook-token: failed to generate random token: %v", err)
	}
	token := hex.EncodeToString(b)

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("hook-token: failed to create directory: %v", err)
		return token
	}
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		log.Printf("hook-token: failed to persist token: %v", err)
	}
	return token
}

func main() {
	if preflight.Run(preflight.Config{
		ScenarioName: "web-console",
	}) {
		return
	}

	dsn := resolveSQLiteDSN()
	db, err := database.Connect(context.Background(), database.Config{
		Driver:       "sqlite",
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	srv := NewServer(db)

	if err := server.Run(server.Config{
		Handler:      srv.Handler(),
		WriteTimeout: 150 * time.Second,
		Cleanup: func(ctx context.Context) error {
			srv.sweeper.Stop()
			srv.sessions.StopReattachWatchdog()
			if srv.codexTailer != nil {
				srv.codexTailer.Stop()
			}
			// Graceful session shutdown: detach from tmux sessions (preserving
			// them for recovery) and kill standard sessions.
			srv.sessions.Shutdown()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
