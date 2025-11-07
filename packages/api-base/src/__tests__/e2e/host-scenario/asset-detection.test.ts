/**
 * E2E Tests: Asset Detection and SPA Fallback
 *
 * Tests that asset requests are correctly detected and handled differently
 * from page requests. Verifies that SPA fallback doesn't interfere with
 * asset loading.
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import type { TestContext } from './setup.js'
import { setupTestEnvironment, cleanupTestEnvironment, makeRequest } from './setup.js'

describe('Asset Detection and SPA Fallback', () => {
  let ctx: TestContext

  beforeAll(async () => {
    ctx = await setupTestEnvironment(1000) // Use port offset to avoid conflicts
  }, 60000)

  afterAll(async () => {
    await cleanupTestEnvironment(ctx)
  })

  it('should return 404 for missing .js files (not HTML)', async () => {
    const response = await makeRequest(ctx.hostUiPort, '/missing-file.js')

    expect(response.status).toBe(404)
    expect(response.body).not.toContain('<!DOCTYPE html>')
    expect(response.body).not.toContain('<html')
    expect(response.headers['content-type']).not.toContain('text/html')
  }, 30000)

  it('should return 404 for missing .css files (not HTML)', async () => {
    const response = await makeRequest(ctx.hostUiPort, '/missing-file.css')

    expect(response.status).toBe(404)
    expect(response.body).not.toContain('<!DOCTYPE html>')
    expect(response.body).not.toContain('<html')
    expect(response.headers['content-type']).not.toContain('text/html')
  }, 30000)

  it('should return 404 for missing .json files (not HTML)', async () => {
    const response = await makeRequest(ctx.hostUiPort, '/missing-file.json')

    expect(response.status).toBe(404)
    expect(response.body).not.toContain('<!DOCTYPE html>')
  }, 30000)

  it('should return 404 for missing .png files (not HTML)', async () => {
    const response = await makeRequest(ctx.hostUiPort, '/missing-image.png')

    expect(response.status).toBe(404)
    expect(response.body).not.toContain('<!DOCTYPE html>')
  }, 30000)

  it('should return 404 for missing .woff files (not HTML)', async () => {
    const response = await makeRequest(ctx.hostUiPort, '/missing-font.woff')

    expect(response.status).toBe(404)
    expect(response.body).not.toContain('<!DOCTYPE html>')
  }, 30000)

  it('should return 404 for Vite HMR paths', async () => {
    const response = await makeRequest(ctx.hostUiPort, '/@vite/client')

    expect(response.status).toBe(404)
    expect(response.body).not.toContain('<!DOCTYPE html>')
  }, 30000)

  it('should return 404 for /assets/ paths', async () => {
    const response = await makeRequest(ctx.hostUiPort, '/assets/missing-file.js')

    expect(response.status).toBe(404)
    expect(response.body).not.toContain('<!DOCTYPE html>')
  }, 30000)

  it('should return 404 for /src/ paths', async () => {
    const response = await makeRequest(ctx.hostUiPort, '/src/missing-file.ts')

    expect(response.status).toBe(404)
    expect(response.body).not.toContain('<!DOCTYPE html>')
  }, 30000)

  it('should handle query strings in asset URLs', async () => {
    const response = await makeRequest(ctx.hostUiPort, '/missing-file.js?v=123456')

    expect(response.status).toBe(404)
    expect(response.body).not.toContain('<!DOCTYPE html>')
  }, 30000)

  it('should handle hash fragments in asset URLs', async () => {
    const response = await makeRequest(ctx.hostUiPort, '/missing-file.css#section')

    expect(response.status).toBe(404)
    expect(response.body).not.toContain('<!DOCTYPE html>')
  }, 30000)

  it('should load existing assets correctly', async () => {
    // CSS
    const cssResponse = await makeRequest(ctx.hostUiPort, '/host-styles.css')
    expect(cssResponse.status).toBe(200)
    expect(cssResponse.headers['content-type']).toContain('text/css')
    expect(cssResponse.body).toContain('background: red')

    // JS
    const jsResponse = await makeRequest(ctx.hostUiPort, '/host-main.js')
    expect(jsResponse.status).toBe(200)
    expect(jsResponse.headers['content-type']).toContain('application/javascript')
    expect(jsResponse.body).toContain('Host main.js loaded')
  }, 30000)

  it('should detect assets through browser (not receiving HTML for JS)', async () => {
    const wrongContentTypes: Array<{ url: string; contentType: string }> = []

    ctx.page.on('response', (response) => {
      const url = response.url()
      const contentType = response.headers()['content-type'] || ''

      // Check if JS files are getting text/html
      if (url.endsWith('.js') && contentType.includes('text/html')) {
        wrongContentTypes.push({ url, contentType })
      }

      // Check if CSS files are getting text/html
      if (url.endsWith('.css') && contentType.includes('text/html')) {
        wrongContentTypes.push({ url, contentType })
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    if (wrongContentTypes.length > 0) {
      console.log('\n=== Wrong Content Types ===')
      wrongContentTypes.forEach((item) => {
        console.log(`  ${item.url} => ${item.contentType}`)
      })
    }

    expect(wrongContentTypes).toEqual([])
  }, 30000)

  it('should not have "Unexpected token <" errors (HTML-as-JS bug)', async () => {
    const syntaxErrors: string[] = []

    ctx.page.on('console', (msg) => {
      if (
        msg.type() === 'error' &&
        (msg.text().includes('Unexpected token') || msg.text().includes('SyntaxError'))
      ) {
        syntaxErrors.push(msg.text())
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    if (syntaxErrors.length > 0) {
      console.log('\n=== Syntax Errors (Possible HTML-as-JS) ===')
      syntaxErrors.forEach((err) => {
        console.log(`  ${err}`)
      })
    }

    expect(syntaxErrors).toEqual([])
  }, 30000)

  it('should correctly identify common file extensions', async () => {
    const testCases = [
      { path: '/file.js', shouldBe404: true },
      { path: '/file.mjs', shouldBe404: true },
      { path: '/file.css', shouldBe404: true },
      { path: '/file.json', shouldBe404: true },
      { path: '/file.svg', shouldBe404: true },
      { path: '/file.png', shouldBe404: true },
      { path: '/file.jpg', shouldBe404: true },
      { path: '/file.woff', shouldBe404: true },
      { path: '/file.woff2', shouldBe404: true },
      { path: '/file.ttf', shouldBe404: true },
      { path: '/file.map', shouldBe404: true },
    ]

    for (const testCase of testCases) {
      const response = await makeRequest(ctx.hostUiPort, testCase.path)

      if (testCase.shouldBe404) {
        expect(response.status).toBe(404)
        expect(response.body).not.toContain('<!DOCTYPE html>')
      }
    }
  }, 30000)

  it('should detect asset prefixes correctly', async () => {
    const testCases = [
      '/@vite/client',
      '/@react-refresh',
      '/assets/file.js',
      '/static/file.js',
      '/public/file.js',
      '/_next/static/file.js',
    ]

    for (const path of testCases) {
      const response = await makeRequest(ctx.hostUiPort, path)

      expect(response.status).toBe(404)
      expect(response.body).not.toContain('<!DOCTYPE html>')
    }
  }, 30000)

  it('should handle case-insensitive extensions', async () => {
    const testCases = [
      '/file.JS', // Uppercase
      '/file.Css', // Mixed case
      '/file.PNG', // Uppercase image
    ]

    for (const path of testCases) {
      const response = await makeRequest(ctx.hostUiPort, path)

      expect(response.status).toBe(404)
      expect(response.body).not.toContain('<!DOCTYPE html>')
    }
  }, 30000)

  it('should verify child assets also use correct detection', async () => {
    // Test child's missing asset through proxy
    const childAssetResponse = await makeRequest(
      ctx.hostUiPort,
      `/apps/${ctx.childId}/proxy/missing-asset.js`
    )

    expect(childAssetResponse.status).toBe(404)
    expect(childAssetResponse.body).not.toContain('<!DOCTYPE html>')

    // Test child's existing asset through proxy
    const childCssResponse = await makeRequest(
      ctx.hostUiPort,
      `/apps/${ctx.childId}/proxy/styles.css`
    )

    expect(childCssResponse.status).toBe(200)
    expect(childCssResponse.headers['content-type']).toContain('text/css')
  }, 30000)

  it('should handle asset detection in browser for child', async () => {
    const wrongContentTypes: Array<{ url: string; contentType: string }> = []

    ctx.page.on('response', (response) => {
      const url = response.url()
      const contentType = response.headers()['content-type'] || ''

      // Check child assets through proxy
      if (url.includes(`/apps/${ctx.childId}/proxy/`)) {
        if (url.endsWith('.js') && contentType.includes('text/html')) {
          wrongContentTypes.push({ url, contentType })
        }
        if (url.endsWith('.css') && contentType.includes('text/html')) {
          wrongContentTypes.push({ url, contentType })
        }
      }
    })

    await ctx.page.goto(`http://127.0.0.1:${ctx.hostUiPort}/index.html`, {
      waitUntil: 'networkidle',
    })

    expect(wrongContentTypes).toEqual([])
  }, 30000)
})
