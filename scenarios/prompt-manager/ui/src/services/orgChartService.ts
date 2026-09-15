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

import { createClient } from '@connectrpc/connect'
import { createScenarioConnectTransport, resolveApiBase } from '@vrooli/api-base'
import { TeamsService } from '@vrooli/proto-types/prompt-manager/v1/teams/teams_pb'
import type { ManagedTeamEdge, OrgEdge, OrgChartApiResponse, UpdateEdgeRequest } from '@/types/orgChart'

// ============================================================================
// API Client
// ============================================================================

const teamsClient = createClient(TeamsService, createScenarioConnectTransport({
  baseUrl: resolveApiBase({ appendSuffix: false }),
}))

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
/**
 * Save all edges to the backend (replaces existing edges).
 */
async function setAllEdges(teamId: string, edges: OrgEdge[]): Promise<void> {
  const existing = await teamsClient.getOrgChart({ teamId })
  await teamsClient.setOrgChart({
    teamId,
    edges: edges.map((edge) => ({ managerAgentId: edge.managerId, reportAgentId: edge.reportId })),
    managedTeamEdges: existing.managedTeamEdges,
  })
}

// ============================================================================
// Org Chart Edge Operations
// ============================================================================

/**
 * Get all edges (reporting relationships) for a team.
 */
export async function getEdges(teamId: string): Promise<OrgEdge[]> {
  try {
    const response = await teamsClient.getOrgChart({ teamId })
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

/** Read the configured team-to-team supervision edges owned by a team. */
export async function getManagedTeamEdges(teamId: string): Promise<ManagedTeamEdge[]> {
  const response = await teamsClient.getOrgChart({ teamId })
  return response.managedTeamEdges.map((edge) => ({
    managerTeamId: edge.managerTeamId,
    managedTeamId: edge.managedTeamId,
    relationship: edge.relationship,
    authorityRef: edge.authorityRef,
    status: edge.status,
  }))
}

export async function getOrgChart(teamId: string): Promise<OrgChartApiResponse> {
  const response = await teamsClient.getOrgChart({ teamId })
  return {
    teamId: response.teamId,
    edges: response.edges.map((edge) => ({ managerAgentId: edge.managerAgentId, reportAgentId: edge.reportAgentId })),
    managedTeamEdges: response.managedTeamEdges.map((edge) => ({
      managerTeamId: edge.managerTeamId,
      managedTeamId: edge.managedTeamId,
      relationship: edge.relationship,
      authorityRef: edge.authorityRef,
      status: edge.status,
    })),
  }
}

/** Replace only this team's outgoing managed-team relationships. */
export async function setManagedTeamEdges(teamId: string, managedTeamEdges: ManagedTeamEdge[]): Promise<void> {
  const response = await teamsClient.getOrgChart({ teamId })
  await teamsClient.setOrgChart({
    teamId,
    edges: response.edges,
    managedTeamEdges,
  })
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

  const response = await teamsClient.updateOrgChartEdge({ teamId, reportAgentId: agentId, managerAgentId: request.managerId })
  const edge = response.edges.find((candidate) => candidate.reportAgentId === agentId)
  return edge ? apiEdgeToOrgEdge(edge) : null
}

/**
 * Remove an edge (remove manager relationship).
 */
export async function removeEdge(teamId: string, agentId: string): Promise<void> {
  await teamsClient.deleteOrgChartEdge({ teamId, reportAgentId: agentId })
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
