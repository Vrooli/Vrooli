/**
 * Observability Module
 *
 * Unified system health, monitoring, and diagnostics.
 *
 * ## Quick Start
 *
 * ```typescript
 * import { createObservabilityCollector, getObservabilityCache } from './observability';
 *
 * // Create collector with dependencies
 * const collector = createObservabilityCollector({
 *   getSessionSummary: () => sessionManager.getSessionSummary(),
 *   getBrowserStatus: () => sessionManager.getBrowserStatus(),
 *   getCleanupStatus: () => cleanup.getStatus(),
 * });
 *
 * // Collect quick data (for polling)
 * const quick = await collector.collect('quick');
 *
 * // Collect with caching
 * const cache = getObservabilityCache();
 * let response = cache.get('standard');
 * if (!response) {
 *   response = await collector.collect('standard');
 *   cache.set('standard', response);
 * }
 * ```
 *
 * ## API Endpoints
 *
 * - `GET /observability` - Main observability endpoint
 * - `GET /observability?depth=quick|standard|deep` - Control response depth
 * - `POST /observability/refresh` - Force cache refresh
 * - `POST /observability/diagnostics/run` - Run specific diagnostics
 *
 * @module observability
 */

// Types
export type {
  // Core types
  ObservabilityDepth,
  ComponentStatus,
  ComponentHealth,
  // Component types
  BrowserComponent,
  SessionsComponent,
  RecordingComponent,
  CleanupComponent,
  MetricsComponent,
  ObservabilityComponents,
  // Config types
  ConfigTier,
  ModifiedConfigOption,
  ConfigComponent,
  // Diagnostics types
  DeepDiagnostics,
  // Response types
  ObservabilitySummary,
  ObservabilityResponse,
  // Run types
  DiagnosticRunRequest,
  DiagnosticRunResponse,
  // Dependency injection types
  SessionSummary,
  BrowserStatusSummary,
  CleanupStatus,
  ObservabilityDependencies,
} from './types';

// Collector
export { ObservabilityCollector, createObservabilityCollector } from './collector';

// Cache
export {
  ObservabilityCache,
  getObservabilityCache,
  resetObservabilityCache,
  DEFAULT_CACHE_TTL_MS,
} from './cache';

// Routes
export {
  handleObservability,
  handleObservabilityRefresh,
  handleDiagnosticsRun,
  handleSessionList,
  handleCleanupRun,
  handleMetrics,
  handleConfigUpdate,
  handleConfigReset,
  handleConfigRuntime,
  type ObservabilityRouteDependencies,
} from './route';
