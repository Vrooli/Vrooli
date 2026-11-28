import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import fs from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { runCapture } from './preview-workspace-capture.mjs';
import { APP_MONITOR_URL } from './bas-perf-client.mjs';

const taskRevision = 'app-monitor.preview-workspace.performance-capture.v1';
const oracleRevision = 'app-monitor.preview-workspace.metrics.v1';
const traceJson = JSON.stringify({ traceEvents: [{ name: 'RunTask' }] });
const trace = {
  contract: 'app-monitor.preview-workspace.trace.v1', captureId: '550e8400-e29b-41d4-a716-446655440000',
  sessionId: 'session-1', byteLength: Buffer.byteLength(traceJson),
  sha256: createHash('sha256').update(traceJson).digest('hex'), traceJson,
};

function fixture({ failMetrics = false } = {}) {
  const requests = [];
  const files = new Map();
  const client = {
    async openSession(input) { requests.push(['open', input]); return { sessionId: 'session-1' }; },
    async task(sessionId, operation, fields) { requests.push(['task', sessionId, operation, fields]); return {}; },
    async capture(sessionId, phase, captureId) {
      requests.push(['capture', sessionId, phase, captureId]);
      const binding = {
        captureId: trace.captureId, sessionId, phase, runId: 'run-1',
        taskRevision, oracleRevision, profileRevision: 'a'.repeat(64), observedAt: new Date().toISOString(),
      };
      if (phase === 'begin') return { binding };
      if (phase === 'finish') return { binding, trace };
      const paneCount = failMetrics ? 1 : 8;
      return { binding, metrics: {
        contract: oracleRevision, iframeCount: 8, paneCount, tabCardCount: 8,
        profilerMarkCount: 3, workspaceStorageBytes: 2048,
      } };
    },
    async closeSession(sessionId) { requests.push(['close', sessionId]); },
  };
  const fsImpl = {
    async mkdir() {},
    async writeFile(file, value) { files.set(file, value); },
  };
  return { client, fsImpl, requests, files };
}

test('runs the workspace benchmark through one BAS session and records bound phases', async () => {
  const f = fixture();
  const result = await runCapture({ argv: ['--panes', '8', '--out', '/tmp/report.json', '--trace', '/tmp/trace.json'], client: f.client, fsImpl: f.fsImpl, log() {} });
  assert.equal(f.requests[0][0], 'open');
  assert.deepEqual(f.requests[0][1], { viewportWidth: 1440, viewportHeight: 1000 });
  assert.equal(f.requests.filter(([kind]) => kind === 'open').length, 1);
  assert.deepEqual(f.requests.filter(([kind]) => kind === 'capture').map(([, , phase]) => phase), [
    'begin', 'after_mount', 'after_scroll', 'after_picker', 'finish',
  ]);
  assert.ok(f.requests.some(([, , operation, fields]) => operation === 'scroll' && fields.delta_y === 1600));
  assert.ok(f.requests.some(([, , operation, fields]) => operation === 'keyboard' && fields.key === 'k'));
  assert.ok(f.requests.some(([, , operation, fields]) => operation === 'fill' && fields.value === 'scenario'));
  assert.deepEqual(result.counters.afterMount, { iframes: 8, panes: 8, tabCards: 8, profilerMarks: 3, storageBytes: 2048 });
  assert.equal(result.errors, null);
  assert.equal(result.errorCollection.status, 'unknown');
  assert.equal(result.capture.taskRevision, taskRevision);
  assert.equal(result.capture.oracleRevision, oracleRevision);
  assert.equal(f.files.get('/tmp/trace.json'), traceJson);
  assert.equal(f.requests.at(-1)[0], 'close');
});

test('rejects metric parity loss and still closes the BAS session', async () => {
  const f = fixture({ failMetrics: true });
  await assert.rejects(runCapture({ argv: ['--panes', '8', '--out', '/tmp/report.json'], client: f.client, fsImpl: f.fsImpl, log() {} }), /invalid workspace metrics/);
  assert.equal(f.requests.at(-1)[0], 'close');
});

test('uses the fixed workspace URL and no private launcher arguments', async () => {
  const f = fixture();
  await runCapture({ argv: ['--panes', '8', '--out', '/tmp/report.json'], client: f.client, fsImpl: f.fsImpl, log() {} });
  assert.equal(APP_MONITOR_URL, 'http://localhost:20000/apps/workspace');
  const navigations = f.requests.filter(([, , operation]) => operation === 'navigate');
  assert.ok(navigations.every(([, , , fields]) => fields.url.startsWith(APP_MONITOR_URL)));
  await assert.rejects(runCapture({ argv: ['--chrome', '/usr/bin/chrome'], client: f.client, fsImpl: f.fsImpl }), /unsupported argument/);
});

test('the production capture script contains no private browser launch or raw CDP lifecycle', async () => {
  const scriptPath = fileURLToPath(new URL('./preview-workspace-capture.mjs', import.meta.url));
  const source = await fs.readFile(scriptPath, 'utf8');
  for (const removedPath of [
    "node:child_process", 'remote-debugging-port', 'user-data-dir', 'Runtime.evaluate',
    'Tracing.start', 'Input.dispatchMouseEvent', 'createPageTarget',
  ]) {
    assert.equal(source.includes(removedPath), false, `superseded private path remains: ${removedPath}`);
  }
  assert.equal(source.includes('BAS_BROWSER_API_URL'), false, 'transport credentials belong only to the client module');
});
