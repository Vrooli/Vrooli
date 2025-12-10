import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { Config } from '../config';
import type { StartSessionRequest, StartSessionResponse, SessionSpec } from '../types';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { InvalidInstructionError } from '../utils';

/**
 * Start session endpoint
 *
 * POST /session/start
 *
 * Response includes:
 * - session_id: Unique session identifier
 * - phase: Current session phase ('ready' for new sessions)
 * - created_at: ISO 8601 timestamp
 * - reused: Whether an existing session was reused (only true for reuse_mode != 'fresh')
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

    // Validate required fields
    if (!request.execution_id || typeof request.execution_id !== 'string') {
      throw new InvalidInstructionError('Missing or invalid execution_id: must be a non-empty string', {
        field: 'execution_id',
        received: typeof request.execution_id,
      });
    }
    if (!request.workflow_id || typeof request.workflow_id !== 'string') {
      throw new InvalidInstructionError('Missing or invalid workflow_id: must be a non-empty string', {
        field: 'workflow_id',
        received: typeof request.workflow_id,
      });
    }
    if (!request.viewport || typeof request.viewport !== 'object') {
      throw new InvalidInstructionError('Missing or invalid viewport: must be an object with width and height', {
        field: 'viewport',
        received: typeof request.viewport,
      });
    }
    if (typeof request.viewport.width !== 'number' || request.viewport.width <= 0) {
      throw new InvalidInstructionError('Invalid viewport.width: must be a positive number', {
        field: 'viewport.width',
        received: request.viewport.width,
      });
    }
    if (typeof request.viewport.height !== 'number' || request.viewport.height <= 0) {
      throw new InvalidInstructionError('Invalid viewport.height: must be a positive number', {
        field: 'viewport.height',
        received: request.viewport.height,
      });
    }

    // Validate reuse_mode if provided
    const validReuseModes = ['fresh', 'clean', 'reuse'];
    if (request.reuse_mode && !validReuseModes.includes(request.reuse_mode)) {
      throw new InvalidInstructionError(`Invalid reuse_mode: must be one of ${validReuseModes.join(', ')}`, {
        field: 'reuse_mode',
        received: request.reuse_mode,
        valid: validReuseModes,
      });
    }

    // Build session spec
    const spec: SessionSpec = {
      execution_id: request.execution_id,
      workflow_id: request.workflow_id,
      viewport: request.viewport,
      reuse_mode: (request.reuse_mode as 'fresh' | 'clean' | 'reuse') || 'fresh',
      base_url: request.base_url,
      labels: request.labels,
      required_capabilities: request.required_capabilities,
      storage_state: request.storage_state,
    };

    // Start session - returns session info including whether it was reused
    const { sessionId, reused, createdAt } = await sessionManager.startSession(spec);

    const response: StartSessionResponse = {
      session_id: sessionId,
      phase: 'ready',
      created_at: createdAt.toISOString(),
      reused: reused || undefined, // Only include if true
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, '/session/start');
  }
}
