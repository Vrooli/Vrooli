/**
 * Browser origin policy for scenario UI servers.
 *
 * A scenario UI server answers every page the operator's browser loads, both
 * through localhost and through the tunnel. Its routes proxy the scenario API,
 * and some APIs trust loopback callers as the owner. Any page the operator
 * visits could otherwise `fetch('http://localhost:<port>/api/...')`, so the
 * browser-facing server, not each API, decides which browser contexts may
 * reach it.
 *
 * The browser states where a request comes from (Fetch Metadata,
 * `Sec-Fetch-Site`). The policy is:
 *
 * - same-origin and user-initiated (`none`) requests pass;
 * - requests without browser context pass: CLIs, services, and relays that
 *   already applied their own policy send neither `Sec-Fetch-Site` nor
 *   `Origin`;
 * - a cross-origin request passes only when its `Origin` is declared in
 *   `allowedOrigins`, when it reads (GET/HEAD) a declared public path, or
 *   when it is a document navigation (a link the operator follows);
 * - everything else is refused with 403 before any route, proxy or static
 *   handler runs.
 *
 * CORS alone is not enough: a cross-site `<form>` POST or a `no-cors` fetch is
 * a "simple request" that reaches the server without a preflight. Fetch
 * Metadata refuses those writes; CORS headers are then emitted only for
 * declared origins (credentialed, exact echo) and public paths (`*`, no
 * credentials).
 *
 * Browsers without Fetch Metadata fall back to comparing `Origin` with `Host`.
 * That needs no configuration for the tunnel, which preserves the public
 * hostname in `Host`.
 *
 * See packages/api-base/SECURITY.md.
 */

import type { IncomingHttpHeaders, IncomingMessage, OutgoingHttpHeaders } from 'node:http'
import type { NextFunction, Request, RequestHandler, Response } from 'express'
import type { OriginPolicyOptions } from '../shared/types.js'

/** Why a request was admitted or refused. */
export type OriginVerdict =
  | 'same-origin'
  | 'no-browser-context'
  | 'navigation'
  | 'allowed-origin'
  | 'public-path'
  | 'refused'

export interface OriginDecision {
  verdict: OriginVerdict
  /** Canonical origin admitted by `allowedOrigins` (verdict `allowed-origin`). */
  origin?: string
  /** Human-readable reason, set when the verdict is `refused`. */
  reason?: string
}

/** The request facts the policy reads. */
export interface OriginPolicyRequest {
  method: string
  /** URL path without the query string. */
  path: string
  headers: IncomingHttpHeaders
}

export interface OriginPolicy {
  /** Canonical declared origins. */
  readonly allowedOrigins: ReadonlySet<string>
  /** Classify an HTTP request. */
  evaluate(request: OriginPolicyRequest): OriginDecision
  /** Classify a WebSocket handshake. Public paths and navigations do not apply. */
  evaluateUpgrade(request: Pick<IncomingMessage, 'headers'>): OriginDecision
  /** Express middleware that enforces the policy and writes CORS headers. */
  middleware(options?: { verbose?: boolean }): RequestHandler
}

/**
 * Request headers that describe the browser context of the page that made a
 * request. A relay to a different origin drops them: they name the relay's
 * origin, not the upstream's, and the relay has already applied its policy.
 */
export const BROWSER_CONTEXT_HEADERS: ReadonlySet<string> = new Set([
  'origin',
  'referer',
  'sec-fetch-site',
  'sec-fetch-mode',
  'sec-fetch-dest',
  'sec-fetch-user',
  'sec-fetch-storage-access',
])

