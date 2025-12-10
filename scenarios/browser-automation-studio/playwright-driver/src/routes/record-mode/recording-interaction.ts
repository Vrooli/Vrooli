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

import { createHash } from 'crypto';
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
 * Frame cache entry for avoiding redundant screenshots.
 * Caches both the raw buffer and the computed hash for ETag generation.
 *
 * NOTE: Playwright only supports 'png' and 'jpeg' screenshot formats.
 * WebP is NOT supported despite better compression. Do not attempt to use
 * type: 'webp' - it will fail at runtime with "expected one of (png|jpeg)".
 */
interface FrameCacheEntry {
  /** Raw JPEG buffer from Playwright screenshot */
  buffer: Buffer;
  /** MD5 hash of the buffer for content comparison */
  hash: string;
  /** Base64-encoded data URI (computed lazily) */
  base64DataUri: string;
  /** Viewport dimensions at capture time */
  width: number;
  height: number;
  /** Timestamp when this frame was captured */
  capturedAt: number;
  /** Quality setting used for this capture */
  quality: number;
  /** Whether this was a full-page capture */
  fullPage: boolean;
}

/** Per-session frame cache */
const frameCache = new Map<string, FrameCacheEntry>();

/** Cache TTL in milliseconds - frames older than this are considered stale */
const FRAME_CACHE_TTL_MS = 150;

/**
 * Compute MD5 hash of a buffer.
 * MD5 is fast and sufficient for content comparison (not security).
 */
function computeFrameHash(buffer: Buffer): string {
  return createHash('md5').update(buffer).digest('hex');
}

/**
 * Get cached frame if still valid, or null if cache miss/expired.
 */
function getCachedFrame(
  sessionId: string,
  quality: number,
  fullPage: boolean
): FrameCacheEntry | null {
  const cached = frameCache.get(sessionId);
  if (!cached) return null;

  const age = Date.now() - cached.capturedAt;

  // Cache miss if expired
  if (age > FRAME_CACHE_TTL_MS) {
    return null;
  }

  // Cache miss if quality/fullPage settings changed
  if (cached.quality !== quality || cached.fullPage !== fullPage) {
    return null;
  }

  return cached;
}

/**
 * Store frame in cache with computed hash.
 */
function cacheFrame(
  sessionId: string,
  buffer: Buffer,
  width: number,
  height: number,
  quality: number,
  fullPage: boolean
): FrameCacheEntry {
  const hash = computeFrameHash(buffer);
  const base64DataUri = `data:image/jpeg;base64,${buffer.toString('base64')}`;

  const entry: FrameCacheEntry = {
    buffer,
    hash,
    base64DataUri,
    width,
    height,
    capturedAt: Date.now(),
    quality,
    fullPage,
  };

  frameCache.set(sessionId, entry);
  return entry;
}

/**
 * Clear frame cache for a session (call on session close/navigation).
 */
export function clearFrameCache(sessionId: string): void {
  frameCache.delete(sessionId);
}

/**
 * Clear all frame caches (call on shutdown).
 */
export function clearAllFrameCaches(): void {
  frameCache.clear();
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

    // Normalize URL: add https:// if no protocol specified
    let normalizedUrl = request.url.trim();
    if (normalizedUrl && !normalizedUrl.match(/^[a-zA-Z][a-zA-Z0-9+.-]*:/)) {
      // No protocol found - add https://
      normalizedUrl = `https://${normalizedUrl}`;
    }

    const waitUntil: NavigateRequest['wait_until'] = request.wait_until || 'load';
    const timeoutMs = request.timeout_ms ?? config.execution.navigationTimeoutMs;

    await session.page.goto(normalizedUrl, { waitUntil, timeout: timeoutMs });

    // Clear frame cache after navigation to ensure fresh frames are captured
    clearFrameCache(sessionId);

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
 * Implements three optimizations:
 * 1. Server-side cache: Avoids redundant Playwright screenshot() calls within TTL
 * 2. Content hashing: MD5 hash enables reliable ETag comparison in API layer
 * 3. Skip identical frames: If new capture matches cached hash, return cached data
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

    // Check cache first - avoid expensive screenshot if we have a recent frame
    const cached = getCachedFrame(sessionId, quality, fullPage);
    if (cached) {
      // Return cached frame without re-capturing
      const response: FrameResponse = {
        session_id: sessionId,
        mime: 'image/jpeg',
        image: cached.base64DataUri,
        width: cached.width,
        height: cached.height,
        captured_at: new Date(cached.capturedAt).toISOString(),
        content_hash: cached.hash,
      };
      sendJson(res, 200, response);
      return;
    }

    // Cache miss or expired - capture new frame
    // NOTE: Playwright only supports 'png' and 'jpeg'. WebP would be ~25% smaller
    // but is not supported - do not use type: 'webp', it fails at runtime.
    const buffer = await session.page.screenshot({
      type: 'jpeg',
      fullPage,
      quality,
    });

    const viewport = session.page.viewportSize();
    const width = viewport?.width || 0;
    const height = viewport?.height || 0;

    // Compute hash and check if content actually changed from last cached frame
    const newHash = computeFrameHash(buffer);
    const previousCached = frameCache.get(sessionId);

    if (previousCached && previousCached.hash === newHash) {
      // Content identical to previous frame - update timestamp but reuse cached data
      // This saves base64 encoding overhead for static pages
      previousCached.capturedAt = Date.now();
      previousCached.quality = quality;
      previousCached.fullPage = fullPage;

      const response: FrameResponse = {
        session_id: sessionId,
        mime: 'image/jpeg',
        image: previousCached.base64DataUri,
        width: previousCached.width,
        height: previousCached.height,
        captured_at: new Date(previousCached.capturedAt).toISOString(),
        content_hash: newHash,
      };
      sendJson(res, 200, response);
      return;
    }

    // New frame content - cache it
    const entry = cacheFrame(sessionId, buffer, width, height, quality, fullPage);

    const response: FrameResponse = {
      session_id: sessionId,
      mime: 'image/jpeg',
      image: entry.base64DataUri,
      width: entry.width,
      height: entry.height,
      captured_at: new Date(entry.capturedAt).toISOString(),
      content_hash: entry.hash,
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
