import express, { type Request, type Response, type Router } from 'express'
import * as http from 'node:http'
import type {
  ScenarioProxyHostOptions,
  ScenarioProxyHostController,
  ScenarioProxyAppMetadata,
  PortEntry,
  ProxyInfo,
} from '../shared/types.js'
import {
  LOOPBACK_HOST,
  LOOPBACK_HOSTS,
  DEFAULT_PROXY_TIMEOUT,
} from '../shared/constants.js'
import { buildProxyMetadata, injectProxyMetadata, injectBaseTag } from './inject.js'
import { proxyToApi, proxyWebSocketUpgrade } from './proxy.js'

const DEFAULT_CACHE_TTL_MS = 30_000
const SLUGIFY_REGEX = /[^a-z0-9]+/g

interface ProxyContext {
  metadata: ProxyInfo
  uiPort: number
  apiPort: number | null
  portLookup: Map<string, number>
}

interface CachedContext {
  context: ProxyContext
  timestamp: number
}

function normalizePrefix(prefix: string): string {
  if (!prefix.startsWith('/')) {
    return `/${prefix.replace(/^\/+/, '')}`.replace(/\/+$/, '') || '/apps'
  }
  return prefix.replace(/\/+$/, '') || '/apps'
}

function normalizeSegment(value: string, fallback: string): string {
  const cleaned = value.replace(/^\/+/, '').replace(/\/+$/, '')
  return cleaned || fallback
}

function normalizePortKey(value: unknown): string {
  if (typeof value === 'number') {
    return String(value)
  }
  if (typeof value !== 'string') {
    return ''
  }
  return value.trim().toLowerCase().replace(SLUGIFY_REGEX, '-')
}

function slugify(value: unknown): string {
  if (typeof value !== 'string') {
    return `port-${Math.random().toString(36).slice(2, 8)}`
  }
  const trimmed = value.trim().toLowerCase().replace(SLUGIFY_REGEX, '-')
  return trimmed || `port-${Math.random().toString(36).slice(2, 8)}`
}

function parsePortNumber(value: unknown): number | null {
  if (typeof value === 'number' && Number.isFinite(value) && value > 0) {
    return Math.round(value)
  }
  if (typeof value === 'string') {
    const parsed = Number(value)
    if (Number.isFinite(parsed) && parsed > 0) {
      return Math.round(parsed)
    }
  }
  if (value && typeof value === 'object' && 'port' in value) {
    return parsePortNumber((value as Record<string, unknown>).port)
  }
  return null
}

function isLikelyUiKey(key: string): boolean {
  return /(ui|front|preview|web|vite)/i.test(key)
}

function buildAliases(key: string, slug: string, port: number): string[] {
  const aliases = new Set<string>()
  aliases.add(key)
  aliases.add(key.toLowerCase())
  aliases.add(key.toUpperCase())
  aliases.add(slug)
  aliases.add(slug.replace(/-/g, '_'))
  aliases.add(String(port))
  if (key.toLowerCase().includes('api')) {
    aliases.add('api')
    aliases.add('api-port')
  }
  if (key.toLowerCase().includes('ui')) {
    aliases.add('ui')
    aliases.add('ui-port')
  }
  return Array.from(aliases).filter(Boolean)
}

function extractProxyRelativeUrl(originalUrl: string, proxySegment: string): string {
  const marker = `/${proxySegment}`
  const queryIndex = originalUrl.indexOf('?')
  const search = queryIndex >= 0 ? originalUrl.slice(queryIndex) : ''
  const withoutQuery = queryIndex >= 0 ? originalUrl.slice(0, queryIndex) : originalUrl
  const markerIndex = withoutQuery.lastIndexOf(marker)
  if (markerIndex === -1) {
    return `/${search || ''}`
  }
  const remainder = withoutQuery.slice(markerIndex + marker.length)
  const normalized = remainder && remainder.length > 0 ? (remainder.startsWith('/') ? remainder : `/${remainder}`) : '/'
  return `${normalized || '/'}${search}`
}

