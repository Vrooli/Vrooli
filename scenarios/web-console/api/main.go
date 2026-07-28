package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"web-console/internal/audioports"
	"web-console/internal/backend"
	"web-console/internal/capabilities"
	"web-console/internal/config"
	"web-console/internal/events"
	"web-console/internal/filepreview"
	"web-console/internal/metrics"
	"web-console/internal/sessionstore"
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
	audioAdminH "web-console/handlers/audio_admin"
	audioRuntimeH "web-console/handlers/audio_runtime"
	capabilitiesH "web-console/handlers/capabilities"
	conversationH "web-console/handlers/conversation"
	eventsH "web-console/handlers/events"
	filePreviewH "web-console/handlers/file_preview"
	hooksH "web-console/handlers/hooks"
	metricsH "web-console/handlers/metrics"

	sessionsH "web-console/handlers/sessions"
	settingsH "web-console/handlers/settings"
	shortcutsH "web-console/handlers/shortcuts"
	terminalH "web-console/handlers/terminal"
	workspaceH "web-console/handlers/workspace"
	audiotoolsint "web-console/integrations/audiotools"
	intai "web-console/internal/ai"

	intsessions "web-console/internal/sessions"

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

	if err := applyColumnMigrations(db); err != nil {
		return err
	}

	if err := migrateSessionsAgentTypeConstraint(db); err != nil {
		return fmt.Errorf("migration: %w", err)
	}

	if err := reconcileDefaultShortcutProfile(db); err != nil {
		return fmt.Errorf("migration: %w", err)
	}

	return nil
}

