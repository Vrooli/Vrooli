/**
 * Config endpoint middleware
 *
 * Creates an Express endpoint that serves runtime configuration to the client.
 * This allows production bundles to discover API/WS URLs dynamically.
 */

import type { Request, Response, RequestHandler } from 'express'
import type { ConfigEndpointOptions, ScenarioConfig } from '../shared/types.js'
import { LOOPBACK_HOST, DEFAULT_API_SUFFIX, DEFAULT_WS_SUFFIX } from '../shared/constants.js'
import { parsePort } from '../shared/utils.js'

/**
 * Build scenario configuration object
 *
 * Constructs the runtime config that will be returned by the /config endpoint.
 *
 * @param options - Configuration options
 * @returns Scenario configuration object
 *
 * @example
 * ```typescript
 * const config = buildScenarioConfig({
 *   apiPort: 8080,
 *   wsPort: 8081,
 *   uiPort: 3000,
 *   serviceName: 'my-scenario',
 *   version: '1.0.0'
 * })
 * ```
 */
export function buildScenarioConfig(options: ConfigEndpointOptions): ScenarioConfig {
  const {
    apiPort,
    apiHost = LOOPBACK_HOST,
    wsPort,
    wsHost,
    uiPort,
    version,
    serviceName,
    additionalConfig = {},
  } = options

  // Parse ports
  const parsedApiPort = parsePort(apiPort)
  const parsedWsPort = parsePort(wsPort) || parsedApiPort
  const parsedUiPort = parsePort(uiPort)

  if (!parsedApiPort) {
    throw new Error('Invalid API port configuration')
  }
  if (!parsedUiPort) {
    throw new Error('Invalid UI port configuration')
  }

  const effectiveApiHost = apiHost || LOOPBACK_HOST
  const effectiveWsHost = wsHost || effectiveApiHost

  // Build URLs
  const apiUrl = `http://${effectiveApiHost}:${parsedApiPort}${DEFAULT_API_SUFFIX}`
  const wsUrl = `ws://${effectiveWsHost}:${parsedWsPort}${DEFAULT_WS_SUFFIX}`

  const config: ScenarioConfig = {
    apiUrl,
    wsUrl,
    apiPort: parsedApiPort,
    wsPort: parsedWsPort,
    uiPort: parsedUiPort,
    ...additionalConfig,
  }

  if (version) {
    config.version = version
  }

  if (serviceName) {
    config.service = serviceName
  }

  return config
}

/**
 * Create config endpoint middleware
 *
 * Returns an Express request handler that serves runtime configuration at /config.
 * The client can fetch this to discover API/WS URLs in production mode.
 *
 * @param options - Configuration endpoint options
 * @returns Express request handler
 *
 * @example
 * ```typescript
 * const configHandler = createConfigEndpoint({
 *   apiPort: process.env.API_PORT || '8080',
 *   uiPort: process.env.UI_PORT || '3000',
 *   serviceName: 'my-scenario',
 *   version: '1.0.0',
 *   cors: true,
 *   includeTimestamp: true,
 *   cacheControl: 'no-cache, no-store, must-revalidate'
 * })
 *
 * app.get('/config', configHandler)
 * ```
 */
export function createConfigEndpoint(options: ConfigEndpointOptions): RequestHandler {
  const {
    configBuilder,
    cors = false,
    includeTimestamp = false,
    cacheControl = true,
  } = options

  // Return handler that serves the config
  return (_req: Request, res: Response) => {
    try {
      // Use custom builder if provided, otherwise use default
      let config: ScenarioConfig
      if (configBuilder) {
        config = configBuilder()
      } else {
        config = buildScenarioConfig(options)
      }

      // Add timestamp if requested
      if (includeTimestamp) {
        config = { ...config, timestamp: Date.now() }
      }

      // Set CORS headers if enabled
      if (cors) {
        res.header('Access-Control-Allow-Origin', '*')
        res.header('Access-Control-Allow-Methods', 'GET, OPTIONS')
        res.header('Access-Control-Allow-Headers', 'Content-Type')
      }

      // Set cache control headers
      if (cacheControl) {
        const cacheValue = typeof cacheControl === 'string'
          ? cacheControl
          : 'no-cache, no-store, must-revalidate'
        res.header('Cache-Control', cacheValue)
      }

      res.json(config)
    } catch (error) {
      res.status(500).json({
        error: 'Failed to build configuration',
        details: error instanceof Error ? error.message : 'Unknown error',
      })
    }
  }
}

/**
 * Create config endpoint with dynamic options
 *
 * Returns an Express request handler that builds config dynamically on each request.
 * Useful when configuration can change at runtime (e.g., reading from environment).
 *
 * @param optionsBuilder - Function that returns config options
 * @returns Express request handler
 *
 * @example
 * ```typescript
 * const configHandler = createDynamicConfigEndpoint(() => ({
 *   apiPort: process.env.API_PORT || '8080',
 *   uiPort: process.env.UI_PORT || '3000',
 *   additionalConfig: {
 *     timestamp: Date.now()
 *   }
 * }))
 *
 * app.get('/config', configHandler)
 * ```
 */
export function createDynamicConfigEndpoint(
  optionsBuilder: () => ConfigEndpointOptions
): RequestHandler {
  return (_req: Request, res: Response) => {
    try {
      const options = optionsBuilder()
      const config = buildScenarioConfig(options)
      res.json(config)
    } catch (error) {
      res.status(500).json({
        error: 'Failed to build configuration',
        details: error instanceof Error ? error.message : 'Unknown error',
      })
    }
  }
}
