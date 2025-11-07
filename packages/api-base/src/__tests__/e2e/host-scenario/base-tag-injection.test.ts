/**
 * E2E Tests: Base Tag Injection
 *
 * Tests that base tags are correctly injected for both host and child scenarios.
 * This is critical for proper asset resolution in nested paths.
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import type { TestContext } from './setup.js'
import { setupTestEnvironment, cleanupTestEnvironment } from './setup.js'

describe('Base Tag Injection', () => {
  let ctx: TestContext

  beforeAll(async () => {
    ctx = await setupTestEnvironment()
  }, 60000)

  afterAll(async () => {
    await cleanupTestEnvironment(ctx)
  })

  it('should inject correct base tag in host page', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const baseHref = await ctx.page.evaluate(() => {
      const base = document.querySelector('base[data-host-self="injected"]')
      return base?.getAttribute('href') || null
    })

    expect(baseHref).toBe('/')
  }, 30000)

  it('should inject correct base tag in child iframe', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    const iframeBaseHref = await iframe.locator('base').getAttribute('href')

    expect(iframeBaseHref).toBe(`/apps/${ctx.childId}/proxy/`)
  }, 30000)

  it('should have different base tags for host and child', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Get host base tag
    const hostBaseHref = await ctx.page.evaluate(() => {
      return document.querySelector('base')?.getAttribute('href') || null
    })

    // Get child base tag
    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    const childBaseHref = await iframe.locator('base').getAttribute('href')

    // They should be different
    expect(hostBaseHref).toBe('/')
    expect(childBaseHref).toBe(`/apps/${ctx.childId}/proxy/`)
    expect(hostBaseHref).not.toBe(childBaseHref)
  }, 30000)

  it('should have data attribute on injected base tags', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Check host base tag has data attribute
    const hostDataAttr = await ctx.page.evaluate(() => {
      const base = document.querySelector('base')
      return base?.getAttribute('data-host-self') || null
    })
    expect(hostDataAttr).toBe('injected')

    // Check child base tag has data attribute
    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    const childDataAttr = await iframe.locator('base').getAttribute('data-host-proxy')
    expect(childDataAttr).toBe('injected')
  }, 30000)

  it('should inject base tag early in head (before other tags)', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Check host base tag position
    const hostBasePosition = await ctx.page.evaluate(() => {
      const head = document.querySelector('head')
      if (!head) return -1

      const children = Array.from(head.children)
      const baseIndex = children.findIndex((el) => el.tagName === 'BASE')
      const linkIndex = children.findIndex((el) => el.tagName === 'LINK')

      return { baseIndex, linkIndex }
    })

    // Base should come before link tags
    expect(hostBasePosition.baseIndex).toBeGreaterThanOrEqual(0)
    if (hostBasePosition.linkIndex >= 0) {
      expect(hostBasePosition.baseIndex).toBeLessThan(hostBasePosition.linkIndex)
    }

    // Check child base tag position
    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    const childBasePosition = await iframe.locator('body').evaluate(() => {
      const head = document.querySelector('head')
      if (!head) return -1

      const children = Array.from(head.children)
      const baseIndex = children.findIndex((el) => el.tagName === 'BASE')
      const linkIndex = children.findIndex((el) => el.tagName === 'LINK')

      return { baseIndex, linkIndex }
    })

    expect(childBasePosition.baseIndex).toBeGreaterThanOrEqual(0)
    if (childBasePosition.linkIndex >= 0) {
      expect(childBasePosition.baseIndex).toBeLessThan(childBasePosition.linkIndex)
    }
  }, 30000)

  it('should have trailing slash in base href', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Check host base href ends with /
    const hostBaseHref = await ctx.page.evaluate(() => {
      return document.querySelector('base')?.getAttribute('href') || ''
    })
    expect(hostBaseHref).toMatch(/\/$/)

    // Check child base href ends with /
    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    const childBaseHref = await iframe.locator('base').getAttribute('href')
    expect(childBaseHref).toMatch(/\/$/)
  }, 30000)

  it('should only inject one base tag per document', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Check host has only one base tag
    const hostBaseCount = await ctx.page.evaluate(() => {
      return document.querySelectorAll('base').length
    })
    expect(hostBaseCount).toBe(1)

    // Check child has only one base tag
    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    const childBaseCount = await iframe.locator('body').evaluate(() => {
      return document.querySelectorAll('base').length
    })
    expect(childBaseCount).toBe(1)
  }, 30000)

  it('should resolve relative URLs correctly with base tag', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Test host relative URL resolution
    const hostResolvedUrl = await ctx.page.evaluate(() => {
      const a = document.createElement('a')
      a.href = 'test-file.js'
      return a.href
    })
    expect(hostResolvedUrl).toBe(`http://127.0.0.1:${ctx.hostUiPort}/test-file.js`)

    // Test child relative URL resolution
    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    const childResolvedUrl = await iframe.locator('body').evaluate(() => {
      const a = document.createElement('a')
      a.href = 'test-file.js'
      return a.href
    })
    expect(childResolvedUrl).toBe(
      `http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/proxy/test-file.js`
    )
  }, 30000)
})
