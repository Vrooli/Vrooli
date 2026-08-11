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
  backlogPlanRender: (kind: string, name: string) => `/backlog/${kind}/${name}/plan-render`,
  backlogPlanAccept: (kind: string, name: string) => `/backlog/${kind}/${name}/plan-accept`,
  backlogPlanAuthor: (kind: string, name: string) => `/backlog/${kind}/${name}/plan-author`,
  backlogPlanRepair: (kind: string, name: string) => `/backlog/${kind}/${name}/plan-repair`,
  backlogNextAction: (kind: string, name: string) => `/backlog/${kind}/${name}/next-action`,
  backlogNextActions: "/backlog/next-actions",
  nextActionsFeed: "/next-actions/feed",
  goalFiles: (name: string) => `/goals/${name}/files`,
  goalFileContent: (name: string, filePath: string) => `/goals/${name}/files/${filePath}`,
  goalFileOperations: (name: string) => `/goals/${name}/files`,
  planWorkshops: "/plan-workshops",
  planWorkshopById: (id: string) => `/plan-workshops/${id}`,
  planWorkshopReview: (id: string) => `/plan-workshops/${id}/review`,
  planWorkshopResponses: (id: string) => `/plan-workshops/${id}/responses`,
  backlogQueue: (kind: string, name: string) => `/backlog/${kind}/${name}/queue`,
  backlogWorkFeed: (kind: string, name: string) => `/backlog/${kind}/${name}/work-feed`,
  backlogRetry: (kind: string, name: string) => `/backlog/${kind}/${name}/retry`,
  backlogDispatchFollowUp: (kind: string, name: string) => `/backlog/${kind}/${name}/follow-up/dispatch`,
  backlogAuthorFollowUp: (kind: string, name: string) => `/backlog/${kind}/${name}/follow-up/author`,
  backlogArchiveTargets: (kind: string, name: string) => `/backlog/${kind}/${name}/archive/targets`,
  backlogArchiveTarget: (kind: string, name: string, targetId: string) => `/backlog/${kind}/${name}/archive/targets/${targetId}`,
  backlogArchiveRequirements: (kind: string, name: string) => `/backlog/${kind}/${name}/archive/requirements`,
  backlogArchiveRequirementsModule: (kind: string, name: string, moduleId: string) => `/backlog/${kind}/${name}/archive/requirements/${moduleId}`,
  backlogArchiveRequirementsModuleMeta: (kind: string, name: string, moduleId: string) => `/backlog/${kind}/${name}/archive/requirements/${moduleId}/meta`,
  backlogArchiveReview: (kind: string, name: string) => `/backlog/${kind}/${name}/archive/review`,
  backlogArchiveItem: (kind: string, name: string) => `/backlog/${kind}/${name}/archive-item`,
  backlogRecreate: (kind: string, name: string) => `/backlog/${kind}/${name}/recreate`,
  backlogResetArtifacts: (kind: string, name: string) => `/backlog/${kind}/${name}/reset-artifacts`,
  backlogExport: "/backlog/export",
  backlogImport: "/backlog/import",
  backlogValidateGlobs: "/backlog/validate-globs",
  backlogSummary: "/backlog/summary",
  backlogPendingQuestions: "/backlog/pending-questions",
  scenarios: "/scenarios",
  scenarioByName: (name: string) => `/scenarios/${name}`,
  scenarioFiles: (name: string) => `/scenarios/${name}/files`,
  scenarioSpecSyncArchive: (name: string) => `/scenarios/${name}/spec-sync-archive`,
  scenarioStart: (name: string) => `/scenarios/${name}/start`,
  scenarioStop: (name: string) => `/scenarios/${name}/stop`,
  scenarioRestart: (name: string) => `/scenarios/${name}/restart`,
  scenarioContext: (name: string) => `/scenarios/${name}/context`,
  scenarioRemediationPreview: (name: string) => `/scenarios/${name}/remediation/preview`,
  scenarioRemediationApply: (name: string) => `/scenarios/${name}/remediation/apply`,
  scenarioMaturityCampaignPreview: (name: string) => `/scenarios/${name}/maturity-campaign/preview`,
  scenarioMaturityCampaignApply: (name: string) => `/scenarios/${name}/maturity-campaign/apply`,
  agentManagerStatus: "/agent-manager/status",
  agentManagerRun: (runId: string) => `/agent-manager/runs/${runId}`,
  agentManagerStopRun: (runId: string) => `/agent-manager/runs/${runId}/stop`,
  settings: "/settings",
  integrations: "/integrations",
  execution: "/execution",
  executionStrategies: "/execution/strategies",
  executionById: (executionId: string) => `/execution/${executionId}`,
  executionPromptTrace: (executionId: string) => `/execution/${executionId}/prompt-trace`,
  executionProgress: (executionId: string) => `/execution/${executionId}/progress`,
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
  agentSessionPromptPreview: (sessionId: string) => `/agent-sessions/${sessionId}/prompt-preview`,
  agentSessionEvents: (sessionId: string) => `/agent-sessions/${sessionId}/events`,
  agentSessionRefresh: (sessionId: string) => `/agent-sessions/${sessionId}/refresh`,
  agentSessionCancel: (sessionId: string) => `/agent-sessions/${sessionId}/cancel`,
  agentSessionApplyProposal: (sessionId: string, proposalId: string) =>
    `/agent-sessions/${sessionId}/proposals/${proposalId}/apply`,
  proposalSessions: "/proposal-sessions",
  agentSessionAcceptKeepRecommendation: (sessionId: string, proposalId: string) =>
    `/agent-sessions/${sessionId}/proposals/${proposalId}/accept-keep`,
  agentSessionReviseMutationProposal: (sessionId: string, proposalId: string) =>
    `/agent-sessions/${sessionId}/proposals/${proposalId}/revise`,
  agentSessionArtifacts: (sessionId: string) => `/agent-sessions/${sessionId}/artifacts`,
  agentSessionArtifactsByEntity: "/artifacts/by-entity",
  gctStatus: "/gct/status",
  promptsCatalog: "/prompts/catalog",
  promptSkills: "/prompts/skills",
  promptSkillById: (skillId: string) => `/prompts/skills/${skillId}`,
  promptSkillVersions: (skillId: string) => `/prompts/skills/${skillId}/versions`,
  promptSkillRevert: (skillId: string, version: number) => `/prompts/skills/${skillId}/revert/${version}`,
  promptsPreview: "/prompts/preview",
  promptExperimentResults: (experimentId: string) => `/prompts/experiments/${experimentId}/results`,
  captures: "/captures",
  captureById: (id: string) => `/captures/${id}`,
  records: "/records",
  recordsCapture: "/records/capture",
  recordById: (id: string) => `/records/${id}`,
  recordCapture: (id: string) => `/records/${id}/capture`,
  recordNarrative: (id: string) => `/records/${id}/narrative`,
  recordSupersede: (id: string) => `/records/${id}/supersede`,
  recordSearch: "/records/search",
  graph: "/graph",
  plan: "/plan",
  planImport: "/plan-import",
  planImportPlans: "/plan-import/plans",
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
  // Continuous goal-directed auto-enqueue toggle (D4, default OFF).
  executionAutoDrain: "/execution/auto-drain",
  // Goals — first-class scope entities (targets + transitive closure).
  goals: "/goals",
  goalByName: (name: string) => `/goals/${name}`,
  goalTargets: (name: string) => `/goals/${name}/targets`,
  goalMilestones: (name: string) => `/goals/${name}/milestones`,
  goalMilestone: (name: string, milestone: string) => `/goals/${name}/milestones/${milestone}`,
  goalMilestoneItems: (name: string, milestone: string) => `/goals/${name}/milestones/${milestone}/items`,
  goalMilestoneReviewRun: (name: string, milestone: string) => `/goals/${name}/milestones/${milestone}/review-run`,
  goalArchiveItem: (name: string) => `/goals/${name}/archive-item`,
  goalCloseOut: (name: string) => `/goals/${name}/close-out`,
  goalPlanRun: (name: string) => `/goals/${name}/plan-run`,
  goalDiscoverRun: (name: string) => `/goals/${name}/discover-run`,
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
