import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { HealthResponse } from '../types';
import { sendJson } from '../middleware';
import { VERSION } from '../constants';

/**
 * Health check endpoint
 *
 * GET /health
 *
 * Returns overall health status:
 * - 'ok': All systems operational
 * - 'degraded': Functional but with issues (e.g., browser not verified yet)
 * - 'error': Critical failure (e.g., browser cannot launch)
 */
export async function handleHealth(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionManager: SessionManager
): Promise<void> {
  const sessionCount = sessionManager.getSessionCount();
  const browserStatus = sessionManager.getBrowserStatus();

  // Determine overall health status
  let status: 'ok' | 'degraded' | 'error' = 'ok';
  if (!browserStatus.healthy) {
    // If browser has a specific error, it's an error state
    // If just not verified yet, it's degraded
    status = browserStatus.error && browserStatus.error !== 'Browser not yet verified' ? 'error' : 'degraded';
  }

  const response: HealthResponse = {
    status,
    timestamp: new Date().toISOString(),
    sessions: sessionCount,
    version: VERSION,
    browser: browserStatus,
  };

  // Return 503 if in error state so load balancers can detect unhealthy instances
  const httpStatus = status === 'error' ? 503 : 200;
  sendJson(res, httpStatus, response);
}
