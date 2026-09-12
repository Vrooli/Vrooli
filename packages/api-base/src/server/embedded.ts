/**
 * Built-in embedded scenario proxy
 *
 * Provides a reusable Express router that resolves scenario ports and proxies
 * requests to embedded scenario UIs. Scenarios opt-in via the `embeddedProxy`
 * option in ServerTemplateOptions.
 */

import * as http from 'node:http'
import { execSync } from 'node:child_process'
import { Router, type Request, type Response } from 'express'

/**
 * Options for the embedded scenario proxy
 */
export interface EmbeddedProxyOptions {
  /** Enable/disable the proxy (default: true when object provided) */
  enabled?: boolean
  /** Restrict to specific scenario names (default: allow all) */
  allowedScenarios?: string[]
  /** TTL for port resolution cache in ms (default: 30000) */
  cacheTtlMs?: number
  /** Timeout for upstream proxy requests in ms (default: 30000) */
  timeoutMs?: number
  /** Upstream host where scenarios run (default: '127.0.0.1') */
  upstreamHost?: string
}

interface CacheEntry {
  port: string
  resolvedAt: number
}

/**
 * Create a port resolver with TTL caching.
 *
 * Looks up the UI port for a scenario by:
 * 1. Checking environment variable (e.g. SCENARIO_NAME_UI_PORT)
 * 2. Running `vrooli scenario port <name> UI_PORT`
 *
 * Results are cached for `cacheTtlMs`. Negative lookups are NOT cached
 * so retries happen immediately when a scenario starts up.
 */
export function createPortResolver(cacheTtlMs = 30_000) {
  const cache = new Map<string, CacheEntry>()

  return (scenarioName: string): string | null => {
    const now = Date.now()
    const cached = cache.get(scenarioName)
    if (cached && now - cached.resolvedAt < cacheTtlMs) {
      return cached.port
    }

    // Try environment variable first
    const envVar = `${scenarioName.replace(/-/g, '_').toUpperCase()}_UI_PORT`
    const envPort = process.env[envVar]
    if (envPort) {
      cache.set(scenarioName, { port: envPort, resolvedAt: now })
      return envPort
    }

    // Fall back to vrooli CLI
    try {
      const output = execSync(`vrooli scenario port ${scenarioName} UI_PORT`, {
        encoding: 'utf-8',
        stdio: ['ignore', 'pipe', 'ignore'],
      }).trim()
      if (output) {
        cache.set(scenarioName, { port: output, resolvedAt: now })
        return output
      }
    } catch {
      // Scenario not running or vrooli not available
    }

    return null
  }
}

/**
 * Resolve the external URL for a scenario based on the request context.
 *
 * - localhost requests → http://localhost:{port}
 * - tunnel/remote requests → swap first subdomain to scenario name
 *
 * Scenarios whose API is written in Go use the peer implementation in
 * packages/api-core/discovery (BrowserURLForHost / ResolveExternalURL), whose
 * suite transcribes the cases below. The two must give the same answer: a link
 * that resolves one way through an Express scenario and another through a Go
 * one is worse than either rule alone, so a change here belongs in both.
 */
export function resolveExternalUrl(
  req: Request,
  scenarioName: string,
  resolvedPort: string,
): string {
  const hostHeader = req.headers.host || 'localhost'
  const hostname = hostHeader.split(':')[0]!
  const isLocal = hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '0.0.0.0'

  if (isLocal) {
    return `http://localhost:${resolvedPort}`
  }

  // Tunnel/remote: swap first subdomain
  const parts = hostname.split('.')
  if (parts.length >= 3) {
    parts[0] = scenarioName
    return `https://${parts.join('.')}`
  }

  // Fallback — just use localhost
  return `http://localhost:${resolvedPort}`
}

/**
 * Create an Express router that handles embedded scenario proxying.
 *
 * Routes:
 *   GET  /:scenario/external-url — returns context-aware URL for "open in new tab"
 *   USE  /:scenario              — strips prefix, proxies to localhost:{port}
 */
export function createEmbeddedProxyRouter(options: EmbeddedProxyOptions = {}): Router {
  const {
    allowedScenarios,
    cacheTtlMs = 30_000,
    timeoutMs = 30_000,
    upstreamHost = '127.0.0.1',
  } = options

  const resolvePort = createPortResolver(cacheTtlMs)
  const router = Router()

  // Gate helper — returns port or sends error response
  function gateScenario(req: Request, res: Response): { scenarioName: string; port: string } | null {
    const scenarioName = req.params.scenario
    if (!scenarioName) {
      res.status(400).json({ error: 'Missing scenario name' })
      return null
    }

    if (allowedScenarios && !allowedScenarios.includes(scenarioName)) {
      res.status(403).json({ error: 'Scenario not allowed', scenario: scenarioName })
      return null
    }

    const port = resolvePort(scenarioName)
    if (!port) {
      res.status(503).json({
        error: 'Scenario not available',
        scenario: scenarioName,
        detail: 'Could not resolve UI port for this scenario',
      })
      return null
    }

    return { scenarioName, port }
  }

  // GET /:scenario/external-url
  router.get('/:scenario/external-url', (req: Request, res: Response) => {
    const gate = gateScenario(req, res)
    if (!gate) return

    const url = resolveExternalUrl(req, gate.scenarioName, gate.port)
    res.json({ url, scenario: gate.scenarioName })
  })

  // Proxy all other requests: /:scenario/*
  router.use('/:scenario', (req: Request, res: Response) => {
    const gate = gateScenario(req, res)
    if (!gate) return

    const { scenarioName, port } = gate
    const target = `http://${upstreamHost}:${port}`
    // Strip /embedded/{scenario} prefix from the original URL
    const rewrittenPath = req.originalUrl.replace(new RegExp(`^/embedded/${scenarioName}`), '') || '/'

    let url: URL
    try {
      url = new URL(rewrittenPath, target)
    } catch (e) {
      res.status(400).json({
        error: 'Invalid URL',
        detail: e instanceof Error ? e.message : String(e),
      })
      return
    }

    const proxyReq = http.request(
      url,
      {
        method: req.method,
        headers: {
          ...req.headers,
          host: `${upstreamHost}:${port}`,
        },
      },
      (proxyRes) => {
        res.writeHead(proxyRes.statusCode || 502, proxyRes.headers)
        proxyRes.pipe(res, { end: true })
      },
    )

    proxyReq.on('error', (error) => {
      console.error(`[embedded-proxy] Error proxying to ${scenarioName}:`, error.message)
      if (!res.headersSent) {
        res.status(502).json({
          error: 'Failed to proxy to scenario UI',
          scenario: scenarioName,
          detail: error.message,
          target,
        })
      } else {
        res.end()
      }
    })

    proxyReq.setTimeout(timeoutMs, () => {
      proxyReq.destroy()
      if (!res.headersSent) {
        res.status(504).json({
          error: 'Proxy timeout',
          scenario: scenarioName,
          detail: 'Request to scenario UI timed out',
        })
      }
    })

    if (req.readable) {
      req.pipe(proxyReq, { end: true })
    } else {
      proxyReq.end()
    }
  })

  return router
}
