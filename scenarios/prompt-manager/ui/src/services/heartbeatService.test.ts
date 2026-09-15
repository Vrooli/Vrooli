import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { connectSlice4Request } from '@/lib/api'
import {
  continueRun,
  getHeartbeat,
  getRunEvents,
  getTeamRunAccounting,
  listHeartbeats,
  listTeamLogs,
  listRuns,
  newConversationRequestId,
  retryRun,
  resetHeartbeatServiceCachesForTests,
} from './heartbeatService'

vi.mock('@/lib/api', () => ({
  API_BASE: 'http://example.test/api/v1',
  // These tests exercise the shared REST error/fetch fallback used by routes
  // outside the migrated slice. Generated-client routing has focused contract
  // tests at the adapter boundary.
  connectSlice4Request: vi.fn().mockResolvedValue({ handled: false }),
}))

vi.mock('@vrooli/api-base', () => ({
  buildApiUrl: (endpoint: string, { baseUrl }: { baseUrl: string }) => `${baseUrl}${endpoint}`,
}))

function mockFetchResponse(response: Response) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response))
}

describe.each(['Connect', 'REST'] as const)('run event wire boundary over %s', (transport) => {
  afterEach(() => {
    vi.mocked(connectSlice4Request).mockResolvedValue({ handled: false })
    vi.unstubAllGlobals()
  })

  function respond(data: unknown) {
    vi.mocked(connectSlice4Request).mockResolvedValue({ handled: transport === 'Connect', data })
    mockFetchResponse(new Response(JSON.stringify(data), {
      status: 200, headers: { 'content-type': 'application/json' },
    }))
  }

  const identity = {
    id: 'aad7f46e-f4d4-4559-9058-845534fc6135',
    run_id: 'bc4df2ce-f2aa-472c-acb6-af8a27410f14',
    sequence: '8', timestamp: '2026-09-12T13:15:02.364059066Z',
  }

  it('retains the observed unspecified event and its position without inventing a payload', async () => {
    respond({ events: [identity] })
    const [event] = await getRunEvents(identity.run_id)
    expect(event).toMatchObject({ id: identity.id, runId: identity.run_id, sequence: 8, eventType: 'unspecified' })
    expect(event?.data).toEqual(identity)
  })

  it('accepts protobuf JSON camel names and preserves additional payload and event fields', async () => {
    const wire = {
      id: identity.id, runId: identity.run_id, timestamp: identity.timestamp,
      eventType: 'RUN_EVENT_TYPE_TOOL_CALL', toolCall: { toolName: 'read', input: { path: 'file' }, future: 17 },
      futureEnvelope: { retained: true },
    }
    respond({ events: [wire] })
    const [event] = await getRunEvents(identity.run_id)
    expect(event).toMatchObject({ sequence: 0, eventType: 'tool_call', runId: identity.run_id,
      data: { tool_name: 'read', toolName: 'read', future: 17 }, raw: wire })
  })

  it('retains future enum names, numbers and payloads for the generic renderer', async () => {
    const events = [
      { ...identity, event_type: 'RUN_EVENT_TYPE_FUTURE', future: { proof: 'retained' } },
      { ...identity, id: 'numeric', event_type: 9001, newPayload: { value: false } },
      { ...identity, id: 'known', event_type: 2, message: { role: 'assistant', content: 'hello' } },
    ]
    respond(events)
    const result = await getRunEvents(identity.run_id)
    expect(result.map((event) => event.eventType)).toEqual(['future', '9001', 'message'])
    expect(result[0]?.data).toEqual(events[0])
    expect(result[1]?.data).toEqual(events[1])
    expect(result[2]?.data).toEqual({ role: 'assistant', content: 'hello' })
  })

  it('preserves typed payload variants such as progress and cost instead of returning empty data', async () => {
    respond({ events: [
      { ...identity, event_type: 'RUN_EVENT_TYPE_STATUS', progress: { phase: 'executing', percent: 5 } },
      { ...identity, id: 'cost', event_type: 'RUN_EVENT_TYPE_METRIC', cost: { total_cost_usd: 0.12 } },
      { ...identity, id: 'artifact', event_type: 'RUN_EVENT_TYPE_ARTIFACT', artifact: { path: 'proof' } },
    ] })
    expect((await getRunEvents(identity.run_id)).map((event) => event.data)).toEqual([
      { phase: 'executing', percent: 5 }, { total_cost_usd: 0.12 }, { path: 'proof' },
    ])
  })

  it.each([null, undefined, []])('accepts empty protobuf event collections (%s)', async (events) => {
    respond({ events })
    await expect(getRunEvents(identity.run_id)).resolves.toEqual([])
  })

  it.each([
    { ...identity, event_type: {} },
    { ...identity, sequence: '8junk' },
    { ...identity, sequence: '9007199254740992' },
    { ...identity, id: null },
    { ...identity, event_type: 'RUN_EVENT_TYPE_MESSAGE', message: 'invalid' },
  ])('rejects malformed wire data at the boundary with context', async (event) => {
    respond({ events: [event] })
    await expect(getRunEvents(identity.run_id)).rejects.toThrow('Validation failed at run events')
  })
})

