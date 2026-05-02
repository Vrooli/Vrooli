/**
 * Member-flow service - API wrapper for topics-graph + drain-status endpoints.
 *
 * Backend endpoints:
 * - GET /topics/graph[?team=<id>]
 * - GET /topics/drain-status[?team=<id>]
 *
 * DOC: docs/agent-system/drafts/topics-schema.md
 */

import { buildApiUrl } from '@vrooli/api-base'
import { API_BASE } from '@/lib/api'
import type {
  TopicsGraphResponse,
  TopicsDrainStatusResponse,
} from '@/types/topicsGraph'

async function apiRequest<T>(endpoint: string): Promise<T> {
  const url = buildApiUrl(endpoint, { baseUrl: API_BASE })
  const response = await fetch(url, {
    headers: { 'Content-Type': 'application/json' },
  })
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error')
    throw new Error(`API error: ${response.status} ${response.statusText} - ${errorText}`)
  }
  return response.json() as Promise<T>
}

/**
 * Fetch the topics graph. When `teamId` is provided, the response contains
 * member nodes for that team plus boundary nodes (cross-team neighbours,
 * external producers, decision queues, PoR sinks, capability-gap registry).
 */
export async function getTopicsGraph(teamId?: string): Promise<TopicsGraphResponse> {
  const qs = teamId ? `?team=${encodeURIComponent(teamId)}` : ''
  return apiRequest<TopicsGraphResponse>(`/topics/graph${qs}`)
}

/**
 * Fetch drain-status (per-prefix queue depth + age + recent throughput).
 * Phase 5 will return real data; today this is a stub note.
 */
export async function getDrainStatus(teamId?: string): Promise<TopicsDrainStatusResponse> {
  const qs = teamId ? `?team=${encodeURIComponent(teamId)}` : ''
  return apiRequest<TopicsDrainStatusResponse>(`/topics/drain-status${qs}`)
}
