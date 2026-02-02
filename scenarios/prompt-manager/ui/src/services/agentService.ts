/**
 * Agent Service - Agent state machine, animations, and API functions.
 *
 * This is the new service for agents, which replaces the legacy member service.
 * For 3D animation and state machine, use memberService.ts which provides
 * shared animation utilities that work for both agents and members.
 *
 * Provides:
 * - Agent API wrapper with caching
 * - Conversion utilities between agent and member formats
 * - Graceful handling of validation errors
 */

import { api } from '@/lib/api'
import { createCacheManager } from '@/lib/cache'
import { ValidationError } from '@/lib/schemas'
import type { Agent, CreateAgentRequest, UpdateAgentRequest, EffectiveSkillsResponse } from '@/lib/schemas'

// Create cache for agents list
const agentsCache = createCacheManager<Agent[]>()

/**
 * Invalidate all caches. Call after mutations.
 */
export function invalidateCache(): void {
  agentsCache.invalidate()
}

/**
 * Get all agents with caching.
 *
 * @param forceRefresh - Skip cache and fetch fresh data
 * @returns Array of all agents (empty array on validation errors)
 */
export async function getAgents(forceRefresh = false): Promise<Agent[]> {
  const cached = agentsCache.getIfValid(forceRefresh)
  if (cached) {
    return cached
  }

  try {
    const data = await api.getAgents()
    agentsCache.set(data)
    return data
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn('[agentService] Invalid API response for getAgents:', error.message)
      return []
    }
    throw error
  }
}

/**
 * Get a single agent by ID.
 * Uses cached data if available, otherwise fetches.
 *
 * @param id - Agent ID
 * @returns The agent, or undefined if not found or invalid
 */
export async function getAgent(id: string): Promise<Agent | undefined> {
  // Try cache first
  const cached = agentsCache.getIfValid()
  if (cached) {
    const cachedAgent = cached.find((a) => a.id === id)
    if (cachedAgent) return cachedAgent
  }

  // Fetch from API
  try {
    return await api.getAgent(id)
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn(`[agentService] Invalid API response for agent ${id}:`, error.message)
      return undefined
    }
    console.error(`[agentService] Failed to get agent ${id}:`, error)
    return undefined
  }
}

/**
 * Create a new agent.
 *
 * @param request - Create request data
 * @returns The created agent
 * @throws ValidationError if API response is invalid
 */
export async function createAgent(request: CreateAgentRequest): Promise<Agent> {
  const agent = await api.createAgent(request)
  invalidateCache()
  return agent
}

/**
 * Update an agent.
 *
 * @param id - Agent ID to update
 * @param updates - Fields to update
 * @returns The updated agent
 * @throws ValidationError if API response is invalid
 */
export async function updateAgent(id: string, updates: UpdateAgentRequest): Promise<Agent> {
  const agent = await api.updateAgent(id, updates)
  invalidateCache()
  return agent
}

/**
 * Get SOUL.md content for an agent.
 */
export async function getAgentSoul(agentId: string): Promise<string> {
  const response = await api.getAgentSoul(agentId)
  return response.content ?? ''
}

/**
 * Update SOUL.md content for an agent.
 */
export async function setAgentSoul(agentId: string, content: string): Promise<void> {
  await api.setAgentSoul(agentId, content)
}

/**
 * Delete an agent.
 *
 * @param id - Agent ID to delete
 */
export async function deleteAgent(id: string): Promise<void> {
  await api.deleteAgent(id)
  invalidateCache()
}

/**
 * Get effective skills for an agent.
 *
 * @param agentId - Agent ID
 * @param teamId - Optional team context for role-based grants
 * @returns Computed skill set (empty skills on validation error)
 */
export async function getEffectiveSkills(agentId: string, teamId?: string): Promise<EffectiveSkillsResponse> {
  try {
    return await api.getEffectiveSkills(agentId, teamId)
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn(`[agentService] Invalid API response for effective skills of ${agentId}:`, error.message)
      return { agentId, skills: [] }
    }
    throw error
  }
}

// Re-export animation utilities from agentAnimationService for convenience
export {
  AgentStateMachine,
  calculateLookRotation,
  calculateIdleSway,
  calculateWaveAnimation,
  calculateCelebrationAnimation,
  easing,
  lerp,
  lerpPosition,
} from './agentAnimationService'
