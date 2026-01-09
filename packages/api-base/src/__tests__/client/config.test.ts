/**
 * Tests for runtime configuration utilities
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  fetchRuntimeConfig,
  getInjectedConfig,
  resolveWithConfig,
} from '../../client/config.js'
import { mockWindow } from '../helpers/mock-window.js'

describe('fetchRuntimeConfig', () => {
  beforeEach(() => {
    // Setup global fetch mock
    global.fetch = vi.fn()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('fetches config from default endpoint', async () => {
    const mockConfig = {
      apiUrl: 'http://localhost:8080/api/v1',
      wsUrl: 'ws://localhost:8080/ws',
      apiPort: '8080',
      uiPort: '3000',
    }

    ;(global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => mockConfig,
    })

    const result = await fetchRuntimeConfig()

    expect(result).toEqual(mockConfig)
    expect(global.fetch).toHaveBeenCalledWith(
      './config',
      expect.objectContaining({
        headers: { accept: 'application/json' },
      })
    )
  })

  it('uses custom endpoint', async () => {
    const mockConfig = {
      apiUrl: 'http://localhost:8080/api/v1',
      wsUrl: 'ws://localhost:8080/ws',
      apiPort: '8080',
      uiPort: '3000',
    }

    ;(global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => mockConfig,
    })

    await fetchRuntimeConfig({ endpoint: '/custom/config' })

    expect(global.fetch).toHaveBeenCalledWith(
      '/custom/config',
      expect.anything()
    )
  })

  it('returns null when fetch fails', async () => {
    ;(global.fetch as any).mockResolvedValueOnce({
      ok: false,
      status: 404,
    })

    const result = await fetchRuntimeConfig()

    expect(result).toBeNull()
  })

  it('returns null when fetch throws', async () => {
    ;(global.fetch as any).mockRejectedValueOnce(new Error('Network error'))

    const result = await fetchRuntimeConfig()

    expect(result).toBeNull()
  })

  it('respects timeout', async () => {
    // Mock a long-running fetch that will be aborted
    ;(global.fetch as any).mockImplementation(() =>
      new Promise((_, reject) => {
        setTimeout(() => reject(new Error('AbortError')), 1000)
      })
    )

    const result = await fetchRuntimeConfig({ timeout: 50 })

    expect(result).toBeNull()
  })

  it('returns null when fetch is not available', async () => {
    const originalFetch = global.fetch
    ;(global as any).fetch = undefined

    const result = await fetchRuntimeConfig()

    expect(result).toBeNull()

    global.fetch = originalFetch
  })
})

describe('getInjectedConfig', () => {
  it('returns config from default global', () => {
    const mockConfig = {
      apiUrl: 'http://localhost:8080/api/v1',
      wsUrl: 'ws://localhost:8080/ws',
      apiPort: '8080',
      uiPort: '3000',
    }

    const win = mockWindow({})
    ;(win as any).__VROOLI_CONFIG__ = mockConfig

    const result = getInjectedConfig({ windowObject: win })

    expect(result).toEqual(mockConfig)
  })

  it('checks custom global names', () => {
    const mockConfig = {
      apiUrl: 'http://localhost:8080/api/v1',
      wsUrl: 'ws://localhost:8080/ws',
      apiPort: '8080',
      uiPort: '3000',
    }

    const win = mockWindow({})
    ;(win as any).__CUSTOM_CONFIG__ = mockConfig

    const result = getInjectedConfig({
      windowObject: win,
      configGlobalName: '__CUSTOM_CONFIG__',
    })

    expect(result).toEqual(mockConfig)
  })

  it('returns null when no config found', () => {
    const win = mockWindow({})

    const result = getInjectedConfig({ windowObject: win })

    expect(result).toBeNull()
  })

  it('returns null when window is undefined', () => {
    const result = getInjectedConfig({ windowObject: undefined })

    expect(result).toBeNull()
  })

  it('returns config even if incomplete', () => {
    const partialConfig = {
      apiUrl: 'http://localhost:8080/api/v1',
      // Missing other fields
    }

    const win = mockWindow({})
    ;(win as any).__VROOLI_CONFIG__ = partialConfig

    const result = getInjectedConfig({ windowObject: win })

    expect(result).toEqual(partialConfig)
  })
})

describe('resolveWithConfig', () => {
  beforeEach(() => {
    global.fetch = vi.fn()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('uses injected config apiUrl', async () => {
    const mockConfig = {
      apiUrl: 'http://localhost:8080/api/v1',
      wsUrl: 'ws://localhost:8080/ws',
      apiPort: '8080',
      uiPort: '3000',
    }

    const win = mockWindow({})
    ;(win as any).__VROOLI_CONFIG__ = mockConfig

    const result = await resolveWithConfig({ windowObject: win })

    expect(result).toBe('http://localhost:8080/api/v1')
    expect(global.fetch).not.toHaveBeenCalled()
  })

  it('falls back to standard resolution when no config', async () => {
    const win = mockWindow({ hostname: 'localhost', origin: 'http://localhost:3000', port: '3000', host: 'localhost:3000' })
    const result = await resolveWithConfig({
      defaultPort: '8080',
      windowObject: win,
    })

    // Should fall back to standard resolution (UI origin on localhost)
    expect(result).toBe('http://localhost:3000')
    expect(global.fetch).not.toHaveBeenCalled()
  })

  it('does not fetch when on remote host resolving to origin', async () => {
    // When on remote host and resolve succeeds to that host, no fetch needed
    const result = await resolveWithConfig({
      defaultPort: '8080',
      windowObject: mockWindow({ hostname: 'remote.com', origin: 'https://remote.com' }),
    })

    // Should resolve to origin, not fetch
    expect(global.fetch).not.toHaveBeenCalled()
    expect(result).toBe('https://remote.com')
  })

  it('fetches config when remote host but resolves to localhost', async () => {
    const mockConfig = {
      apiUrl: 'http://remote.com:8080/api/v1',
      wsUrl: 'ws://remote.com:8080/ws',
      apiPort: '8080',
      uiPort: '3000',
    }

    ;(global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => mockConfig,
    })

    // Force resolution to localhost by not providing origin
    const win = mockWindow({ hostname: 'remote.com' })
    delete win.location?.origin

    const result = await resolveWithConfig({
      defaultPort: '8080',
      windowObject: win,
    })

    // On remote host but resolved to localhost - should fetch config
    expect(global.fetch).toHaveBeenCalled()
    expect(result).toBe('http://remote.com:8080/api/v1')
  })

  it('handles config fetch failure on remote host', async () => {
    ;(global.fetch as any).mockResolvedValueOnce({
      ok: false,
      status: 404,
    })

    // Force resolution to localhost
    const win = mockWindow({ hostname: 'remote.com' })
    delete win.location?.origin

    const result = await resolveWithConfig({
      defaultPort: '8080',
      windowObject: win,
    })

    // With no reliable origin, fall back to loopback API port
    expect(result).toBe('http://127.0.0.1:8080')
  })

  it('injected config takes priority over explicit URL', async () => {
    const mockConfig = {
      apiUrl: 'http://localhost:8080/api/v1',
      wsUrl: 'ws://localhost:8080/ws',
      apiPort: '8080',
      uiPort: '3000',
    }

    const win = mockWindow({})
    ;(win as any).__VROOLI_CONFIG__ = mockConfig

    const result = await resolveWithConfig({
      explicitUrl: 'https://custom.example.com/api',
      windowObject: win,
    })

    // Injected config takes priority over explicit URL
    expect(result).toBe('http://localhost:8080/api/v1')
  })

  it('explicit URL works without injected config', async () => {
    const result = await resolveWithConfig({
      explicitUrl: 'https://custom.example.com/api',
      windowObject: mockWindow({}),
    })

    // With no injected config, explicit URL is used via resolveApiBase
    expect(result).toBe('https://custom.example.com/api')
  })
})
