/**
 * Heartbeat Service - API wrapper for heartbeat operations.
 *
 * Provides:
 * - Heartbeat configuration CRUD
 * - Manual triggering
 * - Execution logs access
 * - Member document (RESPONSIBILITIES.md, HEARTBEAT.md) operations
 */

import { buildApiUrl } from '@vrooli/api-base'
import { API_BASE } from '@/lib/api'

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

export interface TeamLogEntry {
  agentId: string
  agentDisplayName: string
  filename: string
  timestamp: string
  status?: string
}

export interface TeamLogListResponse {
  teamId: string
  logs: TeamLogEntry[]
  total: number
  hasMore: boolean
}

export interface MemberDocResponse {
  teamId: string
  agentId: string
  content: string
}

export interface MemberDocRequest {
  content: string
}

// --- Team State Types ---

export interface HandoffResponse {
  teamId: string
  agentId: string
  content: string
}

export interface HandoffHistoryEntry {
  agentId: string
  runId: string
  timestamp: string
  content: string
}

export interface HandoffHistoryResponse {
  teamId: string
  entries: HandoffHistoryEntry[]
}

export interface TaskNote {
  at: string
  by: string
  text: string
}

export interface TeamTask {
  id: string
  title: string
  status: 'todo' | 'in-progress' | 'blocked' | 'done'
  assignee: string
  priority: string
  createdBy: string
  createdAt: string
  updatedAt: string
  notes?: TaskNote[]
}

export interface TaskBoardResponse {
  teamId: string
  tasks: TeamTask[]
}

export interface AddTaskRequest {
  title: string
  assignee?: string
  priority?: string
  from: string
}

export interface UpdateTaskRequest {
  title?: string
  status?: string
  assignee?: string
  priority?: string
  note?: string
}

export interface DecisionOption {
  key: string
  label: string
  rationale: string
  recommended?: boolean
}

export interface DecisionEntry {
  id: string
  at: string
  by: string
  decision: string
  rationale: string
  context?: string
  supersedes?: string
  status?: 'pending' | 'accepted' | 'rejected' | 'running' | 'completed'
  topic?: string
  description?: string
  options?: DecisionOption[]
  selected?: string | null
  freeform?: string | null
  notes?: string | null
}

export interface UpdateDecisionRequest {
  decision?: string
  rationale?: string
  context?: string
  status?: string
  supersedes?: string
  topic?: string
  description?: string
  options?: DecisionOption[]
  selected?: string | null
  freeform?: string | null
  notes?: string | null
}

export interface DecisionListResponse {
  teamId: string
  entries: DecisionEntry[]
}

export interface PendingDecisionTeamGroup {
  teamId: string
  teamName: string
  entries: DecisionEntry[]
}

export interface AllPendingDecisionsResponse {
  teams: PendingDecisionTeamGroup[]
  totalCount: number
}

export interface AddDecisionRequest {
  by: string
  decision?: string
  rationale?: string
  context?: string
  supersedes?: string
  topic?: string
  options?: DecisionOption[]
}

// --- Knowledge types ---

export interface KnowledgeEntry {
  id: string
  at: string
  by: string
  topic: string
  content: string
  source?: string
  supersedes?: string
}

export interface KnowledgeListResponse {
  teamId: string
  entries: KnowledgeEntry[]
}

export interface AddKnowledgeRequest {
  by: string
  topic: string
  content: string
  source?: string
  supersedes?: string
}

export interface UpdateKnowledgeRequest {
  topic?: string
  content?: string
  source?: string
  supersedes?: string
}

const HEARTBEAT_LIST_CACHE_TTL_MS = 1200
const heartbeatListInFlight = new Map<string, Promise<HeartbeatConfig[]>>()
const heartbeatListCache = new Map<string, { data: HeartbeatConfig[]; fetchedAt: number }>()

function invalidateHeartbeatListCache(teamId?: string) {
  if (teamId === undefined) {
    heartbeatListInFlight.clear()
    heartbeatListCache.clear()
    return
  }
  heartbeatListInFlight.delete(teamId)
  heartbeatListCache.delete(teamId)
}

export function resetHeartbeatServiceCachesForTests() {
  invalidateHeartbeatListCache()
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
    throw new Error(formatApiError(response, errorText))
  }

  if (response.status === 204) {
    return {} as T
  }

  return response.json() as Promise<T>
}

