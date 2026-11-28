import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { SessionState } from '../types/session';
import { parseJsonBody, sendJson } from '../middleware';
import { authorizeOperation } from '../session/permission-policy';
import { AppMonitorTraceCapture } from '../tracing/app-monitor-capture';
import { requireSessionLease } from './session-lease';

function allowed(session: SessionState): boolean {
  const profile = session.spec?.permission_profile;
  if (!profile?.view || !profile.app_monitor_trace || session.spec?.browser_backend !== 'managed')
    return false;
  if (session.phase !== 'ready' && session.phase !== 'recording') return false;
  if (session.instructionInFlight) return false;
  let currentURL = '';
  try {
    currentURL = session.page.url();
  } catch {
    return false;
  }
  let parsed: URL;
  try {
    parsed = new URL(currentURL);
  } catch {
    return false;
  }
  return (
    parsed.pathname === '/apps/workspace' &&
    parsed.search === '' &&
    parsed.hash === '' &&
    authorizeOperation(
      profile,
      'observe',
      currentURL,
      currentURL,
      Date.now(),
      session.createdAt.getTime(),
      session.instructionCount
    ).allowed
  );
}

export async function handleSessionAppMonitorCapture(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  phase: 'begin' | 'finish',
  sessionManager: SessionManager
): Promise<void> {
  const owner = requireSessionLease(req, res, sessionId, sessionManager);
  if (!owner) return;
  const { session } = owner;
  let body: Record<string, unknown>;
  try {
    body = await parseJsonBody(req, {});
  } catch {
    sendJson(res, 400, {
      error: { code: 'INVALID_REQUEST', message: 'capture request is invalid' },
    });
    return;
  }
  if (
    Object.keys(body).some((key) => key !== 'capture_id') ||
    typeof body.capture_id !== 'string' ||
    !/^[0-9a-f-]{36}$/i.test(body.capture_id)
  ) {
    sendJson(res, 400, { error: { code: 'INVALID_REQUEST', message: 'capture ID is required' } });
    return;
  }
  if (!allowed(session)) {
    sendJson(res, 403, {
      error: {
        code: 'BROWSER_PERMISSION_DENIED',
        message: 'App Monitor trace capture is disabled or out of scope',
        kind: 'auth',
        retryable: false,
      },
    });
    return;
  }
  const captureId = body.capture_id;
  if (phase === 'begin') {
    const profile = session.spec?.permission_profile;
    if (session.appMonitorTraceCapture || session.appMonitorTraceCaptureId || !profile) {
      sendJson(res, 409, {
        error: {
          code: 'CAPTURE_STATE_INVALID',
          message: 'App Monitor trace capture is already active',
          kind: 'orchestration',
          retryable: false,
        },
      });
      return;
    }
    const pageId = session.pageBindings.getId(session.page) ?? '';
    try {
      session.appMonitorTraceCapture = await AppMonitorTraceCapture.start(
        captureId,
        profile.app_monitor_trace_max_duration_ms ?? 0,
        profile.app_monitor_trace_max_bytes ?? 0,
        session.page
      );
      session.appMonitorTraceCaptureId = captureId;
      if (!pageId || session.pageBindings.getId(session.page) !== pageId || !allowed(session)) {
        session.appMonitorTraceCapture = undefined;
        session.appMonitorTraceCaptureId = undefined;
        sendJson(res, 409, {
          error: {
            code: 'STALE_PAGE',
            message: 'Active page changed during trace start',
            kind: 'orchestration',
            retryable: true,
          },
        });
        return;
      }
      sendJson(res, 200, {
        contract: 'app-monitor.preview-workspace.trace.v1',
        sessionId,
        captureId,
        status: 'started',
      });
    } catch {
      session.appMonitorTraceCapture = undefined;
      session.appMonitorTraceCaptureId = undefined;
      sendJson(res, 502, {
        error: {
          code: 'TRACE_START_FAILED',
          message: 'App Monitor trace could not start',
          kind: 'orchestration',
          retryable: true,
        },
      });
    }
    return;
  }
  if (!session.appMonitorTraceCapture || session.appMonitorTraceCaptureId !== captureId) {
    sendJson(res, 404, {
      error: {
        code: 'CAPTURE_NOT_FOUND',
        message: 'App Monitor trace capture was not found',
        kind: 'orchestration',
        retryable: false,
      },
    });
    return;
  }
  const pageId = session.pageBindings.getId(session.page) ?? '';
  const capture = session.appMonitorTraceCapture;
  session.appMonitorTraceCapture = undefined;
  session.appMonitorTraceCaptureId = undefined;
  try {
    const artifact = await capture.finish();
    if (!pageId || session.pageBindings.getId(session.page) !== pageId || !allowed(session)) {
      sendJson(res, 409, {
        error: {
          code: 'STALE_PAGE',
          message: 'Active page changed during trace completion',
          kind: 'orchestration',
          retryable: true,
        },
      });
      return;
    }
    if (artifact.captureId !== captureId) {
      sendJson(res, 409, {
        error: {
          code: 'CAPTURE_MISMATCH',
          message: 'App Monitor trace capture binding changed',
          kind: 'auth',
          retryable: false,
        },
      });
      return;
    }
    sendJson(res, 200, { ...artifact, sessionId, pageId });
  } catch {
    sendJson(res, 502, {
      error: {
        code: 'TRACE_CAPTURE_FAILED',
        message: 'App Monitor trace capture failed or exceeded its cap',
        kind: 'orchestration',
        retryable: false,
      },
    });
  }
}