// applyColumnMigrations adds columns to existing tables. ALTER TABLE ADD COLUMN
// errors if the column already exists, so we ignore that specific error. New
// columns declare their DEFAULT so pre-existing rows are backfilled by SQLite
// as part of the ADD COLUMN — origin backfills to 'ui' because every historical
// session was opened from the web UI.
func applyColumnMigrations(db *sql.DB) error {
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
		`ALTER TABLE sessions ADD COLUMN origin TEXT NOT NULL DEFAULT 'ui'`,
		`ALTER TABLE sessions ADD COLUMN owner TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sessions ADD COLUMN display_label TEXT NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_agent ON sessions(agent_type, agent_session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_origin ON sessions(origin)`,
		`CREATE INDEX IF NOT EXISTS idx_conversation_events_session_sequence ON conversation_events(session_id, sequence)`,
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

// migrateSessionsAgentTypeConstraint relaxes the sessions.agent_type CHECK
// constraint to admit 'opencode' and 'grok'. SQLite cannot ALTER a CHECK
// constraint in place, so a DB created before these agent types carries the old
// constraint and would reject inserts for the new runtimes. The fix is the
// canonical SQLite table-rebuild, guarded so it only runs when the live
// constraint predates these values — making it a no-op on fresh DBs (which get
// the up-to-date constraint from schema.sql) and idempotent on re-run.
//
// The column list is enumerated explicitly rather than `SELECT *` so the copy
// is insensitive to physical column ordering (an incrementally ALTER-migrated
// DB can order columns differently from schema.sql).
func migrateSessionsAgentTypeConstraint(db *sql.DB) error {
	var tableSQL string
	if err := db.QueryRow(
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='sessions'`,
	).Scan(&tableSQL); err != nil {
		if err == sql.ErrNoRows {
			return nil // no sessions table yet; schema.sql will create it current
		}
		return fmt.Errorf("inspect sessions table: %w", err)
	}
	// Already admits opencode → nothing to do. Also skip tables with no
	// agent_type CHECK at all (older shapes get columns added by the ALTER
	// block above and never carried the constraint).
	if strings.Contains(tableSQL, "'opencode'") || !strings.Contains(tableSQL, "CHECK(agent_type") {
		return nil
	}

	const cols = `id, backend, shell, cols, rows, policy_mode, policy_duration,
		created_at, detached, status, agent_type, launch_command, agent_session_id,
		cwd, last_rollout_path, last_activity_at, orphaned_at, recovered_into,
		origin, owner, display_label`
	stmts := []string{
		`PRAGMA foreign_keys=off`,
		`ALTER TABLE sessions RENAME TO sessions_legacy_agentcheck`,
		`CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			backend TEXT NOT NULL DEFAULT 'standard',
			shell TEXT NOT NULL DEFAULT '/bin/bash',
			cols INTEGER NOT NULL DEFAULT 80,
			rows INTEGER NOT NULL DEFAULT 24,
			policy_mode TEXT NOT NULL DEFAULT 'never' CHECK(policy_mode IN ('never', 'preset', 'custom')),
			policy_duration TEXT,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
			detached INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'live'
				CHECK(status IN ('live','awaiting_recovery','dismissed')),
			agent_type TEXT NOT NULL DEFAULT 'none'
				CHECK(agent_type IN ('none','codex','claude','opencode','grok')),
			launch_command TEXT NOT NULL DEFAULT '',
			agent_session_id TEXT NOT NULL DEFAULT '',
			cwd TEXT NOT NULL DEFAULT '',
			last_rollout_path TEXT NOT NULL DEFAULT '',
			last_activity_at TEXT NOT NULL DEFAULT '',
			orphaned_at TEXT NOT NULL DEFAULT '',
			recovered_into TEXT NOT NULL DEFAULT '',
			origin TEXT NOT NULL DEFAULT 'ui',
			owner TEXT NOT NULL DEFAULT '',
			display_label TEXT NOT NULL DEFAULT ''
		)`,
		`INSERT INTO sessions (` + cols + `) SELECT ` + cols + ` FROM sessions_legacy_agentcheck`,
		`DROP TABLE sessions_legacy_agentcheck`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_created ON sessions(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_agent ON sessions(agent_type, agent_session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_origin ON sessions(origin)`,
		`PRAGMA foreign_keys=on`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("rebuild sessions constraint (%.40s): %w", stmt, err)
		}
	}
	log.Println("migration: relaxed sessions.agent_type CHECK to include opencode/grok")
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
// substitute via the Set* methods. audio-tools is an optional try-start
// dependency declared in .vrooli/service.json: lifecycle may bring it up, but
// terminal workspace boot must not fail when audio-tools is absent. All
// voice/TTS synthesis, summarization, voice listing, and cache logic lives in
// audio-tools. The web-console-side state is limited to: the small Claude-hook
// routing diagnostics
// (lastTTS* fields), the Claude-hook auto/backend/startMuted preference
// triple (ttsHookConfigState), and the auto-summarize policy cache
// (summarizeAutoPolicy*).
type Server struct {
	db                   *sql.DB
	router               *mux.Router
	sessions             *session.Manager
	hub                  *ConversationHub
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
	claudeTailer         *ClaudeTailer
	grokTailer           *GrokTailer
	opencodeWatcher      *OpenCodeWatcher
	agentCheckpointStore AgentTranscriptCheckpointStore

	// Audio ports — all backed by audio-tools in production.
	speechProcessor audioports.SpeechTextProcessor
	sttPort         audioports.SpeechToText
	ttsPort         audioports.TextToSpeech
	summarizer      audioports.Summarizer

	// Admin / runtime ports backing the audio_admin + audio_runtime
	// handlers that web-console exposes to its own UI. All same-origin
	// from the UI; these ports delegate to audio-tools.
	streamConfigAdmin    audioports.StreamConfigAdmin
	wakeWordAdmin        audioports.WakeWordAdmin
	speakerAdmin         audioports.SpeakerAdmin
	ttsConfigAdmin       audioports.TTSConfigAdmin
	summarizeConfigAdmin audioports.SummarizeConfigAdmin
	playbackRecorder     audioports.PlaybackEventRecorder

	// audioToolsResolver is the live audio-tools URL resolver, kept on
	// the server so consumers can re-query it (e.g. health probes).
	audioToolsResolver audiotoolsint.URLResolver

	// Hook routing diagnostics + auto-config (web-console-internal).
	ttsHookConfigState    hookConfigState
	summarizeAutoPolicyMu sync.RWMutex
	summarizeAutoPolicy   SummarizeAutoPolicy

	conversations *ConversationStore

	// File-preview subsystem: a transport-neutral resolver + an opaque,
	// session-bound, expiring preview-id store the REST blob route serves
	// bytes against. See internal/filepreview and api/file_preview_handlers.go.
	filePreviewResolver *filepreview.Resolver
	filePreviews        *filepreview.Store

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

	// Relocate any pre-runtime-home State-class artifacts (hook-token,
	// voice/TTS/wakeword configs) before anything reads them. Without this,
	// loadOrCreateHookToken silently mints a fresh canonical token whenever the
	// legacy XDG copy is still the only one on disk, breaking the X-Hook-Token
	// contract with claude-code hooks and zeroing the conversation_events
	// stream until .claude/settings.json is hand-edited. See
	// legacy_db_migration_test.go for the regression.
	migrateLegacyStateFiles()

	// Load the small Claude-hook routing preference triple (auto/backend/
	// startMuted). Voice/speed/summarize knobs live in audio-tools — fetched
	// lazily by the UI via the audio-integration module, not loaded here.
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

	// Recover surviving tmux sessions from previous run — ASYNCHRONOUSLY.
	// Reattaching many persisted tmux sessions can take minutes; doing it here
	// synchronously delayed the HTTP listener past the lifecycle health-check
	// timeout, so the scenario was killed as "unhealthy" even though every
	// session was about to come back. Recovery now runs in the background while
	// the server listens immediately; progress is published via
	// sessions.RecoveryProgress() and surfaced on Sessions.List so the UI shows
	// an honest "sessions still recovering" state. MarkRecoveryStarted is called
	// synchronously so a client that lists in the scheduling gap still sees it.
	// The reattach watchdog starts only after recovery completes, so the two do
	// not race to reattach the same session.
	sessions.MarkRecoveryStarted()
	go func() {
		report := sessions.Recover(sessionStore, backendRegistry)
		log.Printf("recovery: recovered=%d adopted=%d awaiting_recovery=%d orphaned_tmux=%d (awaiting_recovery rows preserved for explicit recovery via /api/v1/sessions/recoverable; orphaned_tmux are live sessions we could not adopt and left running)",
			report.Recovered, report.Adopted, report.AwaitingRecovery, report.OrphanedTmux)
		sessions.StartReattachWatchdog()
	}()

	srv := &Server{
		db:              db,
		router:          mux.NewRouter(),
		sessions:        sessions,
		hub:             NewConversationHub(),
		events:          eventLog,
		metrics:         metrics,
		backendRegistry: backendRegistry,
		sessionStore:    sessionStore,
		aiChain:         intai.NewChain(intai.NewOllamaProvider(), intai.NewOpenRouterProvider()),
		shortcuts:       NewSQLShortcutStore(db),
		aiConfig:        intai.NewSQLConfigStore(db),
		sweeper:         session.NewExpirationSweeper(sessions, eventLog, metrics),
		idempotency:     intsessions.NewIdempotencyCache(),
		workspace:       intworkspace.NewSQLStore(db),
		hookAuthToken:   hookToken,
		ttsHookConfigState: hookConfigState{
			cfg:  hookCfg,
			path: hookCfgPath,
		},
		summarizeAutoPolicy:  defaultSummarizeAutoPolicy(),
		codexCheckpointStore: NewSQLCodexCheckpointStore(db),
		agentCheckpointStore: NewSQLAgentTranscriptCheckpointStore(db),
		conversations:        NewConversationStoreWithRepository(NewSQLConversationRepository(db)),
		filePreviewResolver:  &filepreview.Resolver{ProjectRoot: config.ResolveWorkingDir()},
		filePreviews:         filepreview.NewStore(filepreview.DefaultTTL),
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

	// audio-tools powers voice (STT/TTS/wake-word/speaker) but NOT the core
	// terminal workspace. Resolve it lazily and wire the audioports.Remote*
	// adapters: web-console must not refuse to boot (log.Fatal) just because a
	// voice dependency is momentarily unavailable — e.g. audio-tools is still
	// starting, or temporarily failing its own build. With Required:false the
	// client resolves on demand, so voice features report unavailable via the
	// capabilities surface (and fall back to Web Speech / disabled in the UI)
	// and recover automatically once audio-tools is reachable, no restart
	// needed. The lifecycle still declares audio-tools a dependency and starts
	// it first in the normal case. AUDIO_TOOLS_URL pins an explicit URL for
	// dev/test override.
	var atResolver audiotoolsint.URLResolver
	if explicit := strings.TrimSpace(os.Getenv("AUDIO_TOOLS_URL")); explicit != "" {
		atResolver = audiotoolsint.EnvResolver{EnvVar: "AUDIO_TOOLS_URL", Default: explicit}
	} else {
		atResolver = &audiotoolsint.CachedResolver{
			Inner: audiotoolsint.ScenarioResolver{Slug: "audio-tools"},
			TTL:   30 * time.Second,
		}
	}
	atClient, err := audiotoolsint.New(atResolver, audiotoolsint.Policy{
		Required:       false,
		PerCallTimeout: 150 * time.Second,
	})
	if err != nil {
		// Required:false never returns an error today (lazy resolution), but keep
		// this non-fatal so a future change can't silently reintroduce a
		// boot-blocking dependency on a voice add-on.
		log.Printf("audio-tools adoption: not reachable yet (%v); voice features degraded until it is up", err)
	}
	srv.sttPort = &audioports.RemoteSpeechToText{Client: atClient}
	srv.ttsPort = &audioports.RemoteTextToSpeech{Client: atClient}
	srv.speechProcessor = &audioports.RemoteSpeechTextProcessor{Client: atClient}
	srv.summarizer = &audioports.RemoteSummarizer{Client: atClient}
	srv.streamConfigAdmin = &audioports.RemoteStreamConfigAdmin{Client: atClient}
	srv.wakeWordAdmin = &audioports.RemoteWakeWordAdmin{Client: atClient}
	srv.speakerAdmin = &audioports.RemoteSpeakerAdmin{Client: atClient}
	srv.ttsConfigAdmin = &audioports.RemoteTTSConfigAdmin{Client: atClient}
	srv.summarizeConfigAdmin = &audioports.RemoteSummarizeConfigAdmin{Client: atClient}
	srv.playbackRecorder = &audioports.RemotePlaybackEventRecorder{Client: atClient}
	srv.audioToolsResolver = atResolver
	log.Printf("audio-tools adoption: STT/TTS/processor/summarize + admin/runtime ports wired to %s", atClient.BaseURL())

	srv.sweeper.Start()
	// Fan session lifecycle events (created/deleted/terminated) from the event
	// logger onto the SSE hub so externally created/destroyed sessions appear
	// and disappear in every connected browser's sidebar live.
	srv.startSessionLifecycleBridge()
	// sessions.StartReattachWatchdog() runs after async recovery completes (see
	// the recovery goroutine above) so the watchdog and recovery never race to
	// reattach the same session.
	srv.setupRoutes()

	// Start Codex rollout tailer for auto-TTS.
	srv.codexTailer = NewCodexTailer(srv)
	srv.codexTailer.Start()
	log.Println("codex-tailer: started watching for per-session rollout files")

	// Claude hooks deliver promptly when healthy; this cursor-backed reader is
	// the durable fallback for resumed sessions and hook regressions.
	srv.claudeTailer = NewClaudeTailer(srv)
	srv.claudeTailer.Start()
	log.Println("claude-tailer: started watching Claude transcripts")

	// Start Grok transcript tailer (per-session GROK_HOME updates.jsonl).
	srv.grokTailer = NewGrokTailer(srv)
	srv.grokTailer.Start()
	log.Println("grok-tailer: started watching for per-session grok transcripts")

	// Start the OpenCode watcher: owns a loopback `opencode serve` instance,
	// subscribes to its event stream, and reconciles transcripts by directory.
	// Best-effort — a missing/unstartable opencode binary must not fail boot.
	srv.opencodeWatcher = NewOpenCodeWatcher(srv)
	srv.opencodeWatcher.Start()
	log.Println("opencode-watcher: started")

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
		AgentCheckpoints: s.agentCheckpointStore,
		Workspace:        s.workspace,
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

	// Conversation domain (history, cursor, summarize) — Connect-RPC.
	conversationH.Module(newConversationAdapter(s), nil).Mount(s.router)

	// File-preview domain — Connect-RPC FilePreviewService (Resolve,
	// GetTextContent) plus a REST blob/range exception for opaque byte
	// streaming into native browser media elements. The Connect handler is
	// mounted via Module; the blob route is registered directly below because
	// it needs the Server's session lookup + preview store.
	filePreviewH.Module(newFilePreviewAdapter(s), nil).Mount(s.router)
	s.router.HandleFunc("/api/v1/sessions/{id}/file-previews/{previewId}/blob", s.handleFilePreviewBlob).Methods("GET", "HEAD")

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

	// Audio admin / runtime — UI talks same-origin to web-console; the
	// handlers delegate to internal/audioports.* which proxies to
	// audio-tools server-side. The browser never sees audio-tools' host.
	audioAdminH.Module(audioAdminH.Deps{
		StreamConfig:    s.streamConfigAdmin,
		WakeWord:        s.wakeWordAdmin,
		Speaker:         s.speakerAdmin,
		TTSConfig:       s.ttsConfigAdmin,
		SummarizeConfig: s.summarizeConfigAdmin,
	}).Mount(s.router)
	audioRuntimeH.Module(audioRuntimeH.Deps{
		STT:      s.sttPort,
		TTS:      s.ttsPort,
		Playback: s.playbackRecorder,
		Summ:     s.summarizer,
	}).Mount(s.router)

	// Voice streaming WebSocket proxy. Browser opens ws(s)://<web-console>/api/v1/voice/stream;
	// web-console proxies to audio-tools server-side. interoperability rule:
	// the UI never sees audio-tools' host.
	if s.audioToolsResolver != nil {
		s.router.Handle("/api/v1/voice/stream", newVoiceStreamProxy(s.audioToolsResolver))
	}

	// Global conversation event channel (Server-Sent Events). The UI opens ONE
	// stream for ALL sessions; conversation events no longer ride the
	// per-session terminal WebSocket. Raw HTTP handler (not Connect-RPC) because
	// SSE is a long-lived streaming response the browser EventSource consumes.
	s.router.HandleFunc("/api/v1/events/stream", s.handleEventStream).Methods("GET")

	// Hooks — REST webhook receivers (Claude Code CLI dictates wire shape).
	hooksH.Module(hooksH.Deps{
		Stop:         s.handleHookStop,
		PromptSubmit: s.handleHookPromptSubmit,
	}).Mount(s.router)

	// TTS hook routing diagnostics + auto/backend/startMuted preference triple.
	// REST exception per RESTReasonHostHookGlue — this is web-console-internal
	// Claude-hook glue and never crosses scenario boundaries. All audio
	// synthesis flows through Connect against audio-tools (via audio-integration).
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

	migrateLegacyDB(dbPath)

	log.Printf("SQLite database: %s", dbPath)
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		dbPath,
	)
}

