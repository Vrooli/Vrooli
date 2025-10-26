/**
 * Centralized health check type definitions
 * Single source of truth for all health-related API responses
 */

export interface HealthMetrics {
  uptime_seconds?: number;
  uptime_source?: 'api' | 'orchestrator';
  memory_alloc_mb?: number;
  memory_sys_mb?: number;
  goroutines?: number;
  cpu_count?: number;
  gc_runs?: number;
  cpu_usage_percent?: number;
  memory_usage_mb?: number;
  websocket_clients?: number;
  [key: string]: unknown;
}

export interface HealthError {
  code?: string;
  message?: string;
  category?: 'network' | 'authentication' | 'resource' | 'configuration' | 'process';
  retryable?: boolean;
}

export interface HealthDependency {
  connected: boolean;
  latency_ms?: number;
  error?: HealthError | null;
  [key: string]: unknown;
}

export interface ApiHealthResponse {
  status?: string;
  service?: string;
  timestamp?: string;
  readiness?: boolean;
  version?: string;
  dependencies?: Record<string, HealthDependency>;
  metrics?: HealthMetrics;
  [key: string]: unknown;
}

export interface UiHealthResponse {
  status?: string;
  service?: string;
  readiness?: boolean;
  api_connectivity?: {
    connected?: boolean;
    api_url?: string;
    latency_ms?: number | null;
    error?: HealthError | null;
  };
  websocket?: {
    connected?: boolean;
    active_connections?: number;
    error?: HealthError | null;
  };
  metrics?: HealthMetrics;
  [key: string]: unknown;
}

/**
 * Lightweight system status response for quick polling
 * Optimized to avoid expensive full data fetches
 */
export interface SystemStatusResponse {
  status: 'healthy' | 'degraded' | 'unhealthy';
  app_count: number;
  resource_count: number;
  uptime_seconds: number | null;
  checked_at: string;
}

export interface PreviewHealthCheckEntry {
  id: string;
  name: string;
  status: 'pass' | 'warn' | 'fail';
  endpoint?: string | null;
  latency_ms?: number | null;
  latencyMs?: number | null; // Alias for backwards compatibility
  message?: string | null;
  code?: string | null;
  response?: string | null;
}

export interface PreviewHealthDiagnosticsResponse {
  app_id: string;
  app_name?: string | null;
  scenario?: string | null;
  captured_at?: string | null;
  ports?: Record<string, number> | null;
  checks?: PreviewHealthCheckEntry[] | null;
  errors?: (string | null | undefined)[] | null;
}

export interface PreviewHealthDiagnosticsResult {
  appId: string | null;
  appName: string | null;
  scenario: string | null;
  capturedAt: string | null;
  checks: PreviewHealthCheckEntry[];
  errors: string[];
  ports: Record<string, number>;
}
