/**
 * Service Worker API Routes
 *
 * GET /session/:id/service-workers - List all service workers
 * DELETE /session/:id/service-workers - Unregister all service workers
 * DELETE /session/:id/service-workers/:scopeURL - Unregister specific SW
 */

import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { SessionState } from '../types/session';
import { authorizeOperation } from '../session/permission-policy';
import { sendError, sendJson } from '../middleware';
import { requireSessionLease } from './session-lease';

function serviceWorkerDeleteDenied(session: SessionState, scopeURLs: string[]): string | undefined {
  const profile = session.spec?.permission_profile;
  if (!profile) return undefined;
  if (profile.approval !== 'none') return 'service-worker deletion has no driver approval receipt channel';
  let currentURL = '';
  try {
    currentURL = session.page.url();
  } catch {
    return 'current page URL unavailable';
  }
  const urls = [currentURL, ...scopeURLs];
  for (const url of urls) {
    const decision = authorizeOperation(
      profile,
      'service_worker_unregister',
      url,
      undefined,
      Date.now(),
      session.createdAt.getTime(),
      session.instructionCount
    );
    if (!decision.allowed) return decision.reason;
  }
  return undefined;
}

/**
 * GET /session/:id/service-workers
 * Returns list of registered service workers and control settings.
 */
export function handleSessionServiceWorkers(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    const owner = requireSessionLease(req, res, sessionId, sessionManager);
    if (!owner) return Promise.resolve();
    const session = owner.session;
    const swController = session.serviceWorkerController;

    if (!swController) {
      sendJson(res, 200, {
        session_id: sessionId,
        workers: [],
        control: { mode: 'allow' },
        message: 'Service worker controller not initialized',
      });
      return Promise.resolve();
    }

    sendJson(res, 200, {
      session_id: sessionId,
      workers: swController.getWorkers(),
      control: swController.getControl(),
    });
    return Promise.resolve();
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/service-workers`);
    return Promise.resolve();
  }
}

/**
 * DELETE /session/:id/service-workers
 * Unregister all service workers.
 */
export async function handleSessionServiceWorkersDelete(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    const owner = requireSessionLease(req, res, sessionId, sessionManager);
    if (!owner) return;
    const session = owner.session;
    const swController = session.serviceWorkerController;

    if (!swController) {
      sendJson(res, 200, {
        session_id: sessionId,
        unregistered_count: 0,
        message: 'Service worker controller not initialized',
      });
      return;
    }

    const workers = swController.getWorkers();
    const denied = serviceWorkerDeleteDenied(session, workers.length ? workers.map((worker) => worker.scopeURL) : []);
    if (denied) {
      sendJson(res, 403, {
        error: { code: 'BROWSER_PERMISSION_DENIED', message: denied, kind: 'auth', retryable: false },
      });
      return;
    }

    const count = await swController.unregisterAll();

    sendJson(res, 200, {
      session_id: sessionId,
      unregistered_count: count,
    });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/service-workers`);
  }
}

/**
 * DELETE /session/:id/service-workers/:scopeURL
 * Unregister a specific service worker by scope URL.
 */
export async function handleSessionServiceWorkerDelete(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  scopeURL: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    const owner = requireSessionLease(req, res, sessionId, sessionManager);
    if (!owner) return;
    const session = owner.session;
    const swController = session.serviceWorkerController;

    if (!swController) {
      sendJson(res, 404, {
        error: 'Service worker controller not initialized',
      });
      return;
    }

    const decodedScopeURL = decodeURIComponent(scopeURL);
    const denied = serviceWorkerDeleteDenied(session, [decodedScopeURL]);
    if (denied) {
      sendJson(res, 403, {
        error: { code: 'BROWSER_PERMISSION_DENIED', message: denied, kind: 'auth', retryable: false },
      });
      return;
    }
    const success = await swController.unregister(decodedScopeURL);

    if (success) {
      sendJson(res, 200, {
        session_id: sessionId,
        unregistered: decodedScopeURL,
      });
    } else {
      sendJson(res, 404, {
        error: `Service worker not found: ${decodedScopeURL}`,
      });
    }
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/service-workers/${scopeURL}`);
  }
}
