/**
 * Observability Domain
 *
 * Unified health monitoring, diagnostics, and system status.
 *
 * @example
 * ```tsx
 * import { useObservability, useRefreshDiagnostics } from '@/domains/observability';
 *
 * function DiagnosticsPanel() {
 *   const { data, isLoading, refetch } = useObservability({ depth: 'standard' });
 *   const { runDiagnostics, isRunning } = useRefreshDiagnostics();
 *
 *   return (
 *     <div>
 *       <h2>System Status: {data?.status}</h2>
 *       <button onClick={refetch}>Refresh</button>
 *       <button onClick={() => runDiagnostics({ type: 'all' })}>
 *         {isRunning ? 'Running...' : 'Run Full Diagnostics'}
 *       </button>
 *     </div>
 *   );
 * }
 * ```
 */

// Types
export type {
  ObservabilityDepth,
  ComponentStatus,
  ComponentHealth,
  BrowserComponent,
  SessionsComponent,
  RecordingComponent,
  CleanupComponent,
  MetricsComponent,
  ObservabilityComponents,
  ConfigTier,
  ModifiedConfigOption,
  ConfigOption,
  ConfigComponent,
  RecordingDiagnostics,
  DeepDiagnostics,
  ObservabilitySummary,
  ObservabilityResponse,
  DiagnosticRunRequest,
  DiagnosticRunResponse,
  MetricValue,
  MetricData,
  MetricsResponse,
} from './types';

// Hooks
export { useObservability } from './hooks/useObservability';
export {
  useRefreshDiagnostics,
  useRefreshCache,
  useRunCleanup,
  useSessionList,
  type CleanupRunResponse,
  type SessionInfo,
  type SessionListResponse,
} from './hooks/useRefreshDiagnostics';
export { useMetrics } from './hooks/useMetrics';
