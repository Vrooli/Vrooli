/**
 * Skill-related Zod schemas for runtime validation.
 *
 * These schemas match the Go API response shapes from api/skills/models.go.
 * They provide sensible defaults for optional array fields to prevent
 * "undefined is not iterable" crashes.
 *
 * IMPORTANT: Go nil slices serialize to JSON `null`, not `[]`.
 * Array fields must use `.nullable().optional().transform(val => val ?? [])`
 * to handle both null (Go nil) and undefined (missing field).
 */

import { z } from 'zod'
import { FolderTypeSchema } from './common.schema'

/**
 * Helper for array fields that may be null (Go nil slice) or undefined (missing).
 * Transforms both null and undefined to empty array.
 */
const nullableStringArray = z
  .array(z.string())
  .nullable()
  .optional()
  .transform((val) => val ?? [])

/**
 * Skill schema matching the API's Response type.
 *
 * NOTE: Array fields use nullableStringArray to handle Go's nil slice → null
 * serialization. This ensures arrays are never null/undefined in TypeScript.
 */
export const SkillSchema = z.object({
  id: z.string(),
  file: z.string(),
  name: z.string(),
  description: z.string().nullable().optional().transform((val) => val ?? ''),
  content: z.string().nullable().optional().transform((val) => val ?? ''),
  modes: nullableStringArray,
  tags: nullableStringArray,
  icon: z.string().nullable().optional(),
  targetToolId: z.string().nullable().optional(),
  draft: z.boolean().nullable().optional().transform((val) => val ?? false),
  folder: FolderTypeSchema,
  skillDir: z.string().nullable().optional(),    // Absolute path to skill directory
  contentPath: z.string().nullable().optional(), // Absolute path to SKILL.md file
  createdAt: z.string(),
  updatedAt: z.string(),
  usageCount: z.number(),
  lastUsed: z.string().nullable().optional(),
  effectivenessRating: z.number().nullable().optional(),
})

export type Skill = z.infer<typeof SkillSchema>

/**
 * Array of skills with validation.
 */
export const SkillArraySchema = z.array(SkillSchema)

/**
 * CreateSkillRequest schema matching the API's CreateRequest type.
 */
export const CreateSkillRequestSchema = z.object({
  id: z.string().optional(),
  name: z.string().min(1, 'Name is required'),
  description: z.string(),
  content: z.string().min(1, 'Content is required'),
  modes: z.array(z.string()).optional(),
  tags: z.array(z.string()).optional(),
  icon: z.string().optional(),
  targetToolId: z.string().nullable().optional(),
  draft: z.boolean().optional(),
  folder: FolderTypeSchema,
})

export type CreateSkillRequest = z.infer<typeof CreateSkillRequestSchema>

/**
 * UpdateSkillRequest schema matching the API's UpdateRequest type.
 * All fields are optional since this is a partial update.
 */
export const UpdateSkillRequestSchema = z.object({
  file: z.string().optional(),
  name: z.string().optional(),
  description: z.string().optional(),
  content: z.string().optional(),
  modes: z.array(z.string()).optional(),
  tags: z.array(z.string()).optional(),
  icon: z.string().optional(),
  targetToolId: z.string().nullable().optional(),
  draft: z.boolean().optional(),
  folder: FolderTypeSchema.optional(),
})

export type UpdateSkillRequest = z.infer<typeof UpdateSkillRequestSchema>

/**
 * Tag schema from the tags domain.
 */
export const TagSchema = z.object({
  id: z.string(),
  name: z.string(),
  color: z.string().optional(),
  description: z.string().optional(),
})

export type Tag = z.infer<typeof TagSchema>

/**
 * Array of tags.
 */
export const TagArraySchema = z.array(TagSchema)

/**
 * SkillTestRequest for testing skills with Ollama.
 */
export const SkillTestRequestSchema = z.object({
  model: z.string(),
  inputVariables: z.record(z.string(), z.string()).optional(),
  maxTokens: z.number().optional(),
  temperature: z.number().optional(),
})

export type SkillTestRequest = z.infer<typeof SkillTestRequestSchema>

/**
 * SkillTestResult from the testing domain.
 */
export const SkillTestResultSchema = z.object({
  id: z.string(),
  skillId: z.string(),
  model: z.string(),
  inputVariables: z.record(z.string(), z.string()).optional(),
  response: z.string(),
  responseTime: z.number(),
  tokenCount: z.number().optional(),
  rating: z.number().optional(),
  notes: z.string().optional(),
  testedAt: z.string(),
})

export type SkillTestResult = z.infer<typeof SkillTestResultSchema>

/**
 * Array of test results.
 */
export const SkillTestResultArraySchema = z.array(SkillTestResultSchema)

/**
 * UsageResponse from recording skill usage.
 */
export const UsageResponseSchema = z.object({
  status: z.string(),
  usageCount: z.number(),
  lastUsed: z.string(),
})

export type UsageResponse = z.infer<typeof UsageResponseSchema>

/**
 * RatingResponse from setting skill rating.
 */
export const RatingResponseSchema = z.object({
  status: z.string(),
  rating: z.number(),
})

export type RatingResponse = z.infer<typeof RatingResponseSchema>

/**
 * HealthResponse from the health endpoint.
 */
export const HealthResponseSchema = z.object({
  status: z.string(),
  service: z.string(),
  version: z.string(),
  readiness: z.boolean(),
  timestamp: z.string(),
  dependencies: z.record(z.string(), z.string()).optional(),
})

export type HealthResponse = z.infer<typeof HealthResponseSchema>

/**
 * SyncResponse matching the API's SyncResponse type.
 */
export const SyncResponseSchema = z.object({
  skills: SkillArraySchema,
  lastUpdated: z.string(),
  hash: z.string(),
})

export type SyncResponse = z.infer<typeof SyncResponseSchema>

/**
 * DisplayFormat for skill rendering.
 */
export const DisplayFormatSchema = z.enum(['xml', 'markdown', 'json', 'cli', 'raw'])
export type DisplayFormat = z.infer<typeof DisplayFormatSchema>

/**
 * DisplayResponse from /skills/read endpoint.
 * Combined defaults to empty string to prevent crashes when copying to clipboard.
 */
export const DisplayResponseSchema = z.object({
  combined: z.string().optional().default(''),
  skills: z.array(z.object({
    id: z.string(),
    name: z.string(),
    content: z.string(),
  })).optional().default([]),
  format: DisplayFormatSchema.optional(),
  totalTokens: z.number().optional().default(0),
  skillCount: z.number().optional().default(0),
})

export type DisplayResponse = z.infer<typeof DisplayResponseSchema>
