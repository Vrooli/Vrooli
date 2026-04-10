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
 * - PUT  /teams/{id}/org/edges/{reportId} - Set a single edge
 * - DELETE /teams/{id}/org/edges/{reportId} - Remove a single edge
 */

import { buildApiUrl } from '@vrooli/api-base'
import { API_BASE } from '@/lib/api'
import type { OrgEdge, OrgChartApiResponse, SetOrgChartRequest, UpdateEdgeRequest } from '@/types/orgChart'

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
  apiEdge: { managerAgentId: string; reportAgentId: string }
): OrgEdge {
  return {
    id: `${apiEdge.managerAgentId}-${apiEdge.reportAgentId}`,
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
    return response.edges.map((edge) => apiEdgeToOrgEdge(edge))
  } catch (error) {
    // If endpoint doesn't exist yet or team has no org chart, return empty array
    if (error instanceof Error && error.message.includes('404')) {
      return []
    }
    console.warn('[orgChartService] Failed to get edges:', error)
    throw error
  }
}

/**
 * Update an edge (change a member's manager) using the single-edge endpoint.
 */
export async function updateEdge(
  teamId: string,
  agentId: string,
  request: UpdateEdgeRequest
): Promise<OrgEdge | null> {
  if (!request.managerId) {
    await removeEdge(teamId, agentId)
    return null
  }

  const response = await apiRequest<{ managerAgentId: string; reportAgentId: string }>(
    `/teams/${encodeURIComponent(teamId)}/org/edges/${encodeURIComponent(agentId)}`,
    {
      method: 'PUT',
      body: JSON.stringify({ managerAgentId: request.managerId }),
    }
  )

  return apiEdgeToOrgEdge(response)
}

/**
 * Remove an edge (remove manager relationship).
 */
export async function removeEdge(teamId: string, agentId: string): Promise<void> {
  await apiRequest(
    `/teams/${encodeURIComponent(teamId)}/org/edges/${encodeURIComponent(agentId)}`,
    {
      method: 'DELETE',
    }
  )
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
      .map((e) => ({
        id: `${e.managerId}-${e.agentId}`,
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
