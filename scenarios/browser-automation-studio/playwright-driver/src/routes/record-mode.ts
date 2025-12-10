import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { Config } from '../config';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { logger } from '../utils';
import { createRecordModeController } from '../recording/controller';
import {
  initRecordingBuffer,
  bufferRecordedAction,
  getRecordedActions,
  getRecordedActionCount,
  clearRecordedActions,
  removeRecordedActions,
} from '../recording/buffer';
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

interface ReplayPreviewRequest {
  actions: RecordedAction[];
  limit?: number;
  stop_on_failure?: boolean;
  action_timeout?: number;
}

interface ReplayPreviewResponse {
  success: boolean;
  total_actions: number;
  passed_actions: number;
  failed_actions: number;
  results: Array<{
    action_id: string;
    sequence_num: number;
    action_type: string;
    success: boolean;
    duration_ms: number;
    error?: {
      message: string;
      code: string;
      match_count?: number;
      selector?: string;
    };
    screenshot_on_error?: string;
  }>;
  total_duration_ms: number;
  stopped_early: boolean;
}

interface NavigateRequest {
  url: string;
  wait_until?: 'load' | 'domcontentloaded' | 'networkidle' | 'commit';
  timeout_ms?: number;
  capture?: boolean;
}

interface NavigateResponse {
  session_id: string;
  url: string;
  screenshot?: string;
}

interface ScreenshotRequest {
  full_page?: boolean;
  quality?: number;
}

interface ScreenshotResponse {
  session_id: string;
  screenshot: string;
}

type PointerAction = 'move' | 'down' | 'up' | 'click';
type InputType = 'pointer' | 'keyboard' | 'wheel';

interface InputRequest {
  type: InputType;
  session_id?: string;
  // Pointer
  action?: PointerAction;
  x?: number;
  y?: number;
  button?: 'left' | 'right' | 'middle';
  modifiers?: string[];
  // Keyboard
  key?: string;
  text?: string;
  // Wheel
  delta_x?: number;
  delta_y?: number;
}

interface FrameResponse {
  session_id: string;
  mime: 'image/jpeg';
  image: string;
  width: number;
  height: number;
  captured_at: string;
}

interface ViewportRequest {
  width: number;
  height: number;
}

