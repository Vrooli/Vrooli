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
import {
  enqueueIdempotentInput,
  enqueuePageMutation,
  getHeldPointerModifiers,
  InputIDConflictError,
  isPointerDown,
  releasePointerModifiers,
  setPointerDown,
  synchronizePointerModifiers,
} from '../../session/live-input';
import { updateFrameStreamViewport } from '../../frame-streaming';
import type { InputRequest, PointerAction, ViewportRequest, ViewportResponse } from './types';
import { recordingOwner } from './recording-ownership';

// =============================================================================
// Input Handlers
// =============================================================================

type PointerModifier = 'Alt' | 'Control' | 'Meta' | 'Shift';
const pointerModifiers = new Set<PointerModifier>(['Alt', 'Control', 'Meta', 'Shift']);

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

    if (request.type === 'pointer' && !['move', 'down', 'up', 'click'].includes(request.action || 'move')) {
      sendJson(res, 400, { error: 'INVALID_ACTION', message: 'Unsupported pointer action' });
      return;
    }
    if (request.type === 'keyboard' && !request.text && !request.key) {
      sendJson(res, 400, { error: 'MISSING_KEYBOARD_DATA', message: 'Provide key or text for keyboard input' });
      return;
    }
    if (!['pointer', 'keyboard', 'wheel'].includes(request.type)) {
      sendJson(res, 400, { error: 'INVALID_TYPE', message: 'Unsupported input type' });
      return;
    }

    if (request.input_id !== undefined && (typeof request.input_id !== 'string' || request.input_id.length === 0 || request.input_id.length > 128)) {
      sendJson(res, 400, { error: 'INVALID_INPUT_ID', message: 'input_id must be a non-empty string of at most 128 characters' });
      return;
    }

    const action = request.action || 'move';
    const coalesceKey = request.type === 'pointer' && action === 'move'
      ? JSON.stringify((request.modifiers ?? []).filter((modifier) => pointerModifiers.has(modifier as PointerModifier)).sort())
      : undefined;
    const inputPayload = Object.fromEntries(Object.entries(request).filter(
      ([key]) => key !== 'execution_id' && key !== 'lease_id' && key !== 'input_id',
    ));
    const receipt = await enqueueIdempotentInput(page, request.input_id, JSON.stringify(inputPayload), async () => {
      ownedSession();
      switch (request.type) {
        case 'pointer': {
          const x = request.x ?? 0;
          const y = request.y ?? 0;
          const button = request.button || 'left';
          const pointerAction: PointerAction = request.action || 'move';
          const held = getHeldPointerModifiers(page);
          const requested = new Set((request.modifiers ?? []).filter(
            (modifier): modifier is PointerModifier => pointerModifiers.has(modifier as PointerModifier),
          ));
          let downAttempted = false;
          let downCompleted = false;

          try {
            await synchronizePointerModifiers(page, requested, held, ownedSession);
            if (pointerAction === 'move') {
              await page.mouse.move(x, y);
            } else if (pointerAction === 'down') {
              await page.mouse.move(x, y);
              ownedSession();
              setPointerDown(page, true);
              downAttempted = true;
              await page.mouse.down({ button });
              downCompleted = true;
            } else if (pointerAction === 'up') {
              await page.mouse.move(x, y);
              ownedSession();
              await page.mouse.up({ button });
              setPointerDown(page, false);
            } else {
              await page.mouse.click(x, y, { button });
            }
          } catch (error) {
            if (pointerAction === 'down' && downAttempted && !downCompleted) {
              await page.mouse.up({ button }).catch(() => undefined);
              setPointerDown(page, false);
            }
            if (pointerAction === 'up') {
              // The browser may have applied the release before reporting a
              // transport error. Do not retain logical button ownership and
              // thereby suppress modifier recovery or leak it to the next
              // lease; retrying the uncertain button effect would be unsafe.
              setPointerDown(page, false);
            }
            throw error;
          } finally {
            const keepModifiersHeld = (pointerAction === 'down' && downCompleted)
              || ((pointerAction === 'move' || pointerAction === 'up') && isPointerDown(page));
            if (!keepModifiersHeld) await releasePointerModifiers(page, held);
          }
          break;
        }
        case 'wheel':
          await page.mouse.wheel(request.delta_x ?? 0, request.delta_y ?? 0);
          break;
        case 'keyboard':
          if (request.text) await page.keyboard.type(request.text);
          else {
            const key = request.key;
            if (!key) throw new Error('Keyboard key was not supplied after validation');
            const combo = modifiers.length > 0 ? `${modifiers.join('+')}+${key}` : key;
            await page.keyboard.press(combo);
          }
          break;
      }
      ownedSession();
    }, coalesceKey);
    sendJson(res, 200, receipt);
  } catch (error) {
    if (error instanceof InputIDConflictError) {
      sendJson(res, 409, { error: 'INPUT_ID_CONFLICT', message: error.message });
      return;
    }
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
    const pageId = session.pageBindings.getId(page);
    if (!pageId || body.expected_page_id !== pageId) {
      sendJson(res, 409, {error: 'PAGE_CHANGED', message: 'The selected recording tab changed before resize'});
      return;
    }
    const ownedPage = (): void => {
      if (ownedSession().page !== page || session.pageBindings.getId(page) !== pageId) throw new SessionNotFoundError(sessionId);
    };
    const {width, height} = body as unknown as ViewportRequest;
    if (typeof width !== 'number' || typeof height !== 'number' || !Number.isFinite(width) || !Number.isFinite(height) || Math.round(width) <= 0 || Math.round(height) <= 0) {
      sendJson(res, 400, {error: 'INVALID_VIEWPORT', message: 'width and height must be positive finite numbers'});
      return;
    }
    await enqueuePageMutation(page, async () => {
      ownedPage();
      await page.setViewportSize({width: Math.round(width), height: Math.round(height)});
      ownedPage();
      await updateFrameStreamViewport(sessionId, page);
      ownedPage();
    });
    ownedPage();
    const viewport = page.viewportSize();
    if (!viewport) throw new Error('Applied viewport is unavailable');
    const response: ViewportResponse = {session_id: sessionId, driver_page_id: pageId, ...viewport};

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/viewport`);
  }
}
