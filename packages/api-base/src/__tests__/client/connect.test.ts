import { describe, expect, it, vi } from 'vitest'

import { createScenarioConnectTransport } from '../../client/connect.js'

const createConnectTransport = vi.hoisted(() => vi.fn((opts: unknown) => ({ kind: 'transport', opts })))

vi.mock('@connectrpc/connect-web', () => ({
  createConnectTransport,
}))

describe('createScenarioConnectTransport', () => {
  it('passes an explicit base URL and fetch implementation through', () => {
    const fetchImpl = vi.fn() as unknown as typeof fetch
    const transport = createScenarioConnectTransport({
      baseUrl: 'https://api.example.test/api/v1',
      fetch: fetchImpl,
    })

    expect(transport).toEqual({
      kind: 'transport',
      opts: {
        baseUrl: 'https://api.example.test/api/v1',
        fetch: fetchImpl,
      },
    })
  })

  it('resolves the API base when no base URL is provided', () => {
    createScenarioConnectTransport()

    expect(createConnectTransport).toHaveBeenLastCalledWith(
      expect.objectContaining({
        baseUrl: 'http://localhost:3000',
      })
    )
  })
})
