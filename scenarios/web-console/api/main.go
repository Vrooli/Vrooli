package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	}
	checkers := map[string]StatusChecker{
		"whisper-stt": &WhisperChecker{
			BaseURL: "http://localhost:8090",
			Client:  &http.Client{Timeout: 10 * time.Second},
		},
	}
	srv.capabilities = NewCapabilityRegistry(knownCapabilities, checkers, 30*time.Second)
	// Register lightweight liveness-only checkers for fast pre-recording checks.
	// These use GET-only health checks (no test transcription).
	srv.capabilities.SetLivenessCheckers(map[string]StatusChecker{
		"whisper-stt": &ResourceChecker{
			URL:    "http://localhost:8090/",
			Client: &http.Client{Timeout: 5 * time.Second},
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
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
