/**
 * Schema barrel export.
 *
 * All Zod schemas and their inferred types are exported from this module.
 * Import from here to get both runtime validation and TypeScript types.
 *
 * @example
 * import { SkillSchema, type Skill } from '@/lib/schemas'
 */

// Safe parsing utilities
export {
  safeParse,
  parseOrThrow,
  parseOrNull,
  parseArrayFiltered,
  ValidationError,
  type ParseResult,
} from './safeParse'

// Common schemas
export {
  FolderTypeSchema,
  HexColorSchema,
  KebabCaseIdSchema,
  TimestampSchema,
  type FolderType,
} from './common.schema'

// Skill schemas
export {
  SkillSchema,
  SkillArraySchema,
  CreateSkillRequestSchema,
  UpdateSkillRequestSchema,
  TagSchema,
  TagArraySchema,
  SkillTestRequestSchema,
  SkillTestResultSchema,
  SkillTestResultArraySchema,
  UsageResponseSchema,
  RatingResponseSchema,
  HealthResponseSchema,
  SyncResponseSchema,
  DisplayFormatSchema,
  DisplayResponseSchema,
  type Skill,
  type CreateSkillRequest,
  type UpdateSkillRequest,
  type Tag,
  type SkillTestRequest,
  type SkillTestResult,
  type UsageResponse,
  type RatingResponse,
  type HealthResponse,
  type SyncResponse,
  type DisplayFormat,
  type DisplayResponse,
} from './skill.schema'

// Agent schemas
export {
  AgentStatusSchema,
  AgentAppearanceSchema,
  AgentCapabilitySchema,
  AgentCapabilitiesSchema,
  ConnectorTypeSchema,
  AgentConnectorSchema,
  AgentHeartbeatSchema,
  AgentSchema,
  AgentArraySchema,
  CreateAgentRequestSchema,
  UpdateAgentRequestSchema,
  SoulRequestSchema,
  SoulResponseSchema,
  AgentFileEntrySchema,
  AgentFileListResponseSchema,
  AgentFileContentResponseSchema,
  AgentFileWriteRequestSchema,
  AgentFileCreateRequestSchema,
  AgentFileRenameRequestSchema,
  DEFAULT_AGENT_COLORS,
  type AgentStatus,
  type AgentAppearance,
  type AgentCapability,
  type AgentCapabilities,
  type ConnectorType,
  type AgentConnector,
  type AgentHeartbeat,
  type Agent,
  type CreateAgentRequest,
  type UpdateAgentRequest,
  type SoulRequest,
  type SoulResponse,
  type AgentFileEntry,
  type AgentFileListResponse,
  type AgentFileContentResponse,
  type AgentFileWriteRequest,
  type AgentFileCreateRequest,
  type AgentFileRenameRequest,
} from './agent.schema'

// Search schemas
export {
  AISearchResultSchema,
  AISearchOutputSchema,
  SearchMethodSchema,
  AISearchResponseSchema,
  AISearchRequestSchema,
  AISearchStatusSchema,
  AIReindexStatusSchema,
  LinkPreviewDataSchema,
  type AISearchResult,
  type AISearchOutput,
  type SearchMethod,
  type AISearchResponse,
  type AISearchRequest,
  type AISearchStatus,
  type AIReindexStatus,
  type LinkPreviewData,
} from './search.schema'

// Team schemas
export {
  TeamMemberStatusSchema,
  TeamRoleSchema,
  TeamMemberSchema,
  TeamSchema,
  TeamArraySchema,
  TeamDetailsSchema,
  CreateTeamRequestSchema,
  UpdateTeamRequestSchema,
  AddMemberRequestSchema,
  UpdateMemberRequestSchema,
  SetRolesRequestSchema,
  TeamSharedFileEntrySchema,
  TeamSharedFileListResponseSchema,
  TeamSharedFileContentResponseSchema,
  TeamSharedFileWriteRequestSchema,
  TeamSharedFileCreateRequestSchema,
  TeamSharedFileRenameRequestSchema,
  type TeamMemberStatus,
  type TeamRole,
  type TeamMember,
  type Team,
  type TeamDetails,
  type CreateTeamRequest,
  type UpdateTeamRequest,
  type AddMemberRequest,
  type UpdateMemberRequest,
  type SetRolesRequest,
  type TeamSharedFileEntry,
  type TeamSharedFileListResponse,
  type TeamSharedFileContentResponse,
  type TeamSharedFileWriteRequest,
  type TeamSharedFileCreateRequest,
  type TeamSharedFileRenameRequest,
} from './team.schema'

// Template schemas
export {
  AgentFileTemplateSchema,
  AgentFileTemplateListResponseSchema,
  type AgentFileTemplate,
  type AgentFileTemplateListResponse,
} from './template.schema'
