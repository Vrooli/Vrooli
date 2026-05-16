/**
 * API Endpoints - Path constants for API routes
 *
 * This module defines the API endpoint paths as a separate concern from the
 * HTTP client implementation. This separation allows:
 * - Endpoint paths to be used in testing without importing the HTTP client
 * - Clear documentation of the API contract
 * - Easy updates when API routes change
 */

/**
 * API endpoint paths.
 * These are declared as string literals to support completeness scoring detection.
 */
export const API_ENDPOINTS = {
  backlog: "/backlog",
  backlogItem: (kind: string, name: string) => `/backlog/${kind}/${name}`,
  backlogFiles: (kind: string, name: string) => `/backlog/${kind}/${name}/files`,
  backlogFileOperations: (kind: string, name: string) => `/backlog/${kind}/${name}/files`,
  backlogFileContent: (kind: string, name: string, filePath: string) =>
    `/backlog/${kind}/${name}/files/${filePath}`,
  backlogQueue: (kind: string, name: string) => `/backlog/${kind}/${name}/queue`,
  backlogRetry: (kind: string, name: string) => `/backlog/${kind}/${name}/retry`,
  backlogResearch: (kind: string, name: string) => `/backlog/${kind}/${name}/research`,
  backlogWorkshopSave: (kind: string, name: string) => `/backlog/${kind}/${name}/workshop/save`,
  backlogWorkshopDeleteRound: (kind: string, name: string) => `/backlog/${kind}/${name}/workshop/round`,
  backlogWorkshopReset: (kind: string, name: string) => `/backlog/${kind}/${name}/workshop/reset`,
  backlogReWorkshop: (kind: string, name: string) => `/backlog/${kind}/${name}/re-workshop`,
  backlogWorkshopCancelPendingAdvance: (kind: string, name: string) => `/backlog/${kind}/${name}/workshop/pending-advance`,
  backlogArchiveTargets: (kind: string, name: string) => `/backlog/${kind}/${name}/archive/targets`,
  backlogArchiveTarget: (kind: string, name: string, targetId: string) => `/backlog/${kind}/${name}/archive/targets/${targetId}`,
  backlogArchiveRequirements: (kind: string, name: string) => `/backlog/${kind}/${name}/archive/requirements`,
  backlogArchiveRequirementsModule: (kind: string, name: string, moduleId: string) => `/backlog/${kind}/${name}/archive/requirements/${moduleId}`,
  backlogArchiveRequirementsModuleMeta: (kind: string, name: string, moduleId: string) => `/backlog/${kind}/${name}/archive/requirements/${moduleId}/meta`,
  backlogArchiveReview: (kind: string, name: string) => `/backlog/${kind}/${name}/archive/review`,
  backlogArchiveItem: (kind: string, name: string) => `/backlog/${kind}/${name}/archive-item`,
  backlogExport: "/backlog/export",
  backlogImport: "/backlog/import",
  backlogValidateGlobs: "/backlog/validate-globs",
  backlogSummary: "/backlog/summary",
  backlogFeedbackSummary: "/backlog/feedback-summary",
  backlogMaturitySummary: "/backlog/maturity-summary",
  backlogPendingQuestions: "/backlog/pending-questions",
  scenarios: "/scenarios",
  scenarioByName: (name: string) => `/scenarios/${name}`,
  scenarioFiles: (name: string) => `/scenarios/${name}/files`,
  scenarioSpecSyncArchive: (name: string) => `/scenarios/${name}/spec-sync-archive`,
  scenarioStart: (name: string) => `/scenarios/${name}/start`,
  scenarioStop: (name: string) => `/scenarios/${name}/stop`,
  scenarioRestart: (name: string) => `/scenarios/${name}/restart`,
  scenarioContext: (name: string) => `/scenarios/${name}/context`,
  agentManagerStatus: "/agent-manager/status",
  agentManagerRun: (runId: string) => `/agent-manager/runs/${runId}`,
  agentManagerStopRun: (runId: string) => `/agent-manager/runs/${runId}/stop`,
  settings: "/settings",
  execution: "/execution",
  executionById: (executionId: string) => `/execution/${executionId}`,
  executionPromptTrace: (executionId: string) => `/execution/${executionId}/prompt-trace`,
  executionStart: (executionId: string) => `/execution/${executionId}/start`,
  executionCancel: (executionId: string) => `/execution/${executionId}/cancel`,
  executionRetry: (executionId: string) => `/execution/${executionId}/retry`,
  executionFollowUp: (executionId: string) => `/execution/${executionId}/follow-up`,
  executionTriggerReview: (executionId: string) => `/execution/${executionId}/trigger-review`,
  agentActivities: "/agent-activities",
  agentActivityById: (activityId: string) => `/agent-activities/${activityId}`,
  agentSessions: "/agent-sessions",
  agentSessionById: (sessionId: string) => `/agent-sessions/${sessionId}`,
  agentSessionAttachments: (sessionId: string) => `/agent-sessions/${sessionId}/attachments`,
  agentSessionAttachment: (sessionId: string, attachmentId: string) =>
    `/agent-sessions/${sessionId}/attachments/${attachmentId}`,
  agentSessionStart: (sessionId: string) => `/agent-sessions/${sessionId}/start`,
  agentSessionStartupBrief: (sessionId: string) => `/agent-sessions/${sessionId}/startup-brief`,
  agentSessionContinue: (sessionId: string) => `/agent-sessions/${sessionId}/continue`,
  agentSessionEvents: (sessionId: string) => `/agent-sessions/${sessionId}/events`,
  agentSessionRefresh: (sessionId: string) => `/agent-sessions/${sessionId}/refresh`,
  agentSessionCancel: (sessionId: string) => `/agent-sessions/${sessionId}/cancel`,
  agentSessionApplyProposal: (sessionId: string, proposalId: string) =>
    `/agent-sessions/${sessionId}/proposals/${proposalId}/apply`,
  agentSessionArtifacts: (sessionId: string) => `/agent-sessions/${sessionId}/artifacts`,
  agentSessionArtifactsByEntity: "/artifacts/by-entity",
  gctStatus: "/gct/status",
  promptsCatalog: "/prompts/catalog",
  promptSkills: "/prompts/skills",
  promptSkillById: (skillId: string) => `/prompts/skills/${skillId}`,
  promptSkillVersions: (skillId: string) => `/prompts/skills/${skillId}/versions`,
  promptSkillRevert: (skillId: string, version: number) => `/prompts/skills/${skillId}/revert/${version}`,
  promptsPreview: "/prompts/preview",
  promptsSimulate: "/prompts/simulate",
  promptExperimentResults: (experimentId: string) => `/prompts/experiments/${experimentId}/results`,
  captures: "/captures",
  captureById: (id: string) => `/captures/${id}`,
  captureClassify: (id: string) => `/captures/${id}/classify`,
  captureCreateItem: (id: string) => `/captures/${id}/create-item`,
  initiatives: "/initiatives",
  initiativeByName: (name: string) => `/initiatives/${name}`,
  initiativeArchiveItem: (name: string) => `/initiatives/${name}/archive-item`,
  initiativeItems: (name: string) => `/initiatives/${name}/items`,
  initiativeFiles: (name: string) => `/initiatives/${name}/files`,
  initiativeFileOperations: (name: string) => `/initiatives/${name}/files`,
  initiativeFileContent: (name: string, filePath: string) =>
    `/initiatives/${name}/files/${filePath}`,
  // Initiative feedback — user feedback rounds on an initiative.
  // Multi-turn agent dialogue that produces structured mutation proposals
  // the user can selectively accept, reject, revise, or dismiss.
  initiativeFeedback: (name: string) => `/initiatives/${name}/feedback`,
  initiativeFeedbackRound: (name: string, round: number) =>
    `/initiatives/${name}/feedback/${round}`,
  initiativeFeedbackContinue: (name: string, round: number) =>
    `/initiatives/${name}/feedback/${round}/continue`,
  initiativeFeedbackDecide: (name: string, round: number) =>
    `/initiatives/${name}/feedback/${round}/decide`,
  initiativeFeedbackDismiss: (name: string, round: number) =>
    `/initiatives/${name}/feedback/${round}/dismiss`,
  initiativeFeedbackCancel: (name: string, round: number) =>
    `/initiatives/${name}/feedback/${round}/cancel`,
  initiativeFeedbackAttachment: (name: string, round: number, attachmentId: string) =>
    `/initiatives/${name}/feedback/${round}/attachments/${attachmentId}`,
  initiativeFeedbackLock: (name: string) => `/initiatives/${name}/feedback/lock`,
  // Initiative review — final verdict after all member items reach terminal.
  initiativeReviewRounds: (name: string) => `/initiatives/${name}/review`,
  initiativeReviewRound: (name: string, round: number) =>
    `/initiatives/${name}/review/${round}`,
  initiativeReviewTrigger: (name: string) => `/initiatives/${name}/review/trigger`,
  initiativeReviewDecide: (name: string) => `/initiatives/${name}/review/decide`,
  initiativeReviewDecisions: (name: string) => `/initiatives/${name}/review/decisions`,
  initiativeOperatingModeWorkspace: (name: string) =>
    `/initiatives/${name}/operating-mode/workspace`,
  initiativeOperatingModeSwitch: (name: string) =>
    `/initiatives/${name}/operating-mode/switch`,
  initiativeOperatingModeStartPhase: (name: string, phase: string) =>
    `/initiatives/${name}/operating-mode/phases/${phase}/start`,
  initiativeOperatingModeRefreshRound: (name: string, round: number, mode: string) =>
    `/initiatives/${name}/operating-mode/rounds/${round}/refresh?mode=${encodeURIComponent(mode)}`,
  initiativeOperatingModeCancelRound: (name: string, round: number, mode: string) =>
    `/initiatives/${name}/operating-mode/rounds/${round}/cancel?mode=${encodeURIComponent(mode)}`,
  initiativeOperatingModeCompleteItems: (name: string, round: number, mode: string) =>
    `/initiatives/${name}/operating-mode/rounds/${round}/complete-items?mode=${encodeURIComponent(mode)}`,
  initiativeOperatingModeApplyBacklogSync: (name: string, round: number, mode: string) =>
    `/initiatives/${name}/operating-mode/rounds/${round}/apply-backlog-sync?mode=${encodeURIComponent(mode)}`,
  operatingModes: "/operating-modes",
  operatingMode: (mode: string) => `/operating-modes/${encodeURIComponent(mode)}`,
  graph: "/graph",
  overview: "/overview",
  stats: "/stats",
  health: "/health",
  operations: "/operations",
  operationsBrief: "/operations/brief",
  operationsBulkStop: "/operations/bulk-stop",
  // Review evidence endpoints
  reviewRounds: (kind: string, name: string) => `/backlog/${kind}/${name}/review`,
  reviewCapture: (kind: string, name: string, capturePath: string) =>
    `/backlog/${kind}/${name}/review/captures/${capturePath}`,
  reviewVerify: (kind: string, name: string, round: number, evidenceId: string) =>
    `/backlog/${kind}/${name}/review/${round}/verify/${evidenceId}`,
  reviewRequest: (kind: string, name: string, round: number) =>
    `/backlog/${kind}/${name}/review/${round}/request`,
  reviewContinueRequest: (kind: string, name: string, round: number, threadId: string) =>
    `/backlog/${kind}/${name}/review/${round}/request/${threadId}`,
  reviewDismissRequest: (kind: string, name: string, round: number, threadId: string) =>
    `/backlog/${kind}/${name}/review/${round}/request/${threadId}/dismiss`,
  executionTriggerReviewAgent: (executionId: string) =>
    `/execution/${executionId}/trigger-review-agent`,
  // Embedded service endpoints (served at origin root, not under /api/v1)
  embeddedExternalUrl: (serviceName: string) =>
    `/embedded/${encodeURIComponent(serviceName)}/external-url`,
  // AI search
  searchAI: "/search/ai",
  searchAIStatus: "/search/ai/status",
  searchAIReconcile: "/search/ai/reconcile",
  searchAIReconcileStatus: "/search/ai/reconcile/status",
  searchAIReconcileCancel: "/search/ai/reconcile/cancel",
} as const;

/**
 * Type representing valid API endpoint keys.
 */
export type ApiEndpoint = keyof typeof API_ENDPOINTS;