interface ViewportResponse {
  session_id: string;
  width: number;
  height: number;
}

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
    initRecordingBuffer(sessionId);

    logger.info('recording: starting', {
      sessionId,
      hasCallback: !!request.callback_url,
    });

    // Start recording with callback to buffer actions
    const recordingId = await session.recordingController.startRecording({
      sessionId,
      onAction: async (action: RecordedAction) => {
        // Buffer the action
        bufferRecordedAction(sessionId, action);

        logger.debug('recording: action captured', {
          sessionId,
          actionType: action.actionType,
          sequenceNum: action.sequenceNum,
        });

        // If callback URL provided, stream to it
        if (request.callback_url) {
          try {
            await streamActionToCallback(request.callback_url, action);
          } catch (err) {
            logger.warn('recording: callback failed', {
              sessionId,
              error: err instanceof Error ? err.message : String(err),
            });
          }
        }
      },
      onError: (error: Error) => {
        logger.error('recording: error', {
          sessionId,
          error: error.message,
        });
      },
    });

    // Store recording ID on session
    session.recordingId = recordingId;

    logger.info('recording: started', {
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

    logger.info('recording: stopped', {
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
    const bufferedCount = getRecordedActionCount(sessionId);

    const response: RecordingStatusResponse = {
      session_id: sessionId,
      is_recording: state?.isRecording || false,
      recording_id: state?.recordingId,
      action_count: bufferedCount || state?.actionCount || 0,
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

    const actions = getRecordedActions(sessionId);

    // Check for clear query param (parse URL manually since we don't have query parser)
    const url = new URL(req.url || '', `http://localhost`);
    const shouldClear = url.searchParams.get('clear') === 'true';

    if (shouldClear) {
      clearRecordedActions(sessionId);
    }

    sendJson(res, 200, {
      session_id: sessionId,
      actions,
      count: actions.length,
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
 * Replay preview endpoint
 *
 * POST /session/:id/record/replay-preview
 *
 * Replays recorded actions to test if they work before generating a workflow.
 */
export async function handleReplayPreview(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config);
    const request = body as unknown as ReplayPreviewRequest;

    if (!request.actions || !Array.isArray(request.actions) || request.actions.length === 0) {
      sendJson(res, 400, {
        error: 'MISSING_ACTIONS',
        message: 'actions field is required and must be a non-empty array',
      });
      return;
    }

    // Create controller if needed
    if (!session.recordingController) {
      session.recordingController = createRecordModeController(session.page, sessionId);
    }

    logger.info('recording: replay started', {
      sessionId,
      actionCount: request.actions.length,
      limit: request.limit,
      stopOnFailure: request.stop_on_failure ?? true,
    });

    // Execute replay
    const result = await session.recordingController.replayPreview({
      actions: request.actions,
      limit: request.limit,
      stopOnFailure: request.stop_on_failure ?? true,
      actionTimeout: request.action_timeout ?? 10000,
    });

    logger.info('recording: replay complete', {
      sessionId,
      success: result.success,
      passed: result.passedActions,
      failed: result.failedActions,
      durationMs: result.totalDurationMs,
    });

    // Convert to API response format (snake_case)
    const response: ReplayPreviewResponse = {
      success: result.success,
      total_actions: result.totalActions,
      passed_actions: result.passedActions,
      failed_actions: result.failedActions,
      results: result.results.map((r) => ({
        action_id: r.actionId,
        sequence_num: r.sequenceNum,
        action_type: r.actionType,
        success: r.success,
        duration_ms: r.durationMs,
        error: r.error ? {
          message: r.error.message,
          code: r.error.code,
          match_count: r.error.matchCount,
          selector: r.error.selector,
        } : undefined,
        screenshot_on_error: r.screenshotOnError,
      })),
      total_duration_ms: result.totalDurationMs,
      stopped_early: result.stoppedEarly,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/replay-preview`);
  }
}

/**
 * Navigate the recording session to a URL and optionally return a screenshot.
 *
 * POST /session/:id/record/navigate
 */
export async function handleRecordNavigate(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config);
    const request = body as unknown as NavigateRequest;

    if (!request.url || typeof request.url !== 'string') {
      sendJson(res, 400, {
        error: 'MISSING_URL',
        message: 'url field is required',
      });
      return;
    }

    const waitUntil: NavigateRequest['wait_until'] = request.wait_until || 'load';
    const timeoutMs = request.timeout_ms ?? 15000;

    await session.page.goto(request.url, { waitUntil, timeout: timeoutMs });

    let screenshot: string | undefined;
    if (request.capture) {
      try {
        const buffer = await session.page.screenshot({
          fullPage: true,
          type: 'jpeg',
          quality: 70,
        });
        screenshot = `data:image/jpeg;base64,${buffer.toString('base64')}`;
      } catch (err) {
        logger.warn('recording: screenshot capture failed', {
          sessionId,
          error: err instanceof Error ? err.message : String(err),
        });
      }
    }

    const response: NavigateResponse = {
      session_id: sessionId,
      url: session.page.url(),
      screenshot,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/navigate`);
  }
}

/**
 * Capture a screenshot from the current recording page.
 *
 * POST /session/:id/record/screenshot
 */
export async function handleRecordScreenshot(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config);
    const request = body as unknown as ScreenshotRequest;

    const fullPage = request.full_page !== false;
    const quality = request.quality ?? 70;

    const buffer = await session.page.screenshot({
      fullPage,
      type: 'jpeg',
      quality,
    });

    const response: ScreenshotResponse = {
      session_id: sessionId,
      screenshot: `data:image/jpeg;base64,${buffer.toString('base64')}`,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/screenshot`);
  }
}

/**
 * Forward live input events to the active Playwright page.
 *
 * POST /session/:id/record/input
 */
export async function handleRecordInput(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config);
    const request = body as unknown as InputRequest;

    if (!request?.type) {
      sendJson(res, 400, {
        error: 'MISSING_TYPE',
        message: 'type field is required',
      });
      return;
    }

    const page = session.page;
    const modifiers = request.modifiers || [];

    switch (request.type) {
      case 'pointer': {
        const x = request.x ?? 0;
        const y = request.y ?? 0;
        const button = request.button || 'left';
        const action: PointerAction = request.action || 'move';

        if (action === 'move') {
          await page.mouse.move(x, y);
        } else if (action === 'down') {
          await page.mouse.move(x, y);
          await page.mouse.down({ button });
        } else if (action === 'up') {
          await page.mouse.move(x, y);
          await page.mouse.up({ button });
        } else if (action === 'click') {
          await page.mouse.click(x, y, { button });
        } else {
          sendJson(res, 400, {
            error: 'INVALID_ACTION',
            message: `Unsupported pointer action: ${action}`,
          });
          return;
        }
        break;
      }
      case 'wheel': {
        await page.mouse.wheel(request.delta_x ?? 0, request.delta_y ?? 0);
        break;
      }
      case 'keyboard': {
        if (request.text) {
          await page.keyboard.type(request.text);
        } else if (request.key) {
          const combo = modifiers.length > 0 ? `${modifiers.join('+')}+${request.key}` : request.key;
          await page.keyboard.press(combo);
        } else {
          sendJson(res, 400, {
            error: 'MISSING_KEYBOARD_DATA',
            message: 'Provide key or text for keyboard input',
          });
          return;
        }
        break;
      }
      default:
        sendJson(res, 400, {
          error: 'INVALID_TYPE',
          message: `Unsupported input type: ${request.type}`,
        });
        return;
    }

    sendJson(res, 200, {
      status: 'ok',
    });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/input`);
  }
}

/**
 * Get a lightweight frame preview from the active Playwright page.
 *
 * GET /session/:id/record/frame
 */
export async function handleRecordFrame(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  _config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const url = new URL(req.url || '', `http://localhost`);

    const quality = Number(url.searchParams.get('quality')) || 60;
    const fullPage = url.searchParams.get('full_page') === 'true';

    const buffer = await session.page.screenshot({
      type: 'jpeg',
      fullPage,
      quality,
    });

    const viewport = session.page.viewportSize();

    const response: FrameResponse = {
      session_id: sessionId,
      mime: 'image/jpeg',
      image: `data:image/jpeg;base64,${buffer.toString('base64')}`,
      width: viewport?.width || 0,
      height: viewport?.height || 0,
      captured_at: new Date().toISOString(),
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/frame`);
  }
}

/**
 * Update viewport size for the active recording page.
 *
 * POST /session/:id/record/viewport
 */
export async function handleRecordViewport(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config);
    const { width, height } = body as unknown as ViewportRequest;

    if (!width || !height || width <= 0 || height <= 0) {
      sendJson(res, 400, {
        error: 'INVALID_VIEWPORT',
        message: 'width and height must be positive numbers',
      });
      return;
    }

    await session.page.setViewportSize({
      width: Math.round(width),
      height: Math.round(height),
    });

    const viewport = session.page.viewportSize();
    const response: ViewportResponse = {
      session_id: sessionId,
      width: viewport?.width ?? Math.round(width),
      height: viewport?.height ?? Math.round(height),
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/viewport`);
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

  logger.debug('recording: action streamed', {
    status: response.status,
    actionType: action.actionType,
    sequenceNum: action.sequenceNum,
  });
}

/**
 * Clean up action buffer for a session
 */
export function cleanupSessionRecording(sessionId: string): void {
  removeRecordedActions(sessionId);
}
