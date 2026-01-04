/**
 * Service Worker API Routes
 *
 * GET /session/:id/service-workers - List all service workers
 * DELETE /session/:id/service-workers - Unregister all service workers
 * DELETE /session/:id/service-workers/:scopeURL - Unregister specific SW
 */

import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import { sendError, sendJson } from '../middleware';

/**
 * GET /session/:id/service-workers
 * Returns list of registered service workers and control settings.
 */
export async function handleSessionServiceWorkers(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const swController = session.serviceWorkerController;

    if (!swController) {
      sendJson(res, 200, {
        session_id: sessionId,
        workers: [],
        control: { mode: 'allow' },
        message: 'Service worker controller not initialized',
      });
      return;
    }

    sendJson(res, 200, {
      session_id: sessionId,
      workers: swController.getWorkers(),
      control: swController.getControl(),
    });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/service-workers`);
  }
}

/**
 * DELETE /session/:id/service-workers
 * Unregister all service workers.
 */
export async function handleSessionServiceWorkersDelete(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const swController = session.serviceWorkerController;

    if (!swController) {
      sendJson(res, 200, {
        session_id: sessionId,
        unregistered_count: 0,
        message: 'Service worker controller not initialized',
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
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  scopeURL: string,
  sessionManager: SessionManager
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const swController = session.serviceWorkerController;

    if (!swController) {
      sendJson(res, 404, {
        error: 'Service worker controller not initialized',
      });
      return;
    }

    const decodedScopeURL = decodeURIComponent(scopeURL);
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
