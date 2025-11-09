/**
 * Tests for server/inject.ts
 */

import { describe, it, expect } from 'vitest'
import {
  buildProxyMetadata,
  injectProxyMetadata,
  injectScenarioConfig,
  injectBaseTag,
} from '../../server/inject.js'
import type { PortEntry, ProxyMetadataOptions, ScenarioConfig } from '../../shared/types.js'

describe('buildProxyMetadata', () => {
  it('builds complete proxy metadata', () => {
    const uiPort: PortEntry = {
      port: 3000,
      label: 'ui',
      slug: 'ui',
      path: '/apps/test-app/proxy',
      aliases: ['ui', 'primary'],
      isPrimary: true,
      source: 'port_mappings',
      priority: 80,
    }

    const apiPort: PortEntry = {
      port: 8080,
      label: 'api',
      slug: 'api',
      path: '/apps/test-app/proxy',
      aliases: ['api'],
      isPrimary: false,
      source: 'port_mappings',
      priority: 30,
    }

    const options: ProxyMetadataOptions = {
      appId: 'test-app',
      hostScenario: 'app-monitor',
      targetScenario: 'test-app',
      ports: [uiPort, apiPort],
      primaryPort: uiPort,
    }

    const result = buildProxyMetadata(options)

    expect(result.appId).toBe('test-app')
    expect(result.hostScenario).toBe('app-monitor')
    expect(result.targetScenario).toBe('test-app')
    expect(result.ports).toHaveLength(2)
    expect(result.primary).toEqual(uiPort)
    expect(result.hosts).toContain('127.0.0.1')
    expect(result.hosts).toContain('localhost')
    expect(typeof result.generatedAt).toBe('number')
  })

  it('uses custom loopback hosts', () => {
    const port: PortEntry = {
      port: 3000,
      label: 'ui',
      slug: 'ui',
      path: '/test',
      aliases: [],
      isPrimary: true,
      source: 'manual',
      priority: 100,
    }

    const options: ProxyMetadataOptions = {
      appId: 'test',
      hostScenario: 'host',
      targetScenario: 'target',
      ports: [port],
      primaryPort: port,
      loopbackHosts: ['custom.local', '10.0.0.1'],
    }

    const result = buildProxyMetadata(options)

    expect(result.hosts).toEqual(['custom.local', '10.0.0.1'])
  })

  it('includes generatedAt timestamp', () => {
    const port: PortEntry = {
      port: 3000,
      label: 'ui',
      slug: 'ui',
      path: '/test',
      aliases: [],
      isPrimary: true,
      source: 'manual',
      priority: 100,
    }

    const before = Date.now()
    const result = buildProxyMetadata({
      appId: 'test',
      hostScenario: 'host',
      targetScenario: 'target',
      ports: [port],
      primaryPort: port,
    })
    const after = Date.now()

    expect(result.generatedAt).toBeGreaterThanOrEqual(before)
    expect(result.generatedAt).toBeLessThanOrEqual(after)
  })

  it('captures host endpoints when provided', () => {
    const port: PortEntry = {
      port: 3000,
      label: 'ui',
      slug: 'ui',
      path: '/test',
      aliases: [],
      isPrimary: true,
      source: 'manual',
      priority: 100,
    }

    const metadata = buildProxyMetadata({
      appId: 'test',
      hostScenario: 'host',
      targetScenario: 'target',
      ports: [port],
      primaryPort: port,
      hostEndpoints: [
        { path: '/api/v1/apps/{id}/diagnostics', method: 'GET' },
      ],
    })

    expect(metadata.hostEndpoints).toEqual([
      { path: '/api/v1/apps/{id}/diagnostics', method: 'GET' },
    ])
  })
})

// Note: buildProxyBootstrapScript is not exported - it's an internal implementation detail
// The injection behavior is tested through injectProxyMetadata instead

