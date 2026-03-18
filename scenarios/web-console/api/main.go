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

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
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
	return nil
}

// DOC: docs/concepts/ARCHITECTURE.md#system-layers
// Server wires the HTTP router, database connection, and session manager.
type Server struct {
	db              *sql.DB
	router          *mux.Router
	sessions        *SessionManager
	events          *EventLogger
	metrics         *Metrics
	aiChain         *AIProviderChain
	shortcuts       ShortcutStore
	aiConfig        AIConfigStore
	sweeper         *ExpirationSweeper
	idempotency     *idempotencyCache // replay-safe session creation
	capabilities    *CapabilityRegistry
	workspace       WorkspaceStore
	voiceConfigMu   sync.RWMutex
	voiceConfig     VoiceStreamConfig
	voiceConfigPath string
	ttsConfigMu     sync.RWMutex
	ttsConfig       TTSConfig
	ttsConfigPath   string
	hookAuthToken   string
	codexTailer     *CodexTailer
	ttsSynthesizer  TTSSynthesizer
	ttsVoiceLister  TTSVoiceLister
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

	// Resolve voice config path via api-core/storage (mutable state outside deploy dir).
	vcPath := resolveVoiceConfigPath()
	vc, err := loadVoiceConfig(vcPath)
	if err != nil {
		log.Printf("voice-config: using defaults: %v", err)
		vc = DefaultVoiceStreamConfig()
	}
	log.Printf("voice-config: loaded: flush=%dms delta=%d overlap=%d",
		vc.FlushIntervalMs, vc.MinDeltaBytes, vc.OverlapBytes)

	// Resolve TTS config path
	ttsPath := resolveTTSConfigPath()
	ttsCfg, err := loadTTSConfig(ttsPath)
	if err != nil {
		log.Printf("tts-config: using defaults: %v", err)
		ttsCfg = DefaultTTSConfig()
	}
	log.Printf("tts-config: loaded: autoEnabled=%v", ttsCfg.AutoEnabled)

	// Generate or load hook auth token for TTS hook validation
	hookToken := loadOrCreateHookToken(resolveHookTokenPath())

	events := NewEventLogger(1000)
	metrics := NewMetrics()
	sessions := NewSessionManager()
	srv := &Server{
		db:              db,
		router:          mux.NewRouter(),
		sessions:        sessions,
		events:          events,
		metrics:         metrics,
		aiChain:         NewAIProviderChain(NewOllamaProvider(), NewOpenRouterProvider()),
		shortcuts:       NewSQLShortcutStore(db),
		aiConfig:        NewSQLAIConfigStore(db),
		sweeper:         NewExpirationSweeper(sessions, events, metrics),
		idempotency:     newIdempotencyCache(),
		workspace:       NewSQLWorkspaceStore(db),
		voiceConfig:     vc,
		voiceConfigPath: vcPath,
		ttsConfig:       ttsCfg,
		ttsConfigPath:   ttsPath,
		hookAuthToken:   hookToken,
		conversations:   NewConversationStore(),
		lastTTSBySource: make(map[string]conversationAppendSnapshot),
		lastTTSAckBySrc: make(map[string]ttsAckSnapshot),
	}
	whisperURL := getEnvOrDefault("WHISPER_URL", "http://localhost:8090")
	kokoroURL := getEnvOrDefault("KOKORO_URL", "http://localhost:8880")
	ollamaURL := getEnvOrDefault("OLLAMA_URL", "http://localhost:11434")
	openrouterKey := os.Getenv("OPENROUTER_API_KEY")

	checkers := map[string]StatusChecker{
		"whisper-stt": &WhisperChecker{
			BaseURL: whisperURL,
			Client:  &http.Client{Timeout: 10 * time.Second},
		},
		"kokoro-tts": &KokoroChecker{
			BaseURL:       kokoroURL,
			Client:        &http.Client{Timeout: 10 * time.Second},
			ContainerName: "kokoro",
		},
		"ollama": &OllamaChecker{
			BaseURL: ollamaURL,
			Client:  &http.Client{Timeout: 5 * time.Second},
		},
		"openrouter": &OpenRouterChecker{
			APIKey: openrouterKey,
			Client: &http.Client{Timeout: 5 * time.Second},
		},
	}
	srv.capabilities = NewCapabilityRegistry(knownCapabilities, checkers, 30*time.Second)
	srv.ttsSynthesizer = &KokoroSynthesizer{
		BaseURL: kokoroURL,
		Client:  &http.Client{Timeout: 30 * time.Second},
	}
	srv.ttsVoiceLister = &KokoroVoiceLister{
		BaseURL: kokoroURL,
		Client:  &http.Client{Timeout: 5 * time.Second},
	}
	// Register lightweight liveness-only checkers for fast pre-recording checks.
	// These use GET-only health checks (no test transcription/synthesis).
	srv.capabilities.SetLivenessCheckers(map[string]StatusChecker{
		"whisper-stt": &ResourceChecker{
			URL:    whisperURL + "/",
			Client: &http.Client{Timeout: 5 * time.Second},
		},
		"kokoro-tts": &ResourceChecker{
			URL:    kokoroURL + "/v1/audio/voices",
			Client: &http.Client{Timeout: 5 * time.Second},
		},
		"ollama": &ResourceChecker{
			URL:    ollamaURL + "/api/tags",
			Client: &http.Client{Timeout: 5 * time.Second},
		},
		"openrouter": &OpenRouterChecker{
			APIKey: openrouterKey,
		},
	})
	// Warm capability cache so the first /capabilities request returns instantly.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.capabilities.Resolve(ctx)
		log.Println("capabilities: initial check complete")
	}()
	srv.sweeper.Start()
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

	// Session CRUD - [REQ:P0-002a] [REQ:P0-003a]
	s.router.HandleFunc("/api/v1/sessions", s.handleCreateSession).Methods("POST")
	s.router.HandleFunc("/api/v1/sessions", s.handleListSessions).Methods("GET")
	s.router.HandleFunc("/api/v1/sessions/{id}", s.handleGetSession).Methods("GET")
	s.router.HandleFunc("/api/v1/sessions/{id}", s.handleDeleteSession).Methods("DELETE")

	// Session policy - [REQ:P1-001a]
	s.router.HandleFunc("/api/v1/sessions/{id}/policy", s.handleGetPolicy).Methods("GET")
	s.router.HandleFunc("/api/v1/sessions/{id}/policy", s.handleUpdatePolicy).Methods("PUT")

	// WebSocket terminal I/O - [REQ:P0-002b]
	s.router.HandleFunc("/api/v1/sessions/{id}/ws", s.handleTerminalWS).Methods("GET")

	// Image upload for terminal path injection
	s.router.HandleFunc("/api/v1/sessions/{id}/upload", s.handleUpload).Methods("POST")

	// Workspace layout (cross-device pane ordering and tab groups)
	s.router.HandleFunc("/api/v1/workspace/layout", s.handleGetLayout).Methods("GET")
	s.router.HandleFunc("/api/v1/workspace/layout", s.handleSaveLayout).Methods("PUT")
	s.router.HandleFunc("/api/v1/sessions/{id}/conversation", s.handleGetConversationSession).Methods("GET")
	s.router.HandleFunc("/api/v1/sessions/{id}/conversation/cursor", s.handleUpdateConversationCursor).Methods("PUT")
	s.router.HandleFunc("/api/v1/workspace/panes/{session_id}", s.handleUpdatePane).Methods("PUT")
	s.router.HandleFunc("/api/v1/workspace/panes/{session_id}", s.handleDeletePane).Methods("DELETE")
	s.router.HandleFunc("/api/v1/workspace/groups", s.handleCreateGroup).Methods("POST")
	s.router.HandleFunc("/api/v1/workspace/groups/{id}", s.handleUpdateGroup).Methods("PUT")
	s.router.HandleFunc("/api/v1/workspace/groups/{id}", s.handleDeleteGroup).Methods("DELETE")

	// AI command generation - [REQ:P0-005a]
	s.router.HandleFunc("/api/v1/ai/generate", s.handleAIGenerate).Methods("POST")

	// Shortcut profiles - [REQ:P1-002a]
	s.router.HandleFunc("/api/v1/shortcuts", s.handleGetEffectiveShortcuts).Methods("GET")
	s.router.HandleFunc("/api/v1/shortcuts/profiles", s.handleListShortcutProfiles).Methods("GET")
	s.router.HandleFunc("/api/v1/shortcuts/profiles", s.handleUpsertShortcutProfile).Methods("PUT")
	s.router.HandleFunc("/api/v1/shortcuts/profiles/{id}", s.handleDeleteShortcutProfile).Methods("DELETE")

	// AI provider configuration - [REQ:P1-003a] [REQ:P1-003b]
	s.router.HandleFunc("/api/v1/ai/config", s.handleGetAIConfig).Methods("GET")
	s.router.HandleFunc("/api/v1/ai/config", s.handleUpdateAIConfig).Methods("PUT")
	s.router.HandleFunc("/api/v1/ai/health", s.handleGetAIHealth).Methods("GET")

	// Metrics endpoint - [REQ:P1-004b]
	s.router.HandleFunc("/api/v1/metrics", s.handleMetrics).Methods("GET")

	// Events endpoint - [REQ:P1-004a]
	s.router.HandleFunc("/api/v1/events", s.handleEvents).Methods("GET")

	// Voice input capabilities
	s.router.HandleFunc("/api/v1/capabilities", s.handleCapabilities).Methods("GET")
	s.router.HandleFunc("/api/v1/capabilities/liveness", s.handleCapabilitiesLiveness).Methods("GET")
	s.router.HandleFunc("/api/v1/voice/transcribe", s.handleVoiceTranscribe).Methods("POST")
	s.router.HandleFunc("/api/v1/voice/stream", s.handleVoiceStreamWS).Methods("GET")
	s.router.HandleFunc("/api/v1/voice/config", s.handleGetVoiceConfig).Methods("GET")
	s.router.HandleFunc("/api/v1/voice/config", s.handleUpdateVoiceConfig).Methods("PUT")

	// TTS hooks and config
	s.router.HandleFunc("/api/v1/hooks/stop", s.handleHookStop).Methods("POST")
	s.router.HandleFunc("/api/v1/tts/config", s.handleGetTTSConfig).Methods("GET")
	s.router.HandleFunc("/api/v1/tts/config", s.handleUpdateTTSConfig).Methods("PUT")
	s.router.HandleFunc("/api/v1/tts/status", s.handleGetTTSStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/tts/events", s.handlePostTTSEvent).Methods("POST")

	// TTS synthesis and voices (Kokoro backend)
	s.router.HandleFunc("/api/v1/tts/synthesize", s.handleTTSSynthesize).Methods("POST")
	s.router.HandleFunc("/api/v1/tts/voices", s.handleTTSVoices).Methods("GET")
}

