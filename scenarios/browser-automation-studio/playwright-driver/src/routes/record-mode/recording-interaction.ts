/**
 * Recording Interaction Handlers
 *
 * Handles live browser interaction during recording:
 * - Navigation to URLs
 * - Screenshots
 * - Input forwarding (mouse, keyboard, wheel)
 * - Frame capture for live preview
 * - Viewport management
 */

import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../../session';
import type { Config } from '../../config';
import { parseJsonBody, sendJson, sendError } from '../../middleware';
import { logger } from '../../utils';
import type {
  NavigateRequest,
  NavigateResponse,
  ScreenshotRequest,
  ScreenshotResponse,
  InputRequest,
  PointerAction,
  FrameResponse,
  ViewportRequest,
  ViewportResponse,
} from './types';

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