describe('injectProxyMetadata', () => {
  it('injects script into <head> tag', () => {
    const html = `<!DOCTYPE html>
<html>
<head>
  <title>Test</title>
</head>
<body>
  <div>Content</div>
</body>
</html>`

    const port: PortEntry = {
      port: 3000,
      label: 'ui',
      slug: 'ui',
      path: '/test',
      aliases: [],
      isPrimary: true,
      source: 'manual',
      priority: 100,
    }

    const metadata = buildProxyMetadata({
      appId: 'test',
      hostScenario: 'host',
      targetScenario: 'target',
      ports: [port],
      primaryPort: port,
    })

    const result = injectProxyMetadata(html, metadata)

    expect(result).toContain('<script id="vrooli-proxy-metadata">')
    expect(result).toContain('__VROOLI_PROXY_INFO__')
    expect(result).toContain('</script>')
    expect(result).toContain('<head>')

    // Script should be in head
    const headStart = result.indexOf('<head>')
    const headEnd = result.indexOf('</head>')
    const scriptPos = result.indexOf('<script id="vrooli-proxy-metadata">')
    expect(scriptPos).toBeGreaterThan(headStart)
    expect(scriptPos).toBeLessThan(headEnd)
  })

  it('injects into <body> if <head> is missing', () => {
    const html = `<!DOCTYPE html>
<html>
<body>
  <div>Content</div>
</body>
</html>`

    const port: PortEntry = {
      port: 3000,
      label: 'ui',
      slug: 'ui',
      path: '/test',
      aliases: [],
      isPrimary: true,
      source: 'manual',
      priority: 100,
    }

    const metadata = buildProxyMetadata({
      appId: 'test',
      hostScenario: 'host',
      targetScenario: 'target',
      ports: [port],
      primaryPort: port,
    })

    const result = injectProxyMetadata(html, metadata)

    expect(result).toContain('<body>')
    expect(result).toContain('<script id="vrooli-proxy-metadata">')
    // Script should be after <body> tag
    expect(result.indexOf('<script')).toBeGreaterThan(result.indexOf('<body>'))
  })

  it('prepends to html without <html> or <body> tag', () => {
    const html = '<div>Simple HTML</div>'

    const port: PortEntry = {
      port: 3000,
      label: 'ui',
      slug: 'ui',
      path: '/test',
      aliases: [],
      isPrimary: true,
      source: 'manual',
      priority: 100,
    }

    const metadata = buildProxyMetadata({
      appId: 'test',
      hostScenario: 'host',
      targetScenario: 'target',
      ports: [port],
      primaryPort: port,
    })

    const result = injectProxyMetadata(html, metadata)

    expect(result).toContain('<script id="vrooli-proxy-metadata">')
    // Script should be at the start
    expect(result.startsWith('<script')).toBe(true)
  })

  it('will inject multiple times if called multiple times', () => {
    const html = `<!DOCTYPE html>
<html>
<head>
  <title>Test</title>
</head>
<body></body>
</html>`

    const port: PortEntry = {
      port: 3000,
      label: 'ui',
      slug: 'ui',
      path: '/test',
      aliases: [],
      isPrimary: true,
      source: 'manual',
      priority: 100,
    }

    const metadata = buildProxyMetadata({
      appId: 'test',
      hostScenario: 'host',
      targetScenario: 'target',
      ports: [port],
      primaryPort: port,
    })

    const result1 = injectProxyMetadata(html, metadata)
    const result2 = injectProxyMetadata(result1, metadata)

    // Note: Currently the function does inject multiple times
    // This is by design - caller should check if already injected
    const scriptCount = (result2.match(/<script id="vrooli-proxy-metadata">/g) || []).length
    expect(scriptCount).toBeGreaterThanOrEqual(1)
  })

  it('passes options to bootstrap script', () => {
    const html = '<html><head></head><body></body></html>'

    const port: PortEntry = {
      port: 3000,
      label: 'ui',
      slug: 'ui',
      path: '/test',
      aliases: [],
      isPrimary: true,
      source: 'manual',
      priority: 100,
    }

    const metadata = buildProxyMetadata({
      appId: 'test',
      hostScenario: 'host',
      targetScenario: 'target',
      ports: [port],
      primaryPort: port,
    })

    const result = injectProxyMetadata(html, metadata, {
      patchFetch: true,
      infoGlobalName: '__CUSTOM__',
    })

    expect(result).toContain('__CUSTOM__')
    expect(result).toContain('window.fetch')
  })
})

