/**
 * Mock window utilities for testing
 */

import type { WindowLike } from '../../shared/types.js'

export interface MockWindowOptions {
  hostname?: string
  origin?: string
  pathname?: string
  protocol?: string
  port?: string
  host?: string
  globals?: Record<string, unknown>
}

/**
 * Create a mock window object for testing
 */
export function mockWindow(options: MockWindowOptions = {}): WindowLike {
  const {
    hostname = 'localhost',
    origin = `http://${hostname}`,
    pathname = '/',
    protocol = 'http:',
    port = '',
    host = port ? `${hostname}:${port}` : hostname,
    globals = {},
  } = options

  return {
    location: {
      hostname,
      origin,
      pathname,
      protocol,
      port,
      host,
    },
    ...globals,
  }
}

/**
 * Create a mock window in localhost context
 */
export function mockLocalhost(port = '3000', pathname = '/'): WindowLike {
  return mockWindow({
    hostname: 'localhost',
    origin: `http://localhost:${port}`,
    pathname,
    port,
  })
}

/**
 * Create a mock window in remote host context
 */
export function mockRemoteHost(hostname = 'example.com', pathname = '/', https = true): WindowLike {
  const protocol = https ? 'https:' : 'http:'
  return mockWindow({
    hostname,
    origin: `${protocol}//${hostname}`,
    pathname,
    protocol,
  })
}

/**
 * Create a mock window in proxy context
 */
export function mockProxyContext(
  proxyPath = '/apps/scenario/proxy/',
  proxyInfo?: unknown
): WindowLike {
  return mockWindow({
    hostname: 'example.com',
    origin: 'https://example.com',
    pathname: proxyPath,
    protocol: 'https:',
    globals: proxyInfo ? { __VROOLI_PROXY_INFO__: proxyInfo } : {},
  })
}
