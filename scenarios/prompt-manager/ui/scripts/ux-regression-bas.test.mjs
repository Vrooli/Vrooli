import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { createBASUXClient, WORLD_URL } from './bas-ux-client.mjs';
import { runUXRegression } from './ux-regression-bas-core.mjs';

const OPEN = '/browser_automation_studio.v1.agentaccess.BrowserSessionService/OpenSession';
const CLOSE = '/browser_automation_studio.v1.agentaccess.BrowserSessionService/CloseSession';
const ref = n => `e${n}`;
const node = (role, name, n, children = []) => ({ role, name, ref: ref(n), children });

async function fakeOwner({ failOperation, failAt, redirectAt, overflow = false } = {}) {
  const calls = [];
  let observationCount = 0;
  let currentPath = '/world';
  let tab = '';
  const server = createServer(async (request, response) => {
    const chunks = [];
    for await (const chunk of request) chunks.push(chunk);
    const body = chunks.length ? JSON.parse(Buffer.concat(chunks).toString()) : {};
    calls.push({ path: request.url, headers: request.headers, body });
    const reply = (status, value) => { response.writeHead(status, { 'Content-Type': 'application/json' }); response.end(JSON.stringify(value)); };
    if (request.url === OPEN) return reply(200, { sessionId: 'pm-session' });
    if (request.url === CLOSE) return reply(200, { closed: true });
    if (request.url !== '/api/v1/browser/sessions/pm-session/task') return reply(404, {});
    if (body.operation === failOperation && calls.filter(call => call.body.operation === body.operation).length === failAt) return reply(403, { error: 'BROWSER_PERMISSION_DENIED' });
    if (body.operation === 'navigate') { currentPath = new URL(body.url).pathname; tab = ''; return reply(200, { ok: true }); }
    if (body.operation === 'click') {
      if (body.ref === ref(2)) tab = 'Agents';
      if (body.ref === ref(3)) currentPath = '/agents/a-1';
      if (body.ref === ref(4)) tab = 'Teams';
      if (body.ref === ref(5)) currentPath = '/teams/t-1';
      if (body.ref === ref(6)) tab = 'Runs';
      return reply(200, { ok: true });
    }
    if (body.operation === 'screenshot') {
      observationCount++;
      if (redirectAt === observationCount) currentPath = '/login';
      return reply(200, { url: `http://127.0.0.1:21235${currentPath}`, screenshot: { media_type: 'image/png' } });
    }
    if (body.operation === 'read') {
      const children = [];
      if (currentPath === '/world') children.push(node('tab', 'Agents', 2), node('tab', 'Teams', 4), node('tab', 'Runs', 6));
      if (currentPath === '/world' && tab === 'Agents') children.push(node('button', 'Agent One', 3));
      if (currentPath === '/world' && tab === 'Teams') children.push(node('button', 'Team One', 5));
      if (currentPath === '/world' && tab === 'Runs') children.push(node('paragraph', 'No runs yet', 7));
      return reply(200, { tree: {
        url: `http://127.0.0.1:21235`,
        layout: { viewportWidthPx: 1280, viewportHeightPx: 800, documentHeightPx: overflow ? 860 : 800 },
        root: node('document', 'Prompt Manager', 1, children),
      } });
    }
    return reply(400, {});
  });
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  return {
    apiURL: `http://127.0.0.1:${server.address().port}`, calls,
    close: () => new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())),
  };
}

function client(owner) {
  return createBASUXClient({ apiURL: owner.apiURL, identityToken: 'am-id', grantReceipt: 'am-grant' });
}

test('Prompt Manager UX flow uses fixed BAS authority and preserves agent/team/run checks', async () => {
  const owner = await fakeOwner();
  try {
    const result = await runUXRegression({ brokerClient: client(owner) });
    assert.deepEqual(result.checks, ['shell-fit', 'agent-detail', 'team-detail', 'runs-loaded']);
    const open = owner.calls.find(call => call.path === OPEN);
    assert.deepEqual(open.body, { startUrl: WORLD_URL, reuseMode: 'fresh', viewportWidth: 1280, viewportHeight: 800 });
    assert.equal(Object.hasOwn(open.body, 'permissionRef'), false);
    assert.equal(owner.calls.at(-1).path, CLOSE);
    for (const call of owner.calls) {
      assert.equal(call.headers['x-agent-identity-token'], 'am-id');
      assert.equal(call.headers['x-vrooli-browser-grant-receipt'], 'am-grant');
      assert.equal(JSON.stringify(call.body).includes('am-grant'), false);
    }
  } finally { await owner.close(); }
});

test('scope denial and redirect fail closed and always close the session', async () => {
  for (const options of [{ failOperation: 'click', failAt: 1 }, { redirectAt: 2 }]) {
    const owner = await fakeOwner(options);
    try {
      await assert.rejects(runUXRegression({ brokerClient: client(owner) }), /denied|expected world route/);
      assert.equal(owner.calls.at(-1).path, CLOSE);
    } finally { await owner.close(); }
  }
});

test('vertical overflow fails the review and closes the session', async () => {
  const owner = await fakeOwner({ overflow: true });
  try {
    await assert.rejects(runUXRegression({ brokerClient: client(owner) }), /overflows viewport/);
    assert.equal(owner.calls.at(-1).path, CLOSE);
  } finally { await owner.close(); }
});

test('client denies a missing Agent Manager grant before network access', () => {
  assert.throws(() => createBASUXClient({ apiURL: 'http://127.0.0.1:20000', identityToken: 'am-id' }), /identity and browser grant/);
});
