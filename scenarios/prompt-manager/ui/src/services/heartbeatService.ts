/**
 * Heartbeat Service - API wrapper for heartbeat operations.
 *
 * Provides:
 * - Heartbeat configuration CRUD
 * - Manual triggering
 * - Execution logs access
 * - Member document (RESPONSIBILITIES.md, HEARTBEAT.md) operations
 */

import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'

const API_BASE = resolveApiBase({ appendSuffix: true })

// ============================================================================
// Types
// ============================================================================

export interface HeartbeatConfig {
  teamId: string
  agentId: string
  enabled: boolean
  schedule: string
  profileKey?: string
  lastExecution?: HeartbeatExecResult
  nextExecution?: string
  createdAt: string
  updatedAt: string
}

export interface HeartbeatExecResult {
  startedAt: string
  endedAt?: string
  status: 'running' | 'completed' | 'failed'
  runId?: string
  logPath?: string
  error?: string
}

export interface CreateHeartbeatRequest {
  schedule: string
  profileKey?: string
  enabled?: boolean
}

export interface UpdateHeartbeatRequest {
  schedule?: string
  profileKey?: string
  enabled?: boolean
}

export interface TriggerResponse {
  teamId: string
  agentId: string
  runId: string
  status: string
  logPath?: string
}

export interface LogEntry {
  filename: string
  timestamp: string
  status?: string
}

export interface LogListResponse {
  teamId: string
  agentId: string
  logs: LogEntry[]
}

export interface LogContentResponse {
  teamId: string
  agentId: string
  filename: string
  content: string
}

export interface MemberDocResponse {
  teamId: string
  agentId: string
  content: string
}

export interface MemberDocRequest {
  content: string
}

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
  // Merge additional headers if provided
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
// Heartbeat Config Operations
// ============================================================================

/**
 * List all heartbeat configs for a team.
 */
export async function listHeartbeats(teamId: string): Promise<HeartbeatConfig[]> {
  return apiRequest<HeartbeatConfig[]>(`/teams/${encodeURIComponent(teamId)}/heartbeats`)
}

/**
 * Get a single heartbeat config.
 */
export async function getHeartbeat(teamId: string, agentId: string): Promise<HeartbeatConfig | null> {
  try {
    return await apiRequest<HeartbeatConfig>(
      `/teams/${encodeURIComponent(teamId)}/heartbeats/${encodeURIComponent(agentId)}`
    )
  } catch (error) {
    if (error instanceof Error && error.message.includes('404')) {
      return null
    }
    throw error
  }
}

/**
 * Create a heartbeat config.
 */
export async function createHeartbeat(
  teamId: string,
  agentId: string,
  request: CreateHeartbeatRequest
): Promise<HeartbeatConfig> {
  return apiRequest<HeartbeatConfig>(
    `/teams/${encodeURIComponent(teamId)}/heartbeats/${encodeURIComponent(agentId)}`,
    {
      method: 'POST',
      body: JSON.stringify(request),
    }
  )
}

/**
 * Update a heartbeat config.
 */
export async function updateHeartbeat(
  teamId: string,
  agentId: string,
  request: UpdateHeartbeatRequest
): Promise<HeartbeatConfig> {
  return apiRequest<HeartbeatConfig>(
    `/teams/${encodeURIComponent(teamId)}/heartbeats/${encodeURIComponent(agentId)}`,
    {
      method: 'PUT',
      body: JSON.stringify(request),
    }
  )
}

/**
 * Delete a heartbeat config.
 */
export async function deleteHeartbeat(teamId: string, agentId: string): Promise<void> {
  await apiRequest<Record<string, never>>(
    `/teams/${encodeURIComponent(teamId)}/heartbeats/${encodeURIComponent(agentId)}`,
    {
      method: 'DELETE',
    }
  )
}

/**
 * Manually trigger a heartbeat.
 */
export async function triggerHeartbeat(teamId: string, agentId: string): Promise<TriggerResponse> {
  return apiRequest<TriggerResponse>(
    `/teams/${encodeURIComponent(teamId)}/heartbeats/${encodeURIComponent(agentId)}/trigger`,
    {
      method: 'POST',
    }
  )
}

