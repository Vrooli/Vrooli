/**
 * Observability Route Handler
 *
 * Provides the unified health, monitoring, metrics, and session endpoints.
 *
 * ## Endpoints
 *
 * - `GET /observability` - Get observability data
 * - `POST /observability/refresh` - Force cache refresh
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
 * Combines injection stats and route handler stats from all active recording initializers.
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
      avgInjectionTimeMs: 0,
      lastInjectionAt: null,
    },
    route_handler_stats: {
      eventsReceived: 0,
      eventsProcessed: 0,
      eventsDroppedNoHandler: 0,
      eventsWithErrors: 0,
      lastEventAt: null,
      lastEventType: null,
    },
    has_event_handler: false,
    active_session_id: undefined,
  };
  const routeHandlerStats = aggregated.route_handler_stats ?? {
    eventsReceived: 0,
    eventsProcessed: 0,
    eventsDroppedNoHandler: 0,
    eventsWithErrors: 0,
    lastEventAt: null,
    lastEventType: null,
  };
  aggregated.route_handler_stats = routeHandlerStats;

  let foundAny = false;
  let latestEventAt: Date | null = null;
  let latestEventType: string | null = null;
  let activeRecordingSessionId: string | undefined;

  for (const sessionId of sessionIds) {
    try {
      const session = sessionManager.peekSession(sessionId);
      if (session.recordingInitializer) {
        const injectionStats = session.recordingInitializer.getInjectionStats();
        const routeStats = session.recordingInitializer.getRouteHandlerStats();
        foundAny = true;

        // Track the first actively recording session for debugging
        if (!activeRecordingSessionId && session.pipelineManager?.isRecording()) {
          activeRecordingSessionId = sessionId;
        }

        // Aggregate injection stats
        aggregated.injection_stats.attempted += injectionStats.attempted;
        aggregated.injection_stats.successful += injectionStats.successful;
        aggregated.injection_stats.failed += injectionStats.failed;

        // Track most recent injection time across all sessions
        if (
          injectionStats.lastInjectionAt &&
          (!aggregated.injection_stats.lastInjectionAt ||
            injectionStats.lastInjectionAt > aggregated.injection_stats.lastInjectionAt)
        ) {
          aggregated.injection_stats.lastInjectionAt = injectionStats.lastInjectionAt;
        }

        // Note: avgInjectionTimeMs is not aggregated since averaging averages is misleading

        // Aggregate route handler stats
        routeHandlerStats.eventsReceived += routeStats.eventsReceived;
        routeHandlerStats.eventsProcessed += routeStats.eventsProcessed;
        routeHandlerStats.eventsDroppedNoHandler += routeStats.eventsDroppedNoHandler;
        routeHandlerStats.eventsWithErrors += routeStats.eventsWithErrors;

        // Track the most recent event across all sessions
        if (routeStats.lastEventAt) {
          const eventTime = new Date(routeStats.lastEventAt);
          if (!latestEventAt || eventTime > latestEventAt) {
            latestEventAt = eventTime;
            latestEventType = routeStats.lastEventType;
          }
        }

        // Check if any session has an event handler set
        if (session.recordingInitializer.hasEventHandler()) {
          aggregated.has_event_handler = true;
        }
      }
    } catch {
      // Session may have been closed during iteration
    }
  }

  // Set the most recent event info
  if (latestEventAt) {
    routeHandlerStats.lastEventAt = latestEventAt.toISOString();
    routeHandlerStats.lastEventType = latestEventType;
  }

  // Set the active recording session ID for debugging
  aggregated.active_session_id = activeRecordingSessionId;

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
      const session = sessionManager.peekSession(id);
      const idleTimeMs = now - session.lastUsedAt.getTime();

      if (idleTimeMs < config.session.idleTimeoutMs) {
        activeCount++;
      } else {
        idleCount++;
      }

      if (session.pipelineManager?.isRecording()) {
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
export function handleObservabilityRefresh(
  _req: IncomingMessage,
  res: ServerResponse
): void {
  const cache = getObservabilityCache();
  cache.invalidateAll();

  logger.info(scopedLog(LogContext.HEALTH, 'observability cache refreshed'));

  sendJson(res, 200, {
    refreshed: true,
    timestamp: new Date().toISOString(),
  });
}

export function handleSessionList(
  _req: IncomingMessage,
  res: ServerResponse,
  deps: ObservabilityRouteDependencies
): void {
  try {
    const sessions = deps.sessionManager.getSessionList();
    const summary = deps.sessionManager.getSessionSummary();

    sendJson(res, 200, {
      sessions,
      summary: {
        total: summary.total,
        active: summary.active,
        idle: summary.idle,
        active_recordings: summary.active_recordings,
        capacity: summary.capacity,
      },
      timestamp: new Date().toISOString(),
    });
  } catch (error) {
    logger.error(scopedLog(LogContext.HEALTH, 'session list failed'), {
      error: error instanceof Error ? error.message : String(error),
    });

    sendJson(res, 500, {
      error: 'Failed to get session list',
      message: error instanceof Error ? error.message : String(error),
    });
  }
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
        const metricName = parts[0];
        if (!metricName) {
          continue;
        }
        currentMetric = metricName;
        currentHelp = parts.slice(1).join(' ');
        const metricEntry =
          metricsJson[currentMetric] ?? (metricsJson[currentMetric] = { type: '', help: currentHelp, values: [] });
        metricEntry.help = currentHelp;
      } else if (line.startsWith('# TYPE ')) {
        const parts = line.slice(7).split(' ');
        const metricName = parts[0];
        const metricType = parts[1];
        if (!metricName || !metricType) {
          continue;
        }
        currentMetric = metricName;
        currentType = metricType;
        const metricEntry =
          metricsJson[currentMetric] ?? (metricsJson[currentMetric] = { type: currentType, help: '', values: [] });
        metricEntry.type = currentType;
      } else if (line && !line.startsWith('#')) {
        // Parse metric value line
        // Format: metric_name{label="value",label2="value2"} value
        // or: metric_name value
        const match = line.match(/^([a-zA-Z_:][a-zA-Z0-9_:]*)(\{[^}]*\})?\s+(.+)$/);
        if (match) {
          const name = match[1];
          if (!name) {
            continue;
          }
          const labelsStr = match[2] || '';
          const valueRaw = match[3];
          if (!valueRaw) {
            continue;
          }
          const value = parseFloat(valueRaw);

          // Parse labels
          const labels: Record<string, string> = {};
          if (labelsStr) {
            const labelMatches = labelsStr.matchAll(/([a-zA-Z_][a-zA-Z0-9_]*)="([^"]*)"/g);
            for (const labelMatch of labelMatches) {
              const labelName = labelMatch[1];
              const labelValue = labelMatch[2];
              if (!labelName || labelValue === undefined) {
                continue;
              }
              labels[labelName] = labelValue;
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
