/**
 * Observability Collector
 *
 * Aggregates data from all subsystems into a unified observability response.
 * Uses dependency injection for testing seams.
 */

import { VERSION } from '../constants';
import { playwrightProvider } from '../playwright';
import { logger, scopedLog, LogContext } from '../utils';
import type {
  ObservabilityDepth,
  ObservabilityResponse,
  ObservabilitySummary,
  ObservabilityComponents,
  BrowserComponent,
  SessionsComponent,
  RecordingComponent,
  CleanupComponent,
  MetricsComponent,
  ConfigComponent,
  DeepDiagnostics,
  ComponentStatus,
  ObservabilityDependencies,
} from './types';

// =============================================================================
// Constants
// =============================================================================

/** Server start time for uptime calculation */
const SERVER_START_TIME = Date.now();

// =============================================================================
// Collector Class
// =============================================================================

/**
 * Observability collector that aggregates data from subsystems.
 *
 * Design principles:
 * - Pure data collection, no side effects
 * - Dependency injection for testability
 * - Graceful degradation if subsystems fail
 */
export class ObservabilityCollector {
  private deps: ObservabilityDependencies;

  constructor(deps: ObservabilityDependencies) {
    this.deps = deps;
  }

  /**
   * Collect observability data at the specified depth.
   *
   * @param depth - How thorough to be (quick, standard, deep)
   * @returns Observability response
   */
  async collect(depth: ObservabilityDepth = 'quick'): Promise<ObservabilityResponse> {
    const startTime = Date.now();

    try {
      // Always collect quick data
      const quick = this.collectQuick();

      if (depth === 'quick') {
        return quick;
      }

      // Standard adds component details and config
      const components = this.collectComponents();
      const config = this.deps.getConfigSummary?.() ?? this.getDefaultConfigSummary();

      if (depth === 'standard') {
        return {
          ...quick,
          depth: 'standard',
          components,
          config,
        };
      }

      // Deep adds diagnostics
      const diagnostics = await this.collectDeepDiagnostics();

      const durationMs = Date.now() - startTime;
      logger.debug(scopedLog(LogContext.HEALTH, 'observability collected'), {
        depth,
        durationMs,
        status: quick.status,
      });

      return {
        ...quick,
        depth: 'deep',
        components,
        config,
        diagnostics,
      };
    } catch (error) {
      logger.error(scopedLog(LogContext.HEALTH, 'observability collection failed'), {
        error: error instanceof Error ? error.message : String(error),
        depth,
      });

      // Return degraded status on error
      return this.createErrorResponse(error, depth);
    }
  }

  /**
   * Collect quick data (always present).
   */
  private collectQuick(): ObservabilityResponse {
    const browserStatus = this.deps.getBrowserStatus();
    const sessionSummary = this.deps.getSessionSummary();

    const summary: ObservabilitySummary = {
      sessions: sessionSummary.total,
      recordings: sessionSummary.active_recordings,
      browser_connected: browserStatus.healthy,
    };

    // Determine overall status
    const status = this.determineOverallStatus(browserStatus.healthy, browserStatus.error);

    // Ready means we can accept new sessions
    const ready = status === 'ok' && browserStatus.healthy;

    return {
      status,
      ready,
      timestamp: new Date().toISOString(),
      version: VERSION,
      uptime_ms: Date.now() - SERVER_START_TIME,
      depth: 'quick',
      summary,
    };
  }

  /**
   * Collect all component health data.
   */
  private collectComponents(): ObservabilityComponents {
    return {
      browser: this.collectBrowserComponent(),
      sessions: this.collectSessionsComponent(),
      recording: this.collectRecordingComponent(),
      cleanup: this.collectCleanupComponent(),
      metrics: this.collectMetricsComponent(),
    };
  }

  /**
   * Collect browser component health.
   */
  private collectBrowserComponent(): BrowserComponent {
    const browserStatus = this.deps.getBrowserStatus();
    const status: ComponentStatus = browserStatus.healthy ? 'healthy' : 'error';

    return {
      status,
      message: browserStatus.healthy
        ? `Chromium ${browserStatus.version || 'ready'}`
        : browserStatus.error || 'Unknown browser issue',
      hint: browserStatus.error ? this.getBrowserErrorHint(browserStatus.error) : undefined,
      version: browserStatus.version,
      connected: browserStatus.healthy,
      provider: playwrightProvider.name,
      capabilities: {
        evaluate_isolated: playwrightProvider.capabilities.evaluateIsolated,
        expose_binding_isolated: playwrightProvider.capabilities.exposeBindingIsolated,
        has_anti_detection: playwrightProvider.capabilities.hasAntiDetection,
      },
    };
  }

