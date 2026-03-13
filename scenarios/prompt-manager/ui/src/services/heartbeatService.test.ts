import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  createInvestigationRun,
  getHeartbeat,
  listHeartbeats,
  listRuns,
  retryRun,
  resetHeartbeatServiceCachesForTests,
} from './heartbeatService'

vi.mock('@vrooli/api-base', () => ({
  resolveApiBase: () => 'http://example.test/api/v1',
  buildApiUrl: (endpoint: string, { baseUrl }: { baseUrl: string }) => `${baseUrl}${endpoint}`,
}))

function mockFetchResponse(response: Response) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response))
}

describe('heartbeatService api errors', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
    vi.clearAllMocks()
    resetHeartbeatServiceCachesForTests()
  })

  it('returns a concise upstream message for HTML 502 responses', async () => {
    mockFetchResponse(
      new Response('<!DOCTYPE html><html><head><title>502: Bad gateway</title></head><body>Cloudflare</body></html>', {
        status: 502,
        statusText: 'Bad Gateway',
        headers: { 'content-type': 'text/html' },
      })
    )

    await expect(createInvestigationRun(['run-123'])).rejects.toThrow(
      'API error: 502 Bad Gateway - Upstream gateway error before prompt-manager API (edge/tunnel or host).'
    )
  })

  it('includes hop marker when backend sets diagnostic header', async () => {
    mockFetchResponse(
      new Response('<!DOCTYPE html><html><body>bad gateway</body></html>', {
        status: 502,
        statusText: 'Bad Gateway',
        headers: {
          'content-type': 'text/html',
          'x-vrooli-error-hop': 'prompt-manager-api->agent-manager',
        },
      })
    )

    await expect(createInvestigationRun(['run-123'])).rejects.toThrow(
      'API error: 502 Bad Gateway - Upstream gateway error at prompt-manager-api->agent-manager.'
    )
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