// migrateLegacyDB relocates a pre-runtime-home web-console database to the
// canonical path. Before the ~/.vrooli storage migration the DB lived under the
// XDG data dir (~/.local/share/vrooli/web-console/web-console.db). When the
// resolver started pointing at ~/.vrooli the DB was NOT moved, so a rebuilt
// binary opened a brand-new empty DB and recovery treated every live session as
// an orphan (the 2026-05-27 data-loss incident). This one-time, idempotent copy
// closes that gap: when the canonical DB is absent but a legacy copy exists,
// bring it (and any WAL sidecars) across before the DB is opened.
func migrateLegacyDB(dbPath string) {
	if _, err := os.Stat(dbPath); err == nil {
		return // canonical DB already present; nothing to migrate
	} else if !errors.Is(err, os.ErrNotExist) {
		return // unexpected stat error: leave handling to the DB opener
	}

	for _, legacy := range legacyDBCandidates() {
		if legacy == dbPath {
			continue
		}
		if _, err := os.Stat(legacy); err != nil {
			continue
		}
		// Copy the DB plus any WAL sidecars so uncommitted data is preserved;
		// SQLite replays the WAL on first open.
		if err := copyFileIfExists(legacy, dbPath); err != nil {
			log.Printf("legacy-db migration: copy %s: %v", legacy, err)
			continue
		}
		for _, suffix := range []string{"-wal", "-shm"} {
			if err := copyFileIfExists(legacy+suffix, dbPath+suffix); err != nil {
				log.Printf("legacy-db migration: copy %s: %v", legacy+suffix, err)
			}
		}
		log.Printf("legacy-db migration: relocated %s -> %s", legacy, dbPath)
		return
	}
}

