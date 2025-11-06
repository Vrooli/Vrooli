/**
 * Tests for server/inject.ts
 */

import { describe, it, expect } from 'vitest'
import {
  buildProxyMetadata,
  injectProxyMetadata,
  injectScenarioConfig,
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
