import express, { type Request, type Response, type Router } from 'express'
import { createHmac, randomBytes, timingSafeEqual } from 'node:crypto'
import * as http from 'node:http'
import * as net from 'node:net'
import type { LocalPortProxyOptions } from '../shared/types.js'
import { HOP_BY_HOP_HEADERS, LOOPBACK_HOST } from '../shared/constants.js'
import { proxyWebSocketUpgrade } from './proxy.js'
import { BROWSER_CONTEXT_HEADERS } from './origin-policy.js'
import { createBootstrapNonce, SANDBOX_POLICY, streamCss, streamHtml, withBootstrapNonce } from './local-port-proxy-document.js'

const DEFAULT_PREFIX = '/_local'
const DEFAULT_TIMEOUT_MS = 30_000
const FORWARDED_HEADERS = new Set(['forwarded', 'authorization', 'proxy-authorization', 'x-forwarded-for', 'x-forwarded-host', 'x-forwarded-proto'])
const MAX_ACTIVE_WEBSOCKETS = 128
const CAPABILITY_TTL_MS = 12 * 60 * 60 * 1000

export interface LocalPortProxyController {
  router: Router
  handleUpgrade: (req: http.IncomingMessage, socket: net.Socket, head: Buffer) => boolean
  destroy: () => void
}

function normalizePrefix(value?: string): string {
  const prefix = (value || DEFAULT_PREFIX).trim()
  if (!/^\/[A-Za-z0-9/_-]*$/.test(prefix) || prefix.startsWith('//') || prefix.split('/').some((part) => part === '.' || part === '..')) {
    throw new Error(`Invalid local port proxy path prefix: ${value}`)
  }
  return prefix.replace(/\/+$/, '') || DEFAULT_PREFIX
}

function parsePort(value: unknown): number | null {
  if (typeof value !== 'string' || !/^[0-9]{1,5}$/.test(value)) return null
  const port = Number(value)
  return Number.isInteger(port) && port >= 1 && port <= 65535 && String(port) === value ? port : null
}

function isLoopbackAddress(address: string | undefined): boolean {
  const normalized = address?.toLowerCase().replace(/^::ffff:/, '')
  if (normalized === '::1') return true
  const octets = normalized?.split('.') || []
  return octets.length === 4 && octets[0] === '127' && octets.every((octet) => /^\d{1,3}$/.test(octet) && Number(octet) <= 255)
}

function hasTraversal(pathname: string): boolean {
  let decoded = pathname
  try {
    for (let i = 0; i < 32; i++) {
      const next = decodeURIComponent(decoded)
      if (next === decoded) return decoded.split(/[\\/]/).some((segment) => segment === '.' || segment === '..')
      decoded = next
    }
    return true
  } catch { return true }
}

function cookiePrefix(port: number): string {
  return `vroot_p${port}_`
}

function outboundCookies(value: string | undefined, port: number): string | undefined {
  const prefix = cookiePrefix(port)
  const cookies = (value || '').split(';').map((part) => part.trim()).filter((part) => part.startsWith(prefix)).map((part) => part.slice(prefix.length))
  return cookies.length ? cookies.join('; ') : undefined
}

function scopedSetCookie(value: string, port: number, basePath: string): string {
  const parts = value.split(';').map((part) => part.trim())
  const separator = parts[0]!.indexOf('=')
  if (separator <= 0) return value
  parts[0] = `${cookiePrefix(port)}${parts[0]!.slice(0, separator)}=${parts[0]!.slice(separator + 1)}`
  const attributes = parts.slice(1).filter((attribute) => !/^domain\s*=/i.test(attribute) && !/^path\s*=/i.test(attribute))
  attributes.push(`Path=${basePath}/`)
  return [parts[0], ...attributes].join('; ')
}

function proxyPrefix(prefix: string, port: number, upstreamHost: string): string {
  return upstreamHost === '::1' ? `${prefix}/ipv6/${port}` : `${prefix}/${port}`
}

function requestQuery(url: string): string {
  const queryStart = url.indexOf('?')
  return queryStart >= 0 ? url.slice(queryStart) : ''
}

