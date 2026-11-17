/**
 * Built-in embedded scenario proxy
 *
 * Provides a reusable Express router that resolves scenario ports and proxies
 * requests to embedded scenario UIs. Scenarios opt-in via the `embeddedProxy`
 * option in ServerTemplateOptions.
 */

import * as http from 'node:http'
import * as net from 'node:net'
import { execFileSync } from 'node:child_process'
import * as fs from 'node:fs'
import * as os from 'node:os'
import * as path from 'node:path'
import { Router, type Request, type Response } from 'express'
import { isCorsResponseHeader, withoutBrowserContext } from './origin-policy.js'

/**
 * Options for the embedded scenario proxy
 */
export interface EmbeddedProxyOptions {
  /** Enable/disable the proxy (default: true when object provided) */
  enabled?: boolean
  /** Restrict to specific scenario names (default: allow all) */
  allowedScenarios?: string[]
  /** TTL for port resolution cache in ms (default: 2000, matching api-core discovery) */
  cacheTtlMs?: number
  /** Timeout for upstream proxy requests in ms (default: 30000) */
  timeoutMs?: number
  /** Upstream host where scenarios run (default: '127.0.0.1') */
  upstreamHost?: string
}

interface CacheEntry {
  resolution: PortResolution
  resolvedAt: number
}

const DEFAULT_NEGATIVE_CACHE_TTL_MS = 500

export type PortResolution =
  | { port: string; source: 'peer' | 'cli' }
  | { port: null; reason: 'runtime-required' | 'not-found' | 'invalid-target' | 'cli-failed'; detail?: string }
export type PortResolutionFailure = Extract<PortResolution, { port: null }>

export interface PortResolverDependencies {
  /** Explicit runtime-home seam for tests and embedded callers. */
  runtimeHome?: () => string
  /** OS home seam used only when VROOLI_HOME is not configured. */
  homeDir?: () => string
  /** Replaces the canonical CLI recovery rung when embedding or testing. */
  cliLookup?: (scenarioName: string, portKey: string) => PortResolution
}

function normalizePortKey(portKey: string): string {
  const key = portKey.trim().toLowerCase().replace(/_port$/, '')
  return key === 'api' || key === 'ui' ? key : portKey.trim()
}

function validPort(value: unknown): string | null {
  const port = typeof value === 'number' ? value : typeof value === 'string' && /^\d+$/.test(value) ? Number(value) : NaN
  return Number.isInteger(port) && port >= 1 && port <= 65535 ? String(port) : null
}

function peerPort(scenarioName: string, portKey: string, runtimeHome: string): string | null {
  const target = scenarioName.split('@', 1)[0]!
  const variant = scenarioName.includes('@') ? scenarioName.slice(scenarioName.indexOf('@') + 1) : 'live'
  if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(target) || !/^[a-z0-9][a-z0-9-]*$/.test(variant)) return null
  try {
    const peerSlug = variant === 'live' ? target : `${target}@${variant}`
    const recordPath = path.join(runtimeHome, 'peers', `${peerSlug}.json`)
    const record = JSON.parse(fs.readFileSync(recordPath, 'utf8')) as {
      schema_version?: number; scenario?: string; instance?: string; owner_pid?: number
      version?: string; provides?: string[] | null; ports?: Record<string, unknown>
    }
    const recordVariant = record.instance || 'live'
    if (record.schema_version !== 1 || record.scenario !== target || recordVariant !== variant ||
      !Number.isInteger(record.owner_pid) || record.owner_pid! <= 0) return null
    try { process.kill(record.owner_pid!, 0) } catch (error) {
      if ((error as NodeJS.ErrnoException).code !== 'EPERM') return null
    }
    const key = normalizePortKey(portKey)
    if (record.provides != null && !record.provides.includes(key)) return null
    const publishedPort = record.ports?.[key]
    // Peer records are lifecycle-owned JSON with numeric port claims. Keep
    // this stricter than CLI payload parsing, which may serialize numbers as strings.
    if (typeof publishedPort !== 'number') return null
    return validPort(publishedPort)
  } catch {
    return null
  }
}

/** Resolve using the lifecycle peer record before a bounded argv-based CLI recovery lookup. */
function validTarget(scenarioName: string): boolean {
  const [scenario, variant, extra] = scenarioName.split('@')
  return extra === undefined && /^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(scenario ?? '') &&
    (variant === undefined || /^[a-z0-9][a-z0-9-]*$/.test(variant))
}

