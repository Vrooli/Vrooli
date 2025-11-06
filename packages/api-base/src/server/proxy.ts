/**
 * Proxy middleware
 *
 * Functions for proxying HTTP requests from UI server to API server.
 * Follows scenario-auditor secure_tunnel.go rules.
 */

import * as http from 'node:http'
import * as net from 'node:net'
import type { Request, Response, NextFunction, RequestHandler } from 'express'
import { HOP_BY_HOP_HEADERS, DEFAULT_PROXY_TIMEOUT, LOOPBACK_HOST } from '../shared/constants.js'
import type { ProxyOptions } from '../shared/types.js'
import { parsePort } from '../shared/utils.js'

/**
 * Proxy a request to the API server
 *
 * Forwards an HTTP request from the UI server to the API server using Node's
 * http.request (not fetch/axios). Properly handles streaming, headers, timeouts.
 *
 * This follows the pattern required by scenario-auditor/api/rules/ui/secure_tunnel.go
 *
 * @param req - Express request object
 * @param res - Express response object
 * @param targetPath - Path to request on API server
 * @param options - Proxy options
 * @returns Promise that resolves when proxying completes
 *
 * @example
 * ```typescript
 * app.use('/api', async (req, res) => {
 *   await proxyToApi(req, res, req.url, {
 *     apiPort: 8080,
 *     apiHost: 'localhost'
 *   })
 * })
 * ```
 */
export async function proxyToApi(
  req: Request,
  res: Response,
  targetPath: string,
  options: ProxyOptions
): Promise<void> {
  const {
    apiPort,
    apiHost = LOOPBACK_HOST,
    timeout = DEFAULT_PROXY_TIMEOUT,
    headers: additionalHeaders = {},
    verbose = false,
  } = options

  // Parse port
  const port = parsePort(apiPort)
  if (!port) {
    res.status(502).json({
      error: 'API server unavailable',
      details: 'Invalid API_PORT configuration',
      target: null,
    })
    return
  }

  const portNumber = Number.parseInt(port, 10)

  // Build headers
  const proxyHeaders: http.OutgoingHttpHeaders = {}

  // Copy request headers, filtering hop-by-hop headers
  for (const [key, value] of Object.entries(req.headers)) {
    const lowerKey = key.toLowerCase()
    if (HOP_BY_HOP_HEADERS.has(lowerKey)) {
      continue
    }
    proxyHeaders[key] = value
  }

  // Set host header to target
  proxyHeaders.host = `${apiHost}:${portNumber}`

  // Add any additional headers
  Object.assign(proxyHeaders, additionalHeaders)

  // Ensure accept header
  if (!proxyHeaders.accept) {
    proxyHeaders.accept = 'application/json'
  }

  if (verbose) {
    console.log(`[proxy] ${req.method} ${targetPath} -> ${apiHost}:${portNumber}${targetPath}`)
  }

  return new Promise<void>((resolve) => {
    const proxyReq = http.request(
      {
        hostname: apiHost,
        port: portNumber,
        path: targetPath,
        method: req.method,
        headers: proxyHeaders,
        timeout,
      },
      (proxyRes: http.IncomingMessage) => {
        // Set status code
        res.status(proxyRes.statusCode ?? 500)

        // Copy response headers, filtering hop-by-hop headers
        for (const [key, value] of Object.entries(proxyRes.headers)) {
          const lowerKey = key.toLowerCase()
          if (HOP_BY_HOP_HEADERS.has(lowerKey)) {
            continue
          }
          if (value !== undefined && value !== null) {
            res.setHeader(key, value)
          }
        }

        // Stream response body
        proxyRes.pipe(res)

        proxyRes.on('end', () => {
          resolve()
        })
      }
    )

    // Handle proxy request errors
    proxyReq.on('error', (err: Error) => {
      if (verbose) {
        console.error(`[proxy] Error: ${err.message}`)
      }

      if (!res.headersSent) {
        res.status(502).json({
          error: 'API server unavailable',
          details: err.message,
          target: `http://${apiHost}:${portNumber}${targetPath}`,
        })
      }

      resolve()
    })

    // Handle timeout
    proxyReq.on('timeout', () => {
      proxyReq.destroy(new Error('Request to API timed out'))
    })

    // Handle request body for POST/PUT/PATCH
    if (req.method !== 'GET' && req.method !== 'HEAD') {
      // If body has been parsed by express.json(), use the parsed body
      if (req.body !== undefined) {
        const bodyString = typeof req.body === 'string' ? req.body : JSON.stringify(req.body)
        proxyReq.write(bodyString)
        proxyReq.end()
      } else {
        // Otherwise pipe the raw request stream
        req.pipe(proxyReq)
      }
    } else {
      proxyReq.end()
    }
  })
}

