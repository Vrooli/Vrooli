package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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
	"web-console/internal/legacymigrate"
	"web-console/internal/metrics"
	"web-console/internal/sessionstore"
	"web-console/session"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/packages/capabilityprobe"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
	healthstatusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health_status"
	_ "modernc.org/sqlite"
	audiotoolsint "web-console/integrations/audiotools"
	intai "web-console/internal/ai"

	intsessions "web-console/internal/sessions"

	intgrouptemplates "web-console/internal/grouptemplates"
	inthandoffrules "web-console/internal/handoffrules"
	intsnippets "web-console/internal/snippets"
	intworkspace "web-console/internal/workspace"
)

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
	db                    *database.RoutedDB
	roots                 *filerouting.RoutedRoots
	router                *mux.Router
	sessions              *session.Manager
	hub                   *ConversationHub
	events                *events.Logger
	metrics               *metrics.Metrics
	backendRegistry       *backend.Registry
	sessionStore          sessionstore.Store
	aiChain               *intai.Chain
	shortcuts             ShortcutStore
	aiConfig              intai.ConfigStore
	ai                    *intai.Service
	sweeper               *session.ExpirationSweeper
	conversationRetention *conversationRetentionSweeper
	idempotency           *intsessions.IdempotencyCache // replay-safe session creation
	capabilities          *capabilities.Registry
	workspace             intworkspace.Store
	groupTemplates        intgrouptemplates.Store
	handoffRules          inthandoffrules.Store
	snippets              intsnippets.Store
	hookAuthToken         string
	credentialClient      credentialclient.Client
	integrationHubURL     string
	subscriptionResolver  *credentialclient.ConsumerSessionResolver
	entitlements          *entitlementclient.Client
	codexTailer           *CodexTailer
	claudeTailer          *ClaudeTailer
	grokTailer            *GrokTailer
	opencodeWatcher       *OpenCodeWatcher
	agentCheckpointStore  AgentTranscriptCheckpointStore

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
	// barrier on cumulative-offset stdin reconciliation.
	nextWSGen           atomic.Int64
	remoteTargetCatalog func() []targetConnection
	// resolveConsoleURL turns another scenario's slug into a link this browser
	// can open, against the origin the request arrived on. Nil means the seam
	// was never wired, and every cross-scenario link is then withheld rather
	// than guessed — which is why unit tests that build a bare Server see no
	// link and fork no CLI.
	resolveConsoleURL  func(ctx context.Context, scenarioSlug, requestHost string) (string, error)
	monetization       *monetization.Gate
	monetizationOutbox *monetization.Outbox
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
func NewServer(db *database.RoutedDB) *Server {
	ctx := context.Background()
	if err := initSchema(ctx, db.Primary()); err != nil {
		log.Fatalf("Schema initialization failed: %v", err)
	}

	// A leased test pool starts empty. Initializing it through the same
	// applier the primary uses means a test-mode request finds a fully
	// migrated schema instead of "no such table".
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		return initSchema(ctx, pool)
	})

	// Routed file roots are the storage leg of the same lease: session state,
	// uploads, and per-session agent homes resolve through RoutedRoots.Pick so
	// a test-mode request writes into the leased throwaway tree.
	roots := filerouting.New(scenarioPrimaryPaths())

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
		report := sessions.Recover(context.Background(), sessionStore, backendRegistry)
		log.Printf("recovery: recovered=%d adopted=%d awaiting_recovery=%d orphaned_tmux=%d (awaiting_recovery rows preserved for explicit recovery via /api/v1/sessions/recoverable; orphaned_tmux are live sessions we could not adopt and left running)",
			report.Recovered, report.Adopted, report.AwaitingRecovery, report.OrphanedTmux)
		sessions.StartReattachWatchdog()
	}()

	integrationHubURL := strings.TrimSpace(os.Getenv("INTEGRATION_HUB_URL"))
	if integrationHubURL == "" {
		if resolved, resolveErr := discovery.ResolveScenarioURLDefault(context.Background(), "integration-hub"); resolveErr == nil {
			integrationHubURL = resolved
		}
	}
	srv := &Server{
		roots:             roots,
		integrationHubURL: integrationHubURL,
		db:                db,
		router:            mux.NewRouter(),
		resolveConsoleURL: discovery.ResolveExternalURL,
		sessions:          sessions,
		hub:               NewConversationHub(),
		events:            eventLog,
		metrics:           metrics,
		backendRegistry:   backendRegistry,
		sessionStore:      sessionStore,
		aiChain:           intai.NewChain(intai.NewOllamaProvider(), intai.NewOpenRouterProvider(), intai.NewMeteredProvider(os.Getenv("AI_GATEWAY_URL"))),
		shortcuts:         NewSQLShortcutStore(db),
		aiConfig:          intai.NewSQLConfigStore(context.Background(), db),
		sweeper:           session.NewExpirationSweeper(sessions, eventLog, metrics),
		idempotency:       intsessions.NewIdempotencyCache(),
		workspace:         intworkspace.NewSQLStore(db),
		groupTemplates:    intgrouptemplates.NewSQLStore(db),
		handoffRules:      inthandoffrules.NewSQLStore(db),
		snippets:          intsnippets.NewSQLStore(db),
		hookAuthToken:     hookToken,
		ttsHookConfigState: hookConfigState{
			cfg:  hookCfg,
			path: hookCfgPath,
		},
		summarizeAutoPolicy:  defaultSummarizeAutoPolicy(),
		agentCheckpointStore: NewSQLAgentTranscriptCheckpointStore(db),
		conversations:        NewConversationStoreWithRepository(NewSQLConversationRepository(db)),
		filePreviewResolver:  &filepreview.Resolver{ProjectRoot: config.ResolveWorkingDir()},
		filePreviews:         filepreview.NewStore(filepreview.DefaultTTL),
		lastTTSBySource:      make(map[string]conversationAppendSnapshot),
		lastTTSAckBySrc:      make(map[string]ttsAckSnapshot),
		speechProcessor:      audioports.PassthroughSpeechTextProcessor{},
	}
	if pruner, ok := srv.conversations.repository.(conversationEventPruner); ok {
		srv.conversationRetention = newConversationRetentionSweeper(
			pruner,
			func() int { return srv.sessions.GetConfig().ConversationRetentionDays },
			func() int { return srv.sessions.GetConfig().ConversationMaxEventsPerSession },
		)
	}
	if authority, authorityErr := credentialauthority.Default(); authorityErr == nil {
		if credentials, clientErr := credentialclient.NewClient(credentialclient.ClientOptions{Authority: authority}); clientErr == nil {
			srv.credentialClient = credentials
			resolver := &credentialclient.ConsumerSessionResolver{Credentials: credentials, LPBSBaseURL: getEnvOrDefault("LPBS_URL", "http://localhost:15000")}
			srv.subscriptionResolver = resolver
			resolveToken := func(ctx context.Context, baseURL string) (string, error) {
				access, err := resolver.ResolveAt(ctx, baseURL)
				return access.AccessToken, err
			}
			srv.entitlements = entitlementclient.NewClient(resolver.LPBSBaseURL, resolveToken, &http.Client{Timeout: 15 * time.Second})
			srv.monetization = monetization.NewGate(srv.entitlements, resolver, "business_suite")
			store := monetization.NewSQLStore(db, monetization.SQLDialectSQLite)
			transport := &lpbsMonetizationTransport{baseURL: resolver.LPBSBaseURL, resolveToken: resolveToken, client: &http.Client{Timeout: 15 * time.Second}}
			srv.monetizationOutbox = monetization.NewOutbox(store, transport)
			go srv.drainMonetizationOutbox()
		}
	}
	if srv.credentialClient != nil {
		srv.aiChain = intai.NewChain(intai.NewOllamaProvider(), intai.NewOpenRouterProvider(intai.NewCredentialKeyResolver(srv.credentialClient)), intai.NewMeteredProvider(os.Getenv("AI_GATEWAY_URL")))
	}
	srv.systemContext = intai.DiscoverSystemContext(intai.DefaultLookPath)
	log.Printf("system-context: os=%s/%s shell=%s tools-found=%d",
		srv.systemContext.OS, srv.systemContext.Arch,
		srv.systemContext.Shell, intai.CountFoundTools(srv.systemContext.Tools))
	srv.ai = intai.NewService(srv.aiChain, srv.aiConfig, srv.systemContext, srv.events, &srv.metrics.AIGenerations, &srv.metrics.AISuggestions)
	srv.ai.SetActivationEmitter(func(eventType string) {
		srv.emitActivationOnce(context.Background(), eventType)
	})
	if srv.credentialClient != nil {
		srv.ai.SetKeyResolver(intai.NewCredentialKeyResolver(srv.credentialClient))
	}

	ollamaURL := getEnvOrDefault("OLLAMA_URL", "http://localhost:11434")
	openrouterKey := os.Getenv("OPENROUTER_API_KEY")
	bridgeOwnerToken, bridgeReauthToken := resolveBridgeOwnerCredentials()
	// Bridge endpoint selection is a Web Console setting. An empty value is
	// passed through to nodereach for local slug discovery.
	bridgeURL := config.Load().BridgeURL
	if warning := bridgeURLSecurityWarning(bridgeURL); warning != "" {
		log.Printf("bridge endpoint warning: %s", warning)
	}

	var audioToolsClient *audiotoolsint.Client
	audioProviderHealth := func(ctx context.Context) (*healthstatusv1.GetProviderHealthResponse, error) {
		if audioToolsClient == nil {
			return nil, fmt.Errorf("audio-tools client is not initialized")
		}
		return audioToolsClient.ProviderHealth(ctx)
	}
	checkers := newCapabilityCheckers(ollamaURL, openrouterKey, bridgeURL, bridgeOwnerToken, bridgeReauthToken, audioProviderHealth)
	srv.capabilities = capabilities.NewRegistry(capabilities.Known, checkers, 30*time.Second)
	srv.capabilities.SetLivenessCheckers(map[string]capabilities.Checker{
		"ollama": &capabilities.ResourceChecker{
			URL:    ollamaURL + "/api/tags",
			Client: &http.Client{Timeout: 5 * time.Second},
		},
		"openrouter": &capabilities.OpenRouterChecker{
			APIKey: openrouterKey,
		},
		"audio-tools": &capabilities.AudioToolsChecker{Scenario: capabilities.ScenarioChecker{Slug: "audio-tools"}, ProviderHealth: audioProviderHealth, Features: capabilities.Known[0].Features, Timeout: time.Second},
		"vrooli-bridge": &capabilities.BridgeChecker{
			BaseURL: bridgeURL, OwnerToken: bridgeOwnerToken, ReauthToken: bridgeReauthToken,
			Client: &http.Client{Timeout: 3 * time.Second}, Probe: true,
		},
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
	audioToolsClient = atClient
	if err != nil {
		// Required:false never returns an error today (lazy resolution), but keep
		// this non-fatal so a future change can't silently reintroduce a
		// boot-blocking dependency on a voice add-on.
		log.Printf("audio-tools adoption: not reachable yet (%v); voice features degraded until it is up", err)
	}
	audioCredentials := func(ctx context.Context) audiotoolsint.Credentials {
		creds := audiotoolsint.Credentials{BYOKProvider: "openrouter"}
		if srv.credentialClient != nil {
			if key, resolveErr := srv.credentialClient.Resolve(ctx, webConsoleOpenRouterIdentity, webConsoleOpenRouterField); resolveErr == nil {
				creds.BYOKKey = key
			}
		}
		if srv.subscriptionResolver != nil {
			if access, resolveErr := srv.subscriptionResolver.Resolve(ctx); resolveErr == nil {
				creds.LPBSToken = access.AccessToken
				if identity, identityErr := resolveLPBSIdentity(ctx, srv.subscriptionResolver.LPBSBaseURL, access.AccessToken); identityErr == nil {
					creds.UserIdentity = identity
				}
			}
		}
		return creds
	}
	srv.sttPort = &audioports.RemoteSpeechToText{Client: atClient, Credentials: audioCredentials}
	srv.ttsPort = &audioports.RemoteTextToSpeech{Client: atClient, Credentials: audioCredentials}
	srv.speechProcessor = &audioports.RemoteSpeechTextProcessor{Client: atClient}
	srv.summarizer = &audioports.RemoteSummarizer{Client: atClient, Credentials: audioCredentials}
	srv.streamConfigAdmin = &audioports.RemoteStreamConfigAdmin{Client: atClient, Credentials: audioCredentials}
	srv.wakeWordAdmin = &audioports.RemoteWakeWordAdmin{Client: atClient, Credentials: audioCredentials}
	srv.speakerAdmin = &audioports.RemoteSpeakerAdmin{Client: atClient, Credentials: audioCredentials}
	srv.ttsConfigAdmin = &audioports.RemoteTTSConfigAdmin{Client: atClient, Credentials: audioCredentials}
	srv.summarizeConfigAdmin = &audioports.RemoteSummarizeConfigAdmin{Client: atClient, Credentials: audioCredentials}
	srv.playbackRecorder = &audioports.RemotePlaybackEventRecorder{Client: atClient, Credentials: audioCredentials}
	srv.audioToolsResolver = atResolver
	log.Printf("audio-tools adoption: STT/TTS/processor/summarize + admin/runtime ports wired to %s", atClient.BaseURL())

	srv.sweeper.Start()
	if srv.conversationRetention != nil {
		srv.conversationRetention.Start()
	}
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

func bridgeURLSecurityWarning(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "Bridge URL is not a valid absolute URL"
	}
	if u.Scheme != "http" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || net.ParseIP(host).IsLoopback() {
		return ""
	}
	return "a non-loopback Bridge URL uses http; owner credentials and shell traffic are not encrypted (use https/wss)"
}

// newCapabilityCheckers is the production checker map kept separate from
// server construction so tests can assert that every catalogue entry has a
// real status producer. The integrations panel must never silently fall back
// to a fixture or an unregistered capability.
func newCapabilityCheckers(ollamaURL, openrouterKey, bridgeURL, bridgeOwnerToken, bridgeReauthToken string, audioClient ...func(context.Context) (*healthstatusv1.GetProviderHealthResponse, error)) map[string]capabilities.Checker {
	var providerHealth func(context.Context) (*healthstatusv1.GetProviderHealthResponse, error)
	if len(audioClient) > 0 {
		providerHealth = audioClient[0]
	}
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
		"audio-tools":                &capabilities.AudioToolsChecker{Scenario: capabilities.ScenarioChecker{Slug: "audio-tools"}, ProviderHealth: providerHealth, Features: capabilities.Known[0].Features, Timeout: time.Second},
		"session-backend-standard":   &capabilities.StaticChecker{Available: probeStandard},
		"session-backend-persistent": &capabilities.StaticChecker{Available: backend.CheckTmuxAvailable},
		"vrooli-bridge": &capabilities.BridgeChecker{
			BaseURL: bridgeURL, OwnerToken: bridgeOwnerToken, ReauthToken: bridgeReauthToken,
			Client: &http.Client{Timeout: 3 * time.Second}, Probe: true,
		},
	}
	for _, definition := range capabilityprobe.AITools {
		checkers[definition.ID] = capabilities.HostCapabilityChecker{Definition: definition}
	}
	return checkers
}

