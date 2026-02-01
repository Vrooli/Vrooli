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
 * AgentAppearance represents visual appearance for 3D UI.
 */
export const AgentAppearanceSchema = z.object({
  body: HexColorSchema,
  head: HexColorSchema,
  accent: HexColorSchema,
})

export type AgentAppearance = z.infer<typeof AgentAppearanceSchema>

/**
 * Agent schema matching the API's Agent type.
 *
 * NOTE: skills array uses nullableStringArray to handle Go's nil slice → null
 * serialization. This ensures arrays are never null/undefined in TypeScript.
 */
export const AgentSchema = z.object({
  id: z.string(),
  displayName: z.string(),
  status: AgentStatusSchema,
  appearance: AgentAppearanceSchema.nullable().optional(),
  skills: nullableStringArray,
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
  appearance: AgentAppearanceSchema.optional(),
  skills: z.array(z.string()).optional(),
})

export type CreateAgentRequest = z.infer<typeof CreateAgentRequestSchema>

/**
 * UpdateAgentRequest schema matching the API's UpdateRequest type.
 * All fields are optional since this is a partial update.
 */
export const UpdateAgentRequestSchema = z.object({
  displayName: z.string().min(1).max(100).optional(),
  status: AgentStatusSchema.optional(),
  appearance: AgentAppearanceSchema.optional(),
  skills: z.array(z.string()).optional(),
})

export type UpdateAgentRequest = z.infer<typeof UpdateAgentRequestSchema>

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