describe.each(['Connect', 'REST'] as const)('heartbeatService team logs over %s', (transport) => {
  beforeEach(() => {
    vi.mocked(connectSlice4Request).mockResolvedValue({ handled: false })
  })

  afterEach(() => {
    vi.mocked(connectSlice4Request).mockResolvedValue({ handled: false })
    vi.unstubAllGlobals()
  })

  function respond(data: unknown) {
    if (transport === 'Connect') {
      vi.mocked(connectSlice4Request).mockResolvedValue({ handled: true, data })
    } else {
      mockFetchResponse(new Response(JSON.stringify(data), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      }))
    }
  }

  it.each([
    { encoding: 'null', logs: null },
    { encoding: 'omitted', logs: undefined },
    { encoding: 'array', logs: [] },
  ])('returns an empty log list for an empty page encoded as $encoding', async ({ logs }) => {
    respond({ teamId: 'effort-supervision', logs, total: 0, hasMore: false })

    await expect(listTeamLogs('effort-supervision')).resolves.toEqual({
      teamId: 'effort-supervision', logs: [], total: 0, hasMore: false,
    })
  })

  it('preserves log identities and pagination metadata', async () => {
    const logs = [{
      agentId: 'effort-supervisor',
      agentDisplayName: 'Effort Supervisor',
      filename: '2026-09-12T10:00:00Z.log',
      timestamp: '2026-09-12T10:00:00Z',
      status: 'completed',
    }]
    respond({ teamId: 'effort-supervision', logs, total: 3, hasMore: true })

    await expect(listTeamLogs('effort-supervision', { limit: 1 })).resolves.toEqual({
      teamId: 'effort-supervision', logs, total: 3, hasMore: true,
    })
  })

  it('keeps the total for an empty page beyond the available logs', async () => {
    respond({ teamId: 'effort-supervision', logs: null, total: 3, hasMore: false })

    await expect(listTeamLogs('effort-supervision', { offset: 10 })).resolves.toEqual({
      teamId: 'effort-supervision', logs: [], total: 3, hasMore: false,
    })
  })
})

describe('heartbeatService api errors', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
    vi.clearAllMocks()
    resetHeartbeatServiceCachesForTests()
  })

  it('extracts structured JSON error messages', async () => {
    mockFetchResponse(
      new Response(JSON.stringify({ error: 'invalid depth value' }), {
        status: 400,
        statusText: 'Bad Request',
        headers: {
          'content-type': 'application/json',
          'x-vrooli-proxy-hop': 'ui-proxy',
        },
      })
    )

    await expect(listHeartbeats('team-a')).rejects.toThrow(
      'API error: 400 Bad Request (hop: ui-proxy) - invalid depth value'
    )
  })

  it('preserves 404 detection for getHeartbeat', async () => {
    mockFetchResponse(
      new Response(JSON.stringify({ error: 'not found' }), {
        status: 404,
        statusText: 'Not Found',
        headers: { 'content-type': 'application/json' },
      })
    )

    await expect(getHeartbeat('team-a', 'agent-a')).resolves.toBeNull()
  })
})

describe('heartbeatService listHeartbeats coalescing', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
    vi.clearAllMocks()
    resetHeartbeatServiceCachesForTests()
  })

  it('coalesces concurrent requests for the same team', async () => {
    const payload = [{
      teamId: 'team-a',
      agentId: 'agent-a',
      enabled: true,
      schedule: '*/5 * * * *',
      createdAt: '2026-02-17T00:00:00Z',
      updatedAt: '2026-02-17T00:00:00Z',
    }]

    mockFetchResponse(
      new Response(JSON.stringify(payload), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      })
    )

    const [a, b, c] = await Promise.all([
      listHeartbeats('team-a'),
      listHeartbeats('team-a'),
      listHeartbeats('team-a'),
    ])

    expect(a).toEqual(payload)
    expect(b).toEqual(payload)
    expect(c).toEqual(payload)
    expect(fetch).toHaveBeenCalledTimes(1)
  })
})

