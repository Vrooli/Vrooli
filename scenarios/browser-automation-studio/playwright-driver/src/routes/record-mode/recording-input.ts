/**
 * Recording Input
 *
 * Handles live input forwarding to the browser:
 * - Mouse/pointer events (move, click, down, up)
 * - Keyboard events (key press, text input)
 * - Wheel/scroll events
 * - Viewport size changes
 */

import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../../session';
import type { Config } from '../../config';
import { parseJsonBody, sendJson, sendError } from '../../middleware';
import { logger } from '../../utils';
import { updateFrameStreamViewport } from '../../frame-streaming';
import type { InputRequest, PointerAction, ViewportRequest, ViewportResponse } from './types';

// =============================================================================
// Input Handlers
// =============================================================================

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
 * Update viewport size for the active recording page.
 * Also updates the frame streaming viewport to restart screencast at new dimensions.
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

    const roundedWidth = Math.round(width);
    const roundedHeight = Math.round(height);

    // Update Playwright viewport
    await session.page.setViewportSize({
      width: roundedWidth,
      height: roundedHeight,
    });

    // Update frame streaming to restart screencast at new dimensions
    // This is async but we don't need to wait for it - the UI will get updated frames
    // when the screencast restarts. Fire and forget to keep response fast.
    updateFrameStreamViewport(sessionId, {
      width: roundedWidth,
      height: roundedHeight,
    }).then((result) => {
      if (!result.success && !result.skipped) {
        logger.warn('recording: frame stream viewport update failed', {
          sessionId,
          error: result.error,
          viewport: { width: roundedWidth, height: roundedHeight },
        });
      }
    }).catch((error) => {
      logger.error('recording: frame stream viewport update error', {
        sessionId,
        error: error instanceof Error ? error.message : String(error),
      });
    });

    const viewport = session.page.viewportSize();
    const response: ViewportResponse = {
      session_id: sessionId,
      width: viewport?.width ?? roundedWidth,
      height: viewport?.height ?? roundedHeight,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/viewport`);
  }
}