// legacyDBCandidates lists pre-migration database locations, most-specific first.
func legacyDBCandidates() []string {
	var out []string
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		out = append(out, filepath.Join(xdg, "vrooli", "web-console", "web-console.db"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, filepath.Join(home, ".local", "share", "vrooli", "web-console", "web-console.db"))
	}
	return out
}

// legacyStateFiles is the manifest of State-class artifacts the ~/.vrooli
// storage migration must bring across. hook-token.txt is the load-bearing one
// — a fresh canonical token breaks the X-Hook-Token contract with claude-code
// hooks and silently zeros the conversation_events stream. The remaining
// configs default cleanly when absent, so their loss is invisible to the
// failure mode but observable as the user "losing" their voice/TTS preferences;
// migrating them at the same time is the cheap, complete fix.
var legacyStateFiles = []string{
	"hook-token.txt",
	"tts-config.json",
	"tts-hook-config.json",
	"tts-summarize-config.json",
	"voice-config.json",
	"speaker-verification-config.json",
	"wakeword-template.json",
}

// migrateLegacyStateFiles relocates the State-class web-console artifacts the
// 2026-05-27 ~/.vrooli storage migration left behind. It must run BEFORE any
// State-class file is read — most importantly loadOrCreateHookToken, which
// otherwise mints a fresh token on first miss and locks claude-code hooks out.
// Idempotent per file: only migrates when the canonical destination is absent.
func migrateLegacyStateFiles() {
	for _, name := range legacyStateFiles {
		canonical := mustResolveScenarioStoragePath(storage.ClassState, name)
		migrateLegacyStateFile(canonical, name)
	}
}

