/**
 * Graph Service - API wrapper with caching layer.
 *
 * Wraps the existing api.ts client with:
 * - 10-second cache for the full graph
 * - Cache invalidation on regeneration
 * - Graceful handling of validation errors
 */

import { api } from '@/lib/api'
import { createCacheManager } from '@/lib/cache'
import { ValidationError } from '@/lib/schemas'
import type {
  GraphResponse,
  NodeDetailResponse,
  RegenerateResponse,
  NodeListResponse,
  PopularityResponse,
  CircularRefResponse,
  GraphHealthResponse,
  NodeHealthResponse,
  GraphHealthConfigResponse,
} from '@/lib/schemas'

const graphCache = createCacheManager<GraphResponse>(10000)

export function invalidateGraphCache(): void {
  graphCache.invalidate()
}

export async function getGraph(forceRefresh = false): Promise<GraphResponse | null> {
  const cached = graphCache.getIfValid(forceRefresh)
  if (cached) return cached

  try {
    const data = await api.getGraph()
    graphCache.set(data)
    return data
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn('[graphService] Invalid API response for getGraph:', error.message)
      return null
    }
    throw error
  }
}

export async function getGraphNode(id: string): Promise<NodeDetailResponse | null> {
  try {
    return await api.getGraphNode(id)
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn(`[graphService] Invalid API response for node ${id}:`, error.message)
      return null
    }
    console.error(`[graphService] Failed to get node ${id}:`, error)
    return null
  }
}

export async function regenerateGraph(): Promise<RegenerateResponse> {
  const result = await api.regenerateGraph()
  invalidateGraphCache()
  return result
}

export async function getOrphanedSkills(): Promise<NodeListResponse> {
  try {
    return await api.getOrphanedSkills()
  } catch {
    return []
  }
}

export async function getSkilllessAgents(): Promise<NodeListResponse> {
  try {
    return await api.getSkilllessAgents()
  } catch {
    return []
  }
}

export async function getEmptyTeams(): Promise<NodeListResponse> {
  try {
    return await api.getEmptyTeams()
  } catch {
    return []
  }
}

export async function getUnaffiliatedAgents(): Promise<NodeListResponse> {
  try {
    return await api.getUnaffiliatedAgents()
  } catch {
    return []
  }
}

export async function getCLIlessSkills(): Promise<NodeListResponse> {
  try {
    return await api.getCLIlessSkills()
  } catch {
    return []
  }
}

export async function getPopular(limit?: number): Promise<PopularityResponse> {
  try {
    return await api.getPopular(limit)
  } catch {
    return []
  }
}

export async function getCircularRefs(): Promise<CircularRefResponse> {
  try {
    return await api.getCircularRefs()
  } catch {
    return []
  }
}

export async function getGraphHealth(): Promise<GraphHealthResponse | null> {
  try {
    return await api.getGraphHealth()
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn('[graphService] Invalid API response for health:', error.message)
      return null
    }
    throw error
  }
}

export async function getNodeHealth(id: string): Promise<NodeHealthResponse | null> {
  try {
    return await api.getNodeHealth(id)
  } catch {
    return null
  }
}

export async function getGraphHealthConfig(): Promise<GraphHealthConfigResponse | null> {
  try {
    return await api.getGraphHealthConfig()
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn('[graphService] Invalid API response for health config:', error.message)
      return null
    }
    throw error
  }
}

export async function setGraphHealthConfig(
  config: GraphHealthConfigResponse,
): Promise<GraphHealthConfigResponse | null> {
  try {
    return await api.setGraphHealthConfig(config)
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn('[graphService] Invalid API response for set health config:', error.message)
      return null
    }
    throw error
  }
}
