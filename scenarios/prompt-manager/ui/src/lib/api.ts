/**
 * API client for prompt-manager Go API.
 *
 * Mature domains use generated Connect clients. Domains not yet migrated keep
 * their REST transport until their migration slice lands. In particular:
 * - POST /api/v1/prompt-preview - preview constructed prompts
 *
 * REST responses are validated with Zod; generated protobuf messages validate
 * the migrated wire contracts and are mapped into the existing UI models.
 */

import { create, fromJson, toJson, type JsonValue } from '@bufbuild/protobuf'
import { ValueSchema, type Value } from '@bufbuild/protobuf/wkt'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'
import {
  ActionSchema as ActionProtoSchema,
  ActionValidationResultSchema as ActionValidationResultProtoSchema,
  ActionsService,
  AuthorActionRequestSchema as AuthorActionRequestProtoSchema,
  DeleteActionRequestSchema as DeleteActionRequestProtoSchema,
  GetActionRequestSchema as GetActionRequestProtoSchema,
  ListActionsRequestSchema as ListActionsRequestProtoSchema,
  RunActionRequestSchema as RunActionRequestProtoSchema,
  RunActionResponseSchema as RunActionResponseProtoSchema,
  UpdateActionRequestSchema as UpdateActionRequestProtoSchema,
  ValidateActionRequestSchema as ValidateActionRequestProtoSchema,
} from '@vrooli/proto-types/prompt-manager/v1/actions/actions_pb'
import {
  CreateSkillRequestSchema as CreateSkillRequestProtoSchema,
  CreateSkillVariantRequestSchema as CreateSkillVariantRequestProtoSchema,
  DeleteSkillRequestSchema as DeleteSkillRequestProtoSchema,
  DeleteSkillVariantRequestSchema as DeleteSkillVariantRequestProtoSchema,
  GetSkillRequestSchema as GetSkillRequestProtoSchema,
  ListSkillVariantsRequestSchema as ListSkillVariantsRequestProtoSchema,
  ListSkillVersionsRequestSchema as ListSkillVersionsRequestProtoSchema,
  ListSkillsRequestSchema as ListSkillsRequestProtoSchema,
  RateSkillRequestSchema as RateSkillRequestProtoSchema,
  RecordSkillUsageRequestSchema as RecordSkillUsageRequestProtoSchema,
  RevertSkillRequestSchema as RevertSkillRequestProtoSchema,
  SkillSchema as SkillProtoSchema,
  SkillVariantSchema as SkillVariantProtoSchema,
  SkillsService,
  UpdateSkillRequestSchema as UpdateSkillRequestProtoSchema,
} from '@vrooli/proto-types/prompt-manager/v1/skills/skills_pb'
import {
  CreateTagRequestSchema as CreateTagRequestProtoSchema,
  ListTagsRequestSchema as ListTagsRequestProtoSchema,
  TagSchema as TagProtoSchema,
  TagsService,
} from '@vrooli/proto-types/prompt-manager/v1/tags/tags_pb'
import {
  SearchService,
  SearchSkillsRequestSchema as TextSearchSkillsRequestProtoSchema,
  SearchSkillsResponseSchema as TextSearchSkillsResponseProtoSchema,
  SearchSkillContentRequestSchema as SearchSkillContentRequestProtoSchema,
  SearchSkillContentResponseSchema as SearchSkillContentResponseProtoSchema,
} from '@vrooli/proto-types/prompt-manager/v1/search/search_pb'
import {
  AISearchService,
  SearchSkillsRequestSchema as AISearchSkillsRequestProtoSchema,
  SearchSkillsResponseSchema as AISearchSkillsResponseProtoSchema,
  SearchAgentsRequestSchema as AISearchAgentsRequestProtoSchema,
  SearchAgentsResponseSchema as AISearchAgentsResponseProtoSchema,
  SearchActionsRequestSchema as AISearchActionsRequestProtoSchema,
  SearchActionsResponseSchema as AISearchActionsResponseProtoSchema,
  SearchTeamsRequestSchema as AISearchTeamsRequestProtoSchema,
  SearchTeamsResponseSchema as AISearchTeamsResponseProtoSchema,
  GetStatusRequestSchema as GetAISearchStatusRequestProtoSchema,
  GetStatusResponseSchema as GetAISearchStatusResponseProtoSchema,
  ReconcileRequestSchema as ReconcileRequestProtoSchema,
  ReconcileStatusSchema as ReconcileStatusProtoSchema,
  GetReconcileStatusRequestSchema as GetReconcileStatusRequestProtoSchema,
  CancelReconcileRequestSchema as CancelReconcileRequestProtoSchema,
  type ReconcileStatus as ReconcileStatusProto,
} from '@vrooli/proto-types/prompt-manager/v1/aisearch/aisearch_pb'
import {
  DiscoveryService,
  DiscoverRequestSchema as DiscoverRequestProtoSchema,
  DiscoverResponseSchema as DiscoverResponseProtoSchema,
} from '@vrooli/proto-types/prompt-manager/v1/discovery/discovery_pb'
import {
  AgentsService,
  AgentSchema as AgentProtoSchema,
  CreateAgentRequestSchema as CreateAgentRequestProtoSchema,
  CreateFileRequestSchema as CreateAgentFileRequestProtoSchema,
  DeleteAgentRequestSchema as DeleteAgentRequestProtoSchema,
  DeleteFileRequestSchema as DeleteAgentFileRequestProtoSchema,
  FileContentSchema as AgentFileContentProtoSchema,
  GetAgentRequestSchema as GetAgentRequestProtoSchema,
  GetFileRequestSchema as GetAgentFileRequestProtoSchema,
  GetSoulRequestSchema as GetSoulRequestProtoSchema,
  ListAgentsRequestSchema as ListAgentsRequestProtoSchema,
  ListAgentTeamsRequestSchema as ListAgentTeamsRequestProtoSchema,
  ListAgentTeamsResponseSchema as ListAgentTeamsResponseProtoSchema,
  ListFilesRequestSchema as ListAgentFilesRequestProtoSchema,
  ListFilesResponseSchema as ListAgentFilesResponseProtoSchema,
  RenameFileRequestSchema as RenameAgentFileRequestProtoSchema,
  SetFileRequestSchema as SetAgentFileRequestProtoSchema,
  SetSoulRequestSchema as SetSoulRequestProtoSchema,
  SoulSchema as SoulProtoSchema,
  UpdateAgentRequestSchema as UpdateAgentRequestProtoSchema,
} from '@vrooli/proto-types/prompt-manager/v1/agents/agents_pb'
import {
  TeamsService,
  AddMemberRequestSchema as AddTeamMemberRequestProtoSchema,
  CreateSharedFileRequestSchema as CreateTeamSharedFileRequestProtoSchema,
  CreateTeamRequestSchema as CreateTeamRequestProtoSchema,
  DeleteSharedFileRequestSchema as DeleteTeamSharedFileRequestProtoSchema,
  DeleteTeamRequestSchema as DeleteTeamRequestProtoSchema,
  ExportClaudeCodeTeamRequestSchema as ExportClaudeCodeTeamRequestProtoSchema,
  ExportClaudeCodeTeamResponseSchema as ExportClaudeCodeTeamResponseProtoSchema,
  ExclusiveMembersResponseSchema as ExclusiveMembersResponseProtoSchema,
  GetExclusiveMembersRequestSchema as GetExclusiveMembersRequestProtoSchema,
  GetRolesRequestSchema as GetTeamRolesRequestProtoSchema,
  GetSharedFileRequestSchema as GetTeamSharedFileRequestProtoSchema,
  GetTeamRequestSchema as GetTeamRequestProtoSchema,
  ImportClaudeCodeTeamRequestSchema as ImportClaudeCodeTeamRequestProtoSchema,
  ListAvailableClaudeCodeTeamsRequestSchema as ListAvailableCCTeamsRequestProtoSchema,
  ListSharedFilesRequestSchema as ListTeamSharedFilesRequestProtoSchema,
  ListSharedFilesResponseSchema as ListTeamSharedFilesResponseProtoSchema,
  ListTeamsRequestSchema as ListTeamsRequestProtoSchema,
  MemberSchema as TeamMemberProtoSchema,
  RemoveMemberRequestSchema as RemoveTeamMemberRequestProtoSchema,
  RenameSharedFileRequestSchema as RenameTeamSharedFileRequestProtoSchema,
  RoleSchema as TeamRoleProtoSchema,
  SetRolesRequestSchema as SetTeamRolesRequestProtoSchema,
  SetSharedFileRequestSchema as SetTeamSharedFileRequestProtoSchema,
  SharedFileContentSchema as TeamSharedFileContentProtoSchema,
  TeamDetailsSchema as TeamDetailsProtoSchema,
  TeamSchema as TeamProtoSchema,
  UpdateMemberRequestSchema as UpdateTeamMemberRequestProtoSchema,
  UpdateTeamRequestSchema as UpdateTeamRequestProtoSchema,
} from '@vrooli/proto-types/prompt-manager/v1/teams/teams_pb'
import {
  TopicsService,
  AccumulatedSkillsResponseSchema as AccumulatedSkillsResponseProtoSchema,
  CreateTopicRequestSchema as CreateTopicRequestProtoSchema,
  DeleteTopicRequestSchema as DeleteTopicRequestProtoSchema,
  GetAccumulatedSkillsRequestSchema as GetAccumulatedSkillsRequestProtoSchema,
  GetTopicRequestSchema as GetTopicRequestProtoSchema,
  ListTopicsRequestSchema as ListTopicsRequestProtoSchema,
  MatchTopicsRequestSchema as MatchTopicsRequestProtoSchema,
  MatchTopicsResponseSchema as MatchTopicsResponseProtoSchema,
  TopicSchema as TopicProtoSchema,
  UpdateTopicRequestSchema as UpdateTopicRequestProtoSchema,
} from '@vrooli/proto-types/prompt-manager/v1/topics/topics_pb'
import {
  TemplatesService,
  ListAgentFileTemplatesRequestSchema as ListAgentFileTemplatesRequestProtoSchema,
  ListAgentFileTemplatesResponseSchema as ListAgentFileTemplatesResponseProtoSchema,
} from '@vrooli/proto-types/prompt-manager/v1/templates/templates_pb'
import {
  TestingService,
  ListSkillTestHistoryRequestSchema as ListSkillTestHistoryRequestProtoSchema,
  RunSkillTestRequestSchema as RunSkillTestRequestProtoSchema,
  SkillTestResponseSchema as SkillTestResponseProtoSchema,
  SkillTestResultSchema as SkillTestResultProtoSchema,
} from '@vrooli/proto-types/prompt-manager/v1/testing/testing_pb'
import {
  MetadataService,
  FetchOpenGraphRequestSchema as FetchOpenGraphRequestProtoSchema,
  OpenGraphMetadataSchema as OpenGraphMetadataProtoSchema,
} from '@vrooli/proto-types/prompt-manager/v1/metadata/metadata_pb'
import {
  WorldScaleService,
  GetWorldScaleRequestSchema as GetWorldScaleRequestProtoSchema,
  SetWorldScaleRequestSchema as SetWorldScaleRequestProtoSchema,
  WorldScaleSchema as WorldScaleProtoSchema,
} from '@vrooli/proto-types/prompt-manager/v1/worldscale/worldscale_pb'
import {
  WorldSeatsService,
  GetWorldSeatsRequestSchema as GetWorldSeatsRequestProtoSchema,
  SetWorldSeatsRequestSchema as SetWorldSeatsRequestProtoSchema,
} from '@vrooli/proto-types/prompt-manager/v1/worldseats/worldseats_pb'
import {
  ExperimentsService,
  ListExperimentsRequestSchema,
  GetExperimentRequestSchema,
  CreateExperimentRequestSchema,
  UpdateExperimentRequestSchema,
  DeleteExperimentRequestSchema,
  StartExperimentRequestSchema,
  ConcludeExperimentRequestSchema,
  ListOutcomesRequestSchema,
  GetExperimentReportRequestSchema,
  ExperimentSchema as ExperimentWireSchema,
  ExperimentOutcomeSchema,
  ExperimentReportSchema,
} from '@vrooli/proto-types/prompt-manager/v1/experiments/experiments_pb'
import {
  GraphService,
  GetGraphRequestSchema,
  RegenerateGraphRequestSchema,
  ListNodesRequestSchema,
  ListPopularNodesRequestSchema,
  ListCyclesRequestSchema,
  GetNodeRequestSchema,
  GetHealthScoresRequestSchema,
  GetHealthConfigRequestSchema,
  UpdateHealthConfigRequestSchema,
  GraphIndexSchema,
  NodeDetailSchema,
  NodeSchema as GraphNodeWireSchema,
  HealthScoreSchema,
  HealthConfigSchema,
} from '@vrooli/proto-types/prompt-manager/v1/graph/graph_pb'
import {
  MemberflowService,
  EmptyRequestSchema as MemberflowEmptyRequestSchema,
  MemberRequestSchema as MemberflowMemberRequestSchema,
  TeamRequestSchema as MemberflowTeamRequestSchema,
  UpdateMemberTopicsRequestSchema,
} from '@vrooli/proto-types/prompt-manager/v1/memberflow/memberflow_pb'
import {
  HeartbeatService,
  EmptyRequestSchema as HeartbeatEmptyRequestSchema,
  QueryRequestSchema as HeartbeatQueryRequestSchema,
  TeamRequestSchema as HeartbeatTeamRequestSchema,
  TeamQueryRequestSchema as HeartbeatTeamQueryRequestSchema,
  TeamMutationRequestSchema as HeartbeatTeamMutationRequestSchema,
  MemberRequestSchema as HeartbeatMemberRequestSchema,
  MemberQueryRequestSchema as HeartbeatMemberQueryRequestSchema,
  MemberMutationRequestSchema as HeartbeatMemberMutationRequestSchema,
  TaskMutationRequestSchema as HeartbeatTaskMutationRequestSchema,
  BugMutationRequestSchema as HeartbeatBugMutationRequestSchema,
  LogRequestSchema as HeartbeatLogRequestSchema,
  RunRequestSchema as HeartbeatRunRequestSchema,
  RunQueryRequestSchema as HeartbeatRunQueryRequestSchema,
  RunMutationRequestSchema as HeartbeatRunMutationRequestSchema,
  JsonMutationRequestSchema as HeartbeatJsonMutationRequestSchema,
} from '@vrooli/proto-types/prompt-manager/v1/heartbeat/heartbeat_pb'
import type { ZodType } from 'zod'
import {
  parseOrThrow,
  parseArrayFiltered,
  SkillSchema,
  ActionSchema,
  ActionArraySchema,
  ActionMutationResponseSchema,
  ActionValidationResponseSchema,
  ActionRunResponseSchema,
  TagSchema,
  TagArraySchema,
  SkillTestResultSchema,
  SkillTestResultArraySchema,
  UsageResponseSchema,
  RatingResponseSchema,
  HealthResponseSchema,
  DisplayResponseSchema,
  GraphResponseSchema,
  NodeDetailResponseSchema,
  RegenerateResponseSchema,
  NodeListResponseSchema,
  PopularityResponseSchema,
  CircularRefResponseSchema,
  GraphHealthResponseSchema,
  GraphHealthConfigResponseSchema,
  OperatingMapSchema,
  AgentSchema,
  AgentArraySchema,
  SoulResponseSchema,
  AgentFileListResponseSchema,
  AgentFileContentResponseSchema,
  AgentFileTemplateListResponseSchema,
  PromptPreviewResponseSchema,
  StructuredPromptPreviewResponseSchema,
  TeamPromptMatrixResponseSchema,
  AgentTeamsResponseSchema,
  AISearchResponseSchema,
  SkillSearchResponseSchema,
  AISearchStatusSchema,
  AIReindexStatusSchema,
  ContentSearchResponseSchema,
  AIActionSearchResponseSchema,
  AIAgentSearchResponseSchema,
  AITeamSearchResponseSchema,
  DiscoverResponseSchema,
  BudgetConfigSchema,
  DiscoverFilterConfigSchema,
  LinkPreviewDataSchema,
  TeamArraySchema,
  TeamDetailsSchema,
  TeamRoleSchema,
  TeamMemberSchema,
  TeamSharedFileListResponseSchema,
  TeamSharedFileContentResponseSchema,
  AvailableCCTeamSchema,
  ExportCCResponseSchema,
  ExclusiveMembersResponseSchema,
  TopicSchema,
  TopicArraySchema,
  AccumulatedSkillsResponseSchema,
  TopicMatchResponseSchema,
  WorldScaleConfigSchema,
  WorldSeatsConfigSchema,
  type Topic,
  type CreateTopicRequest,
  type UpdateTopicRequest,
  type AccumulatedSkillsResponse,
  type TopicMatchResponse,
  type WorldScaleConfig,
  type WorldSeatsConfig,
  VersionsResponseSchema,
  RevertResponseSchema,
  VariantSchema,
  VariantArraySchema,
  ExperimentSchema,
  ExperimentArraySchema,
  type VersionsResponse,
  type RevertResponse,
  type Variant,
  type CreateVariantRequest,
  type Experiment,
  type CreateExperimentRequest,
  type ConcludeExperimentRequest,
  type Skill,
  type CreateSkillRequest,
  type UpdateSkillRequest,
  type Action,
  type CreateActionRequest,
  type UpdateActionRequest,
  type ActionMutationResponse,
  type ActionValidationResponse,
  type ActionRunRequest,
  type ActionRunResponse,
  type Tag,
  type SkillTestRequest,
  type SkillTestResult,
  type UsageResponse,
  type RatingResponse,
  type HealthResponse,
  type DisplayResponse,
  type DisplayFormat,
  type GraphResponse,
  type NodeDetailResponse,
  type RegenerateResponse,
  type NodeListResponse,
  type PopularityResponse,
  type CircularRefResponse,
  type GraphHealthResponse,
  type NodeHealthResponse,
  type GraphHealthConfigResponse,
  type Agent,
  type CreateAgentRequest,
  type UpdateAgentRequest,
  type SoulResponse,
  type AgentFileListResponse,
  type AgentFileContentResponse,
  type AgentFileCreateRequest,
  type AgentFileRenameRequest,
  type AgentFileTemplateListResponse,
  type PromptPreviewResponse,
  type StructuredPromptPreviewResponse,
  type TeamPromptMatrixResponse,
  type AgentTeamsResponse,
  type AISearchResponse,
  type SkillSearchResponse,
  type AISearchStatus,
  type AIReindexStatus,
  type ContentSearchResponse,
  type AIActionSearchResponse,
  type AIAgentSearchResponse,
  type AITeamSearchResponse,
  type DiscoverResponse,
  type BudgetConfig,
  type DiscoverFilterConfig,
  type LinkPreviewData,
  type FolderType,
  type Team,
  type TeamDetails,
  type TeamRole,
  type TeamMember,
  type CreateTeamRequest,
  type UpdateTeamRequest,
  type AddMemberRequest,
  type UpdateMemberRequest,
  type TeamSharedFileListResponse,
  type TeamSharedFileContentResponse,
  type TeamSharedFileCreateRequest,
  type TeamSharedFileRenameRequest,
  type AvailableCCTeam,
  type ExportCCResponse,
  type ExclusiveMembersResponse,
} from '@/lib/schemas'
import type { SearchFilters, ActionFilters, Folder } from '@/types'

