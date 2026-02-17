import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createInvestigationRun, getHeartbeat, listHeartbeats } from './heartbeatService'

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
