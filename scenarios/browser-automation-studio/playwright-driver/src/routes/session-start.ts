import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { Config } from '../config';
import type { StartSessionRequest, StartSessionResponse, SessionSpec } from '../types';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { logger } from '../utils';

/**
 * Start session endpoint
 *
 * POST /session/start
 */
export async function handleSessionStart(
  req: IncomingMessage,
  res: ServerResponse,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    // Parse request body
    const body = await parseJsonBody(req, config);
    const request = body as unknown as StartSessionRequest;

    // Build session spec
    const spec: SessionSpec = {
      execution_id: request.execution_id,
      workflow_id: request.workflow_id,
      viewport: request.viewport,
      reuse_mode: (request.reuse_mode as 'fresh' | 'clean' | 'reuse') || 'fresh',
      base_url: request.base_url,
      labels: request.labels,
      required_capabilities: request.required_capabilities,
    };

    logger.info('Starting session', {
      executionId: spec.execution_id,
      reuseMode: spec.reuse_mode,
      viewport: spec.viewport,
      requestViewport: request.viewport,
    });

    // Start session
    const sessionId = await sessionManager.startSession(spec);

    const response: StartSessionResponse = {
      session_id: sessionId,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, '/session/start');
  }
}
