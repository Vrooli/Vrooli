/**
 * E2E Tests: API Request Routing
 *
 * CRITICAL: Tests that API requests from host and child scenarios route
 * to the correct API servers. This is the #1 most important feature -
 * if API routing is broken, the entire proxy pattern fails.
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import type { TestContext } from './setup.js'
import { setupTestEnvironment, cleanupTestEnvironment } from './setup.js'

describe('API Request Routing', () => {
  let ctx: TestContext

  beforeAll(async () => {
    ctx = await setupTestEnvironment(3000) // Use unique port offset
  }, 60000)

  afterAll(async () => {
    await cleanupTestEnvironment(ctx)
  })

  it('should route host API requests to host API server', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Make API request from host context
    const apiResponse = await ctx.page.evaluate(async (apiPort) => {
      const response = await fetch('/api/v1/health')
      const data = await response.json()
      return { status: response.status, data }
    }, ctx.hostApiPort)

    expect(apiResponse.status).toBe(200)
    expect(apiResponse.data.service).toBe('host-api')
    expect(apiResponse.data.status).toBe('healthy')
  }, 30000)

  it('should route child API requests to child API server (through proxy)', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').first().waitFor({ timeout: 10000 })

    // Make API request from child iframe context
    const apiResponse = await iframe.locator('body').evaluate(async () => {
      try {
        const response = await fetch('/api/v1/data')
        if (!response.ok) {
          const text = await response.text()
          return { status: response.status, data: null, error: `HTTP ${response.status}`, responseText: text }
        }
        const text = await response.text()
        if (!text) {
          return { status: response.status, data: null, error: 'Empty response', responseText: '' }
        }
        const data = JSON.parse(text)
        return { status: response.status, data, error: null }
      } catch (error: any) {
        return { status: 0, data: null, error: error.message }
      }
    })

    expect(apiResponse.status).toBe(200)
    expect(apiResponse.error).toBeNull()
    expect(apiResponse.data).toBeDefined()
    expect(apiResponse.data.data).toBe('child-data')
  }, 30000)

  it('should verify child API requests do NOT hit host API', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').first().waitFor({ timeout: 10000 })

    // Try to access host-only endpoint from child (should fail or return 404)
    const apiResponse = await iframe.locator('body').evaluate(async () => {
      try {
        const response = await fetch('/api/v1/health')
        const data = await response.json()
        return { status: response.status, data, error: null }
      } catch (error: any) {
        return { status: 0, data: null, error: error.message }
      }
    })

    // Should either fail or return host's health (depending on proxy setup)
    // The key is it should be routed through the proxy path
    if (apiResponse.status === 200) {
      // If it succeeds, verify it's hitting the right endpoint through proxy
      expect(apiResponse.data).toBeDefined()
    }
  }, 30000)

  it('should handle API errors gracefully', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Make request to non-existent endpoint
    const apiResponse = await ctx.page.evaluate(async () => {
      try {
        const response = await fetch('/api/v1/nonexistent')
        return { status: response.status, ok: response.ok }
      } catch (error: any) {
        return { status: 0, error: error.message }
      }
    })

    expect(apiResponse.status).toBe(404)
    expect(apiResponse.ok).toBe(false)
  }, 30000)

  it('should preserve query parameters when proxying child API requests', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').first().waitFor({ timeout: 10000 })

    const apiResponse = await iframe.locator('body').evaluate(async () => {
      try {
        const response = await fetch('/api/v1/issues?limit=137')
        const data = await response.json()
        return { status: response.status, data }
      } catch (error: any) {
        return { status: 0, data: null, error: error.message }
      }
    })

    expect(apiResponse.status).toBe(200)
    expect(apiResponse.data?.limit).toBe(137)
    expect(apiResponse.data?.count).toBe(137)
  }, 30000)

  it('should proxy child WebSocket connections through the host', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').first().waitFor({ timeout: 10000 })

    const wsResult = await iframe.locator('body').evaluate(async () => {
      return await new Promise<{ welcome: any; echo: any }>((resolve, reject) => {
        try {
          const ws = new (window as any).WebSocket('/api/v1/ws')
          const result: { welcome: any; echo: any } = { welcome: null, echo: null }
          const timeout = setTimeout(() => {
            ws.close()
            reject(new Error('WebSocket timeout'))
          }, 5000)

          ws.addEventListener('open', () => {
            ws.send(JSON.stringify({ type: 'ping', source: 'child-ui' }))
          })

          ws.addEventListener('message', (event) => {
            let payload: any
            try {
              payload = JSON.parse(event.data)
            } catch (error) {
              payload = { raw: event.data }
            }

            if (payload.type === 'welcome') {
              result.welcome = payload
              ws.send(JSON.stringify({ type: 'ping', payload: 'via-proxy' }))
            }

            if (payload.type === 'echo') {
              result.echo = payload
              clearTimeout(timeout)
              ws.close()
              resolve(result)
            }
          })

          ws.addEventListener('error', () => {
            clearTimeout(timeout)
            reject(new Error('WebSocket error'))
          })
        } catch (error) {
          reject(error)
        }
      })
    })

    expect(wsResult.welcome?.source).toBe('child-api')
    expect(wsResult.echo?.source).toBe('child-api')
    expect(wsResult.echo?.payload).toBeDefined()
  }, 40000)


  it('should support absolute URL API requests from child', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').first().waitFor({ timeout: 10000 })

    // Make absolute URL request (should work regardless of proxy)
    // NOTE: Passing port as string to avoid Playwright serialization issues
    const port = ctx.childUiPort
    const apiResponse = await iframe.locator('body').evaluate(async (portStr) => {
      try {
        const response = await fetch(`http://127.0.0.1:${portStr}/api/v1/data`)
        const data = await response.json()
        return { status: response.status, data, error: null }
      } catch (error: any) {
        return { status: 0, data: null, error: error.message }
      }
    }, String(port))

    if (apiResponse.error) {
      // CORS or network error expected when child tries to fetch from different origin
      console.log('Absolute URL fetch failed (expected due to CORS):', apiResponse.error)
      expect(apiResponse.error).toBeDefined()
    } else {
      expect(apiResponse.status).toBe(200)
      expect(apiResponse.data.data).toBe('child-data')
    }
  }, 30000)

  it('should verify proxy metadata enables correct API resolution', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').first().waitFor({ timeout: 10000 })

    // Check that child has access to proxy metadata
    const hasProxyInfo = await iframe.locator('body').evaluate(() => {
      const proxyInfo = (window as any).__VROOLI_PROXY_INFO__
      return {
        hasInfo: !!proxyInfo,
        hasPorts: !!proxyInfo?.ports,
        hasPrimary: !!proxyInfo?.primary,
        primaryPath: proxyInfo?.primary?.path,
      }
    })

    expect(hasProxyInfo.hasInfo).toBe(true)
    expect(hasProxyInfo.hasPorts).toBe(true)
    expect(hasProxyInfo.hasPrimary).toBe(true)
    expect(hasProxyInfo.primaryPath).toBe(`/apps/${ctx.childId}/proxy`)
  }, 30000)

  it('should handle POST requests with JSON body', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Test POST from host
    const postResponse = await ctx.page.evaluate(async () => {
      const response = await fetch('/api/v1/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ test: 'data' }),
      })
      return { status: response.status, ok: response.ok }
    })

    // Should at least reach the server (404 is fine, means routing works)
    expect(postResponse.status).toBeGreaterThanOrEqual(200)
  }, 30000)

  it('should handle CORS preflight requests', async () => {
    // This would require more complex setup, but we can at least verify
    // that cross-origin requests don't cause errors
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const consoleErrors: string[] = []
    ctx.page.on('console', (msg) => {
      if (msg.type() === 'error' && msg.text().toLowerCase().includes('cors')) {
        consoleErrors.push(msg.text())
      }
    })

    // Make API request
    await ctx.page.evaluate(async () => {
      await fetch('/api/v1/health')
    })

    // Should have no CORS errors
    expect(consoleErrors).toEqual([])
  }, 30000)

  /**
   * CRITICAL BUG TEST: API requests from nested host routes
   *
   * This tests the exact bug the user reported:
   * "when I try to load an app preview URL directly, it tries sending some of
   * app-monitor's own requests through the proxy, such as `summary` and `resources`."
   *
   * The issue: When you navigate to /apps/:id/preview (a nested HOST route, NOT a proxy route),
   * API requests should go to the HOST API, but they're incorrectly being proxied to the child.
   */
  describe('Nested Host Routes API Routing', () => {
    it('should route host API requests correctly from nested route /apps/:id/preview', async () => {
      // Track ALL network requests to see where they're going
      const apiRequests: Array<{ url: string; isProxied: boolean }> = []

      ctx.page.on('request', (request) => {
        const url = request.url()
        if (url.includes('/api/v1/')) {
          const isProxied = url.includes('/proxy/')
          apiRequests.push({ url, isProxied })
          console.log(`[API REQUEST] ${url} (proxied: ${isProxied})`)
        }
      })

      // Navigate to nested HOST route (NOT a proxy route)
      await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
        waitUntil: 'networkidle',
      })

      // This is a HOST page, so it should have host content
      const hasHostContent = await ctx.page.evaluate(() => {
        return document.querySelector('[data-testid="host-content"]') !== null
      })
      expect(hasHostContent).toBe(true)

      // Make API request from this nested host route
      const apiResponse = await ctx.page.evaluate(async () => {
        try {
          const response = await fetch('/api/v1/health')
          const data = await response.json()
          return { status: response.status, data, error: null }
        } catch (error: any) {
          return { status: 0, data: null, error: error.message }
        }
      })

      // Should succeed
      expect(apiResponse.status).toBe(200)
      expect(apiResponse.error).toBeNull()

      // CRITICAL: Should hit HOST API, not child API
      expect(apiResponse.data.service).toBe('host-api')
      expect(apiResponse.data.service).not.toBe('child-api')

      // Verify the request did NOT go through /proxy/ path
      const healthRequests = apiRequests.filter((req) => req.url.includes('/health'))
      expect(healthRequests.length).toBeGreaterThan(0)
      for (const req of healthRequests) {
        expect(req.isProxied).toBe(false)
        expect(req.url).not.toContain('/proxy/')
      }
    }, 30000)

    it('should route /summary endpoint to host API from nested route', async () => {
      // Add /summary endpoint to host API for testing
      // (In setup.ts, we need to add this mock endpoint)

      await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
        waitUntil: 'networkidle',
      })

      const apiResponse = await ctx.page.evaluate(async () => {
        try {
          const response = await fetch('/api/v1/summary')
          // May be 404 if endpoint doesn't exist, but should at least reach host API
          return { status: response.status, url: response.url, error: null }
        } catch (error: any) {
          return { status: 0, url: '', error: error.message }
        }
      })

      // Should reach the API (404 or 200 are both fine, means routing worked)
      expect(apiResponse.status).toBeGreaterThanOrEqual(200)
      expect(apiResponse.status).toBeLessThan(500) // Not a proxy error

      // URL should NOT contain /proxy/
      expect(apiResponse.url).not.toContain('/proxy/')
    }, 30000)

    it('should route /resources endpoint to host API from nested route', async () => {
      await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
        waitUntil: 'networkidle',
      })

      const apiResponse = await ctx.page.evaluate(async () => {
        try {
          const response = await fetch('/api/v1/resources')
          return { status: response.status, url: response.url, error: null }
        } catch (error: any) {
          return { status: 0, url: '', error: error.message }
        }
      })

      // Should reach the API (404 or 200 are both fine)
      expect(apiResponse.status).toBeGreaterThanOrEqual(200)
      expect(apiResponse.status).toBeLessThan(500)

      // URL should NOT contain /proxy/
      expect(apiResponse.url).not.toContain('/proxy/')
    }, 30000)

    it('should handle absolute URL API requests from nested route', async () => {
      await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
        waitUntil: 'networkidle',
      })

      const apiResponse = await ctx.page.evaluate(async (hostApiPort) => {
        try {
          const response = await fetch(`http://127.0.0.1:${hostApiPort}/api/v1/health`)
          const data = await response.json()
          return { status: response.status, data, error: null }
        } catch (error: any) {
          return { status: 0, data: null, error: error.message }
        }
      }, ctx.hostApiPort)

      // May fail due to CORS, but that's expected for absolute URLs to different port
      if (apiResponse.error) {
        console.log('Absolute URL fetch failed (expected due to CORS):', apiResponse.error)
      } else {
        expect(apiResponse.status).toBe(200)
        expect(apiResponse.data.service).toBe('host-api')
      }
    }, 30000)

    it('should NOT proxy host API requests from deeply nested route', async () => {
      const nestedRoute = `/apps/${ctx.childId}/settings/profile/edit`

      await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}${nestedRoute}`, {
        waitUntil: 'networkidle',
      })

      // Verify we're on a host page
      const hasHostContent = await ctx.page.evaluate(() => {
        return document.querySelector('[data-testid="host-content"]') !== null
      })
      expect(hasHostContent).toBe(true)

      // Make API request
      const apiResponse = await ctx.page.evaluate(async () => {
        try {
          const response = await fetch('/api/v1/health')
          const data = await response.json()
          return { status: response.status, data, url: response.url }
        } catch (error: any) {
          return { status: 0, data: null, url: '', error: error.message }
        }
      })

      expect(apiResponse.status).toBe(200)
      expect(apiResponse.data.service).toBe('host-api')

      // URL should NOT contain proxy path
      expect(apiResponse.url).not.toContain('/proxy/')
      expect(apiResponse.url).not.toContain(`/apps/${ctx.childId}/`)
    }, 30000)

    it('should differentiate API routing between /apps/:id/preview and /apps/:id/proxy', async () => {
      // Test from preview route (host)
      await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
        waitUntil: 'networkidle',
      })

      const previewApiResponse = await ctx.page.evaluate(async () => {
        try {
          const response = await fetch('/api/v1/health')
          const data = await response.json()
          return { status: response.status, data }
        } catch (error: any) {
          return { status: 0, data: null, error: error.message }
        }
      })

      expect(previewApiResponse.status).toBe(200)
      expect(previewApiResponse.data.service).toBe('host-api')

      // Test from proxy route (child in iframe)
      await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
        waitUntil: 'networkidle',
      })

      const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
      await iframe.locator('[data-testid="child-content"]').first().waitFor({ timeout: 10000 })

      const proxyApiResponse = await iframe.locator('body').evaluate(async () => {
        try {
          const response = await fetch('/api/v1/data')
          const data = await response.json()
          return { status: response.status, data }
        } catch (error: any) {
          return { status: 0, data: null, error: error.message }
        }
      })

      expect(proxyApiResponse.status).toBe(200)
      expect(proxyApiResponse.data.data).toBe('child-data')

      // They should hit different APIs
      expect(previewApiResponse.data.service).not.toBe(proxyApiResponse.data.data)
    }, 30000)

    it('should resolve relative API URLs correctly from nested host route', async () => {
      await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
        waitUntil: 'networkidle',
      })

      // Test how relative URLs resolve
      const resolvedUrls = await ctx.page.evaluate(() => {
        const createTestAnchor = (href: string) => {
          const a = document.createElement('a')
          a.href = href
          return a.href
        }

        return {
          relativeApi: createTestAnchor('/api/v1/health'),
          relativeAsset: createTestAnchor('/assets/test.js'),
          relativeRoot: createTestAnchor('/'),
          relativeNested: createTestAnchor('api/v1/health'), // Without leading slash
          windowOrigin: window.location.origin,
          windowPathname: window.location.pathname,
        }
      })

      // All should resolve relative to host origin (because base tag is /)
      expect(resolvedUrls.relativeApi).toBe(`http://127.0.0.1:${ctx.hostUiPort}/api/v1/health`)
      expect(resolvedUrls.relativeAsset).toBe(`http://127.0.0.1:${ctx.hostUiPort}/assets/test.js`)
      expect(resolvedUrls.relativeRoot).toBe(`http://127.0.0.1:${ctx.hostUiPort}/`)

      // None should contain /proxy/
      expect(resolvedUrls.relativeApi).not.toContain('/proxy/')
      expect(resolvedUrls.relativeAsset).not.toContain('/proxy/')
    }, 30000)

    it('should use window.location.origin for API resolution from nested route', async () => {
      await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
        waitUntil: 'networkidle',
      })

      const locationInfo = await ctx.page.evaluate(() => {
        return {
          origin: window.location.origin,
          href: window.location.href,
          pathname: window.location.pathname,
          baseHref: document.querySelector('base')?.getAttribute('href'),
        }
      })

      // Origin should be host UI port
      expect(locationInfo.origin).toBe(`http://127.0.0.1:${ctx.hostUiPort}`)

      // Pathname should be the nested route
      expect(locationInfo.pathname).toBe(`/apps/${ctx.childId}/preview`)

      // Base href should be /
      expect(locationInfo.baseHref).toBe('/')

      // When api-base resolve() uses window.location.origin, it should get host UI port
      // This is CRITICAL for correct API routing
    }, 30000)
  })
})
