/**
 * E2E Tests: Asset URL Resolution from Nested Routes
 *
 * These tests specifically verify that when accessing nested host routes,
 * asset URLs are resolved correctly and DON'T go through proxy paths.
 *
 * This is critical for diagnosing the bug: "when I try to load an app preview
 * URL directly, it tries sending all requests through the proxy"
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import type { TestContext } from './setup.js'
import { setupTestEnvironment, cleanupTestEnvironment } from './setup.js'

describe('Asset URL Resolution from Nested Routes', () => {
  let ctx: TestContext

  beforeAll(async () => {
    ctx = await setupTestEnvironment(3000) // Use port offset to avoid conflicts
  }, 60000)

  afterAll(async () => {
    await cleanupTestEnvironment(ctx)
  })

  it('should resolve CSS URLs correctly from nested route', async () => {
    const hostCssRequests: string[] = []
    const iframeCssRequests: string[] = []

    // Track CSS requests from main document vs iframe
    ctx.page.on('request', (request) => {
      if (request.url().includes('.css')) {
        const frame = request.frame()
        if (frame.parentFrame() === null) {
          // Main document request
          hostCssRequests.push(request.url())
        } else {
          // Iframe request
          iframeCssRequests.push(request.url())
        }
      }
    })

    // Navigate to nested route
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'networkidle',
    })

    // Should have made CSS request from main document
    expect(hostCssRequests.length).toBeGreaterThan(0)

    // ALL HOST CSS requests should be direct to root, NOT through proxy
    for (const url of hostCssRequests) {
      expect(url).not.toContain('/proxy/')
      expect(url).toContain('host-styles.css')
    }

    // CSS should be requested from root: http://host/host-styles.css
    const expectedUrl = `http://127.0.0.1:${ctx.hostUiPort}/host-styles.css`
    expect(hostCssRequests).toContain(expectedUrl)

    // Iframe CSS requests can go through proxy (that's expected)
    // Just log them for verification
    if (iframeCssRequests.length > 0) {
      console.log('  Iframe CSS requests (these can use proxy):', iframeCssRequests)
    }
  }, 30000)

  it('should resolve JS URLs correctly from nested route', async () => {
    const hostJsRequests: string[] = []
    const iframeJsRequests: string[] = []

    // Track JS requests from main document vs iframe
    ctx.page.on('request', (request) => {
      if (request.url().includes('.js')) {
        const frame = request.frame()
        if (frame.parentFrame() === null) {
          // Main document request
          hostJsRequests.push(request.url())
        } else {
          // Iframe request
          iframeJsRequests.push(request.url())
        }
      }
    })

    // Navigate to nested route
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'networkidle',
    })

    // Should have made JS request from main document
    expect(hostJsRequests.length).toBeGreaterThan(0)

    // ALL HOST JS requests should be direct to root, NOT through proxy
    for (const url of hostJsRequests) {
      expect(url).not.toContain('/proxy/')
      expect(url).toContain('host-main.js')
    }

    // JS should be requested from root: http://host/host-main.js
    const expectedUrl = `http://127.0.0.1:${ctx.hostUiPort}/host-main.js`
    expect(hostJsRequests).toContain(expectedUrl)

    // Iframe JS requests can go through proxy (that's expected)
    if (iframeJsRequests.length > 0) {
      console.log('  Iframe JS requests (these can use proxy):', iframeJsRequests)
    }
  }, 30000)

  it('should verify actual request URLs match base tag expectations', async () => {
    const hostRequests: Array<{ url: string; resourceType: string }> = []

    ctx.page.on('request', (request) => {
      // Only track main document requests
      if (request.frame().parentFrame() === null) {
        hostRequests.push({
          url: request.url(),
          resourceType: request.resourceType(),
        })
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'networkidle',
    })

    // Get base tag value
    const baseHref = await ctx.page.evaluate(() => {
      return document.querySelector('base')?.getAttribute('href') || null
    })
    expect(baseHref).toBe('/')

    // Filter for asset requests from main document (stylesheet, script, etc.)
    const hostAssetRequests = hostRequests.filter(
      (req) =>
        req.resourceType === 'stylesheet' ||
        req.resourceType === 'script' ||
        req.resourceType === 'image' ||
        req.resourceType === 'font'
    )

    // ALL HOST asset requests should respect the base tag (go to root)
    for (const req of hostAssetRequests) {
      // Should NOT contain nested path components
      expect(req.url).not.toContain('/apps/')
      expect(req.url).not.toContain('/proxy/')
      expect(req.url).not.toContain('/preview')

      // Should be at root level
      expect(req.url).toMatch(new RegExp(`^http://127\\.0\\.0\\.1:${ctx.hostUiPort}/[^/]+`))
    }
  }, 30000)

  it('should log actual vs expected URLs for debugging', async () => {
    const hostRequests: Array<{ url: string; expected: string; matches: boolean }> = []
    const iframeRequests: string[] = []

    ctx.page.on('request', (request) => {
      const url = request.url()
      const frame = request.frame()

      if (url.includes('.css') || url.includes('.js')) {
        if (frame.parentFrame() === null) {
          // Main document - check if URLs match expected pattern
          const filename = url.split('/').pop() || ''
          const expectedUrl = `http://127.0.0.1:${ctx.hostUiPort}/${filename}`
          hostRequests.push({
            url,
            expected: expectedUrl,
            matches: url === expectedUrl,
          })
        } else {
          // Iframe - just log
          iframeRequests.push(url)
        }
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'networkidle',
    })

    // Log all requests for debugging
    console.log('\n=== Asset Request Analysis ===')
    console.log(`Navigated to: /apps/${ctx.childId}/preview`)
    console.log(`Host Port: ${ctx.hostUiPort}`)
    console.log('\nHost Asset Requests:')
    hostRequests.forEach((req) => {
      console.log(`  Actual:   ${req.url}`)
      console.log(`  Expected: ${req.expected}`)
      console.log(`  ✓ Match:  ${req.matches}\n`)
    })

    console.log('\nIframe Asset Requests (can use proxy):')
    iframeRequests.forEach((url) => {
      console.log(`  ${url}`)
    })

    // All HOST requests should match
    const mismatches = hostRequests.filter((req) => !req.matches)
    if (mismatches.length > 0) {
      console.log('❌ Mismatched URLs found in HOST requests!')
      mismatches.forEach((req) => {
        console.log(`  ${req.url} !== ${req.expected}`)
      })
    }

    expect(mismatches).toEqual([])
  }, 30000)

  it('should NOT make any requests to child proxy path from host page', async () => {
    const proxyRequests: string[] = []

    ctx.page.on('request', (request) => {
      if (request.url().includes('/proxy/')) {
        proxyRequests.push(request.url())
      }
    })

    // Navigate to host's nested route
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'networkidle',
    })

    // Should NOT make ANY requests through proxy path
    // (The iframe loads the proxy route, but host assets should not)
    const hostAssetProxyRequests = proxyRequests.filter(
      (url) => url.includes('host-styles.css') || url.includes('host-main.js')
    )

    if (hostAssetProxyRequests.length > 0) {
      console.log('\n❌ Host assets incorrectly requested through proxy:')
      hostAssetProxyRequests.forEach((url) => {
        console.log(`  ${url}`)
      })
    }

    expect(hostAssetProxyRequests).toEqual([])
  }, 30000)

  it('should differentiate between host iframe and host document requests', async () => {
    const documentRequests: string[] = []
    const iframeRequests: string[] = []

    ctx.page.on('request', (request) => {
      const url = request.url()
      const frame = request.frame()

      if (frame.parentFrame() === null) {
        // Main document request
        documentRequests.push(url)
      } else {
        // Iframe request
        iframeRequests.push(url)
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'networkidle',
    })

    // Main document should load host assets from root
    const mainHostAssets = documentRequests.filter(
      (url) => url.includes('host-styles.css') || url.includes('host-main.js')
    )
    expect(mainHostAssets.length).toBeGreaterThan(0)
    mainHostAssets.forEach((url) => {
      expect(url).not.toContain('/proxy/')
    })

    // Iframe should load child assets through proxy
    const iframeChildAssets = iframeRequests.filter(
      (url) => url.includes('styles.css') || url.includes('main.js')
    )
    if (iframeChildAssets.length > 0) {
      iframeChildAssets.forEach((url) => {
        expect(url).toContain('/proxy/')
      })
    }
  }, 30000)

  it('should resolve relative URLs programmatically using base tag', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'networkidle',
    })

    // Test that JavaScript can resolve relative URLs correctly
    const resolvedUrls = await ctx.page.evaluate(() => {
      const results: Record<string, string> = {}

      // Test various relative URL patterns
      const testUrls = [
        'assets/test.js',
        './assets/test.css',
        '../assets/test.png',
        'host-styles.css',
        '/absolute/path.js',
      ]

      testUrls.forEach((testUrl) => {
        const a = document.createElement('a')
        a.href = testUrl
        results[testUrl] = a.href
      })

      return results
    })

    console.log('\n=== Relative URL Resolution ===')
    Object.entries(resolvedUrls).forEach(([input, output]) => {
      console.log(`  "${input}" => ${output}`)
    })

    // With <base href="/">, all relative URLs should resolve to root
    expect(resolvedUrls['assets/test.js']).toBe(
      `http://127.0.0.1:${ctx.hostUiPort}/assets/test.js`
    )
    expect(resolvedUrls['host-styles.css']).toBe(
      `http://127.0.0.1:${ctx.hostUiPort}/host-styles.css`
    )

    // Absolute paths should remain absolute
    expect(resolvedUrls['/absolute/path.js']).toBe(
      `http://127.0.0.1:${ctx.hostUiPort}/absolute/path.js`
    )

    // None should contain /preview or /apps/
    Object.values(resolvedUrls).forEach((url) => {
      expect(url).not.toContain('/preview')
      expect(url).not.toContain('/apps/')
    })
  }, 30000)

  it('should handle refresh on nested route', async () => {
    // Navigate to nested route
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/preview`, {
      waitUntil: 'networkidle',
    })

    // Verify base tag is correct
    const baseHrefBefore = await ctx.page.evaluate(() => {
      return document.querySelector('base')?.getAttribute('href')
    })
    expect(baseHrefBefore).toBe('/')

    // Refresh the page
    const hostRequests: string[] = []
    ctx.page.on('request', (request) => {
      // Only track main document requests
      if (request.frame().parentFrame() === null) {
        if (request.url().includes('.css') || request.url().includes('.js')) {
          hostRequests.push(request.url())
        }
      }
    })

    await ctx.page.reload({ waitUntil: 'networkidle' })

    // Verify base tag still correct after refresh
    const baseHrefAfter = await ctx.page.evaluate(() => {
      return document.querySelector('base')?.getAttribute('href')
    })
    expect(baseHrefAfter).toBe('/')

    // All HOST asset requests after refresh should still be correct
    hostRequests.forEach((url) => {
      expect(url).not.toContain('/proxy/')
      expect(url).not.toContain('/apps/')
      expect(url).not.toContain('/preview')
    })
  }, 30000)
})
