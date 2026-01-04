/**
 * Observability Types
 *
 * Central type definitions for the unified observability system.
 * These types define the API contract between the backend and frontend.
 */

import type { InjectionStats } from '../recording/context-initializer';
import type { RecordingDiagnosticResult } from '../recording/diagnostics';

// =============================================================================
// Core Types
// =============================================================================

/**
 * Observability depth levels.
 * Controls how thorough the response is.
 */
export type ObservabilityDepth = 'quick' | 'standard' | 'deep';

/**
 * Component health status.
 */
export type ComponentStatus = 'healthy' | 'degraded' | 'error' | 'unknown';

/**
 * Base interface for all component health checks.
 */
export interface ComponentHealth {
  /** Overall status of this component */
  status: ComponentStatus;
  /** Human-readable status message */
  message: string;
  /** Actionable hint when status is not healthy */
  hint?: string;
  /** How long this check took in ms */
  check_duration_ms?: number;
}

// =============================================================================
// Component Health Types
// =============================================================================

/**
 * Browser component health.
 */
export interface BrowserComponent extends ComponentHealth {
  /** Chromium version (e.g., "120.0.6099.109") */
  version?: string;
  /** Whether the browser process is connected */
  connected: boolean;
  /** Playwright provider being used */
  provider: string;
  /** Provider capabilities */
  capabilities?: {
    evaluate_isolated: boolean;
    expose_binding_isolated: boolean;
    has_anti_detection: boolean;
  };
}

/**
 * Sessions component health.
 */
export interface SessionsComponent extends ComponentHealth {
  /** Number of sessions with recent activity */
  active: number;
  /** Number of sessions exceeding idle threshold */
  idle: number;
  /** Total sessions (active + idle) */
  total: number;
  /** Maximum concurrent sessions allowed */
  capacity: number;
  /** Sessions currently in recording mode */
  active_recordings: number;
  /** Idle timeout in milliseconds */
  idle_timeout_ms: number;
}

/**
 * Recording component health.
 */
export interface RecordingComponent extends ComponentHealth {
  /** Script injection statistics */
  injection_stats?: InjectionStats;
  /** Recording script version */
  script_version?: string;
  /** Number of active recording sessions */
  active_count: number;
}

/**
 * Cleanup task component health.
 */
export interface CleanupComponent extends ComponentHealth {
  /** Whether cleanup task is currently running */
  is_running: boolean;
  /** Timestamp of last cleanup run */
  last_run_at?: string;
  /** Milliseconds until next scheduled cleanup */
  next_run_in_ms?: number;
  /** Cleanup interval in milliseconds */
  interval_ms: number;
}

/**
 * Metrics server component health.
 */
export interface MetricsComponent extends ComponentHealth {
  /** Whether metrics server is enabled */
  enabled: boolean;
  /** Port the metrics server is listening on */
  port?: number;
  /** Endpoint path for Prometheus metrics */
  endpoint?: string;
}

/**
 * All component health checks grouped together.
 */
export interface ObservabilityComponents {
  browser: BrowserComponent;
  sessions: SessionsComponent;
  recording: RecordingComponent;
  cleanup: CleanupComponent;
  metrics: MetricsComponent;
}

// =============================================================================
// Configuration Types
// =============================================================================

/**
 * Configuration tier for categorizing options.
 */
export type ConfigTier = 'essential' | 'advanced' | 'internal';

/**
 * A single modified configuration option.
 */
export interface ModifiedConfigOption {
  /** Environment variable name */
  env_var: string;
  /** Configuration tier */
  tier: ConfigTier;
  /** Description of the option */
  description: string;
  /** Current value (as string) */
  current_value: string;
  /** Default value (as string) */
  default_value: string;
}

/**
 * A single configuration option (all options view).
 */
export interface ConfigOption {
  /** Environment variable name */
  env_var: string;
  /** Configuration tier */
  tier: ConfigTier;
  /** Description of the option */
  description: string;
  /** Current value (as string) */
  current_value: string;
  /** Default value (as string) */
  default_value: string;
  /** Whether this option is modified from its default */
  is_modified: boolean;
}

/**
 * Configuration summary for observability.
 */
export interface ConfigComponent {
  /** Human-readable summary */
  summary: string;
  /** Number of options modified from defaults */
  modified_count: number;
  /** Total number of configuration options */
  total_count: number;
  /** Count by tier */
  by_tier: {
    essential: number;
    advanced: number;
    internal: number;
  };
  /** List of modified options (only in standard+ depth) */
  modified_options?: ModifiedConfigOption[];
  /** List of all options organized by tier (only in standard+ depth) */
  all_options?: {
    essential: ConfigOption[];
    advanced: ConfigOption[];
    internal: ConfigOption[];
  };
}

