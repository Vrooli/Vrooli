/**
 * E2E Tests: Child Scenario Through Proxy
 *
 * Tests that child scenarios (embedded/proxied scenarios) load correctly
 * through the host's proxy routes. Verifies iframe content, assets, and metadata.
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import type { TestContext } from './setup.js'
import { setupTestEnvironment, cleanupTestEnvironment } from './setup.js'

describe('Child Scenario: Proxy Loading', () => {
  let ctx: TestContext

  beforeAll(async () => {
    ctx = await setupTestEnvironment()
  }, 60000)

  afterAll(async () => {
    await cleanupTestEnvironment(ctx)
  })

  it('should load child iframe with correct content', async () => {
    const consoleErrors: string[] = []

    ctx.page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text())
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Wait for iframe to load
    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')

    // Child content should be visible in iframe
    const childContent = iframe.locator('[data-testid="child-content"]')
    await childContent.waitFor({ state: 'visible', timeout: 10000 })

    const isVisible = await childContent.isVisible()
    expect(isVisible).toBe(true)

    const text = await childContent.textContent()
    expect(text).toContain('Child Scenario')

    // Should have no console errors
    expect(consoleErrors).toEqual([])
  }, 30000)

  it('should load child assets through proxy', async () => {
    const failedRequests: Array<{ url: string; status: number }> = []

    // Monitor failed requests in child iframe context
    ctx.page.on('response', (response) => {
      if (
        response.status() >= 400 &&
        response.url().includes(`/apps/${ctx.childId}/proxy/`)
      ) {
        failedRequests.push({
          url: response.url(),
          status: response.status(),
        })
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Wait for iframe to fully load
    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    // Should have no failed asset requests
    expect(failedRequests).toEqual([])
  }, 30000)

  it('should load child CSS through proxy', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    // Verify child CSS is loaded
    const childCssLoaded = await iframe.locator('body').evaluate(() => {
      const links = Array.from(document.querySelectorAll('link[rel="stylesheet"]'))
      return links.some((link) => (link as HTMLLinkElement).href.includes('styles.css'))
    })
    expect(childCssLoaded).toBe(true)
  }, 30000)

  it('should load child JavaScript through proxy', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    // Verify child JS is loaded
    const childJsLoaded = await iframe.locator('body').evaluate(() => {
      const scripts = Array.from(document.querySelectorAll('script[src]'))
      return scripts.some((script) => (script as HTMLScriptElement).src.includes('main.js'))
    })
    expect(childJsLoaded).toBe(true)

    // Verify script executed
    const scriptExecuted = await iframe.locator('body').evaluate(() => {
      return (window as any).childLoaded === true
    })
    expect(scriptExecuted).toBe(true)
  }, 30000)

  it('should have proxy metadata injected in child', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    // Check if child has proxy metadata
    const hasProxyInfo = await iframe.locator('body').evaluate(() => {
      return typeof (window as any).__VROOLI_PROXY_INFO__ !== 'undefined'
    })
    expect(hasProxyInfo).toBe(true)

    // Check proxy metadata structure
    const proxyInfo = await iframe.locator('body').evaluate(() => {
      return (window as any).__VROOLI_PROXY_INFO__
    })

    expect(proxyInfo).toHaveProperty('appId')
    expect(proxyInfo.appId).toBe(ctx.childId)
    expect(proxyInfo).toHaveProperty('hostScenario')
    expect(proxyInfo.hostScenario).toBe('host-scenario')
    expect(proxyInfo).toHaveProperty('targetScenario')
    expect(proxyInfo.targetScenario).toBe(ctx.childId)
  }, 30000)

  it('should load child directly (not through proxy) correctly', async () => {
    // Test that child can also be accessed directly without the proxy
    const response = await ctx.page.goto(`http://127.0.0.1:${ctx.childUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    expect(response?.status()).toBe(200)

    // Child content should be visible
    const childContent = ctx.page.locator('[data-testid="child-content"]')
    const isVisible = await childContent.isVisible()
    expect(isVisible).toBe(true)

    const text = await childContent.textContent()
    expect(text).toContain('Child Scenario')
  }, 30000)

  it('should handle proxy path with trailing slash correctly', async () => {
    // Test with trailing slash
    const responseWithSlash = await ctx.page.goto(
      `http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/proxy/index.html`,
      {
        waitUntil: 'networkidle',
      }
    )

    expect(responseWithSlash?.status()).toBe(200)

    const content = await ctx.page.textContent('body')
    expect(content).toContain('Child Scenario')
  }, 30000)
})
