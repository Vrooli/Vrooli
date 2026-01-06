/**
 * Recording Navigation
 *
 * Handles navigation operations for recording sessions:
 * - Navigate to URLs
 * - Reload page
 * - Go back/forward in history
 * - Get navigation state
 * - Get navigation stack (for back/forward popup)
 *
 * Tracks navigation state per session to determine canGoBack/canGoForward.
 */

import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../../session';
import type { Config } from '../../config';
import { parseJsonBody, sendJson, sendError } from '../../middleware';
import { logger } from '../../utils';
import { verifyScriptInjection } from '../../recording';
import { clearFrameCache } from './recording-frames';
import { captureThumbnail, emitHistoryCallback } from './recording-pages';
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
} from './types';

// =============================================================================
// Navigation State Infrastructure
// =============================================================================

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

// =============================================================================
// Navigation Handlers
// =============================================================================

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