const SAFE_READ_METHODS = new Set(['GET', 'HEAD'])
const NON_DOCUMENT_NAVIGATION_DESTINATIONS = new Set(['object', 'embed'])
const ORIGIN_SYNTAX = /^([a-z][a-z0-9+.-]*):\/\/([^/?#@\s]+)$/i
const HEADER_LIST = /^[A-Za-z0-9!#$%&'*+.^_`|~-]+(?:\s*,\s*[A-Za-z0-9!#$%&'*+.^_`|~-]+)*$/
const DEFAULT_ALLOWED_HEADERS = 'Accept, Authorization, Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms, X-CSRF-Token, X-Requested-With'
const ALLOWED_METHODS = 'GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS'
const EXPOSED_HEADERS = 'Link, Server-Timing'
const PREFLIGHT_MAX_AGE_SECONDS = '300'

function headerValue(headers: IncomingHttpHeaders, name: string): string | undefined {
  const value = headers[name]
  const first = Array.isArray(value) ? value[0] : value
  const trimmed = first?.trim()
  return trimmed ? trimmed : undefined
}

/**
 * Canonical `scheme://host[:port]` form of an origin, or null when the value is
 * not a concrete origin. HTTP(S) origins are normalized by the URL parser
 * (lower-case host, default port removed); other schemes such as
 * `chrome-extension://<id>` are lower-cased.
 */
export function canonicalOrigin(value: string): string | null {
  const trimmed = value.trim()
  const match = ORIGIN_SYNTAX.exec(trimmed)
  if (!match || trimmed.includes('*')) return null
  const scheme = match[1]!.toLowerCase()
  if (scheme === 'http' || scheme === 'https') {
    try {
      const parsed = new URL(trimmed)
      return parsed.origin === 'null' ? null : parsed.origin
    } catch {
      return null
    }
  }
  return `${scheme}://${match[2]!.toLowerCase()}`
}

function describeOriginValue(value: unknown): string {
  return typeof value === 'string' ? JSON.stringify(value) : String(value)
}

/**
 * Validate declared origins. Every entry must be one concrete origin; `*`,
 * patterns, `null`, paths and comma-joined lists are configuration errors,
 * because each would admit pages the scenario never named.
 */
export function normalizeAllowedOrigins(value: unknown): Set<string> {
  if (value === undefined) return new Set()
  if (!Array.isArray(value)) {
    throw new Error(
      `allowedOrigins must be an array of exact origins such as ["chrome-extension://<id>"], got ${describeOriginValue(value)}. ` +
      'Same-origin access, including through the tunnel, needs no entry.',
    )
  }
  const origins = new Set<string>()
  for (const entry of value) {
    if (typeof entry !== 'string') {
      throw new Error(`allowedOrigins entries must be strings, got ${describeOriginValue(entry)}`)
    }
    if (entry.includes('*')) {
      throw new Error(
        `allowedOrigins entry ${JSON.stringify(entry)} is a wildcard. Wildcards let any page the operator visits read and change this scenario's data; list each exact origin instead.`,
      )
    }
    const canonical = canonicalOrigin(entry)
    if (!canonical) {
      throw new Error(
        `allowedOrigins entry ${JSON.stringify(entry)} is not an exact origin (scheme://host[:port], no path). ` +
        '"null" is refused because every sandboxed frame on any site sends it.',
      )
    }
    origins.add(canonical)
  }
  return origins
}

type PublicPathMatcher = (path: string) => boolean

function normalizePublicPaths(value: unknown): PublicPathMatcher[] {
  if (value === undefined) return []
  if (!Array.isArray(value)) {
    throw new Error('publicPaths must be an array of path prefixes (strings starting with "/") or regular expressions')
  }
  return value.map((entry): PublicPathMatcher => {
    if (entry instanceof RegExp) {
      const pattern = new RegExp(entry.source, entry.flags.replace(/[gy]/g, ''))
      return (path) => pattern.test(path)
    }
    if (typeof entry === 'string' && entry.startsWith('/') && entry.length > 1 && !entry.includes('*')) {
      return (path) => path.startsWith(entry)
    }
    throw new Error(
      `publicPaths entry ${describeOriginValue(entry)} must be a path prefix such as "/preview/runtime/" or a RegExp; "/" and wildcards are refused because they would make every route public`,
    )
  })
}

function stripDefaultPort(host: string, scheme: string): string {
  if (scheme === 'http:' && host.endsWith(':80')) return host.slice(0, -3)
  if (scheme === 'https:' && host.endsWith(':443')) return host.slice(0, -4)
  return host
}

/** Whether an Origin header names the same authority the request was sent to. */
function originMatchesHost(origin: string, host: string | undefined): boolean {
  if (!host) return false
  try {
    const parsed = new URL(origin)
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return false
    return parsed.host.toLowerCase() === stripDefaultPort(host.toLowerCase(), parsed.protocol)
  } catch {
    return false
  }
}

function classify(
  request: OriginPolicyRequest,
  allowedOrigins: ReadonlySet<string>,
  publicPaths: PublicPathMatcher[],
  kind: 'http' | 'websocket',
): OriginDecision {
  const headers = request.headers
  const site = headerValue(headers, 'sec-fetch-site')?.toLowerCase()
  const origin = headerValue(headers, 'origin')

  if (site) {
    if (site === 'same-origin' || site === 'none') return { verdict: 'same-origin' }
  } else if (origin) {
    if (originMatchesHost(origin, headerValue(headers, 'host'))) return { verdict: 'same-origin' }
  } else {
    return { verdict: 'no-browser-context' }
  }

  const canonical = origin ? canonicalOrigin(origin) : null
  if (canonical && allowedOrigins.has(canonical)) {
    return { verdict: 'allowed-origin', origin: canonical }
  }

  if (kind === 'http') {
    const method = request.method.toUpperCase()
    const preflightMethod = method === 'OPTIONS'
      ? headerValue(headers, 'access-control-request-method')?.toUpperCase()
      : undefined
    const reads = SAFE_READ_METHODS.has(method) || (preflightMethod !== undefined && SAFE_READ_METHODS.has(preflightMethod))
    if (reads && publicPaths.some((matches) => matches(request.path))) {
      return { verdict: 'public-path' }
    }

    const mode = headerValue(headers, 'sec-fetch-mode')?.toLowerCase()
    const destination = headerValue(headers, 'sec-fetch-dest')?.toLowerCase() ?? ''
    if (mode === 'navigate' && SAFE_READ_METHODS.has(method) && !NON_DOCUMENT_NAVIGATION_DESTINATIONS.has(destination)) {
      return { verdict: 'navigation' }
    }
  }

  const subject = kind === 'websocket' ? 'WebSocket handshake' : 'Request'
  const from = origin ? `origin ${origin}` : `a ${site ?? 'cross-origin'} page that sent no Origin`
  return {
    verdict: 'refused',
    reason: `${subject} from ${from} is cross-origin and not declared in this scenario's allowedOrigins`,
  }
}

function applyVary(res: Response, value: string): void {
  const current = res.getHeader('Vary')
  const values = current === undefined ? [] : String(Array.isArray(current) ? current.join(',') : current).split(',').map((item) => item.trim()).filter(Boolean)
  if (values.some((item) => item === '*' || item.toLowerCase() === value.toLowerCase())) return
  res.setHeader('Vary', [...values, value].join(', '))
}

function requestedHeaders(req: Request): string {
  const requested = headerValue(req.headers, 'access-control-request-headers')
  return requested && HEADER_LIST.test(requested) ? requested : DEFAULT_ALLOWED_HEADERS
}

/**
 * Build a policy from declared configuration. Throws on unsafe configuration
 * so a scenario cannot start with an open policy by mistake.
 */
export function createOriginPolicy(options: OriginPolicyOptions = {}): OriginPolicy {
  const allowedOrigins = normalizeAllowedOrigins(options.allowedOrigins)
  const publicPaths = normalizePublicPaths(options.publicPaths)

  const evaluate = (request: OriginPolicyRequest) => classify(request, allowedOrigins, publicPaths, 'http')
  const evaluateUpgrade = (request: Pick<IncomingMessage, 'headers'>) =>
    classify({ method: 'GET', path: '', headers: request.headers }, allowedOrigins, publicPaths, 'websocket')

  const middleware = ({ verbose = false }: { verbose?: boolean } = {}): RequestHandler =>
    (req: Request, res: Response, next: NextFunction) => {
      const decision = evaluate({ method: req.method, path: req.path, headers: req.headers })
      if (allowedOrigins.size > 0) applyVary(res, 'Origin')

      if (decision.verdict === 'refused') {
        if (verbose) console.warn(`[origin-policy] Refused ${req.method} ${req.path}: ${decision.reason}`)
        res.setHeader('Cache-Control', 'no-store')
        res.status(403).json({
          error: 'Cross-origin request refused',
          detail: decision.reason,
          remedy: 'Open the scenario at its own address, or declare the calling origin in allowedOrigins in ui/server.js. See packages/api-base/SECURITY.md.',
        })
        return
      }

      const preflight = req.method === 'OPTIONS' && headerValue(req.headers, 'access-control-request-method') !== undefined
      if (decision.verdict === 'allowed-origin') {
        res.setHeader('Access-Control-Allow-Origin', headerValue(req.headers, 'origin')!)
        res.setHeader('Access-Control-Allow-Credentials', 'true')
        res.setHeader('Access-Control-Expose-Headers', EXPOSED_HEADERS)
        applyVary(res, 'Origin')
        if (preflight) {
          res.setHeader('Access-Control-Allow-Methods', ALLOWED_METHODS)
          res.setHeader('Access-Control-Allow-Headers', requestedHeaders(req))
          res.setHeader('Access-Control-Max-Age', PREFLIGHT_MAX_AGE_SECONDS)
          res.status(204).end()
          return
        }
      } else if (decision.verdict === 'public-path') {
        res.setHeader('Access-Control-Allow-Origin', '*')
        res.setHeader('Access-Control-Expose-Headers', EXPOSED_HEADERS)
        if (preflight) {
          res.setHeader('Access-Control-Allow-Methods', 'GET, HEAD')
          res.setHeader('Access-Control-Allow-Headers', requestedHeaders(req))
          res.setHeader('Access-Control-Max-Age', PREFLIGHT_MAX_AGE_SECONDS)
          res.status(204).end()
          return
        }
      }

      next()
    }

  return { allowedOrigins, evaluate, evaluateUpgrade, middleware }
}

/** Same-origin-only policy used when a caller declares nothing. */
export const SAME_ORIGIN_POLICY: OriginPolicy = createOriginPolicy()

/**
 * Copy request headers for a relay to a different origin, dropping the
 * browser-context headers. The relay's own policy has already admitted the
 * request; forwarding the browser's Origin and Fetch Metadata would make the
 * upstream judge the relay's origin as if it were its own.
 */
export function withoutBrowserContext<T extends IncomingHttpHeaders | OutgoingHttpHeaders>(headers: T): T {
  const copy = {} as Record<string, unknown>
  for (const [key, value] of Object.entries(headers)) {
    if (value === undefined || BROWSER_CONTEXT_HEADERS.has(key.toLowerCase())) continue
    copy[key] = value
  }
  return copy as T
}

/** Whether a response header is a CORS header owned by the browser-facing server. */
export function isCorsResponseHeader(name: string): boolean {
  return name.toLowerCase().startsWith('access-control-')
}

/** Write a refusal to a raw upgrade socket and close it. */
export function refuseUpgrade(socket: { end: (chunk: string) => void; destroy?: () => void }, reason: string): void {
  const body = `${reason}\r\n`
  socket.end(
    'HTTP/1.1 403 Forbidden\r\n' +
    'Content-Type: text/plain; charset=utf-8\r\n' +
    `Content-Length: ${Buffer.byteLength(body)}\r\n` +
    'Connection: close\r\n\r\n' +
    body,
  )
}
