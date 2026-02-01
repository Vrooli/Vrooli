/**
 * Skill Service - API wrapper with caching layer.
 *
 * Wraps the existing api.ts client with:
 * - 5-second cache for list operations
 * - Cache invalidation on mutations
 * - Batch update support
 */

import { api } from '@/lib/api'
import { createCacheManager } from '@/lib/cache'
import type { Skill, CreateSkillRequest, UpdateSkillRequest } from '@/types'

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
 * @returns Array of all skills
 */
export async function getSkills(forceRefresh = false): Promise<Skill[]> {
  const cached = skillsCache.getIfValid(forceRefresh)
  if (cached) {
    return cached
  }

  const data = await api.getSkills()
  skillsCache.set(data)
  return data
}

/**
 * Get a single skill by ID.
 * Always fetches from API since the list cache doesn't include content.
 *
 * @param id - Skill ID
 * @returns The skill, or undefined if not found
 */
export async function getSkill(id: string): Promise<Skill | undefined> {
  // Always fetch from API - list cache doesn't include content
  try {
    return await api.getSkill(id)
  } catch (error) {
    console.error(`[skillService] Failed to get skill ${id}:`, error)
    return undefined
  }
}

/**
 * Create a new skill.
 *
 * @param request - Create request data
 * @returns The created skill
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
 * @returns Matching skills
 */
export async function searchSkills(query: string): Promise<Skill[]> {
  // Use cached data if available for instant search
  const cached = skillsCache.getIfValid()
  if (cached) {
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

  // Fallback to API search
  return api.searchSkills(query)
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
