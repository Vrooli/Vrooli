import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { HealthResponse } from '../types';
import { sendJson } from '../middleware';
import { VERSION } from '../constants';

/**
 * Health check endpoint
 *
 * GET /health
 */
export async function handleHealth(
  req: IncomingMessage,
  res: ServerResponse,
  sessionManager: SessionManager
): Promise<void> {
  const sessionCount = sessionManager.getSessionCount();

  const response: HealthResponse = {
    status: 'ok',
    timestamp: new Date().toISOString(),
    sessions: sessionCount,
    version: VERSION,
  };

  sendJson(res, 200, response);
}
