/**
 * E2E Tests: Isolation Between Host and Child
 *
 * Tests that host and child scenarios don't interfere with each other.
 * Verifies proper isolation of assets, scripts, styles, and global state.
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import type { TestContext } from './setup.js'
import { setupTestEnvironment, cleanupTestEnvironment } from './setup.js'

describe('Isolation: Host and Child', () => {
  let ctx: TestContext

  beforeAll(async () => {
    ctx = await setupTestEnvironment()
  }, 60000)

  afterAll(async () => {
    await cleanupTestEnvironment(ctx)
  })

  it('should have separate document contexts', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    // Host should have its own title
    const hostTitle = await ctx.page.title()
    expect(hostTitle).toBe('Host Scenario')

    // Child should have its own title (in iframe context)
    const childTitle = await iframe.locator('body').evaluate(() => {
      return document.title
    })
    expect(childTitle).toBe('Child Scenario')

    // Titles should be different
    expect(hostTitle).not.toBe(childTitle)
  }, 30000)

  it('should have separate global window objects', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    // Host should have hostLoaded flag
    const hostLoaded = await ctx.page.evaluate(() => {
      return (window as any).hostLoaded === true
    })
    expect(hostLoaded).toBe(true)

    // Host should NOT have childLoaded flag
    const hostHasChildFlag = await ctx.page.evaluate(() => {
      return typeof (window as any).childLoaded !== 'undefined'
    })
    expect(hostHasChildFlag).toBe(false)

    // Child should have childLoaded flag
    const childLoaded = await iframe.locator('body').evaluate(() => {
      return (window as any).childLoaded === true
    })
    expect(childLoaded).toBe(true)

    // Child should NOT have hostLoaded flag
    const childHasHostFlag = await iframe.locator('body').evaluate(() => {
      return typeof (window as any).hostLoaded !== 'undefined'
    })
    expect(childHasHostFlag).toBe(false)
  }, 30000)

  it('should have separate base paths', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    // Host base path
    const hostBasePath = await ctx.page.evaluate(() => {
      return (window as any).hostBasePath
    })
    expect(hostBasePath).toBe('/')

    // Child base path
    const childBasePath = await iframe.locator('body').evaluate(() => {
      return (window as any).childBasePath
    })
    expect(childBasePath).toBe(`/apps/${ctx.childId}/proxy/`)

    // Base paths should be different
    expect(hostBasePath).not.toBe(childBasePath)
  }, 30000)

  it('should not have proxy metadata in host', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Host should NOT have proxy metadata
    const hostHasProxyInfo = await ctx.page.evaluate(() => {
      return typeof (window as any).__VROOLI_PROXY_INFO__ !== 'undefined'
    })
    expect(hostHasProxyInfo).toBe(false)

    // Child should have proxy metadata
    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    const childHasProxyInfo = await iframe.locator('body').evaluate(() => {
      return typeof (window as any).__VROOLI_PROXY_INFO__ !== 'undefined'
    })
    expect(childHasProxyInfo).toBe(true)
  }, 30000)

  it('should load separate CSS for host and child', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    // Host should have host-styles.css
    const hostHasHostStyles = await ctx.page.evaluate(() => {
      const links = Array.from(document.querySelectorAll('link[rel="stylesheet"]'))
      return links.some((link) => (link as HTMLLinkElement).href.includes('host-styles.css'))
    })
    expect(hostHasHostStyles).toBe(true)

    // Host should NOT have child styles.css
    const hostHasChildStyles = await ctx.page.evaluate(() => {
      const links = Array.from(document.querySelectorAll('link[rel="stylesheet"]'))
      return links.some((link) => {
        const href = (link as HTMLLinkElement).href
        return href.includes('styles.css') && !href.includes('host-styles.css')
      })
    })
    expect(hostHasChildStyles).toBe(false)

    // Child should have styles.css
    const childHasStyles = await iframe.locator('body').evaluate(() => {
      const links = Array.from(document.querySelectorAll('link[rel="stylesheet"]'))
      return links.some((link) => (link as HTMLLinkElement).href.includes('styles.css'))
    })
    expect(childHasStyles).toBe(true)

    // Child should NOT have host-styles.css
    const childHasHostStyles = await iframe.locator('body').evaluate(() => {
      const links = Array.from(document.querySelectorAll('link[rel="stylesheet"]'))
      return links.some((link) => (link as HTMLLinkElement).href.includes('host-styles.css'))
    })
    expect(childHasHostStyles).toBe(false)
  }, 30000)

  it('should load separate JavaScript for host and child', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    // Host should have host-main.js
    const hostHasHostJs = await ctx.page.evaluate(() => {
      const scripts = Array.from(document.querySelectorAll('script[src]'))
      return scripts.some((script) => (script as HTMLScriptElement).src.includes('host-main.js'))
    })
    expect(hostHasHostJs).toBe(true)

    // Host should NOT have child main.js
    const hostHasChildJs = await ctx.page.evaluate(() => {
      const scripts = Array.from(document.querySelectorAll('script[src]'))
      return scripts.some((script) => {
        const src = (script as HTMLScriptElement).src
        return src.includes('main.js') && !src.includes('host-main.js')
      })
    })
    expect(hostHasChildJs).toBe(false)

    // Child should have main.js
    const childHasJs = await iframe.locator('body').evaluate(() => {
      const scripts = Array.from(document.querySelectorAll('script[src]'))
      return scripts.some((script) => (script as HTMLScriptElement).src.includes('main.js'))
    })
    expect(childHasJs).toBe(true)

    // Child should NOT have host-main.js
    const childHasHostJs = await iframe.locator('body').evaluate(() => {
      const scripts = Array.from(document.querySelectorAll('script[src]'))
      return scripts.some((script) => (script as HTMLScriptElement).src.includes('host-main.js'))
    })
    expect(childHasHostJs).toBe(false)
  }, 30000)

  it('should not leak styles between host and child', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    // Host background should be red (from host-styles.css)
    const hostBg = await ctx.page.evaluate(() => {
      return getComputedStyle(document.body).backgroundColor
    })

    // Child background should be blue (from styles.css)
    const childBg = await iframe.locator('body').evaluate(() => {
      return getComputedStyle(document.body).backgroundColor
    })

    // They should be different (iframe isolation)
    // Note: Exact color values might vary, but they should be different
    expect(hostBg).not.toBe(childBg)
  }, 30000)

  it('should isolate DOM queries', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    // Host should only find host content
    const hostContentFromHost = await ctx.page.evaluate(() => {
      return document.querySelector('[data-testid="host-content"]') !== null
    })
    expect(hostContentFromHost).toBe(true)

    const childContentFromHost = await ctx.page.evaluate(() => {
      return document.querySelector('[data-testid="child-content"]') !== null
    })
    expect(childContentFromHost).toBe(false) // Child content is in iframe, not accessible

    // Child should only find child content
    const childContentFromChild = await iframe.locator('body').evaluate(() => {
      return document.querySelector('[data-testid="child-content"]') !== null
    })
    expect(childContentFromChild).toBe(true)

    const hostContentFromChild = await iframe.locator('body').evaluate(() => {
      return document.querySelector('[data-testid="host-content"]') !== null
    })
    expect(hostContentFromChild).toBe(false) // Host content is not accessible from iframe
  }, 30000)
})
