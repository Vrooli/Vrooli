/**
 * Server-side utilities for @vrooli/api-base
 *
 * Import from '@vrooli/api-base/server' to access server-side functionality.
 *
 * @example
 * ```typescript
 * import { createScenarioServer } from '@vrooli/api-base/server'
 *
 * const app = createScenarioServer({
 *   uiPort: process.env.UI_PORT || 3000,
 *   apiPort: process.env.API_PORT || 8080,
 *   distDir: './dist',
 *   serviceName: 'my-scenario'
 * })
 *
 * app.listen(process.env.UI_PORT || 3000)
 * ```
 *
 * @module server
 */

// Metadata injection
export {
  buildProxyMetadata,
  injectProxyMetadata,
  injectScenarioConfig,
} from './inject.js'

// Proxy middleware
export {
  proxyToApi,
  createProxyMiddleware,
} from './proxy.js'

// Config endpoint
export {
  buildScenarioConfig,
  createConfigEndpoint,
  createDynamicConfigEndpoint,
} from './config.js'

// Health endpoint
export {
  createHealthEndpoint,
  createSimpleHealthEndpoint,
} from './health.js'

// Full server template
export {
  createScenarioServer,
  startScenarioServer,
} from './template.js'

// Re-export types that are useful for server code
export type {
  ProxyInfo,
  ProxyMetadataOptions,
  ScenarioConfig,
  PortEntry,
  ProxyOptions,
  ConfigEndpointOptions,
  HealthOptions,
  HealthCheckResult,
  ServerTemplateOptions,
} from '../shared/types.js'
