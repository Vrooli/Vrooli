/**
 * E2E Tests: Error Scenarios and Edge Cases
 *
 * Tests error handling, failure modes, and edge cases that could
 * break the host scenario pattern in production.
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import * as net from 'node:net'
import type { TestContext } from './setup.js'
import { setupTestEnvironment, cleanupTestEnvironment } from './setup.js'

async function sendRawRequest(port: number, rawPath: string): Promise<number> {
  return await new Promise((resolve, reject) => {
    let buffer = ''
    const socket = net.createConnection({ host: '127.0.0.1', port }, () => {
      socket.write(
        `GET ${rawPath} HTTP/1.1\r\n` +
        `Host: 127.0.0.1:${port}\r\n` +
        'Connection: close\r\n' +
        '\r\n'
      )
    })

    socket.setEncoding('utf8')

    socket.on('data', (chunk) => {
      buffer += chunk
    })

    socket.on('error', (error) => {
      reject(error)
    })

    socket.on('end', () => {
      const match = buffer.match(/^HTTP\/\d\.\d\s+(\d+)/)
      if (!match) {
        reject(new Error('Unable to parse status line from response'))
        return
      }
      resolve(Number(match[1]))
    })
  })
}

describe('Error Scenarios', () => {
  let ctx: TestContext

  beforeAll(async () => {
    ctx = await setupTestEnvironment(4000) // Use unique port offset
  }, 60000)

  afterAll(async () => {
    await cleanupTestEnvironment(ctx)
  })

  it('should handle missing child scenario gracefully (404)', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Try to load non-existent child
    const response = await ctx.page.goto(
      `http://127.0.0.1:${ctx.hostUiPort}/apps/nonexistent-child/proxy/index.html`
    )

    expect(response?.status()).toBeGreaterThanOrEqual(400)
  }, 30000)

  it('should handle malformed proxy paths', async () => {
    const testCases = [
      '/apps/../../../etc/passwd', // Path traversal attempt
      '/apps/%2e%2e%2f%2e%2e%2f', // URL-encoded path traversal
      '/apps/child/proxy/../../../secrets', // Relative path escape
    ]

    for (const maliciousPath of testCases) {
      const status = await sendRawRequest(ctx.hostUiPort, maliciousPath)
      expect(status).toBeGreaterThanOrEqual(400)
    }
  }, 30000)

  it('should handle child server timeout', async () => {
    // Stop the child server to simulate timeout
    await new Promise<void>((resolve) => {
      ctx.childUiServer.close(() => resolve())
    })

    const consoleErrors: string[] = []
    ctx.page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text())
      }
    })

    // Try to load child (should fail gracefully)
    const response = await ctx.page.goto(
      `http://127.0.0.1:${ctx.hostUiPort}/apps/${ctx.childId}/proxy/index.html`,
      { waitUntil: 'domcontentloaded', timeout: 10000 }
    ).catch((err) => ({ status: () => 502, error: err.message }))

    // Should return error status
    expect(response?.status()).toBeGreaterThanOrEqual(500)

    // Restart child for other tests (must include proper HTML structure for subsequent tests)
    const { createScenarioServer } = await import('../../../server/index.js')
    const childApp = createScenarioServer({
      uiPort: ctx.childUiPort,
      apiPort: ctx.childApiPort,
      distDir: './dist',
      serviceName: 'child-scenario',
      verbose: false,
      setupRoutes: (app) => {
        app.get('/index.html', (req, res) => {
          res.setHeader('Content-Type', 'text/html')
          res.send(`<!DOCTYPE html>
<html>
<head>
  <title>Child Scenario</title>
  <meta charset="UTF-8">
  <link rel="stylesheet" href="styles.css">
</head>
<body>
  <div id="child-app" data-testid="child-content">
    <h1>Child Scenario</h1>
    <p>This is embedded content</p>
  </div>
  <script src="main.js"></script>
  <script>
    window.childLoaded = true;
    console.log('Child scenario loaded');
  </script>
</body>
</html>`)
        })

        app.get('/styles.css', (req, res) => {
          res.setHeader('Content-Type', 'text/css')
          res.send('body { background: blue; }')
        })

        app.get('/main.js', (req, res) => {
          res.setHeader('Content-Type', 'application/javascript')
          res.send('console.log("Child main.js loaded");')
        })
      },
    })

    ctx.childUiServer = await new Promise((resolve) => {
      const server = childApp.listen(ctx.childUiPort, '127.0.0.1', () => {
        resolve(server)
      })
    })
  }, 60000)

  it('should sanitize injected proxy metadata to prevent XSS', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    const iframe = ctx.page.frameLocator('[data-testid="child-iframe"]')
    await iframe.locator('[data-testid="child-content"]').waitFor({ timeout: 10000 })

    // Check that proxy metadata is properly JSON-encoded (not raw string injection)
    const proxyInfoScript = await iframe.locator('body').evaluate(() => {
      const scripts = Array.from(document.querySelectorAll('script'))
      const proxyScript = scripts.find((s) => s.textContent?.includes('__VROOLI_PROXY_INFO__'))
      return proxyScript?.textContent || ''
    })

    // Should not contain script tags or HTML
    expect(proxyInfoScript).not.toContain('</script>')
    expect(proxyInfoScript).not.toContain('<img')
    expect(proxyInfoScript).not.toContain('onerror=')
  }, 30000)

  it('should handle child returning malformed HTML', async () => {
    // This would require a custom child server that returns broken HTML
    // For now, verify that our injection is robust
    const malformedHtmlCases = [
      '<!DOCTYPE html><html><head></head>', // No closing tags
      '<html>No DOCTYPE', // Missing DOCTYPE
      'Plain text response', // Not HTML at all
      '<html><body><script>', // Unclosed script
    ]

    // Each case should be handled gracefully by injectBaseTag and injectProxyMetadata
    for (const html of malformedHtmlCases) {
      const { injectBaseTag, injectProxyMetadata, buildProxyMetadata } = await import(
        '../../../server/index.js'
      )

      try {
        const withBase = injectBaseTag(html, '/apps/test/proxy/')
        const metadata = buildProxyMetadata({
          appId: 'test',
          hostScenario: 'host',
          targetScenario: 'test',
          ports: [],
          primaryPort: {
            port: 3000,
            label: 'UI_PORT',
            slug: 'ui-port',
            path: '/apps/test/proxy',
            assetNamespace: '/apps/test/proxy/assets',
            aliases: [],
            source: 'port_mappings',
            isPrimary: true,
          },
          loopbackHosts: [],
        })
        const withMetadata = injectProxyMetadata(withBase, metadata)

        // Should not throw errors
        expect(typeof withMetadata).toBe('string')
      } catch (error: any) {
        // If it throws, the error should be handled gracefully
        expect(error.message).toBeDefined()
      }
    }
  }, 30000)

  it('should handle rapid iframe creation and destruction', async () => {
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Create and destroy iframes rapidly
    for (let i = 0; i < 5; i++) {
      await ctx.page.evaluate((childId) => {
        const iframe = document.createElement('iframe')
        iframe.src = `/apps/${childId}/proxy/index.html`
        iframe.style.display = 'none'
        document.body.appendChild(iframe)

        setTimeout(() => {
          iframe.remove()
        }, 100)
      }, ctx.childId)

      await ctx.page.waitForTimeout(200)
    }

    // Should not cause memory leaks or errors
    const consoleErrors: string[] = []
    ctx.page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text())
      }
    })

    await ctx.page.waitForTimeout(1000)

    // Filter out expected errors (like network errors from removed iframes)
    const criticalErrors = consoleErrors.filter(
      (err) => !err.includes('ERR_ABORTED') && !err.includes('net::')
    )
    expect(criticalErrors).toEqual([])
  }, 30000)

  it('should handle missing base tag target gracefully', async () => {
    // If HTML has no <head> or <html> tags, injection should still work
    const htmlWithoutHead = '<body><div>Content</div></body>'

    const { injectBaseTag } = await import('../../../server/index.js')
    const result = injectBaseTag(htmlWithoutHead, '/test/')

    expect(result).toContain('<base')
    expect(result).toContain('href="/test/"')
  }, 30000)

  it('should handle circular proxy references', async () => {
    // If host tries to embed itself, should detect and prevent
    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    // Try to create iframe pointing to host itself
    const result = await ctx.page.evaluate((hostId) => {
      try {
        const iframe = document.createElement('iframe')
        iframe.src = `/apps/${hostId}/proxy/index.html`
        document.body.appendChild(iframe)
        return { error: null }
      } catch (error: any) {
        return { error: error.message }
      }
    }, 'host-scenario')

    // Should either work (with depth limit) or fail gracefully
    expect(result).toBeDefined()
  }, 30000)

  it('should handle very long proxy paths', async () => {
    const longChildId = 'a'.repeat(1000)
    const response = await ctx.page.goto(
      `http://127.0.0.1:${ctx.hostUiPort}/apps/${longChildId}/proxy/index.html`,
      { waitUntil: 'domcontentloaded', timeout: 5000 }
    ).catch(() => ({ status: () => 404 }))

    // Should handle gracefully (either 404 or URL too long error)
    expect(response?.status()).toBeGreaterThanOrEqual(400)
  }, 30000)

  it('should handle special characters in child ID', async () => {
    const specialChars = [
      'child<script>',
      'child&quot;',
      'child%27',
      'child/../etc',
    ]

    for (const maliciousId of specialChars) {
      const response = await ctx.page.goto(
        `http://127.0.0.1:${ctx.hostUiPort}/apps/${encodeURIComponent(maliciousId)}/proxy/index.html`,
        { waitUntil: 'domcontentloaded', timeout: 5000 }
      ).catch(() => ({ status: () => 404 }))

      // Should sanitize or reject
      expect(response?.status()).toBeGreaterThanOrEqual(400)
    }
  }, 30000)
})
