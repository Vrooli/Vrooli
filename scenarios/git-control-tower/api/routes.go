package main

import (
	"git-control-tower/ssh"

	"github.com/vrooli/api-core/health"
)

// DOC: docs/reference/api-endpoints.md
// setupRoutes wires every HTTP endpoint exposed by git-control-tower.
// When adding or removing routes here, update docs/reference/api-endpoints.md
// in the same change so the catalog stays accurate.
func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	// Uses api-core/health for standardized response format
	healthHandler := health.New().
		Version("1.0.0").
		Check(health.DB(s.db), health.Critical).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/status", s.handleRepoStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/diff", s.handleDiff).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/history", s.handleRepoHistory).Methods("GET")
	// Repo registry endpoints
	s.router.HandleFunc("/api/v1/repos", s.handleRepoList).Methods("GET")
	s.router.HandleFunc("/api/v1/repos/active", s.handleRepoActive).Methods("GET")
	s.router.HandleFunc("/api/v1/repos/active", s.handleRepoSetActive).Methods("POST")
	s.router.HandleFunc("/api/v1/repos/open", s.handleRepoOpen).Methods("POST")
	s.router.HandleFunc("/api/v1/repos/clone", s.handleRepoClone).Methods("POST")
	s.router.HandleFunc("/api/v1/repos/{id}", s.handleRepoRemove).Methods("DELETE")
	s.router.HandleFunc("/api/v1/repo/stage", s.handleStage).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/unstage", s.handleUnstage).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/commit", s.handleCommit).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/precommit", s.handlePrecommitGet).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/precommit", s.handlePrecommitSave).Methods("PUT")
	s.router.HandleFunc("/api/v1/repo/precommit/run", s.handlePrecommitRun).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/approved-changes", s.handleApprovedChanges).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/approved-changes/preview", s.handleApprovedChangesPreview).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/provenance", s.handleProvenance).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/sync-status", s.handleSyncStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/discard", s.handleDiscard).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/ignore", s.handleIgnore).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/grouping-rules", s.handleGetGroupingRules).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/grouping-rules", s.handleSaveGroupingRules).Methods("PUT")
	s.router.HandleFunc("/api/v1/repo/gitignore/health", s.handleGitignoreHealth).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/gitignore/move", s.handleGitignoreMove).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/push", s.handlePush).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/pull", s.handlePull).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/upstream-action", s.handleUpstreamAction).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/branches", s.handleRepoBranches).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/branch/create", s.handleBranchCreate).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/branch/switch", s.handleBranchSwitch).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/branch/publish", s.handleBranchPublish).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/files", s.handleFiles).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/files/dir", s.handleDirectoryList).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/files/content", s.handleSaveFileContent).Methods("PUT")
	s.router.HandleFunc("/api/v1/repo/files/delete", s.handleDeletePath).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/related", s.handleRelatedFiles).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/search/content", s.handleContentSearch).Methods("GET")
	s.router.HandleFunc("/api/v1/capabilities", s.handleCapabilities).Methods("GET")
	s.router.HandleFunc("/api/v1/scenarios", s.handleScenarioList).Methods("GET")
	s.router.HandleFunc("/api/v1/scenarios/{slug}/envelope", s.handleScenarioEnvelope).Methods("GET")
	s.router.HandleFunc("/api/v1/audit", s.handleAuditQuery).Methods("GET")

	// Credentials management endpoints
	s.router.HandleFunc("/api/v1/credentials", s.handleListCredentials).Methods("GET")
	s.router.HandleFunc("/api/v1/credentials", s.handleSaveCredential).Methods("POST")
	s.router.HandleFunc("/api/v1/credentials/{id}", s.handleDeleteCredential).Methods("DELETE")
	s.router.HandleFunc("/api/v1/credentials/test", s.handleTestCredential).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/remote/url", s.handleUpdateRemoteURL).Methods("POST")

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

	// Workflow capture endpoints
	s.router.HandleFunc("/api/v1/repo/workflow-capture", s.handleWorkflowCapture).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/workflow-captures", s.handleWorkflowCaptureList).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/workflow-captures/{id}", s.handleWorkflowCaptureDetail).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/workflow-captures/{id}/video/{filename}", s.handleWorkflowCaptureVideo).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/workflow-captures/{id}", s.handleWorkflowCaptureDelete).Methods("DELETE")

	// Test-genie endpoints
	s.router.HandleFunc("/api/v1/repo/test-execution", s.handleTestExecution).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/test-executions", s.handleTestExecutionList).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/test-executions/{id}", s.handleTestExecutionDetail).Methods("GET")

	// Tidiness-manager endpoints
	s.router.HandleFunc("/api/v1/repo/tidiness-score", s.handleTidinessScore).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/tidiness-issues", s.handleTidinessIssues).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/tidiness-staleness", s.handleTidinessStaleness).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/tidiness-scan", s.handleTidinessLightScan).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/tidiness-scenario", s.handleTidinessScenarioDetail).Methods("GET")

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

	// Auditor endpoints
	s.router.HandleFunc("/api/v1/repo/rules-run", s.handleAuditorRunCheck).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/rules-job/{jobId}", s.handleAuditorJobStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/rules", s.handleAuditorRules).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/rules-fix", s.handleAuditorFix).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/rules-violations", s.handleAuditorViolations).Methods("GET")

	// Review endpoints
	s.router.HandleFunc("/api/v1/review/summary", s.handleReviewSummary).Methods("GET")
	s.router.HandleFunc("/api/v1/review/run", s.handleReviewRun).Methods("POST")
	s.router.HandleFunc("/api/v1/review/run/{jobId}", s.handleReviewJobStatus).Methods("GET")

	// SSH key management endpoints
	s.router.HandleFunc("/api/v1/ssh/keys", ssh.HandleListKeys(s.sshDeps)).Methods("GET")
	s.router.HandleFunc("/api/v1/ssh/keys/generate", ssh.HandleGenerateKey(s.sshDeps)).Methods("POST")
	s.router.HandleFunc("/api/v1/ssh/keys/public", ssh.HandleGetPublicKey(s.sshDeps)).Methods("POST")
	s.router.HandleFunc("/api/v1/ssh/keys/test", ssh.HandleTestConnection(s.sshDeps)).Methods("POST")
	s.router.HandleFunc("/api/v1/ssh/keys", ssh.HandleDeleteKey(s.sshDeps)).Methods("DELETE")
}
