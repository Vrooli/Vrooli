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
  nextExecutions?: string[]
  createdAt: string
  updatedAt: string
}

export interface HeartbeatExecResult {
  startedAt: string
  endedAt?: string
  status: 'running' | 'completed' | 'failed' | 'cancelled'
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

// ============================================================================
// Running Agents
// ============================================================================

export interface RunningAgentEntry {
  teamId: string
  agentId: string
  agentName: string
  teamName: string
  runId: string
  startedAt: string
  duration: string
}

export interface RunningAgentsResponse {
  count: number
  agents: RunningAgentEntry[]
}

export interface StopAgentResponse {
  teamId: string
  agentId: string
  runId: string
  status: string
}

/**
 * List all currently running heartbeat agents.
 */
export async function listRunningAgents(): Promise<RunningAgentsResponse> {
  return apiRequest<RunningAgentsResponse>('/heartbeats/running')
}

/**
 * Stop a running heartbeat agent.
 */
export async function stopRunningAgent(teamId: string, agentId: string): Promise<StopAgentResponse> {
  return apiRequest<StopAgentResponse>(
    `/heartbeats/running/${encodeURIComponent(teamId)}/${encodeURIComponent(agentId)}/stop`,
    { method: 'POST' }
  )
}

// ============================================================================
// Run Details
// ============================================================================

export interface RunActions {
  canInvestigate: boolean
  canApplyInvestigation: boolean
  canDelete: boolean
  canStop: boolean
  canRetry: boolean
  canContinue: boolean
}

export interface RunDetails {
  id: string
  taskId: string
  profileId?: string
  status: string
  startedAt?: string
  endedAt?: string
  error?: string
  tag?: string
  sessionId?: string
  actions?: RunActions
}

export interface ListRunsResponse {
  runs: RunDetails[]
  total: number
  hasMore: boolean
}

/**
 * Fetch details for a single run by ID.
 */
export async function getRunDetails(runId: string): Promise<RunDetails> {
  const raw = await apiRequest<{ run: { id: string; task_id: string; agent_profile_id?: string; status: string; started_at?: string; ended_at?: string; error_msg?: string; tag?: string; session_id?: string; actions?: { can_investigate?: boolean; can_apply_investigation?: boolean; can_delete?: boolean; can_stop?: boolean; can_retry?: boolean; can_continue?: boolean } } }>(
    `/runs/${encodeURIComponent(runId)}`
  )
  const r = raw.run
  return {
    id: r.id,
    taskId: r.task_id,
    profileId: r.agent_profile_id,
    status: r.status,
    startedAt: r.started_at,
    endedAt: r.ended_at,
    error: r.error_msg,
    tag: r.tag,
    sessionId: r.session_id,
    actions: r.actions ? {
      canInvestigate: r.actions.can_investigate ?? false,
      canApplyInvestigation: r.actions.can_apply_investigation ?? false,
      canDelete: r.actions.can_delete ?? false,
      canStop: r.actions.can_stop ?? false,
      canRetry: r.actions.can_retry ?? false,
      canContinue: r.actions.can_continue ?? false,
    } : undefined,
  }
}

/**
 * List runs with optional filtering.
 */
export async function listRuns(opts?: {
  status?: string
  tagPrefix?: string
  limit?: number
  offset?: number
}): Promise<ListRunsResponse> {
  const params = new URLSearchParams()
  if (opts?.status) params.set('status', opts.status)
  if (opts?.tagPrefix) params.set('tag_prefix', opts.tagPrefix)
  if (opts?.limit !== undefined) params.set('limit', String(opts.limit))
  if (opts?.offset !== undefined) params.set('offset', String(opts.offset))
  const qs = params.toString()
  const endpoint = `/runs${qs ? `?${qs}` : ''}`

  const raw = await apiRequest<{ runs?: Array<{ id: string; task_id: string; agent_profile_id?: string; status: string; started_at?: string; ended_at?: string; error_msg?: string; tag?: string; session_id?: string }>; total?: number; has_more?: boolean }>(endpoint)
  const runs = (raw.runs ?? []).map((r) => ({
    id: r.id,
    taskId: r.task_id,
    profileId: r.agent_profile_id,
    status: r.status,
    startedAt: r.started_at,
    endedAt: r.ended_at,
    error: r.error_msg,
    tag: r.tag,
    sessionId: r.session_id,
  }))
  return {
    runs,
    total: raw.total ?? runs.length,
    hasMore: raw.has_more ?? false,
  }
}

/**
 * Continue a run with an additional message.
 */
export async function continueRun(runId: string, message: string): Promise<void> {
  await apiRequest<Record<string, never>>(
    `/runs/${encodeURIComponent(runId)}/continue`,
    {
      method: 'POST',
      body: JSON.stringify({ message }),
    }
  )
}

/**
 * Create an investigation run for one or more failed runs.
 */
export async function createInvestigationRun(runIds: string[], opts?: {
  depth?: string
  customContext?: string
}): Promise<RunDetails> {
  const raw = await apiRequest<{ run: { id: string; task_id: string; agent_profile_id?: string; status: string; started_at?: string; ended_at?: string; error_msg?: string; tag?: string; session_id?: string } }>(
    '/runs/investigate',
    {
      method: 'POST',
      body: JSON.stringify({
        run_ids: runIds,
        depth: opts?.depth,
        custom_context: opts?.customContext,
      }),
    }
  )
  const r = raw.run
  return {
    id: r.id,
    taskId: r.task_id,
    profileId: r.agent_profile_id,
    status: r.status,
    startedAt: r.started_at,
    endedAt: r.ended_at,
    error: r.error_msg,
    tag: r.tag,
    sessionId: r.session_id,
  }
}

/**
 * Create an investigation-apply run from an investigation run.
 */
export async function createInvestigationApplyRun(
  investigationRunId: string,
  customContext?: string
): Promise<RunDetails> {
  const raw = await apiRequest<{ run: { id: string; task_id: string; agent_profile_id?: string; status: string; started_at?: string; ended_at?: string; error_msg?: string; tag?: string; session_id?: string } }>(
    '/runs/investigation-apply',
    {
      method: 'POST',
      body: JSON.stringify({
        investigation_run_id: investigationRunId,
        custom_context: customContext,
      }),
    }
  )
  const r = raw.run
  return {
    id: r.id,
    taskId: r.task_id,
    profileId: r.agent_profile_id,
    status: r.status,
    startedAt: r.started_at,
    endedAt: r.ended_at,
    error: r.error_msg,
    tag: r.tag,
    sessionId: r.session_id,
  }
}

// ============================================================================
// Run Events
// ============================================================================

export interface RunEvent {
  id: string
  runId: string
  sequence: number
  eventType: 'log' | 'message' | 'tool_call' | 'tool_result' | 'status' | 'metric' | 'error'
  timestamp: string
  data: Record<string, unknown>
}

/**
 * Map proto event type enums to short names used by the UI.
 * Agent-manager uses protojson names like "RUN_EVENT_TYPE_MESSAGE".
 */
const EVENT_TYPE_MAP: Record<string, RunEvent['eventType']> = {
  RUN_EVENT_TYPE_MESSAGE: 'message',
  RUN_EVENT_TYPE_TOOL_CALL: 'tool_call',
  RUN_EVENT_TYPE_TOOL_RESULT: 'tool_result',
  RUN_EVENT_TYPE_STATUS: 'status',
  RUN_EVENT_TYPE_METRIC: 'metric',
  RUN_EVENT_TYPE_LOG: 'log',
  RUN_EVENT_TYPE_ERROR: 'error',
}

/** Agent-manager event shape (snake_case protojson with typed payload fields). */
interface RawRunEvent {
  id: string
  run_id: string
  sequence?: string | number
  event_type: string
  timestamp: string
  // Payload is one of these, keyed by short type name:
  message?: Record<string, unknown>
  tool_call?: Record<string, unknown>
  tool_result?: Record<string, unknown>
  status?: Record<string, unknown>
  metric?: Record<string, unknown>
  log?: Record<string, unknown>
  error?: Record<string, unknown>
  // Fallback for unknown types
  data?: Record<string, unknown>
}

/** Normalize a single raw event into the UI-friendly RunEvent shape. */
function normalizeEvent(raw: RawRunEvent): RunEvent {
  const shortType = EVENT_TYPE_MAP[raw.event_type] ?? (raw.event_type?.toLowerCase().replace('run_event_type_', '') as RunEvent['eventType'])

  // Extract the typed payload — agent-manager nests it under the short type key
  const payload: Record<string, unknown> =
    raw.message ?? raw.tool_call ?? raw.tool_result ??
    raw.status ?? raw.metric ?? raw.log ?? raw.error ??
    raw.data ?? {}

  return {
    id: raw.id,
    runId: raw.run_id,
    sequence: typeof raw.sequence === 'string' ? parseInt(raw.sequence, 10) || 0 : (raw.sequence ?? 0),
    eventType: shortType,
    timestamp: raw.timestamp,
    data: payload,
  }
}

/**
 * Fetch events for a run, optionally starting after a given sequence number.
 */
export async function getRunEvents(runId: string, opts?: {
  afterSequence?: number
  limit?: number
}): Promise<RunEvent[]> {
  const params = new URLSearchParams()
  if (opts?.afterSequence !== undefined) {
    params.set('after_sequence', String(opts.afterSequence))
  }
  if (opts?.limit !== undefined) {
    params.set('limit', String(opts.limit))
  }
  const qs = params.toString()
  const endpoint = `/runs/${encodeURIComponent(runId)}/events${qs ? `?${qs}` : ''}`

  // Agent-manager wraps events in {"events": [...]}
  const raw = await apiRequest<{ events?: RawRunEvent[] } | RawRunEvent[]>(endpoint)
  const rawEvents = Array.isArray(raw) ? raw : (raw.events ?? [])
  return rawEvents.map(normalizeEvent)
}

// ============================================================================
// Schedule Presets
// ============================================================================

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