// migrateLegacyStateFile is the per-file primitive behind migrateLegacyStateFiles.
// Split out so tests can exercise a single artifact under a temp HOME without
// depending on the full scenario-storage resolver.
func migrateLegacyStateFile(canonicalPath, name string) {
	if _, err := os.Stat(canonicalPath); err == nil {
		return
	} else if !errors.Is(err, os.ErrNotExist) {
		return
	}
	for _, legacy := range legacyStateCandidates(name) {
		if legacy == canonicalPath {
			continue
		}
		info, err := os.Stat(legacy)
		if err != nil {
			continue
		}
		if err := copyFileWithMode(legacy, canonicalPath, info.Mode().Perm()); err != nil {
			log.Printf("legacy-state migration: copy %s: %v", legacy, err)
			continue
		}
		log.Printf("legacy-state migration: relocated %s -> %s", legacy, canonicalPath)
		return
	}
}

// legacyStateCandidates lists pre-migration State-class locations, most-specific
// first. Unlike legacyDBCandidates, these live under XDG_STATE_HOME
// (~/.local/state), so a separate resolver is required.
func legacyStateCandidates(name string) []string {
	var out []string
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		out = append(out, filepath.Join(xdg, "vrooli", "web-console", name))
	}
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, filepath.Join(home, ".local", "state", "vrooli", "web-console", name))
	}
	return out
}

// copyFileWithMode copies src to dst with an explicit mode, creating dst's
// parent directory. Distinct from copyFileIfExists because hook-token.txt is
// sensitive and must land 0o600, not the default 0o644.
func copyFileWithMode(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// copyFileIfExists copies src to dst, creating dst's parent dir. A missing src
// is a no-op (returns nil) so optional WAL sidecars can be attempted blindly.
func copyFileIfExists(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer in.Close() //nolint:errcheck

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
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
			if srv.claudeTailer != nil {
				srv.claudeTailer.Stop()
			}
			if srv.grokTailer != nil {
				srv.grokTailer.Stop()
			}
			if srv.opencodeWatcher != nil {
				srv.opencodeWatcher.Stop()
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
