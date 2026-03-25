/**
 * Topic Service - API wrapper with caching layer.
 *
 * Wraps the existing api.ts client with:
 * - 5-second cache for list operations
 * - Cache invalidation on mutations
 * - Graceful handling of validation errors
 */

import { api } from '@/lib/api'
import { createCacheManager } from '@/lib/cache'
import { ValidationError } from '@/lib/schemas'
import type {
  Topic,
  CreateTopicRequest,
  UpdateTopicRequest,
  AccumulatedSkillsResponse,
  TopicMatchResponse,
} from '@/lib/schemas'

// Create cache for topics list
const topicsCache = createCacheManager<Topic[]>()

/**
 * Invalidate all caches. Call after mutations.
 */
export function invalidateCache(): void {
  topicsCache.invalidate()
}

/**
 * Get all topics with caching.
 *
 * @param forceRefresh - Skip cache and fetch fresh data
 * @returns Array of all topics (empty array on validation errors)
 */
export async function getTopics(forceRefresh = false): Promise<Topic[]> {
  const cached = topicsCache.getIfValid(forceRefresh)
  if (cached) {
    return cached
  }

  try {
    const data = await api.getTopics()
    topicsCache.set(data)
    return data
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn('[topicService] Invalid API response for getTopics:', error.message)
      return []
    }
    throw error
  }
}

/**
 * Get a single topic by ID.
 *
 * @param id - Topic ID
 * @returns The topic, or undefined if not found or invalid
 */
export async function getTopic(id: string): Promise<Topic | undefined> {
  try {
    return await api.getTopic(id)
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn(`[topicService] Invalid API response for topic ${id}:`, error.message)
      return undefined
    }
    console.error(`[topicService] Failed to get topic ${id}:`, error)
    return undefined
  }
}

/**
 * Create a new topic.
 *
 * @param request - Create request data
 * @returns The created topic
 * @throws ValidationError if API response is invalid
 */
export async function createTopic(request: CreateTopicRequest): Promise<Topic> {
  const topic = await api.createTopic(request)
  invalidateCache()
  return topic
}

/**
 * Update a single topic.
 *
 * @param id - Topic ID to update
 * @param updates - Fields to update
 * @returns The updated topic
 * @throws ValidationError if API response is invalid
 */
export async function updateTopic(id: string, updates: UpdateTopicRequest): Promise<Topic> {
  const topic = await api.updateTopic(id, updates)
  invalidateCache()
  return topic
}

/**
 * Delete a topic.
 *
 * @param id - Topic ID to delete
 */
export async function deleteTopic(id: string): Promise<void> {
  await api.deleteTopic(id)
  invalidateCache()
}

/**
 * Get accumulated skills for a topic (includes inherited skills from ancestors).
 *
 * @param id - Topic ID
 * @returns Accumulated skills response with ancestry chain
 */
export async function getAccumulatedSkills(id: string): Promise<AccumulatedSkillsResponse> {
  return api.getAccumulatedSkills(id)
}

/**
 * AI-powered topic search.
 *
 * @param queries - Search queries
 * @param limit - Maximum results
 * @returns Matched topics with scores
 */
export async function matchTopics(queries: string[], limit?: number): Promise<TopicMatchResponse> {
  return api.matchTopics(queries, limit)
}

/**
 * Search topics by query (client-side filtering).
 *
 * @param query - Search query
 * @returns Matching topics (empty array on errors)
 */
export async function searchTopics(query: string): Promise<Topic[]> {
  const cached = topicsCache.getIfValid()
  const topics = cached ?? await getTopics()
  const lowerQuery = query.toLowerCase()

  return topics.filter(
    (t) =>
      t.name.toLowerCase().includes(lowerQuery) ||
      t.description.toLowerCase().includes(lowerQuery) ||
      t.skills.some((s) => s.toLowerCase().includes(lowerQuery))
  )
}
