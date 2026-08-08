/**
 * Team-related Zod schemas for runtime validation.
 *
 * These schemas match the Go API response shapes from api/teams/models.go.
 *
 * IMPORTANT: Go nil slices serialize to JSON `null`, not `[]`.
 * Array fields must handle both null and undefined.
 */

import { z } from 'zod'

const nullableStringArray = z
  .array(z.string())
  .nullable()
  .optional()
  .transform((val) => val ?? [])

export const TeamMemberStatusSchema = z.enum(['active', 'inactive', 'pending'])
export type TeamMemberStatus = z.infer<typeof TeamMemberStatusSchema>

export const TeamRoleSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().optional(),
})
export type TeamRole = z.infer<typeof TeamRoleSchema>

export const TeamMemberSchema = z.object({
  agentId: z.string(),
  displayName: z.string(),
  roles: nullableStringArray,
  status: z.string(),
})
export type TeamMember = z.infer<typeof TeamMemberSchema>

export const RuntimeModeSchema = z.enum(['multi-process', 'single-process'])
export type RuntimeMode = z.infer<typeof RuntimeModeSchema>

export const CoordinationPatternSchema = z.enum(['independent', 'peer', 'leader-led'])
export type CoordinationPattern = z.infer<typeof CoordinationPatternSchema>

export const ReportingModeSchema = z.enum(['none', 'org-chart', 'leader'])
export type ReportingMode = z.infer<typeof ReportingModeSchema>

export const MessagingModeSchema = z.enum(['disabled', 'async-inbox', 'in-session'])
export type MessagingMode = z.infer<typeof MessagingModeSchema>

export const QueuePolicySchema = z.enum(['serialized', 'bounded-parallel'])
export type QueuePolicy = z.infer<typeof QueuePolicySchema>

const PathRefSchema = z.object({
  base: z.enum(['repo-root', 'team-root', 'team-shared', 'team-member', 'agent-root']).optional(),
  path: z.string().optional(),
  memberId: z.string().optional(),
  agentId: z.string().optional(),
  required: z.boolean().optional(),
  optionalReason: z.string().optional(),
})

const WriteRefSchema = PathRefSchema.extend({
  kind: z.enum(['handoff', 'knowledge', 'task', 'inbox-message']).optional(),
})

export const OperatingContractSchema = z.object({
  schemaVersion: z.literal(1),
  documents: z.object({
    planOfRecord: z.array(z.object({
      id: z.string(),
      paths: z.array(PathRefSchema),
      writePolicy: z.string(),
      consumers: z.array(z.string()).optional(),
      rationale: z.string().optional(),
      required: z.boolean().optional(),
      optionalReason: z.string().optional(),
    })).optional().default([]),
    sharedState: z.array(z.object({
      id: z.string(),
      path: PathRefSchema,
      ownerMemberId: z.string().optional(),
      kind: z.string(),
      required: z.boolean(),
      optionalReason: z.string().optional(),
    })).optional().default([]),
  }),
  knowledgeTopics: z.record(z.string(), z.object({
    ownerMemberId: z.string(),
    retention: z.string().optional(),
  })),
  members: z.record(z.string(), z.object({
    lane: z.string(),
    allowedWrites: z.array(WriteRefSchema).optional(),
    forbiddenWrites: z.array(WriteRefSchema).optional(),
    safetyCriticalRules: z.array(z.string()).optional(),
    readOnlyModeBehavior: z.object({
      stillWriteKnowledge: z.boolean(),
      stillWriteHandoff: z.boolean(),
    }),
    taskParameters: z.record(z.string(), z.unknown()).optional(),
  })),
})
export type OperatingContract = z.infer<typeof OperatingContractSchema>

export const CoordinationCapabilitiesSchema = z.object({
  showOrgContext: z.boolean(),
  injectInbox: z.boolean(),
  allowPeerTriggers: z.boolean(),
  showTaskBoardGuidance: z.boolean(),
  showKnowledgeLogGuidance: z.boolean(),
  requireHandoff: z.boolean(),
})
export type CoordinationCapabilities = z.infer<typeof CoordinationCapabilitiesSchema>