/**
 * Create Express middleware for proxying API requests
 *
 * Returns a middleware function that proxies all requests to the API server.
 * Automatically handles path rewriting and error cases.
 *
 * @param options - Proxy options
 * @returns Express middleware
 *
 * @example
 * ```typescript
 * const proxyMiddleware = createProxyMiddleware({
 *   apiPort: process.env.API_PORT || 8080,
 *   apiHost: 'localhost',
 *   verbose: true
 * })
 *
 * app.use('/api', proxyMiddleware)
 * ```
 */
export function createProxyMiddleware(options: ProxyOptions): RequestHandler {
  return async (req: Request, res: Response, next: NextFunction) => {
    try {
      // Construct full path (preserves query params, etc.)
      const fullPath = req.url.startsWith('/api') ? req.url : `/api${req.url}`
      await proxyToApi(req, res, fullPath, options)
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error'
      console.error(`[proxy] Unexpected error: ${message}`)

      if (!res.headersSent) {
        res.status(500).json({
          error: 'Internal proxy error',
          details: message,
        })
      }
      next(error)
    }
  }
}

/**
 * Proxy WebSocket upgrade request
 *
 * Handles WebSocket upgrade requests by establishing a TCP tunnel between
 * the client and API server. This preserves the WebSocket protocol intact.
 *
 * @param req - HTTP upgrade request
 * @param clientSocket - Client socket
 * @param head - Initial data buffer
 * @param options - Proxy options
 *
 * @example
 * ```typescript
 * server.on('upgrade', (req, socket, head) => {
 *   if (req.url?.startsWith('/ws')) {
 *     proxyWebSocketUpgrade(req, socket, head, {
 *       apiPort: process.env.API_PORT || 8080
 *     })
 *   }
 * })
 * ```
 */