/** Convert a CLI error code into the resolver's retry class; unknown and legacy codes stay transient. */
export function classifyPortLookupFailureCode(code: unknown, detail?: unknown): PortResolution {
  const message = typeof detail === 'string' ? detail : 'Scenario port lookup failed'
  return code === 'scenario_not_running'
    ? { port: null, reason: 'not-found', detail: message }
    : { port: null, reason: 'cli-failed', detail: message }
}

/** Parse the canonical CLI's JSON result without inferring failure class from free text. */
export function parseScenarioPortCLIResponse(output: string): PortResolution {
  let payload: { success?: boolean; port?: unknown; error?: unknown; error_code?: unknown }
  try {
    payload = JSON.parse(output) as typeof payload
  } catch {
    return { port: null, reason: 'cli-failed', detail: 'CLI returned invalid JSON' }
  }
  if (payload.success === true) {
    const port = typeof payload.port === 'number' ? validPort(payload.port) : null
    return port ? { port, source: 'cli' } : { port: null, reason: 'cli-failed', detail: 'CLI returned an invalid port' }
  }
  return classifyPortLookupFailureCode(payload.error_code, payload.error)
}

function lookupPortWithCli(scenarioName: string, portKey: string): PortResolution {
  try {
    const output = execFileSync(process.env.VROOLI_BIN || 'vrooli', ['scenario', 'port', scenarioName, portKey, '--json'], {
      encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'], timeout: 5000, windowsHide: true,
    }).trim()
    return parseScenarioPortCLIResponse(output)
  } catch (error) {
    const failure = error as NodeJS.ErrnoException & { stdout?: Buffer | string; stderr?: Buffer | string }
    const detail = [failure.stdout, failure.stderr, failure.message].map((part) => String(part ?? '').trim()).filter(Boolean).join(': ')
    const stdout = failure.stdout?.toString().trim() ?? ''
    try {
      const parsed = parseScenarioPortCLIResponse(stdout)
      if ('reason' in parsed && parsed.reason === 'not-found') return parsed
    } catch { /* Non-JSON failure is a CLI/runtime transport failure. */ }
    if (process.env.VROOLI_DESKTOP_MODE === 'true') return { port: null, reason: 'runtime-required', detail }
    return { port: null, reason: 'cli-failed', detail }
  }
}

/** Resolve via lifecycle peers locally and delegate recovery to the canonical CLI.
 * In desktop mode the CLI owns authenticated runtime IPC discovery; host peer
 * files are deliberately bypassed.
 */
export function resolveScenarioPort(scenarioName: string, portKey = 'UI_PORT', dependencies: PortResolverDependencies = {}): PortResolution {
  if (!validTarget(scenarioName)) return { port: null, reason: 'invalid-target', detail: 'Expected scenario or scenario@variant slug' }

  const desktopMode = process.env.VROOLI_DESKTOP_MODE === 'true'
  if (!desktopMode) {
    const homeDir = dependencies.homeDir ?? os.homedir
    const runtimeHome = dependencies.runtimeHome?.() || process.env.VROOLI_HOME?.trim() || path.join(homeDir(), '.vrooli')
    const localPort = peerPort(scenarioName, portKey, runtimeHome)
    if (localPort) return { port: localPort, source: 'peer' }

  }

  return (dependencies.cliLookup ?? lookupPortWithCli)(scenarioName, portKey)
}


/** Structured resolver keeps failure classes available to server APIs. */
export function createPortResolutionResolver(cacheTtlMs = 2_000, portKey = 'UI_PORT', dependencies: PortResolverDependencies = {}) {
  const cache = new Map<string, CacheEntry>()
  return (scenarioName: string): PortResolution => {
    const now = Date.now()
    const cached = cache.get(scenarioName)
    if (cached) {
      const ttl = cached.resolution.port ? cacheTtlMs : DEFAULT_NEGATIVE_CACHE_TTL_MS
      if (now - cached.resolvedAt < ttl) return cached.resolution
    }
    const result = resolveScenarioPort(scenarioName, portKey, dependencies)
    // Cache only a definite missing-target result. Invalid targets, required
    // desktop IPC, and transient CLI/runtime errors must be retried immediately.
    if (result.port || ('reason' in result && result.reason === 'not-found')) {
      cache.set(scenarioName, { resolution: result, resolvedAt: now })
    }
    return result
  }
}

/**
 * Create a port resolver with TTL caching.
 *
 * Looks up a scenario port from the lifecycle peer record first, then runs
 * `vrooli scenario port <name> <portKey> --json` as bounded recovery.
 *
 * Successful results are cached for `cacheTtlMs` (2 seconds by default).
 * Definite missing-target results use a 500ms negative cache to limit repeated
 * CLI recovery while allowing newly started scenarios to be found promptly.
 * Invalid targets, required desktop IPC, and transient recovery failures are
 * never cached.
 */