function isApiPath(relativeUrl: string): boolean {
  const pathname = (relativeUrl || '/').split('?')[0].replace(/^\/+/, '').toLowerCase()
  return pathname === 'api' || pathname.startsWith('api/')
}

function isHtmlLikeRequest(req: Request, relativeUrl: string): boolean {
  if (req.method !== 'GET') {
    return false
  }
  const acceptHeader = req.headers.accept
  if (typeof acceptHeader === 'string' && acceptHeader.includes('text/html')) {
    return true
  }
  const pathOnly = (relativeUrl || '/').split('?')[0].toLowerCase()
  return pathOnly === '/' || pathOnly.endsWith('.html') || pathOnly.endsWith('.htm')
}

function chooseAppId(appIdParam: string, metadata?: ScenarioProxyAppMetadata): string {
  if (metadata?.id && typeof metadata.id === 'string') {
    return metadata.id
  }
  if (metadata?.appId && typeof metadata.appId === 'string') {
    return metadata.appId
  }
  return appIdParam
}

function deduceScenarioName(metadata?: ScenarioProxyAppMetadata): string | undefined {
  return (
    (metadata?.scenario_name as string | undefined) ||
    (metadata?.scenarioName as string | undefined) ||
    (metadata?.scenario as string | undefined) ||
    (metadata?.name as string | undefined)
  )
}

function readPortMappings(metadata?: ScenarioProxyAppMetadata): Record<string, unknown> {
  if (!metadata) {
    return {}
  }
  if (metadata.port_mappings && typeof metadata.port_mappings === 'object') {
    return metadata.port_mappings as Record<string, unknown>
  }
  if (metadata.portMappings && typeof metadata.portMappings === 'object') {
    return metadata.portMappings as Record<string, unknown>
  }
  return {}
}

function pickPrimaryPort(entries: Array<{ key: string; port: number }>, metadata?: ScenarioProxyAppMetadata): number | null {
  const config = metadata?.config
  if (config && typeof config === 'object') {
    const primary = parsePortNumber((config as Record<string, unknown>).primary_port)
    if (primary) {
      return primary
    }
  }
  for (const entry of entries) {
    if (isLikelyUiKey(entry.key)) {
      return entry.port
    }
  }
  if (entries.length > 0) {
    return entries[0].port
  }
  return null
}

function pickApiPort(entries: Array<{ key: string; port: number }>, metadata?: ScenarioProxyAppMetadata): number | null {
  const config = metadata?.config
  if (config && typeof config === 'object') {
    const api = parsePortNumber((config as Record<string, unknown>).api_port)
    if (api) {
      return api
    }
  }
  for (const entry of entries) {
    if (entry.key.toLowerCase().includes('api')) {
      return entry.port
    }
  }
  return null
}