// Handler returns the router wrapped with CORS, security, and panic-recovery
// middleware.
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
	// Dev-only routed-isolation control plane: workflow-health calls
	// InstallTestPool/Heartbeat/Clear here before running a mutating BAS case,
	// which is what lets those cases run without touching the operator's real
	// database or storage roots. It mounts on the root mux because gorilla's
	// Handle returns *mux.Route and so does not satisfy devrouting.Mux.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, s.db, s.roots)
	rootMux.Handle("/", securityHeadersMiddleware(handlers.RecoveryHandler()(cors(s.router))))

	// TestModeMiddleware marks requests carrying X-Vrooli-Test-Mode: 1 so
	// RoutedDB and RoutedRoots send them to the leased test pool/roots. It is
	// a no-op pass-through in production mode.
	return apihttp.TestModeMiddleware(rootMux)
}

// securityHeadersMiddleware stamps the baseline browser protections on every
// response, including health and error responses produced outside a route
// handler. HSTS is emitted only for HTTPS (or a TLS-terminating proxy that
// forwards the original scheme) so local HTTP development is not pinned to an
// unavailable HTTPS endpoint.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "0")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
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

// httpWriteTimeout bounds every response this server writes.
//
// It is exported within the package rather than inlined at the listener
// because at least one handler has to fit inside it: a governed capability
// install relays a command to another machine and then waits for that machine
// to confirm the result, and a budget larger than this timeout would simply
// have the response cut off mid-flight with nothing to show for the wait.
// See capabilityInstallBudget, which is asserted to be smaller.
const httpWriteTimeout = 150 * time.Second

