/**
 * E2E Tests: Nested Host Routes
 *
 * CRITICAL TEST: This tests the exact scenario that breaks in app-monitor.
 *
 * When accessing a host's own nested UI route (like /apps/scenario-auditor/preview),
 * the host's assets must load correctly from the root, NOT be proxied to a child.
 *
 * This is the bug the user reported: "when I try to load an app preview URL directly,
 * it tries sending all requests through the proxy - even app-monitor's own requests."
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import type { TestContext } from './setup.js'
import { setupTestEnvironment, cleanupTestEnvironment, makeRequest } from './setup.js'

describe('Host Scenario: Nested Host Routes', () => {
  let ctx: TestContext

  beforeAll(async () => {
    ctx = await setupTestEnvironment(2000) // Use port offset to avoid conflicts
  }, 60000)

  afterAll(async () => {
    await cleanupTestEnvironment(ctx)
  })

  /**
   * CRITICAL TEST: This is the exact scenario that fails in production
   *
   * When you directly navigate to /apps/scenario-auditor/preview (a host UI route),
   * the browser should:
   * 1. Receive the host's index.html
   * 2. Have <base href="/"> injected
   * 3. Load all assets from the host server root
   * 4. NOT try to proxy assets to the child scenario
   */
  it('should serve host assets correctly from nested route /apps/:id/preview', async () => {
    const nestedRoute = `/apps/${ctx.childId}/preview`
    const failedRequests: Array<{ url: string; status: number }> = []
    const assetRequests: Array<{ url: string; fromHost: boolean }> = []

    // Monitor all network requests
    ctx.page.on('response', (response) => {
      const url = response.url()

      // Track failed requests
      if (response.status() >= 400) {
        failedRequests.push({
          url,
          status: response.status(),
        })
      }

      // Track where assets are being loaded from
      if (url.includes('host-styles.css') || url.includes('host-main.js')) {
        const fromHost = !url.includes('/proxy/')
        assetRequests.push({ url, fromHost })
      }
    })

    // Navigate to nested host route (NOT a proxy route)
    const response = await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}${nestedRoute}`, {
      waitUntil: 'networkidle',
      timeout: 30000,
    })

    // 1. Should return 200
    expect(response?.status()).toBe(200)

    // 2. Should serve host HTML (SPA fallback)
    const content = await response?.text()
    expect(content).toContain('Host Scenario')
    expect(content).toContain('data-testid="host-content"')

    // 3. Should have base tag injected
    const baseHref = await ctx.page.evaluate(() => {
      const base = document.querySelector('base')
      return base?.getAttribute('href') || null
    })
    expect(baseHref).toBe('/')

    // 4. Host content should be visible
    const hostContent = ctx.page.locator('[data-testid="host-content"]')
    await hostContent.waitFor({ timeout: 10000 })
    const isVisible = await hostContent.isVisible()
    expect(isVisible).toBe(true)

    // 5. CRITICAL: All host assets should load from root, not through proxy
    expect(assetRequests.length).toBeGreaterThan(0)
    for (const request of assetRequests) {
      expect(request.fromHost).toBe(true)
      expect(request.url).not.toContain('/proxy/')
    }

    // 6. Should have NO failed asset requests
    const assetFailures = failedRequests.filter(
      (req) => req.url.includes('.js') || req.url.includes('.css')
    )
    if (assetFailures.length > 0) {
      console.log('\n=== Failed Asset Requests ===')
      assetFailures.forEach((req) => {
        console.log(`  ${req.url} => ${req.status}`)
      })
    }
    expect(assetFailures).toEqual([])

    // 7. Host scripts should execute correctly
    const hostLoaded = await ctx.page.evaluate(() => {
      return (window as any).hostLoaded === true
    })
    expect(hostLoaded).toBe(true)
  }, 60000)

  it('should serve host assets correctly from deeply nested route /apps/:id/settings/profile', async () => {
    const nestedRoute = `/apps/${ctx.childId}/settings/profile`
    const failedRequests: Array<{ url: string; status: number }> = []

    ctx.page.on('response', (response) => {
      if (response.status() >= 400) {
        failedRequests.push({
          url: response.url(),
          status: response.status(),
        })
      }
    })

    const response = await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}${nestedRoute}`, {
      waitUntil: 'networkidle',
      timeout: 30000,
    })

    expect(response?.status()).toBe(200)

    // Should have base tag
    const baseHref = await ctx.page.evaluate(() => {
      return document.querySelector('base')?.getAttribute('href') || null
    })
    expect(baseHref).toBe('/')

    // Host content should be visible
    const hostContent = ctx.page.locator('[data-testid="host-content"]')
    await hostContent.waitFor({ timeout: 10000 })
    expect(await hostContent.isVisible()).toBe(true)

    // No failed asset requests
    const assetFailures = failedRequests.filter(
      (req) => req.url.includes('.js') || req.url.includes('.css')
    )
    expect(assetFailures).toEqual([])
  }, 60000)

  it('should NOT inject base tag for proxy routes', async () => {
    const proxyRoute = `/apps/${ctx.childId}/proxy/index.html`

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}${proxyRoute}`, {
      waitUntil: 'networkidle',
    })

    // Should have base tag for child (different from host)
    const baseHref = await ctx.page.evaluate(() => {
      return document.querySelector('base')?.getAttribute('href') || null
    })
    expect(baseHref).toBe(`/apps/${ctx.childId}/proxy/`)

    // Should NOT have host's base tag
    const hostBaseTag = await ctx.page.evaluate(() => {
      return document.querySelector('base[data-host-self]') !== null
    })
    expect(hostBaseTag).toBe(false)

    // Should have child's base tag
    const childBaseTag = await ctx.page.evaluate(() => {
      return document.querySelector('base[data-host-proxy]') !== null
    })
    expect(childBaseTag).toBe(true)
  }, 30000)

  it('should handle multiple nested segments correctly /apps/:id/deep/nested/route', async () => {
    const nestedRoute = `/apps/${ctx.childId}/dashboard/analytics/reports/weekly`

    const response = await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}${nestedRoute}`, {
      waitUntil: 'networkidle',
    })

    expect(response?.status()).toBe(200)

    // Base tag should still be /
    const baseHref = await ctx.page.evaluate(() => {
      return document.querySelector('base')?.getAttribute('href') || null
    })
    expect(baseHref).toBe('/')

    // Assets should resolve correctly
    const resolvedAssetUrl = await ctx.page.evaluate(() => {
      const a = document.createElement('a')
      a.href = 'assets/test.js'
      return a.href
    })
    expect(resolvedAssetUrl).toBe(`http://127.0.0.1:${ctx.hostUiPort}/assets/test.js`)
    expect(resolvedAssetUrl).not.toContain('/proxy/')
  }, 30000)

  it('should differentiate between /apps/:id/preview and /apps/:id/proxy', async () => {
    // Test preview route (host's own route)
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'networkidle',
    })

    const previewBase = await ctx.page.evaluate(() => {
      return document.querySelector('base')?.getAttribute('href')
    })
    expect(previewBase).toBe('/')

    const previewContent = await ctx.page.evaluate(() => {
      return document.querySelector('[data-testid="host-content"]') !== null
    })
    expect(previewContent).toBe(true)

    // Test proxy route (child scenario)
    await ctx.page.goto(
      `http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/proxy/index.html`,
      {
        waitUntil: 'networkidle',
      }
    )

    const proxyBase = await ctx.page.evaluate(() => {
      return document.querySelector('base')?.getAttribute('href')
    })
    expect(proxyBase).toBe(`/apps/${ctx.childId}/proxy/`)

    const proxyContent = await ctx.page.evaluate(() => {
      return document.querySelector('[data-testid="child-content"]') !== null
    })
    expect(proxyContent).toBe(true)

    // They should be completely different
    expect(previewBase).not.toBe(proxyBase)
  }, 60000)

  it('should handle direct asset requests from nested routes', async () => {
    // First, navigate to nested route to set up the context
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'networkidle',
    })

    // Now manually request an asset (simulating what browser would do with relative URL)
    const assetResponse = await makeRequest(ctx.hostUiPort, '/host-styles.css')

    expect(assetResponse.status).toBe(200)
    expect(assetResponse.headers['content-type']).toContain('text/css')
    expect(assetResponse.body).toContain('background: red')
  }, 30000)

  it('should not confuse /apps/:id/public with /apps/:id/proxy', async () => {
    const publicRoute = `/apps/${ctx.childId}/public/dashboard`

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}${publicRoute}`, {
      waitUntil: 'networkidle',
    })

    // Should serve host content with base /
    const baseHref = await ctx.page.evaluate(() => {
      return document.querySelector('base')?.getAttribute('href')
    })
    expect(baseHref).toBe('/')

    // Should have host content
    const hasHostContent = await ctx.page.evaluate(() => {
      return document.querySelector('[data-testid="host-content"]') !== null
    })
    expect(hasHostContent).toBe(true)

    // Should NOT have child content
    const hasChildContent = await ctx.page.evaluate(() => {
      return document.querySelector('[data-testid="child-content"]') !== null
    })
    expect(hasChildContent).toBe(false)
  }, 30000)

  it('should verify no console errors on nested route access', async () => {
    const consoleErrors: string[] = []

    ctx.page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text())
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'networkidle',
    })

    if (consoleErrors.length > 0) {
      console.log('\n=== Console Errors ===')
      consoleErrors.forEach((err) => {
        console.log(`  ${err}`)
      })
    }

    // Filter out expected errors (if any)
    const unexpectedErrors = consoleErrors.filter((err) => {
      // Add any expected/acceptable errors here
      return true
    })

    expect(unexpectedErrors).toEqual([])
  }, 30000)

  it('should handle query parameters on nested routes', async () => {
    const routeWithQuery = `/apps/${ctx.childId}/preview?tab=details&view=grid`

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}${routeWithQuery}`, {
      waitUntil: 'networkidle',
    })

    // Should still have correct base tag
    const baseHref = await ctx.page.evaluate(() => {
      return document.querySelector('base')?.getAttribute('href')
    })
    expect(baseHref).toBe('/')

    // Should preserve query parameters in location
    const currentUrl = await ctx.page.evaluate(() => {
      return window.location.href
    })
    expect(currentUrl).toContain('tab=details')
    expect(currentUrl).toContain('view=grid')
  }, 30000)

  it('should handle hash fragments on nested routes', async () => {
    const routeWithHash = `/apps/${ctx.childId}/preview#section-details`

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}${routeWithHash}`, {
      waitUntil: 'networkidle',
    })

    // Should still have correct base tag
    const baseHref = await ctx.page.evaluate(() => {
      return document.querySelector('base')?.getAttribute('href')
    })
    expect(baseHref).toBe('/')

    // Should preserve hash in location
    const currentHash = await ctx.page.evaluate(() => {
      return window.location.hash
    })
    expect(currentHash).toBe('#section-details')
  }, 30000)

  /**
   * Performance test: Nested routes should be as fast as root routes
   */
  it('should load nested routes with similar performance to root routes', async () => {
    // Measure root route load time
    const rootStartTime = Date.now()
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })
    const rootLoadTime = Date.now() - rootStartTime

    // Measure nested route load time
    const nestedStartTime = Date.now()
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'networkidle',
    })
    const nestedLoadTime = Date.now() - nestedStartTime

    // Nested should not be significantly slower (allow 50% overhead for middleware)
    expect(nestedLoadTime).toBeLessThan(rootLoadTime * 1.5)
  }, 60000)
})
