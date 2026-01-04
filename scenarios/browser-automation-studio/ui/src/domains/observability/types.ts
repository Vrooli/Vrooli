/**
 * Observability Types
 *
 * Shared types for the observability domain.
 * These mirror the backend types from playwright-driver/src/observability/types.ts
 */

export type ObservabilityDepth = 'quick' | 'standard' | 'deep';
export type ComponentStatus = 'healthy' | 'degraded' | 'error';

export interface ComponentHealth {
  status: ComponentStatus;
  message?: string;
  hint?: string;
}

export interface BrowserComponent extends ComponentHealth {
  version?: string;
  connected: boolean;
  provider?: string;
  capabilities?: {
    evaluate_isolated?: boolean;
    expose_binding_isolated?: boolean;
    has_anti_detection?: boolean;
  };
}

export interface SessionsComponent extends ComponentHealth {
  active: number;
  idle: number;
  total: number;
  capacity: number;
  idle_timeout_ms: number;
  active_recordings: number;
}

export interface RecordingComponent extends ComponentHealth {
  script_version?: string;
  injection_stats?: {
    successful: number;
    failed: number;
    total: number;
  };
}

export interface CleanupComponent {
  is_running: boolean;
  last_run_at?: string;
  next_run_in_ms?: number;
  interval_ms: number;
}

export interface MetricsComponent {
  enabled: boolean;
  port?: number;
  endpoint?: string;
}

export interface ObservabilityComponents {
  browser: BrowserComponent;
  sessions: SessionsComponent;
  recording: RecordingComponent;
  cleanup: CleanupComponent;
  metrics: MetricsComponent;
}

export type ConfigTier = 'essential' | 'advanced' | 'expert';

export interface ModifiedConfigOption {
  env_var: string;
  tier: ConfigTier;
  current_value: unknown;
  default_value?: unknown;
}

export interface ConfigComponent {
  summary: string;
  modified_count: number;
  modified_options?: ModifiedConfigOption[];
}

export interface RecordingDiagnostics {
  ready: boolean;
  timestamp: string;
  durationMs: number;
  level: 'quick' | 'standard' | 'full';
  issues: Array<{
    severity: 'error' | 'warning' | 'info';
    category: string;
    message: string;
    suggestion?: string;
    docs_link?: string;
  }>;
  provider?: {
    name: string;
    evaluateIsolated: boolean;
    exposeBindingIsolated: boolean;
  };
}

export interface DeepDiagnostics {
  recording?: RecordingDiagnostics;
  prometheus_snapshot?: Record<string, number>;
}

export interface ObservabilitySummary {
  sessions: number;
  recordings: number;
  browser_connected: boolean;
}

export interface ObservabilityResponse {
  // Always present (quick+)
  status: 'ok' | 'degraded' | 'error';
  ready: boolean;
  timestamp: string;
  version: string;
  uptime_ms: number;
  depth: ObservabilityDepth;
  summary: ObservabilitySummary;

  // standard+ only
  components?: ObservabilityComponents;
  config?: ConfigComponent;

  // deep only
  diagnostics?: DeepDiagnostics;

  // Cache metadata
  cached?: boolean;
  cached_at?: string;
  cache_ttl_ms?: number;
}

export interface DiagnosticRunRequest {
  type: 'recording' | 'browser' | 'all';
  session_id?: string;
  options?: {
    level?: 'quick' | 'standard' | 'full';
    timeout_ms?: number;
  };
}

export interface DiagnosticRunResponse {
  started_at: string;
  completed_at: string;
  duration_ms: number;
  results: {
    recording?: RecordingDiagnostics;
  };
}

/**
 * Metrics types for the JSON metrics endpoint.
 */
export interface MetricValue {
  labels: Record<string, string>;
  value: number;
}

export interface MetricData {
  type: string;
  help: string;
  values: MetricValue[];
}

export interface MetricsResponse {
  summary: {
    total_metrics: number;
    timestamp: string;
    config: {
      enabled: boolean;
      port?: number;
    };
  };
  metrics: Record<string, MetricData>;
}
