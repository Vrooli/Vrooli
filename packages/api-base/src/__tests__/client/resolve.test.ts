/**
 * Tests for API base URL resolution
 */

import { describe, it, expect } from 'vitest'
import { resolveApiBase, resolveWsBase } from '../../client/resolve.js'
import { mockWindow, mockLocalhost, mockRemoteHost, mockProxyContext } from '../helpers/mock-window.js'
import { mockProxyInfo } from '../helpers/test-utils.js'

describe('resolveApiBase', () => {
  describe('explicit URL', () => {
    it('uses explicit URL when provided', () => {
      const result = resolveApiBase({
        explicitUrl: 'https://explicit.com/api',
        windowObject: mockLocalhost(),
      })

      expect(result).toBe('https://explicit.com/api')
    })

    it('normalizes explicit URL (removes trailing slash)', () => {
      const result = resolveApiBase({
        explicitUrl: 'https://explicit.com/api/',
      })

      expect(result).toBe('https://explicit.com/api')
    })

    it('appends suffix when appendSuffix is true', () => {
      const result = resolveApiBase({
        explicitUrl: 'https://explicit.com',
        appendSuffix: true,
      })

      expect(result).toBe('https://explicit.com/api/v1')
    })

    it('does not duplicate suffix', () => {
      const result = resolveApiBase({
        explicitUrl: 'https://explicit.com/api/v1',
        appendSuffix: true,
      })

      expect(result).toBe('https://explicit.com/api/v1')
    })
  })

  describe('localhost context', () => {
    it('uses UI origin for localhost scenarios', () => {
      const result = resolveApiBase({
        windowObject: mockLocalhost('3000', '/'),
        defaultPort: '8080',
      })

      expect(result).toBe('http://localhost:3000')
    })

    it('keeps localhost hostname so requests stay inside the UI proxy', () => {
      const result = resolveApiBase({
        windowObject: mockLocalhost(),
      })

      expect(result).toBe('http://localhost:3000')
    })

    it('appends suffix when requested for localhost', () => {
      const result = resolveApiBase({
        windowObject: mockLocalhost(),
        defaultPort: '8080',
        appendSuffix: true,
      })

      expect(result).toBe('http://localhost:3000/api/v1')
    })
  })

  describe('remote host context', () => {
    it('uses window origin for remote host', () => {
      const result = resolveApiBase({
        windowObject: mockRemoteHost('example.com', '/'),
      })

      expect(result).toBe('https://example.com')
    })

    it('respects protocol from window', () => {
      const result = resolveApiBase({
        windowObject: mockRemoteHost('example.com', '/', false),
      })

      expect(result).toBe('http://example.com')
    })

    it('appends suffix for remote host', () => {
      const result = resolveApiBase({
        windowObject: mockRemoteHost('example.com'),
        appendSuffix: true,
      })

      expect(result).toBe('https://example.com/api/v1')
    })
  })

  describe('proxy metadata context', () => {
    it('reads proxy info from window global', () => {
      const proxyInfo = mockProxyInfo({
        primary: {
          port: 8080,
          label: 'UI',
          normalizedLabel: 'ui',
          slug: 'ui',
          source: 'test',
          isPrimary: true,
          path: '/apps/test/proxy',
          aliases: [],
        },
      })

      const result = resolveApiBase({
        windowObject: mockWindow({
          hostname: 'example.com',
          origin: 'https://example.com',
          pathname: '/apps/test/proxy/',
          globals: { __VROOLI_PROXY_INFO__: proxyInfo },
        }),
      })

      expect(result).toBe('/apps/test/proxy')
    })

    it('works with legacy __APP_MONITOR_PROXY_INFO__', () => {
      const proxyInfo = mockProxyInfo({
        primary: {
          port: 8080,
          label: 'UI',
          normalizedLabel: 'ui',
          slug: 'ui',
          source: 'test',
          isPrimary: true,
          path: '/apps/legacy/proxy',
          aliases: [],
        },
      })

      const result = resolveApiBase({
        windowObject: mockWindow({
          hostname: 'example.com',
          origin: 'https://example.com',
          pathname: '/apps/legacy/proxy/',
          globals: { __APP_MONITOR_PROXY_INFO__: proxyInfo },
        }),
      })

      expect(result).toBe('/apps/legacy/proxy')
    })

    it('supports custom proxy global names', () => {
      const proxyInfo = mockProxyInfo({
        primary: {
          port: 8080,
          label: 'UI',
          normalizedLabel: 'ui',
          slug: 'ui',
          source: 'test',
          isPrimary: true,
          path: '/custom/proxy',
          aliases: [],
        },
      })

      const result = resolveApiBase({
        windowObject: mockWindow({
          hostname: 'example.com',
          origin: 'https://example.com',
          pathname: '/custom/proxy/',
          globals: { __CUSTOM_PROXY__: proxyInfo },
        }),
        proxyGlobalNames: ['__CUSTOM_PROXY__'],
      })

      expect(result).toBe('/custom/proxy')
    })
  })

  describe('path-based proxy detection', () => {
    it('derives proxy base from /apps/ path pattern', () => {
      const result = resolveApiBase({
        windowObject: mockWindow({
          hostname: 'example.com',
          origin: 'https://example.com',
          pathname: '/apps/scenario-name/proxy/',
        }),
      })

      expect(result).toBe('https://example.com/apps/scenario-name/proxy')
    })

    it('derives proxy base from generic /proxy pattern', () => {
      const result = resolveApiBase({
        windowObject: mockWindow({
          hostname: 'example.com',
          origin: 'https://example.com',
          pathname: '/embed/custom-scenario/proxy/',
        }),
      })

      expect(result).toBe('https://example.com/embed/custom-scenario/proxy')
    })

    it('handles trailing paths after /proxy', () => {
      const result = resolveApiBase({
        windowObject: mockWindow({
          hostname: 'example.com',
          origin: 'https://example.com',
          pathname: '/apps/scenario/proxy/some/path',
        }),
      })

      expect(result).toBe('https://example.com/apps/scenario/proxy')
    })
  })

  describe('fallback behavior', () => {
    it('uses UI origin when a browser window exists', () => {
      const result = resolveApiBase({ windowObject: undefined })
      expect(result).toBe('http://localhost:3000')
    })

    it('falls back to loopback when no window is available', () => {
      const originalWindow = (globalThis as any).window
      delete (globalThis as any).window

      try {
        const result = resolveApiBase({ windowObject: undefined, defaultPort: '9999' })
        expect(result).toBe('http://127.0.0.1:9999')
      } finally {
        if (originalWindow) {
          (globalThis as any).window = originalWindow
        }
      }
    })
  })

  describe('suffix handling', () => {
    it('uses default suffix when appendSuffix is true', () => {
      const result = resolveApiBase({
        windowObject: mockLocalhost(),
        defaultPort: '8080',
        appendSuffix: true,
      })

      expect(result).toBe('http://localhost:3000/api/v1')
    })

    it('uses custom suffix', () => {
      const result = resolveApiBase({
        windowObject: mockLocalhost(),
        defaultPort: '8080',
        appendSuffix: true,
        apiSuffix: '/v2/api',
      })

      expect(result).toBe('http://localhost:3000/v2/api')
    })

    it('does not append suffix by default', () => {
      const result = resolveApiBase({
        windowObject: mockLocalhost(),
        defaultPort: '8080',
      })

      expect(result).not.toContain('/api')
    })
  })
})

