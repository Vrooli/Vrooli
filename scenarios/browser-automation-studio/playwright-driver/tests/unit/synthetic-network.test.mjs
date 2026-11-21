import test from 'node:test';
import assert from 'node:assert/strict';
import { SYNTHETIC_PROFILES, createSyntheticRouteHandler, isApplicationApi, syntheticResponse } from '../../scripts/synthetic-network.mjs';

const profile = SYNTHETIC_PROFILES['web-console'];

test('Web Console API and fetch traffic is classified as application data', () => {
  assert.equal(isApplicationApi('http://localhost:3000/health', 'fetch', profile), true);
  assert.equal(isApplicationApi('http://localhost:3000/vrooli.web_console.v1.session.SessionService/List', 'xhr', profile), true);
  assert.equal(isApplicationApi('http://localhost:3000/api/v1/sessions/x/file-previews/y', 'image', profile), true);
  assert.equal(isApplicationApi('https://elsewhere.invalid/account', 'fetch', profile), true);
  assert.equal(isApplicationApi('http://localhost:3000/assets/app.js', 'script', profile), false);
});

test('synthetic responses contain no captured request or account data', () => {
  assert.deepEqual(syntheticResponse('http://localhost:3000/health', profile), {
    status: 200,
    contentType: 'application/json; charset=utf-8',
    body: JSON.stringify(profile.health),
  });
  assert.equal(syntheticResponse('http://localhost:3000/api/v1/business-accounts', profile).body, '{}');
});

test('opt-in Web Console session profile hydrates a wholly synthetic workspace', () => {
  const sessionProfile = SYNTHETIC_PROFILES['web-console-session'];
  const session = JSON.parse(syntheticResponse(
    'http://localhost:3000/vrooli.web_console.v1.sessions.SessionsService/List', sessionProfile,
  ).body);
  const layout = JSON.parse(syntheticResponse(
    'http://localhost:3000/vrooli.web_console.v1.workspace.WorkspaceService/GetLayout', sessionProfile,
  ).body);

  assert.deepEqual(session.sessions.map(({ id, displayLabel, origin }) => ({ id, displayLabel, origin })), [
    { id: 'synthetic-session-0001', displayLabel: 'Synthetic session', origin: 'SESSION_ORIGIN_UI' },
  ]);
  assert.equal(session.recovery.inProgress, false);
  assert.equal(layout.activePane, 'synthetic-session-0001');
  assert.deepEqual(layout.panes.map(({ sessionId }) => sessionId), ['synthetic-session-0001']);
  assert.equal(syntheticResponse(
    'http://localhost:3000/api/v1/coding-backup?harness=&session_id=synthetic-session-0001', sessionProfile,
  ).body, 'null');
  assert.equal(syntheticResponse(
    'http://localhost:3000/vrooli.web_console.v1.sessions.SessionsService/ListArchived', sessionProfile,
  ).body, JSON.stringify({ sessions: [], total: 0 }));
  assert.equal(syntheticResponse(
    'http://localhost:3000/vrooli.web_console.v1.sessions.SessionsService/Create', sessionProfile,
  ).body, '{}');
});

test('session fixture response is opt-in and never changes the base profile', () => {
  assert.equal(syntheticResponse(
    'http://localhost:3000/vrooli.web_console.v1.sessions.SessionsService/List', profile,
  ).body, '{}');
  assert.equal(SYNTHETIC_PROFILES['web-console-session'].responses['/vrooli.web_console.v1.sessions.SessionsService/List'].sessions[0].id,
    'synthetic-session-0001');
});

test('synthetic event streams return the required media type and a bounded fixture', () => {
  const response = syntheticResponse('http://localhost:21233/api/v1/events/stream', SYNTHETIC_PROFILES['web-console'], 'eventsource');
  assert.match(response.contentType, /^text\/event-stream/);
  assert.equal(response.body, 'event: synthetic-ready\ndata: {}\n\n');
});

test('route handler continues only the served document and same-origin static assets', async () => {
  const records = [];
  const handler = createSyntheticRouteHandler({ origin: 'http://localhost:3000', profile, onRecord: (row) => records.push(row) });
  const outcome = async ({ url, type, method = 'GET', navigation = false }) => {
    const result = [];
    const request = { url: () => url, resourceType: () => type, method: () => method, isNavigationRequest: () => navigation };
    await handler({ request: () => request, continue: async () => result.push('continue'), fulfill: async (r) => result.push(['fulfill', r]), abort: async () => result.push('abort') });
    return result[0];
  };
  assert.equal(await outcome({ url: 'http://localhost:3000/', type: 'document', navigation: true }), 'continue');
  assert.equal(await outcome({ url: 'http://localhost:3000/assets/app.js', type: 'script' }), 'continue');
  assert.equal((await outcome({ url: 'http://localhost:3000/api/v1/sessions', type: 'fetch' }))[0], 'fulfill');
  assert.equal(await outcome({ url: 'https://elsewhere.invalid/track', type: 'script' }), 'abort');
  assert.equal(await outcome({ url: 'http://localhost:3000/api/v1/sessions', type: 'fetch', method: 'POST' }).then((v) => v[0]), 'fulfill');
  assert.equal(records.length, 3);
  assert.ok(records.every((row) => row.outcome === 'synthetic' || row.outcome === 'blocked'));
});
