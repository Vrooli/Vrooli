package main

import (
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/provenance"
)

// DOC: docs/reference/api-endpoints.md
// setupRoutes wires every HTTP endpoint exposed by git-control-tower.
// When adding or removing routes here, update docs/reference/api-endpoints.md
// in the same change so the catalog stays accurate.
func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	// Agent identity is verified by api-core when an identity token is
	// present. It remains attribution only; policygate separately verifies the
	// human principal from scenario-authenticator.
	s.router.Use(provenance.Middleware(provenance.CLIUtilVerifier{}))
	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	// Uses api-core/health for standardized response format
	healthHandler := health.New().
		Version("1.0.0").
		Check(health.DB(s.db.Primary()), health.Critical).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
	// Advisory drafts are exposed through AdvisoryService.Draft. The mention
	// route below remains a documented webhook_receiver exception because its
	// ingress contract is the provider delivery payload, not a typed caller.
	s.router.HandleFunc("/api/v1/advisory/mention", s.handleAdvisoryMention).Methods("POST")
	// Capability and host-neutral collaboration reads use IntegrationService.
	// Sync status is exposed through RepoService.GetSyncStatus.
	// Resolved repository groups are exposed through RepoService.GetRepoGroups.
	// Scenario status, envelope and isolation reads use ScenarioService Connect.
	// GCT audit queries use AuditService.ListEntries.
	// Source distribution reads use the typed SourceDistributionService.

	// Visual capture endpoints
	s.router.HandleFunc("/api/v1/repo/visual-capture", s.handleVisualCapture).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/visual-captures/{id}/screenshot/{filename}", s.handleVisualCaptureScreenshot).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/visual-captures/{id}/video/{filename}", s.handleVisualCaptureVideo).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/visual-captures/{id}", s.handleVisualCaptureDelete).Methods("DELETE")
	s.router.HandleFunc("/api/v1/repo/visual-capture-storage", s.handleVisualCaptureClearAll).Methods("DELETE")

	// Generic opaque evidence bytes. Metadata and authorization handles come
	// from EvidenceService; paths never cross this boundary.
	s.router.HandleFunc("/api/v1/repo/test-runs/{runId}/artifacts/{artifactId}", s.handleRunArtifact).Methods("GET")

	// Bounded blame reads use RepoService.GetBlame.

	// Agent-manager endpoints
	s.router.HandleFunc("/api/v1/agent/attachments/upload", s.handleAttachmentUpload).Methods("POST")
	s.router.HandleFunc("/api/v1/agent/run", s.handleAgentRunCreate).Methods("POST")
	s.router.HandleFunc("/api/v1/agent/runs/{id}/continue", s.handleAgentRunContinue).Methods("POST")
	s.router.HandleFunc("/api/v1/agent/runs/{id}/approve", s.handleAgentRunApprove).Methods("POST")
	s.router.HandleFunc("/api/v1/agent/runs/{id}/reject", s.handleAgentRunReject).Methods("POST")
	s.router.HandleFunc("/api/v1/agent/runs/{id}/stop", s.handleAgentRunStop).Methods("POST")
	// Staged-message operational metrics use AdvisoryService.GetStagedMessageMetrics.

	// Review reads and run admission use the typed ReviewService Connect surface;
	// no review REST routes are registered.

	// Connect-RPC handlers (proto-first surface). Worktree is the first
	// proto+Connect domain in GCT; see api/connect_wiring.go.
	s.mountConnectHandlers()
}