export const RuntimeSchema = z.object({
  mode: RuntimeModeSchema,
})
export type Runtime = z.infer<typeof RuntimeSchema>

export const CoordinationSchema = z.object({
  pattern: CoordinationPatternSchema,
  leadAgentId: z.string().optional(),
  reportingMode: ReportingModeSchema,
  messagingMode: MessagingModeSchema,
  capabilities: CoordinationCapabilitiesSchema,
})
export type Coordination = z.infer<typeof CoordinationSchema>

export const ExecutionSchema = z.object({
  queuePolicy: QueuePolicySchema,
  maxConcurrentRuns: z.number().int().min(1),
})
export type Execution = z.infer<typeof ExecutionSchema>

export const TeamSchema = z.object({
  id: z.string(),
  displayName: z.string(),
  mission: z.string().optional(),
  enabled: z.boolean().optional().default(false),
  runtime: RuntimeSchema,
  coordination: CoordinationSchema,
  execution: ExecutionSchema,
  operatingContract: OperatingContractSchema,
  memberCount: z.number().int(),
  createdAt: z.string(),
  updatedAt: z.string(),
})
export type Team = z.infer<typeof TeamSchema>

export const TeamArraySchema = z.array(TeamSchema)

export const TeamDetailsSchema = TeamSchema.extend({
  roles: z.array(TeamRoleSchema).nullable().optional().transform((val) => val ?? []),
  members: z.array(TeamMemberSchema).nullable().optional().transform((val) => val ?? []),
})
export type TeamDetails = z.infer<typeof TeamDetailsSchema>

export const CreateTeamRequestSchema = z.object({
  id: z.string().optional(),
  displayName: z.string().min(1, 'Display name is required').max(100, 'Display name must be 100 characters or less'),
  mission: z.string().max(500).optional(),
  runtime: RuntimeSchema,
  coordination: CoordinationSchema,
  execution: ExecutionSchema,
  operatingContract: OperatingContractSchema,
})
export type CreateTeamRequest = z.infer<typeof CreateTeamRequestSchema>

export const UpdateTeamRequestSchema = z.object({
  displayName: z.string().min(1).max(100).optional(),
  mission: z.string().max(500).optional(),
  enabled: z.boolean().optional(),
  runtime: RuntimeSchema.optional(),
  coordination: CoordinationSchema.optional(),
  execution: ExecutionSchema.optional(),
  operatingContract: OperatingContractSchema.optional(),
})
export type UpdateTeamRequest = z.infer<typeof UpdateTeamRequestSchema>

export const AddMemberRequestSchema = z.object({
  agentId: z.string().min(1, 'Agent ID is required'),
  roles: z.array(z.string()).optional(),
})
export type AddMemberRequest = z.infer<typeof AddMemberRequestSchema>

export const UpdateMemberRequestSchema = z.object({
  roles: z.array(z.string()).optional(),
  status: z.string().optional(),
})
export type UpdateMemberRequest = z.infer<typeof UpdateMemberRequestSchema>

export const SetRolesRequestSchema = z.object({
  roles: z.array(TeamRoleSchema),
})
export type SetRolesRequest = z.infer<typeof SetRolesRequestSchema>

export const TeamSharedFileEntrySchema = z.object({
  path: z.string(),
  isDir: z.boolean(),
  size: z.number().int().nonnegative().optional(),
})
export type TeamSharedFileEntry = z.infer<typeof TeamSharedFileEntrySchema>

export const TeamSharedFileListResponseSchema = z.object({
  teamId: z.string(),
  files: z.array(TeamSharedFileEntrySchema),
})
export type TeamSharedFileListResponse = z.infer<typeof TeamSharedFileListResponseSchema>

export const TeamSharedFileContentResponseSchema = z.object({
  teamId: z.string(),
  path: z.string(),
  content: z.string(),
})
export type TeamSharedFileContentResponse = z.infer<typeof TeamSharedFileContentResponseSchema>

export const TeamSharedFileWriteRequestSchema = z.object({
  content: z.string(),
})
export type TeamSharedFileWriteRequest = z.infer<typeof TeamSharedFileWriteRequestSchema>

