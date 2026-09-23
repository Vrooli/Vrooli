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
import { logger, SessionNotFoundError } from '../../utils';
import { recordingOwner } from './recording-ownership';
import type { ActivePageRequest, ActivePageResponse, HistoryEntryCallback } from './types';
import { clearFrameCache } from './recording-frames';

/** Read the existing document; tab chrome must never refetch its HTML. */
export async function readFaviconUrl(page: Page): Promise<string> {
  try {
    return await page.evaluate(() => {
      const icon = document.querySelector<HTMLLinkElement>('link[rel~="icon" i][href]');
      const url = new URL(icon?.href || '/favicon.ico', document.baseURI);
      return ['http:', 'https:'].includes(url.protocol) || url.href.startsWith('data:image/') ? url.href : '';
    });
  } catch {
    return '';
  }
}

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

/** Register a recording page once, including pages discovered by the context event. */
export function registerRecordingPage(
  session: ReturnType<SessionManager['getSession']>,
  page: Page
): string {
  const existing = session.pageToIdMap.get(page);
  if (existing) return existing;
  const id = randomUUID();
  if (!session.pages.includes(page)) session.pages.push(page);
  session.pageIdMap.set(id, page);
  session.pageToIdMap.set(page, id);
  return id;
}

// Explicit rollback and recording close callbacks converge on the same registry cleanup.
export function unregisterRecordingPage(session: ReturnType<SessionManager['getSession']>, page: Page): void {
  const id = session.pageToIdMap.get(page);
  if (id) session.pageIdMap.delete(id);
  session.pageToIdMap.delete(page);
  const index = session.pages.indexOf(page);
  if (index !== -1) session.pages.splice(index, 1);
  if (session.page === page) {
    session.frameStack.length = 0;
    const next = session.pages.find(candidate => !candidate.isClosed());
    if (next) session.page = next;
    clearFrameCache(session.id);
  }
  session.currentPageIndex = session.pages.indexOf(session.page);
}

function selectRecordingPage(session: ReturnType<SessionManager['getSession']>, page: Page): void {
  if (session.page !== page) session.frameStack.length = 0;
  session.page = page;
  session.currentPageIndex = session.pages.indexOf(page);
  clearFrameCache(session.id);
}

/** Close the admitted browser page and return the resulting browser selection. */
export async function handleRecordClosePage(
  req: IncomingMessage, res: ServerResponse, sessionId: string,
  sessionManager: SessionManager, config: Config
): Promise<void> {
  try {
    const body = await parseJsonBody(req, config);
    const ownedSession = recordingOwner(body, sessionId, sessionManager);
    const session = ownedSession();
    const { page_id: pageId } = body as unknown as ActivePageRequest;
    if (!pageId) {
      sendJson(res, 400, { error: 'MISSING_PAGE_ID', message: 'page_id field is required' });
      return;
    }
    const page = session.pageIdMap.get(pageId);
    if (!page) {
      sendJson(res, 404, { error: 'PAGE_NOT_FOUND', message: `Page ${pageId} not found` });
      return;
    }
    sessionManager.updateActivity(sessionId);
    await page.close();
    ownedSession();
    unregisterRecordingPage(session, page);
    sendJson(res, 200, { closed_page_id: pageId, active_page_id: session.pageToIdMap.get(session.page) ?? '' });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/close-page`);
  }
}

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
    const body = await parseJsonBody(req, config);
    const ownedSession = recordingOwner(body, sessionId, sessionManager);
    const session = ownedSession();
    sessionManager.updateActivity(sessionId);
    const request = body as { url?: string };

    // Default to about:blank if no URL provided
    const url = request.url || 'about:blank';

    // Create a new page in the browser context
    const newPage = await session.context.newPage();

    let pageId: string;
    let title: string;
    let faviconUrl: string;
    try {
      ownedSession();
      pageId = registerRecordingPage(session, newPage);
      await newPage.goto(url, { waitUntil: 'domcontentloaded', timeout: config.execution.navigationTimeoutMs });
      ownedSession();
      [title, faviconUrl] = await Promise.all([newPage.title().catch(() => ''), readFaviconUrl(newPage)]);
      ownedSession();
      if (newPage.isClosed() || session.pageIdMap.get(pageId) !== newPage) throw new SessionNotFoundError(sessionId);
    } catch (error) {
      try {
        await newPage.close();
        unregisterRecordingPage(session, newPage);
      } catch (cleanupError) {
        throw new Error(`New tab command failed: ${String(error)}; page cleanup failed: ${String(cleanupError)}`);
      }
      throw error;
    }

    // Switch to the new page
    selectRecordingPage(session, newPage);

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
      favicon_url: faviconUrl,
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
    const body = await parseJsonBody(req, config);
    const ownedSession = recordingOwner(body, sessionId, sessionManager);
    const session = ownedSession();
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

    sessionManager.updateActivity(sessionId);
    const title = await targetPage.title().catch(() => '');
    ownedSession();
    if (targetPage.isClosed() || session.pageIdMap.get(request.page_id) !== targetPage) throw new SessionNotFoundError(sessionId);

    // Get the previous page ID for logging
    const previousPageId = session.pageToIdMap.get(session.page) || 'unknown';

    selectRecordingPage(session, targetPage);

    // Get page info for response
    const url = targetPage.url();

    logger.info('recording: active page switched', {
      sessionId,
      previousPageId,
      newPageId: request.page_id,
      pageIndex: session.currentPageIndex,
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