function isLoopbackHostname(hostname: string): boolean {
  const host = hostname.toLowerCase().replace(/^\[|\]$/g, '')
  if (host === 'localhost' || host === 'localhost.' || host === '0.0.0.0' || host === '::1') return true
  const octets = host.split('.')
  return octets.length === 4 && octets[0] === '127' && octets.every((octet) => /^\d{1,3}$/.test(octet) && Number(octet) <= 255)
}

function proxyLocation(value: string, port: number, targetPrefix: string, requestPath = '/'): string {
  try {
    const parsed = new URL(value, `http://127.0.0.1:${port}${requestPath}`)
    const effectivePort = Number(parsed.port || (parsed.protocol === 'https:' || parsed.protocol === 'wss:' ? '443' : '80'))
    if ((parsed.protocol === 'http:' || parsed.protocol === 'ws:') && isLoopbackHostname(parsed.hostname) && effectivePort === port) {
      return `${targetPrefix}${parsed.pathname}${parsed.search}${parsed.hash}`
    }
  } catch { /* retain a non-URL Location header verbatim */ }
  return value
}

// The proxy is a different origin from every service it reaches, and it has
// already enforced its own capability policy. The browser's Origin, Referer and
// Fetch Metadata describe the proxy page (often an opaque sandbox origin), so
// they are not forwarded: a scenario's own origin policy would otherwise
// refuse every proxied request as cross-site.
function isRelayedHeader(lower: string): boolean {
  return !HOP_BY_HOP_HEADERS.has(lower) && !FORWARDED_HEADERS.has(lower) && !BROWSER_CONTEXT_HEADERS.has(lower) && lower !== 'cookie'
}

function copyRequestHeaders(req: Request, port: number, upstreamHost: string): http.OutgoingHttpHeaders {
  const headers: http.OutgoingHttpHeaders = {}
  for (const [key, value] of Object.entries(req.headers)) {
    if (value === undefined || !isRelayedHeader(key.toLowerCase())) continue
    headers[key] = value
  }
  const cookie = outboundCookies(req.headers.cookie, port)
  if (cookie) headers.cookie = cookie
  const authorityHost = net.isIPv6(upstreamHost) ? `[${upstreamHost}]` : upstreamHost
  headers.host = `${authorityHost}:${port}`
  // The proxy owns its path and never forwards the browser's tunnel authority.
  const wantsRewrite = String(req.headers.accept || '').toLowerCase().includes('text/html') || /\.(?:html?|css)$/i.test(req.path || '')
  if (wantsRewrite) headers['accept-encoding'] = 'identity'
  return headers
}

function issueCapability(secret: Buffer, port: number, upstreamHost: string): string {
  const expiresAt = Date.now() + CAPABILITY_TTL_MS
  const signature = createHmac('sha256', secret).update(`${upstreamHost}:${port}:${expiresAt}`).digest('base64url')
  return `${expiresAt}.${signature}`
}

function verifyCapability(secret: Buffer, port: number, upstreamHost: string, capability: string | undefined): boolean {
  if (!capability) return false
  const match = /^(\d{13})\.([A-Za-z0-9_-]{43})$/.exec(capability)
  if (!match) return false
  const expiresAt = Number(match[1])
  if (expiresAt <= Date.now() || expiresAt > Date.now() + CAPABILITY_TTL_MS + 60_000) return false
  const expected = createHmac('sha256', secret).update(`${upstreamHost}:${port}:${expiresAt}`).digest()
  let supplied: Buffer
  try { supplied = Buffer.from(match[2]!, 'base64url') } catch { return false }
  return supplied.length === expected.length && timingSafeEqual(supplied, expected)
}

function requestProtocol(req: Pick<http.IncomingMessage, 'headers' | 'socket'>): string {
  const forwarded = req.headers['x-forwarded-proto']
  const firstForwarded = Array.isArray(forwarded) ? forwarded[0] : forwarded
  const candidate = firstForwarded?.split(',')[0]?.trim().toLowerCase()
  if (candidate === 'http' || candidate === 'https') return `${candidate}:`
  return (req.socket as net.Socket & { encrypted?: boolean }).encrypted ? 'https:' : 'http:'
}

