/**
 * E2E Tests: Host Pages and Assets
 *
 * Tests that the host scenario (e.g., app-monitor) correctly serves its own pages
 * and assets. Verifies that host's own content loads without errors.
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import type { TestContext } from './setup.js'
import { setupTestEnvironment, cleanupTestEnvironment } from './setup.js'

describe('Host Scenario: Pages and Assets', () => {
  let ctx: TestContext

  beforeAll(async () => {
    ctx = await setupTestEnvironment()
  }, 60000)

  afterAll(async () => {
    await cleanupTestEnvironment(ctx)
  })

  it('should load host page without errors', async () => {
    const consoleErrors: string[] = []

    // Capture console errors
    ctx.page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text())
      }
    })

    // Navigate to host page
    const response = await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Should return 200
    expect(response?.status()).toBe(200)

    // Should have no console errors
    expect(consoleErrors).toEqual([])

    // Host content should be visible
    const hostContent = ctx.page.locator('[data-testid="host-content"]')
    const isVisible = await hostContent.isVisible()
    expect(isVisible).toBe(true)

    const text = await hostContent.textContent()
    expect(text).toContain('Host Scenario')
  }, 30000)

  it('should load host CSS correctly', async () => {
    const failedRequests: Array<{ url: string; status: number }> = []

    // Monitor failed requests for host CSS
    ctx.page.on('response', (response) => {
      if (response.status() >= 400 && response.url().includes('host-styles.css')) {
        failedRequests.push({
          url: response.url(),
          status: response.status(),
        })
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Should have no failed CSS requests
    expect(failedRequests).toEqual([])

    // Verify CSS link exists in DOM
    const hostCssLoaded = await ctx.page.evaluate(() => {
      const links = Array.from(document.querySelectorAll('link[rel="stylesheet"]'))
      return links.some((link) => (link as HTMLLinkElement).href.includes('host-styles.css'))
    })
    expect(hostCssLoaded).toBe(true)
  }, 30000)

  it('should load host JavaScript correctly', async () => {
    const failedRequests: Array<{ url: string; status: number }> = []

    // Monitor failed requests for host JS
    ctx.page.on('response', (response) => {
      if (response.status() >= 400 && response.url().includes('host-main.js')) {
        failedRequests.push({
          url: response.url(),
          status: response.status(),
        })
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Should have no failed JS requests
    expect(failedRequests).toEqual([])

    // Verify JS script exists in DOM
    const hostJsLoaded = await ctx.page.evaluate(() => {
      const scripts = Array.from(document.querySelectorAll('script[src]'))
      return scripts.some((script) => (script as HTMLScriptElement).src.includes('host-main.js'))
    })
    expect(hostJsLoaded).toBe(true)

    // Verify script executed
    const scriptExecuted = await ctx.page.evaluate(() => {
      return (window as any).hostLoaded === true
    })
    expect(scriptExecuted).toBe(true)
  }, 30000)

  it('should handle missing host assets with 404', async () => {
    const response = await ctx.page.goto(
      `http://127.0.0.1:${ctx.hostUiPort}/missing-asset.js`,
      {
        waitUntil: 'networkidle',
      }
    )

    // Should return 404
    expect(response?.status()).toBe(404)

    // Should not return HTML (verify SPA fallback doesn't kick in for assets)
    const contentType = response?.headers()['content-type'] || ''
    expect(contentType).not.toContain('text/html')
  }, 30000)

  it('should not have cross-origin errors', async () => {
    const consoleMessages: Array<{ type: string; text: string }> = []

    ctx.page.on('console', (msg) => {
      consoleMessages.push({
        type: msg.type(),
        text: msg.text(),
      })
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Filter for CORS errors
    const corsErrors = consoleMessages.filter(
      (msg) =>
        msg.type === 'error' &&
        (msg.text.toLowerCase().includes('cors') ||
          msg.text.toLowerCase().includes('cross-origin'))
    )

    expect(corsErrors).toEqual([])
  }, 30000)

  it('should have correct Content-Type headers', async () => {
    // Check HTML
    const htmlResponse = await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`)
    const htmlContentType = htmlResponse?.headers()['content-type'] || ''
    expect(htmlContentType).toContain('text/html')

    // Check CSS
    const cssResponse = await ctx.page.goto(
      `http://127.0.0.1:${ctx.hostUiPort}/host-styles.css`
    )
    const cssContentType = cssResponse?.headers()['content-type'] || ''
    expect(cssContentType).toContain('text/css')

    // Check JS
    const jsResponse = await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/host-main.js`)
    const jsContentType = jsResponse?.headers()['content-type'] || ''
    expect(jsContentType).toContain('application/javascript')
  }, 30000)
})