export function proxyWebSocketUpgrade(
  req: http.IncomingMessage,
  clientSocket: any,
  head: Buffer,
  options: ProxyOptions
): void {
  const {
    apiPort,
    apiHost = LOOPBACK_HOST,
    verbose = false,
  } = options

  const port = parsePort(apiPort)
  if (!port) {
    const errorMsg = `Invalid API_PORT configuration: ${apiPort}`
    console.error(`[ws-proxy] ${errorMsg}`)
    clientSocket.write(
      'HTTP/1.1 502 Bad Gateway\r\n' +
      'Content-Type: text/plain\r\n' +
      'Connection: close\r\n\r\n' +
      `${errorMsg}\r\n`
    )
    clientSocket.destroy()
    return
  }

  const portNumber = Number.parseInt(port, 10)

  if (verbose) {
    console.log(`[ws-proxy] Upgrade request: ${req.method} ${req.url} -> ${apiHost}:${portNumber}`)
    console.log(`[ws-proxy] Client headers:`, Object.keys(req.headers).join(', '))
  }

  // Establish TCP connection to API server
  const upstream = net.connect(portNumber, apiHost, () => {
    if (verbose) {
      console.log(`[ws-proxy] Connected to upstream ${apiHost}:${portNumber}`)
    }

    // Configure sockets
    upstream.setNoDelay(true)
    clientSocket.setNoDelay(true)
    upstream.setTimeout(0)
    clientSocket.setTimeout(0)

    if (typeof upstream.setKeepAlive === 'function') {
      upstream.setKeepAlive(true, 0)
    }
    if (typeof clientSocket.setKeepAlive === 'function') {
      clientSocket.setKeepAlive(true, 0)
    }

    // Build upgrade request headers
    // For WebSocket upgrades, we must preserve ALL headers including WebSocket-specific ones
    const headers: Record<string, string | string[]> = {}
    for (const [key, value] of Object.entries(req.headers)) {
      if (value !== undefined) {
        headers[key] = value as string | string[]
      }
    }

    // Validate critical WebSocket headers are present
    const wsVersion = headers['sec-websocket-version']
    const wsKey = headers['sec-websocket-key']
    if (!wsVersion || !wsKey) {
      const missing = []
      if (!wsVersion) missing.push('Sec-WebSocket-Version')
      if (!wsKey) missing.push('Sec-WebSocket-Key')
      console.error(`[ws-proxy] Missing required WebSocket headers: ${missing.join(', ')}`)
      if (verbose) {
        console.error(`[ws-proxy] Available headers:`, Object.keys(headers))
      }
    }

    // Override critical headers for proxying
    headers.host = `${apiHost}:${portNumber}`
    if (!headers.connection) {
      headers.connection = 'Upgrade'
    }
    if (!headers.upgrade) {
      headers.upgrade = 'websocket'
    }

    // Write HTTP upgrade request to upstream
    const requestLine = `${req.method} ${req.url} HTTP/${req.httpVersion}\r\n`
    const headerLines = Object.entries(headers)
      .flatMap(([key, value]) => {
        if (Array.isArray(value)) {
          return value.map(v => `${key}: ${v}`)
        }
        if (typeof value === 'string' && value.length > 0) {
          return [`${key}: ${value}`]
        }
        return []
      })
      .join('\r\n')

    if (verbose) {
      console.log(`[ws-proxy] Forwarding upgrade request with headers:`)
      console.log(`[ws-proxy]   Sec-WebSocket-Version: ${wsVersion}`)
      console.log(`[ws-proxy]   Sec-WebSocket-Key: ${wsKey ? '[present]' : '[missing]'}`)
      console.log(`[ws-proxy]   Connection: ${headers.connection}`)
      console.log(`[ws-proxy]   Upgrade: ${headers.upgrade}`)
    }

    upstream.write(`${requestLine}${headerLines}\r\n\r\n`)

    // Forward initial data if present
    if (head && head.length > 0) {
      if (verbose) {
        console.log(`[ws-proxy] Forwarding ${head.length} bytes of initial data`)
      }
      upstream.write(head)
    }

    // Pipe bidirectional data
    upstream.pipe(clientSocket)
    clientSocket.pipe(upstream)
  })

  // Error handling
  const teardown = (reason?: any) => {
    if (reason && verbose) {
      console.error('[ws-proxy] Tearing down:', reason)
    }
    try {
      clientSocket.destroy()
    } catch (err) {
      if (verbose) {
        console.error('[ws-proxy] Client destroy error:', err)
      }
    }
    try {
      upstream.destroy()
    } catch (err) {
      if (verbose) {
        console.error('[ws-proxy] Upstream destroy error:', err)
      }
    }
  }

  upstream.on('error', (error: Error) => {
    console.error(`[ws-proxy] Upstream connection error: ${error.message} (${apiHost}:${portNumber})`)
    if (verbose) {
      console.error(`[ws-proxy] Full error:`, error)
    }
    teardown({ stage: 'upstream-error', message: error.message, host: apiHost, port: portNumber })
  })

  clientSocket.on('error', (error: Error) => {
    if (verbose) {
      console.error(`[ws-proxy] Client socket error: ${error.message}`)
    }
    teardown({ stage: 'client-error', message: error.message })
  })

  clientSocket.on('close', (hadError: boolean) => {
    if (verbose) {
      console.log(`[ws-proxy] Client closed (error: ${hadError})`)
    }
    upstream.end()
  })

  upstream.on('close', (hadError: boolean) => {
    if (verbose) {
      console.log(`[ws-proxy] Upstream closed (error: ${hadError})`)
    }
    clientSocket.end()
  })
}
