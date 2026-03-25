/**
 * Topic-related Zod schemas for runtime validation.
 *
 * These schemas match the Go API response shapes from api/topics/models.go.
 * They provide sensible defaults for optional array fields to prevent
 * "undefined is not iterable" crashes.
 *
 * IMPORTANT: Go nil slices serialize to JSON `null`, not `[]`.
 * Array fields must use `.nullable().optional().transform(val => val ?? [])`
 * to handle both null (Go nil) and undefined (missing field).
 */

import { z } from 'zod'

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
 * Topic schema matching the API's Response type.
 *
 * NOTE: Array fields use nullableStringArray to handle Go's nil slice -> null
 * serialization. This ensures arrays are never null/undefined in TypeScript.
 */
export const TopicSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().nullable().optional().transform((val) => val ?? ''),
  parentTopicId: z.string().nullable().optional(),
  skills: nullableStringArray,
  icon: z.string().nullable().optional(),
  content: z.string().nullable().optional().transform((val) => val ?? ''),
  status: z.string(),
  createdAt: z.string(),
  updatedAt: z.string(),
})

export type Topic = z.infer<typeof TopicSchema>

/**
 * Array of topics with validation.
 */
export const TopicArraySchema = z.array(TopicSchema)

/**
 * CreateTopicRequest schema matching the API's CreateRequest type.
 */
export const CreateTopicRequestSchema = z.object({
  id: z.string().optional(),
  name: z.string().min(1, 'Name is required'),
  description: z.string().optional(),
  parentTopicId: z.string().nullable().optional(),
  skills: z.array(z.string()).optional(),
  icon: z.string().optional(),
  content: z.string().optional(),
})

export type CreateTopicRequest = z.infer<typeof CreateTopicRequestSchema>

/**
 * UpdateTopicRequest schema matching the API's UpdateRequest type.
 * All fields are optional since this is a partial update.
 */
export const UpdateTopicRequestSchema = z.object({
  name: z.string().optional(),
  description: z.string().optional(),
  parentTopicId: z.string().nullable().optional(),
  skills: z.array(z.string()).optional(),
  icon: z.string().optional(),
  content: z.string().optional(),
  status: z.string().optional(),
})

export type UpdateTopicRequest = z.infer<typeof UpdateTopicRequestSchema>

/**
 * AccumulatedSkills response from GET /topics/{id}/skills.
 */
export const AccumulatedSkillsResponseSchema = z.object({
  topicId: z.string(),
  ancestry: z.array(z.string()).nullable().optional().transform((val) => val ?? []),
  skills: nullableStringArray,
})

export type AccumulatedSkillsResponse = z.infer<typeof AccumulatedSkillsResponseSchema>

/**
 * TopicMatch response from POST /topics/match.
 */
export const TopicMatchRequestSchema = z.object({
  queries: z.array(z.string()),
  limit: z.number().optional(),
})

export type TopicMatchRequest = z.infer<typeof TopicMatchRequestSchema>

export const TopicMatchResultSchema = z.object({
  topic: TopicSchema,
  score: z.number(),
  matchedQuery: z.string().optional(),
})

export type TopicMatchResult = z.infer<typeof TopicMatchResultSchema>

export const TopicMatchResponseSchema = z.array(TopicMatchResultSchema)

export type TopicMatchResponse = z.infer<typeof TopicMatchResponseSchema>
