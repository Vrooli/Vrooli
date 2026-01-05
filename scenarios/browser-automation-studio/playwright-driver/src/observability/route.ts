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
import { getObservabilityConfigSummary, CONFIG_TIER_METADATA } from '../config';
import { sendJson } from '../middleware';
import { logger, scopedLog, LogContext, metrics } from '../utils';
import {
  setRuntimeValue,
  resetRuntimeValue,
  getRuntimeConfigState,
  type SetConfigResult,
} from '../runtime-config';
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
import type {
  RecordingDiagnosticResult,
  DiagnosticIssue,
  DiagnosticSeverity,
  DiagnosticCheck,
  EventFlowTestResult,
} from '../recording/diagnostics';
import { runRecordingPipelineTest } from '../recording/self-test';

// =============================================================================
// UI-Compatible Types
// =============================================================================

/** Issue format expected by the UI */
interface UIDiagnosticIssue {
  severity: 'error' | 'warning' | 'info';
  category: string;
  message: string;
  suggestion?: string;
  docs_link?: string;
}

/** Recording diagnostics result formatted for UI consumption */
interface UIRecordingDiagnostics {
  ready: boolean;
  timestamp: string;
  durationMs: number;
  level: 'quick' | 'standard' | 'full';
  /** All checks performed with their status for breakdown display */
  checks?: DiagnosticCheck[];
  issues: UIDiagnosticIssue[];
  provider?: {
    name: string;
    evaluateIsolated: boolean;
    exposeBindingIsolated: boolean;
  };
  /** Event flow test result with detailed diagnostics (FULL level only) */
  eventFlowTest?: EventFlowTestResult;
}

/**
 * Extract category from diagnostic code.
 * Maps DIAGNOSTIC_CODES prefixes to user-friendly categories.
 */
function codeToCategory(code: string): string {
  if (code.startsWith('SCRIPT_')) return 'script';
  if (code.startsWith('INJECTION_')) return 'injection';
  if (code.startsWith('EVENT_')) return 'event';
  if (code.startsWith('PROVIDER_')) return 'provider';
  if (code.startsWith('CDP_')) return 'cdp';
  return 'general';
}

/**
 * Map DiagnosticSeverity enum to string literal.
 */
function severityToString(severity: DiagnosticSeverity): 'error' | 'warning' | 'info' {
  // DiagnosticSeverity enum values are 'error', 'warning', 'info'
  return severity as unknown as 'error' | 'warning' | 'info';
}

/**
 * Transform backend diagnostic result to UI-compatible format.
 */
