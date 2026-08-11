// Barrel re-export for backwards compatibility
export {
  parseProtoResponse,
  requireProtoField,
  toProtoJson,
  buildMessage,
} from "./shared";
export type { ProtoSchema } from "./shared";

export {
  listBacklogResponseSchema,
  backlogItemResponseSchema,
  backlogFilesResponseSchema,
  backlogFileResponseSchema,
  backlogFileOperationResponseSchema,
  queueBacklogResponseSchema,
  backlogResearchResponseSchema,
  mapProtoBacklogItem,
  mapProtoBacklogFile,
} from "./backlog-contracts";

export {
  listScenariosResponseSchema,
  scenarioResponseSchema,
  deleteScenarioResponseSchema,
  specSyncArchiveResponseSchema,
  scenarioFilesResponseSchema,
  previewScenarioRemediationResponseSchema,
  applyScenarioRemediationResponseSchema,
  previewScenarioMaturityCampaignResponseSchema,
  applyScenarioMaturityCampaignResponseSchema,
  DeleteScenarioRequestSchema,
  PreserveFilesRequestSchema,
  SpecSyncArchiveRequestSchema,
  PreviewScenarioRemediationRequestSchema,
  ApplyScenarioRemediationRequestSchema,
  PreviewScenarioMaturityCampaignRequestSchema,
  ApplyScenarioMaturityCampaignRequestSchema,
  mapProtoScenario,
  mapProtoScenarioFile,
  mapDeleteScenarioResponse,
  mapSpecSyncArchiveResponse,
} from "./scenario-contracts";

export {
  listExecutionResponseSchema,
  executionResponseSchema,
  CreateExecutionRequestSchema,
  FollowUpExecutionRequestSchema,
  mapProtoExecutionRecord,
} from "./execution-contracts";

export {
  settingsResponseSchema,
  mapProtoSettings,
  mapProtoPolicyProjection,
} from "./settings-contracts";

export {
  listAgentActivitiesResponseSchema,
  agentActivityResponseSchema,
  mapProtoAgentActivity,
} from "./agent-activity-contracts";

export {
  agentManagerStatusResponseSchema,
} from "./agent-manager-contracts";

export {
  graphResponseSchema,
} from "./graph-contracts";

export {
  planBoardResponseSchema,
} from "./plan-contracts";

export {
  listAgentSessionsResponseSchema,
  getAgentSessionResponseSchema,
  getAgentSessionStartupBriefResponseSchema,
  previewAgentSessionPromptResponseSchema,
  createAgentSessionResponseSchema,
  startAgentSessionResponseSchema,
  continueAgentSessionResponseSchema,
  listAgentSessionEventsResponseSchema,
  refreshAgentSessionResponseSchema,
  cancelAgentSessionResponseSchema,
  deleteAgentSessionResponseSchema,
  applyAgentSessionProposalResponseSchema,
  listAgentSessionArtifactsResponseSchema,
  getArtifactsByEntityResponseSchema,
  uploadAgentSessionAttachmentsResponseSchema,
  mapProtoAgentSession,
  mapProtoAgentSessionMessage,
  mapProtoAgentSessionContextItem,
  mapProtoAgentSessionAttachment,
  mapProtoAgentSessionProposal,
  mapProtoAgentSessionArtifact,
  mapProtoAgentSessionAttribution,
  mapProtoAgentSessionRunEvent,
} from "./agent-session-contracts";
