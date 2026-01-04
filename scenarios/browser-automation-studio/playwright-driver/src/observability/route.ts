/**
 * Observability Route Handler
 *
 * Provides unified observability endpoint for health, monitoring, and diagnostics.
 *
 * ## Endpoints
 *
 * - `GET /observability` - Get observability data
 * - `POST /observability/refresh` - Force cache refresh
 * - `POST /observability/diagnostics/run` - Run specific diagnostics
 */

import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { SessionCleanup } from '../session/cleanup';
import type { Config } from '../config';
import { getObservabilityConfigSummary } from '../config';
import { sendJson } from '../middleware';
import { logger, scopedLog, LogContext, metrics } from '../utils';
import { createObservabilityCollector, getObservabilityCache } from './index';
import type {
  ObservabilityDepth,
  ObservabilityDependencies,
  DiagnosticRunRequest,
  DiagnosticRunResponse,
  SessionSummary,
  CleanupStatus,
  RecordingStats,
} from './types';
import { VERSION } from '../constants';

// =============================================================================
// Types
// =============================================================================

/** Dependencies for the observability route */
export interface ObservabilityRouteDependencies {
  sessionManager: SessionManager;
  sessionCleanup: SessionCleanup;
  config: Config;
}

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Aggregate recording stats from all sessions.
 * Combines injection stats from all active recording initializers.
 */
function aggregateRecordingStats(
  sessionManager: SessionManager
): RecordingStats | undefined {
  const sessionIds = sessionManager.getAllSessionIds();
  if (sessionIds.length === 0) {
    return undefined;
  }

  const aggregated: RecordingStats = {
    script_version: VERSION,
    injection_stats: {
      attempted: 0,
      successful: 0,
      failed: 0,
      skipped: 0,
      total: 0,
      methods: {
        head: 0,
        HEAD: 0,
        doctype: 0,
        prepend: 0,
      },
    },
  };

  let foundAny = false;

  for (const sessionId of sessionIds) {
    try {
      const session = sessionManager.getSession(sessionId);
      if (session.recordingInitializer) {
        const stats = session.recordingInitializer.getInjectionStats();
        foundAny = true;

        aggregated.injection_stats.attempted += stats.attempted;
        aggregated.injection_stats.successful += stats.successful;
        aggregated.injection_stats.failed += stats.failed;
        aggregated.injection_stats.skipped += stats.skipped;
        aggregated.injection_stats.total = aggregated.injection_stats.attempted;
        aggregated.injection_stats.methods.head += stats.methods.head;
        aggregated.injection_stats.methods.HEAD += stats.methods.HEAD;
        aggregated.injection_stats.methods.doctype += stats.methods.doctype;
        aggregated.injection_stats.methods.prepend += stats.methods.prepend;
      }
    } catch {
      // Session may have been closed during iteration
    }
  }

  return foundAny ? aggregated : undefined;
}

/**
 * Parse query string from URL.
 */
function parseQueryString(url: string): URLSearchParams {
  const queryIndex = url.indexOf('?');
  if (queryIndex === -1) return new URLSearchParams();
  return new URLSearchParams(url.slice(queryIndex + 1));
}

/**
 * Validate depth parameter.
 */
function parseDepth(value: string | null): ObservabilityDepth {
  if (value === 'quick' || value === 'standard' || value === 'deep') {
    return value;
  }
  return 'quick'; // default
}

/**
 * Create session summary from session manager.
 */
function createSessionSummary(
  sessionManager: SessionManager,
  config: Config
): SessionSummary {
  const sessionIds = sessionManager.getAllSessionIds();
  const now = Date.now();

  let activeCount = 0;
  let idleCount = 0;
  let recordingCount = 0;

  for (const id of sessionIds) {
    try {
      const session = sessionManager.getSession(id);
      const idleTimeMs = now - session.lastUsedAt.getTime();

      if (idleTimeMs < config.session.idleTimeoutMs) {
        activeCount++;
      } else {
        idleCount++;
      }

      if (session.recordingController?.isRecording()) {
        recordingCount++;
      }
    } catch {
      // Session closed during iteration - expected during cleanup
    }
  }

  return {
    total: sessionIds.length,
    active: activeCount,
    idle: idleCount,
    active_recordings: recordingCount,
    idle_timeout_ms: config.session.idleTimeoutMs,
    capacity: config.session.maxConcurrent,
  };
}

/**
 * Create cleanup status from session cleanup.
 */
function createCleanupStatus(
  sessionCleanup: SessionCleanup,
  config: Config
): CleanupStatus {
  return {
    is_running: sessionCleanup.isRunningCleanup(),
    last_run_at: sessionCleanup.getLastRunAt() ?? undefined,
    interval_ms: config.session.cleanupIntervalMs,
  };
}

// =============================================================================
// Route Handlers
// =============================================================================

