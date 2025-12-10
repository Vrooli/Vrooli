import { Registry, Counter, Histogram, Gauge } from 'prom-client';

/**
 * Prometheus metrics for observability.
 *
 * Key metrics for operators to monitor:
 * - playwright_driver_sessions: Current session load
 * - playwright_driver_instruction_duration_ms: Instruction performance
 * - playwright_driver_instruction_errors_total: Error rates by type
 * - playwright_driver_recording_actions_total: Recording activity
 */
export class Metrics {
  private registry: Registry;

  public sessionCount: Gauge<'state'>;
  public instructionDuration: Histogram<'type' | 'success'>;
  public instructionErrors: Counter<'type' | 'error_kind'>;
  public screenshotSize: Histogram<never>;
  public sessionDuration: Histogram<never>;
  public cleanupFailures: Counter<'operation'>;
  /** Total recorded actions across all sessions */
  public recordingActionsTotal: Counter<never>;
  /** Active recording sessions */
  public recordingSessionsActive: Gauge<never>;
  /** Telemetry capture failures (screenshot, DOM) */
  public telemetryFailures: Counter<'type'>;
  /** Recording callback streaming failures */
  public recordingCallbackFailures: Counter<'reason'>;
  /** Circuit breaker state changes */
  public circuitBreakerStateChanges: Counter<'session_id' | 'state'>;

  constructor() {
    this.registry = new Registry();

    this.sessionCount = new Gauge({
      name: 'playwright_driver_sessions',
      help: 'Current number of sessions by state (active, total, idle)',
      labelNames: ['state'] as const,
      registers: [this.registry],
    });

    this.instructionDuration = new Histogram({
      name: 'playwright_driver_instruction_duration_ms',
      help: 'Instruction execution duration in milliseconds',
      labelNames: ['type', 'success'] as const,
      buckets: [10, 50, 100, 250, 500, 1000, 2500, 5000, 10000],
      registers: [this.registry],
    });

    this.instructionErrors = new Counter({
      name: 'playwright_driver_instruction_errors_total',
      help: 'Total number of instruction errors by type and error kind',
      labelNames: ['type', 'error_kind'] as const,
      registers: [this.registry],
    });

    this.screenshotSize = new Histogram({
      name: 'playwright_driver_screenshot_size_bytes',
      help: 'Screenshot size in bytes',
      buckets: [10000, 50000, 100000, 250000, 500000, 1000000],
      registers: [this.registry],
    });

    this.sessionDuration = new Histogram({
      name: 'playwright_driver_session_duration_ms',
      help: 'Session lifetime duration in milliseconds',
      buckets: [1000, 5000, 10000, 30000, 60000, 300000, 600000],
      registers: [this.registry],
    });

    this.cleanupFailures = new Counter({
      name: 'playwright_driver_cleanup_failures_total',
      help: 'Total number of cleanup failures by operation type (page_close, context_close, tracing_stop, browser_close, recording_stop)',
      labelNames: ['operation'] as const,
      registers: [this.registry],
    });

    this.recordingActionsTotal = new Counter({
      name: 'playwright_driver_recording_actions_total',
      help: 'Total number of user actions recorded across all sessions',
      registers: [this.registry],
    });

    this.recordingSessionsActive = new Gauge({
      name: 'playwright_driver_recording_sessions_active',
      help: 'Current number of sessions with active recording',
      registers: [this.registry],
    });

    this.telemetryFailures = new Counter({
      name: 'playwright_driver_telemetry_failures_total',
      help: 'Total number of telemetry capture failures (screenshot, dom)',
      labelNames: ['type'] as const,
      registers: [this.registry],
    });

    this.recordingCallbackFailures = new Counter({
      name: 'playwright_driver_recording_callback_failures_total',
      help: 'Total number of recording callback streaming failures by reason (network, timeout, http_error)',
      labelNames: ['reason'] as const,
      registers: [this.registry],
    });

    this.circuitBreakerStateChanges = new Counter({
      name: 'playwright_driver_circuit_breaker_state_changes_total',
      help: 'Circuit breaker state changes for recording callbacks (opened, closed)',
      labelNames: ['session_id', 'state'] as const,
      registers: [this.registry],
    });
  }

  getRegistry(): Registry {
    return this.registry;
  }

  async getMetrics(): Promise<string> {
    return this.registry.metrics();
  }
}

// Default metrics instance
export const metrics = new Metrics();