function formatApiError(response: Response, rawBody: string): string {
  const status = response.status
  const statusText = response.statusText || 'Error'
  const body = (rawBody || '').trim()
  const contentType = response.headers.get('content-type') || ''
  const hop = response.headers.get('x-vrooli-error-hop') || response.headers.get('x-vrooli-proxy-hop')
  const hopSuffix = hop ? ` (hop: ${hop})` : ''

  // Handle gateway/proxy HTML pages (e.g., Cloudflare/nginx) with a concise message.
  if (status >= 500 && looksLikeHtml(body, contentType)) {
    if (hop) {
      return `API error: ${status} ${statusText} - Upstream gateway error at ${hop}.`
    }
    return `API error: ${status} ${statusText} - Upstream gateway error before prompt-manager API (edge/tunnel or host).`
  }

  // Prefer structured JSON errors when available.
  const jsonMessage = extractJsonErrorMessage(body, contentType)
  if (jsonMessage) {
    return `API error: ${status} ${statusText}${hopSuffix} - ${jsonMessage}`
  }

  const sanitized = sanitizeBody(body)
  return `API error: ${status} ${statusText}${hopSuffix} - ${sanitized || 'Unknown error'}`
}

function looksLikeHtml(body: string, contentType: string): boolean {
  if (contentType.toLowerCase().includes('text/html')) return true
  const lower = body.toLowerCase()
  return lower.includes('<!doctype html') || lower.includes('<html')
}

function extractJsonErrorMessage(body: string, contentType: string): string | null {
  if (!body) return null
  const mayBeJson = contentType.toLowerCase().includes('application/json') || body.startsWith('{')
  if (!mayBeJson) return null

  try {
    const parsed = JSON.parse(body) as Record<string, unknown>
    const message = parsed.message
    if (typeof message === 'string' && message.trim()) return message.trim()
    const error = parsed.error
    if (typeof error === 'string' && error.trim()) return error.trim()
  } catch {
    return null
  }
  return null
}

function sanitizeBody(body: string): string {
  if (!body) return ''
  const normalized = body.replace(/\s+/g, ' ').trim()
  const limit = 220
  if (normalized.length <= limit) return normalized
  return `${normalized.slice(0, limit)}...`
}

// ============================================================================
// Heartbeat Config Operations
// ============================================================================

/**
 * List all heartbeat configs for a team.
 */
export async function listHeartbeats(teamId: string): Promise<HeartbeatConfig[]> {
  const now = Date.now()
  const cached = heartbeatListCache.get(teamId)
  if (cached && now - cached.fetchedAt < HEARTBEAT_LIST_CACHE_TTL_MS) {
    return cached.data
  }

  const inFlight = heartbeatListInFlight.get(teamId)
  if (inFlight) {
    return inFlight
  }

  const request = apiRequest<HeartbeatConfig[]>(`/teams/${encodeURIComponent(teamId)}/heartbeats`)
    .then((data) => {
      heartbeatListCache.set(teamId, { data, fetchedAt: Date.now() })
      return data
    })
    .finally(() => {
      heartbeatListInFlight.delete(teamId)
    })

  heartbeatListInFlight.set(teamId, request)
  return request
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
  const response = await apiRequest<HeartbeatConfig>(
    `/teams/${encodeURIComponent(teamId)}/heartbeats/${encodeURIComponent(agentId)}`,
    {
      method: 'POST',
      body: JSON.stringify(request),
    }
  )
  invalidateHeartbeatListCache(teamId)
  return response
}

/**
 * Update a heartbeat config.
 */
export async function updateHeartbeat(
  teamId: string,
  agentId: string,
  request: UpdateHeartbeatRequest
): Promise<HeartbeatConfig> {
  const response = await apiRequest<HeartbeatConfig>(
    `/teams/${encodeURIComponent(teamId)}/heartbeats/${encodeURIComponent(agentId)}`,
    {
      method: 'PUT',
      body: JSON.stringify(request),
    }
  )
  invalidateHeartbeatListCache(teamId)
  return response
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
  invalidateHeartbeatListCache(teamId)
}

/**
 * Manually trigger a heartbeat.
 */