/**
 * Convert UI request models into protobuf JSON without leaking JavaScript-only
 * `undefined` values across the generated-client boundary.
 *
 * Optional object properties are omitted, matching JSON.stringify and the
 * protobuf JSON mapping. Undefined array entries are rejected because dropping
 * one would silently change the meaning and indexes of a repeated field.
 */
export function toProtoJson(value: unknown, path = '$'): JsonValue {
  if (value === null || typeof value === 'string' || typeof value === 'boolean') {
    return value
  }
  if (typeof value === 'number') {
    if (!Number.isFinite(value)) {
      throw new TypeError(`Non-finite protobuf JSON number at ${path}`)
    }
    return value
  }
  if (Array.isArray(value)) {
    return value.map((entry, index) => {
      if (entry === undefined) {
        throw new TypeError(`Undefined protobuf JSON array entry at ${path}[${index}]`)
      }
      return toProtoJson(entry, `${path}[${index}]`)
    })
  }
  if (typeof value === 'object') {
    const result: Record<string, JsonValue> = {}
    for (const [key, entry] of Object.entries(value)) {
      if (entry !== undefined) {
        result[key] = toProtoJson(entry, `${path}.${key}`)
      }
    }
    return result
  }
  throw new TypeError(`Unsupported protobuf JSON value at ${path}: ${typeof value}`)
}

function resolvePromptManagerApiBase(): string {
  if (import.meta.env.MODE === 'test') {
    return '/api/v1'
  }

  // Use @vrooli/api-base for automatic API resolution across all deployment contexts.
  return resolveApiBase({ appendSuffix: true })
}

