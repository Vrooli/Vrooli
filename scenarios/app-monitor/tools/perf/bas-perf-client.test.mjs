import test from 'node:test';
import assert from 'node:assert/strict';
import { APP_MONITOR_URL, createAppMonitorBASClient } from './bas-perf-client.mjs';

function fakeFetch(responses = []) {
  const requests = [];
  const fetchImpl = async (url, options) => {
    requests.push({ url, options });
    const next = responses.shift() ?? { status: 200, payload: {} };
    return { ok: next.status >= 200 && next.status < 300, status: next.status, json: async () => next.payload };
  };
  return { fetchImpl, requests };
}

test('requires the trusted run context before any network request', () => {
  const { fetchImpl, requests } = fakeFetch();
  assert.throws(() => createAppMonitorBASClient({ apiURL: 'http://localhost', identityToken: '', grantReceipt: '', fetchImpl }), /Agent Manager identity/);
  assert.equal(requests.length, 0);
});

test('opens only the fixed App Monitor route and uses typed capture phases', async () => {
  const { fetchImpl, requests } = fakeFetch([
    { status: 200, payload: { sessionId: 'session_1' } },
    { status: 200, payload: { binding: { captureId: 'capture-1' } } },
    { status: 200, payload: { binding: { captureId: 'capture-1' }, metrics: {} } },
    { status: 204, payload: {} },
  ]);
  const client = createAppMonitorBASClient({ apiURL: 'http://bas.test/', identityToken: 'identity-secret', grantReceipt: 'grant-secret', fetchImpl });
  await client.openSession({ viewportWidth: 1440, viewportHeight: 900 });
  await client.capture('session_1', 'begin');
  await client.capture('session_1', 'after_mount', 'capture-1');
  await client.closeSession('session_1');

  assert.match(requests[0].url, /BrowserSessionService\/OpenSession$/);
  assert.equal(JSON.parse(requests[0].options.body).startUrl, APP_MONITOR_URL);
  assert.equal(JSON.parse(requests[0].options.body).reuseMode, 'fresh');
  assert.equal(requests[1].url, 'http://bas.test/browser/sessions/session_1/app-monitor-capture');
  assert.deepEqual(JSON.parse(requests[1].options.body), { phase: 'begin' });
  assert.deepEqual(JSON.parse(requests[2].options.body), { phase: 'after_mount', capture_id: 'capture-1' });
  for (const request of requests) {
    assert.equal(request.options.redirect, 'error');
    assert.equal(request.options.headers['X-Agent-Identity-Token'], 'identity-secret');
    assert.equal(request.options.headers['X-Vrooli-Browser-Grant-Receipt'], 'grant-secret');
    assert.doesNotMatch(request.options.body, /script|evaluate|cdp|profile|credential/i);
  }
});

test('does not follow redirects or include upstream error bodies', async () => {
  const { fetchImpl } = fakeFetch([{ status: 403, payload: { error: 'sensitive detail' } }]);
  const client = createAppMonitorBASClient({ apiURL: 'http://bas.test', identityToken: 'identity', grantReceipt: 'grant', fetchImpl });
  await assert.rejects(client.openSession({ viewportWidth: 1440, viewportHeight: 900 }), /HTTP 403/);
});
