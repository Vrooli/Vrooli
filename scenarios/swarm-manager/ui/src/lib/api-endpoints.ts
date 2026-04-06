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
  backlogResearch: (kind: string, name: string) => `/backlog/${kind}/${name}/research`,
  backlogWorkshopSave: (kind: string, name: string) => `/backlog/${kind}/${name}/workshop/save`,
  backlogWorkshopDeleteRound: (kind: string, name: string) => `/backlog/${kind}/${name}/workshop/round`,
  backlogWorkshopReset: (kind: string, name: string) => `/backlog/${kind}/${name}/workshop/reset`,
  backlogWorkshopCancelPendingAdvance: (kind: string, name: string) => `/backlog/${kind}/${name}/workshop/pending-advance`,
  backlogArchiveTargets: (kind: string, name: string) => `/backlog/${kind}/${name}/archive/targets`,
  backlogArchiveTarget: (kind: string, name: string, targetId: string) => `/backlog/${kind}/${name}/archive/targets/${targetId}`,
  backlogArchiveRequirements: (kind: string, name: string) => `/backlog/${kind}/${name}/archive/requirements`,
  backlogArchiveRequirementsModule: (kind: string, name: string, moduleId: string) => `/backlog/${kind}/${name}/archive/requirements/${moduleId}`,
  backlogArchiveRequirementsModuleMeta: (kind: string, name: string, moduleId: string) => `/backlog/${kind}/${name}/archive/requirements/${moduleId}/meta`,
  backlogArchiveReview: (kind: string, name: string) => `/backlog/${kind}/${name}/archive/review`,
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
  initiativeItems: (name: string) => `/initiatives/${name}/items`,
  initiativeFiles: (name: string) => `/initiatives/${name}/files`,
  initiativeFileOperations: (name: string) => `/initiatives/${name}/files`,
  initiativeFileContent: (name: string, filePath: string) =>
    `/initiatives/${name}/files/${filePath}`,
  graph: "/graph",
  overview: "/overview",
  stats: "/stats",
  health: "/health",
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
} as const;

/**
 * Type representing valid API endpoint keys.
 */
export type ApiEndpoint = keyof typeof API_ENDPOINTS;
