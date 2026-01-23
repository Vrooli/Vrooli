/**
 * Prompt Service - API wrapper with caching layer.
 *
 * Wraps the existing api.ts client with:
 * - 5-second cache for list operations
 * - Cache invalidation on mutations
 * - Batch update support
 */

import { api } from '@/lib/api'
import type { Prompt, CreatePromptRequest, UpdatePromptRequest } from '@/types'

// Cache configuration
const CACHE_TTL_MS = 5000 // 5 seconds

// Cache state
interface CacheEntry<T> {
  data: T
  timestamp: number
}

let promptsCache: CacheEntry<Prompt[]> | null = null

/**
 * Check if cache entry is still valid.
 */
function isCacheValid<T>(entry: CacheEntry<T> | null): entry is CacheEntry<T> {
  if (!entry) return false
  return Date.now() - entry.timestamp < CACHE_TTL_MS
}

/**
 * Invalidate all caches. Call after mutations.
 */
export function invalidateCache(): void {
  promptsCache = null
}

/**
 * Get all prompts with caching.
 *
 * @param forceRefresh - Skip cache and fetch fresh data
 * @returns Array of all prompts
 */
export async function getPrompts(forceRefresh = false): Promise<Prompt[]> {
  if (!forceRefresh && isCacheValid(promptsCache)) {
    return promptsCache.data
  }

  const data = await api.getPrompts()
  promptsCache = { data, timestamp: Date.now() }
  return data
}

/**
 * Get a single prompt by ID.
 * Always fetches from API since the list cache doesn't include content.
 *
 * @param id - Prompt ID
 * @returns The prompt, or undefined if not found
 */
export async function getPrompt(id: string): Promise<Prompt | undefined> {
  // Always fetch from API - list cache doesn't include content
  try {
    return await api.getPrompt(id)
  } catch (error) {
    console.error(`[promptService] Failed to get prompt ${id}:`, error)
    return undefined
  }
}

/**
 * Create a new prompt.
 *
 * @param request - Create request data
 * @returns The created prompt
 */
export async function createPrompt(request: CreatePromptRequest): Promise<Prompt> {
  const prompt = await api.createPrompt(request)
  invalidateCache()
  return prompt
}

/**
 * Update a single prompt.
 *
 * @param id - Prompt ID to update
 * @param updates - Fields to update
 * @returns The updated prompt
 */
export async function updatePrompt(id: string, updates: UpdatePromptRequest): Promise<Prompt> {
  const prompt = await api.updatePrompt(id, updates)
  invalidateCache()
  return prompt
}

/**
 * Batch update multiple prompts.
 *
 * @param updates - Map of prompt ID to update request
 * @returns Map of prompt ID to updated prompt (or error)
 */
export async function updatePrompts(
  updates: Map<string, UpdatePromptRequest>
): Promise<Map<string, Prompt | Error>> {
  const results = new Map<string, Prompt | Error>()

  // Process updates in parallel
  const promises = Array.from(updates.entries()).map(async ([id, update]) => {
    try {
      const prompt = await api.updatePrompt(id, update)
      results.set(id, prompt)
    } catch (error) {
      results.set(id, error instanceof Error ? error : new Error(String(error)))
    }
  })

  await Promise.all(promises)
  invalidateCache()

  return results
}

/**
 * Delete a prompt.
 *
 * @param id - Prompt ID to delete
 */
export async function deletePrompt(id: string): Promise<void> {
  await api.deletePrompt(id)
  invalidateCache()
}

/**
 * Search prompts by query.
 * Uses cached data when available for faster results.
 *
 * @param query - Search query
 * @returns Matching prompts
 */
export async function searchPrompts(query: string): Promise<Prompt[]> {
  // Use cached data if available for instant search
  if (isCacheValid(promptsCache)) {
    const lowerQuery = query.toLowerCase()
    return promptsCache.data.filter(
      (p) =>
        p.name.toLowerCase().includes(lowerQuery) ||
        p.description.toLowerCase().includes(lowerQuery) ||
        p.content.toLowerCase().includes(lowerQuery) ||
        p.tags.some((t) => t.toLowerCase().includes(lowerQuery)) ||
        p.modes.some((m) => m.toLowerCase().includes(lowerQuery))
    )
  }

  // Fallback to API search
  return api.searchPrompts(query)
}

/**
 * Get all unique tags from prompts.
 *
 * @returns Array of unique tags sorted alphabetically
 */
export async function getAllTags(): Promise<string[]> {
  const prompts = await getPrompts()
  const tags = new Set<string>()
  for (const prompt of prompts) {
    for (const tag of prompt.tags) {
      tags.add(tag)
    }
  }
  return Array.from(tags).sort()
}