export function createPortResolver(cacheTtlMs = 2_000, portKey = 'UI_PORT', dependencies: PortResolverDependencies = {}) {
  const resolve = createPortResolutionResolver(cacheTtlMs, portKey, dependencies)
  return (scenarioName: string): string | null => resolve(scenarioName).port
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
  const forwardedHost = req.headers['x-forwarded-host']
  const hostHeader = (Array.isArray(forwardedHost) ? forwardedHost[0] : forwardedHost)?.split(',')[0]?.trim() || req.headers.host || ''
  const hostname = hostWithoutPort(hostHeader)
  const isLocal = hostname === 'localhost' || hostname === '0.0.0.0' || (net.isIP(hostname) === 4 && isIPInRanges(hostname, ['127.0.0.0/8'])) ||
    (net.isIP(hostname) === 6 && isIPInRanges(hostname, ['::1/128']))

  if (isLocal) {
    return `http://localhost:${resolvedPort}`
  }

  if (net.isIP(hostname) !== 0 && isIPInRanges(hostname, ['10.0.0.0/8', '172.16.0.0/12', '192.168.0.0/16', 'fc00::/7'])) {
    return `http://${net.isIP(hostname) === 6 ? `[${hostname}]` : hostname}:${resolvedPort}`
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

function hostWithoutPort(value: string): string {
  const host = value.trim()
  if (host.startsWith('[')) {
    const end = host.indexOf(']')
    return end > 0 ? host.slice(1, end) : host
  }
  const firstColon = host.indexOf(':')
  return firstColon >= 0 && firstColon === host.lastIndexOf(':') ? host.slice(0, firstColon) : host
}

function isIPInRanges(address: string, ranges: string[]): boolean {
  const blockList = new net.BlockList()
  for (const range of ranges) {
    const [subnet, mask] = range.split('/')
    blockList.addSubnet(subnet!, Number(mask), net.isIP(subnet!) === 6 ? 'ipv6' : 'ipv4')
  }
  return blockList.check(address, net.isIP(address) === 6 ? 'ipv6' : 'ipv4')
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
    cacheTtlMs = 2_000,
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

    // The embedded scenario is a different origin; this server's origin
    // policy already admitted the request, so the browser context is dropped.
    const headers: http.OutgoingHttpHeaders = {
      ...withoutBrowserContext(req.headers),
      host: `${upstreamHost}:${port}`,
    }
    let parsedBody: Buffer | undefined
    const hasBody = Number(req.headers['content-length']) > 0 || req.headers['transfer-encoding'] !== undefined
    if (!req.readable && hasBody) {
      // A preceding Express parser consumed the stream. Reframe its retained
      // representation before opening the upstream request: the original length
      // and compression describe bytes that are no longer available.
      try {
        const mediaType = req.headers['content-type']?.split(';')[0]?.trim()
        if (Buffer.isBuffer(req.body)) {
          parsedBody = req.body
        } else if (req.body !== undefined && req.is(['json', 'application/*+json'])) {
          const json = JSON.stringify(req.body)
          if (json === undefined) throw new Error('JSON body is unavailable')
          parsedBody = Buffer.from(json, 'utf8')
          headers['content-type'] = `${mediaType}; charset=utf-8`
        } else if (typeof req.body === 'string') {
          parsedBody = Buffer.from(req.body, 'utf8')
          headers['content-type'] = `${mediaType || 'text/plain'}; charset=utf-8`
        } else {
          throw new Error('Consumed body has no supported representation')
        }
      } catch {
        res.status(400).json({ error: 'Cannot forward consumed request body', scenario: scenarioName })
        return
      }
      headers['content-length'] = parsedBody.length
      delete headers['transfer-encoding']
      delete headers['content-encoding']
    }

    const proxyReq = http.request(
      url,
      {
        method: req.method,
        headers,
      },
      (proxyRes) => {
        // CORS headers already written by this server's origin policy win;
        // the embedded scenario's answer describes its own origin.
        const responseHeaders: http.OutgoingHttpHeaders = {}
        for (const [key, value] of Object.entries(proxyRes.headers)) {
          if (value !== undefined && !isCorsResponseHeader(key)) responseHeaders[key] = value
        }
        res.writeHead(proxyRes.statusCode || 502, responseHeaders)
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

    if (parsedBody !== undefined) {
      proxyReq.end(parsedBody)
    } else if (req.readable) {
      req.pipe(proxyReq, { end: true })
    } else {
      proxyReq.end()
    }
  })

  return router
}
