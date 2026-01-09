/**
 * Recording Pages
 *
 * Handles multi-tab page management and history callbacks for recording sessions.
 * Includes:
 * - Creating new pages/tabs
 * - Switching active page for frame streaming and input forwarding
 * - History callback support for navigation tracking
 */

import { randomUUID } from 'crypto';
import type { IncomingMessage, ServerResponse } from 'http';
import type { Page } from 'rebrowser-playwright';
import type { SessionManager } from '../../session';
import type { Config } from '../../config';
import { parseJsonBody, sendJson, sendError } from '../../middleware';
import { logger } from '../../utils';
import type { ActivePageRequest, ActivePageResponse, HistoryEntryCallback } from './types';
import { clearFrameCache } from './recording-frames';

// =============================================================================
// History Callback Support
// =============================================================================

/** Timeout for history callback requests */
const HISTORY_CALLBACK_TIMEOUT_MS = 5000;

/**
 * Capture a small thumbnail of the current page for history entries.
 * Returns base64-encoded JPEG or undefined on failure.
 */
export async function captureThumbnail(page: Page, quality: number): Promise<string | undefined> {
  try {
    const buffer = await page.screenshot({
      type: 'jpeg',
      quality,
      // Capture just the visible viewport, not full page
      fullPage: false,
    });
    return buffer.toString('base64');
  } catch (err) {
    logger.warn('recording: thumbnail capture failed', {
      error: err instanceof Error ? err.message : String(err),
    });
    return undefined;
  }
}

/**
 * Emit a history entry callback to the configured URL.
 * Fire-and-forget - failures are logged but don't affect navigation.
 * Exported for use by page-events.ts and recording-navigation.ts for framenavigated events.
 */
export async function emitHistoryCallback(
  config: Config,
  sessionId: string,
  url: string,
  title: string,
  navigationType: 'navigate' | 'back' | 'forward',
  thumbnail?: string
): Promise<void> {
  const callbackUrl = config.history.callbackUrl;
  if (!callbackUrl) {
    return; // History callback not configured
  }

  const payload: HistoryEntryCallback = {
    session_id: sessionId,
    entry: {
      id: randomUUID(),
      url,
      title,
      timestamp: new Date().toISOString(),
      thumbnail,
    },
    navigation_type: navigationType,
  };

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), HISTORY_CALLBACK_TIMEOUT_MS);

  try {
    const response = await fetch(callbackUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
      signal: controller.signal,
    });

    if (!response.ok) {
      logger.warn('recording: history callback returned error', {
        sessionId,
        status: response.status,
        statusText: response.statusText,
      });
    }
  } catch (err) {
    logger.warn('recording: history callback failed', {
      sessionId,
      error: err instanceof Error ? err.message : String(err),
    });
  } finally {
    clearTimeout(timeoutId);
  }
}

// =============================================================================
// Multi-Tab Page Handlers
// =============================================================================

/**
 * Create a new page (tab) in the recording session.
 *
 * POST /session/:id/record/new-page
 *
 * Creates a new browser tab and switches to it. The new page will trigger
 * the page_created event callback to notify the API.
 */
export async function handleRecordNewPage(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config);
    const request = body as { url?: string };

    // Default to about:blank if no URL provided
    const url = request.url || 'about:blank';

    // Create a new page in the browser context
    const newPage = await session.context.newPage();

    // Generate a UUID for the new page
    const pageId = crypto.randomUUID();

    // Navigate to the URL
    await newPage.goto(url, { waitUntil: 'domcontentloaded' }).catch(() => {
      // Ignore navigation errors for about:blank
    });

    // Register the page in session tracking
    session.pages.push(newPage);
    session.pageIdMap.set(pageId, newPage);
    session.pageToIdMap.set(newPage, pageId);

    // Switch to the new page
    session.currentPageIndex = session.pages.length - 1;
    session.page = newPage;

    // Clear frame cache for this session
    clearFrameCache(sessionId);

    // Get page info
    const title = await newPage.title().catch(() => '');

    logger.info('recording: new page created by user request', {
      sessionId,
      pageId,
      url,
      title,
      totalPages: session.pages.length,
    });

    sendJson(res, 201, {
      driver_page_id: pageId,
      url: newPage.url(),
      title,
    });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/new-page`);
  }
}

/**
 * Switch the active page for frame streaming and input forwarding.
 *
 * POST /session/:id/record/active-page
 *
 * This endpoint enables multi-tab recording by allowing the client to switch
 * which page receives frame streaming and input forwarding.
 *
 * The page_id is a UUID assigned by the driver when pages are created.
 * This ID is sent to the API in page_created events and stored for reference.
 */
export async function handleRecordActivePage(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config);
    const request = body as unknown as ActivePageRequest;

    if (!request.page_id) {
      sendJson(res, 400, {
        error: 'MISSING_PAGE_ID',
        message: 'page_id field is required',
      });
      return;
    }

    // Find the page by ID using the page ID map
    const targetPage = session.pageIdMap.get(request.page_id);

    if (!targetPage) {
      // List available page IDs for debugging
      const availableIds = Array.from(session.pageIdMap.keys());
      sendJson(res, 404, {
        error: 'PAGE_NOT_FOUND',
        message: `Page with ID ${request.page_id} not found.`,
        available_page_ids: availableIds,
      });
      return;
    }

    // Check if the page is still open
    if (targetPage.isClosed()) {
      sendJson(res, 410, {
        error: 'PAGE_CLOSED',
        message: `Page ${request.page_id} has been closed`,
      });
      return;
    }

    // Get the previous page ID for logging
    const previousPageId = session.pageToIdMap.get(session.page) || 'unknown';

    // Find the index of the target page in the pages array
    const pageIndex = session.pages.indexOf(targetPage);
    if (pageIndex !== -1) {
      session.currentPageIndex = pageIndex;
    }

    // Update session.page to point to the active page
    session.page = targetPage;

    // Clear frame cache for this session to ensure fresh frames after switch
    clearFrameCache(sessionId);

    // Get page info for response
    const url = targetPage.url();
    const title = await targetPage.title().catch(() => '');

    logger.info('recording: active page switched', {
      sessionId,
      previousPageId,
      newPageId: request.page_id,
      pageIndex,
      url,
      title,
    });

    const response: ActivePageResponse = {
      session_id: sessionId,
      active_page_id: request.page_id,
      url,
      title,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/active-page`);
  }
}
