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

/**
 * Route handler statistics for event flow diagnostics.
 */
export interface RouteHandlerStats {
  eventsReceived: number;
  eventsProcessed: number;
  eventsDroppedNoHandler: number;
  eventsWithErrors: number;
  lastEventAt: string | null;
  lastEventType: string | null;
}

/**
 * Browser-side telemetry from recording script.
 */
export interface BrowserTelemetry {
  eventsDetected: number;
  eventsCaptured: number;
  eventsSent: number;
  eventsSendSuccess: number;
  eventsSendFailed: number;
  lastEventAt: number | null;
  lastEventType: string | null;
  lastError: string | null;
}

/**
 * Result of real click simulation test.
 */
export interface RealClickTestResult {
  attempted: boolean;
  clickDetected: boolean;
  eventSent: boolean;
  telemetryBefore?: BrowserTelemetry;
  telemetryAfter?: BrowserTelemetry;
  error?: string;
}

export interface RecordingComponent extends ComponentHealth {
  script_version?: string;
  active_count: number;
  /** Whether an event handler is currently set */
  has_event_handler?: boolean;
  injection_stats?: {
    attempted: number;
    successful: number;
    failed: number;
    skipped: number;
    total: number;
    methods?: {
      head: number;
      HEAD: number;
      doctype: number;
      prepend: number;
    };
  };
  /** Route handler statistics for event flow diagnostics */
  route_handler_stats?: RouteHandlerStats;
  /** ID of the first actively recording session (for debug endpoint) */
  active_session_id?: string;
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

export type ConfigTier = 'essential' | 'advanced' | 'internal';
export type ConfigDataType = 'boolean' | 'integer' | 'float' | 'string' | 'enum';

export interface ModifiedConfigOption {
  env_var: string;
  tier: ConfigTier;
  description?: string;
  current_value: string;
  default_value?: string;
}

export interface ConfigOption {
  env_var: string;
  tier: ConfigTier;
  description: string;
  current_value: string;
  default_value: string;
  is_modified: boolean;
  /** Data type for UI rendering and validation */
  data_type: ConfigDataType;
  /** For numeric types: minimum allowed value */
  min?: number;
  /** For numeric types: maximum allowed value */
  max?: number;
  /** For enum types: allowed values */
  enum_values?: string[];
  /** Whether this option can be changed at runtime without restart */
  editable: boolean;
}

/** Result of a config update request */
export interface ConfigUpdateResult {
  success: boolean;
  env_var?: string;
  new_value?: string;
  previous_value?: string;
  error?: string;
  validation?: {
    data_type: ConfigDataType;
    min?: number;
    max?: number;
    enum_values?: string[];
  };
}

/** Request to update a config value */
export interface ConfigUpdateRequest {
  value: string;
}

/** Runtime config state */
export interface RuntimeConfigState {
  overrides: Record<string, {
    value: string;
    setAt: string;
    previousValue?: string;
  }>;
  last_updated_at: string | null;
  total_overrides: number;
}

export interface ConfigComponent {
  summary: string;
  modified_count: number;
  total_count?: number;
  by_tier?: {
    essential: number;
    advanced: number;
    internal: number;
  };
  modified_options?: ModifiedConfigOption[];
  all_options?: {
    essential: ConfigOption[];
    advanced: ConfigOption[];
    internal: ConfigOption[];
  };
}

/**
 * Extended event flow test result with detailed diagnostics.
 * Mirrors the backend EventFlowTestResult type.
 */
export interface EventFlowTestResult {
  /** Whether the overall test passed */
  passed: boolean;
  /** Whether the console test event was sent */
  eventSent: boolean;
  /** Whether the console test event was received by Playwright */
  eventReceived: boolean;
  /** Round-trip latency for console event (ms) */
  latencyMs?: number;
  /** Error message if test failed */
  error?: string;
  /** Page URL at time of test */
  pageUrl?: string;
  /** Whether page is a valid HTTP(S) page */
  pageValid?: boolean;
  /** Script status from CDP evaluation in MAIN context */
  scriptStatus?: {
    loaded: boolean;
    ready: boolean;
    inMainContext: boolean;
    handlersCount: number;
    version?: string | null;
  };
  /** Result of testing the actual fetch event path */
  fetchTest?: {
    sent: boolean;
    status: number | null;
    error: string | null;
  };
  /** Console capture statistics during test */
  consoleCapture?: {
    count: number;
    hasErrors: boolean;
    samples?: Array<{ type: string; text: string }>;
  };
  /** Browser-side telemetry from recording script */
  browserTelemetry?: BrowserTelemetry;
  /** Result of real click simulation test */
  realClickTest?: RealClickTestResult;
}

/**
 * Status of a single diagnostic check.
 */
export type DiagnosticCheckStatus = 'passed' | 'failed' | 'warning' | 'skipped';

/**
 * A single diagnostic check that was performed.
 */
export interface DiagnosticCheck {
  /** Unique identifier for the check */
  id: string;
  /** Human-readable name of the check */
  name: string;
  /** Category for grouping */
  category: 'script' | 'context' | 'events' | 'telemetry';
  /** Status of this check */
  status: DiagnosticCheckStatus;
  /** Brief description of what was checked */
  description: string;
  /** Value or result of the check (for display) */
  value?: string;
  /** Additional details if failed or warning */
  details?: string;
}

export interface RecordingDiagnostics {
  ready: boolean;
  timestamp: string;
  durationMs: number;
  level: 'quick' | 'standard' | 'full';
  /** All checks performed with their status */
  checks: DiagnosticCheck[];
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
  /** Event flow test result (FULL level only) */
  eventFlowTest?: EventFlowTestResult;
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
