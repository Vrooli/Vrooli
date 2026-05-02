import { vi, type Mock } from 'vitest'

export interface FetchCall {
  input: RequestInfo | URL
  init?: RequestInit
}

interface FetchGuardOptions {
  allow?: Array<string | RegExp | ((url: string) => boolean)>
  response?: Response
}

export interface FetchGuard {
  calls: FetchCall[]
  fetchMock: Mock<[input: RequestInfo | URL, init?: RequestInit], Promise<Response>>
  restore: () => void
}

function requestUrl(input: RequestInfo | URL): string {
  if (typeof input === 'string') return input
  if (input instanceof URL) return input.toString()
  return input.url
}

function isAllowed(url: string, allow: NonNullable<FetchGuardOptions['allow']>): boolean {
  return allow.some((rule) => {
    if (typeof rule === 'string') return url.includes(rule)
    if (rule instanceof RegExp) return rule.test(url)
    return rule(url)
  })
}

export function installFetchGuard(options: FetchGuardOptions = {}): FetchGuard {
  const calls: FetchCall[] = []
  const originalFetch = globalThis.fetch
  const allow = options.allow ?? []
  const defaultResponse = options.response ?? new Response('{}', {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })

  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = requestUrl(input)
    calls.push({ input, init })

    if (isAllowed(url, allow)) {
      return defaultResponse.clone()
    }

    throw new Error(`Unexpected fetch in unit test: ${url}`)
  })

  globalThis.fetch = fetchMock as unknown as typeof fetch

  return {
    calls,
    fetchMock,
    restore: () => {
      globalThis.fetch = originalFetch
    },
  }
}

export function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    statusText: init.statusText,
    headers: {
      'Content-Type': 'application/json',
      ...Object.fromEntries(new Headers(init.headers).entries()),
    },
  })
}