function buildProxyContext(
  appIdParam: string,
  metadata: ScenarioProxyAppMetadata,
  options: {
    hostScenario: string
    loopbackHosts: string[]
    appsPrefix: string
    proxySegment: string
    portsSegment: string
  }
): ProxyContext {
  const portMappings = readPortMappings(metadata)
  const entries: Array<{ key: string; port: number; normalized: string }> = []

  for (const [key, value] of Object.entries(portMappings)) {
    const port = parsePortNumber(value)
    if (!port) {
      continue
    }
    entries.push({ key, port, normalized: key.toLowerCase() })
  }

  if (entries.length === 0) {
    throw new Error(`App ${appIdParam} has no port mappings`)
  }

  const uiPort = pickPrimaryPort(entries, metadata)
  if (!uiPort) {
    throw new Error(`Unable to determine primary/UI port for ${appIdParam}`)
  }

  const apiPort = pickApiPort(entries, metadata) || uiPort
  const portLookup = new Map<string, number>()
  const ports: PortEntry[] = []
  const resolvedAppId = chooseAppId(appIdParam, metadata)
  const scenarioName = deduceScenarioName(metadata)

  for (const entry of entries) {
    const slug = slugify(entry.key)
    const isPrimary = entry.port === uiPort
    const basePath = isPrimary
      ? `${options.appsPrefix}/${resolvedAppId}/${options.proxySegment}`
      : `${options.appsPrefix}/${resolvedAppId}/${options.portsSegment}/${slug}/${options.proxySegment}`
    const aliases = buildAliases(entry.key, slug, entry.port)
    aliases.forEach((alias) => {
      const normalized = normalizePortKey(alias)
      if (normalized) {
        portLookup.set(normalized, entry.port)
      }
    })

    ports.push({
      appId: resolvedAppId,
      port: entry.port,
      label: entry.key,
      normalizedLabel: entry.normalized,
      slug,
      source: 'port_mappings',
      priority: isPrimary ? 100 : 50,
      isPrimary,
      path: basePath,
      aliases,
      assetNamespace: `${basePath}/assets`,
    })
  }

  const primaryPort = ports.find((entry) => entry.isPrimary) || ports[0]
  if (!primaryPort) {
    throw new Error(`Unable to determine primary port entry for ${appIdParam}`)
  }

  const metadataPayload = buildProxyMetadata({
    appId: resolvedAppId,
    hostScenario: options.hostScenario,
    targetScenario: scenarioName || resolvedAppId,
    ports,
    primaryPort,
    loopbackHosts: options.loopbackHosts,
  })

  return {
    metadata: metadataPayload,
    uiPort,
    apiPort,
    portLookup,
  }
}

async function fetchHtmlFromUi(options: {
  upstreamHost: string
  port: number
  path: string
  headers: http.OutgoingHttpHeaders
  timeout: number
}): Promise<{ status: number; headers: http.IncomingHttpHeaders; body: string }> {
  return new Promise((resolve, reject) => {
    const request = http.request(
      {
        hostname: options.upstreamHost,
        port: options.port,
        path: options.path,
        method: 'GET',
        headers: options.headers,
        timeout: options.timeout,
      },
      (response) => {
        let data = ''
        response.setEncoding('utf8')
        response.on('data', (chunk) => {
          data += chunk
        })
        response.on('end', () => {
          resolve({
            status: response.statusCode || 500,
            headers: response.headers,
            body: data,
          })
        })
      }
    )

    request.on('error', (error) => {
      reject(error)
    })

    request.on('timeout', () => {
      request.destroy(new Error('Timed out fetching proxied HTML'))
    })

    request.end()
  })
}

function forwardHttpRequest(
  req: Request,
  res: Response,
  relativeUrl: string,
  targetPort: number,
  options: { upstreamHost: string; timeout: number; verbose: boolean }
): Promise<void> {
  const normalizedUrl = relativeUrl.startsWith('/') ? relativeUrl : `/${relativeUrl}`
  const proxyReq = Object.create(req)
  proxyReq.url = normalizedUrl
  return proxyToApi(proxyReq, res, normalizedUrl, {
    apiPort: targetPort,
    apiHost: options.upstreamHost,
    timeout: options.timeout,
    verbose: options.verbose,
  })
}

function cloneHeaders(headers: http.IncomingHttpHeaders): Record<string, string | string[]> {
  const result: Record<string, string | string[]> = {}
  for (const [key, value] of Object.entries(headers)) {
    if (value === undefined) {
      continue
    }
    if (key.toLowerCase() === 'content-length') {
      continue
    }
    result[key] = value
  }
  return result
}

