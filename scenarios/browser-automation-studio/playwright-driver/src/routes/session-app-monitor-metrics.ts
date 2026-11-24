import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { SessionState } from '../types/session';
import { sendJson } from '../middleware';
import { authorizeOperation } from '../session/permission-policy';
import { requireSessionLease } from './session-lease';

export const APP_MONITOR_METRICS_CONTRACT = 'app-monitor.preview-workspace.metrics.v1';

export interface AppMonitorMetricValues {
  iframeCount: number;
  paneCount: number;
  tabCardCount: number;
  profilerMarkCount: number;
  workspaceStorageBytes: number;
}

export interface AppMonitorMetrics extends AppMonitorMetricValues {
  contract: typeof APP_MONITOR_METRICS_CONTRACT;
  sessionId: string;
  pageId: string;
  url: string;
  observedAt: string;
}

export type AppMonitorMetricsReader = (session: SessionState) => Promise<AppMonitorMetricValues>;

const metricKeys = [
  'iframeCount',
  'paneCount',
  'tabCardCount',
  'profilerMarkCount',
  'workspaceStorageBytes',
] as const;

function metricsAreValid(value: unknown): value is AppMonitorMetricValues {
  if (!value || typeof value !== 'object') return false;
  const record = value as Record<string, unknown>;
  return (
    metricKeys.every((key) => Number.isSafeInteger(record[key]) && (record[key] as number) >= 0) &&
    Object.keys(record).every((key) => (metricKeys as readonly string[]).includes(key))
  );
}

/** Read a fixed App Monitor metric envelope. The caller cannot supply script,
 * selector, URL, trace category, or output path. */
export async function handleSessionAppMonitorMetrics(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  read: AppMonitorMetricsReader = readFixedAppMonitorMetrics
): Promise<void> {
  if (
    typeof req.url !== 'string' ||
    new URL(req.url, 'http://localhost').search ||
    new URL(req.url, 'http://localhost').hash
  ) {
    sendJson(res, 400, {
      error: {
        code: 'INVALID_REQUEST',
        message: 'metric observation accepts no query or fragment',
      },
    });
    return;
  }
  const owner = requireSessionLease(req, res, sessionId, sessionManager);
  if (!owner) return;
  const { session } = owner;
  const profile = session.spec?.permission_profile;
  if (
    !profile?.view ||
    !profile.app_monitor_metrics ||
    session.spec?.browser_backend !== 'managed'
  ) {
    sendJson(res, 403, {
      error: {
        code: 'BROWSER_PERMISSION_DENIED',
        message: 'App Monitor metric capture is disabled',
        kind: 'auth',
        retryable: false,
      },
    });
    return;
  }
  if ((session.phase !== 'ready' && session.phase !== 'recording') || session.instructionInFlight) {
    sendJson(res, 409, {
      error: {
        code: 'SESSION_BUSY',
        message: 'Session cannot be observed now',
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
  let parsed: URL | undefined;
  try {
    parsed = new URL(currentURL);
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
  if (
    !parsed ||
    parsed.pathname !== '/apps/workspace' ||
    parsed.search ||
    parsed.hash ||
    !decision.allowed
  ) {
    sendJson(res, 403, {
      error: {
        code: 'BROWSER_PERMISSION_DENIED',
        message: 'Current route is outside the App Monitor metric scope',
        kind: 'auth',
        retryable: false,
      },
    });
    return;
  }
  const pageId = session.pageBindings.getId(session.page) ?? '';
  if (!pageId) {
    sendJson(res, 409, {
      error: {
        code: 'STALE_PAGE',
        message: 'Active browser page is unavailable',
        kind: 'orchestration',
        retryable: true,
      },
    });
    return;
  }
  try {
    const metrics = await read(session);
    if (
      !metricsAreValid(metrics) ||
      session.pageBindings.getId(session.page) !== pageId ||
      session.page.url() !== currentURL ||
      session.instructionInFlight
    ) {
      sendJson(res, 409, {
        error: {
          code: 'STALE_PAGE',
          message: 'Active page changed during metric observation',
          kind: 'orchestration',
          retryable: true,
        },
      });
      return;
    }
    sendJson(res, 200, {
      contract: APP_MONITOR_METRICS_CONTRACT,
      sessionId,
      pageId,
      url: currentURL,
      observedAt: new Date().toISOString(),
      ...metrics,
    });
  } catch {
    sendJson(res, 502, {
      error: {
        code: 'METRIC_READ_FAILED',
        message: 'App Monitor metric observation failed',
        kind: 'orchestration',
        retryable: true,
      },
    });
  }
}

async function readFixedAppMonitorMetrics(session: SessionState): Promise<AppMonitorMetricValues> {
  return session.page.evaluate(() => {
    const workspace = localStorage.getItem('app-monitor:preview-workspace-v1') ?? '';
    return {
      iframeCount: document.querySelectorAll('.preview-pane iframe').length,
      paneCount: document.querySelectorAll('.preview-pane').length,
      tabCardCount: document.querySelectorAll('.tab-card').length,
      profilerMarkCount: performance
        .getEntriesByType('measure')
        .filter((entry) => entry.name.includes('React') || entry.name.includes('Profiler')).length,
      workspaceStorageBytes: new Blob([workspace]).size,
    };
  });
}
