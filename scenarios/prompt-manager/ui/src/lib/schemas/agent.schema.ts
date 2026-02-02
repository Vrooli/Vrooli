/**
 * Agent-related Zod schemas for runtime validation.
 *
 * These schemas match the Go API response shapes from api/agents/models.go.
 *
 * IMPORTANT: Go nil slices serialize to JSON `null`, not `[]`.
 * Array fields must handle both null and undefined.
 */

import { z } from 'zod'
import { HexColorSchema } from './common.schema'

/**
 * Helper for array fields that may be null (Go nil slice) or undefined (missing).
 */
const nullableStringArray = z
  .array(z.string())
  .nullable()
  .optional()
  .transform((val) => val ?? [])

/**
 * Agent status values.
 */
export const AgentStatusSchema = z.enum(['active', 'inactive', 'suspended'])
export type AgentStatus = z.infer<typeof AgentStatusSchema>

/**
 * Connector types for external systems.
 */
export const ConnectorTypeSchema = z.enum(['scenario', 'mcp', 'api'])
export type ConnectorType = z.infer<typeof ConnectorTypeSchema>

/**
 * AgentAppearance represents visual appearance for 3D UI.
 */
export const AgentAppearanceSchema = z.object({
  body: HexColorSchema,
  head: HexColorSchema,
  accent: HexColorSchema,
})

export type AgentAppearance = z.infer<typeof AgentAppearanceSchema>

/**
 * AgentCapability represents a single capability with verbs.
 */
export const AgentCapabilitySchema = z.object({
  capabilityId: z.string(),
  verbs: nullableStringArray,
})

export type AgentCapability = z.infer<typeof AgentCapabilitySchema>

/**
 * AgentCapabilities represents capability requirements and provisions.
 */
export const AgentCapabilitiesSchema = z.object({
  provides: z.array(AgentCapabilitySchema).nullable().optional().transform((val) => val ?? []),
  requires: z.array(AgentCapabilitySchema).nullable().optional().transform((val) => val ?? []),
})

export type AgentCapabilities = z.infer<typeof AgentCapabilitiesSchema>

/**
 * AgentConnector represents an external system connector.
 */
export const AgentConnectorSchema = z.object({
  type: ConnectorTypeSchema,
  id: z.string(),
  enabled: z.boolean().default(true),
})

export type AgentConnector = z.infer<typeof AgentConnectorSchema>

/**
 * AgentHeartbeat represents health monitoring configuration.
 */
export const AgentHeartbeatSchema = z.object({
  intervalSeconds: z.number().int().positive().optional(),
  timeoutSeconds: z.number().int().positive().optional(),
  maxMissedBeats: z.number().int().positive().optional(),
})

export type AgentHeartbeat = z.infer<typeof AgentHeartbeatSchema>

/**
 * Agent schema matching the API's Agent type.
 *
 * NOTE: skills array uses nullableStringArray to handle Go's nil slice → null
 * serialization. This ensures arrays are never null/undefined in TypeScript.
 */
export const AgentSchema = z.object({
  id: z.string(),
  displayName: z.string(),
  description: z.string().optional(),
  status: AgentStatusSchema,
  appearance: AgentAppearanceSchema.nullable().optional(),
  capabilities: AgentCapabilitiesSchema.nullable().optional(),
  connectors: z.array(AgentConnectorSchema).nullable().optional().transform((val) => val ?? []),
  defaultProfileRef: z.string().optional(),
  heartbeat: AgentHeartbeatSchema.nullable().optional(),
  tags: nullableStringArray,
  skills: nullableStringArray,
  agentDir: z.string().nullable().optional(),
  createdAt: z.string(),
  updatedAt: z.string(),
})

export type Agent = z.infer<typeof AgentSchema>

/**
 * Array of agents with validation.
 */
export const AgentArraySchema = z.array(AgentSchema)

/**
 * CreateAgentRequest schema matching the API's CreateRequest type.
 */