describe('resolveWsBase', () => {
  it('converts http to ws', () => {
    const result = resolveWsBase({
      windowObject: mockLocalhost(),
      defaultPort: '8080',
    })

    expect(result).toBe('ws://localhost:3000')
  })

  it('converts https to wss', () => {
    const result = resolveWsBase({
      windowObject: mockRemoteHost('example.com'),
    })

    expect(result).toBe('wss://example.com')
  })

  it('appends WebSocket suffix when requested', () => {
    const result = resolveWsBase({
      windowObject: mockLocalhost(),
      defaultPort: '8080',
      appendSuffix: true,
    })

    expect(result).toBe('ws://localhost:3000/ws')
  })

  it('works with proxy context', () => {
    const proxyInfo = mockProxyInfo({
      primary: {
        port: 8080,
        label: 'UI',
        normalizedLabel: 'ui',
        slug: 'ui',
        source: 'test',
        isPrimary: true,
        path: '/apps/test/proxy',
        aliases: [],
      },
    })

    const result = resolveWsBase({
      windowObject: mockWindow({
        hostname: 'example.com',
        origin: 'https://example.com',
        pathname: '/apps/test/proxy/',
        globals: { __VROOLI_PROXY_INFO__: proxyInfo },
      }),
    })

    // Proxy metadata returns an origin-relative path so the request stays on the host origin
    expect(result).toBe('/apps/test/proxy')
  })
})
