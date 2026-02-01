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
 */

import { api } from '@/lib/api'
import { createCacheManager } from '@/lib/cache'
import type { Agent, CreateAgentRequest, UpdateAgentRequest, EffectiveSkillsResponse } from '@/types/agent'

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
 * @returns Array of all agents
 */
export async function getAgents(forceRefresh = false): Promise<Agent[]> {
  const cached = agentsCache.getIfValid(forceRefresh)
  if (cached) {
    return cached
  }

  const data = await api.getAgents()
  agentsCache.set(data)
  return data
}

/**
 * Get a single agent by ID.
 * Uses cached data if available, otherwise fetches.
 *
 * @param id - Agent ID
 * @returns The agent, or undefined if not found
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
    console.error(`[agentService] Failed to get agent ${id}:`, error)
    return undefined
  }
}

/**
 * Create a new agent.
 *
 * @param request - Create request data
 * @returns The created agent
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
 */
export async function updateAgent(id: string, updates: UpdateAgentRequest): Promise<Agent> {
  const agent = await api.updateAgent(id, updates)
  invalidateCache()
  return agent
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
 * @returns Computed skill set
 */
export async function getEffectiveSkills(agentId: string, teamId?: string): Promise<EffectiveSkillsResponse> {
  return api.getEffectiveSkills(agentId, teamId)
}

// Re-export animation utilities from memberService for convenience
export {
  MemberStateMachine,
  calculateLookRotation,
  calculateIdleSway,
  calculateWaveAnimation,
  calculateCelebrationAnimation,
  easing,
  lerp,
  lerpPosition,
} from './memberService'