// =============================================================================
// Deep Diagnostics Types
// =============================================================================

/**
 * Deep diagnostics results.
 * Only included when depth=deep.
 */
export interface DeepDiagnostics {
  /** Recording diagnostics result */
  recording?: RecordingDiagnosticResult;
  /** Prometheus metrics snapshot */
  prometheus_snapshot?: Record<string, number>;
}

// =============================================================================
// Main Response Types
// =============================================================================

/**
 * Quick summary for lightweight polling.
 */
export interface ObservabilitySummary {
  /** Total session count */
  sessions: number;
  /** Active recording count */
  recordings: number;
  /** Whether browser is connected */
  browser_connected: boolean;
}

/**
 * Main observability response.
 * Structure varies based on depth level.
 */
export interface ObservabilityResponse {
  // === Always present (quick+) ===

  /** Overall status based on component health */
  status: 'ok' | 'degraded' | 'error';
  /** Whether the system is ready to accept new sessions */
  ready: boolean;
  /** ISO 8601 timestamp of this response */
  timestamp: string;
  /** Driver version */
  version: string;
  /** Server uptime in milliseconds */
  uptime_ms: number;
  /** Depth level of this response */
  depth: ObservabilityDepth;
  /** Quick summary stats */
  summary: ObservabilitySummary;

  // === Standard+ only ===

  /** Per-component health details */
  components?: ObservabilityComponents;
  /** Configuration summary */
  config?: ConfigComponent;

  // === Deep only ===

  /** Deep diagnostics results */
  diagnostics?: DeepDiagnostics;

  // === Cache metadata ===

  /** Whether this response was served from cache */
  cached?: boolean;
  /** When the cache was populated */
  cached_at?: string;
  /** Cache time-to-live in milliseconds */
  cache_ttl_ms?: number;
}

// =============================================================================
// Diagnostic Run Types
// =============================================================================

/**
 * Request to run diagnostics manually.
 */
export interface DiagnosticRunRequest {
  /** Type of diagnostic to run */
  type: 'recording' | 'browser' | 'all';
  /** Optional session ID for session-specific diagnostics */
  session_id?: string;
  /** Diagnostic options */
  options?: {
    /** Diagnostic level (quick, standard, full) */
    level?: 'quick' | 'standard' | 'full';
    /** Timeout in milliseconds */
    timeout_ms?: number;
  };
}

/**
 * Response from a diagnostic run.
 */
export interface DiagnosticRunResponse {
  /** When the diagnostic started */
  started_at: string;
  /** When the diagnostic completed */
  completed_at: string;
  /** How long the diagnostic took */
  duration_ms: number;
  /** Results of the diagnostic run */
  results: DeepDiagnostics;
}

// =============================================================================
// Collector Interface (for dependency injection)
// =============================================================================

/**
 * Session summary for observability.
 * This is what SessionManager provides to the collector.
 */
export interface SessionSummary {
  total: number;
  active: number;
  idle: number;
  active_recordings: number;
  idle_timeout_ms: number;
  capacity: number;
}

/**
 * Browser status for observability.
 * This is what BrowserManager provides to the collector.
 */
export interface BrowserStatusSummary {
  healthy: boolean;
  version?: string;
  error?: string;
}

/**
 * Cleanup status for observability.
 * This is what SessionCleanup provides to the collector.
 */
export interface CleanupStatus {
  is_running: boolean;
  last_run_at?: Date;
  interval_ms: number;
}

/**
 * Recording statistics for observability.
 * Aggregated from all sessions with recording initializers.
 */
export interface RecordingStats {
  /** Script version hash or identifier */
  script_version: string;
  /** Aggregated injection statistics */
  injection_stats: InjectionStats;
}

/**
 * Dependencies for the observability collector.
 * Allows dependency injection for testing.
 */
export interface ObservabilityDependencies {
  /** Get session summary */
  getSessionSummary: () => SessionSummary;
  /** Get browser status */
  getBrowserStatus: () => BrowserStatusSummary;
  /** Get cleanup status */
  getCleanupStatus: () => CleanupStatus;
  /** Get metrics config (optional) */
  getMetricsConfig?: () => { enabled: boolean; port?: number };
  /** Get recording stats (optional) */
  getRecordingStats?: () => RecordingStats | undefined;
  /** Run recording diagnostics (optional, for deep level) */
  runRecordingDiagnostics?: () => Promise<RecordingDiagnosticResult>;
  /** Get config summary */
  getConfigSummary?: () => ConfigComponent;
}
