import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { mkdtemp, readFile, readdir, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { createBASBrowserClient } from './bas-browser-client.mjs';
import { captureSettingsReview, SETTINGS_URL, VIEWPORTS, VIEWPORT_HEIGHT } from './capture-settings-review-core.mjs';

const OPEN_PATH = '/browser_automation_studio.v1.agentaccess.BrowserSessionService/OpenSession';
const CLOSE_PATH = '/browser_automation_studio.v1.agentaccess.BrowserSessionService/CloseSession';
const PNG = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/vmsAAAAASUVORK5CYII=';

async function startFakeBAS({ redirectAtWidth, denyAtWidth, overflowAtWidth } = {}) {
  const calls = [];
  const server = createServer(async (request, response) => {
    const parts = [];
    for await (const part of request) parts.push(part);
    const body = parts.length ? JSON.parse(Buffer.concat(parts).toString()) : {};
    calls.push({ method: request.method, path: request.url, headers: request.headers, body });
    const reply = (status, value) => {
      response.writeHead(status, { 'Content-Type': 'application/json' });
      response.end(JSON.stringify(value));
    };
    if (request.url === OPEN_PATH) {
      return reply(200, { sessionId: `session-${body.viewportWidth}` });
    }
    if (request.url === CLOSE_PATH) return reply(200, { closed: true });
    const match = /^\/api\/v1\/browser\/sessions\/(session-(390|768))\/task$/.exec(request.url);
    if (!match) return reply(404, {});
    const width = Number(match[2]);
    if (width === denyAtWidth && body.operation === 'screenshot') return reply(403, { error: 'BROWSER_PERMISSION_DENIED' });
    if (body.operation === 'screenshot') {
      return reply(200, {
        session_id: match[1],
        url: width === redirectAtWidth ? 'http://127.0.0.1:24993/login' : SETTINGS_URL,
        screenshot: { data: PNG, media_type: 'image/png', width, height: VIEWPORT_HEIGHT },
      });
    }
    if (body.operation === 'read') {
      return reply(200, {
        session_id: match[1],
        tree: {
          url: 'http://127.0.0.1:24993',
          layout: {
            viewportWidthPx: width,
            documentWidthPx: width === overflowAtWidth ? width + 24 : width,
          },
        },
      });
    }
    return reply(400, { error: 'unsupported operation' });
  });
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  const address = server.address();
  return {
    apiURL: `http://127.0.0.1:${address.port}`,
    calls,
    close: () => new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())),
  };
}

test('BAS client opens fixed viewports, saves Broker screenshots and checks typed layout evidence', async () => {
  const fake = await startFakeBAS();
  const tempDir = await mkdtemp(join(tmpdir(), 'cc-broker-settings-'));
  const outputDir = join(tempDir, 'evidence');
  try {
    const client = createBASBrowserClient({
      apiURL: fake.apiURL,
      identityToken: 'am-identity-fixture',
      grantReceipt: 'am-browser-grant-fixture',
    });
    const result = await captureSettingsReview({ brokerClient: client, outputDir });
    assert.deepEqual(result.map(item => item.width), VIEWPORTS);
    assert.deepEqual((await readdir(outputDir)).sort(), ['settings-390.png', 'settings-768.png']);
    assert.equal((await readFile(join(outputDir, 'settings-390.png'))).subarray(0, 8).toString('hex'), '89504e470d0a1a0a');

    const opens = fake.calls.filter(call => call.path === OPEN_PATH);
    assert.deepEqual(opens.map(call => [call.body.startUrl, call.body.viewportWidth, call.body.viewportHeight, call.body.reuseMode]), [
      [SETTINGS_URL, 390, VIEWPORT_HEIGHT, 'fresh'],
      [SETTINGS_URL, 768, VIEWPORT_HEIGHT, 'fresh'],
    ]);
    for (const open of opens) assert.equal(Object.hasOwn(open.body, 'permissionRef'), false, 'caller must not select the trusted profile');
    assert.deepEqual(fake.calls.filter(call => call.path.endsWith('/task')).map(call => call.body.operation), ['screenshot', 'read', 'screenshot', 'read']);
    for (const call of fake.calls) {
      assert.equal(call.headers['x-agent-identity-token'], 'am-identity-fixture');
      assert.equal(call.headers['x-vrooli-browser-grant-receipt'], 'am-browser-grant-fixture');
      assert.equal(JSON.stringify(call.body).includes('am-browser-grant-fixture'), false, 'grant must stay out of request bodies');
    }
    assert.equal(fake.calls.filter(call => call.path === CLOSE_PATH).length, 2);
  } finally {
    await fake.close();
    await rm(tempDir, { recursive: true, force: true });
  }
});

test('BAS client requires the existing Agent Manager run identity and grant', () => {
  assert.throws(() => createBASBrowserClient({ apiURL: 'http://127.0.0.1:20000', identityToken: 'identity' }), /identity and browser grant/);
  assert.throws(() => createBASBrowserClient({ apiURL: 'http://127.0.0.1:20000', grantReceipt: 'grant' }), /identity and browser grant/);
});

test('out-of-scope denial and unexpected redirects stop before layout observation', async () => {
  for (const scenario of [{ denyAtWidth: 390 }, { redirectAtWidth: 390 }]) {
    const fake = await startFakeBAS(scenario);
    const tempDir = await mkdtemp(join(tmpdir(), 'cc-broker-settings-denied-'));
    const outputDir = join(tempDir, 'evidence');
    try {
      const client = createBASBrowserClient({ apiURL: fake.apiURL, identityToken: 'identity', grantReceipt: 'grant' });
      await assert.rejects(captureSettingsReview({ brokerClient: client, outputDir }), /BAS task denied|fixed Command Center settings route/);
      assert.deepEqual(fake.calls.filter(call => call.path.endsWith('/task')).map(call => call.body.operation), ['screenshot']);
      assert.equal(fake.calls.filter(call => call.path === CLOSE_PATH).length, 1);
      assert.deepEqual(await readdir(outputDir), []);
    } finally {
      await fake.close();
      await rm(tempDir, { recursive: true, force: true });
    }
  }
});

test('horizontal overflow fails the review and still closes the owned session', async () => {
  const fake = await startFakeBAS({ overflowAtWidth: 390 });
  const tempDir = await mkdtemp(join(tmpdir(), 'cc-broker-settings-overflow-'));
  const outputDir = join(tempDir, 'evidence');
  try {
    const client = createBASBrowserClient({ apiURL: fake.apiURL, identityToken: 'identity', grantReceipt: 'grant' });
    await assert.rejects(captureSettingsReview({ brokerClient: client, outputDir }), /horizontal overflow at 390px/);
    assert.deepEqual(fake.calls.filter(call => call.path.endsWith('/task')).map(call => call.body.operation), ['screenshot', 'read']);
    assert.equal(fake.calls.filter(call => call.path === CLOSE_PATH).length, 1);
  } finally {
    await fake.close();
    await rm(tempDir, { recursive: true, force: true });
  }
});