export const CreateAgentRequestSchema = z.object({
  id: z.string().optional(),
  displayName: z.string().min(1, 'Display name is required').max(100, 'Display name must be 100 characters or less'),
  description: z.string().max(500).optional(),
  appearance: AgentAppearanceSchema.optional(),
  capabilities: AgentCapabilitiesSchema.optional(),
  connectors: z.array(AgentConnectorSchema).optional(),
  defaultProfileRef: z.string().optional(),
  heartbeat: AgentHeartbeatSchema.optional(),
  tags: z.array(z.string()).optional(),
  skills: z.array(z.string()).optional(),
})

export type CreateAgentRequest = z.infer<typeof CreateAgentRequestSchema>

/**
 * UpdateAgentRequest schema matching the API's UpdateRequest type.
 * All fields are optional since this is a partial update.
 */
export const UpdateAgentRequestSchema = z.object({
  displayName: z.string().min(1).max(100).optional(),
  description: z.string().max(500).optional(),
  status: AgentStatusSchema.optional(),
  appearance: AgentAppearanceSchema.optional(),
  capabilities: AgentCapabilitiesSchema.optional(),
  connectors: z.array(AgentConnectorSchema).optional(),
  defaultProfileRef: z.string().optional(),
  heartbeat: AgentHeartbeatSchema.optional(),
  tags: z.array(z.string()).optional(),
  skills: z.array(z.string()).optional(),
})

export type UpdateAgentRequest = z.infer<typeof UpdateAgentRequestSchema>

/**
 * Soul request/response schemas for /agents/{id}/soul.
 */
export const SoulRequestSchema = z.object({
  content: z.string(),
})

export type SoulRequest = z.infer<typeof SoulRequestSchema>

export const SoulResponseSchema = z.object({
  agentId: z.string(),
  content: z.string(),
})

export type SoulResponse = z.infer<typeof SoulResponseSchema>

/**
 * Agent file schemas for /agents/{id}/files endpoints.
 */
export const AgentFileEntrySchema = z.object({
  path: z.string(),
  isDir: z.boolean(),
  size: z.number().int().nonnegative().optional(),
})

export type AgentFileEntry = z.infer<typeof AgentFileEntrySchema>

export const AgentFileListResponseSchema = z.object({
  agentId: z.string(),
  files: z.array(AgentFileEntrySchema),
})

export type AgentFileListResponse = z.infer<typeof AgentFileListResponseSchema>

export const AgentFileContentResponseSchema = z.object({
  agentId: z.string(),
  path: z.string(),
  content: z.string(),
})

export type AgentFileContentResponse = z.infer<typeof AgentFileContentResponseSchema>

export const AgentFileWriteRequestSchema = z.object({
  content: z.string(),
})

export type AgentFileWriteRequest = z.infer<typeof AgentFileWriteRequestSchema>

export const AgentFileCreateRequestSchema = z.object({
  path: z.string().min(1),
  content: z.string().optional(),
  isDir: z.boolean().optional(),
})

export type AgentFileCreateRequest = z.infer<typeof AgentFileCreateRequestSchema>

export const AgentFileRenameRequestSchema = z.object({
  from: z.string().min(1),
  to: z.string().min(1),
})

export type AgentFileRenameRequest = z.infer<typeof AgentFileRenameRequestSchema>

/**
 * EffectiveSkillsResponse from /agents/{id}/effective-skills.
 */
export const EffectiveSkillsResponseSchema = z.object({
  agentId: z.string(),
  teamId: z.string().nullable().optional(),
  skills: nullableStringArray,
})

export type EffectiveSkillsResponse = z.infer<typeof EffectiveSkillsResponseSchema>

/**
 * Default agent colors for new agents.
 */
export const DEFAULT_AGENT_COLORS: AgentAppearance = {
  body: '#4f46e5', // indigo-600
  head: '#818cf8', // indigo-400
  accent: '#c7d2fe', // indigo-200
}