// Handler returns the router wrapped with panic-recovery middleware so that
// an unexpected panic in a handler returns a 500 instead of crashing the server.
func (s *Server) Handler() http.Handler {
	return handlers.RecoveryHandler()(s.router)
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

// resolveVoiceConfigPath returns the voice config file path using api-core/storage.
// Falls back to a path relative to the binary if storage resolution fails.
func resolveVoiceConfigPath() string {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		log.Printf("voice-config: storage resolver failed, using fallback: %v", err)
		return fallbackVoiceConfigPath()
	}

	opts := storage.Options{ScenarioID: "web-console"}
	if _, err := storage.EnsureClassDir(resolver, opts, storage.ClassState, 0); err != nil {
		log.Printf("voice-config: ensure state dir failed, using fallback: %v", err)
		return fallbackVoiceConfigPath()
	}

	path, err := resolver.Path(opts, storage.ClassState, "voice-config.json")
	if err != nil {
		log.Printf("voice-config: resolve path failed, using fallback: %v", err)
		return fallbackVoiceConfigPath()
	}
	return path
}

func fallbackVoiceConfigPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "..", "store", "voice-config.json")
}

// resolveTTSConfigPath returns the TTS config file path using api-core/storage.
// Falls back to a path relative to the binary if storage resolution fails.
func resolveTTSConfigPath() string {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		log.Printf("tts-config: storage resolver failed, using fallback: %v", err)
		return fallbackTTSConfigPath()
	}

	opts := storage.Options{ScenarioID: "web-console"}
	if _, err := storage.EnsureClassDir(resolver, opts, storage.ClassState, 0); err != nil {
		log.Printf("tts-config: ensure state dir failed, using fallback: %v", err)
		return fallbackTTSConfigPath()
	}

	path, err := resolver.Path(opts, storage.ClassState, "tts-config.json")
	if err != nil {
		log.Printf("tts-config: resolve path failed, using fallback: %v", err)
		return fallbackTTSConfigPath()
	}
	return path
}

func fallbackTTSConfigPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "..", "store", "tts-config.json")
}

// resolveHookTokenPath returns the hook token file path using api-core/storage.
func resolveHookTokenPath() string {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		log.Printf("hook-token: storage resolver failed, using fallback: %v", err)
		return fallbackHookTokenPath()
	}

	opts := storage.Options{ScenarioID: "web-console"}
	if _, err := storage.EnsureClassDir(resolver, opts, storage.ClassState, 0); err != nil {
		log.Printf("hook-token: ensure state dir failed, using fallback: %v", err)
		return fallbackHookTokenPath()
	}

	path, err := resolver.Path(opts, storage.ClassState, "hook-token.txt")
	if err != nil {
		log.Printf("hook-token: resolve path failed, using fallback: %v", err)
		return fallbackHookTokenPath()
	}
	return path
}

func fallbackHookTokenPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "..", "store", "hook-token.txt")
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
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error {
			srv.sweeper.Stop()
			if srv.codexTailer != nil {
				srv.codexTailer.Stop()
			}
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