export const API_BASE = resolvePromptManagerApiBase()
const CONNECT_BASE = import.meta.env.MODE === 'test'
  ? window.location.origin
  : resolveApiBase({ appendSuffix: false })
const connectTransport = createConnectTransport({ baseUrl: CONNECT_BASE })
const skillsClient = createClient(SkillsService, connectTransport)
const actionsClient = createClient(ActionsService, connectTransport)
const tagsClient = createClient(TagsService, connectTransport)
const searchClient = createClient(SearchService, connectTransport)
const aiSearchClient = createClient(AISearchService, connectTransport)
const discoveryClient = createClient(DiscoveryService, connectTransport)
const agentsClient = createClient(AgentsService, connectTransport)
const teamsClient = createClient(TeamsService, connectTransport)
const topicsClient = createClient(TopicsService, connectTransport)
const templatesClient = createClient(TemplatesService, connectTransport)
const testingClient = createClient(TestingService, connectTransport)
const metadataClient = createClient(MetadataService, connectTransport)
const worldScaleClient = createClient(WorldScaleService, connectTransport)
const worldSeatsClient = createClient(WorldSeatsService, connectTransport)
const experimentsClient = createClient(ExperimentsService, connectTransport)
const graphClient = createClient(GraphService, connectTransport)
const memberflowClient = createClient(MemberflowService, connectTransport)
const heartbeatClient = createClient(HeartbeatService, connectTransport)

type Slice4Result = { handled: true; data: unknown } | { handled: false }

// connectSlice4Request is the compatibility edge for UI modules that still
// express requests as method/path pairs. Network traffic is nevertheless
// generated-client Connect traffic; callers can migrate their model mapping
// incrementally without keeping the retired REST routes alive.
export async function connectSlice4Request(endpoint: string, options?: RequestInit): Promise<Slice4Result> {
  const parsed = new URL(endpoint, 'http://prompt-manager.local')
  const s = parsed.pathname.split('/').filter(Boolean).map(decodeURIComponent)
  const method = (options?.method ?? 'GET').toUpperCase()
  const body = typeof options?.body === 'string' && options.body ? JSON.parse(options.body) as JsonValue : undefined
  const query = Object.fromEntries(parsed.searchParams.entries())
  const callOptions = options?.headers ? { headers: options.headers } : undefined

  if (s[0] === 'experiments' || (s[0] === 'skills' && s[2] === 'experiments')) {
    if (method === 'GET' && s[0] === 'skills') return { handled: true, data: (await experimentsClient.listExperiments(create(ListExperimentsRequestSchema, { skillId: s[1] }), callOptions)).experiments.map(v => toJson(ExperimentWireSchema, v)) }
    if (method === 'GET' && s.length === 1) return { handled: true, data: (await experimentsClient.listExperiments(create(ListExperimentsRequestSchema), callOptions)).experiments.map(v => toJson(ExperimentWireSchema, v)) }
    if (method === 'POST' && s.length === 1) return { handled: true, data: toJson(ExperimentWireSchema, await experimentsClient.createExperiment(fromJson(CreateExperimentRequestSchema, body ?? {}), callOptions)) }
    if (method === 'GET' && s.length === 2) return { handled: true, data: toJson(ExperimentWireSchema, await experimentsClient.getExperiment(create(GetExperimentRequestSchema, { experimentId: s[1] }), callOptions)) }
    if (method === 'PUT' && s.length === 2) return { handled: true, data: toJson(ExperimentWireSchema, await experimentsClient.updateExperiment(fromJson(UpdateExperimentRequestSchema, toProtoJson({ ...(body as object), experimentId: s[1] })), callOptions)) }
    if (method === 'DELETE' && s.length === 2) { await experimentsClient.deleteExperiment(create(DeleteExperimentRequestSchema, { experimentId: s[1] }), callOptions); return { handled: true, data: {} } }
    if (method === 'POST' && s[2] === 'start') return { handled: true, data: toJson(ExperimentWireSchema, await experimentsClient.startExperiment(create(StartExperimentRequestSchema, { experimentId: s[1] }), callOptions)) }
    if (method === 'POST' && s[2] === 'conclude') return { handled: true, data: toJson(ExperimentWireSchema, await experimentsClient.concludeExperiment(fromJson(ConcludeExperimentRequestSchema, toProtoJson({ ...(body as object), experimentId: s[1] })), callOptions)) }
    if (method === 'GET' && s[2] === 'outcomes') return { handled: true, data: (await experimentsClient.listOutcomes(create(ListOutcomesRequestSchema, { experimentId: s[1] }), callOptions)).outcomes.map(v => toJson(ExperimentOutcomeSchema, v)) }
    if (method === 'GET' && s[2] === 'report') return { handled: true, data: toJson(ExperimentReportSchema, await experimentsClient.getExperimentReport(create(GetExperimentReportRequestSchema, { experimentId: s[1] }), callOptions)) }
  }

  if (s[0] === 'graph') {
    if (method === 'GET' && s.length === 1) return { handled: true, data: toJson(GraphIndexSchema, await graphClient.getGraph(create(GetGraphRequestSchema), callOptions)) }
    if (method === 'POST' && s[1] === 'regenerate') return { handled: true, data: toJson(GraphIndexSchema, await graphClient.regenerateGraph(create(RegenerateGraphRequestSchema), callOptions)) }
    const nodeList = async (kind: string) => {
      const req = create(ListNodesRequestSchema)
      const response = kind === 'orphans' ? await graphClient.listOrphanedSkills(req, callOptions) : kind === 'skillless' ? await graphClient.listSkilllessAgents(req, callOptions) : kind === 'empty-teams' ? await graphClient.listEmptyTeams(req, callOptions) : await graphClient.listUnaffiliatedAgents(req, callOptions)
      return response.nodes.map(v => toJson(GraphNodeWireSchema, v))
    }
    if (method === 'GET' && ['orphans', 'skillless', 'empty-teams', 'unaffiliated'].includes(s[1] ?? '')) return { handled: true, data: await nodeList(s[1]!) }
    if (method === 'GET' && s[1] === 'popular') return { handled: true, data: (await graphClient.listPopularNodes(create(ListPopularNodesRequestSchema, { limit: Number(query.limit ?? 0) }), callOptions)).nodes.map(v => toJson(GraphNodeWireSchema, v)) }
    if (method === 'GET' && s[1] === 'cycles') return { handled: true, data: (await graphClient.listCycles(create(ListCyclesRequestSchema), callOptions)).cycles.map(v => v.nodeIds) }
    if (method === 'GET' && s[1] === 'health') return { handled: true, data: (await graphClient.getHealthScores(create(GetHealthScoresRequestSchema), callOptions)).scores.map(v => toJson(HealthScoreSchema, v)) }
    if (method === 'GET' && s[1] === 'health-config') return { handled: true, data: toJson(HealthConfigSchema, await graphClient.getHealthConfig(create(GetHealthConfigRequestSchema), callOptions)) }
    if (method === 'PUT' && s[1] === 'health-config') return { handled: true, data: toJson(HealthConfigSchema, await graphClient.updateHealthConfig(create(UpdateHealthConfigRequestSchema, { config: fromJson(HealthConfigSchema, body ?? {}) }), callOptions)) }
    if (method === 'GET' && s[1] === 'nodes' && s.length === 3) return { handled: true, data: toJson(NodeDetailSchema, await graphClient.getNode(create(GetNodeRequestSchema, { nodeId: s[2] }), callOptions)) }
  }

  const empty = create(MemberflowEmptyRequestSchema)
  if (method === 'GET' && s[0] === 'teams' && s[2] === 'members' && s[4] === 'topics') return { handled: true, data: plainValue((await memberflowClient.getMemberTopics(create(MemberflowMemberRequestSchema, { teamId: s[1], agentId: s[3] }), callOptions)).data) }
  if (method === 'PUT' && s[0] === 'teams' && s[2] === 'members' && s[4] === 'topics') return { handled: true, data: plainValue((await memberflowClient.updateMemberTopics(create(UpdateMemberTopicsRequestSchema, { teamId: s[1], agentId: s[3], topics: body === undefined ? undefined : fromJson(ValueSchema, body) }), callOptions)).data) }
  if (method === 'GET' && s[0] === 'teams' && s[2] === 'topics') return { handled: true, data: plainValue((await memberflowClient.getTeamTopics(create(MemberflowTeamRequestSchema, { teamId: s[1] }), callOptions)).data) }
  if (method === 'GET' && s[0] === 'topics' && s[1] === 'graph') return { handled: true, data: plainValue((await memberflowClient.getTopicGraph(empty, callOptions)).data) }
  if (method === 'GET' && s[0] === 'topics' && s[1] === 'rules') return { handled: true, data: plainValue((await memberflowClient.getRules(empty, callOptions)).data) }
  if (method === 'GET' && s[0] === 'topics' && s[1] === 'drain-status') return { handled: true, data: plainValue((await memberflowClient.getDrainStatus(empty, callOptions)).data) }
  if (method === 'GET' && s[0] === 'objectives') return { handled: true, data: plainValue((await memberflowClient.getObjectives(empty, callOptions)).data) }
  if (method === 'GET' && s[0] === 'orientation-cost') return { handled: true, data: plainValue((await memberflowClient.getOrientationCost(empty, callOptions)).data) }
  if (method === 'GET' && s[0] === 'instruments') return { handled: true, data: plainValue((await memberflowClient.getInstruments(empty, callOptions)).data) }
  if (method === 'GET' && s[0] === 'operating-models') {
    const response = s[1] === 'map' ? await memberflowClient.getOperatingMap(empty, callOptions) : s[1] === 'validate' ? await memberflowClient.validateOperatingModels(empty, callOptions) : s[1] === 'diff' ? await memberflowClient.diffOperatingModels(empty, callOptions) : s[1] === 'coverage' ? await memberflowClient.getOperatingModelCoverage(empty, callOptions) : await memberflowClient.getOperatingModels(empty, callOptions)
    return { handled: true, data: plainValue(response.data) }
  }
  const heartbeat = await connectHeartbeatRequest(s, method, body, query, callOptions)
  if (heartbeat.handled) return heartbeat
  return { handled: false }
}

