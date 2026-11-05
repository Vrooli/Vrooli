/**
 * Member-flow service - API wrapper for topics-graph + drain-status endpoints
 * and the inbox-flow surface (per-member topic declarations and the
 * inbox-router-drain mechanics).
 *
 * Backend endpoints:
 * - GET    /topics/graph[?team=<id>]
 * - GET    /topics/drain-status[?team=<id>]
 * - GET    /teams/{id}/members/{agentId}/topics
 * - GET    /teams/{id}/knowledge?topic_prefix=<prefix>[&last=N]   (inbox listing)
 * - PUT    /teams/{id}/knowledge/{knowledgeId}                    (promote: retag)
 * - DELETE /teams/{id}/knowledge/{knowledgeId}                    (drop)
 *
 * DOC: docs/agent-system/TOPICS_SCHEMA.md
 *      docs/agent-system/INTAKE_PIPELINE.md
 */

import { buildApiUrl } from '@vrooli/api-base'
import { API_BASE, connectSlice4Request } from '@/lib/api'
import type {
  TopicDeclaration,
  TopicsGraphResponse,
  TopicsDrainStatusResponse,
} from '@/types/topicsGraph'
import type { KnowledgeEntry, KnowledgeListResponse } from '@/services/heartbeatService'

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  parseJson?: boolean
}

async function apiRequest<T>(endpoint: string, opts: RequestOptions = {}): Promise<T> {
  const url = buildApiUrl(endpoint, { baseUrl: API_BASE })
  const init: RequestInit = {
    method: opts.method ?? 'GET',
    headers: { 'Content-Type': 'application/json' },
  }
  if (opts.body !== undefined) {
    init.body = JSON.stringify(opts.body)
  }
  const migrated = await connectSlice4Request(endpoint, init)
  if (migrated.handled) {
    return migrated.data as T
  }
  const response = await fetch(url, init)
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error')
    throw new Error(`API error: ${response.status} ${response.statusText} - ${errorText}`)
  }
  if (opts.parseJson === false) {
    return undefined as T
  }
  return response.json() as Promise<T>
}

/**
 * Fetch the topics graph. When `teamId` is provided, the response contains
 * member nodes for that team plus boundary nodes (cross-team neighbours,
 * external producers, Swarm Manager work feed, PoR sinks, capability work registry).
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

export interface MemberTopicsResponse {
  team: string
  member: string
  exists: boolean
  topics: TopicDeclaration
}

/**
 * Fetch a single member's topics.json. Used by MemberDetailPanel to decide
 * whether to surface the Inbox tab (visible only when intake[] is non-empty).
 */
export async function getMemberTopics(
  teamId: string,
  agentId: string,
): Promise<MemberTopicsResponse> {
  return apiRequest<MemberTopicsResponse>(
    `/teams/${encodeURIComponent(teamId)}/members/${encodeURIComponent(agentId)}/topics`,
  )
}

/**
 * List unrouted inbox entries under a topic prefix. Wraps the team
 * knowledge-list endpoint with a prefix filter — by the inbox-router-drain
 * invariant, every entry under an `<inbox-name>/*` prefix is unrouted.
 */
export async function listInboxEntries(
  teamId: string,
  prefix: string,
  opts: { limit?: number } = {},
): Promise<KnowledgeEntry[]> {
  const params = new URLSearchParams()
  params.set('topic_prefix', prefix)
  if (opts.limit !== undefined) params.set('last', String(opts.limit))
  const response = await apiRequest<KnowledgeListResponse>(
    `/teams/${encodeURIComponent(teamId)}/knowledge?${params.toString()}`,
  )
  return response.entries
}

/**
 * Promote an inbox entry by retagging it to a destination topic. Translates
 * to `prompt-manager team knowledge-update <team> <id> --topic=<destination>`.
 * After this call, the entry no longer appears under the inbox prefix — it
 * leaves the unrouted set, which is the inbox-router-drain invariant.
 */
export async function promoteInboxEntry(
  teamId: string,
  knowledgeId: string,
  destinationTopic: string,
): Promise<KnowledgeEntry> {
  return apiRequest<KnowledgeEntry>(
    `/teams/${encodeURIComponent(teamId)}/knowledge/${encodeURIComponent(knowledgeId)}`,
    { method: 'PUT', body: { topic: destinationTopic } },
  )
}

/**
 * Drop an inbox entry as weak / duplicate / out-of-scope. Translates to
 * `prompt-manager team knowledge-delete <team> <id>`.
 */
export async function dropInboxEntry(teamId: string, knowledgeId: string): Promise<void> {
  await apiRequest<unknown>(
    `/teams/${encodeURIComponent(teamId)}/knowledge/${encodeURIComponent(knowledgeId)}`,
    { method: 'DELETE', parseJson: false },
  )
}
