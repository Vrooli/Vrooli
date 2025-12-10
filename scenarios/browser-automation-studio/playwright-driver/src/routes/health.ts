import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { HealthResponse, HealthCheck } from '../types';
import { sendJson } from '../middleware';
import { VERSION } from '../constants';

/**
 * Health check endpoint
 *
 * GET /health
 *
 * Returns overall health status:
 * - 'ok': All systems operational, ready to accept traffic
 * - 'degraded': Functional but with issues (e.g., browser not verified yet)
 * - 'error': Critical failure (e.g., browser cannot launch)
 *
 * Response includes:
 * - status: Overall health status
 * - ready: Boolean indicating if the driver is ready to accept sessions
 * - sessions: Current session count
 * - active_recordings: Number of sessions with active recording
 * - browser: Browser health details
 * - uptime_ms: Server uptime in milliseconds
 * - checks: Individual component check results for debugging
 *
 * Semantic meaning of fields:
 * - status='ok' + ready=true: Accept traffic
 * - status='degraded': May have issues but can attempt operations
 * - status='error': Do not route traffic here
 *
 * Each check includes an actionable hint when in 'fail' state to help
 * operators diagnose and resolve issues.
 */

// Track server start time for uptime calculation
const serverStartTime = Date.now();

/**
 * Get actionable hint for browser errors
 */
function getBrowserErrorHint(error: string | undefined): string | undefined {
  if (!error) return undefined;

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

export async function handleHealth(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionManager: SessionManager
): Promise<void> {
  const sessionCount = sessionManager.getSessionCount();
  const browserStatus = sessionManager.getBrowserStatus();

  // Count active recordings
  // Temporal hardening: getAllSessionIds() returns a snapshot array, and sessions
  // can be closed concurrently by the cleanup task or explicit close requests.
  // The try/catch handles SessionNotFoundError gracefully for sessions that
  // close during iteration.
  const sessionIds = sessionManager.getAllSessionIds();
  let activeRecordings = 0;
  for (const id of sessionIds) {
    try {
      const session = sessionManager.getSession(id);
      if (session.recordingController?.isRecording()) {
        activeRecordings++;
      }
    } catch {
      // Session closed between snapshot and access - expected during cleanup
    }
  }

  // Individual component checks for transparency with actionable hints
  const browserCheck: HealthCheck = browserStatus.healthy
    ? {
        status: 'pass',
        message: `Chromium ${browserStatus.version || 'ready'}`,
      }
    : {
        status: 'fail',
        message: browserStatus.error || 'Unknown browser issue',
        hint: getBrowserErrorHint(browserStatus.error),
      };

  const checks = {
    browser: browserCheck,
    sessions: {
      status: 'pass' as const,
      message: `${sessionCount} active session(s)`,
    },
    recordings: {
      status: 'pass' as const,
      message: `${activeRecordings} active recording(s)`,
    },
  };

  // Determine overall health status
  let status: 'ok' | 'degraded' | 'error' = 'ok';
  if (!browserStatus.healthy) {
    // If browser has a specific error, it's an error state
    // If just not verified yet, it's degraded
    status = browserStatus.error && browserStatus.error !== 'Browser not yet verified' ? 'error' : 'degraded';
  }

  // Ready means we can accept new sessions
  const ready = status === 'ok' && browserStatus.healthy;

  const response: HealthResponse = {
    status,
    ready,
    timestamp: new Date().toISOString(),
    sessions: sessionCount,
    active_recordings: activeRecordings,
    version: VERSION,
    browser: browserStatus,
    uptime_ms: Date.now() - serverStartTime,
    checks,
  };

  // Return 503 if in error state so load balancers can detect unhealthy instances
  const httpStatus = status === 'error' ? 503 : 200;
  sendJson(res, httpStatus, response);
}
