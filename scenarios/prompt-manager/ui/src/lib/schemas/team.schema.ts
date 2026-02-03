/**
 * Team-related Zod schemas for runtime validation.
 *
 * These schemas match the Go API response shapes from api/teams/models.go.
 *
 * IMPORTANT: Go nil slices serialize to JSON `null`, not `[]`.
 * Array fields must handle both null and undefined.
 */

import { z } from 'zod'

/**
 * Helper for array fields that may be null (Go nil slice) or undefined (missing).
 */
const nullableStringArray = z
  .array(z.string())
  .nullable()
  .optional()
  .transform((val) => val ?? [])

/**
 * Team member status values.
 */
export const TeamMemberStatusSchema = z.enum(['active', 'inactive', 'pending'])
export type TeamMemberStatus = z.infer<typeof TeamMemberStatusSchema>

/**
 * TeamRole represents a role within a team.
 */
export const TeamRoleSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().optional(),
})

export type TeamRole = z.infer<typeof TeamRoleSchema>

/**
 * TeamMember represents a member of a team.
 */
export const TeamMemberSchema = z.object({
  agentId: z.string(),
  displayName: z.string(),
  roles: nullableStringArray,
  status: z.string(),
})

export type TeamMember = z.infer<typeof TeamMemberSchema>

/**
 * Team schema matching the API's Response type (brief version).
 */
export const TeamSchema = z.object({
  id: z.string(),
  displayName: z.string(),
  mission: z.string().optional(),
  enabled: z.boolean().optional().default(false),
  memberCount: z.number().int(),
  createdAt: z.string(),
  updatedAt: z.string(),
})

export type Team = z.infer<typeof TeamSchema>

/**
 * Array of teams with validation.
 */
export const TeamArraySchema = z.array(TeamSchema)

/**
 * TeamDetails schema matching the API's TeamDetailsResponse type.
 * Extends Team with full details including roles and members.
 */
export const TeamDetailsSchema = TeamSchema.extend({
  roles: z.array(TeamRoleSchema).nullable().optional().transform((val) => val ?? []),
  members: z.array(TeamMemberSchema).nullable().optional().transform((val) => val ?? []),
})

export type TeamDetails = z.infer<typeof TeamDetailsSchema>

/**
 * CreateTeamRequest schema matching the API's CreateRequest type.
 */
export const CreateTeamRequestSchema = z.object({
  id: z.string().optional(),
  displayName: z.string().min(1, 'Display name is required').max(100, 'Display name must be 100 characters or less'),
  mission: z.string().max(500).optional(),
})

export type CreateTeamRequest = z.infer<typeof CreateTeamRequestSchema>

/**
 * UpdateTeamRequest schema matching the API's UpdateRequest type.
 * All fields are optional since this is a partial update.
 */
export const UpdateTeamRequestSchema = z.object({
  displayName: z.string().min(1).max(100).optional(),
  mission: z.string().max(500).optional(),
  enabled: z.boolean().optional(),
})

export type UpdateTeamRequest = z.infer<typeof UpdateTeamRequestSchema>

/**
 * AddMemberRequest schema matching the API's AddMemberRequest type.
 */
export const AddMemberRequestSchema = z.object({
  agentId: z.string().min(1, 'Agent ID is required'),
  roles: z.array(z.string()).optional(),
})

export type AddMemberRequest = z.infer<typeof AddMemberRequestSchema>

/**
 * UpdateMemberRequest schema matching the API's UpdateMemberRequest type.
 */
export const UpdateMemberRequestSchema = z.object({
  roles: z.array(z.string()).optional(),
  status: z.string().optional(),
})

export type UpdateMemberRequest = z.infer<typeof UpdateMemberRequestSchema>

/**
 * SetRolesRequest schema for setting team roles.
 */
export const SetRolesRequestSchema = z.object({
  roles: z.array(TeamRoleSchema),
})

export type SetRolesRequest = z.infer<typeof SetRolesRequestSchema>

/**
 * Team shared file schemas for /teams/{id}/shared/files endpoints.
 */
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
