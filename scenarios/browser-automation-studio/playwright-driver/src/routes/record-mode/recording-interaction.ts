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

import { createHash, randomUUID } from 'crypto';
import type { IncomingMessage, ServerResponse } from 'http';
import type { Page } from 'rebrowser-playwright';
import type { SessionManager } from '../../session';
import type { Config } from '../../config';
import { parseJsonBody, sendJson, sendError } from '../../middleware';
import { logger } from '../../utils';
import { RECORDING_FRAME_CACHE_TTL_MS } from '../../constants';
import { updateFrameStreamViewport } from '../../frame-streaming';
import type {
  NavigateRequest,
  NavigateResponse,
  ReloadRequest,
  ReloadResponse,
  GoBackRequest,
  GoBackResponse,
  GoForwardRequest,
  GoForwardResponse,
  NavigationStateResponse,
  ScreenshotRequest,
  ScreenshotResponse,
  InputRequest,
  PointerAction,
  FrameResponse,
  ViewportRequest,
  ViewportResponse,
  ActivePageRequest,
  ActivePageResponse,
  HistoryEntryCallback,
} from './types';
import { verifyScriptInjection } from '../../recording/verification';

/**
 * A single entry in the navigation stack.
 */
interface NavigationStackEntry {
  /** URL of the page */
  url: string;
  /** Title of the page */
  title: string;
  /** Timestamp when this entry was added/visited */
  timestamp: string;
}

/**
 * Navigation state tracking per session.
 * Tracks the current position in browser history to determine canGoBack/canGoForward.
 * Also tracks the full stack of URLs/titles for the right-click popup feature.
 */
interface NavigationState {
  /** Current position in history (0-indexed) */
  historyIndex: number;
  /** Total history length */
  historyLength: number;
  /** Navigation stack entries (URL/title for each position) */
  stack: NavigationStackEntry[];
}

/** Per-session navigation state */
const navigationStateMap = new Map<string, NavigationState>();

/**
 * Get or initialize navigation state for a session.
 */
function getNavigationState(sessionId: string): NavigationState {
  let state = navigationStateMap.get(sessionId);
  if (!state) {
    state = { historyIndex: 0, historyLength: 1, stack: [] };
    navigationStateMap.set(sessionId, state);
  }
  return state;
}

/**
 * Get the navigation stack for a session.
 * Returns entries for back stack, current, and forward stack.
 */
export function getNavigationStack(sessionId: string): {
  backStack: NavigationStackEntry[];
  current: NavigationStackEntry | null;
  forwardStack: NavigationStackEntry[];
} {
  const state = navigationStateMap.get(sessionId);
  if (!state || state.stack.length === 0) {
    return { backStack: [], current: null, forwardStack: [] };
  }

  const current = state.stack[state.historyIndex] ?? null;
  // Back stack: entries before current (most recent first)
  const backStack = state.stack.slice(0, state.historyIndex).reverse();
  // Forward stack: entries after current
  const forwardStack = state.stack.slice(state.historyIndex + 1);

  return { backStack, current, forwardStack };
}

/**
 * Clear navigation state for a session (call on session close).
 */
export function clearNavigationState(sessionId: string): void {
  navigationStateMap.delete(sessionId);
}

/**
 * Update navigation state after a navigate (goto) action.
 * When navigating to a new URL, forward history is cleared.
 */
function onNavigate(sessionId: string, url: string, title: string): void {
  const state = getNavigationState(sessionId);

  // Create the new entry
  const entry: NavigationStackEntry = {
    url,
    title,
    timestamp: new Date().toISOString(),
  };

  if (state.stack.length === 0) {
    // First navigation - add at index 0
    state.stack.push(entry);
    state.historyIndex = 0;
    state.historyLength = 1;
  } else {
    // Subsequent navigation - truncate forward history, then add new entry
    // Slice includes entries from 0 up to (and including) current index
    state.stack = state.stack.slice(0, state.historyIndex + 1);
    state.stack.push(entry);
    state.historyIndex = state.stack.length - 1;
    state.historyLength = state.stack.length;
  }
}

/**
 * Update stack entry after navigating back or forward.
 * The URL/title may have changed if the page did client-side navigation.
 */
