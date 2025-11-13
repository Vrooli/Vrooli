import express, { type Request, type Response, type Router } from 'express'
import * as http from 'node:http'
import * as net from 'node:net'
import type {
  ScenarioProxyHostOptions,
  ScenarioProxyHostController,
  ScenarioProxyAppMetadata,
  PortEntry,
  ProxyInfo,
  HostEndpointDefinition,
} from '../shared/types.js'
import {
  LOOPBACK_HOST,
  LOOPBACK_HOSTS,
  DEFAULT_PROXY_TIMEOUT,
} from '../shared/constants.js'
import { buildProxyMetadata, injectProxyMetadata, injectBaseTag } from './inject.js'
import { proxyToApi, proxyWebSocketUpgrade } from './proxy.js'
import { resolveProxyAgent } from './agent.js'

const DEFAULT_CACHE_TTL_MS = 30_000
const UPSTREAM_CHECK_TIMEOUT_MS = 500
const SLUGIFY_REGEX = /[^a-z0-9]+/g
const PATH_TRAVERSAL_PATTERN = /(^|\/|\\)\.\.(?=\/|\\|$)/

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

interface HtmlCacheEntry {
  status: number
  headers: http.IncomingHttpHeaders
  body: string
  injected: boolean
  timestamp: number
  insertedAt: number
}

interface HtmlCachePointer {
  appId: string
  path: string
  insertedAt: number
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
  const normalized = (
    remainder && remainder.length > 0
      ? (remainder.startsWith('/') ? remainder : `/${remainder}`)
      : '/'
  )
  return `${normalized || '/'}${search}`
}

function hasPathTraversal(value: string | undefined | null): boolean {
  if (!value) {
    return false
  }
  let decoded = value
  try {
    decoded = decodeURIComponent(value)
  } catch {
    decoded = value
  }
  let normalized = decoded.split('\\').join('/')
  while (normalized.includes('//')) {
    normalized = normalized.replace('//', '/')
  }
  return PATH_TRAVERSAL_PATTERN.test(normalized)
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

function normalizeHostEndpoints(entries?: HostEndpointDefinition[]): HostEndpointDefinition[] {
  if (!Array.isArray(entries)) {
    return []
  }

  const seen = new Set<string>()
  const normalized: HostEndpointDefinition[] = []

  for (const entry of entries) {
    if (!entry || typeof entry.path !== 'string') {
      continue
    }
    const trimmedPath = entry.path.trim()
    if (!trimmedPath) {
      continue
    }
    const normalizedPath = trimmedPath.startsWith('/')
      ? trimmedPath
      : `/${trimmedPath.replace(/^\/+/, '')}`
    const method = typeof entry.method === 'string' && entry.method.trim()
      ? entry.method.trim().toUpperCase()
      : undefined
    const key = `${method || 'ANY'} ${normalizedPath}`
    if (seen.has(key)) {
      continue
    }
    seen.add(key)
    normalized.push({ path: normalizedPath, method })
  }

  return normalized
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
    hostEndpoints: HostEndpointDefinition[]
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
    hostEndpoints: options.hostEndpoints,
  })

  return {
    metadata: metadataPayload,
    uiPort,
    apiPort,
    portLookup,
  }
}

const HEAD_CLOSE_TAG = '</head>'
const HEAD_BUFFER_LIMIT = 512 * 1024 // 512KB safety net

class StreamingHeadInjector {
  private buffer = ''
  private injected = false

  constructor(private readonly options: {
    metadata: ProxyInfo
    patchFetch: boolean
    basePath: string
    baseTagAttribute: string
  }) {}

  process(chunk: string): string {
    if (this.injected) {
      return chunk
    }

    this.buffer += chunk
    const lower = this.buffer.toLowerCase()
    const closeIndex = lower.indexOf(HEAD_CLOSE_TAG)

    if (closeIndex === -1) {
      if (this.buffer.length > HEAD_BUFFER_LIMIT) {
        return this.flush()
      }
      return ''
    }

    const endIndex = closeIndex + HEAD_CLOSE_TAG.length
    const headPortion = this.buffer.slice(0, endIndex)
    const remainder = this.buffer.slice(endIndex)

    const transformedHead = this.applyInjections(headPortion)
    this.injected = true
    this.buffer = ''

    return transformedHead + remainder
  }