  /**
   * Collect sessions component health.
   */
  private collectSessionsComponent(): SessionsComponent {
    const sessionSummary = this.deps.getSessionSummary();
    const utilizationPercent = (sessionSummary.total / sessionSummary.capacity) * 100;

    // Determine status based on capacity utilization
    let status: ComponentStatus = 'healthy';
    let message = `${sessionSummary.active} active, ${sessionSummary.idle} idle of ${sessionSummary.capacity} max`;

    if (utilizationPercent >= 100) {
      status = 'error';
      message = `At capacity: ${sessionSummary.total}/${sessionSummary.capacity} sessions`;
    } else if (utilizationPercent >= 80) {
      status = 'degraded';
      message = `High utilization: ${sessionSummary.total}/${sessionSummary.capacity} sessions (${utilizationPercent.toFixed(0)}%)`;
    }

    return {
      status,
      message,
      hint: status === 'error' ? 'Close unused sessions or increase MAX_SESSIONS' : undefined,
      active: sessionSummary.active,
      idle: sessionSummary.idle,
      total: sessionSummary.total,
      capacity: sessionSummary.capacity,
      active_recordings: sessionSummary.active_recordings,
      idle_timeout_ms: sessionSummary.idle_timeout_ms,
    };
  }

  /**
   * Collect recording component health.
   */
  private collectRecordingComponent(): RecordingComponent {
    const sessionSummary = this.deps.getSessionSummary();
    const recordingStats = this.deps.getRecordingStats?.();

    // Calculate success rate if we have injection stats
    let status: ComponentStatus = 'healthy';
    let message = `${sessionSummary.active_recordings} active recording(s)`;
    let hint: string | undefined;

    if (recordingStats?.injection_stats) {
      const stats = recordingStats.injection_stats;
      if (stats.attempted > 0) {
        const successRate = (stats.successful / stats.attempted) * 100;
        message = `${sessionSummary.active_recordings} recording(s), ${successRate.toFixed(0)}% injection success`;
        if (successRate < 80) {
          status = 'degraded';
        } else if (successRate < 50) {
          status = 'error';
        }
      }
    }

    // Check route handler stats for event flow issues
    if (recordingStats?.route_handler_stats) {
      const routeStats = recordingStats.route_handler_stats;
      if (routeStats.eventsDroppedNoHandler > 0) {
        status = 'error';
        hint = `${routeStats.eventsDroppedNoHandler} events dropped - no handler set. Recording may not be capturing events.`;
      } else if (routeStats.eventsWithErrors > 0) {
        status = 'degraded';
        hint = `${routeStats.eventsWithErrors} events had handler errors.`;
      }
    }

    // Check if event handler is set
    if (recordingStats?.has_event_handler === false && sessionSummary.active_recordings > 0) {
      status = 'error';
      hint = 'Recording is active but no event handler is set. Events will be dropped.';
    }

    if (!hint && status !== 'healthy') {
      hint = 'Check browser console for JavaScript errors or CSP issues';
    }

    return {
      status,
      message,
      hint,
      active_count: sessionSummary.active_recordings,
      injection_stats: recordingStats?.injection_stats,
      route_handler_stats: recordingStats?.route_handler_stats,
      script_version: recordingStats?.script_version,
      has_event_handler: recordingStats?.has_event_handler,
      active_session_id: recordingStats?.active_session_id,
    };
  }

  /**
   * Collect cleanup component health.
   */
  private collectCleanupComponent(): CleanupComponent {
    const cleanupStatus = this.deps.getCleanupStatus();

    return {
      status: 'healthy',
      message: cleanupStatus.is_running ? 'Cleanup in progress' : 'Cleanup idle',
      is_running: cleanupStatus.is_running,
      last_run_at: cleanupStatus.last_run_at?.toISOString(),
      interval_ms: cleanupStatus.interval_ms,
      // next_run_in_ms is calculated if we have last_run_at
      next_run_in_ms: cleanupStatus.last_run_at
        ? Math.max(0, cleanupStatus.interval_ms - (Date.now() - cleanupStatus.last_run_at.getTime()))
        : undefined,
    };
  }

