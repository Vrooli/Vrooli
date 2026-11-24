import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { SessionState } from '../types/session';
import { sendJson } from '../middleware';
import { authorizeOperation } from '../session/permission-policy';
import {
  observeAccessibilityTree,
  type AccessibilityTreeObservation,
} from '../tracing/accessibility-observation';
import { requireSessionLease } from './session-lease';

export type AccessibilityTreeObserver = (
  page: SessionState['page']
) => Promise<AccessibilityTreeObservation>;

/** GET /session/:id/accessibility-tree. The request identifies the expected
 * current page but cannot provide browser/CDP handles or caller identity. */
export async function handleSessionAccessibilityTree(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  observe: AccessibilityTreeObserver = observeAccessibilityTree
): Promise<void> {
  const owner = requireSessionLease(req, res, sessionId, sessionManager);
  if (!owner) return;
  const { session } = owner;
  const profile = session.spec?.permission_profile;
  if (!profile?.view) {
    sendJson(res, 403, {
      error: {
        code: 'BROWSER_PERMISSION_DENIED',
        message: 'browser viewing is disabled',
        kind: 'auth',
        retryable: false,
      },
    });
    return;
  }
  if (session.phase !== 'ready' && session.phase !== 'recording') {
    sendJson(res, 409, {
      error: {
        code: 'SESSION_BUSY',
        message: `Session cannot be observed while ${session.phase}`,
        kind: 'orchestration',
        retryable: true,
      },
    });
    return;
  }
  if (session.instructionInFlight) {
    sendJson(res, 409, {
      error: {
        code: 'SESSION_BUSY',
        message: 'A browser instruction is in flight',
        kind: 'orchestration',
        retryable: true,
      },
    });
    return;
  }

  const expectedPageID =
    typeof req.url === 'string'
      ? (new URL(req.url, 'http://localhost').searchParams.get('expected_page_id') ?? '')
      : '';
  const pageID = session.pageBindings.getId(session.page) ?? '';
  if (!expectedPageID || !pageID || expectedPageID !== pageID) {
    sendJson(res, 409, {
      error: {
        code: 'STALE_PAGE',
        message: 'Expected active page does not match the current session page',
        kind: 'orchestration',
        retryable: true,
      },
    });
    return;
  }

  let currentURL = '';
  try {
    currentURL = session.page.url();
  } catch {
    /* fail closed below */
  }
  const decision = authorizeOperation(
    profile,
    'observe',
    currentURL,
    currentURL,
    Date.now(),
    session.createdAt.getTime(),
    session.instructionCount
  );
  if (!decision.allowed) {
    sendJson(res, 403, {
      error: {
        code: 'BROWSER_PERMISSION_DENIED',
        message: decision.reason,
        kind: 'auth',
        retryable: false,
      },
    });
    return;
  }

  try {
    session.accessibilityReferenceTargets = new Map();
    const tree = await observe(session.page);
    // Discard a result if the selected page or its URL changed while CDP was
    // reading it. This never returns a snapshot of a different active target.
    if (
      session.pageBindings.getId(session.page) !== pageID ||
      session.page.url() !== currentURL ||
      session.instructionInFlight
    ) {
      sendJson(res, 409, {
        error: {
          code: 'STALE_PAGE',
          message: 'Active page changed during accessibility observation',
          kind: 'orchestration',
          retryable: true,
        },
      });
      return;
    }
    const references = tree.agentReferenceTargets ?? new Map();
    const expiresAt = Date.now() + 5_000;
    session.accessibilityReferenceTargets = new Map(
      [...references].map(([ref, target]) => [ref, { ...target, pageId: pageID, url: currentURL, expiresAt }])
    );
    // The selector table is non-enumerable and is never serialized. The only
    // target handle exposed to the caller is the opaque, short-lived ref.
    const publicTree = { ...tree };
    sendJson(res, 200, { session_id: sessionId, page_id: pageID, tree: publicTree });
  } catch {
    sendJson(res, 503, {
      error: {
        code: 'ACCESSIBILITY_OBSERVATION_UNAVAILABLE',
        message: 'accessibility observation is unavailable',
        kind: 'infra',
        retryable: true,
      },
    });
  }
}
