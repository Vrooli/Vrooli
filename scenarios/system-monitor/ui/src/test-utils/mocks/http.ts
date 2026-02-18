import { vi } from 'vitest'

export type FetchResponseInit = {
  ok?: boolean
  status?: number
  body?: unknown
}

export const createFetchResponse = (init: FetchResponseInit = {}): Response => {
  const { ok = true, status = 200, body = {} } = init
  return {
    ok,
    status,
    json: vi.fn().mockResolvedValue(body),
    text: vi.fn().mockResolvedValue(JSON.stringify(body)),
  } as unknown as Response
}

export const mockFetchResolved = (init: FetchResponseInit = {}) => {
  const response = createFetchResponse(init)
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response))
  return response
}