async function connectHeartbeatRequest(s: string[], method: string, rawBody: JsonValue | undefined, query: Record<string, string>, callOptions: Parameters<typeof heartbeatClient.getHeartbeat>[1]): Promise<Slice4Result> {
  const body: Value | undefined = rawBody === undefined ? undefined : fromJson(ValueSchema, rawBody)
  const json = (response: { data?: Value }) => ({ handled: true, data: plainValue(response.data) } as const)
  if (s[0] === 'tasks' && method === 'POST') return json(await heartbeatClient.createTask(create(HeartbeatJsonMutationRequestSchema, { body }), callOptions))
  if (s[0] === 'runs') {
    if (s.length === 1 && method === 'GET') return json(await heartbeatClient.listRuns(create(HeartbeatQueryRequestSchema, { query }), callOptions))
    if (s.length === 1 && method === 'POST') return json(await heartbeatClient.createRun(create(HeartbeatJsonMutationRequestSchema, { body }), callOptions))
    if (s[1] === 'investigate') return json(await heartbeatClient.createInvestigationRun(create(HeartbeatJsonMutationRequestSchema, { body }), callOptions))
    if (s[1] === 'investigation-apply') return json(await heartbeatClient.createInvestigationApplyRun(create(HeartbeatJsonMutationRequestSchema, { body }), callOptions))
    if (s.length === 2) return json(await heartbeatClient.getRun(create(HeartbeatRunRequestSchema, { runId: s[1] }), callOptions))
    if (s[2] === 'retry') return json(await heartbeatClient.retryRun(create(HeartbeatRunMutationRequestSchema, { runId: s[1], body }), callOptions))
    if (s[2] === 'continue') return json(await heartbeatClient.continueRun(create(HeartbeatRunMutationRequestSchema, { runId: s[1], body }), callOptions))
    if (s[2] === 'events') return json(await heartbeatClient.getRunEvents(create(HeartbeatRunQueryRequestSchema, { runId: s[1], query }), callOptions))
  }
  if (s[0] === 'heartbeat-attempts') return json(await heartbeatClient.listHeartbeatAttempts(create(HeartbeatQueryRequestSchema, { query }), callOptions))
  if (s[0] === 'prompt-preview') return json(await heartbeatClient.previewPrompt(create(HeartbeatJsonMutationRequestSchema, { body }), callOptions))
  if (s[0] === 'prompt-preview-structured') return json(await heartbeatClient.previewPromptStructured(create(HeartbeatJsonMutationRequestSchema, { body }), callOptions))
  if (s[0] === 'heartbeats') {
    if (s[1] === 'control') {
      if (s.length === 2) return json(await heartbeatClient.getHeartbeatControl(create(HeartbeatEmptyRequestSchema), callOptions))
      if (s[2] === 'policy') return json(await heartbeatClient.updateHeartbeatControlPolicy(create(HeartbeatJsonMutationRequestSchema, { body }), callOptions))
      if (s[2] === 'pause') return json(await heartbeatClient.pauseHeartbeatControl(create(HeartbeatJsonMutationRequestSchema, { body }), callOptions))
      if (s[2] === 'resume') return json(await heartbeatClient.resumeHeartbeatControl(create(HeartbeatJsonMutationRequestSchema, { body }), callOptions))
    }
    if (s[1] === 'running' && s.length === 2) return json(await heartbeatClient.listRunning(create(HeartbeatEmptyRequestSchema), callOptions))
    if (s[1] === 'running' && s[4] === 'stop') return json(await heartbeatClient.stopRunning(create(HeartbeatMemberMutationRequestSchema, { teamId: s[2], agentId: s[3], body, query }), callOptions))
  }
  if (s[0] !== 'teams' || !s[1]) return { handled: false }
  const teamId = s[1]
  if (s[2] === 'heartbeats') {
    if (s.length === 3) return json(await heartbeatClient.listHeartbeats(create(HeartbeatTeamRequestSchema, { teamId }), callOptions))
    if (s[3] === 'control') {
      if (s.length === 4) return json(await heartbeatClient.getTeamHeartbeatControl(create(HeartbeatTeamRequestSchema, { teamId }), callOptions))
      const request = create(HeartbeatTeamMutationRequestSchema, { teamId, body, query })
      if (s[4] === 'policy') return json(await heartbeatClient.updateTeamHeartbeatControlPolicy(request, callOptions))
      if (s[4] === 'pause') return json(await heartbeatClient.pauseTeamHeartbeatControl(request, callOptions))
      if (s[4] === 'resume') return json(await heartbeatClient.resumeTeamHeartbeatControl(request, callOptions))
    }
    if (s[3] === 'logs') return json(await heartbeatClient.listTeamLogs(create(HeartbeatTeamQueryRequestSchema, { teamId, query }), callOptions))
    const agentId = s[3]
    if (!agentId) return { handled: false }
    if (s.length === 4) {
      if (method === 'GET') return json(await heartbeatClient.getHeartbeat(create(HeartbeatMemberRequestSchema, { teamId, agentId }), callOptions))
      if (method === 'DELETE') return json(await heartbeatClient.deleteHeartbeat(create(HeartbeatMemberRequestSchema, { teamId, agentId }), callOptions))
      const request = create(HeartbeatMemberMutationRequestSchema, { teamId, agentId, body, query })
      return method === 'POST' ? json(await heartbeatClient.createHeartbeat(request, callOptions)) : json(await heartbeatClient.updateHeartbeat(request, callOptions))
    }
    if (s[4] === 'trigger') return json(await heartbeatClient.triggerHeartbeat(create(HeartbeatMemberMutationRequestSchema, { teamId, agentId, body, query }), callOptions))
    if (s[4] === 'logs' && s.length === 5) return json(await heartbeatClient.listLogs(create(HeartbeatMemberQueryRequestSchema, { teamId, agentId, query }), callOptions))
    if (s[4] === 'logs' && s[5]) return json(await heartbeatClient.getLog(create(HeartbeatLogRequestSchema, { teamId, agentId, logId: s[5] }), callOptions))
  }
  if (s[2] === 'trigger') return json(await heartbeatClient.triggerTeam(create(HeartbeatTeamMutationRequestSchema, { teamId, body, query }), callOptions))
  if (s[2] === 'execution-status') return json(await heartbeatClient.getTeamExecutionStatus(create(HeartbeatTeamRequestSchema, { teamId }), callOptions))
  if (s[2] === 'queue' && s[3] === 'running') return json(await heartbeatClient.clearTeamQueueRunning(create(HeartbeatMemberMutationRequestSchema, { teamId, agentId: s[4], body, query }), callOptions))
  if (s[2] === 'members' && s[3] && s[4]) {
    const member = create(HeartbeatMemberRequestSchema, { teamId, agentId: s[3] })
    const mutation = create(HeartbeatMemberMutationRequestSchema, { teamId, agentId: s[3], body, query })
    if (s[4] === 'responsibilities') return method === 'GET' ? json(await heartbeatClient.getResponsibilities(member, callOptions)) : json(await heartbeatClient.setResponsibilities(mutation, callOptions))
    if (s[4] === 'heartbeat-instructions') return method === 'GET' ? json(await heartbeatClient.getHeartbeatInstructions(member, callOptions)) : json(await heartbeatClient.setHeartbeatInstructions(mutation, callOptions))
    if (s[4] === 'context') return json(await heartbeatClient.getMemberContext(member, callOptions))
    if (s[4] === 'handoff') return method === 'GET' ? json(await heartbeatClient.getLastHandoff(member, callOptions)) : json(await heartbeatClient.clearLastHandoff(mutation, callOptions))
  }
  if (s[2] === 'handoff-history') return method === 'GET' ? json(await heartbeatClient.getHandoffHistory(create(HeartbeatTeamQueryRequestSchema, { teamId, query }), callOptions)) : json(await heartbeatClient.clearHandoffHistory(create(HeartbeatTeamMutationRequestSchema, { teamId, body, query }), callOptions))
  if (s[2] === 'tasks' && s.length === 3) return method === 'GET' ? json(await heartbeatClient.getTaskBoard(create(HeartbeatTeamQueryRequestSchema, { teamId, query }), callOptions)) : json(await heartbeatClient.addTask(create(HeartbeatTeamMutationRequestSchema, { teamId, body, query }), callOptions))
  if (s[2] === 'tasks' && s[3]) { const request = create(HeartbeatTaskMutationRequestSchema, { teamId, taskId: s[3], body }); return method === 'DELETE' ? json(await heartbeatClient.deleteTask(request, callOptions)) : json(await heartbeatClient.updateTask(request, callOptions)) }
  if (s[2] === 'bugs' && s[3] === 'capture') return json(await heartbeatClient.captureBug(create(HeartbeatTeamMutationRequestSchema, { teamId, body, query }), callOptions))
  if (s[2] === 'bugs' && s[4] === 'capture') return json(await heartbeatClient.repairBug(create(HeartbeatBugMutationRequestSchema, { teamId, draftId: s[3], body }), callOptions))
  if (s[2] === 'retention') return json(await heartbeatClient.getRetention(create(HeartbeatTeamRequestSchema, { teamId }), callOptions))
  if (s[2] === 'prune') return json(await heartbeatClient.pruneSharedState(create(HeartbeatTeamMutationRequestSchema, { teamId, body, query }), callOptions))
  if (s[2] === 'prompt-matrix') return json(await heartbeatClient.previewPromptMatrix(create(HeartbeatTeamQueryRequestSchema, { teamId, query }), callOptions))
  return { handled: false }
}

function plainValue(value: Value | undefined): JsonValue | undefined {
  return value === undefined ? undefined : toJson(ValueSchema, value)
}
if (import.meta.env.MODE !== 'test') {
  console.log('[prompt-manager api] API_BASE resolved to:', API_BASE)
}

/**
 * Static folder definitions.
 * The API uses folder-based organization.
 * Folders determine git behavior, not editability:
 * - core: Important skills (git-tracked)
 * - local: Personal skills (gitignored)
 * - drafts: Work in progress skills
 */
export const FOLDERS: Folder[] = [
  {
    id: 'core',
    name: 'Core',
    description: 'Important skills (git-tracked)',
    icon: 'shield',
    skillCount: 0,  // Updated dynamically
  },
  {
    id: 'local',
    name: 'Local',
    description: 'Personal skills (gitignored)',
    icon: 'folder',
    skillCount: 0,
  },
  {
    id: 'drafts',
    name: 'Drafts',
    description: 'Work in progress skills',
    icon: 'edit',
    skillCount: 0,
  },
  {
    id: 'scenario',
    name: 'Scenario',
    description: 'Owned by a scenario, read from its own skills/ root',
    icon: 'package',
    skillCount: 0,
  },
]

/**
 * AI search request parameters (used for request body, not validated)
 */
export interface AISearchRequest {
  query: string
  limit?: number
  output?: 'results' | 'combined' | 'both'
  format?: DisplayFormat
  renderLimit?: number
}

