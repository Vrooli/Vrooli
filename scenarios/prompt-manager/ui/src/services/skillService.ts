/**
 * Skill Service - API wrapper with caching layer.
 *
 * Wraps the existing api.ts client with:
 * - 5-second cache for list operations
 * - Cache invalidation on mutations
 * - Batch update support
 * - Graceful handling of validation errors
 */

import { api } from '@/lib/api'
import { createCacheManager } from '@/lib/cache'
import { ValidationError } from '@/lib/schemas'
import type { Skill, CreateSkillRequest, UpdateSkillRequest } from '@/lib/schemas'

// Create cache for skills list
const skillsCache = createCacheManager<Skill[]>()

/**
 * Invalidate all caches. Call after mutations.
 */
export function invalidateCache(): void {
  skillsCache.invalidate()
}

/**
 * Get all skills with caching.
 *
 * @param forceRefresh - Skip cache and fetch fresh data
 * @returns Array of all skills (empty array on validation errors)
 */
export async function getSkills(forceRefresh = false): Promise<Skill[]> {
  const cached = skillsCache.getIfValid(forceRefresh)
  if (cached) {
    return cached
  }

  try {
    const data = await api.getSkills()
    skillsCache.set(data)
    return data
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn('[skillService] Invalid API response for getSkills:', error.message)
      return []
    }
    throw error
  }
}

/**
 * Get a single skill by ID.
 * Always fetches from API since the list cache doesn't include content.
 *
 * @param id - Skill ID
 * @returns The skill, or undefined if not found or invalid
 */
export async function getSkill(id: string): Promise<Skill | undefined> {
  // Always fetch from API - list cache doesn't include content
  try {
    return await api.getSkill(id)
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn(`[skillService] Invalid API response for skill ${id}:`, error.message)
      return undefined
    }
    console.error(`[skillService] Failed to get skill ${id}:`, error)
    return undefined
  }
}

/**
 * Create a new skill.
 *
 * @param request - Create request data
 * @returns The created skill
 * @throws ValidationError if API response is invalid
 */
export async function createSkill(request: CreateSkillRequest): Promise<Skill> {
  const skill = await api.createSkill(request)
  invalidateCache()
  return skill
}

/**
 * Update a single skill.
 *
 * @param id - Skill ID to update
 * @param updates - Fields to update
 * @returns The updated skill
 * @throws ValidationError if API response is invalid
 */
export async function updateSkill(id: string, updates: UpdateSkillRequest): Promise<Skill> {
  const skill = await api.updateSkill(id, updates)
  invalidateCache()
  return skill
}

/**
 * Batch update multiple skills.
 *
 * @param updates - Map of skill ID to update request
 * @returns Map of skill ID to updated skill (or error)
 */
export async function updateSkills(
  updates: Map<string, UpdateSkillRequest>
): Promise<Map<string, Skill | Error>> {
  const results = new Map<string, Skill | Error>()

  // Process updates in parallel
  const promises = Array.from(updates.entries()).map(async ([id, update]) => {
    try {
      const skill = await api.updateSkill(id, update)
      results.set(id, skill)
    } catch (error) {
      results.set(id, error instanceof Error ? error : new Error(String(error)))
    }
  })

  await Promise.all(promises)
  invalidateCache()

  return results
}

/**
 * Delete a skill.
 *
 * @param id - Skill ID to delete
 */
export async function deleteSkill(id: string): Promise<void> {
  await api.deleteSkill(id)
  invalidateCache()
}

/**
 * Search skills by query.
 * Uses cached data when available for faster results.
 *
 * @param query - Search query
 * @returns Matching skills (empty array on errors)
 */
export async function searchSkills(query: string): Promise<Skill[]> {
  try {
    return await api.searchSkills(query)
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn('[skillService] Invalid API response for searchSkills:', error.message)
      return []
    }
    const cached = skillsCache.getIfValid()
    if (!cached) throw error

    const lowerQuery = query.toLowerCase()
    return cached.filter(
      (p) =>
        p.name.toLowerCase().includes(lowerQuery) ||
        p.description.toLowerCase().includes(lowerQuery) ||
        p.content.toLowerCase().includes(lowerQuery) ||
        p.tags.some((t) => t.toLowerCase().includes(lowerQuery)) ||
        p.modes.some((m) => m.toLowerCase().includes(lowerQuery))
    )
  }
}

/**
 * Search skills and return matching IDs in server relevance order.
 * Used by the sidebar quick search, which already owns the loaded skill list.
 */
export async function searchSkillIds(query: string): Promise<string[]> {
  const response = await api.searchSkillResults(query)
  return response.results.map((result) => result.id)
}

/**
 * Content search - line-level content matches.
 *
 * @param query - Search query
 * @param options - Content search options
 * @returns Content search response
 */
export async function searchSkillContent(
  query: string,
  options?: {
    tags?: string[]
    folders?: string[]
    caseSensitive?: boolean
    wholeWord?: boolean
    regex?: boolean
    limit?: number
  }
) {
  return api.searchSkillContent({
    query,
    ...options,
  })
}

/**
 * Get all unique tags from skills.
 *
 * @returns Array of unique tags sorted alphabetically
 */
export async function getAllTags(): Promise<string[]> {
  const skills = await getSkills()
  const tags = new Set<string>()
  for (const skill of skills) {
    for (const tag of skill.tags) {
      tags.add(tag)
    }
  }
  return Array.from(tags).sort()
}

/**
 * AI Search - semantic search using embeddings.
 * Falls back to text search if AI resources unavailable.
 *
 * @param query - Search query
 * @param limit - Maximum results (default 5)
 * @returns AI search response with results and method used
 */
export async function aiSearch(query: string, limit = 5) {
  return api.aiSearch(query, limit)
}

/**
 * Get AI search system status.
 *
 * @returns Status of Ollama, Qdrant, and index count
 */
export async function getAISearchStatus() {
  return api.getAISearchStatus()
}

/**
 * Rebuild AI search index from all skills.
 *
 * @returns Reindex results (indexed, skipped, errors)
 */
export async function reindexAISearch() {
  return api.reindexAISearch()
}

/**
 * Get AI search reindex job status.
 */
export async function getAISearchReindexStatus() {
  return api.getAISearchReindexStatus()
}

/**
 * Cancel active AI search reindex job.
 */
export async function cancelAISearchReindex() {
  return api.cancelAISearchReindex()
}