// ============================================================================
// Heartbeat Logs
// ============================================================================

/**
 * List execution logs for a member.
 */
export async function listLogs(teamId: string, agentId: string): Promise<LogEntry[]> {
  const response = await apiRequest<LogListResponse>(
    `/teams/${encodeURIComponent(teamId)}/heartbeats/${encodeURIComponent(agentId)}/logs`
  )
  return response.logs
}

/**
 * Get log content.
 */
export async function getLog(teamId: string, agentId: string, logId: string): Promise<string> {
  const response = await apiRequest<LogContentResponse>(
    `/teams/${encodeURIComponent(teamId)}/heartbeats/${encodeURIComponent(agentId)}/logs/${encodeURIComponent(logId)}`
  )
  return response.content
}

// ============================================================================
// Member Documents
// ============================================================================

/**
 * Get RESPONSIBILITIES.md content for a team member.
 */
export async function getResponsibilities(teamId: string, agentId: string): Promise<string> {
  const response = await apiRequest<MemberDocResponse>(
    `/teams/${encodeURIComponent(teamId)}/members/${encodeURIComponent(agentId)}/responsibilities`
  )
  return response.content
}

/**
 * Set RESPONSIBILITIES.md content for a team member.
 */
export async function setResponsibilities(teamId: string, agentId: string, content: string): Promise<void> {
  await apiRequest<MemberDocResponse>(
    `/teams/${encodeURIComponent(teamId)}/members/${encodeURIComponent(agentId)}/responsibilities`,
    {
      method: 'PUT',
      body: JSON.stringify({ content } as MemberDocRequest),
    }
  )
}

/**
 * Get HEARTBEAT.md content for a team member.
 */
export async function getHeartbeatInstructions(teamId: string, agentId: string): Promise<string> {
  const response = await apiRequest<MemberDocResponse>(
    `/teams/${encodeURIComponent(teamId)}/members/${encodeURIComponent(agentId)}/heartbeat-instructions`
  )
  return response.content
}

/**
 * Set HEARTBEAT.md content for a team member.
 */
export async function setHeartbeatInstructions(teamId: string, agentId: string, content: string): Promise<void> {
  await apiRequest<MemberDocResponse>(
    `/teams/${encodeURIComponent(teamId)}/members/${encodeURIComponent(agentId)}/heartbeat-instructions`,
    {
      method: 'PUT',
      body: JSON.stringify({ content } as MemberDocRequest),
    }
  )
}

// ============================================================================
// Convenience Functions
// ============================================================================

/**
 * Enable heartbeat with schedule.
 */
export async function enableHeartbeat(
  teamId: string,
  agentId: string,
  schedule: string,
  profileKey?: string
): Promise<HeartbeatConfig> {
  // Check if config exists
  const existing = await getHeartbeat(teamId, agentId)

  if (existing) {
    return updateHeartbeat(teamId, agentId, {
      schedule,
      profileKey,
      enabled: true,
    })
  } else {
    return createHeartbeat(teamId, agentId, {
      schedule,
      profileKey,
      enabled: true,
    })
  }
}

/**
 * Disable heartbeat.
 */
export async function disableHeartbeat(teamId: string, agentId: string): Promise<HeartbeatConfig | null> {
  const existing = await getHeartbeat(teamId, agentId)
  if (!existing) {
    return null
  }
  return updateHeartbeat(teamId, agentId, { enabled: false })
}

/**
 * Common cron schedule presets.
 */
export const SCHEDULE_PRESETS = [
  { label: 'Every hour', value: '0 * * * *' },
  { label: 'Every 6 hours', value: '0 */6 * * *' },
  { label: 'Every 12 hours', value: '0 */12 * * *' },
  { label: 'Daily at midnight', value: '0 0 * * *' },
  { label: 'Daily at 9am', value: '0 9 * * *' },
  { label: 'Weekly on Monday', value: '0 0 * * 1' },
]