  /**
   * Collect metrics component health.
   */
  private collectMetricsComponent(): MetricsComponent {
    const metricsConfig = this.deps.getMetricsConfig?.() ?? { enabled: false };

    return {
      status: metricsConfig.enabled ? 'healthy' : 'unknown',
      message: metricsConfig.enabled
        ? `Metrics available on port ${metricsConfig.port}`
        : 'Metrics server disabled',
      enabled: metricsConfig.enabled,
      port: metricsConfig.port,
      endpoint: metricsConfig.enabled ? '/metrics' : undefined,
    };
  }

  /**
   * Collect deep diagnostics.
   */
  private async collectDeepDiagnostics(): Promise<DeepDiagnostics> {
    const diagnostics: DeepDiagnostics = {};

    // Run recording diagnostics if available
    if (this.deps.runRecordingDiagnostics) {
      try {
        diagnostics.recording = await this.deps.runRecordingDiagnostics();
      } catch (error) {
        logger.warn(scopedLog(LogContext.HEALTH, 'recording diagnostics failed'), {
          error: error instanceof Error ? error.message : String(error),
        });
      }
    }

    // TODO: Add prometheus_snapshot if needed

    return diagnostics;
  }

  /**
   * Determine overall status from component health.
   */
  private determineOverallStatus(browserHealthy: boolean, browserError?: string): 'ok' | 'degraded' | 'error' {
    if (!browserHealthy) {
      // If browser has a specific error, it's an error state
      // If just not verified yet, it's degraded
      if (browserError && browserError !== 'Browser not yet verified') {
        return 'error';
      }
      return 'degraded';
    }

    return 'ok';
  }

  /**
   * Get actionable hint for browser errors.
   */
  private getBrowserErrorHint(error: string): string | undefined {
    const errorLower = error.toLowerCase();

    if (errorLower.includes('not found') || errorLower.includes('chromium')) {
      return 'Install Chromium: npx playwright install chromium';
    }
    if (errorLower.includes('sandbox') || errorLower.includes('setuid')) {
      return 'Disable sandbox with --no-sandbox or configure kernel.unprivileged_userns_clone=1';
    }
    if (errorLower.includes('memory') || errorLower.includes('oom')) {
      return 'Increase available memory or reduce MAX_SESSIONS';
    }
    if (errorLower.includes('display') || errorLower.includes('x11')) {
      return 'Set HEADLESS=true or configure a virtual display (Xvfb)';
    }
    if (errorLower.includes('permission') || errorLower.includes('denied')) {
      return 'Check file permissions on the browser executable and /tmp directory';
    }
    if (errorLower.includes('timeout')) {
      return 'Browser startup timed out. Check system resources and network connectivity';
    }
    if (error === 'Browser not yet verified') {
      return 'Server is still starting up. Wait a moment and retry.';
    }

    return 'Check server logs for detailed error information';
  }

  /**
   * Get default config summary when not provided.
   */
  private getDefaultConfigSummary(): ConfigComponent {
    return {
      summary: 'Configuration summary not available',
      modified_count: 0,
      by_tier: {
        essential: 0,
        advanced: 0,
        internal: 0,
      },
    };
  }

  /**
   * Create an error response when collection fails.
   */
  private createErrorResponse(error: unknown, depth: ObservabilityDepth): ObservabilityResponse {
    return {
      status: 'error',
      ready: false,
      timestamp: new Date().toISOString(),
      version: VERSION,
      uptime_ms: Date.now() - SERVER_START_TIME,
      depth,
      summary: {
        sessions: 0,
        recordings: 0,
        browser_connected: false,
      },
    };
  }
}

// =============================================================================
// Factory Function
// =============================================================================

/**
 * Create an observability collector with real dependencies.
 * This is the production factory function.
 */
export function createObservabilityCollector(deps: ObservabilityDependencies): ObservabilityCollector {
  return new ObservabilityCollector(deps);
}