  flush(): string {
    if (this.injected) {
      return ''
    }

    const leftover = this.buffer
    this.buffer = ''
    this.injected = true
    if (!leftover) {
      return ''
    }

    return this.applyInjections(leftover)
  }

  private applyInjections(html: string): string {
    const { metadata, patchFetch, basePath, baseTagAttribute } = this.options

    let output = injectProxyMetadata(html, metadata, { patchFetch })
    output = injectBaseTag(output, basePath, {
      skipIfExists: true,
      dataAttribute: baseTagAttribute,
    })
    return output
  }
}

async function streamProxiedHtml(options: {
  appId: string
  targetUrl: string
  res: Response
  upstreamHost: string
  upstreamPort: number
  requestHeaders: http.IncomingHttpHeaders
  timeout: number
  metadata: ProxyInfo
  basePath: string
  patchFetch: boolean
  baseTagAttribute: string
  proxyHeaderName?: string
  upstreamAgent?: http.Agent
  shouldCache: (response: { status: number; headers: http.IncomingHttpHeaders }) => boolean
  onCacheStore: (payload: {
    status: number
    headers: http.IncomingHttpHeaders
    body: string
  }) => void
  onCacheSkip: () => void
}): Promise<void> {
  return await new Promise((resolve, reject) => {
    let settled = false
    const fail = (error: Error) => {
      if (settled) {
        return
      }
      settled = true
      if (!options.res.headersSent) {
        options.res.status(502).json({
          error: 'Failed to proxy application',
          details: error.message,
        })
      } else {
        options.res.destroy(error)
      }
      options.onCacheSkip()
      reject(error)
    }

    const request = http.request(
      {
        hostname: options.upstreamHost,
        port: options.upstreamPort,
        path: options.targetUrl,
        method: 'GET',
        headers: {
          ...options.requestHeaders,
          host: `${options.upstreamHost}:${options.upstreamPort}`,
        },
        timeout: options.timeout,
        agent: options.upstreamAgent,
      },
      (upstreamRes) => {
        const status = upstreamRes.statusCode || 500
        const headers = upstreamRes.headers
        const injector = new StreamingHeadInjector({
          metadata: options.metadata,
          patchFetch: options.patchFetch,
          basePath: options.basePath,
          baseTagAttribute: options.baseTagAttribute,
        })

        const cacheable = options.shouldCache({ status, headers })
        const cachedChunks: string[] = []

        options.res.status(status)
        const headerEntries = cloneHeaders(headers)
        for (const [key, value] of Object.entries(headerEntries)) {
          options.res.setHeader(key, value as string)
        }

        if (options.proxyHeaderName) {
          options.res.setHeader(options.proxyHeaderName, options.appId)
        }

        options.res.setHeader('Cache-Control', 'no-store, no-cache, must-revalidate, max-age=0')

        upstreamRes.setEncoding('utf8')

        upstreamRes.on('data', (chunk: string) => {
          const output = injector.process(chunk)
          if (!output) {
            return
          }
          options.res.write(output)
          if (cacheable) {
            cachedChunks.push(output)
          }
        })

        upstreamRes.on('end', () => {
          const tail = injector.flush()
          if (tail) {
            options.res.write(tail)
            if (cacheable) {
              cachedChunks.push(tail)
            }
          }
          options.res.end()

          if (cacheable) {
            options.onCacheStore({
              status,
              headers: copyIncomingHeaders(headers),
              body: cachedChunks.join(''),
            })
          } else {
            options.onCacheSkip()
          }

          settled = true
          resolve()
        })

        upstreamRes.on('error', (error: Error) => {
          fail(error)
        })
      }
    )

    request.on('error', (error: Error) => {
      fail(error)
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
  options: { upstreamHost: string; timeout: number; verbose: boolean; agent?: http.Agent }
): Promise<void> {
  const normalizedUrl = relativeUrl.startsWith('/') ? relativeUrl : `/${relativeUrl}`
  const proxyReq = Object.create(req)
  proxyReq.url = normalizedUrl
  return proxyToApi(proxyReq, res, normalizedUrl, {
    apiPort: targetPort,
    apiHost: options.upstreamHost,
    timeout: options.timeout,
    verbose: options.verbose,
    agent: options.agent,
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

function copyIncomingHeaders(headers: http.IncomingHttpHeaders): http.IncomingHttpHeaders {
  const clone: http.IncomingHttpHeaders = {}
  for (const [key, value] of Object.entries(headers)) {
    if (Array.isArray(value)) {
      clone[key] = [...value]
      continue
    }
    if (typeof value === 'string' || typeof value === 'number') {
      clone[key] = value
    }
  }
  return clone
}

export function createScenarioProxyHost(options: ScenarioProxyHostOptions): ScenarioProxyHostController {
  if (!options.hostScenario) {
    throw new Error('hostScenario option is required')
  }
  if (typeof options.fetchAppMetadata !== 'function') {
    throw new Error('fetchAppMetadata option is required')
  }

  const hostEndpoints = normalizeHostEndpoints(options.hostEndpoints)
  const appsPrefix = normalizePrefix(options.appsPathPrefix ?? '/apps')
  const proxySegment = normalizeSegment(options.proxyPathSegment ?? 'proxy', 'proxy')
  const portsSegment = normalizeSegment(options.portsPathSegment ?? 'ports', 'ports')
  const loopbackHosts = options.loopbackHosts && options.loopbackHosts.length > 0
    ? options.loopbackHosts
    : Array.from(LOOPBACK_HOSTS)
  const router: Router = express.Router()
  const cache = new Map<string, CachedContext>()
  const cacheTtl = options.cacheTtlMs ?? DEFAULT_CACHE_TTL_MS
  const htmlCacheEnabled = options.cacheProxyHtml !== false
  const htmlCacheTtl = options.proxyHtmlCacheTtlMs ?? cacheTtl
  const htmlCacheMaxEntries = options.proxyHtmlCacheMaxEntries ?? 200
  const htmlCache = new Map<string, Map<string, HtmlCacheEntry>>()
  const htmlCacheOrder: HtmlCachePointer[] = []
  let htmlCacheEntries = 0
  const htmlCacheActive = htmlCacheEnabled && htmlCacheTtl > 0 && htmlCacheMaxEntries > 0
  const upstreamHost = options.upstreamHost ?? LOOPBACK_HOST
  const timeout = options.timeoutMs ?? DEFAULT_PROXY_TIMEOUT
  const verbose = Boolean(options.verbose)
  const patchFetch = options.patchFetch ?? false
  const childBaseTagAttr = options.childBaseTagAttribute || 'data-proxy-host'
  const proxyHeaderName = options.proxiedAppHeader || 'x-vrooli-proxied-app'
  const upstreamAgent = resolveProxyAgent({
    agent: options.proxyAgent,
    keepAlive: options.proxyKeepAlive,
  })

  function clearHtmlCacheEntries(appId?: string): void {
    if (!htmlCacheEntries) {
      return
    }
    if (!appId) {
      htmlCache.clear()
      htmlCacheOrder.length = 0
      htmlCacheEntries = 0
      return
    }
    const bucket = htmlCache.get(appId)
    if (!bucket) {
      return
    }
    const removed = bucket.size
    htmlCache.delete(appId)
    htmlCacheEntries = Math.max(0, htmlCacheEntries - removed)
    for (let i = htmlCacheOrder.length - 1; i >= 0; i -= 1) {
      if (htmlCacheOrder[i].appId === appId) {
        htmlCacheOrder.splice(i, 1)
      }
    }
  }

function removeHtmlCacheEntry(appId: string, path: string, expectedInsertedAt?: number): void {
  const bucket = htmlCache.get(appId)
  if (!bucket) {
    return
  }
  const normalizedPath = path || '/'
  const entry = bucket.get(normalizedPath)
  if (!entry) {
    return
  }
  if (expectedInsertedAt && entry.insertedAt !== expectedInsertedAt) {
    return
  }
  bucket.delete(normalizedPath)
  htmlCacheEntries = Math.max(0, htmlCacheEntries - 1)
  if (bucket.size === 0) {
    htmlCache.delete(appId)
  }
}

  function enforceHtmlCacheLimit(): void {
    if (!htmlCacheActive) {
      return
    }
    while (htmlCacheEntries > htmlCacheMaxEntries) {
      const oldest = htmlCacheOrder.shift()
      if (!oldest) {
        htmlCacheEntries = Math.min(htmlCacheEntries, htmlCacheMaxEntries)
        break
      }
      removeHtmlCacheEntry(oldest.appId, oldest.path, oldest.insertedAt)
    }
  }

function setCachedHtmlEntry(appId: string, path: string, payload: Omit<HtmlCacheEntry, 'timestamp' | 'insertedAt'>): void {
    if (!htmlCacheActive) {
      return
    }
    const normalizedPath = path || '/'
    let bucket = htmlCache.get(appId)
    if (!bucket) {
      bucket = new Map<string, HtmlCacheEntry>()
      htmlCache.set(appId, bucket)
    }
    const now = Date.now()
    const wasPresent = bucket.has(normalizedPath)
    bucket.set(normalizedPath, {
      ...payload,
      timestamp: now,
      insertedAt: now,
    })
    if (!wasPresent) {
      htmlCacheEntries += 1
    }
    htmlCacheOrder.push({ appId, path: normalizedPath, insertedAt: now })
    enforceHtmlCacheLimit()
  }

  function getCachedHtmlEntry(appId: string, path: string): HtmlCacheEntry | null {
    if (!htmlCacheActive) {
      return null
    }
    const normalizedPath = path || '/'
    const bucket = htmlCache.get(appId)
    if (!bucket) {
      return null
    }
    const entry = bucket.get(normalizedPath)
    if (!entry) {
      return null
    }
    if (Date.now() - entry.timestamp > htmlCacheTtl) {
      removeHtmlCacheEntry(appId, normalizedPath)
      return null
    }
    return entry
  }

  function responseIsCacheable(response: { status: number; headers: http.IncomingHttpHeaders }): boolean {
    if (!htmlCacheActive) {
      return false
    }
    if (response.status !== 200) {
      return false
    }
    const contentTypeHeader = response.headers['content-type']
    const contentType = Array.isArray(contentTypeHeader) ? contentTypeHeader[0] : contentTypeHeader
    if (!contentType || typeof contentType !== 'string' || !contentType.includes('text/html')) {
      return false
    }
    if (response.headers['set-cookie']) {
      return false
    }
    return true
  }

  async function ensureUpstreamReachable(port: number): Promise<boolean> {
    if (!htmlCacheActive) {
      return true
    }
    return await new Promise((resolve) => {
      let settled = false
      const finish = (value: boolean) => {
        if (settled) {
          return
        }
        settled = true
        resolve(value)
      }

      const socket = net.createConnection({ host: upstreamHost, port }, () => {
        socket.end()
        finish(true)
      })

      const timeoutForCheck = Math.max(50, Math.min(timeout, UPSTREAM_CHECK_TIMEOUT_MS))
      socket.setTimeout(timeoutForCheck, () => {
        socket.destroy()
        finish(false)
      })

      socket.on('error', () => {
        finish(false)
      })
    })
  }

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
      hostEndpoints,
    })

    cache.set(appId, { context, timestamp: now })
    return context
  }

  async function handleScenarioProxyRequest(req: Request, res: Response): Promise<void> {
    const appId = req.params.appId
    if (hasPathTraversal(appId)) {
      res.status(400).json({ error: 'Invalid proxy path' })
      return
    }
    const relativeUrl = extractProxyRelativeUrl(req.originalUrl || req.url || '/', proxySegment)
    if (hasPathTraversal(relativeUrl)) {
      res.status(400).json({ error: 'Invalid proxy path' })
      return
    }
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
          agent: upstreamAgent,
        })
        return
      }

      if (isHtmlLikeRequest(req, relativeUrl)) {
        const targetUrl = relativeUrl.startsWith('/') ? relativeUrl : `/${relativeUrl}`
        let htmlResponse = getCachedHtmlEntry(appId, targetUrl)

        if (htmlResponse) {
          const upstreamReachable = await ensureUpstreamReachable(context.uiPort)
          if (!upstreamReachable) {
            removeHtmlCacheEntry(appId, targetUrl)
            htmlResponse = null
          }
        }

        if (htmlResponse) {
          let body = htmlResponse.body
          if (!htmlResponse.injected) {
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

        const metadataPayload = { ...context.metadata, generatedAt: Date.now() }
        const basePath = `${context.metadata.primary.path || `${appsPrefix}/${appId}/${proxySegment}`}/`

        await streamProxiedHtml({
          appId,
          targetUrl,
          res,
          upstreamHost,
          upstreamPort: context.uiPort,
          requestHeaders: req.headers,
          timeout,
          metadata: metadataPayload,
          basePath,
          patchFetch,
          baseTagAttribute: childBaseTagAttr,
          proxyHeaderName,
          upstreamAgent,
          shouldCache: responseIsCacheable,
          onCacheStore: ({ status, headers, body }) => {
            setCachedHtmlEntry(appId, targetUrl, {
              status,
              headers,
              body,
              injected: true,
            })
          },
          onCacheSkip: () => {
            removeHtmlCacheEntry(appId, targetUrl)
          },
        })
        return
      }

      await forwardHttpRequest(req, res, relativeUrl, context.uiPort, {
        upstreamHost,
        timeout,
        verbose,
        agent: upstreamAgent,
      })
    } catch (error) {
      removeHtmlCacheEntry(appId, relativeUrl)
      console.error(`[proxy-host] Failed to proxy app ${appId}:`, error instanceof Error ? error.message : error)
      if (!res.headersSent) {
        res.status(502).json({ error: 'Failed to proxy application', details: error instanceof Error ? error.message : 'Unknown error' })
      }
    }
  }

  async function handlePortProxyRequest(req: Request, res: Response): Promise<void> {
    const appId = req.params.appId
    const portKey = req.params.portKey
    if (hasPathTraversal(appId)) {
      res.status(400).json({ error: 'Invalid proxy path' })
      return
    }
    const relativeUrl = extractProxyRelativeUrl(req.originalUrl || req.url || '/', proxySegment)
    if (hasPathTraversal(relativeUrl) || hasPathTraversal(portKey)) {
      res.status(400).json({ error: 'Invalid proxy path' })
      return
    }
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
        agent: upstreamAgent,
      })
    } catch (error) {
      console.error(`[proxy-host] Failed to proxy port ${appId}/${portKey}:`, error instanceof Error ? error.message : error)
      if (!res.headersSent) {
        res.status(502).json({ error: 'Failed to proxy port', details: error instanceof Error ? error.message : 'Unknown error' })
      }
    }
  }

  function registerRoutes(): void {
    router.use((req, res, next) => {
      const rawUrl = typeof req.originalUrl === 'string'
        ? req.originalUrl
        : (req.url || '')

      if (!rawUrl) {
        next()
        return
      }

      const pathOnly = rawUrl.split('?')[0] || '/'
      const hitsAppsPrefix = pathOnly === appsPrefix || pathOnly.startsWith(`${appsPrefix}/`)

      if (hitsAppsPrefix && hasPathTraversal(rawUrl)) {
        res.status(400).json({ error: 'Invalid proxy path' })
        return
      }

      next()
    })

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
        if (hasPathTraversal(appId)) {
          socket.destroy()
          return true
        }
        if (hasPathTraversal(portKey) || rest.some((segment) => hasPathTraversal(segment))) {
          socket.destroy()
          return true
        }
        const relativePath = rest.length > 0 ? `/${rest.join('/')}${search}` : `/${search || ''}`
        if (hasPathTraversal(relativePath)) {
          socket.destroy()
          return true
        }
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
        if (hasPathTraversal(appId)) {
          socket.destroy()
          return true
        }
        if (rest.some((segment) => hasPathTraversal(segment))) {
          socket.destroy()
          return true
        }
        const relativePath = rest.length > 0 ? `/${rest.join('/')}${search}` : `/${search || ''}`
        if (hasPathTraversal(relativePath)) {
          socket.destroy()
          return true
        }
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
      clearHtmlCacheEntries(appId)
    } else {
      cache.clear()
      clearHtmlCacheEntries()
    }
  }

  return {
    router,
    handleUpgrade,
    invalidate,
    clearCache: () => {
      cache.clear()
      clearHtmlCacheEntries()
    },
  }
}
