const SERVICE_PATH = '/browser_automation_studio.v1.agentaccess.BrowserSessionService';
export const APP_MONITOR_URL = 'http://localhost:20000/apps/workspace';

function apiOrigin(raw) {
  if (!raw?.trim()) throw new Error('BAS_BROWSER_API_URL is required from the Agent Manager run context');
  const base = new URL(raw);
  if (!['http:', 'https:'].includes(base.protocol) || base.username || base.password || base.search || base.hash || !['', '/'].includes(base.pathname)) {
    throw new Error('BAS_BROWSER_API_URL must be an HTTP(S) API origin without credentials or a path');
  }
  return base.origin;
}

export function createAppMonitorBASClient({ apiURL, identityToken, grantReceipt, fetchImpl = fetch }) {
  if (!identityToken?.trim() || !grantReceipt?.trim()) throw new Error('Agent Manager identity and browser grant are required from the run context');
  const origin = apiOrigin(apiURL);
  const headers = {
    'Content-Type': 'application/json', Accept: 'application/json',
    'X-Agent-Identity-Token': identityToken,
    'X-Vrooli-Browser-Grant-Receipt': grantReceipt,
  };
  const connect = async (method, body) => {
    const response = await fetchImpl(`${origin}${SERVICE_PATH}/${method}`, {
      method: 'POST', redirect: 'error', headers, body: JSON.stringify(body),
    });
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(`BAS ${method} denied with HTTP ${response.status}`);
    return payload;
  };
  const sessionPath = (sessionId, suffix) => {
    if (!/^[A-Za-z0-9_-]{1,128}$/.test(sessionId ?? '')) throw new Error('BAS returned an invalid session ID');
    return `/browser/sessions/${encodeURIComponent(sessionId)}${suffix}`;
  };
  const postSession = async (sessionId, suffix, body) => {
    const response = await fetchImpl(`${origin}${sessionPath(sessionId, suffix)}`, {
      method: 'POST', redirect: 'error', headers, body: JSON.stringify(body),
    });
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(`BAS browser operation denied with HTTP ${response.status}`);
    return payload;
  };
  return {
    openSession({ viewportWidth, viewportHeight }) {
      return connect('OpenSession', { startUrl: APP_MONITOR_URL, reuseMode: 'fresh', viewportWidth, viewportHeight });
    },
    task(sessionId, operation, fields = {}) {
      return postSession(sessionId, '/task', { operation, ...fields });
    },
    capture(sessionId, phase, captureId) {
      const body = { phase };
      if (captureId) body.capture_id = captureId;
      return postSession(sessionId, '/app-monitor-capture', body);
    },
    closeSession(sessionId) {
      return postSession(sessionId, '/close', {});
    },
  };
}

export function createAppMonitorBASClientFromEnv(options = {}) {
  return createAppMonitorBASClient({
    apiURL: process.env.BAS_BROWSER_API_URL,
    identityToken: process.env.VROOLI_AGENT_IDENTITY_TOKEN,
    grantReceipt: process.env.VROOLI_BROWSER_GRANT_RECEIPT,
    ...options,
  });
}
