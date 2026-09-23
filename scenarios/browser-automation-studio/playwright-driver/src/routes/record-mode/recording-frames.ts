/**
 * Recording Frames
 *
 * Identical preview reads share pending capture and a short-lived result.
 * Source, geometry and fidelity bound reuse; content hashes support HTTP ETags.
 */

import { createHash } from 'crypto';
import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../../session';
import type { Config } from '../../config';
import { parseJsonBody, sendJson, sendError } from '../../middleware';
import { RECORDING_FRAME_CACHE_TTL_MS } from '../../constants';
import { captureFrameSource, sameFrameSource } from '../../frame-streaming/frame';
import type { ScreenshotRequest, ScreenshotResponse, FrameResponse } from './types';

// =============================================================================
// Frame Cache Infrastructure
// =============================================================================

/**
 * Frame cache entry for avoiding redundant screenshots.
 * Caches the encoded image and hash within one immutable frame source.
 *
 * NOTE: Playwright only supports 'png' and 'jpeg' screenshot formats.
 * WebP is NOT supported despite better compression. Do not attempt to use
 * type: 'webp' - it will fail at runtime with "expected one of (png|jpeg)".
 */
interface CapturedFrame {
  key: string;
  /** MD5 hash of the buffer for content comparison */
  hash: string;
  /** Base64-encoded JPEG data URI */
  base64DataUri: string;
  /** Viewport dimensions at capture time */
  width: number;
  height: number;
  /** Timestamp when this frame was captured */
  capturedAt: number;
}

interface FrameCacheSlot {
  frame?: CapturedFrame;
  pending?: { key: string; result: Promise<CapturedFrame | null> };
}

/** One cache lifetime per session; deletion also retires pending captures. */
const frameCache = new Map<string, FrameCacheSlot>();

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
 * Shares identical in-flight reads and caches completed frames within the TTL.
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
    const page = session.page;
    const source = captureFrameSource(session, page);
    const url = new URL(req.url || '', 'http://localhost');
    const requestedPage = url.searchParams.get('page_id');
    const rejectChanged = () => sendJson(res, 409, {error:'FRAME_SOURCE_CHANGED', message:'The preview page or session lease changed'});
    if (!source || (requestedPage && requestedPage !== source.page_id)) { rejectChanged(); return; }
    const quality = Number(url.searchParams.get('quality')) || 60;
    const fullPage = url.searchParams.get('full_page') === 'true';
    const scale = session.spec.frame_scale ?? 'css';
    const viewport = page.viewportSize();
    const pageUrl = page.url();
    const slot: FrameCacheSlot = frameCache.get(sessionId) ?? {};
    frameCache.set(sessionId, slot);
    const owns = () => {
      const currentViewport = page.viewportSize();
      return frameCache.get(sessionId) === slot && page.url() === pageUrl
        && currentViewport?.width === viewport?.width && currentViewport?.height === viewport?.height
        && sameFrameSource(captureFrameSource(sessionManager.getSession(sessionId), page), source);
    };
    const key = JSON.stringify([source, quality, fullPage, scale, viewport, pageUrl]);
    const pageTitle = await page.title().catch(() => '');
    if (!owns()) { rejectChanged(); return; }

    let frame = slot.frame;
    if (!frame || frame.key !== key || Date.now() - frame.capturedAt > RECORDING_FRAME_CACHE_TTL_MS) {
      let pending = slot.pending;
      if (!pending || pending.key !== key) {
        const options: Parameters<typeof page.screenshot>[0] = { type: 'jpeg', quality, scale };
        if (fullPage) options.fullPage = true;
        else if (viewport) options.clip = { x: 0, y: 0, width: viewport.width, height: viewport.height };
        pending = { key, result: Promise.resolve().then(async () => {
          try {
            const buffer = await page.screenshot(options);
            if (!owns()) return null;
            const captured: CapturedFrame = {
              key, hash: createHash('md5').update(buffer).digest('hex'),
              base64DataUri: `data:image/jpeg;base64,${buffer.toString('base64')}`,
              width: viewport?.width ?? 0, height: viewport?.height ?? 0, capturedAt: Date.now(),
            };
            // An older fidelity request may finish, but cannot replace newer work.
            if (slot.pending === pending) slot.frame = captured;
            return captured;
          } finally {
            if (slot.pending === pending) slot.pending = undefined;
          }
        }) };
        slot.pending = pending;
      }
      frame = await pending.result ?? undefined;
    }
    if (!frame || !owns()) { rejectChanged(); return; }
    const response: FrameResponse = {
      session_id: sessionId,
      source,
      mime: 'image/jpeg',
      image: frame.base64DataUri,
      width: frame.width,
      height: frame.height,
      captured_at: new Date(frame.capturedAt).toISOString(),
      content_hash: frame.hash,
      page_title: pageTitle,
      page_url: pageUrl,
    };
    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/frame`);
  }
}