export async function triggerHeartbeat(teamId: string, agentId: string): Promise<TriggerResponse> {
  const response = await apiRequest<TriggerResponse>(
    `/teams/${encodeURIComponent(teamId)}/heartbeats/${encodeURIComponent(agentId)}/trigger`,
    {
      method: 'POST',
    }
  )
  invalidateHeartbeatListCache(teamId)
  return response
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

/**
 * List execution logs across all team members.
 */
export async function listTeamLogs(teamId: string, opts?: {
  limit?: number
  offset?: number
  agentId?: string
}): Promise<TeamLogListResponse> {
  const params = new URLSearchParams()
  if (opts?.limit !== undefined) params.set('limit', String(opts.limit))
  if (opts?.offset !== undefined) params.set('offset', String(opts.offset))
  if (opts?.agentId) params.set('agentId', opts.agentId)
  const qs = params.toString()
  return apiRequest<TeamLogListResponse>(
    `/teams/${encodeURIComponent(teamId)}/heartbeats/logs${qs ? `?${qs}` : ''}`
  )
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
  teamId?: string
  agentId?: string
  actions?: RunActions
}

export interface ListRunsResponse {
  runs: RunDetails[]
  total: number
  hasMore: boolean
}

function normalizeRunStatus(status: string): string {
  const normalized = status.trim().toLowerCase()
  if (!normalized) return status
  if (
    normalized.includes('completed') ||
    normalized.includes('complete') ||
    normalized.includes('succeeded') ||
    normalized.includes('success')
  ) {
    return 'completed'
  }
  if (
    normalized.includes('running') ||
    normalized.includes('in_progress') ||
    normalized.includes('in progress')
  ) {
    return 'running'
  }
  if (
    normalized.includes('failed') ||
    normalized.includes('failure') ||
    normalized.includes('error')
  ) {
    return 'failed'
  }
  if (
    normalized.includes('cancelled') ||
    normalized.includes('canceled')
  ) {
    return 'cancelled'
  }
  if (
    normalized.includes('pending') ||
    normalized.includes('queued') ||
    normalized.includes('waiting')
  ) {
    return 'pending'
  }
  return normalized
}

/**
 * Fetch details for a single run by ID.
 */
export async function getRunDetails(runId: string): Promise<RunDetails> {
  const raw = await apiRequest<{ run: { id: string; task_id: string; agent_profile_id?: string; status: string; started_at?: string; ended_at?: string; error_msg?: string; tag?: string; session_id?: string; actions?: { can_investigate?: boolean; can_apply_investigation?: boolean; can_delete?: boolean; can_stop?: boolean; can_retry?: boolean; can_continue?: boolean } }; team_id?: string; agent_id?: string }>(
    `/runs/${encodeURIComponent(runId)}`
  )
  const r = raw.run
  return {
    id: r.id,
    taskId: r.task_id,
    profileId: r.agent_profile_id,
    status: normalizeRunStatus(r.status),
    startedAt: r.started_at,
    endedAt: r.ended_at,
    error: r.error_msg,
    tag: r.tag,
    sessionId: r.session_id,
    teamId: raw.team_id,
    agentId: raw.agent_id,
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
  profileKey?: string
  taskId?: string
  investigatesRunId?: string
  appliesInvestigationRunId?: string
  limit?: number
  offset?: number
}): Promise<ListRunsResponse> {
  const params = new URLSearchParams()
  if (opts?.status) params.set('status', opts.status)
  if (opts?.tagPrefix) params.set('tag_prefix', opts.tagPrefix)
  if (opts?.profileKey) params.set('profile_key', opts.profileKey)
  if (opts?.taskId) params.set('task_id', opts.taskId)
  if (opts?.investigatesRunId) params.set('investigates_run_id', opts.investigatesRunId)
  if (opts?.appliesInvestigationRunId) params.set('applies_investigation_run_id', opts.appliesInvestigationRunId)
  if (opts?.limit !== undefined) params.set('limit', String(opts.limit))
  if (opts?.offset !== undefined) params.set('offset', String(opts.offset))
  const qs = params.toString()
  const endpoint = `/runs${qs ? `?${qs}` : ''}`

  const raw = await apiRequest<{ runs?: Array<{ id: string; task_id: string; agent_profile_id?: string; status: string; started_at?: string; ended_at?: string; error_msg?: string; tag?: string; session_id?: string }>; total?: number; has_more?: boolean }>(endpoint)
  const runs = (raw.runs ?? []).map((r) => ({
    id: r.id,
    taskId: r.task_id,
    profileId: r.agent_profile_id,
    status: normalizeRunStatus(r.status),
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
 * Retry a heartbeat run by re-triggering the mapped team/member heartbeat.
 */
export async function retryRun(runId: string): Promise<TriggerResponse> {
  return apiRequest<TriggerResponse>(
    `/runs/${encodeURIComponent(runId)}/retry`,
    {
      method: 'POST',
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
    status: normalizeRunStatus(r.status),
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
    status: normalizeRunStatus(r.status),
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
  const shortType = EVENT_TYPE_MAP[raw.event_type] ?? (raw.event_type.toLowerCase().replace('run_event_type_', '') as RunEvent['eventType'])

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
// Task & Run Creation (Chat)
// ============================================================================

export interface TaskDetails {
  id: string
  title: string
  description: string
  scopePath: string
  projectRoot?: string
}

/**
 * Create a task for agent execution.
 */
export async function createTask(opts: {
  title: string
  description: string
  scopePath: string
  projectRoot?: string
}): Promise<TaskDetails> {
  const raw = await apiRequest<{ task: { id: string; title: string; description: string; scope_path: string; project_root?: string } }>(
    '/tasks',
    {
      method: 'POST',
      body: JSON.stringify({
        task: {
          title: opts.title,
          description: opts.description,
          scope_path: opts.scopePath,
          project_root: opts.projectRoot,
        },
      }),
    }
  )
  const t = raw.task
  return {
    id: t.id,
    title: t.title,
    description: t.description,
    scopePath: t.scope_path,
    projectRoot: t.project_root,
  }
}

/**
 * Create a run for a task.
 */
export async function createRun(opts: {
  taskId: string
  profileKey?: string
}): Promise<RunDetails> {
  const body: Record<string, unknown> = { task_id: opts.taskId }
  if (opts.profileKey) {
    body.profile_ref = { profile_key: opts.profileKey }
  }
  const raw = await apiRequest<{ run: { id: string; task_id: string; agent_profile_id?: string; status: string; started_at?: string; ended_at?: string; error_msg?: string; tag?: string; session_id?: string; actions?: { can_investigate?: boolean; can_apply_investigation?: boolean; can_delete?: boolean; can_stop?: boolean; can_retry?: boolean; can_continue?: boolean } } }>(
    '/runs',
    {
      method: 'POST',
      body: JSON.stringify(body),
    }
  )
  const r = raw.run
  return {
    id: r.id,
    taskId: r.task_id,
    profileId: r.agent_profile_id,
    status: normalizeRunStatus(r.status),
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

// ============================================================================
// Team State Operations (Handoff, Task Board, Decisions)
// ============================================================================

export async function getLastHandoff(teamId: string, agentId: string): Promise<HandoffResponse> {
  return apiRequest<HandoffResponse>(`/teams/${encodeURIComponent(teamId)}/members/${encodeURIComponent(agentId)}/handoff`)
}

export async function getHandoffHistory(
  teamId: string,
  opts?: { agent?: string; last?: number }
): Promise<HandoffHistoryResponse> {
  const params = new URLSearchParams()
  if (opts?.agent) params.set('agent', opts.agent)
  if (opts?.last) params.set('last', String(opts.last))
  const qs = params.toString()
  return apiRequest<HandoffHistoryResponse>(`/teams/${encodeURIComponent(teamId)}/handoff-history${qs ? `?${qs}` : ''}`)
}

export async function clearHandoffHistory(teamId: string, agentId?: string): Promise<void> {
  const params = agentId ? `?agent=${encodeURIComponent(agentId)}` : ''
  await apiRequest<undefined>(`/teams/${encodeURIComponent(teamId)}/handoff-history${params}`, {
    method: 'DELETE',
  })
}

export async function clearLastHandoff(teamId: string, agentId: string): Promise<void> {
  await apiRequest<undefined>(`/teams/${encodeURIComponent(teamId)}/members/${encodeURIComponent(agentId)}/handoff`, {
    method: 'DELETE',
  })
}

export async function getTaskBoard(teamId: string): Promise<TaskBoardResponse> {
  return apiRequest<TaskBoardResponse>(`/teams/${encodeURIComponent(teamId)}/tasks`)
}

export async function addTask(teamId: string, request: AddTaskRequest): Promise<TeamTask> {
  return apiRequest<TeamTask>(`/teams/${encodeURIComponent(teamId)}/tasks`, {
    method: 'POST',
    body: JSON.stringify(request),
  })
}

export async function updateTask(
  teamId: string,
  taskId: string,
  request: UpdateTaskRequest
): Promise<TeamTask> {
  return apiRequest<TeamTask>(`/teams/${encodeURIComponent(teamId)}/tasks/${encodeURIComponent(taskId)}`, {
    method: 'PATCH',
    body: JSON.stringify(request),
  })
}

export async function deleteTask(teamId: string, taskId: string): Promise<void> {
  await apiRequest<undefined>(`/teams/${encodeURIComponent(teamId)}/tasks/${encodeURIComponent(taskId)}`, {
    method: 'DELETE',
  })
}

export async function getDecisions(
  teamId: string,
  opts?: { context?: string; status?: string; last?: number }
): Promise<DecisionListResponse> {
  const params = new URLSearchParams()
  if (opts?.context) params.set('context', opts.context)
  if (opts?.status) params.set('status', opts.status)
  if (opts?.last) params.set('last', String(opts.last))
  const qs = params.toString()
  return apiRequest<DecisionListResponse>(`/teams/${encodeURIComponent(teamId)}/decisions${qs ? `?${qs}` : ''}`)
}

export async function getAllPendingDecisions(): Promise<AllPendingDecisionsResponse> {
  return apiRequest<AllPendingDecisionsResponse>('/decisions/pending')
}

export async function addDecision(
  teamId: string,
  request: AddDecisionRequest
): Promise<DecisionEntry> {
  return apiRequest<DecisionEntry>(`/teams/${encodeURIComponent(teamId)}/decisions`, {
    method: 'POST',
    body: JSON.stringify(request),
  })
}

export async function updateDecision(
  teamId: string,
  decisionId: string,
  request: UpdateDecisionRequest
): Promise<DecisionEntry> {
  return apiRequest<DecisionEntry>(`/teams/${encodeURIComponent(teamId)}/decisions/${encodeURIComponent(decisionId)}`, {
    method: 'PATCH',
    body: JSON.stringify(request),
  })
}

export async function deleteDecision(teamId: string, decisionId: string): Promise<void> {
  await apiRequest<undefined>(`/teams/${encodeURIComponent(teamId)}/decisions/${encodeURIComponent(decisionId)}`, {
    method: 'DELETE',
  })
}

// ============================================================================
// Knowledge Log
// ============================================================================

export async function getKnowledge(
  teamId: string,
  opts?: { topic?: string; last?: number }
): Promise<KnowledgeListResponse> {
  const params = new URLSearchParams()
  if (opts?.topic) params.set('topic', opts.topic)
  if (opts?.last) params.set('last', String(opts.last))
  const qs = params.toString()
  return apiRequest<KnowledgeListResponse>(`/teams/${encodeURIComponent(teamId)}/knowledge${qs ? `?${qs}` : ''}`)
}

export async function addKnowledge(
  teamId: string,
  request: AddKnowledgeRequest
): Promise<KnowledgeEntry> {
  return apiRequest<KnowledgeEntry>(`/teams/${encodeURIComponent(teamId)}/knowledge`, {
    method: 'POST',
    body: JSON.stringify(request),
  })
}

export async function updateKnowledge(
  teamId: string,
  knowledgeId: string,
  request: UpdateKnowledgeRequest
): Promise<KnowledgeEntry> {
  return apiRequest<KnowledgeEntry>(`/teams/${encodeURIComponent(teamId)}/knowledge/${encodeURIComponent(knowledgeId)}`, {
    method: 'PATCH',
    body: JSON.stringify(request),
  })
}

export async function deleteKnowledge(teamId: string, knowledgeId: string): Promise<void> {
  await apiRequest<undefined>(`/teams/${encodeURIComponent(teamId)}/knowledge/${encodeURIComponent(knowledgeId)}`, {
    method: 'DELETE',
  })
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