function updateStackEntry(sessionId: string, url: string, title: string): void {
  const state = getNavigationState(sessionId);
  if (state.stack[state.historyIndex]) {
    state.stack[state.historyIndex] = {
      ...state.stack[state.historyIndex],
      url,
      title,
    };
  }
}

/**
 * Compute canGoBack and canGoForward from navigation state.
 */
function computeNavigationAbility(sessionId: string): { canGoBack: boolean; canGoForward: boolean } {
  const state = getNavigationState(sessionId);
  return {
    canGoBack: state.historyIndex > 0,
    canGoForward: state.historyIndex < state.historyLength - 1,
  };
}

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

// Frame cache TTL is defined in constants.ts (RECORDING_FRAME_CACHE_TTL_MS)

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

// ========================================================================
// History Callback Support
// ========================================================================

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
 * Exported for use by recording-lifecycle.ts for framenavigated events.
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

    // Verify recording script loaded after navigation
    // This is critical for detecting injection failures on external URLs
    try {
      const verification = await verifyScriptInjection(session.page);
      if (!verification.loaded) {
        logger.warn('recording: script not loaded after navigation', {
          sessionId,
          url: normalizedUrl,
          verification: {
            loaded: verification.loaded,
            ready: verification.ready,
            inMainContext: verification.inMainContext,
            handlersCount: verification.handlersCount,
            error: verification.error,
          },
        });
      } else if (!verification.ready) {
        logger.warn('recording: script loaded but not ready after navigation', {
          sessionId,
          url: normalizedUrl,
          verification: {
            loaded: verification.loaded,
            ready: verification.ready,
            inMainContext: verification.inMainContext,
            handlersCount: verification.handlersCount,
          },
        });
      } else if (!verification.inMainContext) {
        logger.warn('recording: script running in ISOLATED context (should be MAIN)', {
          sessionId,
          url: normalizedUrl,
          verification,
        });
      } else {
        logger.info('recording: script verified after navigation', {
          sessionId,
          url: normalizedUrl,
          handlersCount: verification.handlersCount,
          version: verification.version,
        });
      }
    } catch (verifyError) {
      logger.warn('recording: verification failed after navigation', {
        sessionId,
        url: normalizedUrl,
        error: verifyError instanceof Error ? verifyError.message : String(verifyError),
      });
    }

    // Clear frame cache after navigation to ensure fresh frames are captured
    clearFrameCache(sessionId);

    // Get updated page info
    const url = session.page.url();
    const title = await session.page.title().catch(() => '');

    // Update navigation state with URL/title (new navigation clears forward history)
    onNavigate(sessionId, url, title);

    const { canGoBack, canGoForward } = computeNavigationAbility(sessionId);

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

    // Emit history callback (fire-and-forget)
    const thumbnail = config.history.thumbnailEnabled
      ? await captureThumbnail(session.page, config.history.thumbnailQuality)
      : undefined;
    emitHistoryCallback(config, sessionId, url, title, 'navigate', thumbnail).catch(() => {
      // Error already logged in emitHistoryCallback
    });

    const response: NavigateResponse = {
      session_id: sessionId,
      url,
      title,
      can_go_back: canGoBack,
      can_go_forward: canGoForward,
      screenshot,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/navigate`);
  }
}

/**
 * Reload the current page in the recording session.
 *
 * POST /session/:id/record/reload
 */
export async function handleRecordReload(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config);
    const request = body as unknown as ReloadRequest;

    const waitUntil = request.wait_until || 'load';
    const timeoutMs = request.timeout_ms ?? config.execution.navigationTimeoutMs;

    await session.page.reload({ waitUntil, timeout: timeoutMs });

    // Clear frame cache after reload to ensure fresh frames are captured
    clearFrameCache(sessionId);

    // Get updated page info
    const url = session.page.url();
    const title = await session.page.title().catch(() => '');
    const { canGoBack, canGoForward } = computeNavigationAbility(sessionId);

    const response: ReloadResponse = {
      session_id: sessionId,
      url,
      title,
      can_go_back: canGoBack,
      can_go_forward: canGoForward,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/reload`);
  }
}

/**
 * Navigate back in browser history.
 *
 * POST /session/:id/record/go-back
 */
