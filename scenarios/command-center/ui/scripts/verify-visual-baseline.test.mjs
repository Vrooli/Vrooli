import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import pngjs from "/home/matthalloran8/Vrooli/scenarios/prompt-manager/ui/node_modules/pngjs/lib/png.js";
import { verifyVisualBaseline, TIMESTAMPS_MS, VIEWPORT } from './verify-visual-baseline-core.mjs';

const { PNG } = pngjs;
const png = (r) => PNG.sync.write({ width: 1, height: 1, data: Buffer.from([r, 20, 30, 255]) });
const ORIGIN = 'http://127.0.0.1:21235';

async function fixture({ changedAt } = {}) {
  const root = await mkdtemp(join(tmpdir(), 'cc-visual-baseline-'));
  const configRoot = join(root, 'rooms');
  const baselineRoot = join(root, 'baseline');
  await mkdir(configRoot, { recursive: true });
  await mkdir(join(baselineRoot, 'alpha'), { recursive: true });
  await writeFile(join(configRoot, 'alpha.json'), '{}\n');
  for (const timestamp of TIMESTAMPS_MS) {
    await writeFile(join(baselineRoot, 'alpha', `${timestamp}.png`), png(timestamp === changedAt ? 90 : 10));
  }
  const calls = [];
  let activeURL = '';
  const client = {
    async openSession(request) { calls.push({ operation: 'open', request }); return { sessionId: 'session-1' }; },
    async runTask(sessionId, request) {
      calls.push({ operation: 'task', sessionId, request });
      if (request.operation === 'navigate') { activeURL = request.url; return { sessionId, url: activeURL }; }
      return { sessionId, url: activeURL, screenshot: { data: png(10).toString('base64'), media_type: 'image/png', width: 1, height: 1 } };
    },
    async closeSession(sessionId) { calls.push({ operation: 'close', sessionId }); },
  };
  return { root, configRoot, baselineRoot, calls, client };
}

test('visual baseline uses one granted Broker session and exact canvas screenshot contract', async () => {
  const f = await fixture();
  try {
    const message = await verifyVisualBaseline({ brokerClient: f.client, baseURL: ORIGIN, configRoot: f.configRoot, baselineRoot: f.baselineRoot, accept: false, png: PNG });
    assert.equal(message, `Verified zero-diff baseline for 1 rooms × ${TIMESTAMPS_MS.length} timestamps.`);
    assert.deepEqual(f.calls.filter(call => call.operation === 'open')[0].request, { startUrl: ORIGIN + '/', viewportWidth: VIEWPORT.width, viewportHeight: VIEWPORT.height });
    const tasks = f.calls.filter(call => call.operation === 'task').map(call => call.request);
    assert.equal(tasks.length, TIMESTAMPS_MS.length * 2);
    for (let index = 0; index < tasks.length; index += 2) {
      assert.equal(tasks[index].operation, 'navigate');
      assert.match(tasks[index].url, /^http:\/\/127\.0\.0\.1:21235\/alpha\?visualDiff=1&visualDiffAt=/);
      assert.deepEqual(tasks[index + 1], { operation: 'screenshot', test_id: 'scene-canvas-pixels', wait_until: 'networkidle', settle_ms: 250 });
    }
    assert.deepEqual(f.calls.at(-1), { operation: 'close', sessionId: 'session-1' });
  } finally { await rm(f.root, { recursive: true, force: true }); }
});

test('pixel mismatch fails and still closes the owned session', async () => {
  const f = await fixture({ changedAt: 250 });
  try {
    await assert.rejects(verifyVisualBaseline({ brokerClient: f.client, baseURL: ORIGIN, configRoot: f.configRoot, baselineRoot: f.baselineRoot, accept: false, png: PNG }), /alpha\/250: 1 pixels differ/);
    assert.deepEqual(f.calls.at(-1), { operation: 'close', sessionId: 'session-1' });
  } finally { await rm(f.root, { recursive: true, force: true }); }
});

test('explicit accept mode replaces each golden and writes the manifest', async () => {
  const f = await fixture({ changedAt: 250 });
  try {
    const message = await verifyVisualBaseline({ brokerClient: f.client, baseURL: ORIGIN, configRoot: f.configRoot, baselineRoot: f.baselineRoot, accept: true, png: PNG });
    assert.match(message, /^Captured zero-diff baseline/);
    assert.equal(PNG.sync.read(await readFile(join(f.baselineRoot, 'alpha', '250.png'))).data[0], 10);
    assert.equal(JSON.parse(await readFile(join(f.baselineRoot, 'baseline.json'))).rooms[0], 'alpha');
    assert.deepEqual(f.calls.at(-1), { operation: 'close', sessionId: 'session-1' });
  } finally { await rm(f.root, { recursive: true, force: true }); }
});

test('redirect and bad observation evidence fail closed and close the owned session', async () => {
  const f = await fixture();
  const original = f.client.runTask;
  f.client.runTask = async (sessionId, request) => request.operation === 'screenshot'
    ? { sessionId, url: 'http://127.0.0.1:21235/login', screenshot: { data: 'not-png', media_type: 'image/png' } }
    : original(sessionId, request);
  try {
    await assert.rejects(verifyVisualBaseline({ brokerClient: f.client, baseURL: ORIGIN, configRoot: f.configRoot, baselineRoot: f.baselineRoot, accept: false, png: PNG }), /left the expected visual room route/);
    assert.deepEqual(f.calls.at(-1), { operation: 'close', sessionId: 'session-1' });
  } finally { await rm(f.root, { recursive: true, force: true }); }
});
