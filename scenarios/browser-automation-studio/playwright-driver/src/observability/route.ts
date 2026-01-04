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
import { sendJson } from '../middleware';
import { logger, scopedLog, LogContext } from '../utils';
import { createObservabilityCollector, getObservabilityCache } from './index';
import type {
  ObservabilityDepth,
  ObservabilityDependencies,
  DiagnosticRunRequest,
  DiagnosticRunResponse,
  SessionSummary,
  CleanupStatus,
} from './types';

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
    // Note: SessionCleanup doesn't currently track last_run_at
    // This can be added in Phase 3
    last_run_at: undefined,
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
    // getConfigSummary would need to be implemented if we want config details
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
        if (session_id) {
          try {
            const session = deps.sessionManager.getSession(session_id);
            const { runRecordingDiagnostics, RecordingDiagnosticLevel } = await import('../recording/diagnostics');

            const level = options?.level === 'full'
              ? RecordingDiagnosticLevel.FULL
              : options?.level === 'standard'
                ? RecordingDiagnosticLevel.STANDARD
                : RecordingDiagnosticLevel.QUICK;

            results.recording = await runRecordingDiagnostics(session.page, session.context, {
              level,
              timeoutMs: options?.timeout_ms ?? 5000,
            });
          } catch (error) {
            logger.warn(scopedLog(LogContext.HEALTH, 'recording diagnostics failed'), {
              sessionId: session_id,
              error: error instanceof Error ? error.message : String(error),
            });
          }
        } else {
          // No session specified - can't run recording diagnostics
          logger.warn(scopedLog(LogContext.HEALTH, 'no session_id for recording diagnostics'));
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