/**
 * GET /observability
 *
 * Main observability endpoint. Returns health, monitoring, and diagnostic data.
 *
 * Query parameters:
 * - depth: 'quick' (default) | 'standard' | 'deep'
 * - no_cache: 'true' to bypass cache
 */
export async function handleObservability(
  req: IncomingMessage,
  res: ServerResponse,
  deps: ObservabilityRouteDependencies
): Promise<void> {
  const { sessionManager, sessionCleanup, config } = deps;

  // Parse query parameters
  const query = parseQueryString(req.url || '');
  const depth = parseDepth(query.get('depth'));
  const noCache = query.get('no_cache') === 'true';

  // Check cache first (unless bypassed)
  const cache = getObservabilityCache();
  if (!noCache) {
    const cached = cache.get(depth);
    if (cached) {
      logger.debug(scopedLog(LogContext.HEALTH, 'serving cached observability'), {
        depth,
        cachedAt: cached.cached_at,
      });
      sendJson(res, 200, cached);
      return;
    }
  }

  // Create collector with dependencies
  const collectorDeps: ObservabilityDependencies = {
    getSessionSummary: () => createSessionSummary(sessionManager, config),
    getBrowserStatus: () => sessionManager.getBrowserStatus(),
    getCleanupStatus: () => createCleanupStatus(sessionCleanup, config),
    getMetricsConfig: () => ({
      enabled: config.metrics.enabled,
      port: config.metrics.port,
    }),
    getRecordingStats: () => aggregateRecordingStats(sessionManager),
    getConfigSummary: () => getObservabilityConfigSummary(),
  };

  const collector = createObservabilityCollector(collectorDeps);

  // Collect data
  const response = await collector.collect(depth);

  // Cache the result
  cache.set(depth, response);

  // Send response
  const httpStatus = response.status === 'error' ? 503 : 200;
  sendJson(res, httpStatus, response);
}

/**
 * POST /observability/refresh
 *
 * Force cache refresh. Returns { refreshed: true, timestamp: string }.
 */
export async function handleObservabilityRefresh(
  _req: IncomingMessage,
  res: ServerResponse
): Promise<void> {
  const cache = getObservabilityCache();
  cache.invalidateAll();

  logger.info(scopedLog(LogContext.HEALTH, 'observability cache refreshed'));

  sendJson(res, 200, {
    refreshed: true,
    timestamp: new Date().toISOString(),
  });
}

/**
 * POST /observability/diagnostics/run
 *
 * Run specific diagnostics manually.
 *
 * Request body:
 * - type: 'recording' | 'browser' | 'all'
 * - session_id?: string (for session-specific diagnostics)
 * - options?: { level?: 'quick' | 'standard' | 'full', timeout_ms?: number }
 */
export async function handleDiagnosticsRun(
  req: IncomingMessage,
  res: ServerResponse,
  deps: ObservabilityRouteDependencies
): Promise<void> {
  // Read request body
  let body = '';
  req.on('data', (chunk) => {
    body += chunk.toString();
  });

  req.on('end', async () => {
    const startedAt = new Date();

    try {
      const request: DiagnosticRunRequest = JSON.parse(body || '{}');
      const { type, session_id, options } = request;

      logger.info(scopedLog(LogContext.HEALTH, 'running diagnostics'), {
        type,
        sessionId: session_id,
        level: options?.level,
      });

      // For now, we only support recording diagnostics via the existing system
      // Future: Add more diagnostic types
      let results: DiagnosticRunResponse['results'] = {};

      if (type === 'recording' || type === 'all') {
        // Recording diagnostics require an active session with a page
        // Auto-select a session if none is provided
        let targetSessionId = session_id;
        if (!targetSessionId) {
          const allSessionIds = deps.sessionManager.getAllSessionIds();
          if (allSessionIds.length > 0) {
            // Prefer a session that's actively recording, otherwise pick first available
            for (const sid of allSessionIds) {
              try {
                const session = deps.sessionManager.getSession(sid);
                if (session.recordingController?.isRecording()) {
                  targetSessionId = sid;
                  break;
                }
              } catch {
                // Session may have been closed
              }
            }
            // If no recording session found, use the first available
            if (!targetSessionId) {
              targetSessionId = allSessionIds[0];
            }
          }
        }

        if (targetSessionId) {
          try {
            const session = deps.sessionManager.getSession(targetSessionId);
            const { runRecordingDiagnostics, RecordingDiagnosticLevel } = await import('../recording/diagnostics');

            const level = options?.level === 'full'
              ? RecordingDiagnosticLevel.FULL
              : options?.level === 'standard'
                ? RecordingDiagnosticLevel.STANDARD
                : RecordingDiagnosticLevel.QUICK;

            results.recording = await runRecordingDiagnostics(session.page, session.context, {
              level,
              timeoutMs: options?.timeout_ms ?? 5000,
              contextInitializer: session.recordingInitializer,
            });
          } catch (error) {
            logger.warn(scopedLog(LogContext.HEALTH, 'recording diagnostics failed'), {
              sessionId: targetSessionId,
              error: error instanceof Error ? error.message : String(error),
            });
            // Return an error result instead of silently failing
            results.recording = {
              ready: false,
              timestamp: new Date().toISOString(),
              durationMs: 0,
              level: options?.level || 'quick',
              issues: [{
                severity: 'error',
                category: 'session',
                message: `Recording diagnostics failed: ${error instanceof Error ? error.message : String(error)}`,
                suggestion: 'Check the browser console for JavaScript errors',
              }],
            };
          }
        } else {
          // No sessions available - return a structured response
          logger.warn(scopedLog(LogContext.HEALTH, 'no sessions available for recording diagnostics'));
          results.recording = {
            ready: false,
            timestamp: new Date().toISOString(),
            durationMs: 0,
            level: options?.level || 'quick',
            issues: [{
              severity: 'warning',
              category: 'session',
              message: 'No active browser sessions available for diagnostics',
              suggestion: 'Start a browser session first by navigating to a page',
            }],
          };
        }
      }

      const completedAt = new Date();

      const response: DiagnosticRunResponse = {
        started_at: startedAt.toISOString(),
        completed_at: completedAt.toISOString(),
        duration_ms: completedAt.getTime() - startedAt.getTime(),
        results,
      };

      sendJson(res, 200, response);
    } catch (error) {
      logger.error(scopedLog(LogContext.HEALTH, 'diagnostics run failed'), {
        error: error instanceof Error ? error.message : String(error),
      });

      sendJson(res, 500, {
        error: 'Failed to run diagnostics',
        message: error instanceof Error ? error.message : String(error),
      });
    }
  });
}