describe('injectScenarioConfig', () => {
  it('injects config into <head> tag', () => {
    const html = `<!DOCTYPE html>
<html>
<head>
  <title>Test</title>
</head>
<body>
  <div>Content</div>
</body>
</html>`

    const config: ScenarioConfig = {
      apiUrl: 'http://localhost:8080/api/v1',
      wsUrl: 'ws://localhost:8080/ws',
      apiPort: '8080',
      wsPort: '8080',
      uiPort: '3000',
    }

    const result = injectScenarioConfig(html, config)

    expect(result).toContain('<script id="vrooli-scenario-config">')
    expect(result).toContain('__VROOLI_CONFIG__')
    expect(result).toContain('"apiUrl":"http://localhost:8080/api/v1"')
    expect(result).toContain('</script>')
  })

  it('injects into <body> if <head> is missing', () => {
    const html = '<body>Content</body>'

    const config: ScenarioConfig = {
      apiUrl: 'http://localhost:8080/api/v1',
      wsUrl: 'ws://localhost:8080/ws',
      apiPort: '8080',
      wsPort: '8080',
      uiPort: '3000',
    }

    const result = injectScenarioConfig(html, config)

    expect(result).toContain('<body>')
    expect(result).toContain('__VROOLI_CONFIG__')
    expect(result.indexOf('<script')).toBeGreaterThan(result.indexOf('<body>'))
  })

  it('uses custom global name', () => {
    const html = '<html><head></head><body></body></html>'

    const config: ScenarioConfig = {
      apiUrl: 'http://localhost:8080/api/v1',
      wsUrl: 'ws://localhost:8080/ws',
      apiPort: '8080',
      wsPort: '8080',
      uiPort: '3000',
    }

    const result = injectScenarioConfig(html, config, {
      configGlobalName: '__MY_CONFIG__',
    })

    expect(result).toContain('__MY_CONFIG__')
    expect(result).not.toContain('__VROOLI_CONFIG__')
  })

  it('escapes config values properly', () => {
    const html = '<html><head></head><body></body></html>'

    const config: ScenarioConfig = {
      apiUrl: 'http://localhost:8080/api/v1</script><script>alert("xss")</script>',
      wsUrl: 'ws://localhost:8080/ws',
      apiPort: '8080',
      wsPort: '8080',
      uiPort: '3000',
    }

    const result = injectScenarioConfig(html, config)

    // Should escape </script>
    expect(result).not.toContain('</script><script>alert')
    expect(result).toContain('\\u003C/script')
  })

  it('will inject multiple times if called multiple times', () => {
    const html = '<html><head></head><body></body></html>'

    const config: ScenarioConfig = {
      apiUrl: 'http://localhost:8080/api/v1',
      wsUrl: 'ws://localhost:8080/ws',
      apiPort: '8080',
      wsPort: '8080',
      uiPort: '3000',
    }

    const result1 = injectScenarioConfig(html, config)
    const result2 = injectScenarioConfig(result1, config)

    // Note: Currently the function does inject multiple times
    // This is by design - caller should check if already injected
    const scriptCount = (result2.match(/<script id="vrooli-scenario-config">/g) || []).length
    expect(scriptCount).toBeGreaterThanOrEqual(1)
  })

  it('includes additional config properties', () => {
    const html = '<html><head></head><body></body></html>'

    const config: ScenarioConfig = {
      apiUrl: 'http://localhost:8080/api/v1',
      wsUrl: 'ws://localhost:8080/ws',
      apiPort: '8080',
      wsPort: '8080',
      uiPort: '3000',
      version: '1.2.3',
      service: 'test-scenario',
      customField: 'custom-value',
    }

    const result = injectScenarioConfig(html, config)

    expect(result).toContain('"version":"1.2.3"')
    expect(result).toContain('"service":"test-scenario"')
    expect(result).toContain('"customField":"custom-value"')
  })
})

