import { Registry, Counter, Histogram, Gauge } from 'prom-client';

export class Metrics {
  private registry: Registry;

  public sessionCount: Gauge<'state'>;
  public instructionDuration: Histogram<'type' | 'success'>;
  public instructionErrors: Counter<'type' | 'error_kind'>;
  public screenshotSize: Histogram<never>;
  public sessionDuration: Histogram<never>;
  public cleanupFailures: Counter<'operation'>;

  constructor() {
    this.registry = new Registry();

    this.sessionCount = new Gauge({
      name: 'playwright_driver_sessions',
      help: 'Current number of sessions by state',
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
      help: 'Total number of instruction errors',
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
      help: 'Total number of cleanup failures by operation type (page_close, context_close, tracing_stop, browser_close)',
      labelNames: ['operation'] as const,
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
