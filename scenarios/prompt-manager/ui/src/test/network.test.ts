import { afterEach, describe, expect, it } from 'vitest'
import { installFetchGuard, jsonResponse } from './network'

let restoreFetch: (() => void) | null = null

afterEach(() => {
  restoreFetch?.()
  restoreFetch = null
})

describe('installFetchGuard', () => {
  it('rejects unexpected fetches with the requested URL', async () => {
    const guard = installFetchGuard()
    restoreFetch = guard.restore

    await expect(fetch('http://localhost:3000/api/v1/world-scale'))
      .rejects
      .toThrow('Unexpected fetch in unit test: http://localhost:3000/api/v1/world-scale')
    expect(guard.calls).toHaveLength(1)
  })

  it('allows declared fetches and returns a JSON response', async () => {
    const guard = installFetchGuard({
      allow: ['/api/v1/health'],
      response: jsonResponse({ ok: true }),
    })
    restoreFetch = guard.restore

    const response = await fetch('/api/v1/health')

    await expect(response.json()).resolves.toEqual({ ok: true })
    expect(guard.fetchMock).toHaveBeenCalledOnce()
  })
})
