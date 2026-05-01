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
  DeleteScenarioRequestSchema,
  PreserveFilesRequestSchema,
  SpecSyncArchiveRequestSchema,
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
  listAgentSessionsResponseSchema,
  getAgentSessionResponseSchema,
  createAgentSessionResponseSchema,
  continueAgentSessionResponseSchema,
  refreshAgentSessionResponseSchema,
  cancelAgentSessionResponseSchema,
  applyAgentSessionProposalResponseSchema,
  listAgentSessionArtifactsResponseSchema,
  getArtifactsByEntityResponseSchema,
  mapProtoAgentSession,
  mapProtoAgentSessionMessage,
  mapProtoAgentSessionProposal,
  mapProtoAgentSessionArtifact,
  mapProtoAgentSessionAttribution,
} from "./agent-session-contracts";
