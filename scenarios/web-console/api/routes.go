package main

import (
	"net/http"
	"time"

	aiH "web-console/handlers/ai"
	audioAdminH "web-console/handlers/audio_admin"
	audioRuntimeH "web-console/handlers/audio_runtime"
	capabilitiesH "web-console/handlers/capabilities"
	conversationH "web-console/handlers/conversation"
	eventsH "web-console/handlers/events"
	filePreviewH "web-console/handlers/file_preview"
	groupTemplatesH "web-console/handlers/grouptemplates"
	handoffRulesH "web-console/handlers/handoffrules"
	hooksH "web-console/handlers/hooks"
	metricsH "web-console/handlers/metrics"
	sessionsH "web-console/handlers/sessions"
	settingsH "web-console/handlers/settings"
	shortcutsH "web-console/handlers/shortcuts"
	snippetsH "web-console/handlers/snippets"
	terminalH "web-console/handlers/terminal"
	workspaceH "web-console/handlers/workspace"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/health"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

// setupRoutes is the transport assembly point for the API. Domain handlers
// own their request semantics; this file owns only mounting and the small set
// of deliberate REST exceptions (WebSocket, SSE, uploads, and blob streaming).
func (s *Server) setupRoutes() {
	s.router.Use(requestIDMiddleware)
	s.router.Use(loggingMiddleware)
	// Cross-scenario links are resolved against the origin the browser used, so
	// the host has to survive the hop into a Connect handler. Go moves the Host
	// header onto Request.Host, where connect.Request.Header() cannot see it.
	s.router.Use(discovery.ExternalHostMiddleware)

	healthBuilder := health.New().Version("1.0.0")
	if s.db != nil {
		healthBuilder = healthBuilder.Check(health.DB(s.db.Primary()), health.Critical)
	}
	healthHandler := healthBuilder.Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
	s.registerOwnerCleanupRoutes()
	voiceGate := monetizationGate{gate: s.monetization, outbox: s.monetizationOutbox}
	s.router.Handle("/api/v1/monetization/voice", monetization.InjectEntitlement(http.HandlerFunc(voiceGate.voiceSynthesis))).Methods(http.MethodPost)

	sessionsH.Module(&sessionsH.Adapter{
		Manager:             s.sessions,
		Store:               s.sessionStore,
		Idempotency:         s.idempotency,
		Events:              s.events,
		Metrics:             s.metrics,
		Conversations:       s.conversations,
		CodexCheckpoints:    s.agentCheckpointStore,
		AgentCheckpoints:    s.agentCheckpointStore,
		Workspace:           s.workspace,
		CopyCodexHome:       copyCodexHome,
		RemoveAgentHomes:    removeSessionAgentHomes,
		AgentHistoryPresent: archivedAgentHistoryPresent,
		RetentionPolicy: func() sessionsH.ArchiveRetentionPolicy {
			cfg := s.sessions.GetConfig()
			return sessionsH.ArchiveRetentionPolicy{
				MessageLessAge: time.Duration(cfg.ArchiveMessageLessAgeDays) * 24 * time.Hour,
				AgentHomeAge:   time.Duration(cfg.ArchiveAgentHomeAgeDays) * 24 * time.Hour,
				MaxBytes:       cfg.ArchiveMaxBytes,
			}
		},
		AgentHistorySize:  archivedAgentHistorySize,
		PruneAgentHistory: pruneArchivedAgentHistory,
		Remote:            s,
	}, nil).Mount(s.router)
	s.mountTargetCatalog()
	s.mountMachines()
	s.mountDevices()

	terminalH.Module(&terminalH.Adapter{Manager: s.sessions}, terminalH.LegacyDeps{
		Upload: s.handleUpload,
		WS:     s.handleTerminalWS,
	}, nil).Mount(s.router)

	workspaceH.Module(&workspaceH.Adapter{Store: s.workspace, Events: s.events}, nil).Mount(s.router)
	// Templates and rules are plain configuration: the domain Store already
	// satisfies the handler's Service seam, so there is no adapter to add.
	groupTemplatesH.Module(s.groupTemplates, nil).Mount(s.router)
	handoffRulesH.Module(s.handoffRules, nil).Mount(s.router)
	snippetsH.Module(s.snippets, nil).Mount(s.router)
	conversationH.Module(newConversationAdapter(s), nil).Mount(s.router)
	filePreviewH.Module(newFilePreviewAdapter(s), nil).Mount(s.router)
	s.router.HandleFunc("/api/v1/sessions/{id}/file-previews/{previewId}/blob", s.handleFilePreviewBlob).Methods("GET", "HEAD")
	settingsH.Module(newSettingsAdapter(s), nil).Mount(s.router)
	shortcutsH.Module(newShortcutsAdapter(s), nil).Mount(s.router)
	aiH.Module(&aiH.Adapter{Backend: s.ai}, nil).Mount(s.router)
	metricsH.Module(&metricsH.Adapter{Metrics: s.metrics}, nil).Mount(s.router)
	eventsH.Module(&eventsH.Adapter{Logger: s.events}, nil).Mount(s.router)
	capabilitiesH.Module(&capabilitiesH.Adapter{
		Registry:        s.capabilities,
		BackendRegistry: s.backendRegistry,
		DefaultBackend:  func() string { return string(s.sessions.GetConfig().DefaultBackend) },
		RemoteInstall:   s.installCapabilityRemote,
		ConfirmInstall:  s.confirmCapabilityInstall,
	}, nil).Mount(s.router)

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

	if s.audioToolsResolver != nil {
		s.router.Handle("/api/v1/voice/stream", newVoiceStreamProxy(s.audioToolsResolver))
	}
	s.router.HandleFunc("/api/v1/events/stream", s.handleEventStream).Methods("GET")
	hooksH.Module(hooksH.Deps{Stop: s.handleHookStop, PromptSubmit: s.handleHookPromptSubmit}).Mount(s.router)
	s.registerTTSHookRoutes()
}
