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
	s.router.HandleFunc("/api/v1/collaboration/status", s.handleCollaborationStatus).Methods("GET")
	// Sync status is exposed through RepoService.GetSyncStatus.
	// Resolved repository groups are exposed through RepoService.GetRepoGroups.
	s.router.HandleFunc("/api/v1/capabilities", s.handleCapabilities).Methods("GET")
	s.router.HandleFunc("/api/v1/scenarios", s.handleScenarioList).Methods("GET")
	s.router.HandleFunc("/api/v1/scenarios/{slug}/envelope", s.handleScenarioEnvelope).Methods("GET")
	s.router.HandleFunc("/api/v1/scenarios/{slug}/isolation", s.handleScenarioIsolation).Methods("GET")
	s.router.HandleFunc("/api/v1/audit", s.handleAuditQuery).Methods("GET")
	// Source distributions are a read-only projection of the authoritative
	// scenario-to-repository ramp. No GCT route assembles or publishes.
	s.router.HandleFunc("/api/v1/source-distributions", s.sourceDistributionList).Methods("GET")
	s.router.HandleFunc("/api/v1/source-distributions/{id}", s.sourceDistributionDetail).Methods("GET")

	// Visual capture endpoints
	s.router.HandleFunc("/api/v1/repo/visual-capture", s.handleVisualCapture).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/visual-captures", s.handleVisualCaptureList).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/visual-captures/{id}", s.handleVisualCaptureDetail).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/visual-captures/{id}/screenshot/{filename}/path", s.handleVisualCaptureScreenshotPath).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/visual-captures/{id}/screenshot/{filename}", s.handleVisualCaptureScreenshot).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/visual-captures/{id}/video/{filename}", s.handleVisualCaptureVideo).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/visual-capture-storage", s.handleVisualCaptureStorageStats).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/visual-captures/{id}", s.handleVisualCaptureDelete).Methods("DELETE")
	s.router.HandleFunc("/api/v1/repo/visual-capture-storage", s.handleVisualCaptureClearAll).Methods("DELETE")

	// Generic opaque evidence bytes. Metadata and authorization handles come
	// from EvidenceService; paths never cross this boundary.
	s.router.HandleFunc("/api/v1/repo/test-runs/{runId}/artifacts/{artifactId}", s.handleRunArtifact).Methods("GET")

	// Tidiness-manager endpoints
	s.router.HandleFunc("/api/v1/repo/tidiness-score", s.handleTidinessScore).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/tidiness-issues", s.handleTidinessIssues).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/tidiness-staleness", s.handleTidinessStaleness).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/tidiness-scan", s.handleTidinessLightScan).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/tidiness-scenario", s.handleTidinessScenarioDetail).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/blame", s.handleBlame).Methods("GET")

	// Agent-manager endpoints
	s.router.HandleFunc("/api/v1/agent/attachments/upload", s.handleAttachmentUpload).Methods("POST")
	s.router.HandleFunc("/api/v1/agent/profiles", s.handleAgentProfiles).Methods("GET")
	s.router.HandleFunc("/api/v1/agent/run", s.handleAgentRunCreate).Methods("POST")
	s.router.HandleFunc("/api/v1/agent/runs", s.handleAgentRunList).Methods("GET")
	s.router.HandleFunc("/api/v1/agent/runs/{id}", s.handleAgentRunDetail).Methods("GET")
	s.router.HandleFunc("/api/v1/agent/runs/{id}/events", s.handleAgentRunEvents).Methods("GET")
	s.router.HandleFunc("/api/v1/agent/runs/{id}/diff", s.handleAgentRunDiff).Methods("GET")
	s.router.HandleFunc("/api/v1/agent/runs/{id}/continue", s.handleAgentRunContinue).Methods("POST")
	s.router.HandleFunc("/api/v1/agent/runs/{id}/approve", s.handleAgentRunApprove).Methods("POST")
	s.router.HandleFunc("/api/v1/agent/runs/{id}/reject", s.handleAgentRunReject).Methods("POST")
	s.router.HandleFunc("/api/v1/agent/runs/{id}/stop", s.handleAgentRunStop).Methods("POST")

	// Review read endpoints remain REST during the typed migration. Review run
	// admission is owned by ReviewService.Start.
	s.router.HandleFunc("/api/v1/review/summary", s.handleReviewSummary).Methods("GET")
	// POST /api/v1/review/run retired; use ReviewService.Start.
	s.router.HandleFunc("/api/v1/review/run/{jobId}", s.handleReviewJobStatus).Methods("GET")

	// Connect-RPC handlers (proto-first surface). Worktree is the first
	// proto+Connect domain in GCT; see api/connect_wiring.go.
	s.mountConnectHandlers()
}
