/**
 * Team Service - API wrapper with caching and error handling.
 *
 * Provides:
 * - Team API wrapper with caching
 * - Graceful handling of validation errors
 * - Cache invalidation on mutations
 */

import { api } from '@/lib/api'
import { createCacheManager } from '@/lib/cache'
import { ValidationError } from '@/lib/schemas'
import type {
  Team,
  TeamDetails,
  TeamRole,
  TeamMember,
  CreateTeamRequest,
  UpdateTeamRequest,
  AddMemberRequest,
  UpdateMemberRequest,
  TeamSharedFileEntry,
  TeamSharedFileCreateRequest,
  TeamSharedFileRenameRequest,
} from '@/lib/schemas'

// Create cache for teams list
const teamsCache = createCacheManager<Team[]>()

/**
 * Invalidate all caches. Call after mutations.
 */
export function invalidateCache(): void {
  teamsCache.invalidate()
}

/**
 * Get all teams with caching.
 *
 * @param forceRefresh - Skip cache and fetch fresh data
 * @returns Array of all teams (empty array on validation errors)
 */
export async function getTeams(forceRefresh = false): Promise<Team[]> {
  const cached = teamsCache.getIfValid(forceRefresh)
  if (cached) {
    return cached
  }

  try {
    const data = await api.getTeams()
    teamsCache.set(data)
    return data
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn('[teamService] Invalid API response for getTeams:', error.message)
      return []
    }
    throw error
  }
}

/**
 * Get a single team by ID with full details.
 *
 * @param id - Team ID
 * @returns The team details, or undefined if not found or invalid
 */
export async function getTeam(id: string): Promise<TeamDetails | undefined> {
  try {
    return await api.getTeam(id)
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn(`[teamService] Invalid API response for team ${id}:`, error.message)
      return undefined
    }
    console.error(`[teamService] Failed to get team ${id}:`, error)
    return undefined
  }
}

/**
 * Create a new team.
 *
 * @param request - Create request data
 * @returns The created team with details
 * @throws ValidationError if API response is invalid
 */
export async function createTeam(request: CreateTeamRequest): Promise<TeamDetails> {
  const team = await api.createTeam(request)
  invalidateCache()
  return team
}

/**
 * Update a team.
 *
 * @param id - Team ID to update
 * @param updates - Fields to update
 * @returns The updated team with details
 * @throws ValidationError if API response is invalid
 */
export async function updateTeam(id: string, updates: UpdateTeamRequest): Promise<TeamDetails> {
  const team = await api.updateTeam(id, updates)
  invalidateCache()
  return team
}

/**
 * Delete a team.
 *
 * @param id - Team ID to delete
 */
export async function deleteTeam(id: string): Promise<void> {
  await api.deleteTeam(id)
  invalidateCache()
}

/**
 * Add a member to a team.
 *
 * @param teamId - Team ID
 * @param request - Add member request
 * @returns The added team member
 */
export async function addTeamMember(teamId: string, request: AddMemberRequest): Promise<TeamMember> {
  const member = await api.addTeamMember(teamId, request)
  invalidateCache()
  return member
}

/**
 * Update a team member.
 *
 * @param teamId - Team ID
 * @param agentId - Agent ID of the member
 * @param request - Update member request
 * @returns The updated team member
 */
export async function updateTeamMember(
  teamId: string,
  agentId: string,
  request: UpdateMemberRequest
): Promise<TeamMember> {
  const member = await api.updateTeamMember(teamId, agentId, request)
  invalidateCache()
  return member
}

/**
 * Remove a member from a team.
 *
 * @param teamId - Team ID
 * @param agentId - Agent ID of the member
 */
export async function removeTeamMember(teamId: string, agentId: string): Promise<void> {
  await api.removeTeamMember(teamId, agentId)
  invalidateCache()
}

/**
 * Get team roles.
 *
 * @param teamId - Team ID
 * @returns Array of team roles
 */
export async function getTeamRoles(teamId: string): Promise<TeamRole[]> {
  try {
    return await api.getTeamRoles(teamId)
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn(`[teamService] Invalid API response for team roles ${teamId}:`, error.message)
      return []
    }
    throw error
  }
}

/**
 * Set team roles.
 *
 * @param teamId - Team ID
 * @param roles - Array of roles to set
 * @returns The updated roles
 */
export async function setTeamRoles(teamId: string, roles: TeamRole[]): Promise<TeamRole[]> {
  const result = await api.setTeamRoles(teamId, roles)
  invalidateCache()
  return result
}

/**
 * List files in a team's shared folder.
 */
export async function listTeamSharedFiles(teamId: string): Promise<TeamSharedFileEntry[]> {
  const response = await api.listTeamSharedFiles(teamId)
  return response.files
}

/**
 * Get content for a team shared file.
 */
export async function getTeamSharedFileContent(teamId: string, path: string): Promise<string> {
  const response = await api.getTeamSharedFileContent(teamId, path)
  return response.content
}

/**
 * Update content for a team shared file.
 */
export async function setTeamSharedFileContent(teamId: string, path: string, content: string): Promise<void> {
  await api.setTeamSharedFileContent(teamId, path, content)
}

/**
 * Create a new team shared file or directory.
 */
export async function createTeamSharedFile(teamId: string, request: TeamSharedFileCreateRequest): Promise<void> {
  await api.createTeamSharedFile(teamId, request)
}

/**
 * Rename a team shared file.
 */
export async function renameTeamSharedFile(teamId: string, request: TeamSharedFileRenameRequest): Promise<void> {
  await api.renameTeamSharedFile(teamId, request)
}

/**
 * Delete a team shared file.
 */
export async function deleteTeamSharedFile(teamId: string, path: string): Promise<void> {
  await api.deleteTeamSharedFile(teamId, path)
}