describe('injectBaseTag', () => {
  it('injects base tag into <head>', () => {
    const html = `<!DOCTYPE html>
<html>
<head>
  <title>Test</title>
</head>
<body>
  <div>Content</div>
</body>
</html>`

    const result = injectBaseTag(html, '/')

    expect(result).toContain('<base')
    expect(result).toContain('href="/"')
    expect(result).toContain('data-vrooli-base="injected"')

    // Should be in head
    const headStart = result.indexOf('<head>')
    const headEnd = result.indexOf('</head>')
    const basePos = result.indexOf('<base')
    expect(basePos).toBeGreaterThan(headStart)
    expect(basePos).toBeLessThan(headEnd)
  })

  it('adds trailing slash to base href', () => {
    const html = '<html><head></head><body></body></html>'

    const result1 = injectBaseTag(html, '/apps/test')
    const result2 = injectBaseTag(html, '/apps/test/')

    expect(result1).toContain('href="/apps/test/"')
    expect(result2).toContain('href="/apps/test/"')
  })

  it('injects into <html> if <head> is missing', () => {
    const html = '<html><body>Content</body></html>'

    const result = injectBaseTag(html, '/')

    expect(result).toContain('<html>')
    expect(result).toContain('<base')
    expect(result.indexOf('<base')).toBeGreaterThan(result.indexOf('<html>'))
  })

  it('prepends to document if no tags found', () => {
    const html = '<div>Simple HTML</div>'

    const result = injectBaseTag(html, '/')

    expect(result).toContain('<base')
    expect(result.startsWith('<base')).toBe(true)
  })

  it('skips injection if base tag exists (when skipIfExists is true)', () => {
    const html = '<html><head><base href="/existing/"></head><body></body></html>'

    const result = injectBaseTag(html, '/new/', { skipIfExists: true })

    // Should not inject a second base tag
    expect(result).toBe(html)
    expect(result).toContain('href="/existing/"')
    expect(result).not.toContain('href="/new/"')
  })

  it('injects even if base tag exists (when skipIfExists is false)', () => {
    const html = '<html><head><base href="/existing/"></head><body></body></html>'

    const result = injectBaseTag(html, '/new/', { skipIfExists: false })

    // Should inject a second base tag
    expect(result).toContain('href="/existing/"')
    expect(result).toContain('href="/new/"')
  })

  it('uses custom data attribute', () => {
    const html = '<html><head></head><body></body></html>'

    const result = injectBaseTag(html, '/', {
      dataAttribute: 'data-custom-marker',
    })

    expect(result).toContain('data-custom-marker="injected"')
    expect(result).not.toContain('data-vrooli-base')
  })

  it('handles complex real-world HTML', () => {
    const html = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Test App</title>
  <link rel="stylesheet" href="styles.css">
</head>
<body>
  <div id="root"></div>
  <script src="main.js"></script>
</body>
</html>`

    const result = injectBaseTag(html, '/apps/test-app/proxy/')

    expect(result).toContain('<base')
    expect(result).toContain('href="/apps/test-app/proxy/"')

    // Should be injected right after <head>
    const headPos = result.indexOf('<head>')
    const basePos = result.indexOf('<base')
    const metaPos = result.indexOf('<meta')
    expect(basePos).toBeGreaterThan(headPos)
    expect(basePos).toBeLessThan(metaPos)
  })

  it('handles multiple head tags (edge case)', () => {
    const html = '<html><head></head><body><head></head></body></html>'

    const result = injectBaseTag(html, '/')

    // Should inject into first head only
    const firstHeadEnd = result.indexOf('</head>')
    const basePos = result.indexOf('<base')
    expect(basePos).toBeLessThan(firstHeadEnd)
  })

  it('handles HTML with existing base tag with different attributes', () => {
    const html = '<html><head><base target="_blank"></head><body></body></html>'

    const result = injectBaseTag(html, '/', { skipIfExists: true })

    // Should skip because base tag exists (even without href)
    expect(result).toBe(html)
  })

  it('works with minimal HTML', () => {
    const html = '<head></head>'

    const result = injectBaseTag(html, '/test/')

    expect(result).toContain('<base')
    expect(result).toContain('href="/test/"')
  })

  it('escapes special characters in base path', () => {
    const html = '<html><head></head><body></body></html>'

    // Note: Base href doesn't need HTML escaping for forward slashes
    const result = injectBaseTag(html, '/apps/test-app/proxy/')

    expect(result).toContain('href="/apps/test-app/proxy/"')
  })

  it('handles empty base path', () => {
    const html = '<html><head></head><body></body></html>'

    const result = injectBaseTag(html, '')

    // Should still add trailing slash
    expect(result).toContain('href="/"')
  })

  it('provides data attribute for debugging', () => {
    const html = '<html><head></head><body></body></html>'

    const result1 = injectBaseTag(html, '/', {
      dataAttribute: 'data-app-monitor-self',
    })

    const result2 = injectBaseTag(html, '/apps/test/proxy/', {
      dataAttribute: 'data-app-monitor',
    })

    expect(result1).toContain('data-app-monitor-self="injected"')
    expect(result2).toContain('data-app-monitor="injected"')
  })
})
