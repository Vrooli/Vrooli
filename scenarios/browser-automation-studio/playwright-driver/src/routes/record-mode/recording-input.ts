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
import { SessionNotFoundError } from '../../utils';
import { updateFrameStreamViewport } from '../../frame-streaming';
import type { InputRequest, PointerAction, ViewportRequest, ViewportResponse } from './types';
import { recordingOwner } from './recording-ownership';

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
    const body = await parseJsonBody(req, config);
    const request = body as unknown as InputRequest;

    if (!request?.type) {
      sendJson(res, 400, {
        error: 'MISSING_TYPE',
        message: 'type field is required',
      });
      return;
    }

    const ownedSession = recordingOwner(body, sessionId, sessionManager);
    const session = ownedSession();
    sessionManager.updateActivity(sessionId);
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
          ownedSession();
          await page.mouse.down({ button });
        } else if (action === 'up') {
          await page.mouse.move(x, y);
          ownedSession();
          await page.mouse.up({ button });
        } else if (action === 'click') {
          await page.mouse.click(x, y, { button });
        } else {
          sendJson(res, 400, {
            error: 'INVALID_ACTION',
            message: 'Unsupported pointer action',
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
          message: 'Unsupported input type',
        });
        return;
    }

    ownedSession();
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
    const body = await parseJsonBody(req, config);
    const ownedSession = recordingOwner(body, sessionId, sessionManager);
    const session = ownedSession();
    const page = session.page;
    const pageId = session.pageToIdMap.get(page);
    if (!pageId || body.expected_page_id !== pageId) {
      sendJson(res, 409, {error: 'PAGE_CHANGED', message: 'The selected recording tab changed before resize'});
      return;
    }
    const ownedPage = () => {
      if (ownedSession().page !== page || session.pageToIdMap.get(page) !== pageId) throw new SessionNotFoundError(sessionId);
    };
    const {width, height} = body as unknown as ViewportRequest;
    if (typeof width !== 'number' || typeof height !== 'number' || !Number.isFinite(width) || !Number.isFinite(height) || Math.round(width) <= 0 || Math.round(height) <= 0) {
      sendJson(res, 400, {error: 'INVALID_VIEWPORT', message: 'width and height must be positive finite numbers'});
      return;
    }
    await page.setViewportSize({width: Math.round(width), height: Math.round(height)});
    ownedPage();
    await updateFrameStreamViewport(sessionId, page);
    ownedPage();
    const viewport = page.viewportSize();
    if (!viewport) throw new Error('Applied viewport is unavailable');
    const response: ViewportResponse = {session_id: sessionId, driver_page_id: pageId, ...viewport};

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/viewport`);
  }
}
