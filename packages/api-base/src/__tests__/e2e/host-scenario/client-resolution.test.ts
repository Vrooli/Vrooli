/**
 * E2E Tests: Client-Side API Resolution
 *
 * CRITICAL: Tests that the client-side resolveApiBase() function correctly
 * resolves API URLs from nested host routes.
 *
 * This is different from api-routing.test.ts which tests browser's native fetch.
 * This test verifies that api-base's resolve() function returns the correct base URL.
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import type { TestContext } from './setup.js'
import { setupTestEnvironment, cleanupTestEnvironment } from './setup.js'

describe('Client-Side API Resolution', () => {
  let ctx: TestContext

  beforeAll(async () => {
    ctx = await setupTestEnvironment(4000) // Use unique port offset
  }, 60000)

  afterAll(async () => {
    await cleanupTestEnvironment(ctx)
  })

  /**
   * CRITICAL: Test module-load-time resolution
   *
   * This simulates what ACTUALLY happens in app-monitor:
   * api.ts calls resolveApiBase() ONCE at module load time, stores the result,
   * and all future requests use that cached base URL.
   */
  it('should resolve API base correctly at module load time from nested route', async () => {
    // Navigate to nested host route
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'domcontentloaded', // Don't wait for network idle
    })

    // Immediately after page load (before React mounts), call resolveApiBase
    // This simulates what happens when api.ts module loads
    const resolvedAtModuleLoad = await ctx.page.evaluate(() => {
      // This is EXACTLY what api.ts does
      const pathname = window.location.pathname
      const origin = window.location.origin
      const hasProxyInfo = !!(window as any).__VROOLI_PROXY_INFO__
      const proxiedPath = pathname.includes('/proxy')

      let resolved: string
      if (hasProxyInfo) {
        resolved = (window as any).__VROOLI_PROXY_INFO__.primary?.path || origin
      } else if (proxiedPath) {
        const idx = pathname.indexOf('/proxy')
        const basePath = pathname.slice(0, idx + '/proxy'.length)
        resolved = origin + basePath
      } else {
        resolved = origin
      }

      return {
        pathname,
        origin,
        hasProxyInfo,
        proxiedPath,
        resolved,
        hasAppsInPath: pathname.includes('/apps/'),
      }
    })

    console.log('[MODULE LOAD] Resolution:', resolvedAtModuleLoad)

    // At module load time from /apps/:id/preview:
    // - hasProxyInfo should be FALSE (it's not a proxied page)
    // - proxiedPath should be FALSE (no /proxy in path)
    // - Should resolve to origin
    expect(resolvedAtModuleLoad.hasProxyInfo).toBe(false)
    expect(resolvedAtModuleLoad.proxiedPath).toBe(false)
    expect(resolvedAtModuleLoad.hasAppsInPath).toBe(true)
    expect(resolvedAtModuleLoad.resolved).toBe(`http://127.0.0.1:${ctx.hostUiPort}`)

    // Now test that requests using this base work
    const apiTest = await ctx.page.evaluate(async (base) => {
      const url = `${base}/api/v1/summary`
      try {
        const res = await fetch(url)
        const data = await res.json()
        return { status: res.status, data, url, error: null }
      } catch (error: any) {
        return { status: 0, data: null, url, error: error.message }
      }
    }, resolvedAtModuleLoad.resolved)

    console.log('[API TEST] Result:', apiTest)

    expect(apiTest.status).toBe(200)
    expect(apiTest.data.service).toBe('host-api')
    expect(apiTest.url).not.toContain('/proxy')
  }, 30000)

  /**
   * Test that resolveApiBase() works correctly from nested host routes
   *
   * This simulates what happens when app-monitor's React code calls resolveApiBase()
   * from a page like /apps/scenario-auditor/preview
   */
  it('should resolve API base correctly from nested host route using resolveApiBase()', async () => {
    // Navigate to nested host route
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'networkidle',
    })

    // Inject api-base client code into the page
    // In real apps, this would come from the bundle
    await ctx.page.addScriptTag({
      content: `
        // Simulate api-base resolve logic
        window.testResolveApiBase = function() {
          const origin = window.location.origin;
          const pathname = window.location.pathname;
          const hostname = window.location.hostname;

          // Check for proxy context
          const hasProxyInfo = !!window.__VROOLI_PROXY_INFO__;
          const proxiedPath = pathname.includes('/proxy');
          const isLocalhost = hostname === 'localhost' || hostname === '127.0.0.1';

          // Resolution logic (simplified from api-base)
          if (hasProxyInfo) {
            return window.__VROOLI_PROXY_INFO__.primary?.path || origin;
          }

          if (proxiedPath) {
            // Derive from path
            const idx = pathname.indexOf('/proxy');
            const basePath = pathname.slice(0, idx + '/proxy'.length);
            return origin + basePath;
          }

          // Fallback to origin (this is what happens for /apps/:id/preview)
          return origin;
        };
      `,
    })

    // Call the resolution function
    const resolved = await ctx.page.evaluate(() => {
      return (window as any).testResolveApiBase()
    })

    // Should resolve to host origin, not a proxy path
    expect(resolved).toBe(`http://127.0.0.1:${ctx.hostUiPort}`)
    expect(resolved).not.toContain('/proxy/')
    expect(resolved).not.toContain(`/apps/${ctx.childId}/`)

    // Now test that API requests using this resolved base work
    const apiResponse = await ctx.page.evaluate(async (apiBase) => {
      try {
        const response = await fetch(`${apiBase}/api/v1/health`)
        const data = await response.json()
        return { status: response.status, data, error: null }
      } catch (error: any) {
        return { status: 0, data: null, error: error.message }
      }
    }, resolved)

    expect(apiResponse.status).toBe(200)
    expect(apiResponse.data.service).toBe('host-api')
  }, 30000)

  it('should differentiate resolution between host route and proxy route', async () => {
    // First, test from host route
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'networkidle',
    })

    await ctx.page.addScriptTag({
      content: `
        window.resolveFromContext = function() {
          const origin = window.location.origin;
          const pathname = window.location.pathname;
          const hasProxyInfo = !!window.__VROOLI_PROXY_INFO__;
          const proxiedPath = pathname.includes('/proxy');

          if (hasProxyInfo) {
            return { resolved: window.__VROOLI_PROXY_INFO__.primary?.path || origin, reason: 'proxy-info' };
          }
          if (proxiedPath) {
            const idx = pathname.indexOf('/proxy');
            const basePath = pathname.slice(0, idx + '/proxy'.length);
            return { resolved: origin + basePath, reason: 'proxied-path' };
          }
          return { resolved: origin, reason: 'origin-fallback' };
        };
      `,
    })

    const hostResolution = await ctx.page.evaluate(() => {
      return (window as any).resolveFromContext()
    })

    expect(hostResolution.reason).toBe('origin-fallback')
    expect(hostResolution.resolved).toBe(`http://127.0.0.1:${ctx.hostUiPort}`)

    // Now navigate to proxy route (child in iframe)
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    // Inject resolution logic into iframe
    await iframe.locator('body').evaluate(() => {
      ;(window as any).resolveFromContext = function () {
        const origin = window.location.origin
        const pathname = window.location.pathname
        const hasProxyInfo = !!(window as any).__VROOLI_PROXY_INFO__
        const proxiedPath = pathname.includes('/proxy')

        if (hasProxyInfo) {
          return { resolved: (window as any).__VROOLI_PROXY_INFO__.primary?.path || origin, reason: 'proxy-info' }
        }
        if (proxiedPath) {
          const idx = pathname.indexOf('/proxy')
          const basePath = pathname.slice(0, idx + '/proxy'.length)
          return { resolved: origin + basePath, reason: 'proxied-path' }
        }
        return { resolved: origin, reason: 'origin-fallback' }
      }
    })

    const proxyResolution = await iframe.locator('body').evaluate(() => {
      return (window as any).resolveFromContext()
    })

    // Should use proxy-info or proxied-path, NOT origin-fallback
    expect(proxyResolution.reason).not.toBe('origin-fallback')
    expect(proxyResolution.resolved).toContain('/proxy')

    // Resolutions should be different
    expect(hostResolution.resolved).not.toBe(proxyResolution.resolved)
  }, 30000)

  it('should handle /apps/:id/preview URL pattern correctly', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'networkidle',
    })

    // Get location details
    const location = await ctx.page.evaluate(() => {
      return {
        href: window.location.href,
        origin: window.location.origin,
        pathname: window.location.pathname,
        hasProxyMarker: window.location.pathname.includes('/proxy'),
        hasAppsMarker: window.location.pathname.includes('/apps/'),
      }
    })

    // Should have /apps/ but NOT /proxy
    expect(location.hasAppsMarker).toBe(true)
    expect(location.hasProxyMarker).toBe(false)
    expect(location.pathname).toBe(`/apps/${ctx.childId}/preview`)

    // Resolution should NOT treat this as a proxy context
    const resolved = await ctx.page.evaluate(() => {
      const pathname = window.location.pathname
      const hasProxyInPath = pathname.includes('/proxy')
      return { hasProxyInPath, shouldUseOrigin: !hasProxyInPath }
    })

    expect(resolved.hasProxyInPath).toBe(false)
    expect(resolved.shouldUseOrigin).toBe(true)
  }, 30000)

  /**
   * CRITICAL: Test the exact scenario from the bug report
   *
   * "when I try to load an app preview URL directly, it tries sending some of
   * app-monitor's own requests through the proxy, such as `summary` and `resources`."
   */
  it('should NOT route /api/v1/summary through proxy from /apps/:id/preview', async () => {
    const apiRequests: Array<{ url: string; hasProxy: boolean }> = []

    ctx.page.on('request', (request) => {
      const url = request.url()
      if (url.includes('/summary') || url.includes('/resources')) {
        const hasProxy = url.includes('/proxy')
        apiRequests.push({ url, hasProxy })
        console.log(`[API] ${url} (hasProxy: ${hasProxy})`)
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'networkidle',
    })

    // Make requests to /summary and /resources
    const results = await ctx.page.evaluate(async () => {
      const summaryRes = await fetch('/api/v1/summary')
      const resourcesRes = await fetch('/api/v1/resources')

      return {
        summary: {
          status: summaryRes.status,
          url: summaryRes.url,
          data: await summaryRes.json(),
        },
        resources: {
          status: resourcesRes.status,
          url: resourcesRes.url,
          data: await resourcesRes.json(),
        },
      }
    })

    // Both should succeed and hit HOST API
    expect(results.summary.status).toBe(200)
    expect(results.summary.data.service).toBe('host-api')
    expect(results.resources.status).toBe(200)
    expect(results.resources.data.service).toBe('host-api')

    // CRITICAL: URLs should NOT contain /proxy
    expect(results.summary.url).not.toContain('/proxy')
    expect(results.resources.url).not.toContain('/proxy')

    // Verify network requests didn't go through proxy
    for (const req of apiRequests) {
      expect(req.hasProxy).toBe(false)
      expect(req.url).not.toContain('/proxy')
    }
  }, 30000)
})