/**
 * Content search request parameters.
 */
export interface ContentSearchRequest {
  query: string
  tags?: string[]
  folders?: string[]
  caseSensitive?: boolean
  wholeWord?: boolean
  regex?: boolean
  limit?: number
}

class ApiClient {
  /**
   * Make an API request with schema validation.
   *
   * @param endpoint - API endpoint path
   * @param options - Fetch options
   * @param schema - Zod schema for response validation
   * @returns Validated response data
   * @throws ValidationError if schema validation fails
   */
  private async request<T>(
    endpoint: string,
    options: RequestInit | undefined,
    schema: ZodType<T>
  ): Promise<T> {
    const migrated = await connectSlice4Request(endpoint, options)
    if (migrated.handled) {
      return parseOrThrow(schema, migrated.data, endpoint)
    }
    const url = buildApiUrl(endpoint, { baseUrl: API_BASE })
    console.log('[prompt-manager api] Fetching:', url)
    try {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      }
      if (options?.headers) {
        const optHeaders = options.headers as Record<string, string>
        Object.assign(headers, optHeaders)
      }
      const response = await fetch(url, {
        ...options,
        headers,
      })

      console.log('[prompt-manager api] Response status:', response.status, response.statusText)

      if (!response.ok) {
        const errorText = await response.text().catch(() => 'Unknown error')
        throw new Error(`API error: ${response.status} ${response.statusText} - ${errorText}`)
      }

      // Handle 204 No Content responses
      if (response.status === 204) {
        return {} as T
      }

      const data: unknown = await response.json()
      return parseOrThrow(schema, data, endpoint)
    } catch (error) {
      console.error('[prompt-manager api] Fetch error:', error)
      throw error
    }
  }

  // Folder methods (computed from skills, not a separate API)
  async getFolders(): Promise<Folder[]> {
    // Get all skills and compute folder counts
    const skills = await this.getSkills()
    const counts: Record<FolderType, number> = { core: 0, local: 0, drafts: 0, scenario: 0 }

    for (const skill of skills) {
      if (skill.folder in counts) {
        counts[skill.folder]++
      }
    }

    return FOLDERS.map(folder => ({
      ...folder,
      skillCount: counts[folder.id],
    }))
  }

  // Skill methods - aligned with api/skills/handlers.go
  async getSkills(filters?: SearchFilters): Promise<Skill[]> {
    const response = await skillsClient.listSkills(create(ListSkillsRequestProtoSchema, {
      tag: filters?.tag ?? '', folder: filters?.folder ?? '', modes: filters?.modes ?? [],
    }))
    // A skill whose shape this build does not recognize (for example a pack
    // added by the API after the UI shipped) must not hide every other
    // skill. Drop the unreadable record and warn instead of rejecting the list.
    return parseArrayFiltered(SkillSchema, response.skills.map(skill => toJson(SkillProtoSchema, skill)), 'SkillsService.ListSkills')
  }

  async getSkillsByFolder(folder: FolderType): Promise<Skill[]> {
    return this.getSkills({ folder })
  }

  async getSkill(id: string): Promise<Skill> {
    const response = await skillsClient.getSkill(create(GetSkillRequestProtoSchema, { id }))
    if (!response.skill) throw new Error('SkillsService.GetSkill returned no skill')
    return parseOrThrow(SkillSchema, toJson(SkillProtoSchema, response.skill), 'SkillsService.GetSkill')
  }

  async createSkill(skill: CreateSkillRequest): Promise<Skill> {
    const request = fromJson(CreateSkillRequestProtoSchema, toProtoJson(skill))
    const response = await skillsClient.createSkill(request)
    if (!response.skill) throw new Error('SkillsService.CreateSkill returned no skill')
    return parseOrThrow(SkillSchema, toJson(SkillProtoSchema, response.skill), 'SkillsService.CreateSkill')
  }

  async updateSkill(id: string, updates: UpdateSkillRequest): Promise<Skill> {
    const request = fromJson(UpdateSkillRequestProtoSchema, toProtoJson({ ...updates, id }))
    request.replaceModes = updates.modes !== undefined
    request.replaceTags = updates.tags !== undefined
    const response = await skillsClient.updateSkill(request)
    if (!response.skill) throw new Error('SkillsService.UpdateSkill returned no skill')
    return parseOrThrow(SkillSchema, toJson(SkillProtoSchema, response.skill), 'SkillsService.UpdateSkill')
  }

  async deleteSkill(id: string): Promise<void> {
    await skillsClient.deleteSkill(create(DeleteSkillRequestProtoSchema, { id }))
  }

  // Action methods - aligned with api/actions/handlers.go
  async getActions(filters?: ActionFilters): Promise<Action[]> {
    const response = await actionsClient.listActions(create(ListActionsRequestProtoSchema, {
      pack: filters?.pack ?? '', status: filters?.status ?? '', owner: filters?.owner ?? '', tag: filters?.tag ?? '',
    }))
    return parseOrThrow(ActionArraySchema, response.actions.map(action => toJson(ActionProtoSchema, action)), 'ActionsService.ListActions')
  }

  async getAction(id: string): Promise<Action> {
    const response = await actionsClient.getAction(create(GetActionRequestProtoSchema, { id }))
    if (!response.action) throw new Error('ActionsService.GetAction returned no action')
    return parseOrThrow(ActionSchema, toJson(ActionProtoSchema, response.action), 'ActionsService.GetAction')
  }

  async createAction(action: CreateActionRequest): Promise<ActionMutationResponse> {
    const contract = fromJson(ActionProtoSchema, toProtoJson(action))
    const response = await actionsClient.authorAction(create(AuthorActionRequestProtoSchema, { contract, pack: action.pack ?? '', apply: true }))
    return parseOrThrow(ActionMutationResponseSchema, {
      action: response.rendered ? toJson(ActionProtoSchema, response.rendered) : undefined,
      validation: response.validation ? toJson(ActionValidationResultProtoSchema, response.validation) : undefined,
    }, 'ActionsService.AuthorAction')
  }

  async updateAction(id: string, updates: UpdateActionRequest): Promise<ActionMutationResponse> {
    const action = fromJson(ActionProtoSchema, toProtoJson({ ...updates, id }))
    const response = await actionsClient.updateAction(create(UpdateActionRequestProtoSchema, { id, action }))
    return parseOrThrow(ActionMutationResponseSchema, {
      action: response.action ? toJson(ActionProtoSchema, response.action) : undefined,
      validation: response.validation ? toJson(ActionValidationResultProtoSchema, response.validation) : undefined,
    }, 'ActionsService.UpdateAction')
  }

  async deleteAction(id: string, hard = false): Promise<void> {
    await actionsClient.deleteAction(create(DeleteActionRequestProtoSchema, { id, hard }))
  }

  async validateAction(id: string): Promise<ActionValidationResponse> {
    const response = await actionsClient.validateAction(create(ValidateActionRequestProtoSchema, { id }))
    if (!response.validation) throw new Error('ActionsService.ValidateAction returned no validation')
    return parseOrThrow(ActionValidationResponseSchema, toJson(ActionValidationResultProtoSchema, response.validation), 'ActionsService.ValidateAction')
  }

  async runAction(id: string, request: ActionRunRequest): Promise<ActionRunResponse> {
    const rpcRequest = fromJson(RunActionRequestProtoSchema, toProtoJson({ ...request, id }))
    const response = await actionsClient.runAction(rpcRequest)
    return parseOrThrow(ActionRunResponseSchema, toJson(RunActionResponseProtoSchema, response), 'ActionsService.RunAction')
  }

  // Usage tracking
  async recordUsage(id: string): Promise<UsageResponse> {
    const response = await skillsClient.recordSkillUsage(create(RecordSkillUsageRequestProtoSchema, { id }))
    return parseOrThrow(UsageResponseSchema, response, 'SkillsService.RecordSkillUsage')
  }

  async setRating(id: string, rating: number, notes?: string): Promise<RatingResponse> {
    const response = await skillsClient.rateSkill(create(RateSkillRequestProtoSchema, { id, rating, notes }))
    return parseOrThrow(RatingResponseSchema, response, 'SkillsService.RateSkill')
  }

  // Tags
  async getTags(): Promise<Tag[]> {
    const response = await tagsClient.listTags(create(ListTagsRequestProtoSchema))
    return parseOrThrow(TagArraySchema, response.tags.map(tag => toJson(TagProtoSchema, tag)), 'TagsService.ListTags')
  }

  async createTag(tag: Omit<Tag, 'id'>): Promise<Tag> {
    const response = await tagsClient.createTag(fromJson(CreateTagRequestProtoSchema, toProtoJson(tag)))
    if (!response.tag) throw new Error('TagsService.CreateTag returned no tag')
    return parseOrThrow(TagSchema, toJson(TagProtoSchema, response.tag), 'TagsService.CreateTag')
  }

  // Testing (requires Ollama)
  async testSkill(id: string, request: SkillTestRequest): Promise<SkillTestResult> {
    const response = await testingClient.runSkillTest(create(RunSkillTestRequestProtoSchema, {
      skillId: id,
      role: request.model,
      variables: request.inputVariables ?? {},
      maxTokens: request.maxTokens,
      temperature: request.temperature,
    }))
    const wire = toJson(SkillTestResponseProtoSchema, response) as Record<string, JsonValue>
    return parseOrThrow(SkillTestResultSchema, {
      id: response.testId,
      skillId: id,
      model: response.role,
      response: response.response,
      responseTime: response.responseTime,
      tokenCount: response.tokenCount,
      testedAt: wire.testedAt,
    }, 'TestingService.RunSkillTest')
  }

  async getTestHistory(id: string, limit?: number): Promise<SkillTestResult[]> {
    const response = await testingClient.listSkillTestHistory(create(ListSkillTestHistoryRequestProtoSchema, {
      skillId: id,
      limit: limit ?? 0,
    }))
    const results = response.results.map((result) => {
      const wire = toJson(SkillTestResultProtoSchema, result) as Record<string, JsonValue>
      let inputVariables: Record<string, string> | undefined
      if (result.inputVariables) {
        try {
          inputVariables = JSON.parse(result.inputVariables) as Record<string, string>
        } catch {
          inputVariables = undefined
        }
      }
      return {
        id: result.id,
        skillId: result.skillId,
        model: result.role,
        inputVariables,
        response: result.response ?? '',
        responseTime: result.responseTime ?? 0,
        tokenCount: result.tokenCount,
        rating: result.rating,
        notes: result.notes,
        testedAt: wire.testedAt,
      }
    })
    return parseOrThrow(SkillTestResultArraySchema, results, 'TestingService.ListSkillTestHistory')
  }

  async searchSkillResults(query: string, filters?: SearchFilters): Promise<SkillSearchResponse> {
    const response = await searchClient.searchSkills(create(TextSearchSkillsRequestProtoSchema, {
      query, tag: filters?.tag ?? '', folder: filters?.folder ?? '',
    }))
    return parseOrThrow(SkillSearchResponseSchema, toJson(TextSearchSkillsResponseProtoSchema, response), 'SearchService.SearchSkills')
  }

  // Search via the API. The server returns lightweight search rows; map them
  // back onto list skills for existing callers.
  async searchSkills(query: string, filters?: SearchFilters): Promise<Skill[]> {
    const [searchResponse, allSkills] = await Promise.all([
      this.searchSkillResults(query, filters),
      this.getSkills(filters),
    ])

    const skillsById = new Map(allSkills.map((skill) => [skill.id, skill]))
    return searchResponse.results
      .map((result) => skillsById.get(result.id))
      .filter((skill): skill is Skill => skill != null)
  }

  // Content Search - line-level content matches
  async searchSkillContent(request: ContentSearchRequest): Promise<ContentSearchResponse> {
    const response = await searchClient.searchSkillContent(create(SearchSkillContentRequestProtoSchema, request))
    return parseOrThrow(ContentSearchResponseSchema, toJson(SearchSkillContentResponseProtoSchema, response), 'SearchService.SearchSkillContent')
  }

  // Display skills
  async displaySkills(identifiers: string[], format: DisplayFormat = 'xml'): Promise<DisplayResponse> {
    return this.request<DisplayResponse>(
      '/skills/read',
      {
        method: 'POST',
        body: JSON.stringify({ identifiers, output: 'combined', format }),
      },
      DisplayResponseSchema
    )
  }

  // Health check
  async healthCheck(): Promise<HealthResponse> {
    return this.request<HealthResponse>('/health', undefined, HealthResponseSchema)
  }

  // AI Search - semantic search using Ollama embeddings and Qdrant
  async aiSearch(query: string, limit = 5, options?: Omit<AISearchRequest, 'query' | 'limit'>): Promise<AISearchResponse> {
    const response = await aiSearchClient.searchSkills(create(AISearchSkillsRequestProtoSchema, { query, limit, ...options }))
    return parseOrThrow(AISearchResponseSchema, toJson(AISearchSkillsResponseProtoSchema, response), 'AISearchService.SearchSkills')
  }

  async getAISearchStatus(): Promise<AISearchStatus> {
    const response = await aiSearchClient.getStatus(create(GetAISearchStatusRequestProtoSchema))
    return parseOrThrow(AISearchStatusSchema, toJson(GetAISearchStatusResponseProtoSchema, response), 'AISearchService.GetStatus')
  }

  async reindexAISearch(): Promise<AIReindexStatus> {
    const response = await aiSearchClient.reconcile(create(ReconcileRequestProtoSchema))
    return this.mapReconcileStatus(response.status)
  }

  async getAISearchReindexStatus(): Promise<AIReindexStatus> {
    const response = await aiSearchClient.getReconcileStatus(create(GetReconcileStatusRequestProtoSchema))
    return this.mapReconcileStatus(response)
  }

  async cancelAISearchReindex(): Promise<AIReindexStatus> {
    const response = await aiSearchClient.cancelReconcile(create(CancelReconcileRequestProtoSchema))
    return this.mapReconcileStatus(response)
  }

  async aiSearchAgents(query: string, limit = 5): Promise<AIAgentSearchResponse> {
    const response = await aiSearchClient.searchAgents(create(AISearchAgentsRequestProtoSchema, { query, limit }))
    return parseOrThrow(AIAgentSearchResponseSchema, toJson(AISearchAgentsResponseProtoSchema, response), 'AISearchService.SearchAgents')
  }

  async aiSearchActions(query: string, limit = 5): Promise<AIActionSearchResponse> {
    const response = await aiSearchClient.searchActions(create(AISearchActionsRequestProtoSchema, { query, limit }))
    return parseOrThrow(AIActionSearchResponseSchema, toJson(AISearchActionsResponseProtoSchema, response), 'AISearchService.SearchActions')
  }

  async aiSearchTeams(query: string, limit = 5): Promise<AITeamSearchResponse> {
    const response = await aiSearchClient.searchTeams(create(AISearchTeamsRequestProtoSchema, { query, limit }))
    return parseOrThrow(AITeamSearchResponseSchema, toJson(AISearchTeamsResponseProtoSchema, response), 'AISearchService.SearchTeams')
  }

  // Unified discover - topic + skill search with budgeting
  async discover(
    queries: string[],
    complexity?: string,
    limit = 10,
    type?: 'skill' | 'action' | 'all'
  ): Promise<DiscoverResponse> {
    const response = await discoveryClient.discover(create(DiscoverRequestProtoSchema, {
      queries, complexity: complexity ?? '', limit, type: type ?? '',
    }))
    return parseOrThrow(DiscoverResponseSchema, toJson(DiscoverResponseProtoSchema, response), 'DiscoveryService.Discover')
  }

  private mapReconcileStatus(status: ReconcileStatusProto | undefined): AIReindexStatus {
    if (!status) throw new Error('AISearchService returned no reconcile status')
    const raw = toJson(ReconcileStatusProtoSchema, status) as Record<string, unknown>
    const result = raw.lastResult as { collections?: Array<{ upserted?: number; deleted?: number }>; errors?: unknown[] } | undefined
    const plan = raw.lastPlan as { collections?: Array<{ unchangedCount?: number; toUpsert?: unknown[]; toDelete?: unknown[] }> } | undefined
    const indexed = result?.collections?.reduce((sum, row) => sum + (row.upserted ?? 0), 0) ?? 0
    const skipped = plan?.collections?.reduce((sum, row) => sum + (row.unchangedCount ?? 0), 0) ?? 0
    const total = plan?.collections?.reduce((sum, row) => sum + (row.unchangedCount ?? 0) + (row.toUpsert?.length ?? 0) + (row.toDelete?.length ?? 0), 0) ?? 0
    return parseOrThrow(AIReindexStatusSchema, {
      running: raw.running ?? false,
      startedAt: raw.startedAt || undefined,
      finishedAt: raw.finishedAt || undefined,
      indexed,
      skipped,
      errors: result?.errors?.length ?? 0,
      total,
      canceled: raw.canceled || undefined,
      error: raw.lastError || undefined,
    }, 'AISearchService.ReconcileStatus')
  }

  // Budget configuration
  async getBudgetConfig(): Promise<BudgetConfig> {
    return this.request<BudgetConfig>('/config/budgets', undefined, BudgetConfigSchema)
  }

  async setBudgetConfig(config: BudgetConfig): Promise<BudgetConfig> {
    return this.request<BudgetConfig>(
      '/config/budgets',
      {
        method: 'PUT',
        body: JSON.stringify(config),
      },
      BudgetConfigSchema
    )
  }

  // Discover filter configuration
  async getDiscoverFilterConfig(): Promise<DiscoverFilterConfig> {
    return this.request<DiscoverFilterConfig>(
      '/config/discover-filters',
      undefined,
      DiscoverFilterConfigSchema
    )
  }

  async setDiscoverFilterConfig(config: DiscoverFilterConfig): Promise<DiscoverFilterConfig> {
    return this.request<DiscoverFilterConfig>(
      '/config/discover-filters',
      {
        method: 'PUT',
        body: JSON.stringify(config),
      },
      DiscoverFilterConfigSchema
    )
  }

  // Agent methods
  async getAgents(): Promise<Agent[]> {
    const response = await agentsClient.listAgents(create(ListAgentsRequestProtoSchema))
    return parseOrThrow(AgentArraySchema, response.agents.map(agent => toJson(AgentProtoSchema, agent)), 'AgentsService.ListAgents')
  }

  async getAgent(id: string): Promise<Agent> {
    const response = await agentsClient.getAgent(create(GetAgentRequestProtoSchema, { id }))
    return parseOrThrow(AgentSchema, toJson(AgentProtoSchema, response), 'AgentsService.GetAgent')
  }

  async createAgent(agent: CreateAgentRequest): Promise<Agent> {
    const response = await agentsClient.createAgent(fromJson(CreateAgentRequestProtoSchema, toProtoJson({ agent })))
    return parseOrThrow(AgentSchema, toJson(AgentProtoSchema, response), 'AgentsService.CreateAgent')
  }

  async updateAgent(id: string, updates: UpdateAgentRequest): Promise<Agent> {
    const response = await agentsClient.updateAgent(fromJson(UpdateAgentRequestProtoSchema, toProtoJson({
      id,
      agent: updates,
      updateMask: Object.keys(updates).join(','),
    })))
    return parseOrThrow(AgentSchema, toJson(AgentProtoSchema, response), 'AgentsService.UpdateAgent')
  }

  async getAgentSoul(id: string): Promise<SoulResponse> {
    const response = await agentsClient.getSoul(create(GetSoulRequestProtoSchema, { id }))
    return parseOrThrow(SoulResponseSchema, toJson(SoulProtoSchema, response), 'AgentsService.GetSoul')
  }

  async setAgentSoul(id: string, content: string): Promise<SoulResponse> {
    const response = await agentsClient.setSoul(create(SetSoulRequestProtoSchema, { id, content }))
    return parseOrThrow(SoulResponseSchema, toJson(SoulProtoSchema, response), 'AgentsService.SetSoul')
  }

  async listAgentFiles(id: string): Promise<AgentFileListResponse> {
    const response = await agentsClient.listFiles(create(ListAgentFilesRequestProtoSchema, { id }))
    return parseOrThrow(AgentFileListResponseSchema, toJson(ListAgentFilesResponseProtoSchema, response), 'AgentsService.ListFiles')
  }

  async getAgentFileTemplates(): Promise<AgentFileTemplateListResponse> {
    const response = await templatesClient.listAgentFileTemplates(create(ListAgentFileTemplatesRequestProtoSchema))
    return parseOrThrow(AgentFileTemplateListResponseSchema, toJson(ListAgentFileTemplatesResponseProtoSchema, response), 'TemplatesService.ListAgentFileTemplates')
  }

  async getAgentFileContent(id: string, path: string): Promise<AgentFileContentResponse> {
    const response = await agentsClient.getFile(create(GetAgentFileRequestProtoSchema, { id, path }))
    return parseOrThrow(AgentFileContentResponseSchema, toJson(AgentFileContentProtoSchema, response), 'AgentsService.GetFile')
  }

  async setAgentFileContent(id: string, path: string, content: string): Promise<void> {
    await agentsClient.setFile(create(SetAgentFileRequestProtoSchema, { id, path, content }))
  }

  async createAgentFile(id: string, request: AgentFileCreateRequest): Promise<void> {
    await agentsClient.createFile(fromJson(CreateAgentFileRequestProtoSchema, toProtoJson({ id, ...request })))
  }

  async renameAgentFile(id: string, request: AgentFileRenameRequest): Promise<void> {
    await agentsClient.renameFile(create(RenameAgentFileRequestProtoSchema, { id, ...request }))
  }

  async deleteAgentFile(id: string, path: string): Promise<void> {
    await agentsClient.deleteFile(create(DeleteAgentFileRequestProtoSchema, { id, path }))
  }

  async previewPrompt(agentId: string, teamId?: string): Promise<PromptPreviewResponse> {
    return this.request<PromptPreviewResponse>(
      '/prompt-preview',
      {
        method: 'POST',
        body: JSON.stringify({ agentId, teamId }),
      },
      PromptPreviewResponseSchema
    )
  }

  async previewPromptStructured(agentId: string, teamId?: string): Promise<StructuredPromptPreviewResponse> {
    return this.request<StructuredPromptPreviewResponse>(
      '/prompt-preview-structured',
      {
        method: 'POST',
        body: JSON.stringify({ agentId, teamId }),
      },
      StructuredPromptPreviewResponseSchema
    )
  }

  async getTeamPromptMatrix(teamId: string): Promise<TeamPromptMatrixResponse> {
    return this.request<TeamPromptMatrixResponse>(
      `/teams/${encodeURIComponent(teamId)}/prompt-matrix`,
      { method: 'GET' },
      TeamPromptMatrixResponseSchema,
    )
  }

  async deleteAgent(id: string): Promise<void> {
    await agentsClient.deleteAgent(create(DeleteAgentRequestProtoSchema, { id }))
  }

  async getAgentTeams(id: string): Promise<AgentTeamsResponse> {
    const response = await agentsClient.listAgentTeams(create(ListAgentTeamsRequestProtoSchema, { id }))
    return parseOrThrow(AgentTeamsResponseSchema, toJson(ListAgentTeamsResponseProtoSchema, response), 'AgentsService.ListAgentTeams')
  }

  // Team methods
  async getTeams(): Promise<Team[]> {
    const response = await teamsClient.listTeams(create(ListTeamsRequestProtoSchema))
    return parseOrThrow(TeamArraySchema, response.teams.map(team => toJson(TeamProtoSchema, team)), 'TeamsService.ListTeams')
  }

  async getTeam(id: string): Promise<TeamDetails> {
    const response = await teamsClient.getTeam(create(GetTeamRequestProtoSchema, { id }))
    return parseOrThrow(TeamDetailsSchema, toJson(TeamDetailsProtoSchema, response), 'TeamsService.GetTeam')
  }

  async createTeam(team: CreateTeamRequest): Promise<TeamDetails> {
    const response = await teamsClient.createTeam(fromJson(CreateTeamRequestProtoSchema, toProtoJson({ team })))
    return parseOrThrow(TeamDetailsSchema, toJson(TeamDetailsProtoSchema, response), 'TeamsService.CreateTeam')
  }

  async updateTeam(id: string, updates: UpdateTeamRequest): Promise<TeamDetails> {
    const response = await teamsClient.updateTeam(fromJson(UpdateTeamRequestProtoSchema, toProtoJson({
      id,
      team: updates,
      updateMask: Object.keys(updates).join(','),
    })))
    return parseOrThrow(TeamDetailsSchema, toJson(TeamDetailsProtoSchema, response), 'TeamsService.UpdateTeam')
  }

  async deleteTeam(id: string): Promise<void> {
    await teamsClient.deleteTeam(create(DeleteTeamRequestProtoSchema, { id }))
  }

  async getTeamExclusiveMembers(teamId: string): Promise<ExclusiveMembersResponse> {
    const response = await teamsClient.getExclusiveMembers(create(GetExclusiveMembersRequestProtoSchema, { teamId }))
    return parseOrThrow(ExclusiveMembersResponseSchema, toJson(ExclusiveMembersResponseProtoSchema, response), 'TeamsService.GetExclusiveMembers')
  }

  async addTeamMember(teamId: string, request: AddMemberRequest): Promise<TeamMember> {
    const response = await teamsClient.addMember(create(AddTeamMemberRequestProtoSchema, {
      teamId,
      agentId: request.agentId,
      roles: request.roles ?? [],
    }))
    return parseOrThrow(TeamMemberSchema, toJson(TeamMemberProtoSchema, response), 'TeamsService.AddMember')
  }

  async updateTeamMember(teamId: string, agentId: string, request: UpdateMemberRequest): Promise<TeamMember> {
    const response = await teamsClient.updateMember(create(UpdateTeamMemberRequestProtoSchema, {
      teamId,
      agentId,
      roles: request.roles ?? [],
      status: request.status,
    }))
    return parseOrThrow(TeamMemberSchema, toJson(TeamMemberProtoSchema, response), 'TeamsService.UpdateMember')
  }

  async removeTeamMember(teamId: string, agentId: string): Promise<void> {
    await teamsClient.removeMember(create(RemoveTeamMemberRequestProtoSchema, { teamId, agentId }))
  }

  async getTeamRoles(teamId: string): Promise<TeamRole[]> {
    const response = await teamsClient.getRoles(create(GetTeamRolesRequestProtoSchema, { teamId }))
    return parseOrThrow(TeamRoleSchema.array(), response.roles.map(role => toJson(TeamRoleProtoSchema, role)), 'TeamsService.GetRoles')
  }

  async setTeamRoles(teamId: string, roles: TeamRole[]): Promise<TeamRole[]> {
    const response = await teamsClient.setRoles(fromJson(SetTeamRolesRequestProtoSchema, toProtoJson({ teamId, roles })))
    return parseOrThrow(TeamRoleSchema.array(), response.roles.map(role => toJson(TeamRoleProtoSchema, role)), 'TeamsService.SetRoles')
  }

  async listTeamSharedFiles(teamId: string): Promise<TeamSharedFileListResponse> {
    const response = await teamsClient.listSharedFiles(create(ListTeamSharedFilesRequestProtoSchema, { teamId }))
    return parseOrThrow(TeamSharedFileListResponseSchema, toJson(ListTeamSharedFilesResponseProtoSchema, response), 'TeamsService.ListSharedFiles')
  }

  async getTeamSharedFileContent(teamId: string, path: string): Promise<TeamSharedFileContentResponse> {
    const response = await teamsClient.getSharedFile(create(GetTeamSharedFileRequestProtoSchema, { teamId, path }))
    return parseOrThrow(TeamSharedFileContentResponseSchema, toJson(TeamSharedFileContentProtoSchema, response), 'TeamsService.GetSharedFile')
  }

  async setTeamSharedFileContent(teamId: string, path: string, content: string): Promise<void> {
    await teamsClient.setSharedFile(create(SetTeamSharedFileRequestProtoSchema, { teamId, path, content }))
  }

  async createTeamSharedFile(teamId: string, request: TeamSharedFileCreateRequest): Promise<void> {
    await teamsClient.createSharedFile(fromJson(CreateTeamSharedFileRequestProtoSchema, toProtoJson({ teamId, ...request })))
  }

  async renameTeamSharedFile(teamId: string, request: TeamSharedFileRenameRequest): Promise<void> {
    await teamsClient.renameSharedFile(create(RenameTeamSharedFileRequestProtoSchema, { teamId, ...request }))
  }

  async deleteTeamSharedFile(teamId: string, path: string): Promise<void> {
    await teamsClient.deleteSharedFile(create(DeleteTeamSharedFileRequestProtoSchema, { teamId, path }))
  }

  // Claude Code interop methods
  async listAvailableCCTeams(): Promise<AvailableCCTeam[]> {
    const response = await teamsClient.listAvailableClaudeCodeTeams(create(ListAvailableCCTeamsRequestProtoSchema))
    return parseOrThrow(AvailableCCTeamSchema.array(), response.teams, 'TeamsService.ListAvailableClaudeCodeTeams')
  }

  async importClaudeCodeTeam(teamName: string): Promise<TeamDetails> {
    const response = await teamsClient.importClaudeCodeTeam(create(ImportClaudeCodeTeamRequestProtoSchema, { teamName }))
    return parseOrThrow(TeamDetailsSchema, toJson(TeamDetailsProtoSchema, response), 'TeamsService.ImportClaudeCodeTeam')
  }

  async exportClaudeCodeTeam(teamId: string): Promise<ExportCCResponse> {
    const response = await teamsClient.exportClaudeCodeTeam(create(ExportClaudeCodeTeamRequestProtoSchema, { teamId }))
    const wire = toJson(ExportClaudeCodeTeamResponseProtoSchema, response) as Record<string, JsonValue>
    return parseOrThrow(ExportCCResponseSchema, wire.export, 'TeamsService.ExportClaudeCodeTeam')
  }

  // World scale methods
  async getWorldScale(): Promise<WorldScaleConfig> {
    const response = await worldScaleClient.getWorldScale(create(GetWorldScaleRequestProtoSchema))
    return parseOrThrow(WorldScaleConfigSchema, toJson(WorldScaleProtoSchema, response), 'WorldScaleService.GetWorldScale')
  }

  async setWorldScale(config: WorldScaleConfig): Promise<WorldScaleConfig> {
    const response = await worldScaleClient.setWorldScale(create(SetWorldScaleRequestProtoSchema, { scale: config }))
    return parseOrThrow(WorldScaleConfigSchema, toJson(WorldScaleProtoSchema, response), 'WorldScaleService.SetWorldScale')
  }

  // World seats methods
  async getWorldSeats(): Promise<WorldSeatsConfig> {
    const response = await worldSeatsClient.getWorldSeats(create(GetWorldSeatsRequestProtoSchema))
    const config = Object.fromEntries(response.groups.map(group => [
      group.furnitureType,
      group.seats.map(seat => ({
        position: [seat.position?.x ?? 0, seat.position?.y ?? 0, seat.position?.z ?? 0],
        rotation: seat.rotation,
      })),
    ]))
    return parseOrThrow(WorldSeatsConfigSchema, config, 'WorldSeatsService.GetWorldSeats')
  }

  async setWorldSeats(config: WorldSeatsConfig): Promise<WorldSeatsConfig> {
    const response = await worldSeatsClient.setWorldSeats(create(SetWorldSeatsRequestProtoSchema, {
      seats: {
        groups: Object.entries(config).map(([furnitureType, seats]) => ({
          furnitureType,
          seats: seats.map(seat => ({
            position: { x: seat.position[0], y: seat.position[1], z: seat.position[2] },
            rotation: seat.rotation,
          })),
        })),
      },
    }))
    const saved = Object.fromEntries(response.groups.map(group => [
      group.furnitureType,
      group.seats.map(seat => ({
        position: [seat.position?.x ?? 0, seat.position?.y ?? 0, seat.position?.z ?? 0],
        rotation: seat.rotation,
      })),
    ]))
    return parseOrThrow(WorldSeatsConfigSchema, saved, 'WorldSeatsService.SetWorldSeats')
  }

  // Graph methods - aligned with api/graph/handlers.go
  async getGraph(): Promise<GraphResponse> {
    return this.request<GraphResponse>('/graph', undefined, GraphResponseSchema)
  }

  async getOperatingMap(): Promise<import('@/lib/schemas').OperatingMap> {
    return this.request('/operating-models/map', undefined, OperatingMapSchema)
  }

  async getGraphNode(id: string): Promise<NodeDetailResponse> {
    return this.request<NodeDetailResponse>(
      `/graph/nodes/${encodeURIComponent(id)}`,
      undefined,
      NodeDetailResponseSchema
    )
  }

  async regenerateGraph(): Promise<RegenerateResponse> {
    return this.request<RegenerateResponse>(
      '/graph/regenerate',
      { method: 'POST' },
      RegenerateResponseSchema
    )
  }

  async getOrphanedSkills(): Promise<NodeListResponse> {
    return this.request<NodeListResponse>(
      '/graph/orphans',
      undefined,
      NodeListResponseSchema
    )
  }

  async getSkilllessAgents(): Promise<NodeListResponse> {
    return this.request<NodeListResponse>(
      '/graph/skillless',
      undefined,
      NodeListResponseSchema
    )
  }

  async getEmptyTeams(): Promise<NodeListResponse> {
    return this.request<NodeListResponse>(
      '/graph/empty-teams',
      undefined,
      NodeListResponseSchema
    )
  }

  async getUnaffiliatedAgents(): Promise<NodeListResponse> {
    return this.request<NodeListResponse>(
      '/graph/unaffiliated',
      undefined,
      NodeListResponseSchema
    )
  }

  async getCLIlessSkills(): Promise<NodeListResponse> {
    // No dedicated backend endpoint - compute from full graph client-side.
    // Fetch the full graph and filter skills that have no code-usage edges.
    const idx = await this.getGraph()
    const hasCLI = new Set<string>()
    for (const e of idx.graph.edges) {
      if (e.kind === 'code-usage') hasCLI.add(e.from)
    }
    return idx.graph.nodes.filter((n) => n.type === 'skill' && !hasCLI.has(n.id))
  }

  async getPopular(limit?: number): Promise<PopularityResponse> {
    const params = new URLSearchParams()
    if (limit) params.set('limit', limit.toString())
    const qs = params.toString()
    return this.request<PopularityResponse>(
      `/graph/popular${qs ? `?${qs}` : ''}`,
      undefined,
      PopularityResponseSchema
    )
  }

  async getCircularRefs(): Promise<CircularRefResponse> {
    return this.request<CircularRefResponse>(
      '/graph/cycles',
      undefined,
      CircularRefResponseSchema
    )
  }

  async getGraphHealth(): Promise<GraphHealthResponse> {
    return this.request<GraphHealthResponse>(
      '/graph/health',
      undefined,
      GraphHealthResponseSchema
    )
  }

  async getGraphHealthConfig(): Promise<GraphHealthConfigResponse> {
    return this.request<GraphHealthConfigResponse>(
      '/graph/health-config',
      undefined,
      GraphHealthConfigResponseSchema
    )
  }

  async setGraphHealthConfig(config: GraphHealthConfigResponse): Promise<GraphHealthConfigResponse> {
    return this.request<GraphHealthConfigResponse>(
      '/graph/health-config',
      {
        method: 'PUT',
        body: JSON.stringify(config),
      },
      GraphHealthConfigResponseSchema
    )
  }

  async getNodeHealth(id: string): Promise<NodeHealthResponse | null> {
    // No per-node health endpoint - filter from full health scores
    const scores = await this.getGraphHealth()
    return scores.find((s) => s.nodeId === id) ?? null
  }

  // Topic methods
  async getTopics(): Promise<Topic[]> {
    const response = await topicsClient.listTopics(create(ListTopicsRequestProtoSchema))
    return parseOrThrow(TopicArraySchema, response.topics.map(topic => toJson(TopicProtoSchema, topic)), 'TopicsService.ListTopics')
  }

  async getTopic(id: string): Promise<Topic> {
    const response = await topicsClient.getTopic(create(GetTopicRequestProtoSchema, { id }))
    return parseOrThrow(TopicSchema, toJson(TopicProtoSchema, response), 'TopicsService.GetTopic')
  }

  async createTopic(topic: CreateTopicRequest): Promise<Topic> {
    const response = await topicsClient.createTopic(fromJson(CreateTopicRequestProtoSchema, toProtoJson({ topic })))
    return parseOrThrow(TopicSchema, toJson(TopicProtoSchema, response), 'TopicsService.CreateTopic')
  }

  async updateTopic(id: string, updates: UpdateTopicRequest): Promise<Topic> {
    const response = await topicsClient.updateTopic(fromJson(UpdateTopicRequestProtoSchema, toProtoJson({
      id,
      topic: updates,
      updateMask: Object.keys(updates).join(','),
    })))
    return parseOrThrow(TopicSchema, toJson(TopicProtoSchema, response), 'TopicsService.UpdateTopic')
  }

  async deleteTopic(id: string): Promise<void> {
    await topicsClient.deleteTopic(create(DeleteTopicRequestProtoSchema, { id }))
  }

  async getAccumulatedSkills(id: string): Promise<AccumulatedSkillsResponse> {
    const response = await topicsClient.getAccumulatedSkills(create(GetAccumulatedSkillsRequestProtoSchema, { id }))
    return parseOrThrow(AccumulatedSkillsResponseSchema, toJson(AccumulatedSkillsResponseProtoSchema, response), 'TopicsService.GetAccumulatedSkills')
  }

  async matchTopics(queries: string[], limit?: number): Promise<TopicMatchResponse> {
    const response = await topicsClient.matchTopics(create(MatchTopicsRequestProtoSchema, { queries, limit: limit ?? 0 }))
    return parseOrThrow(TopicMatchResponseSchema, toJson(MatchTopicsResponseProtoSchema, response), 'TopicsService.MatchTopics')
  }

  // Version history methods
  async getSkillVersions(skillId: string): Promise<VersionsResponse> {
    const response = await skillsClient.listSkillVersions(create(ListSkillVersionsRequestProtoSchema, { id: skillId }))
    return parseOrThrow(VersionsResponseSchema, response, 'SkillsService.ListSkillVersions')
  }

  async revertSkillVersion(skillId: string, version: number): Promise<RevertResponse> {
    const response = await skillsClient.revertSkill(create(RevertSkillRequestProtoSchema, { id: skillId, version }))
    return parseOrThrow(RevertResponseSchema, response, 'SkillsService.RevertSkill')
  }

  // Variant methods - aligned with api/skills/variant_handlers.go
  async listVariants(skillId: string): Promise<Variant[]> {
    const response = await skillsClient.listSkillVariants(create(ListSkillVariantsRequestProtoSchema, { skillId }))
    return parseOrThrow(VariantArraySchema, response.variants.map(variant => toJson(SkillVariantProtoSchema, variant)), 'SkillsService.ListSkillVariants')
  }

  async createVariant(skillId: string, req: CreateVariantRequest): Promise<Variant> {
    const request = fromJson(CreateSkillVariantRequestProtoSchema, toProtoJson({ ...req, skillId }))
    const response = await skillsClient.createSkillVariant(request)
    if (!response.variant) throw new Error('SkillsService.CreateSkillVariant returned no variant')
    return parseOrThrow(VariantSchema, toJson(SkillVariantProtoSchema, response.variant), 'SkillsService.CreateSkillVariant')
  }

  async deleteVariant(skillId: string, variantId: string): Promise<void> {
    await skillsClient.deleteSkillVariant(create(DeleteSkillVariantRequestProtoSchema, { skillId, variantId }))
  }

  // Experiment methods - aligned with api/skills/experiment_handlers.go
  async listExperimentsBySkill(skillId: string): Promise<Experiment[]> {
    return this.request<Experiment[]>(
      `/skills/${encodeURIComponent(skillId)}/experiments`,
      undefined,
      ExperimentArraySchema
    )
  }

  async getExperiment(experimentId: string): Promise<Experiment> {
    return this.request<Experiment>(
      `/experiments/${encodeURIComponent(experimentId)}`,
      undefined,
      ExperimentSchema
    )
  }

  async createExperiment(req: CreateExperimentRequest): Promise<Experiment> {
    return this.request<Experiment>(
      '/experiments',
      {
        method: 'POST',
        body: JSON.stringify(req),
      },
      ExperimentSchema
    )
  }

  async startExperiment(experimentId: string): Promise<Experiment> {
    return this.request<Experiment>(
      `/experiments/${encodeURIComponent(experimentId)}/start`,
      { method: 'POST' },
      ExperimentSchema
    )
  }

  async concludeExperiment(experimentId: string, req: ConcludeExperimentRequest): Promise<Experiment> {
    return this.request<Experiment>(
      `/experiments/${encodeURIComponent(experimentId)}/conclude`,
      {
        method: 'POST',
        body: JSON.stringify(req),
      },
      ExperimentSchema
    )
  }
}

export const api = new ApiClient()

// -----------------------------------------------------------------------------
// Link Preview API Functions
// -----------------------------------------------------------------------------

/**
 * Fetch OpenGraph metadata preview for a URL.
 * @param url - The URL to fetch preview for
 * @returns Preview data or null if unavailable
 */
export async function fetchLinkPreview(url: string): Promise<LinkPreviewData | null> {
  const response = await metadataClient.fetchOpenGraph(create(FetchOpenGraphRequestProtoSchema, { url }))
  const wire = toJson(OpenGraphMetadataProtoSchema, response) as Record<string, JsonValue>
  const normalized: Record<string, JsonValue> = { ...wire, site_name: response.siteName }
  delete normalized.siteName
  return parseOrThrow(LinkPreviewDataSchema, normalized, 'MetadataService.FetchOpenGraph')
}
