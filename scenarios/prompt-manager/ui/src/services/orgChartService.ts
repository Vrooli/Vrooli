/**
 * Org Chart Service - API wrapper for org chart operations.
 *
 * Provides:
 * - Org chart edge (reporting relationships) CRUD
 * - Conversion between frontend OrgEdge format and backend API format
 *
 * Backend endpoints:
 * - GET  /teams/{id}/org - Get all edges
 * - PUT  /teams/{id}/org - Replace all edges
 */

import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'
import type { OrgEdge, OrgChartApiResponse, SetOrgChartRequest, UpdateEdgeRequest } from '@/types/orgChart'

const API_BASE = resolveApiBase({ appendSuffix: true })

// ============================================================================
// API Client
// ============================================================================

async function apiRequest<T>(
  endpoint: string,
  options?: RequestInit
): Promise<T> {
  const url = buildApiUrl(endpoint, { baseUrl: API_BASE })

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (options?.headers) {
    const extraHeaders = options.headers as Record<string, string>
    Object.assign(headers, extraHeaders)
  }

  const response = await fetch(url, {
    ...options,
    headers,
  })

  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error')
    throw new Error(`API error: ${response.status} ${response.statusText} - ${errorText}`)
  }

  if (response.status === 204) {
    return {} as T
  }

  return response.json() as Promise<T>
}

// ============================================================================
// Internal Helpers
// ============================================================================

/**
 * Convert backend API edge format to frontend OrgEdge format.
 */
function apiEdgeToOrgEdge(
  apiEdge: { managerAgentId: string; reportAgentId: string },
  index: number
): OrgEdge {
  return {
    id: `edge-${index}`,
    managerId: apiEdge.managerAgentId,
    reportId: apiEdge.reportAgentId,
  }
}

/**
 * Convert frontend OrgEdge format to backend API edge format.
 */
function orgEdgeToApiEdge(edge: OrgEdge): { managerAgentId: string; reportAgentId: string } {
  return {
    managerAgentId: edge.managerId,
    reportAgentId: edge.reportId,
  }
}

/**
 * Save all edges to the backend (replaces existing edges).
 */
async function setAllEdges(teamId: string, edges: OrgEdge[]): Promise<void> {
  const request: SetOrgChartRequest = {
    edges: edges.map(orgEdgeToApiEdge),
  }

  await apiRequest<OrgChartApiResponse>(
    `/teams/${encodeURIComponent(teamId)}/org`,
    {
      method: 'PUT',
      body: JSON.stringify(request),
    }
  )
}

// ============================================================================
// Org Chart Edge Operations
// ============================================================================

/**
 * Get all edges (reporting relationships) for a team.
 */
export async function getEdges(teamId: string): Promise<OrgEdge[]> {
  try {
    const response = await apiRequest<OrgChartApiResponse>(
      `/teams/${encodeURIComponent(teamId)}/org`
    )
    return response.edges.map(apiEdgeToOrgEdge)
  } catch (error) {
    // If endpoint doesn't exist yet or team has no org chart, return empty array
    if (error instanceof Error && (error.message.includes('404') || error.message.includes('500'))) {
      return []
    }
    console.warn('[orgChartService] Failed to get edges:', error)
    return []
  }
}

/**
 * Update an edge (change a member's manager).
 * Since the backend only supports replacing all edges, we:
 * 1. Get current edges
 * 2. Remove existing edge for this agent
 * 3. Add new edge if managerId provided
 * 4. Save all edges
 */
export async function updateEdge(
  teamId: string,
  agentId: string,
  request: UpdateEdgeRequest
): Promise<OrgEdge | null> {
  try {
    // 1. Get current edges
    const currentEdges = await getEdges(teamId)

    // 2. Remove any existing edge for this agent (as a report)
    const filteredEdges = currentEdges.filter((e) => e.reportId !== agentId)

    // 3. Add new edge if managerId provided
    if (request.managerId) {
      filteredEdges.push({
        id: `edge-${Date.now()}`,
        managerId: request.managerId,
        reportId: agentId,
      })
    }

    // 4. Save all edges
    await setAllEdges(teamId, filteredEdges)

    // 5. Return the new edge (or null if removed)
    if (request.managerId) {
      return filteredEdges.find((e) => e.reportId === agentId) ?? null
    }
    return null
  } catch (error) {
    console.error('[orgChartService] Failed to update edge:', error)
    return null
  }
}

/**
 * Remove an edge (remove manager relationship).
 */
export async function removeEdge(teamId: string, agentId: string): Promise<void> {
  try {
    const currentEdges = await getEdges(teamId)
    const filteredEdges = currentEdges.filter((e) => e.reportId !== agentId)
    await setAllEdges(teamId, filteredEdges)
  } catch (error) {
    console.error('[orgChartService] Failed to remove edge:', error)
  }
}

/**
 * Batch update edges (for drag-drop reordering).
 * Replaces all edges with the provided set.
 */
export async function batchUpdateEdges(
  teamId: string,
  edges: Array<{ agentId: string; managerId: string | null }>
): Promise<OrgEdge[]> {
  try {
    // Filter out null managers and convert to OrgEdge format
    const newEdges: OrgEdge[] = edges
      .filter((e): e is { agentId: string; managerId: string } => e.managerId !== null)
      .map((e, idx) => ({
        id: `edge-${idx}`,
        managerId: e.managerId,
        reportId: e.agentId,
      }))

    await setAllEdges(teamId, newEdges)
    return newEdges
  } catch (error) {
    console.error('[orgChartService] Failed to batch update edges:', error)
    return []
  }
}