function sameOrigin(origin: string | undefined, host: string | undefined, protocol?: string): boolean {
  if (!origin || !host || origin === 'null') return false
  try {
    const parsed = new URL(origin)
    return parsed.host.toLowerCase() === host.toLowerCase() && (!protocol || parsed.protocol === protocol)
  } catch { return false }
}

function htmlAttribute(value: string): string {
  return value.replace(/[&"'<>]/g, (character) => ({ '&': '&amp;', '"': '&quot;', "'": '&#39;', '<': '&lt;', '>': '&gt;' })[character]!)
}

/**
 * Create an opt-in proxy that can reach only numeric TCP ports on loopback.
 * It streams request and response bodies, scopes cookies by target port, and
 * sandboxes HTML into an opaque origin so a local service cannot run with the
 * Web Console origin's privileges.
 */
export function createLocalPortProxy(options: LocalPortProxyOptions = {}): LocalPortProxyController {
  const prefix = normalizePrefix(options.pathPrefix)
  const timeoutMs = Math.max(100, Math.min(options.timeoutMs ?? DEFAULT_TIMEOUT_MS, 120_000))
  const excludedPorts = new Set((options.excludedPorts ?? []).map((value) => parsePort(String(value))).filter((value): value is number => value !== null))
  const agent = new http.Agent({ keepAlive: true, maxSockets: 16, maxTotalSockets: 128, maxFreeSockets: 4, timeout: timeoutMs })
  const capabilitySecret = randomBytes(32)
  const router = express.Router({ caseSensitive: true, strict: false })
  let activeWebSockets = 0

  router.use((req, res, next) => {
    if (!isLoopbackAddress(req.socket.remoteAddress)) {
      res.status(403).json({ error: 'Local port proxy is available only through a loopback client connection' })
      return
    }
    next()
  })

  const proxyRequest = (req: Request, res: Response, portValue: string, upstreamHost: string = LOOPBACK_HOST): void => {
    const port = parsePort(portValue)
    if (!port || excludedPorts.has(port)) {
      res.status(400).json({ error: 'Invalid or unavailable local port' })
      return
    }
    const origin = req.headers.origin
    if (!['GET', 'HEAD', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'].includes(req.method.toUpperCase())) {
      res.status(405).setHeader('Allow', 'GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS').end()
      return
    }
    const pathname = req.url.split('?')[0] || '/'
    const routeRoot = upstreamHost === '::1' ? `/ipv6/${port}` : `/${port}`
    if (!pathname.startsWith(`${routeRoot}/`) && pathname !== routeRoot) {
      res.status(400).json({ error: 'Invalid local proxy route' })
      return
    }
    let suffix = pathname.slice(routeRoot.length) || '/'
    let capability: string | undefined
    let basePath = proxyPrefix(prefix, port, upstreamHost)
    if (suffix.startsWith('/')) {
      const nextSlash = suffix.indexOf('/', 1)
      const candidate = suffix.slice(1, nextSlash < 0 ? undefined : nextSlash)
      if (/^\d{13}\./.test(candidate)) {
        if (!verifyCapability(capabilitySecret, port, upstreamHost, candidate)) {
          res.status(403).json({ error: 'Local proxy capability expired or invalid' })
          return
        }
        capability = candidate
        basePath += `/${capability}`
        suffix = nextSlash < 0 ? '/' : suffix.slice(nextSlash) || '/'
      }
    }
    const requestPath = `${basePath}${suffix}`
    if (hasTraversal(requestPath)) {
      res.status(400).json({ error: 'Invalid proxy path' })
      return
    }
    const accessControlPreflight = req.method.toUpperCase() === 'OPTIONS' && req.headers['access-control-request-method']
    const fetchSite = req.headers['sec-fetch-site']
    const acceptsHtmlDocument = req.method === 'GET'
      && String(req.headers.accept || '').toLowerCase().includes('text/html')
    const sameOriginNavigation = acceptsHtmlDocument
      && (!origin || sameOrigin(origin, req.headers.host, requestProtocol(req)))
      && (fetchSite === 'same-origin' || fetchSite === 'none')
    // Different localhost ports are same-site but not same-origin. Browsers
    // label a clicked link from Web Console to such a proxy route `same-site`;
    // accept only a top-level document navigation so subresources and fetches
    // still need a capability.
    const sameSiteDocumentNavigation = acceptsHtmlDocument
      && !origin
      && fetchSite === 'same-site'
      && req.headers['sec-fetch-mode'] === 'navigate'
      && req.headers['sec-fetch-dest'] === 'document'
    const crossSiteDocumentNavigation = acceptsHtmlDocument
      && !origin
      && fetchSite === 'cross-site'
      && req.headers['sec-fetch-mode'] === 'navigate'
      && req.headers['sec-fetch-dest'] === 'document'
    // Links opened from a separate chat/site arrive as cross-site top-level
    // navigations. Permit only a user-activated document navigation; the
    // redirect grants a short-lived capability to that sandboxed page. Safari
    // sends Fetch Metadata but does not send Sec-Fetch-User, so it receives a
    // non-scriptable confirmation page and must make a same-origin user click
    // before any local service request is made.
    const userActivatedCrossSiteNavigation = crossSiteDocumentNavigation
      && req.headers['sec-fetch-user'] === '?1'
    const bootstrapNavigation = sameOriginNavigation || sameSiteDocumentNavigation || userActivatedCrossSiteNavigation
    if (!capability && !bootstrapNavigation) {
      if (crossSiteDocumentNavigation) {
        const retryPath = htmlAttribute(req.originalUrl)
        res.status(200)
        res.setHeader('Content-Type', 'text/html; charset=utf-8')
        res.setHeader('Cache-Control', 'no-store')
        res.setHeader('Referrer-Policy', 'no-referrer')
        res.setHeader('X-Frame-Options', 'DENY')
        res.setHeader('Content-Security-Policy', "default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
        res.end(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Open local app</title><main><h1>Open this local app?</h1><p>Continue to open the requested app through Web Console.</p><a id="local-proxy-continue" href="${retryPath}">Continue</a></main></html>`)
        return
      }
      res.status(403).json({ error: 'A local proxy capability is required' })
      return
    }
    if (origin && origin !== 'null' && !sameOrigin(origin, req.headers.host, requestProtocol(req))) {
      res.status(403).json({ error: 'Cross-origin requests cannot use the local port proxy' })
      return
    }
    if (fetchSite === 'cross-site'
      && ((!capability && !userActivatedCrossSiteNavigation) || (origin && origin !== 'null' && !sameOrigin(origin, req.headers.host, requestProtocol(req))))) {
      res.status(403).json({ error: 'Cross-site requests cannot use the local port proxy' })
      return
    }
    if (accessControlPreflight) {
      if (!capability || origin !== 'null') {
        res.status(403).end()
        return
      }
      res.status(204)
      res.setHeader('access-control-allow-origin', 'null')
      res.setHeader('access-control-allow-credentials', 'true')
      res.setHeader('access-control-allow-methods', 'GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS')
      if (typeof req.headers['access-control-request-headers'] === 'string') {
        res.setHeader('access-control-allow-headers', req.headers['access-control-request-headers'])
      }
      res.setHeader('access-control-max-age', '600')
      res.end()
      return
    }
    if (!capability && bootstrapNavigation) {
      const issued = issueCapability(capabilitySecret, port, upstreamHost)
      const targetBase = `${proxyPrefix(prefix, port, upstreamHost)}/${issued}`
      const query = requestQuery(req.url)
      res.status(302).setHeader('Location', `${targetBase}${suffix}${query}`)
      res.setHeader('Cache-Control', 'no-store')
      res.end()
      return
    }
    if (options.verbose) console.log(`[local-port-proxy] ${req.method} ${port}${suffix}${capability ? ' (capability)' : ''}`)

    const upstreamReq = http.request({
      hostname: upstreamHost,
      port,
      method: req.method,
      path: `${suffix}${requestQuery(req.url)}`,
      headers: copyRequestHeaders(req, port, upstreamHost),
      agent,
      timeout: timeoutMs,
    }, (upstreamRes) => {
      const isHtml = /^text\/html\b/i.test(String(upstreamRes.headers['content-type'] || ''))
      if (isHtml && !capability) {
        upstreamRes.resume()
        const issued = issueCapability(capabilitySecret, port, upstreamHost)
        const targetBase = `${proxyPrefix(prefix, port, upstreamHost)}/${issued}`
        const query = requestQuery(req.url)
        res.status(302).setHeader('Location', `${targetBase}${suffix}${query}`)
        res.setHeader('Cache-Control', 'no-store')
        res.end()
        return
      }
      res.status(upstreamRes.statusCode || 502)
      res.vary('Origin')
      // The proxied document runs in an opaque sandbox origin. Chromium may
      // send Origin: null on its CORS preflight but omit Origin on the actual
      // request (notably for credentialed RPC POSTs), while still requiring
      // ACAO on the response. Only a valid capability with null or absent
      // Origin gets the opaque-origin CORS response; present foreign origins
      // were rejected above.
      if (capability && (origin === 'null' || origin === undefined)) {
        res.setHeader('access-control-allow-origin', 'null')
        res.setHeader('access-control-allow-credentials', 'true')
        res.setHeader('access-control-allow-methods', 'GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS')
        if (typeof req.headers['access-control-request-headers'] === 'string') {
          res.setHeader('access-control-allow-headers', req.headers['access-control-request-headers'])
        }
      }
      let issuedRedirectCapability = false
      for (const [key, value] of Object.entries(upstreamRes.headers)) {
        const lower = key.toLowerCase()
        if (value === undefined || HOP_BY_HOP_HEADERS.has(lower) || lower.startsWith('access-control-')) continue
        if (lower === 'location') {
          if (!capability) {
            const issued = issueCapability(capabilitySecret, port, upstreamHost)
            basePath += `/${issued}`
            issuedRedirectCapability = true
          }
          const requestPath = `${suffix}${requestQuery(req.url)}`
          res.setHeader(key, Array.isArray(value) ? value.map((item) => proxyLocation(item, port, basePath, requestPath)) : proxyLocation(value, port, basePath, requestPath))
        } else if (lower === 'set-cookie') {
          const values = Array.isArray(value) ? value : [value]
          res.setHeader(key, values.map((item) => scopedSetCookie(item, port, proxyPrefix(prefix, port, upstreamHost))))
        } else {
          res.setHeader(key, value)
        }
      }
      if (capability || issuedRedirectCapability) res.setHeader('Cache-Control', 'no-store')
      // Prevent browser code in the proxied document from inheriting the
      // operator UI's origin. The lack of allow-same-origin is intentional.
      if (isHtml) {
        const existing = upstreamRes.headers['content-security-policy']
        const policies = existing === undefined ? [] : Array.isArray(existing) ? existing : [existing]
        const bootstrapNonce = policies.length ? createBootstrapNonce() : undefined
        const proxiedPolicies = bootstrapNonce ? policies.map((policy) => withBootstrapNonce(policy, bootstrapNonce)) : policies
        res.setHeader('content-security-policy', [...proxiedPolicies, SANDBOX_POLICY])
        res.setHeader('referrer-policy', 'no-referrer')
        res.setHeader('cache-control', 'no-store')
        const encoding = String(upstreamRes.headers['content-encoding'] || '').trim().toLowerCase()
        if (encoding && encoding !== 'identity' && encoding !== 'gzip' && encoding !== 'deflate' && encoding !== 'br') {
          upstreamRes.destroy()
          res.status(502).end('Unsupported content encoding for proxied HTML')
        } else {
          streamHtml(res, upstreamRes, basePath, port, capability!, bootstrapNonce)
        }
      } else if (/^text\/css\b/i.test(String(upstreamRes.headers['content-type'] || ''))) {
        streamCss(res, upstreamRes, basePath)
      } else {
        upstreamRes.pipe(res)
      }
    })

    upstreamReq.on('timeout', () => upstreamReq.destroy(new Error('Local service request timed out')))
    upstreamReq.on('error', (error) => {
      if (options.verbose) console.error(`[local-port-proxy] ${error.message}`)
      if (!res.headersSent) res.status(502).setHeader('Retry-After', '1').json({ error: 'Local service unavailable' })
      else res.destroy(error)
    })
    req.on('aborted', () => upstreamReq.destroy())
    res.on('close', () => { if (!res.writableEnded) upstreamReq.destroy() })
    upstreamReq.on('response', (upstreamRes) => {
      upstreamRes.on('error', (error) => {
        if (options.verbose) console.error(`[local-port-proxy] upstream response: ${error.message}`)
        if (res.headersSent) res.destroy(error)
      })
    })
    req.pipe(upstreamReq)
  }

  router.all('/ipv6/:port', (req, res) => proxyRequest(req, res, req.params.port!, '::1'))
  router.all('/ipv6/:port/*', (req, res) => proxyRequest(req, res, req.params.port!, '::1'))
  router.all('/:port', (req, res) => proxyRequest(req, res, req.params.port!))
  router.all('/:port/*', (req, res) => proxyRequest(req, res, req.params.port!))

  const handleUpgrade = (req: http.IncomingMessage, socket: net.Socket, head: Buffer): boolean => {
    if (!req.url) return false
    let parsed: URL
    try { parsed = new URL(req.url, 'http://localhost') } catch { return false }
    if (!parsed.pathname.startsWith(`${prefix}/`)) return false
    if (!isLoopbackAddress(socket.remoteAddress)) {
      socket.end('HTTP/1.1 403 Forbidden\r\nConnection: close\r\n\r\n')
      return true
    }
    const rest = parsed.pathname.slice(prefix.length + 1).split('/')
    const ipv6 = rest[0] === 'ipv6'
    const upstreamHost = ipv6 ? '::1' : LOOPBACK_HOST
    const rawPort = ipv6 ? rest[1] : rest[0]
    const port = parsePort(rawPort)
    const baseSegments = ipv6 ? 2 : 1
    const pathCapability = rest[baseSegments]
    const pathTokenValid = verifyCapability(capabilitySecret, port || 0, upstreamHost, pathCapability)
    const capability = pathTokenValid ? pathCapability : undefined
    const pathSegments = rest.slice(baseSegments + (pathTokenValid ? 1 : 0))
    const path = pathSegments.length ? `/${pathSegments.join('/')}` : '/'
    if (!port || excludedPorts.has(port) || hasTraversal(path)) {
      socket.destroy()
      return true
    }
    const foreignOrigin = req.headers.origin
      && req.headers.origin !== 'null'
      && !sameOrigin(req.headers.origin, req.headers.host, requestProtocol(req))
    if (!capability || foreignOrigin) {
      socket.end('HTTP/1.1 403 Forbidden\r\nConnection: close\r\n\r\n')
      return true
    }
    if (req.headers.upgrade?.toLowerCase() !== 'websocket' || req.headers['sec-websocket-version'] !== '13' || !req.headers['sec-websocket-key']) {
      socket.end('HTTP/1.1 400 Bad Request\r\nConnection: close\r\n\r\n')
      return true
    }
    if (activeWebSockets >= MAX_ACTIVE_WEBSOCKETS) {
      socket.end('HTTP/1.1 503 Service Unavailable\r\nConnection: close\r\nRetry-After: 1\r\n\r\n')
      return true
    }
    activeWebSockets++
    let released = false
    const release = () => {
      if (released) return
      released = true
      activeWebSockets = Math.max(0, activeWebSockets - 1)
    }
    socket.once('close', release)
    socket.once('error', release)
    socket.setTimeout(timeoutMs, () => socket.destroy())
    const headers: Record<string, string | string[]> = {}
    for (const [key, value] of Object.entries(req.headers)) {
      const lower = key.toLowerCase()
      // WebSocket handshakes keep their Upgrade/Connection hop headers.
      if (value !== undefined && !FORWARDED_HEADERS.has(lower) && !BROWSER_CONTEXT_HEADERS.has(lower) && lower !== 'cookie' && lower !== 'host') headers[key] = value
    }
    const cookie = outboundCookies(req.headers.cookie, port)
    if (cookie) headers.cookie = cookie
    const proxiedReq = Object.create(req) as http.IncomingMessage
    const rawQuery = requestQuery(req.url)
    proxiedReq.url = `${path}${rawQuery}`
    proxiedReq.headers = headers
    proxyWebSocketUpgrade(proxiedReq, socket, head, { apiPort: port, apiHost: upstreamHost, verbose: options.verbose })
    return true
  }

  return { router, handleUpgrade, destroy: () => agent.destroy() }
}
