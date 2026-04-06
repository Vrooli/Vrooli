/**
 * Version history Zod schemas for runtime validation.
 *
 * These schemas match the Go API response shapes from api/skills/models.go
 * (SkillVersion, VersionsResponse, RevertResponse).
 */

import { z } from 'zod'

/**
 * A single skill version entry from the history.
 */
export const SkillVersionSchema = z.object({
  version: z.number(),
  content: z.string(),
  name: z.string(),
  updatedAt: z.string(),
  createdBy: z.string().nullable().optional().transform((val) => val ?? ''),
})

export type SkillVersion = z.infer<typeof SkillVersionSchema>

/**
 * Response from GET /api/v1/skills/{id}/versions.
 */
export const VersionsResponseSchema = z.object({
  skillId: z.string(),
  current: z.number(),
  versions: z.array(SkillVersionSchema).nullable().optional().transform((val) => val ?? []),
})

export type VersionsResponse = z.infer<typeof VersionsResponseSchema>

/**
 * Response from POST /api/v1/skills/{id}/revert/{version}.
 */
export const RevertResponseSchema = z.object({
  skillId: z.string(),
  revertedTo: z.number(),
  newVersion: z.number(),
  restoredAt: z.string(),
})

export type RevertResponse = z.infer<typeof RevertResponseSchema>
