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
 * The active browser page owns navigation history.
 */

import type { IncomingMessage, ServerResponse } from 'http';
import { isOperational, type SessionManager } from '../../session';
import type { Page } from 'rebrowser-playwright';
import { createCDPSession, detachCDPSession } from '../../session/cdp-session';
import type { Config } from '../../config';
import { parseJsonBody, sendJson, sendError } from '../../middleware';
import { logger, SessionNotFoundError } from '../../utils';
import { recordingOwner } from './recording-lifecycle';
import { verifyScriptInjection } from '../../recording';
import { clearFrameCache } from './recording-frames';
import { captureThumbnail, emitHistoryCallback } from './recording-pages';
import type { NavigateRequest, NavigationResponse, NavigationStateResponse } from './types';

// Every history observation owns a short-lived attachment to the original page.
async function readBrowserHistory(page: Page, assertCurrent: () => void) {
  const cdp = await createCDPSession(page);
  let history;
  try {
    assertCurrent();
    history = await cdp.send('Page.getNavigationHistory');
  } finally {
    await detachCDPSession(cdp);
  }
  assertCurrent();
  return history;
}

function navigationAbility(history: Awaited<ReturnType<typeof readBrowserHistory>>) {
  return {
    can_go_back: history.currentIndex > 0,
    can_go_forward: history.currentIndex < history.entries.length - 1,
  };
}

type NavigationOperation = 'navigate' | 'reload' | 'go-back' | 'go-forward';

// All recording navigation uses the same lease/page admission and completion.
function navigationHandler(operation: NavigationOperation) {
  return async (
    req: IncomingMessage,
    res: ServerResponse,
    sessionId: string,
    sessionManager: SessionManager,
    config: Config
  ): Promise<void> => {
    try {
      const body = await parseJsonBody(req, config);
      const ownedSession = recordingOwner(body, sessionId, sessionManager);
      const session = ownedSession();
      const page = session.page;
      const ownedPage = () => {
        if (ownedSession().page !== page) throw new SessionNotFoundError(sessionId);
      };
      sessionManager.updateActivity(sessionId);
      const request = body as unknown as NavigateRequest;
      const options = { waitUntil: request.wait_until || 'load', timeout: request.timeout_ms ?? config.execution.navigationTimeoutMs };
      const direction = operation === 'go-back' ? -1 : operation === 'go-forward' ? 1 : 0;
      const before = direction ? await readBrowserHistory(page, ownedPage) : undefined;
      const noHistory = () => sendJson(res, 400, {
        error: direction < 0 ? 'CANNOT_GO_BACK' : 'CANNOT_GO_FORWARD',
        message: direction < 0 ? 'No history to go back to' : 'No forward history to navigate to',
      });
      let normalizedUrl = request.url;
      if (operation === 'navigate') {
        if (!normalizedUrl || typeof normalizedUrl !== 'string') {
          sendJson(res, 400, { error: 'MISSING_URL', message: 'url field is required' });
          return;
        }
        normalizedUrl = normalizedUrl.trim();
        if (normalizedUrl && !normalizedUrl.match(/^[a-zA-Z][a-zA-Z0-9+.-]*:/)) normalizedUrl = `https://${normalizedUrl}`;
        await page.goto(normalizedUrl, options);
      } else if (operation === 'reload') {
        await page.reload(options);
      } else {
        if (!before || before.currentIndex + direction < 0 || before.currentIndex + direction >= before.entries.length) return noHistory();
        // Same-document navigation can succeed with a null Playwright response.
        await (direction < 0 ? page.goBack(options) : page.goForward(options));
      }
      ownedPage();
      if (operation === 'navigate') {
        // Verify recording script loaded after navigation
        // This is critical for detecting injection failures on external URLs
        try {
          const verification = await verifyScriptInjection(page);
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

        ownedPage();
      }
      const url = page.url();
      const title = await page.title().catch(() => '');
      ownedPage();
      let screenshot: string | undefined;
      if (operation === 'navigate' && request.capture) {
        try {
          const buffer = await page.screenshot({ fullPage: true, type: 'jpeg', quality: 70 });
          screenshot = `data:image/jpeg;base64,${buffer.toString('base64')}`;
        } catch (error) {
          logger.warn('recording: screenshot capture failed', { sessionId, error: error instanceof Error ? error.message : String(error) });
        }
        ownedPage();
      }
      const thumbnail = operation === 'navigate' && config.history.thumbnailEnabled
        ? await captureThumbnail(page, config.history.thumbnailQuality) : undefined;
      ownedPage();

      const history = await readBrowserHistory(page, ownedPage);
      if (before && history.entries[history.currentIndex]?.id === before.entries[before.currentIndex]?.id) return noHistory();
      // Publish cache/callback/response only after all awaited work is current.
      clearFrameCache(sessionId);
      if (operation === 'navigate') {
        emitHistoryCallback(config, sessionId, url, title, 'navigate', thumbnail).catch(() => {
          // The callback owner logs delivery failures.
        });
      }
      const response: NavigationResponse = {
        session_id: sessionId, url, title,
        ...navigationAbility(history), screenshot,
      };
      sendJson(res, 200, response);
    } catch (error) {
      sendError(res, error as Error, `/session/${sessionId}/record/${operation}`);
    }
  };
}

export const handleRecordNavigate = navigationHandler('navigate');
export const handleRecordReload = navigationHandler('reload');
export const handleRecordGoBack = navigationHandler('go-back');
export const handleRecordGoForward = navigationHandler('go-forward');

// Read handlers bind the selected Session/page, without requiring mutation authority.
function navigationReadHandler(stack: boolean) {
  return async (
    _req: IncomingMessage, res: ServerResponse, sessionId: string,
    sessionManager: SessionManager, _config: Config
  ): Promise<void> => {
    try {
      const session = sessionManager.getSession(sessionId);
      const page = session.page;
      const assertCurrent = () => {
        if (sessionManager.peekSession(sessionId) !== session || session.page !== page || !isOperational(session.phase)) {
          throw new SessionNotFoundError(sessionId);
        }
      };
      assertCurrent();
      const title = stack ? '' : await page.title().catch(() => '');
      assertCurrent();
      const history = await readBrowserHistory(page, assertCurrent);
      if (stack) {
        const entries = history.entries.map(({ url, title }) => ({ url, title }));
        sendJson(res, 200, {
          session_id: sessionId,
          back_stack: entries.slice(0, history.currentIndex).reverse(),
          current: entries[history.currentIndex] ?? null,
          forward_stack: entries.slice(history.currentIndex + 1),
        });
      } else {
        const response: NavigationStateResponse = {
          session_id: sessionId, url: page.url(), title, ...navigationAbility(history),
        };
        sendJson(res, 200, response);
      }
    } catch (error) {
      sendError(res, error as Error, `/session/${sessionId}/record/navigation-${stack ? 'stack' : 'state'}`);
    }
  };
}

export const handleRecordNavigationState = navigationReadHandler(false);
export const handleRecordNavigationStack = navigationReadHandler(true);