const requestIDKey contextKey = "request_id"

// requestIDMiddleware generates a short request ID and stores it in context.
// The ID is returned in the X-Request-ID response header so clients and logs
// can correlate errors to the same request.
func requestIDMiddleware(next http.Handler) http.Handler {
	var counter uint64
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := fmt.Sprintf("req-%d-%d", time.Now().UnixMilli()%(100*1000), atomic.AddUint64(&counter, 1))
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

	legacymigrate.MigrateDatabase(dbPath)

	log.Printf("SQLite database: %s", dbPath)
	dsn, err := storage.SQLiteDSNAt(dbPath, storage.SQLiteTuning{})
	if err != nil {
		log.Fatalf("build sqlite dsn: %v", err)
	}
	return dsn
}

// migrateLegacyStateFiles adapts the application storage resolver to the
// standalone legacy-migration package and runs before any state file is read.
func migrateLegacyStateFiles() {
	legacymigrate.MigrateStateFiles(func(name string) string {
		return mustResolveScenarioStoragePath(storage.ClassState, name)
	}, legacymigrate.DefaultStateFiles)
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
					// Runnability is checked before shape. A hook whose target
					// no longer exists fails on every turn while every
					// string-level check still passes, which is exactly how
					// message capture broke silently: the registered command
					// pointed at a helper script that had been deleted.
					if missing, reason := claudeHookCommandTargetMissing(hook.Command); missing {
						return false, "hook_unrunnable", "Claude Stop hook is registered but cannot run: " + reason, settingsPath
					}
					if !strings.Contains(hook.Command, "web-console") || !strings.Contains(hook.Command, "hooks dispatch") || !strings.Contains(hook.Command, "--event 'Stop'") {
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

// claudeHookCommandTargetMissing reports whether a registered hook command
// names something that cannot be executed, and why.
//
// String-matching a hook command proves only that the settings file says the
// right thing. It does not prove the hook works, and the difference is not
// academic: a registered command pointing at a deleted script fails on every
// single turn while reporting itself healthy. This check closes that gap.
//
// Interpreter invocations are unwrapped one level, because that is the form
// that hides the failure — `bash /path/to/hook.sh` resolves fine on the
// strength of `bash` alone while the script it runs is long gone.
func claudeHookCommandTargetMissing(command string) (bool, string) {
	tokens := splitShellTokens(command)
	if len(tokens) == 0 {
		return true, "the command is empty"
	}
	if missing, reason := executableTargetMissing(tokens[0]); missing {
		return true, reason
	}
	if isShellInterpreter(tokens[0]) && len(tokens) > 1 {
		script := tokens[1]
		// Skip interpreter flags; the script path is the first bare argument.
		for len(tokens) > 1 && strings.HasPrefix(script, "-") {
			tokens = tokens[1:]
			script = tokens[1]
		}
		if !strings.HasPrefix(script, "-") {
			if _, err := os.Stat(script); os.IsNotExist(err) {
				return true, "the script it runs no longer exists at " + script
			}
		}
	}
	return false, ""
}

// executableTargetMissing resolves one command word the way a shell would.
func executableTargetMissing(word string) (bool, string) {
	if word == "" {
		return true, "the command is empty"
	}
	if strings.ContainsRune(word, os.PathSeparator) {
		info, err := os.Stat(word)
		if os.IsNotExist(err) {
			return true, "no such file at " + word
		}
		if err == nil && info.Mode()&0o111 == 0 {
			return true, word + " is not executable"
		}
		return false, ""
	}
	if _, err := exec.LookPath(word); err != nil {
		return true, word + " is not on PATH"
	}
	return false, ""
}

func isShellInterpreter(word string) bool {
	switch filepath.Base(word) {
	case "sh", "bash", "zsh", "dash", "ksh":
		return true
	}
	return false
}

// splitShellTokens splits a command on whitespace while honoring the single and
// double quoting that hook registration emits around paths and tokens. It is
// intentionally not a full shell parser: it only needs to recover the words a
// registered hook command is built from.
func splitShellTokens(command string) []string {
	var tokens []string
	var current strings.Builder
	var quote rune
	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	for _, r := range command {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t':
			flush()
		default:
			current.WriteRune(r)
		}
	}
	flush()
	return tokens
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
		s.conversations.RecordPlaybackStage(context.Background(), event.SessionID, event.EventID, event.Stage)
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
	// database.Open (not Connect) returns a *RoutedDB: production requests see
	// exactly the underlying pool until a test lease is installed, after which
	// only requests carrying the test-mode marker are diverted.
	db, err := database.Open(context.Background(), database.Config{
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
		WriteTimeout: httpWriteTimeout,
		Cleanup: func(ctx context.Context) error {
			srv.sweeper.Stop()
			if srv.conversationRetention != nil {
				srv.conversationRetention.Stop()
			}
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