export function createScenarioProxyHost(options: ScenarioProxyHostOptions): ScenarioProxyHostController {
  if (!options.hostScenario) {
    throw new Error('hostScenario option is required')
  }
  if (typeof options.fetchAppMetadata !== 'function') {
    throw new Error('fetchAppMetadata option is required')
  }

  const appsPrefix = normalizePrefix(options.appsPathPrefix ?? '/apps')
  const proxySegment = normalizeSegment(options.proxyPathSegment ?? 'proxy', 'proxy')
  const portsSegment = normalizeSegment(options.portsPathSegment ?? 'ports', 'ports')
  const loopbackHosts = options.loopbackHosts && options.loopbackHosts.length > 0
    ? options.loopbackHosts
    : Array.from(LOOPBACK_HOSTS)
  const router: Router = express.Router()
  const cache = new Map<string, CachedContext>()
  const cacheTtl = options.cacheTtlMs ?? DEFAULT_CACHE_TTL_MS
  const upstreamHost = options.upstreamHost ?? LOOPBACK_HOST
  const timeout = options.timeoutMs ?? DEFAULT_PROXY_TIMEOUT
  const verbose = Boolean(options.verbose)
  const patchFetch = options.patchFetch ?? false
  const childBaseTagAttr = options.childBaseTagAttribute || 'data-proxy-host'
  const proxyHeaderName = options.proxiedAppHeader || 'x-vrooli-proxied-app'

  async function getProxyContext(appId: string): Promise<ProxyContext> {
    const now = Date.now()
    const cached = cache.get(appId)
    if (cached && now - cached.timestamp < cacheTtl) {
      return cached.context
    }

    const metadata = await options.fetchAppMetadata(appId)
    if (!metadata) {
      throw new Error(`App ${appId} not found`)
    }

    const context = buildProxyContext(appId, metadata, {
      hostScenario: options.hostScenario,
      loopbackHosts,
      appsPrefix,
      proxySegment,
      portsSegment,
    })

    cache.set(appId, { context, timestamp: now })
    return context
  }

  async function handleScenarioProxyRequest(req: Request, res: Response): Promise<void> {
    const appId = req.params.appId
    const relativeUrl = extractProxyRelativeUrl(req.originalUrl || req.url || '/', proxySegment)
    if (verbose) {
      console.log(`[proxy-host] ${appId} -> ${relativeUrl}`)
    }

    try {
      const context = await getProxyContext(appId)

      if (isApiPath(relativeUrl)) {
        await forwardHttpRequest(req, res, relativeUrl, context.apiPort || context.uiPort, {
          upstreamHost,
          timeout,
          verbose,
        })
        return
      }

      if (isHtmlLikeRequest(req, relativeUrl)) {
        const targetUrl = relativeUrl.startsWith('/') ? relativeUrl : `/${relativeUrl}`
        const htmlResponse = await fetchHtmlFromUi({
          upstreamHost,
          port: context.uiPort,
          path: targetUrl,
          headers: {
            ...req.headers,
            host: `${upstreamHost}:${context.uiPort}`,
          },
          timeout,
        })

        let body = htmlResponse.body
        const contentType = htmlResponse.headers['content-type'] || 'text/html'

        if (typeof body === 'string' && contentType.includes('text/html')) {
          const metadataPayload = { ...context.metadata, generatedAt: Date.now() }
          body = injectProxyMetadata(body, metadataPayload, { patchFetch })
          const basePath = `${context.metadata.primary.path || `${appsPrefix}/${appId}/${proxySegment}`}/`
          body = injectBaseTag(body, basePath, {
            skipIfExists: true,
            dataAttribute: childBaseTagAttr,
          })
        }

        const headerEntries = cloneHeaders(htmlResponse.headers)
        for (const [key, value] of Object.entries(headerEntries)) {
          res.setHeader(key, value as string)
        }

        if (proxyHeaderName) {
          res.setHeader(proxyHeaderName, appId)
        }
        res.setHeader('Cache-Control', 'no-store, no-cache, must-revalidate, max-age=0')
        res.status(htmlResponse.status).send(body)
        return
      }

      await forwardHttpRequest(req, res, relativeUrl, context.uiPort, {
        upstreamHost,
        timeout,
        verbose,
      })
    } catch (error) {
      console.error(`[proxy-host] Failed to proxy app ${appId}:`, error instanceof Error ? error.message : error)
      if (!res.headersSent) {
        res.status(502).json({ error: 'Failed to proxy application', details: error instanceof Error ? error.message : 'Unknown error' })
      }
    }
  }

  async function handlePortProxyRequest(req: Request, res: Response): Promise<void> {
    const appId = req.params.appId
    const portKey = req.params.portKey
    const relativeUrl = extractProxyRelativeUrl(req.originalUrl || req.url || '/', proxySegment)
    if (verbose) {
      console.log(`[proxy-host] ${appId}/${portKey} -> ${relativeUrl}`)
    }

    try {
      const context = await getProxyContext(appId)
      const targetPort = context.portLookup.get(normalizePortKey(portKey))
      if (!targetPort) {
        res.status(404).json({ error: `Port ${portKey} not found for ${appId}` })
        return
      }

      await forwardHttpRequest(req, res, relativeUrl, targetPort, {
        upstreamHost,
        timeout,
        verbose,
      })
    } catch (error) {
      console.error(`[proxy-host] Failed to proxy port ${appId}/${portKey}:`, error instanceof Error ? error.message : error)
      if (!res.headersSent) {
        res.status(502).json({ error: 'Failed to proxy port', details: error instanceof Error ? error.message : 'Unknown error' })
      }
    }
  }

  function registerRoutes(): void {
    const portRoute = `${appsPrefix}/:appId/${portsSegment}/:portKey/${proxySegment}`
    const appRoute = `${appsPrefix}/:appId/${proxySegment}`

    router.all(`${portRoute}/*`, (req, res) => {
      void handlePortProxyRequest(req, res)
    })
    router.all(portRoute, (req, res) => {
      void handlePortProxyRequest(req, res)
    })

    router.all(`${appRoute}/*`, (req, res) => {
      void handleScenarioProxyRequest(req, res)
    })
    router.all(appRoute, (req, res) => {
      void handleScenarioProxyRequest(req, res)
    })
  }

  registerRoutes()

  async function handleUpgrade(req: http.IncomingMessage, socket: any, head: Buffer): Promise<boolean> {
    if (!req.url) {
      return false
    }

    try {
      const url = new URL(req.url, `http://${req.headers.host || 'localhost'}`)
      const pathname = url.pathname
      const search = url.search || ''

      if (!pathname.startsWith(`${appsPrefix}/`)) {
        return false
      }

      const remainder = pathname.slice(appsPrefix.length + 1)
      const segments = remainder.split('/')

      // /apps/:appId/ports/:portKey/proxy/... pattern
      if (segments.length >= 4 && segments[1] === portsSegment && segments[3] === proxySegment) {
        const [appId, , portKey, , ...rest] = segments
        const relativePath = rest.length > 0 ? `/${rest.join('/')}${search}` : `/${search || ''}`
        const context = await getProxyContext(decodeURIComponent(appId))
        const targetPort = context.portLookup.get(normalizePortKey(portKey))
        if (!targetPort) {
          socket.destroy()
          return true
        }
        const proxyReq = Object.create(req)
        proxyReq.url = relativePath
        proxyWebSocketUpgrade(proxyReq, socket, head, {
          apiPort: targetPort,
          apiHost: upstreamHost,
          verbose,
        })
        return true
      }

      // /apps/:appId/proxy/... pattern
      if (segments.length >= 2 && segments[1] === proxySegment) {
        const [appId, , ...rest] = segments
        const relativePath = rest.length > 0 ? `/${rest.join('/')}${search}` : `/${search || ''}`
        const context = await getProxyContext(decodeURIComponent(appId))
        const targetPort = isApiPath(relativePath) ? context.apiPort || context.uiPort : context.uiPort
        const proxyReq = Object.create(req)
        proxyReq.url = relativePath
        proxyWebSocketUpgrade(proxyReq, socket, head, {
          apiPort: targetPort,
          apiHost: upstreamHost,
          verbose,
        })
        return true
      }
    } catch (error) {
      console.error('[proxy-host] WebSocket upgrade failed:', error instanceof Error ? error.message : error)
      socket.destroy()
      return true
    }

    return false
  }

  function invalidate(appId?: string): void {
    if (appId) {
      cache.delete(appId)
    } else {
      cache.clear()
    }
  }

  return {
    router,
    handleUpgrade,
    invalidate,
    clearCache: () => cache.clear(),
  }
}