/**
 * GET /observability/metrics
 *
 * Get metrics in JSON format (as opposed to Prometheus text format).
 * Returns all registered metrics with their current values.
 */
export async function handleMetrics(
  _req: IncomingMessage,
  res: ServerResponse,
  deps: ObservabilityRouteDependencies
): Promise<void> {
  try {
    // Get raw Prometheus metrics text
    const metricsText = await metrics.getMetrics();

    // Parse Prometheus format into JSON
    const metricsJson: Record<string, { type: string; help: string; values: Array<{ labels: Record<string, string>; value: number }> }> = {};

    let currentMetric = '';
    let currentType = '';
    let currentHelp = '';

    for (const line of metricsText.split('\n')) {
      if (line.startsWith('# HELP ')) {
        const parts = line.slice(7).split(' ');
        currentMetric = parts[0];
        currentHelp = parts.slice(1).join(' ');
        if (!metricsJson[currentMetric]) {
          metricsJson[currentMetric] = { type: '', help: currentHelp, values: [] };
        } else {
          metricsJson[currentMetric].help = currentHelp;
        }
      } else if (line.startsWith('# TYPE ')) {
        const parts = line.slice(7).split(' ');
        currentMetric = parts[0];
        currentType = parts[1];
        if (!metricsJson[currentMetric]) {
          metricsJson[currentMetric] = { type: currentType, help: '', values: [] };
        } else {
          metricsJson[currentMetric].type = currentType;
        }
      } else if (line && !line.startsWith('#')) {
        // Parse metric value line
        // Format: metric_name{label="value",label2="value2"} value
        // or: metric_name value
        const match = line.match(/^([a-zA-Z_:][a-zA-Z0-9_:]*)(\{[^}]*\})?\s+(.+)$/);
        if (match) {
          const name = match[1];
          const labelsStr = match[2] || '';
          const value = parseFloat(match[3]);

          // Parse labels
          const labels: Record<string, string> = {};
          if (labelsStr) {
            const labelMatches = labelsStr.matchAll(/([a-zA-Z_][a-zA-Z0-9_]*)="([^"]*)"/g);
            for (const labelMatch of labelMatches) {
              labels[labelMatch[1]] = labelMatch[2];
            }
          }

          // Get the base metric name (remove _bucket, _count, _sum suffixes)
          const baseName = name.replace(/_bucket$|_count$|_sum$|_total$/, '');

          if (!metricsJson[baseName]) {
            metricsJson[baseName] = { type: '', help: '', values: [] };
          }

          metricsJson[baseName].values.push({
            labels: { ...labels, _suffix: name.replace(baseName, '') || '_value' },
            value,
          });
        }
      }
    }

    // Add summary stats
    const summary = {
      total_metrics: Object.keys(metricsJson).length,
      timestamp: new Date().toISOString(),
      config: {
        enabled: deps.config.metrics.enabled,
        port: deps.config.metrics.port,
      },
    };

    sendJson(res, 200, { summary, metrics: metricsJson });
  } catch (error) {
    logger.error(scopedLog(LogContext.HEALTH, 'metrics fetch failed'), {
      error: error instanceof Error ? error.message : String(error),
    });

    sendJson(res, 500, {
      error: 'Failed to fetch metrics',
      message: error instanceof Error ? error.message : String(error),
    });
  }
}
