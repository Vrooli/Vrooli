import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { Config } from '../config';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { logger } from '../utils';
import { createRecordModeController } from '../recording/controller';
import type { RecordedAction } from '../recording/types';

/**
 * Request types for record mode endpoints
 */
interface StartRecordingRequest {
  /** Optional callback URL to stream actions to (for API integration) */
  callback_url?: string;
}

interface StartRecordingResponse {
  recording_id: string;
  session_id: string;
  started_at: string;
}

interface StopRecordingResponse {
  recording_id: string;
  session_id: string;
  action_count: number;
  stopped_at: string;
}

interface RecordingStatusResponse {
  session_id: string;
  is_recording: boolean;
  recording_id?: string;
  action_count: number;
  started_at?: string;
}

interface ValidateSelectorRequest {
  selector: string;
}

interface ValidateSelectorResponse {
  valid: boolean;
  match_count: number;
  selector: string;
  error?: string;
}

/** Collected actions waiting to be fetched */
const actionBuffers: Map<string, RecordedAction[]> = new Map();

/**
 * Start recording endpoint
 *
 * POST /session/:id/record/start
 *
 * Starts recording user actions in the browser session.
 * Actions are captured and can be retrieved via the /actions endpoint
 * or streamed to a callback URL.
 */
export async function handleRecordStart(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config);
    const request = body as unknown as StartRecordingRequest;

    // Check if already recording
    if (session.recordingController?.isRecording()) {
      sendJson(res, 409, {
        error: 'RECORDING_IN_PROGRESS',
        message: 'Recording is already in progress for this session',
        recording_id: session.recordingId,
      });
      return;
    }

    // Create recording controller if not exists
    if (!session.recordingController) {
      session.recordingController = createRecordModeController(session.page, sessionId);
    }

    // Initialize action buffer for this session
    actionBuffers.set(sessionId, []);

    // Start recording with callback to buffer actions
    const recordingId = await session.recordingController.startRecording({
      sessionId,
      onAction: async (action: RecordedAction) => {
        // Buffer the action
        const buffer = actionBuffers.get(sessionId);
        if (buffer) {
          buffer.push(action);
        }

        // If callback URL provided, stream to it
        if (request.callback_url) {
          try {
            await streamActionToCallback(request.callback_url, action);
          } catch (err) {
            logger.warn('Failed to stream action to callback', {
              sessionId,
              callbackUrl: request.callback_url,
              error: err instanceof Error ? err.message : String(err),
            });
          }
        }

        logger.debug('Recorded action', {
          sessionId,
          recordingId,
          actionType: action.actionType,
          sequenceNum: action.sequenceNum,
        });
      },
      onError: (error: Error) => {
        logger.error('Recording error', {
          sessionId,
          recordingId,
          error: error.message,
        });
      },
    });

    // Store recording ID on session
    session.recordingId = recordingId;

    logger.info('Recording started', {
      sessionId,
      recordingId,
    });

    const response: StartRecordingResponse = {
      recording_id: recordingId,
      session_id: sessionId,
      started_at: new Date().toISOString(),
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/start`);
  }
}

/**
 * Stop recording endpoint
 *
 * POST /session/:id/record/stop
 *
 * Stops the current recording session and returns summary.
 */
export async function handleRecordStop(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);

    if (!session.recordingController?.isRecording()) {
      sendJson(res, 404, {
        error: 'NO_RECORDING',
        message: 'No recording in progress for this session',
      });
      return;
    }

    const result = await session.recordingController.stopRecording();

    logger.info('Recording stopped', {
      sessionId,
      recordingId: result.recordingId,
      actionCount: result.actionCount,
    });

    // Clear recording ID from session
    const recordingId = session.recordingId;
    session.recordingId = undefined;

    const response: StopRecordingResponse = {
      recording_id: recordingId || result.recordingId,
      session_id: sessionId,
      action_count: result.actionCount,
      stopped_at: new Date().toISOString(),
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/stop`);
  }
}

/**
 * Get recording status endpoint
 *
 * GET /session/:id/record/status
 *
 * Returns current recording status for the session.
 */
export async function handleRecordStatus(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);

    const state = session.recordingController?.getState();
    const buffer = actionBuffers.get(sessionId);

    const response: RecordingStatusResponse = {
      session_id: sessionId,
      is_recording: state?.isRecording || false,
      recording_id: state?.recordingId,
      action_count: buffer?.length || state?.actionCount || 0,
      started_at: state?.startedAt,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/status`);
  }
}

/**
 * Get recorded actions endpoint
 *
 * GET /session/:id/record/actions
 *
 * Returns all buffered actions for the session.
 * Optionally clears the buffer after retrieval.
 */
export async function handleRecordActions(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    // Verify session exists
    sessionManager.getSession(sessionId);

    const buffer = actionBuffers.get(sessionId) || [];

    // Check for clear query param (parse URL manually since we don't have query parser)
    const url = new URL(req.url || '', `http://localhost`);
    const shouldClear = url.searchParams.get('clear') === 'true';

    if (shouldClear) {
      actionBuffers.set(sessionId, []);
    }

    sendJson(res, 200, {
      session_id: sessionId,
      actions: buffer,
      count: buffer.length,
    });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/actions`);
  }
}

/**
 * Validate selector endpoint
 *
 * POST /session/:id/record/validate-selector
 *
 * Validates a selector on the current page.
 */
export async function handleValidateSelector(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config);
    const request = body as unknown as ValidateSelectorRequest;

    if (!request.selector) {
      sendJson(res, 400, {
        error: 'MISSING_SELECTOR',
        message: 'selector field is required',
      });
      return;
    }

    // Create controller if needed for validation
    if (!session.recordingController) {
      session.recordingController = createRecordModeController(session.page, sessionId);
    }

    const validation = await session.recordingController.validateSelector(request.selector);

    const response: ValidateSelectorResponse = {
      valid: validation.valid,
      match_count: validation.matchCount,
      selector: validation.selector,
      error: validation.error,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/validate-selector`);
  }
}

/**
 * Stream an action to a callback URL
 */
async function streamActionToCallback(callbackUrl: string, action: RecordedAction): Promise<void> {
  const response = await fetch(callbackUrl, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(action),
  });

  if (!response.ok) {
    throw new Error(`Callback returned ${response.status}: ${response.statusText}`);
  }
}

/**
 * Clean up action buffer for a session
 */
export function cleanupSessionRecording(sessionId: string): void {
  actionBuffers.delete(sessionId);
}