function transformDiagnosticsForUI(result: RecordingDiagnosticResult): UIRecordingDiagnostics {
  return {
    ready: result.ready,
    timestamp: result.timestamp,
    durationMs: result.durationMs,
    level: result.level,
    // Include all checks performed for detailed breakdown display
    checks: result.checks,
    issues: result.issues.map((issue: DiagnosticIssue) => ({
      severity: severityToString(issue.severity),
      category: codeToCategory(issue.code),
      message: issue.message,
      suggestion: issue.suggestion,
    })),
    provider: result.provider,
    // Include event flow test result for detailed diagnostics (FULL level only)
    eventFlowTest: result.eventFlowTest,
  };
}

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
      skipped: 0,
      total: 0,
      methods: {
        head: 0,
        HEAD: 0,
        doctype: 0,
        prepend: 0,
      },
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

  let foundAny = false;
  let latestEventAt: Date | null = null;
  let latestEventType: string | null = null;
  let activeRecordingSessionId: string | undefined;

  for (const sessionId of sessionIds) {
    try {
      const session = sessionManager.getSession(sessionId);
      if (session.recordingInitializer) {
        const injectionStats = session.recordingInitializer.getInjectionStats();
        const routeStats = session.recordingInitializer.getRouteHandlerStats();
        foundAny = true;

        // Track the first actively recording session for debugging
        if (!activeRecordingSessionId && session.recordingController?.isRecording()) {
          activeRecordingSessionId = sessionId;
        }

        // Aggregate injection stats
        aggregated.injection_stats.attempted += injectionStats.attempted;
        aggregated.injection_stats.successful += injectionStats.successful;
        aggregated.injection_stats.failed += injectionStats.failed;
        aggregated.injection_stats.skipped += injectionStats.skipped;
        aggregated.injection_stats.total = aggregated.injection_stats.attempted;
        aggregated.injection_stats.methods.head += injectionStats.methods.head;
        aggregated.injection_stats.methods.HEAD += injectionStats.methods.HEAD;
        aggregated.injection_stats.methods.doctype += injectionStats.methods.doctype;
        aggregated.injection_stats.methods.prepend += injectionStats.methods.prepend;

        // Aggregate route handler stats
        aggregated.route_handler_stats!.eventsReceived += routeStats.eventsReceived;
        aggregated.route_handler_stats!.eventsProcessed += routeStats.eventsProcessed;
        aggregated.route_handler_stats!.eventsDroppedNoHandler += routeStats.eventsDroppedNoHandler;
        aggregated.route_handler_stats!.eventsWithErrors += routeStats.eventsWithErrors;

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
    aggregated.route_handler_stats!.lastEventAt = latestEventAt.toISOString();
    aggregated.route_handler_stats!.lastEventType = latestEventType;
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

            const rawResult = await runRecordingDiagnostics(session.page, session.context, {
              level,
              timeoutMs: options?.timeout_ms ?? 5000,
              contextInitializer: session.recordingInitializer,
            });

            // Transform to UI-compatible format with category instead of code
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            results.recording = transformDiagnosticsForUI(rawResult) as any;
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
 * GET /observability/sessions
 *
 * Get detailed list of all active browser sessions.
 * Returns session metadata for diagnostics and monitoring.
 */
export async function handleSessionList(
  _req: IncomingMessage,
  res: ServerResponse,
  deps: ObservabilityRouteDependencies
): Promise<void> {
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
 * POST /observability/cleanup/run
 *
 * Trigger manual cleanup of idle sessions.
 * Returns the number of sessions cleaned up.
 */
export async function handleCleanupRun(
  _req: IncomingMessage,
  res: ServerResponse,
  deps: ObservabilityRouteDependencies
): Promise<void> {
  const startedAt = new Date();

  logger.info(scopedLog(LogContext.HEALTH, 'manual cleanup triggered'));

  try {
    // Get session count before cleanup
    const beforeCount = deps.sessionManager.getSessionCount();

    // Run cleanup
    await deps.sessionManager.cleanupIdleSessions();

    // Get session count after cleanup
    const afterCount = deps.sessionManager.getSessionCount();
    const cleanedUp = beforeCount - afterCount;

    const completedAt = new Date();

    sendJson(res, 200, {
      success: true,
      cleaned_up: cleanedUp,
      remaining_sessions: afterCount,
      started_at: startedAt.toISOString(),
      completed_at: completedAt.toISOString(),
      duration_ms: completedAt.getTime() - startedAt.getTime(),
    });
  } catch (error) {
    logger.error(scopedLog(LogContext.HEALTH, 'manual cleanup failed'), {
      error: error instanceof Error ? error.message : String(error),
    });

    sendJson(res, 500, {
      error: 'Failed to run cleanup',
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

/**
 * PUT /observability/config/:env_var
 *
 * Update a runtime configuration value.
 * Only works for options marked as `editable: true` in CONFIG_TIER_METADATA.
 *
 * Request body: { value: string }
 * Response: SetConfigResult
 */
export async function handleConfigUpdate(
  req: IncomingMessage,
  res: ServerResponse,
  envVar: string
): Promise<void> {
  // Read request body
  let body = '';
  req.on('data', (chunk) => {
    body += chunk.toString();
  });

  req.on('end', () => {
    try {
      const request = JSON.parse(body || '{}');
      const { value } = request;

      if (value === undefined) {
        sendJson(res, 400, {
          success: false,
          error: 'Missing required field: value',
        });
        return;
      }

      logger.info(scopedLog(LogContext.CONFIG, 'config update requested'), {
        envVar,
        newValue: value,
      });

      const result = setRuntimeValue(envVar, String(value));

      // Invalidate observability cache since config changed
      if (result.success) {
        const cache = getObservabilityCache();
        cache.invalidateAll();
      }

      sendJson(res, result.success ? 200 : 400, result);
    } catch (error) {
      logger.error(scopedLog(LogContext.CONFIG, 'config update failed'), {
        envVar,
        error: error instanceof Error ? error.message : String(error),
      });

      sendJson(res, 500, {
        success: false,
        error: 'Failed to update configuration',
        message: error instanceof Error ? error.message : String(error),
      });
    }
  });
}

/**
 * DELETE /observability/config/:env_var
 *
 * Reset a runtime configuration value back to its environment/default value.
 *
 * Response: { success: boolean, env_var: string, reset: boolean, current_value: string }
 */
export async function handleConfigReset(
  _req: IncomingMessage,
  res: ServerResponse,
  envVar: string
): Promise<void> {
  try {
    logger.info(scopedLog(LogContext.CONFIG, 'config reset requested'), { envVar });

    const wasReset = resetRuntimeValue(envVar);

    // Get the new effective value
    const meta = CONFIG_TIER_METADATA[envVar];
    const envValue = process.env[envVar];
    const currentValue = envValue ?? (meta?.defaultValue !== undefined ? String(meta.defaultValue) : '');

    // Invalidate observability cache
    if (wasReset) {
      const cache = getObservabilityCache();
      cache.invalidateAll();
    }

    sendJson(res, 200, {
      success: true,
      env_var: envVar,
      reset: wasReset,
      current_value: currentValue,
    });
  } catch (error) {
    logger.error(scopedLog(LogContext.CONFIG, 'config reset failed'), {
      envVar,
      error: error instanceof Error ? error.message : String(error),
    });

    sendJson(res, 500, {
      success: false,
      error: 'Failed to reset configuration',
      message: error instanceof Error ? error.message : String(error),
    });
  }
}

/**
 * GET /observability/config/runtime
 *
 * Get the current state of all runtime configuration overrides.
 *
 * Response: RuntimeConfigState
 */
export async function handleConfigRuntime(
  _req: IncomingMessage,
  res: ServerResponse
): Promise<void> {
  try {
    const state = getRuntimeConfigState();
    sendJson(res, 200, state);
  } catch (error) {
    logger.error(scopedLog(LogContext.CONFIG, 'failed to get runtime config state'), {
      error: error instanceof Error ? error.message : String(error),
    });

    sendJson(res, 500, {
      error: 'Failed to get runtime config state',
      message: error instanceof Error ? error.message : String(error),
    });
  }
}

/**
 * POST /observability/pipeline-test
 *
 * Run an automated end-to-end test of the recording pipeline.
 * This is fully autonomous - it creates a temporary session if needed.
 *
 * Request body (optional):
 * {
 *   "timeout_ms": 30000,  // Test timeout (default: 30000)
 * }
 *
 * Response: PipelineTestResponse (same format as session-specific endpoint)
 */
export async function handlePipelineTest(
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
    const startTime = Date.now();
    let tempSessionId: string | undefined;
    let createdTempSession = false;

    try {
      const request = JSON.parse(body || '{}');
      const timeoutMs = request.timeout_ms ?? 30000;

      logger.info(scopedLog(LogContext.HEALTH, 'autonomous pipeline test starting'), {
        timeoutMs,
      });

      // Try to find an existing session to use, otherwise create a temporary one
      const existingSessionIds = deps.sessionManager.getAllSessionIds();
      let session;

      if (existingSessionIds.length > 0) {
        // Use an existing session
        tempSessionId = existingSessionIds[0];
        session = deps.sessionManager.getSession(tempSessionId);
        logger.debug(scopedLog(LogContext.HEALTH, 'using existing session for pipeline test'), {
          sessionId: tempSessionId,
        });
      } else {
        // Create a temporary session
        logger.info(scopedLog(LogContext.HEALTH, 'creating temporary session for pipeline test'));

        const result = await deps.sessionManager.startSession({
          execution_id: `pipeline-test-${Date.now()}`,
          workflow_id: 'pipeline-test',
          viewport: { width: 1280, height: 720 },
          reuse_mode: 'fresh',
        });

        tempSessionId = result.sessionId;
        createdTempSession = true;
        session = deps.sessionManager.getSession(tempSessionId);

        logger.debug(scopedLog(LogContext.HEALTH, 'temporary session created'), {
          sessionId: tempSessionId,
        });
      }

      // Ensure we have a recording initializer
      if (!session.recordingInitializer) {
        throw new Error('Recording initializer not set on session - context may not have been initialized properly');
      }

      // Build server base URL for the test page
      // This is needed because the test uses a data URL which has no origin,
      // so we inject a <base> tag to allow relative URLs like /__vrooli_recording_event__
      const serverBaseUrl = `http://${deps.config.server.host}:${deps.config.server.port}`;

      // Run the pipeline test
      const result = await runRecordingPipelineTest(
        session.page,
        session.context,
        session.recordingInitializer,
        {
          timeoutMs,
          captureConsole: true,
          serverBaseUrl,
        }
      );

      // Clean up temporary session if we created one
      if (createdTempSession && tempSessionId) {
        try {
          await deps.sessionManager.closeSession(tempSessionId);
          logger.debug(scopedLog(LogContext.HEALTH, 'temporary session cleaned up'), {
            sessionId: tempSessionId,
          });
        } catch (cleanupError) {
          logger.warn(scopedLog(LogContext.HEALTH, 'failed to clean up temporary session'), {
            sessionId: tempSessionId,
            error: cleanupError instanceof Error ? cleanupError.message : String(cleanupError),
          });
        }
      }

      // Log result summary
      const durationMs = Date.now() - startTime;
      if (result.success) {
        logger.info(scopedLog(LogContext.HEALTH, 'autonomous pipeline test PASSED'), {
          durationMs,
          usedTempSession: createdTempSession,
          stepsCompleted: result.steps.filter(s => s.passed).length,
          totalSteps: result.steps.length,
        });
      } else {
        logger.warn(scopedLog(LogContext.HEALTH, 'autonomous pipeline test FAILED'), {
          durationMs,
          usedTempSession: createdTempSession,
          failurePoint: result.failurePoint,
          failureMessage: result.failureMessage,
        });
      }

      // Build response (same format as session-specific endpoint)
      const response = {
        success: result.success,
        timestamp: result.timestamp,
        duration_ms: result.durationMs,
        failure_point: result.failurePoint,
        failure_message: result.failureMessage,
        suggestions: result.suggestions,
        steps: result.steps.map(step => ({
          name: step.name,
          passed: step.passed,
          duration_ms: step.durationMs,
          error: step.error,
          details: step.details,
        })),
        diagnostics: {
          test_page_url: result.diagnostics.testPageUrl,
          test_page_injected: result.diagnostics.testPageInjected,
          script_status_before: result.diagnostics.scriptStatusBefore,
          script_status_after: result.diagnostics.scriptStatusAfter,
          telemetry_before: result.diagnostics.telemetryBefore,
          telemetry_after: result.diagnostics.telemetryAfter,
          route_stats_before: result.diagnostics.routeStatsBefore,
          route_stats_after: result.diagnostics.routeStatsAfter,
          events_captured: result.diagnostics.eventsCaptured,
          console_messages: result.diagnostics.consoleMessages.slice(0, 50),
        },
        // Extra info for autonomous mode
        used_temp_session: createdTempSession,
        session_id: tempSessionId,
      };

      sendJson(res, 200, response);
    } catch (error) {
      // Clean up on error if we created a temp session
      if (createdTempSession && tempSessionId) {
        try {
          await deps.sessionManager.closeSession(tempSessionId);
        } catch {
          // Ignore cleanup errors on failure path
        }
      }

      logger.error(scopedLog(LogContext.HEALTH, 'autonomous pipeline test failed'), {
        error: error instanceof Error ? error.message : String(error),
        durationMs: Date.now() - startTime,
      });

      sendJson(res, 500, {
        success: false,
        error: 'Pipeline test failed',
        message: error instanceof Error ? error.message : String(error),
        hint: 'Check browser connectivity and ensure the driver is running correctly',
      });
    }
  });
}