export const TeamSharedFileCreateRequestSchema = z.object({
  path: z.string().min(1),
  content: z.string().optional(),
  isDir: z.boolean().optional(),
})
export type TeamSharedFileCreateRequest = z.infer<typeof TeamSharedFileCreateRequestSchema>

export const TeamSharedFileRenameRequestSchema = z.object({
  from: z.string().min(1),
  to: z.string().min(1),
})
export type TeamSharedFileRenameRequest = z.infer<typeof TeamSharedFileRenameRequestSchema>

export const AvailableCCTeamSchema = z.object({
  name: z.string(),
  memberCount: z.number().int(),
})
export type AvailableCCTeam = z.infer<typeof AvailableCCTeamSchema>

export const ImportCCRequestSchema = z.object({
  teamName: z.string().min(1),
})
export type ImportCCRequest = z.infer<typeof ImportCCRequestSchema>

export const ExclusiveMemberSchema = z.object({
  agentId: z.string(),
  displayName: z.string(),
})
export type ExclusiveMember = z.infer<typeof ExclusiveMemberSchema>

export const ExclusiveMembersResponseSchema = z.object({
  teamId: z.string(),
  members: z.array(ExclusiveMemberSchema).nullable().optional().transform((val) => val ?? []),
})
export type ExclusiveMembersResponse = z.infer<typeof ExclusiveMembersResponseSchema>

export const ExportCCResponseSchema = z.unknown()
export type ExportCCResponse = unknown

// Keep these preset defaults aligned with api/teamconfig/config.go.
export const DEFAULT_INDEPENDENT_CAPABILITIES: CoordinationCapabilities = {
  showOrgContext: false,
  injectInbox: false,
  allowPeerTriggers: false,
  showTaskBoardGuidance: true,
  showKnowledgeLogGuidance: true,
  requireHandoff: true,
}

export const DEFAULT_PEER_CAPABILITIES: CoordinationCapabilities = {
  showOrgContext: true,
  injectInbox: true,
  allowPeerTriggers: true,
  showTaskBoardGuidance: true,
  showKnowledgeLogGuidance: true,
  requireHandoff: true,
}

export const DEFAULT_LEADER_LED_CAPABILITIES: CoordinationCapabilities = {
  showOrgContext: true,
  injectInbox: false,
  allowPeerTriggers: false,
  showTaskBoardGuidance: true,
  showKnowledgeLogGuidance: true,
  requireHandoff: true,
}

export function buildIndependentCoordination(): Coordination {
  return {
    pattern: 'independent',
    reportingMode: 'none',
    messagingMode: 'disabled',
    capabilities: { ...DEFAULT_INDEPENDENT_CAPABILITIES },
  }
}

export function buildPeerCoordination(): Coordination {
  return {
    pattern: 'peer',
    reportingMode: 'org-chart',
    messagingMode: 'async-inbox',
    capabilities: { ...DEFAULT_PEER_CAPABILITIES },
  }
}

export function buildLeaderLedCoordination(
  leadAgentId: string,
  runtimeMode: RuntimeMode = 'single-process',
): Coordination {
  return {
    pattern: 'leader-led',
    leadAgentId,
    reportingMode: 'leader',
    messagingMode: runtimeMode === 'single-process' ? 'in-session' : 'async-inbox',
    capabilities: {
      ...DEFAULT_LEADER_LED_CAPABILITIES,
      injectInbox: runtimeMode === 'multi-process',
    },
  }
}

export function buildBoundedParallelExecution(maxConcurrentRuns = 2): Execution {
  return {
    queuePolicy: 'bounded-parallel',
    maxConcurrentRuns,
  }
}

export function buildSerializedExecution(): Execution {
  return {
    queuePolicy: 'serialized',
    maxConcurrentRuns: 1,
  }
}

export function buildDefaultCreateTeamRequest(displayName: string): CreateTeamRequest {
  return {
    displayName,
    runtime: { mode: 'multi-process' },
    coordination: buildIndependentCoordination(),
    execution: buildBoundedParallelExecution(2),
    operatingContract: {
      schemaVersion: 1,
      documents: { planOfRecord: [], sharedState: [] },
      knowledgeTopics: {},
      members: {},
    },
  }
}
