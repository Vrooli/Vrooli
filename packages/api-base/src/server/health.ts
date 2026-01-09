/**
 * Health endpoint middleware
 *
 * Creates an Express endpoint that reports service health and API connectivity.
 * Used by monitoring systems and lifecycle orchestration.
 */

import * as http from 'node:http'
import type { Request, Response, RequestHandler } from 'express'
import type { HealthOptions, HealthCheckResult } from '../shared/types.js'
import { LOOPBACK_HOST, DEFAULT_HEALTH_CHECK_TIMEOUT } from '../shared/constants.js'
import { parsePort } from '../shared/utils.js'

/**
 * Check API connectivity
 *
 * Attempts to connect to the API server and checks its /health endpoint.
 *
 * @param apiHost - API server hostname
 * @param apiPort - API server port
 * @param timeout - Request timeout in milliseconds
 * @returns API connectivity status
 *
 * @internal
 */
async function checkApiConnectivity(
  apiHost: string,
  apiPort: number,
  timeout: number
): Promise<{
  connected: boolean
  latencyMs: number | null
  error: { code: string; message: string; category: string; retryable: boolean } | null
  upstream?: unknown
}> {
  const startTime = Date.now()

  return new Promise((resolve) => {
    const req = http.request(
      {
        hostname: apiHost,
        port: apiPort,
        path: '/health',
        method: 'GET',
        timeout,
      },
      (res: http.IncomingMessage) => {
        const latencyMs = Date.now() - startTime
        const connected = res.statusCode !== undefined && res.statusCode >= 200 && res.statusCode < 300

        let body = ''
        res.on('data', (chunk: Buffer) => {
          body += chunk.toString()
        })

        res.on('end', () => {
          let upstream: unknown = null
          try {
            upstream = JSON.parse(body)
          } catch {
            upstream = body
          }

          if (!connected) {
            resolve({
              connected: false,
              latencyMs,
              error: {
                code: `HTTP_${res.statusCode || 'UNKNOWN'}`,
                message: `API returned status ${res.statusCode || 'unknown'}`,
                category: 'network',
                retryable: true,
              },
              upstream,
            })
          } else {
            resolve({
              connected: true,
              latencyMs,
              error: null,
              upstream,
            })
          }
        })
      }
    )

    req.on('error', (error: Error) => {
      resolve({
        connected: false,
        latencyMs: Date.now() - startTime,
        error: {
          code: 'CONNECTION_ERROR',
          message: `Failed to connect to API: ${error.message}`,
          category: 'network',
          retryable: true,
        },
      })
    })

    req.on('timeout', () => {
      req.destroy()
      resolve({
        connected: false,
        latencyMs: Date.now() - startTime,
        error: {
          code: 'TIMEOUT',
          message: 'API health check timed out',
          category: 'network',
          retryable: true,
        },
      })
    })

    req.end()
  })
}

/**
 * Build health check result
 *
 * Constructs the health check response object.
 *
 * @param options - Health check options
 * @param apiConnectivity - API connectivity status (if checked)
 * @param customHealth - Custom health check results
 * @returns Health check result object
 *
 * @internal
 */
function buildHealthCheckResult(
  options: HealthOptions,
  apiConnectivity?: {
    connected: boolean
    latencyMs: number | null
    error: { code: string; message: string; category: string; retryable: boolean } | null
    upstream?: unknown
  },
  customHealth?: Record<string, unknown>
): HealthCheckResult {
  const { serviceName, version } = options
  const timestamp = new Date().toISOString()

  // Determine overall status
  let status: 'healthy' | 'degraded' | 'unhealthy' = 'healthy'

  if (apiConnectivity && !apiConnectivity.connected) {
    status = 'degraded'
  }

  const result: HealthCheckResult = {
    status,
    service: serviceName,
    timestamp,
    readiness: true,
  }

  if (version) {
    result.version = version
  }

  // Add API connectivity info if checked
  if (apiConnectivity) {
    const parsedApiPort = parsePort(options.apiPort)
    result.api_connectivity = {
      connected: apiConnectivity.connected,
      api_url: parsedApiPort ? `http://${options.apiHost || LOOPBACK_HOST}:${parsedApiPort}/health` : null,
      last_check: timestamp,
      error: apiConnectivity.error,
      latency_ms: apiConnectivity.latencyMs,
      upstream: apiConnectivity.upstream,
    }
  }

  // Add custom health check results
  if (customHealth) {
    Object.assign(result, customHealth)
  }

  return result
}

/**
 * Create health endpoint middleware
 *
 * Returns an Express request handler that reports service health.
 * Optionally checks API connectivity if apiPort is provided.
 *
 * @param options - Health endpoint options
 * @returns Express request handler
 *
 * @example
 * ```typescript
 * const healthHandler = createHealthEndpoint({
 *   serviceName: 'my-scenario-ui',
 *   version: '1.0.0',
 *   apiPort: 8080,
 *   timeout: 5000
 * })
 *
 * app.get('/health', healthHandler)
 * ```
 */
export function createHealthEndpoint(options: HealthOptions): RequestHandler {
  const {
    serviceName,
    version,
    apiPort,
    apiHost = LOOPBACK_HOST,
    timeout = DEFAULT_HEALTH_CHECK_TIMEOUT,
    customHealthCheck,
  } = options

  return async (_req: Request, res: Response) => {
    let apiConnectivity: Awaited<ReturnType<typeof checkApiConnectivity>> | undefined
    let customHealth: Record<string, unknown> | undefined

    // Check API connectivity if apiPort provided
    if (apiPort) {
      const parsedApiPort = parsePort(apiPort)
      if (parsedApiPort) {
        const portNumber = Number.parseInt(parsedApiPort, 10)
        apiConnectivity = await checkApiConnectivity(apiHost, portNumber, timeout)
      } else {
        apiConnectivity = {
          connected: false,
          latencyMs: null,
          error: {
            code: 'MISSING_CONFIG',
            message: 'API_PORT environment variable not configured',
            category: 'configuration',
            retryable: false,
          },
        }
      }
    }

    // Run custom health check if provided
    if (customHealthCheck) {
      try {
        customHealth = await customHealthCheck()
      } catch (error) {
        customHealth = {
          custom_health_error: error instanceof Error ? error.message : 'Unknown error',
        }
      }
    }

    const result = buildHealthCheckResult(
      { serviceName, version, apiPort, apiHost, timeout },
      apiConnectivity,
      customHealth
    )

    // Set status code based on health
    const statusCode = result.status === 'healthy' ? 200 : result.status === 'degraded' ? 503 : 503

    res.status(statusCode).json(result)
  }
}

/**
 * Create simple health endpoint
 *
 * Returns a minimal health endpoint that just reports healthy.
 * Useful for services that don't need API connectivity checks.
 *
 * @param serviceName - Service name
 * @param version - Optional service version
 * @returns Express request handler
 *
 * @example
 * ```typescript
 * app.get('/health', createSimpleHealthEndpoint('my-service', '1.0.0'))
 * ```
 */
export function createSimpleHealthEndpoint(
  serviceName: string,
  version?: string
): RequestHandler {
  return (_req: Request, res: Response) => {
    const result: HealthCheckResult = {
      status: 'healthy',
      service: serviceName,
      timestamp: new Date().toISOString(),
      readiness: true,
    }

    if (version) {
      result.version = version
    }

    res.json(result)
  }
}
