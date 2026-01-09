/**
 * Recording Frames
 *
 * Handles frame capture and screenshot operations for recording sessions.
 * Includes caching to avoid redundant screenshot() calls.
 *
 * Frame Cache Optimizations:
 * 1. Server-side cache: Avoids redundant Playwright screenshot() calls within TTL
 * 2. Content hashing: MD5 hash enables reliable ETag comparison in API layer
 * 3. Skip identical frames: If new capture matches cached hash, return cached data
 */

import { createHash } from 'crypto';
import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../../session';
import type { Config } from '../../config';
import { parseJsonBody, sendJson, sendError } from '../../middleware';
import { RECORDING_FRAME_CACHE_TTL_MS } from '../../constants';
import type { ScreenshotRequest, ScreenshotResponse, FrameResponse } from './types';

// =============================================================================
// Frame Cache Infrastructure
// =============================================================================

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

  // Cache miss if expired (TTL from constants.ts)
  if (age > RECORDING_FRAME_CACHE_TTL_MS) {
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

// =============================================================================
// Frame/Screenshot Handlers
// =============================================================================

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
      // Still fetch current page metadata even for cached frames
      const cachedPageTitle = await session.page.title().catch(() => '');
      const cachedPageUrl = session.page.url();

      // Return cached frame without re-capturing
      const response: FrameResponse = {
        session_id: sessionId,
        mime: 'image/jpeg',
        image: cached.base64DataUri,
        width: cached.width,
        height: cached.height,
        captured_at: new Date(cached.capturedAt).toISOString(),
        content_hash: cached.hash,
        page_title: cachedPageTitle,
        page_url: cachedPageUrl,
      };
      sendJson(res, 200, response);
      return;
    }

    // Cache miss or expired - capture new frame
    // NOTE: Playwright only supports 'png' and 'jpeg'. WebP would be ~25% smaller
    // but is not supported - do not use type: 'webp', it fails at runtime.
    const viewport = session.page.viewportSize();
    const width = viewport?.width || 0;
    const height = viewport?.height || 0;

    // Get page metadata for history/display purposes
    const pageTitle = await session.page.title().catch(() => '');
    const pageUrl = session.page.url();

    // Use explicit clip for viewport-only capture to ensure consistent behavior
    // and reduce bandwidth for pages with scrollable content
    const screenshotOptions: Parameters<typeof session.page.screenshot>[0] = {
      type: 'jpeg',
      quality,
    };

    if (fullPage) {
      screenshotOptions.fullPage = true;
    } else if (viewport) {
      // Clip to viewport for consistent, smaller frames
      screenshotOptions.clip = {
        x: 0,
        y: 0,
        width: viewport.width,
        height: viewport.height,
      };
    }

    const buffer = await session.page.screenshot(screenshotOptions);

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
        page_title: pageTitle,
        page_url: pageUrl,
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
      page_title: pageTitle,
      page_url: pageUrl,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/frame`);
  }
}