export async function handleRecordGoBack(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config);
    const request = body as unknown as GoBackRequest;

    const waitUntil = request.wait_until || 'load';
    const timeoutMs = request.timeout_ms ?? config.execution.navigationTimeoutMs;

    // Check if we can go back
    const stateBefore = getNavigationState(sessionId);
    if (stateBefore.historyIndex <= 0) {
      sendJson(res, 400, {
        error: 'CANNOT_GO_BACK',
        message: 'No history to go back to',
      });
      return;
    }

    // Perform the navigation
    const result = await session.page.goBack({ waitUntil, timeout: timeoutMs });

    if (result === null) {
      // goBack returns null if there was no history to navigate to
      sendJson(res, 400, {
        error: 'CANNOT_GO_BACK',
        message: 'No history to go back to',
      });
      return;
    }

    // Update navigation state
    stateBefore.historyIndex--;

    // Clear frame cache after navigation
    clearFrameCache(sessionId);

    // Get updated page info
    const url = session.page.url();
    const title = await session.page.title().catch(() => '');

    // Update the stack entry in case URL/title changed (e.g., SPA navigation)
    updateStackEntry(sessionId, url, title);

    const { canGoBack, canGoForward } = computeNavigationAbility(sessionId);

    // Note: We don't emit history callbacks for back navigation because
    // we're revisiting an existing history entry, not creating a new one.

    const response: GoBackResponse = {
      session_id: sessionId,
      url,
      title,
      can_go_back: canGoBack,
      can_go_forward: canGoForward,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/go-back`);
  }
}

/**
 * Navigate forward in browser history.
 *
 * POST /session/:id/record/go-forward
 */
export async function handleRecordGoForward(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config);
    const request = body as unknown as GoForwardRequest;

    const waitUntil = request.wait_until || 'load';
    const timeoutMs = request.timeout_ms ?? config.execution.navigationTimeoutMs;

    // Check if we can go forward
    const stateBefore = getNavigationState(sessionId);
    if (stateBefore.historyIndex >= stateBefore.historyLength - 1) {
      sendJson(res, 400, {
        error: 'CANNOT_GO_FORWARD',
        message: 'No forward history to navigate to',
      });
      return;
    }

    // Perform the navigation
    const result = await session.page.goForward({ waitUntil, timeout: timeoutMs });

    if (result === null) {
      // goForward returns null if there was no forward history
      sendJson(res, 400, {
        error: 'CANNOT_GO_FORWARD',
        message: 'No forward history to navigate to',
      });
      return;
    }

    // Update navigation state
    stateBefore.historyIndex++;

    // Clear frame cache after navigation
    clearFrameCache(sessionId);

    // Get updated page info
    const url = session.page.url();
    const title = await session.page.title().catch(() => '');

    // Update the stack entry in case URL/title changed (e.g., SPA navigation)
    updateStackEntry(sessionId, url, title);

    const { canGoBack, canGoForward } = computeNavigationAbility(sessionId);

    // Note: We don't emit history callbacks for forward navigation because
    // we're revisiting an existing history entry, not creating a new one.

    const response: GoForwardResponse = {
      session_id: sessionId,
      url,
      title,
      can_go_back: canGoBack,
      can_go_forward: canGoForward,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/go-forward`);
  }
}

/**
 * Get the current navigation state without performing any navigation.
 *
 * GET /session/:id/record/navigation-state
 */
export async function handleRecordNavigationState(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  _config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);

    const url = session.page.url();
    const title = await session.page.title().catch(() => '');
    const { canGoBack, canGoForward } = computeNavigationAbility(sessionId);

    const response: NavigationStateResponse = {
      session_id: sessionId,
      url,
      title,
      can_go_back: canGoBack,
      can_go_forward: canGoForward,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/navigation-state`);
  }
}

/**
 * Get the navigation stack for back/forward history popup.
 *
 * GET /session/:id/record/navigation-stack
 */
export async function handleRecordNavigationStack(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  _sessionManager: SessionManager,
  _config: Config
): Promise<void> {
  try {
    const { backStack, current, forwardStack } = getNavigationStack(sessionId);

    sendJson(res, 200, {
      session_id: sessionId,
      back_stack: backStack,
      current,
      forward_stack: forwardStack,
    });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/navigation-stack`);
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
