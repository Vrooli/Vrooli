/**
 * Services Layer - Data access seams for Swarm Manager
 *
 * This module provides service objects that encapsulate API operations.
 * Each service is a seam - it can be substituted for testing without
 * mocking the entire HTTP layer.
 *
 * Usage:
 * - Import services for direct use in components or hooks
 * - Use factory functions (createXxxService) to inject mock clients in tests
 */

// Backlog
export { backlogService, createBacklogService } from "./backlog-service";
export { autoFilerService, createAutoFilerService } from "./auto-filer-service";
export type { IBacklogService, QueueResponse } from "./backlog-service";

// Captures
export { captureService, createCaptureService } from "./capture-service";
export type { ICaptureService, CreateCaptureResponse, ClassifyResponse } from "./capture-service";

// Scenarios
export { scenariosService, createScenariosService } from "./scenarios-service";
export type { IScenariosService } from "./scenarios-service";

// Settings
export { settingsService, createSettingsService } from "./settings-service";
export type { ISettingsService } from "./settings-service";

// Integration status
export { integrationStatusService, createIntegrationStatusService } from "./integration-status-service";
export type {
  IIntegrationStatusService,
  IntegrationAvailability,
  IntegrationStatus,
  IntegrationStatusResponse,
} from "./integration-status-service";

// Agent Manager
export { agentManagerService, createAgentManagerService } from "./agent-manager-service";
export type { IAgentManagerService, AgentRunDetails } from "./agent-manager-service";

// Agent Activity
export { agentActivityService, createAgentActivityService } from "./agent-activity-service";
export type { IAgentActivityService, ListAgentActivitiesFilters } from "./agent-activity-service";

// Agent Sessions
export { agentSessionService, createAgentSessionService } from "./agent-session-service";
export { proposalSessionService, createProposalSessionService } from "./proposal-session-service";
export type {
  ApplyAgentSessionProposalResult,
  CreateAgentSessionArgs,
  ContinueAgentSessionArgs,
  IAgentSessionService,
  ListAgentSessionsFilters,
} from "./agent-session-service";

// Execution
export { executionService, createExecutionService } from "./execution-service";
export type { IExecutionService, CreateExecutionRequest, ListExecutionFilters } from "./execution-service";

// Initiatives
export { initiativeService, createInitiativeService } from "./initiative-service";
export type { IInitiativeService } from "./initiative-service";

// Goals
export { goalsService, createGoalsService } from "./goals-service";
export type { IGoalsService } from "./goals-service";

// Prompt Center
export { promptService, createPromptService } from "./prompt-service";
export type {
	IPromptService,
	PromptPreviewResponse,
} from "./prompt-service";

// Graph
export { graphService, createGraphService } from "./graph-service";
export type { IGraphService, GraphProjection, GraphProjectionMeta } from "./graph-service";

// Stats
export { statsService, createStatsService } from "./stats-service";
export type { IStatsService } from "./stats-service";

// GCT
export { gctService, createGCTService } from "./gct-service";
export type { IGCTService, GCTStatusResponse } from "./gct-service";

// Review
export { reviewService, createReviewService } from "./review-service";
export type {
  IReviewService,
  ReviewRound,
  ReviewRoundStatus,
  EvidenceItem,
  EvidenceType,
  RequestThread,
} from "./review-service";


// Initiative Review
export { initiativeReviewService, createInitiativeReviewService } from "./initiative-review-service";
export type { IInitiativeReviewService, DecideInitiativeReviewArgs } from "./initiative-review-service";

// Embedded Services
export { embeddedService, createEmbeddedService } from "./embedded-service";
export type { IEmbeddedService } from "./embedded-service";
