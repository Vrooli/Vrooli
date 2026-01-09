/**
 * E2E Tests: Network Monitoring
 *
 * Tests network request tracking, console monitoring, and error detection.
 * Helps identify issues like failed requests, console errors, or missing resources.
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import type { TestContext } from './setup.js'
import { setupTestEnvironment, cleanupTestEnvironment } from './setup.js'

describe('Network Monitoring', () => {
  let ctx: TestContext

  beforeAll(async () => {
    ctx = await setupTestEnvironment()
  }, 60000)

  afterAll(async () => {
    await cleanupTestEnvironment(ctx)
  })

  it('should track all network requests and identify failures', async () => {
    const allRequests: Array<{
      url: string
      method: string
      resourceType: string
      status?: number
      failure?: string
    }> = []

    // Monitor all requests
    ctx.page.on('request', (request) => {
      allRequests.push({
        url: request.url(),
        method: request.method(),
        resourceType: request.resourceType(),
      })
    })

    // Monitor responses
    ctx.page.on('response', (response) => {
      const req = allRequests.find((r) => r.url === response.url())
      if (req) {
        req.status = response.status()
      }
    })

    // Monitor failures
    ctx.page.on('requestfailed', (request) => {
      const req = allRequests.find((r) => r.url === request.url())
      if (req) {
        req.failure = request.failure()?.errorText || 'unknown'
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Filter to important requests
    const documentRequests = allRequests.filter((r) => r.resourceType === 'document')
    const scriptRequests = allRequests.filter((r) => r.resourceType === 'script')
    const stylesheetRequests = allRequests.filter((r) => r.resourceType === 'stylesheet')
    const failedRequests = allRequests.filter((r) => r.failure || (r.status && r.status >= 400))

    // Log summary
    console.log('\n=== Network Request Summary ===')
    console.log(`Total requests: ${allRequests.length}`)
    console.log(`Document requests: ${documentRequests.length}`)
    console.log(`Script requests: ${scriptRequests.length}`)
    console.log(`Stylesheet requests: ${stylesheetRequests.length}`)
    console.log(`Failed requests: ${failedRequests.length}`)

    if (failedRequests.length > 0) {
      console.log('\n=== Failed Requests ===')
      failedRequests.forEach((req) => {
        console.log(`  ${req.method} ${req.url}`)
        console.log(`    Type: ${req.resourceType}`)
        console.log(`    Status: ${req.status || 'N/A'}`)
        console.log(`    Failure: ${req.failure || 'N/A'}`)
      })
    }

    // Should have no failed requests
    expect(failedRequests).toEqual([])
  }, 30000)

  it('should collect all console messages from host and iframe', async () => {
    const consoleMessages: Array<{
      type: string
      text: string
      location: string
    }> = []

    ctx.page.on('console', (msg) => {
      consoleMessages.push({
        type: msg.type(),
        text: msg.text(),
        location: msg.location().url || 'unknown',
      })
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Wait a bit for iframe console messages
    await ctx.page.waitForTimeout(2000)

    console.log('\n=== Console Messages ===')
    consoleMessages.forEach((msg) => {
      console.log(`[${msg.type.toUpperCase()}] ${msg.text}`)
      console.log(`  Location: ${msg.location}`)
    })

    // Filter errors and warnings
    const errors = consoleMessages.filter((m) => m.type === 'error')
    const warnings = consoleMessages.filter((m) => m.type === 'warning')

    if (errors.length > 0) {
      console.log('\n=== ERRORS FOUND ===')
      errors.forEach((err) => {
        console.log(`  ${err.text}`)
        console.log(`  Location: ${err.location}`)
      })
    }

    if (warnings.length > 0) {
      console.log('\n=== WARNINGS FOUND ===')
      warnings.forEach((warn) => {
        console.log(`  ${warn.text}`)
        console.log(`  Location: ${warn.location}`)
      })
    }

    // Should have no errors
    expect(errors).toEqual([])
  }, 30000)

  it('should detect JavaScript errors', async () => {
    const jsErrors: string[] = []

    ctx.page.on('pageerror', (error) => {
      jsErrors.push(error.message)
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Wait for any delayed errors
    await ctx.page.waitForTimeout(1000)

    if (jsErrors.length > 0) {
      console.log('\n=== JavaScript Errors ===')
      jsErrors.forEach((err) => {
        console.log(`  ${err}`)
      })
    }

    expect(jsErrors).toEqual([])
  }, 30000)

  it('should verify resource types are correct', async () => {
    const resources: Record<string, number> = {}

    ctx.page.on('response', (response) => {
      const contentType = response.headers()['content-type'] || 'unknown'
      const url = response.url()

      // Categorize by extension
      if (url.endsWith('.js')) {
        resources.javascript = (resources.javascript || 0) + 1
      } else if (url.endsWith('.css')) {
        resources.stylesheet = (resources.stylesheet || 0) + 1
      } else if (url.endsWith('.html')) {
        resources.html = (resources.html || 0) + 1
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    console.log('\n=== Resource Types ===')
    console.log(`JavaScript files: ${resources.javascript || 0}`)
    console.log(`Stylesheets: ${resources.stylesheet || 0}`)
    console.log(`HTML documents: ${resources.html || 0}`)

    // Should have loaded at least some resources
    expect(resources.javascript).toBeGreaterThan(0)
    expect(resources.stylesheet).toBeGreaterThan(0)
    expect(resources.html).toBeGreaterThan(0)
  }, 30000)

  it('should measure page load performance', async () => {
    const startTime = Date.now()

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const loadTime = Date.now() - startTime

    console.log(`\n=== Page Load Performance ===`)
    console.log(`Total load time: ${loadTime}ms`)

    // Page should load in reasonable time (< 5 seconds)
    expect(loadTime).toBeLessThan(5000)
  }, 30000)

  it('should detect mixed content warnings', async () => {
    const mixedContentWarnings: string[] = []

    ctx.page.on('console', (msg) => {
      if (
        msg.type() === 'warning' &&
        msg.text().toLowerCase().includes('mixed content')
      ) {
        mixedContentWarnings.push(msg.text())
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    expect(mixedContentWarnings).toEqual([])
  }, 30000)

  it('should detect CSP violations', async () => {
    const cspViolations: string[] = []

    ctx.page.on('console', (msg) => {
      if (
        msg.type() === 'error' &&
        (msg.text().toLowerCase().includes('content security policy') ||
          msg.text().toLowerCase().includes('csp'))
      ) {
        cspViolations.push(msg.text())
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    expect(cspViolations).toEqual([])
  }, 30000)

  it('should verify response headers are reasonable', async () => {
    const response = await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`)
    const headers = response?.headers() || {}

    // Should have content-type
    expect(headers['content-type']).toBeDefined()
    expect(headers['content-type']).toContain('text/html')

    // Should not expose sensitive information
    expect(headers['x-powered-by']).toBeUndefined()
  }, 30000)

  it('should track timing for all requests', async () => {
    const requestTimings: Array<{
      url: string
      duration: number
    }> = []

    const requestStartTimes = new Map<string, number>()

    ctx.page.on('request', (request) => {
      requestStartTimes.set(request.url(), Date.now())
    })

    ctx.page.on('response', (response) => {
      const startTime = requestStartTimes.get(response.url())
      if (startTime) {
        const duration = Date.now() - startTime
        requestTimings.push({
          url: response.url(),
          duration,
        })
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Find slowest requests
    const slowRequests = requestTimings.filter((r) => r.duration > 1000).sort((a, b) => b.duration - a.duration)

    if (slowRequests.length > 0) {
      console.log('\n=== Slow Requests (>1s) ===')
      slowRequests.forEach((req) => {
        console.log(`  ${req.duration}ms - ${req.url}`)
      })
    }

    // No request should take more than 3 seconds (generous threshold for CI)
    const tooSlowRequests = requestTimings.filter((r) => r.duration > 3000)
    expect(tooSlowRequests).toEqual([])
  }, 30000)
})