describe('heartbeatService listRuns filters', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
    vi.clearAllMocks()
    resetHeartbeatServiceCachesForTests()
  })

  it('forwards profile_key and task_id filters', async () => {
    mockFetchResponse(
      new Response(JSON.stringify({ runs: [], total: 0, has_more: false }), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      })
    )

    await listRuns({
      profileKey: 'prompt-manager-heartbeat',
      taskId: 'task-123',
      limit: 10,
    })

    expect(fetch).toHaveBeenCalledTimes(1)
    const callArgs = vi.mocked(fetch).mock.calls[0] ?? []
    const url = String(callArgs[0] as string | URL)
    expect(url).toContain('/runs?')
    expect(url).toContain('profile_key=prompt-manager-heartbeat')
    expect(url).toContain('task_id=task-123')
    expect(url).toContain('limit=10')
  })
})

describe.each(['Connect', 'REST'] as const)('run accounting wire evidence over %s', (transport) => {
  afterEach(() => {
    vi.mocked(connectSlice4Request).mockResolvedValue({ handled: false })
    vi.unstubAllGlobals()
  })

  it('retains actual models, resume identity and terminal standing with missing metrics unknown', async () => {
    const data = { runs: [{
      id: 'owner-run', task_id: 'task', status: 'RUN_STATUS_COMPLETE',
      requested_model: 'requested', actual_model: 'actual', harness_kind: 'codex',
      session_id: 'session', import_source_harness: 'codex', import_source_session_id: 'session',
      terminal_class: 'verdict', stop_reason: 'blocked',
      summary: { tokens_used: 123 },
      work_references: [{ kind: 'effort', id: 'effort-a', relationship: 'supervisor', visibility: 'WORK_REFERENCE_VISIBILITY_PUBLIC', verified: true }],
    }, { id: 'missing', task_id: 'other', status: 'RUN_STATUS_RUNNING' }], total: 2, has_more: true }
    vi.mocked(connectSlice4Request).mockResolvedValue({ handled: transport === 'Connect', data })
    mockFetchResponse(new Response(JSON.stringify(data), { status: 200 }))
    const result = await listRuns()
    expect(result.hasMore).toBe(true)
    expect(result.runs[0]).toMatchObject({
      requestedModel: 'requested', actualModel: 'actual', harnessKind: 'codex', sessionId: 'session',
      importSourceHarness: 'codex', importSourceSessionId: 'session', terminalClass: 'verdict', stopReason: 'blocked',
      reportedSummary: { tokensUsed: 123, costEstimate: null }, workReferences: data.runs[0]?.work_references,
    })
    expect(result.runs[1]).toMatchObject({ actualModel: null, reportedSummary: null })
    expect(result.runs[0]).not.toHaveProperty('accepted', true)
  })

  it.each([
    { summary: undefined, expected: null },
    { summary: {}, expected: { tokensUsed: null, costEstimate: null } },
    { summary: { tokens_used: 0, cost_estimate: 0 }, expected: { tokensUsed: 0, costEstimate: 0 } },
    { summary: { costEstimate: 0.4 }, expected: { tokensUsed: null, costEstimate: 0.4 } },
  ])('preserves reported metric presence without claiming qualified usage ($summary)', async ({ summary, expected }) => {
    const data = { runs: [{ id: 'owner-run', task_id: 'task', status: 'RUN_STATUS_RUNNING', summary,
      requestedModel: 'requested-only', terminalClass: 'interruption', stopReason: 'usage_window' }], total: 1 }
    vi.mocked(connectSlice4Request).mockResolvedValue({ handled: transport === 'Connect', data })
    mockFetchResponse(new Response(JSON.stringify(data), { status: 200 }))
    const [run] = (await listRuns()).runs
    expect(run).toMatchObject({ actualModel: null, requestedModel: 'requested-only', reportedSummary: expected,
      terminalClass: 'interruption', stopReason: 'usage_window' })
  })

  it.each([{ tokens_used: -1 }, { tokens_used: 'unknown' }, { cost_estimate: 'free' }, 'unavailable'])(
    'rejects malformed reported metrics instead of showing zero (%s)', async (summary) => {
      const data = { runs: [{ id: 'owner-run', task_id: 'task', status: 'RUN_STATUS_RUNNING', summary }], total: 1 }
      vi.mocked(connectSlice4Request).mockResolvedValue({ handled: transport === 'Connect', data })
      mockFetchResponse(new Response(JSON.stringify(data), { status: 200 }))
      await expect(listRuns()).rejects.toThrow()
    },
  )
})

