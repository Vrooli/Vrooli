const SERVICE_PATH = '/browser_automation_studio.v1.agentaccess.BrowserSessionService';
const WORLD_URL = 'http://127.0.0.1:21235/world';

function apiOrigin(raw) {
  if (!raw?.trim()) throw new Error('BAS_BROWSER_API_URL is required from the Agent Manager run context');
  const base = new URL(raw);
  if (!['http:', 'https:'].includes(base.protocol) || base.username || base.password || base.search || base.hash || !['', '/'].includes(base.pathname)) {
    throw new Error('BAS_BROWSER_API_URL must be an HTTP(S) API origin without credentials or a path');
  }
  return base.origin;
}

export function createBASUXClient({ apiURL, identityToken, grantReceipt, fetchImpl = fetch }) {
  if (!identityToken?.trim() || !grantReceipt?.trim()) throw new Error('Agent Manager identity and browser grant are required from the run context');
  const origin = apiOrigin(apiURL);
  const headers = {
    'Content-Type': 'application/json', Accept: 'application/json',
    'X-Agent-Identity-Token': identityToken,
    'X-Vrooli-Browser-Grant-Receipt': grantReceipt,
  };
  async function connect(method, body) {
    const response = await fetchImpl(`${origin}${SERVICE_PATH}/${method}`, { method: 'POST', redirect: 'error', headers, body: JSON.stringify(body) });
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(`BAS ${method} denied with HTTP ${response.status}`);
    return payload;
  }
  return {
    openWorldSession() {
      return connect('OpenSession', { startUrl: WORLD_URL, reuseMode: 'fresh', viewportWidth: 1280, viewportHeight: 800 });
    },
    async task(sessionId, operation, fields = {}) {
      if (!/^[A-Za-z0-9_-]{1,128}$/.test(sessionId)) throw new Error('BAS returned an invalid session ID');
      const response = await fetchImpl(`${origin}/api/v1/browser/sessions/${encodeURIComponent(sessionId)}/task`, {
        method: 'POST', redirect: 'error', headers, body: JSON.stringify({ operation, ...fields }),
      });
      const payload = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(`BAS task denied with HTTP ${response.status}`);
      return payload;
    },
    closeSession(sessionId) { return connect('CloseSession', { sessionId }); },
  };
}

export function createBASUXClientFromEnv(options = {}) {
  return createBASUXClient({
    apiURL: process.env.BAS_BROWSER_API_URL,
    identityToken: process.env.VROOLI_AGENT_IDENTITY_TOKEN,
    grantReceipt: process.env.VROOLI_BROWSER_GRANT_RECEIPT,
    ...options,
  });
}

export { WORLD_URL };
