/**
 * Action Service - API wrapper with caching layer.
 *
 * Keeps UI integration thin: the API remains the source of truth for Action
 * schema validation, command ownership validation, and mutation behavior.
 */

import { api } from '@/lib/api'
import { createCacheManager } from '@/lib/cache'
import { ValidationError } from '@/lib/schemas'
import type {
  Action,
  ActionFilters,
  CreateActionRequest,
  UpdateActionRequest,
  ActionMutationResponse,
  ActionValidationResponse,
  ActionRunRequest,
  ActionRunResponse,
} from '@/types'

const actionsCache = createCacheManager<Action[]>()

export function invalidateCache(): void {
  actionsCache.invalidate()
}

export async function getActions(
  filters?: ActionFilters,
  forceRefresh = false
): Promise<Action[]> {
  const cacheable = !filters || Object.keys(filters).length === 0
  if (cacheable) {
    const cached = actionsCache.getIfValid(forceRefresh)
    if (cached) return cached
  }

  try {
    const data = await api.getActions(filters)
    if (cacheable) actionsCache.set(data)
    return data
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn('[actionService] Invalid API response for getActions:', error.message)
      return []
    }
    throw error
  }
}

export async function getAction(id: string): Promise<Action | undefined> {
  try {
    return await api.getAction(id)
  } catch (error) {
    if (error instanceof ValidationError) {
      console.warn(`[actionService] Invalid API response for action ${id}:`, error.message)
      return undefined
    }
    console.error(`[actionService] Failed to get action ${id}:`, error)
    return undefined
  }
}

export async function createAction(
  request: CreateActionRequest
): Promise<ActionMutationResponse> {
  const response = await api.createAction(request)
  invalidateCache()
  return response
}

export async function updateAction(
  id: string,
  updates: UpdateActionRequest
): Promise<ActionMutationResponse> {
  const response = await api.updateAction(id, updates)
  invalidateCache()
  return response
}

export async function deleteAction(id: string, hard = false): Promise<void> {
  await api.deleteAction(id, hard)
  invalidateCache()
}

export async function validateAction(id: string): Promise<ActionValidationResponse> {
  return api.validateAction(id)
}

export async function runAction(
  id: string,
  request: ActionRunRequest
): Promise<ActionRunResponse> {
  return api.runAction(id, request)
}

export async function searchActions(query: string): Promise<Action[]> {
  const cached = actionsCache.getIfValid()
  if (cached) {
    const lowerQuery = query.toLowerCase()
    return cached.filter((action) => {
      const argv = action.command.argv.join(' ')
      const owner = `${action.owner.type}:${action.owner.id}`
      return (
        action.id.toLowerCase().includes(lowerQuery) ||
        action.name.toLowerCase().includes(lowerQuery) ||
        action.description.toLowerCase().includes(lowerQuery) ||
        action.status.toLowerCase().includes(lowerQuery) ||
        owner.toLowerCase().includes(lowerQuery) ||
        argv.toLowerCase().includes(lowerQuery) ||
        action.tags.some((tag) => tag.toLowerCase().includes(lowerQuery))
      )
    })
  }

  const actions = await getActions()
  const lowerQuery = query.toLowerCase()
  return actions.filter((action) =>
    action.id.toLowerCase().includes(lowerQuery) ||
    action.name.toLowerCase().includes(lowerQuery) ||
    action.description.toLowerCase().includes(lowerQuery) ||
    action.tags.some((tag) => tag.toLowerCase().includes(lowerQuery))
  )
}