describe.each(['Connect', 'REST'] as const)('team accounting projection over %s', (transport) => {
  afterEach(() => {
    vi.mocked(connectSlice4Request).mockResolvedValue({ handled: false })
    vi.unstubAllGlobals()
  })

  function respond(overrides: Record<string, unknown> = {}) {
    const data = {
      teamId: 'team', agentId: 'member', windowStart: '2026-09-12T00:00:00Z', windowEnd: '2026-09-13T00:00:00Z', observedAt: '2026-09-12T15:00:00Z',
      knownRuns: 1, observedRuns: 0, unavailableRuns: 1, unqueriedRuns: 0, observedExecutions: 0, duplicateExecutions: 0,
      actualModels: {}, unknownModelExecutions: 0, runtimeStates: {}, terminalReasons: {},
      usage: { tokens: null, costUSD: null, qualifiedRuns: 0, reportedTokenRuns: 0, reportedCostRuns: 0, partial: true },
      coverage: { partial: true, declarationsRead: 1, declarationLimit: 500, ownerReadLimit: 50, invalidTimestamps: 0, limitations: ['Partial declarations'] },
      runs: [{ runId: 'known', agentIds: ['member'], declaredAt: '2026-09-12T10:00:00Z', availability: 'unavailable' }],
      ...overrides,
    }
    vi.mocked(connectSlice4Request).mockResolvedValue({ handled: transport === 'Connect', data })
    mockFetchResponse(new Response(JSON.stringify(data), { status: 200 }))
  }

  it('retains known owner identities through an outage with usage unknown', async () => {
    respond()
    const got = await getTeamRunAccounting('team', 'member')
    expect(got).toMatchObject({ knownRuns: 1, observedRuns: 0, unavailableRuns: 1, usage: { tokens: null, costUSD: null }, coverage: { partial: true } })
    expect(got.runs[0]?.runId).toBe('known')
    expect(connectSlice4Request).toHaveBeenCalledWith('/runs?accounting=true&team_id=team&agent_id=member', expect.anything())
  })

  it.each([{ teamId: 'other' }, { agentId: 'other' }, { knownRuns: undefined }, { unavailableRuns: 0 }, { usage: {} }])(
    'rejects wrong attribution or incomplete coverage instead of fabricating totals (%s)', async (overrides) => {
      respond(overrides)
      await expect(getTeamRunAccounting('team', 'member')).rejects.toThrow()
    },
  )
})

describe('heartbeatService retryRun', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
    vi.clearAllMocks()
    resetHeartbeatServiceCachesForTests()
  })

  it('calls retry endpoint for the run', async () => {
    mockFetchResponse(
      new Response(JSON.stringify({ teamId: 'team-1', agentId: 'agent-1', runId: 'run-2', status: 'running' }), {
        status: 202,
        headers: { 'content-type': 'application/json' },
      })
    )

    const resp = await retryRun('run-1')

    expect(resp.runId).toBe('run-2')
    expect(fetch).toHaveBeenCalledTimes(1)
    const retryCallArgs = vi.mocked(fetch).mock.calls[0] ?? []
    const retryUrl = String(retryCallArgs[0] as string | URL)
    expect(retryUrl).toContain('/runs/run-1/retry')
    expect((retryCallArgs[1] as RequestInit | undefined)?.method).toBe('POST')  // eslint-disable-line @typescript-eslint/no-unnecessary-type-assertion -- vi.mocked returns unknown[]
  })
})

describe('heartbeatService conversation turns', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('forwards the stable request id so a retried turn can be deduplicated', async () => {
    mockFetchResponse(
      new Response(JSON.stringify({ run: { id: 'run-1', task_id: 'task-1', status: 'RUN_STATUS_RUNNING' } }), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      })
    )

    await continueRun('run-1', 'second turn', { requestId: 'turn-2' })

    expect(fetch).toHaveBeenCalledTimes(1)
    const callArgs = vi.mocked(fetch).mock.calls[0] ?? []
    const url = String(callArgs[0] as string | URL)
    expect(url).toContain('/runs/run-1/continue')
    const init = callArgs[1]
    expect(init?.method).toBe('POST')
    expect(JSON.parse(init?.body as string)).toEqual({
      message: 'second turn',
      request_id: 'turn-2',
    })
  })

  it('generates a UUID-shaped request identity', () => {
    expect(newConversationRequestId()).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/
    )
  })
})
