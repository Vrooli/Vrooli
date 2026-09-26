import { handleRecordStart, handleRecordStop } from '../../src/routes/record-mode/recording-lifecycle';
import { handleStreamSettings } from '../../src/routes/record-mode/recording-diagnostics-routes';
import { appendFile, mkdir } from 'node:fs/promises';
import { dirname } from 'node:path';
/**
 * Pipeline E2E Tests
 *
 * Tests the full recording pipeline in a CI-friendly way:
 * - Self-contained (no external server needed)
 * - Tests all event types (click, input, scroll, focus, navigation)
 * - Validates TimelineEntry structure
 * - Mirrors the diagnostic pipeline test logic from self-test.ts
 *
 * These tests ensure the recording pipeline works correctly and can be
 * run as part of `pnpm test` in CI environments.
 */

import { setupPageLifecycleListeners } from '../../src/routes/record-mode/page-events';
import { handleRecordNewPage } from '../../src/routes/record-mode/recording-pages';
import { createMockHttpRequest, createMockHttpResponse } from '../helpers';
import { chromium, Browser, BrowserContext, Page } from 'rebrowser-playwright';
import * as http from 'http';
import { once } from 'node:events';
import { WebSocketServer } from 'ws';
import { SessionManager } from '../../src/session/manager';
import { createTestConfig } from '../helpers/test-config';
import * as driverConfig from '../../src/config';
import { PollingStrategy, CdpScreencastStrategy, type StreamingHandle } from '../../src/frame-streaming/strategies';
import { startFrameStreaming, stopFrameStreaming, getFrameStreamSettings } from '../../src/frame-streaming';
import {
  createRecordingContextInitializer,
  createRecordingPipelineManager,
  RecordingContextInitializer,
  RecordingPipelineManager,
  TimelineEntry,
  waitForScriptReady,
  getTimelineEntries,
  acknowledgeTimelineEntries,
} from '../../src/recording';
import { ActionType } from '../../src/proto/recording';

// Increase timeout for comprehensive E2E testing
jest.setTimeout(120000);

/**
 * Test page HTML with all interactive elements needed for pipeline testing.
 * Includes scrollable content, buttons, inputs, and navigation links.
 */
const TEST_PAGE_HTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Pipeline Test Page</title>
  <style>
    body { min-height: 200vh; padding: 20px; font-family: sans-serif; }
    .section { margin: 20px 0; padding: 20px; background: #f5f5f5; border-radius: 8px; }
    button, input { padding: 10px 20px; margin: 5px; }
    .spacer { height: 500px; }
  </style>
</head>
<body>
  <h1>Pipeline E2E Test Page</h1>

  <div class="section">
    <h2>Click Test</h2>
    <button id="test-btn" data-testid="test-button">Click Me</button>
  </div>

  <div class="section">
    <h2>Input Test</h2>
    <input type="text" id="test-input" data-testid="test-input" placeholder="Type here" />
  </div>

  <div class="section">
    <h2>Navigation Test</h2>
    <a id="test-link" href="/page-2">Navigate to Page 2</a>
  </div>

  <div class="spacer"></div>
  <div class="section">
    <p>Bottom of page - scroll content</p>
  </div>
  <script>
    window.__fixtureClickCount = 0;
    document.addEventListener('click', (event) => {
      const target = event.target;
      if (target instanceof Element && target.closest('#test-btn')) window.__fixtureClickCount += 1;
    }, { capture: true });
  </script>
</body>
</html>`;

const PAGE_2_HTML = `<!DOCTYPE html>
<html>
<head><title>Page 2</title></head>
<body style="min-height: 200vh; padding: 20px;">
  <h1>Page 2</h1>
  <button id="page-2-btn">Button on Page 2</button>
  <a id="back-link" href="/">Back to Main</a>
</body>
</html>`;

/**
 * Test server providing pages with interactive elements.
 */
class PipelineTestServer {
  private server: http.Server | null = null;
  private port = 0;

  async start(): Promise<number> {
    return new Promise((resolve) => {
      this.server = http.createServer((req, res) => {
        const path = req.url || '/';

        if (path === '/' || path === '/index.html') {
          res.writeHead(200, { 'Content-Type': 'text/html' });
          res.end(TEST_PAGE_HTML);
          return;
        }

        if (path === '/page-2') {
          res.writeHead(200, { 'Content-Type': 'text/html' });
          res.end(PAGE_2_HTML);
          return;
        }

        res.writeHead(404);
        res.end('Not Found');
      });

      this.server.listen(0, () => {
        const server = this.server;
        if (!server) {
          throw new Error('Test server failed to initialize');
        }
        const address = server.address();
        if (!address || typeof address === 'string') {
          throw new Error('Unexpected server address for test server');
        }
        this.port = address.port;
        resolve(this.port);
      });
    });
  }

  async stop(): Promise<void> {
    return new Promise((resolve) => {
      if (this.server) {
        this.server.close(() => resolve());
      } else {
        resolve();
      }
    });
  }

  getUrl(path: string = '/'): string {
    return `http://localhost:${this.port}${path}`;
  }
}

/**
 * Simulate a real scroll using CDP Input.dispatchMouseEvent with wheel type.
 */
async function simulateRealScroll(page: Page, deltaY: number = 200): Promise<void> {
  const client = await page.context().newCDPSession(page);
  try {
    const viewport = page.viewportSize();
    const x = (viewport?.width || 800) / 2;
    const y = (viewport?.height || 600) / 2;

    await client.send('Input.dispatchMouseEvent', {
      type: 'mouseWheel',
      x,
      y,
      deltaX: 0,
      deltaY,
    });
  } finally {
    await client.detach().catch(() => {});
  }
}

/**
 * Helper to get action type from TimelineEntry.
 */
function getActionType(entry: TimelineEntry): ActionType | undefined {
  return entry.action?.type;
}

async function recordPassiveFidelitySemantics(caseName: string, actionTypes: string[], assertions: string[]): Promise<void> {
  const path = process.env.BAS_PASSIVE_FIDELITY_SEMANTICS_OBSERVATIONS?.trim();
  if (!path) return;
  await mkdir(dirname(path), { recursive: true });
  await appendFile(path, `${JSON.stringify({ case: caseName, observedAt: new Date().toISOString(), actionTypes, assertions })}\n`);
}

function getTelemetryUrl(entry: TimelineEntry): string | undefined {
  const telemetry = entry.telemetry as { url?: unknown } | undefined;
  if (!telemetry) {
    return undefined;
  }
  return typeof telemetry.url === 'string' ? telemetry.url : undefined;
}

function getEntryUrl(entry: TimelineEntry): string | undefined {
  const rawUrl = (entry as { url?: unknown }).url;
  return typeof rawUrl === 'string' ? rawUrl : undefined;
}

describe('Pipeline E2E Tests', () => {
  let browser: Browser;
  let server: PipelineTestServer;

  beforeAll(async () => {
    browser = await chromium.launch({ headless: true });
    server = new PipelineTestServer();
    await server.start();
  });

  afterAll(async () => {
    await browser.close();
    await server.stop();
  });

  it.each(['protocol', 'poll'] as const)('ends readiness when its page closes during %s without further protocol attempts', async (stage) => {
    // No injection: the browser remains unready until the fixture closes it.
    const context = await browser.newContext();
    const page = await context.newPage();
    const createSession = context.newCDPSession.bind(context);
    let firstAttached!: () => void;
    const attached = new Promise<void>((resolve) => { firstAttached = resolve; });
    let firstChecked!: () => void;
    const checked = new Promise<void>((resolve) => { firstChecked = resolve; });
    let attemptsAfterClose = 0;
    jest.spyOn(context, 'newCDPSession').mockImplementation(async (target) => {
      if (page.isClosed()) attemptsAfterClose++;
      const session = await createSession(target);
      const detach = session.detach.bind(session);
      session.detach = async () => { await detach(); firstChecked(); };
      firstAttached();
      return session;
    });
    try {
      const readiness = waitForScriptReady(page, 1000, 500).then(
        () => 'unexpected readiness result',
        (error: Error) => error.message,
      );
      await (stage === 'protocol' ? attached : checked);
      // Let the completed check reach its inter-poll wait before closing.
      if (stage === 'poll') await new Promise<void>((resolve) => setImmediate(resolve));
      await page.close();
      expect(await readiness).toMatch(/page.*closed/i);
      expect(attemptsAfterClose).toBe(0);
    } finally {
      await context.close();
    }
  });

  describe('complete pipeline validation', () => {
    let context: BrowserContext;
    let page: Page;
    let pageIdentities: WeakMap<Page, string>;
    let initializer: RecordingContextInitializer;
    let pipelineManager: RecordingPipelineManager;
    let capturedEntries: TimelineEntry[];

    beforeEach(async () => {
      context = await browser.newContext();
      initializer = createRecordingContextInitializer({});
      await initializer.initialize(context);
      page = await context.newPage();
      pageIdentities = new WeakMap([[page, 'pipeline-main-tab']]);

      pipelineManager = createRecordingPipelineManager(page, context, initializer, {
        sessionId: 'pipeline-e2e-test',
        getDriverPageId: (target) => pageIdentities.get(target),
      });
      await pipelineManager.initialize();
      capturedEntries = [];
    });

    afterEach(async () => {
      if (pipelineManager.isRecording()) {
        await pipelineManager.stopRecording();
      }
      await context.close();
    });

    it.each([false, true])('preserves application console evidence without recorder noise (capturing=%s)', async (capturing) => {
      const consoleEvents: Array<{ type: string; text: string }> = [];
      const observer = await context.newCDPSession(page);
      observer.on('Runtime.consoleAPICalled', (event: { type: string; args: Array<{ value?: unknown }> }) => {
        consoleEvents.push({ type: event.type, text: event.args.map(argument => String(argument.value)).join(' ') });
      });
      try {
        await observer.send('Runtime.enable');
        await page.goto(server.getUrl('/'));
        if (capturing) await pipelineManager.startRecording({
          sessionId: 'pipeline-e2e-test', onEntry: entry => { capturedEntries.push(entry); },
        });
        await page.click('#test-btn');
        if (capturing) await pipelineManager.stopRecording();
        expect(capturedEntries.filter(entry => getActionType(entry) === ActionType.CLICK)).toHaveLength(capturing ? 1 : 0);
        const telemetry = await observer.send('Runtime.evaluate', {
          expression: '({ready:window.__vrooli_recording_ready,detected:window.__vrooli_recording_telemetry.eventsDetected})',
          returnByValue: true,
        });
        expect(telemetry.result.value).toMatchObject({ ready: true });
        expect(telemetry.result.value.detected).toBeGreaterThan(0);
        await observer.send('Runtime.evaluate', { expression: 'console.error("application-error-sentinel")' });
        expect(consoleEvents).toContainEqual({ type: 'error', text: 'application-error-sentinel' });
        expect(consoleEvents.filter(event => event.text !== 'application-error-sentinel')).toEqual([]);
      } finally {
        await observer.detach();
      }
    });

    it('starts native preview without a second DOM wait and keeps it stopped after delayed load completion [REQ:BAS-RH-J22]', async () => {
      await page.goto(server.getUrl('/'));
      await pipelineManager.verifyPipeline({ timeoutMs: 5000 });
      const configuration = jest.spyOn(driverConfig, 'loadConfig').mockReturnValue(createTestConfig());
      const frames = new WebSocketServer({ host: '127.0.0.1', port: 0 });
      await once(frames, 'listening');
      const address = frames.address();
      if (typeof address === 'string') throw new Error('Expected WebSocket port');
      let releaseDom!: () => void;
      let enteredDom!: () => void;
      const dom = new Promise<void>((resolve) => { releaseDom = resolve; });
      const entered = new Promise<void>((resolve) => { enteredDom = resolve; });
      const load = jest.spyOn(page, 'waitForLoadState').mockImplementation(async () => { enteredDom(); await dom; });
      let received!: () => void;
      const receivedFrame = new Promise<void>((resolve) => { received = resolve; });
      frames.on('connection', (socket) => socket.once('message', () => received()));
      const session = { id: 'pipeline-e2e-test', spec: { execution_id: 'owner', workflow_id: 'fixture', reuse_mode: 'fresh', viewport: { width: 800, height: 600 } }, page, pipelineManager, phase: 'ready', ownerExecutionId: 'owner', leaseId: 'lease', pageToIdMap: new WeakMap([[page, 'initial-page']]) };
      const manager = {
        getSession: () => session,
        updateActivity: jest.fn(),
        getSessionForLease: (_id: string, owner: string, lease: string) => {
          if (owner !== session.ownerExecutionId || lease !== session.leaseId) throw new Error('Lease changed');
          return session;
        },
        setSessionPhase: (_id: string, phase: string) => { session.phase = phase; return true; },
      } as unknown as SessionManager;
      const response = createMockHttpResponse();
      let start: Promise<void> | undefined;
      let deadline: ReturnType<typeof setTimeout> | undefined;
      try {
        start = handleRecordStart(createMockHttpRequest({ method: 'POST', body: {
          execution_id: 'owner', lease_id: 'lease',
          frame_callback_url: `http://127.0.0.1:${address.port}/frames`,
        } }), response, session.id, manager, createTestConfig());
        const first = await Promise.race([start.then(() => 'started'), entered.then(() => 'extra-dom-wait')]);
        if (first === 'started') {
          await Promise.race([receivedFrame, new Promise<never>((_, reject) => {
            deadline = setTimeout(() => reject(new Error('Native recording preview exceeded5000ms')), 5000);
          })]);
          clearTimeout(deadline);
        }
        const stopped = createMockHttpResponse();
        await handleRecordStop(createMockHttpRequest({ method: 'POST', body: { execution_id: 'owner', lease_id: 'lease' } }), stopped, session.id, manager);
        releaseDom(); await start;
        expect(stopped.statusCode).toBe(200);
        expect(first).toBe('started');
        expect(response.statusCode).toBe(200);
        expect(load).not.toHaveBeenCalled();
        expect(getFrameStreamSettings(session.id)).toBeNull();
        expect(pipelineManager.isRecording()).toBe(false);
      } finally {
        releaseDom(); if (deadline) clearTimeout(deadline);
        await start;
        await stopFrameStreaming(session.id);
        if (pipelineManager.isRecording()) await pipelineManager.stopRecording();
        // This route used pull delivery. The fixture consumes its own entries
        // before ACK so the next test does not inherit uncommitted observations.
        const entries = getTimelineEntries(session.id);
        capturedEntries.push(...entries);
        acknowledgeTimelineEntries(session.id, entries.map((entry) => entry.id));
        for (const socket of frames.clients) socket.terminate();
        await new Promise<void>((resolve) => frames.close(() => resolve()));
        load.mockRestore(); configuration.mockRestore();
      }
    });

    it('retains a rejected browser event until the same identity is acknowledged', async () => {
      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);
      let recoveredEntryId: string | undefined;
      let resolveRecovered!: (id: string) => void;
      const recovered = new Promise<string>((resolve) => { resolveRecovered = resolve; });
      await pipelineManager.startRecording({
        sessionId: 'pipeline-e2e-test',
        recordingId: 'browser-retry',
        onEntry: (entry) => {
          if (getActionType(entry) === ActionType.CLICK) {
            recoveredEntryId = entry.id;
            resolveRecovered(entry.id);
          }
        },
      });
      const posts: Array<{ id?: string; actionType?: string }> = [];
      let allowAcknowledgement = false;
      await page.route('**/__vrooli_recording_event__', async (route) => {
        const event = route.request().postDataJSON() as { id?: string; actionType?: string };
        if (event.actionType !== 'click') { await route.fulfill({ status: 200, body: JSON.stringify({ ok: true, entry_id: event.id }) }); return; }
        posts.push(event);
        const accepted = allowAcknowledgement && posts.length > 1;
        if (accepted) { await route.fallback(); return; }
        await route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ ok: false, entry_id: event.id }) });
      });
      const failed = page.waitForResponse((r) => r.url().endsWith('/__vrooli_recording_event__') && r.status() === 503);
      await page.click('#test-btn');
      await (await failed).finished();
      expect(posts[0]?.id).toMatch(/^[0-9a-f-]{36}$/i);
      const pending = await page.evaluate(() => JSON.parse(sessionStorage.getItem('__vrooli_pending_events__') || '[]') as Array<{data:{id:string}}>);
      expect(pending.some(item => item.data.id === posts[0]?.id)).toBe(true);
      await page.reload({ waitUntil: 'domcontentloaded' });
      await waitForScriptReady(page, 5000);
      const recoveredPending = await page.evaluate(() => JSON.parse(sessionStorage.getItem('__vrooli_pending_events__') || '[]') as Array<{data:{id:string}}>);
      expect(recoveredPending.some(item => item.data.id === posts[0]?.id)).toBe(true);
      allowAcknowledgement = true;
      let deadline: ReturnType<typeof setTimeout> | undefined;
      try {
        await Promise.race([recovered, new Promise<never>((_, reject) => { deadline = setTimeout(() => reject(new Error('rejected browser event was not recovered after reload')), 6000); })]);
      } finally { if (deadline) clearTimeout(deadline); }
      expect(recoveredEntryId).toBe(posts[0]?.id);
      expect(posts.every((event) => event.id === posts[0]?.id)).toBe(true);
    });

    it('does not acknowledge a browser event when delivery rejects', async () => {
      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);
      const attempts: Array<{id: string; sequence: number}> = [];
      let allowRecovery = false;
      await pipelineManager.startRecording({
        sessionId: 'pipeline-e2e-test', recordingId: 'rejected-delivery',
        onEntry: (entry) => {
          if (getActionType(entry) === ActionType.CLICK) {
            attempts.push({ id: entry.id, sequence: entry.sequenceNum });
            if (!allowRecovery) throw new Error('journal commit rejected');
          }
        },
      });
      const response = page.waitForResponse((r) => r.url().endsWith('/__vrooli_recording_event__') &&
        r.request().postDataJSON()?.actionType === 'click');
      await page.click('#test-btn');
      expect((await response).status()).toBeGreaterThanOrEqual(500);
      expect(attempts.length).toBeGreaterThan(0);
      try {
        await expect(pipelineManager.stopRecording()).rejects.toThrow('Recording delivery returned 500');
        expect(pipelineManager.isRecording()).toBe(true);
      } finally { allowRecovery = true; }
      const stopped = await pipelineManager.stopRecording();
      expect(stopped.actionCount).toBe(2);
      expect(new Set(attempts.map((attempt) => `${attempt.id}/${attempt.sequence}`)).size).toBe(1);
    });

    it('does not report stopped while an admitted delivery is uncommitted', async () => {
      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);
      let admit!: () => void;
      let commit!: () => void;
      const admitted = new Promise<void>((resolve) => { admit = resolve; });
      const committed = new Promise<void>((resolve) => { commit = resolve; });
      await pipelineManager.startRecording({
        sessionId: 'pipeline-e2e-test', recordingId: 'delayed-delivery',
        onEntry: async (entry) => {
          if (getActionType(entry) === ActionType.CLICK) { admit(); await committed; }
        },
      });
      await page.click('#test-btn');
      await admitted;
      let reportedSuccess = false;
      const stop = pipelineManager.stopRecording().then(() => { reportedSuccess = true; });
      try {
        // The controlled commit gate stays closed during this observation.
        await new Promise<void>((resolve) => setTimeout(resolve, 100));
        expect(reportedSuccess).toBe(false);
      } finally {
        commit();
        await stop;
      }
      expect(reportedSuccess).toBe(true);
    });

    it('flushes the final input before its debounce timer when stopped', async () => {
      await page.goto(server.getUrl('/'));
      await pipelineManager.startRecording({
        sessionId: 'pipeline-e2e-test',
        onEntry: (entry) => { capturedEntries.push(entry); },
      });
      await page.fill('#test-input', 'final value');
      const result = await pipelineManager.stopRecording();
      const inputs = capturedEntries.filter((entry) => getActionType(entry) === ActionType.INPUT);
      expect(inputs).toHaveLength(1);
      expect(inputs[0]?.action?.params).toMatchObject({ case: 'input', value: { value: 'final value' } });
      expect(result.actionCount).toBe(capturedEntries.length);
    });

    it('joins an admitted start before stop and leaves browser capture inactive', async () => {
      await page.goto(server.getUrl('/'));
      let admit!: () => void;
      let commit!: () => void;
      const admitted = new Promise<void>((resolve) => { admit = resolve; });
      const committed = new Promise<void>((resolve) => { commit = resolve; });
      const start = pipelineManager.startRecording({
        sessionId: 'pipeline-e2e-test', recordingId: 'overlapping-start-stop',
        onEntry: async (entry) => { capturedEntries.push(entry); admit(); await committed; },
      });
      await Promise.race([admitted, start]);
      const stop = pipelineManager.stopRecording();
      try {
        expect(pipelineManager.getState().phase).toBe('starting');
        await expect(pipelineManager.startRecording({ sessionId: 'pipeline-e2e-test', onEntry: () => {} })).rejects.toThrow('already owned');
      } finally { commit(); }
      await start;
      const stopped = await stop;
      expect(stopped.actionCount).toBe(1);
      expect(pipelineManager.isRecording()).toBe(false);
      await page.click('#test-btn');
      await page.waitForTimeout(100);
      expect(capturedEntries).toHaveLength(1);
      const pending = await page.evaluate(() => JSON.parse(sessionStorage.getItem('__vrooli_pending_events__') || '[]'));
      expect(pending).toEqual([]);
    });

    it('retains the queued navigation when a slow click delivery crosses origins', async () => {
      await page.goto(server.getUrl('/'));
      const target = server.getUrl('/page-2').replace('localhost', '127.0.0.1');
      await page.evaluate((url) => { (document.querySelector('#test-link') as HTMLAnchorElement).href = url; }, target);
      let commit!: () => void;
      const committed = new Promise<void>((resolve) => { commit = resolve; });
      await pipelineManager.startRecording({
        sessionId: 'pipeline-e2e-test', recordingId: 'cross-origin-pending',
        onEntry: async (entry) => {
          if (getActionType(entry) === ActionType.CLICK) await committed;
          capturedEntries.push(entry);
        },
      });
      try {
        await Promise.all([page.waitForURL(target), page.click('#test-link')]);
      } finally { commit(); }
      await pipelineManager.stopRecording();
      expect(capturedEntries.some((entry) => entry.action?.params.case === 'navigate' && entry.action.params.value.url === target)).toBe(true);
    });

    it('captures a dynamically attached frame after recording has started', async () => {
      await page.goto(server.getUrl('/'));
      await pipelineManager.startRecording({
        sessionId: 'pipeline-e2e-test',
        onEntry: (entry) => { capturedEntries.push(entry); },
      });
      const target = server.getUrl('/').replace('localhost', '127.0.0.1');
      await page.evaluate((url) => {
        const frame = document.createElement('iframe'); frame.src = url; frame.id = 'late-frame'; document.body.appendChild(frame);
      }, target);
      await page.frameLocator('#late-frame').locator('#test-btn').click();
      await pipelineManager.stopRecording();
      const clicks = capturedEntries.filter((entry) => getActionType(entry) === ActionType.CLICK);
      expect(clicks).toHaveLength(1);
      expect(clicks[0]?.telemetry?.framePath).toEqual(['#late-frame']);
    });

    it('preserves distinct tab identities for equal selectors in the recording timeline', async () => {
      await page.goto(server.getUrl('/'));
      await pipelineManager.startRecording({ sessionId: 'pipeline-e2e-test', onEntry: (entry) => { capturedEntries.push(entry); } });

      const secondPage = await context.newPage();
      pageIdentities.set(secondPage, 'pipeline-secondary-tab');
      await secondPage.goto(server.getUrl('/'));
      await waitForScriptReady(secondPage, 5000);
      await secondPage.click('#test-btn');
      await page.bringToFront();
      await page.click('#test-btn');
      await pipelineManager.stopRecording();

      const clicks = capturedEntries.filter((entry) => getActionType(entry) === ActionType.CLICK);
      expect(clicks).toHaveLength(2);
      expect(clicks.map((entry) => entry.telemetry?.driverPageId)).toEqual([
        'pipeline-secondary-tab',
        'pipeline-main-tab',
      ]);
    });

    it('[CRITICAL] should capture all core event types in single session', async () => {
      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'pipeline-e2e-test',
        recordingId: 'full-pipeline',
        onEntry: (entry) => {
          capturedEntries.push(entry);
        },
      });

      // 1. Click
      await page.click('#test-btn');
      await page.waitForTimeout(300);

      // 2. Input
      await page.type('#test-input', 'test');
      await page.waitForTimeout(500);

      // 3. Scroll (page has min-height: 200vh)
      await simulateRealScroll(page, 200);
      await page.waitForTimeout(700);

      await pipelineManager.stopRecording();

      // Validate all event types captured
      const clicks = capturedEntries.filter(e => getActionType(e) === ActionType.CLICK);
      const inputs = capturedEntries.filter(e => getActionType(e) === ActionType.INPUT);
      const scrolls = capturedEntries.filter(e => getActionType(e) === ActionType.SCROLL);

      expect(clicks.length).toBeGreaterThan(0);
      expect(inputs.length).toBeGreaterThan(0);
      expect(scrolls.length).toBeGreaterThan(0);

      const fixtureClicks = await page.evaluate(() => (window as any).__fixtureClickCount as number);
      expect(fixtureClicks).toBe(1);
      expect(await page.locator('#test-input').inputValue()).toBe('test');
      expect(await page.evaluate(() => window.scrollY)).toBeGreaterThan(50);

      const clickParams = clicks[0]?.action?.params;
      expect(clickParams?.case).toBe('click');
      if (clickParams?.case === 'click') {
        expect(clickParams.value.selector).toContain('test-button');
        expect(clickParams.value.clickCount).toBe(1);
      }
      const inputParams = inputs.at(-1)?.action?.params;
      expect(inputParams?.case).toBe('input');
      if (inputParams?.case === 'input') {
        expect(inputParams.value.selector).toContain('test-input');
        expect(inputParams.value.value).toBe('test');
      }
      const scrollParams = scrolls.at(-1)?.action?.params;
      expect(scrollParams?.case).toBe('scroll');
      if (scrollParams?.case === 'scroll') {
        expect(scrollParams.value.deltaY).toBeGreaterThan(0);
      }

      // Validate TimelineEntry structure
      const entry = capturedEntries[0];
      if (!entry) {
        throw new Error('Expected at least one captured entry');
      }
      expect(entry.id).toBeDefined();
      expect(entry.sequenceNum).toBeDefined();
      expect(entry.timestamp).toBeDefined();
      expect(entry.action).toBeDefined();
      const sequences = capturedEntries.map((captured) => captured.sequenceNum);
      expect(new Set(sequences).size).toBe(sequences.length);
      expect(sequences.every((sequence, index) => index === 0 || sequence > (sequences[index - 1] ?? -1))).toBe(true);
      await recordPassiveFidelitySemantics(
        'core-events',
        ['click', 'type', 'scroll'],
        ['one independent click effect', 'typed value and input selector retained', 'positive scroll delta retained', 'unique increasing sequence numbers']
      );
    });

    it('[CRITICAL] should have clean state between test runs', async () => {
      // Reset stats to ensure clean state
      initializer.resetStats();

      const statsBefore = initializer.getInjectionStats();
      expect(statsBefore.attempted).toBe(0);
      expect(statsBefore.successful).toBe(0);

      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);

      const statsAfter = initializer.getInjectionStats();

      // Should have exactly 1 injection attempt for this test
      expect(statsAfter.attempted).toBe(1);
      expect(statsAfter.successful).toBe(1);
    });

    it('[CRITICAL] should capture navigation events', async () => {
      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'pipeline-e2e-test',
        recordingId: 'nav-test',
        onEntry: (entry) => {
          capturedEntries.push(entry);
        },
      });

      // Click navigation link
      await page.click('#test-link');
      await page.waitForURL('**/page-2', { timeout: 5000 });
      await page.waitForTimeout(500);

      await pipelineManager.stopRecording();

      // Validate navigation event was captured
      const navEvents = capturedEntries.filter(e => getActionType(e) === ActionType.NAVIGATE);
      const clickEvents = capturedEntries.filter(e => getActionType(e) === ActionType.CLICK);

      // Click that triggered navigation should be captured
      expect(clickEvents.length).toBeGreaterThan(0);

      // Navigation event should be captured
      expect(navEvents.length).toBeGreaterThan(0);

      // Navigation should contain the target URL (check both telemetry.url and top-level url)
      const navToPage2 = navEvents.find((entry) => {
        const telemetryUrl = getTelemetryUrl(entry);
        const entryUrl = getEntryUrl(entry);
        return telemetryUrl?.includes('/page-2') || entryUrl?.includes('/page-2');
      });
      expect(navToPage2).toBeDefined();
      await recordPassiveFidelitySemantics(
        'navigation',
        ['click', 'navigate'],
        ['click triggering navigation retained', 'navigation entry identifies /page-2']
      );
    });

    it('[CRITICAL] should continue capturing events after navigation', async () => {
      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'pipeline-e2e-test',
        recordingId: 'post-nav-test',
        onEntry: (entry) => {
          capturedEntries.push(entry);
        },
      });

      // Click before navigation
      await page.click('#test-btn');
      await page.waitForTimeout(200);

      // Navigate
      await page.click('#test-link');
      await page.waitForURL('**/page-2', { timeout: 5000 });
      await waitForScriptReady(page, 5000);
      await page.waitForTimeout(300);

      const entriesAfterNav = capturedEntries.length;

      // Click on new page
      await page.click('#page-2-btn');
      await page.waitForTimeout(500);

      await pipelineManager.stopRecording();

      // Should have captured events after navigation
      const postNavEntries = capturedEntries.slice(entriesAfterNav);
      const postNavClicks = postNavEntries.filter(e => getActionType(e) === ActionType.CLICK);

      // CRITICAL: Events after navigation must be captured
      expect(postNavClicks.length).toBeGreaterThan(0);
      await recordPassiveFidelitySemantics(
        'capture-after-navigation',
        ['click', 'navigate', 'click'],
        ['pre-navigation click retained', 'navigation completed', 'post-navigation click retained']
      );
    });

    it('should maintain correct event sequence ordering', async () => {
      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);

      await pipelineManager.startRecording({
        sessionId: 'pipeline-e2e-test',
        recordingId: 'sequence-test',
        onEntry: (entry) => {
          capturedEntries.push(entry);
        },
      });

      // Perform multiple actions
      await page.click('#test-btn');
      await page.waitForTimeout(200);
      await page.type('#test-input', 'abc');
      await page.waitForTimeout(500);
      await simulateRealScroll(page, 100);
      await page.waitForTimeout(700);

      await pipelineManager.stopRecording();

      // Verify sequence numbers are ascending
      const sequenceNums = capturedEntries
        .map(e => e.sequenceNum)
        .filter((n): n is number => n !== undefined);

      for (let i = 1; i < sequenceNums.length; i++) {
        const current = sequenceNums[i];
        const previous = sequenceNums[i - 1];
        if (current === undefined || previous === undefined) {
          throw new Error('Missing sequence number when validating ordering');
        }
        expect(current).toBeGreaterThan(previous);
      }

      // Verify timestamps are ascending (or equal for rapid events)
      const timestamps = capturedEntries
        .map(e => e.timestamp)
        .filter((t): t is NonNullable<typeof t> => t !== undefined)
        .map(t => Number(t.seconds) * 1000 + Number(t.nanos) / 1000000);

      for (let i = 1; i < timestamps.length; i++) {
        const current = timestamps[i];
        const previous = timestamps[i - 1];
        if (current === undefined || previous === undefined) {
          throw new Error('Missing timestamp when validating ordering');
        }
        expect(current).toBeGreaterThanOrEqual(previous);
      }
    });

    it('retains 10000 unique native fixture actions across one rejected journal delivery', async () => {
      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);

      const clickEntries: TimelineEntry[] = [];
      const attemptsById = new Map<string, number>();
      let rejectedEntryId: string | undefined;
      let rejectOnce = true;
      let notifyProgress: (() => void) | undefined;
      await pipelineManager.startRecording({
        sessionId: 'pipeline-e2e-test',
        recordingId: 'passive-fidelity-10k',
        onEntry: (entry) => {
          if (getActionType(entry) !== ActionType.CLICK) {
            capturedEntries.push(entry);
            return;
          }

          const attempts = (attemptsById.get(entry.id) ?? 0) + 1;
          attemptsById.set(entry.id, attempts);
          if (rejectOnce && clickEntries.length === 5000) {
            rejectOnce = false;
            rejectedEntryId = entry.id;
            throw new Error('controlled transient journal rejection');
          }

          clickEntries.push(entry);
          capturedEntries.push(entry);
          notifyProgress?.();
        },
      });

      const waitForClicks = (target: number): Promise<void> => {
        if (clickEntries.length >= target) return Promise.resolve();
        return new Promise((resolve, reject) => {
          const timeout = setTimeout(() => {
            notifyProgress = undefined;
            reject(new Error(`Only ${clickEntries.length}/${target} clicks reached the journal`));
          }, 30000);
          notifyProgress = () => {
            if (clickEntries.length < target) return;
            clearTimeout(timeout);
            notifyProgress = undefined;
            resolve();
          };
        });
      };

      const startedAt = performance.now();
      const batchSize = 25;
      const buttonBox = await page.locator('#test-btn').boundingBox();
      if (!buttonBox) throw new Error('Fixture button has no visible bounds');
      for (let target = batchSize; target <= 10000; target += batchSize) {
        const reachedTarget = waitForClicks(target);
        for (let index = 0; index < batchSize; index += 1) {
          await page.mouse.click(buttonBox.x + buttonBox.width / 2, buttonBox.y + buttonBox.height / 2);
        }
        await reachedTarget;
      }
      const elapsedMs = performance.now() - startedAt;
      const independentFixtureClicks = await page.evaluate(() => (window as Window & { __fixtureClickCount: number }).__fixtureClickCount);

      await pipelineManager.stopRecording();

      const ids = clickEntries.map((entry) => entry.id);
      const sequences = clickEntries.map((entry) => entry.sequenceNum);
      const receipt = {
        producer: 'playwright-driver/tests/integration/pipeline-e2e.test.ts',
        scenario: 'native Playwright mouse input captured through the recording event route and ordered journal callback',
        browserVersion: browser.version(),
        platform: process.platform,
        architecture: process.arch,
        actions: clickEntries.length,
        independentFixtureClicks,
        uniqueActionIds: new Set(ids).size,
        strictlyIncreasingSequence: sequences.every((sequence, index) => index === 0 || sequence > (sequences[index - 1] ?? -1)),
        injectedFault: 'one journal callback rejection at action 5001; browser retries the same event identity',
        rejectedActionAttempts: attemptsById.get(rejectedEntryId ?? '') ?? 0,
        dispatchBatchSize: batchSize,
        elapsedMs: Math.round(elapsedMs),
        limitation: 'No driver-process crash or persisted API/database journal is exercised; this is partial passive-fidelity evidence.',
      };
      // eslint-disable-next-line no-console
      console.log(`BAS_PASSIVE_FIDELITY_DIAGNOSTIC ${JSON.stringify(receipt)}`);
      expect(clickEntries).toHaveLength(10000);
      expect(independentFixtureClicks).toBe(10000);
      expect(receipt.uniqueActionIds).toBe(10000);
      expect(rejectedEntryId).toBeDefined();
      expect(attemptsById.get(rejectedEntryId ?? '')).toBe(2);
      expect(receipt.strictlyIncreasingSequence).toBe(true);
      expect(elapsedMs).toBeLessThan(120000);
    }, 120000);
  });

  describe('state management', () => {
    let context: BrowserContext;
    let page: Page;
    let initializer: RecordingContextInitializer;
    let pipelineManager: RecordingPipelineManager;

    beforeEach(async () => {
      context = await browser.newContext();
      initializer = createRecordingContextInitializer({});
      await initializer.initialize(context);
      page = await context.newPage();

      pipelineManager = createRecordingPipelineManager(page, context, initializer, {
        sessionId: 'state-test',
      });
      await pipelineManager.initialize();
    });

    afterEach(async () => {
      if (pipelineManager.isRecording()) {
        await pipelineManager.stopRecording();
      }
      await context.close();
    });

    it('should reset injection stats correctly', async () => {
      // First navigation to trigger injection
      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);

      const statsAfterFirst = initializer.getInjectionStats();
      expect(statsAfterFirst.attempted).toBeGreaterThan(0);
      expect(statsAfterFirst.successful).toBeGreaterThan(0);

      // Reset stats
      initializer.resetStats();

      const statsAfterReset = initializer.getInjectionStats();
      expect(statsAfterReset.attempted).toBe(0);
      expect(statsAfterReset.successful).toBe(0);
      expect(statsAfterReset.failed).toBe(0);
      expect(statsAfterReset.avgInjectionTimeMs).toBe(0);
      expect(statsAfterReset.lastInjectionAt).toBeNull();
    });

    it('should reset route handler stats correctly', async () => {
      await page.goto(server.getUrl('/'));
      await waitForScriptReady(page, 5000);

      // Start recording and generate some events
      const capturedEntries: TimelineEntry[] = [];
      await pipelineManager.startRecording({
        sessionId: 'state-test',
        recordingId: 'route-stats-test',
        onEntry: (entry) => {
          capturedEntries.push(entry);
        },
      });

      await page.click('#test-btn');
      await page.waitForTimeout(300);

      await pipelineManager.stopRecording();

      const routeStatsAfterEvents = initializer.getRouteHandlerStats();
      expect(routeStatsAfterEvents.eventsReceived).toBeGreaterThan(0);

      // Reset stats
      initializer.resetStats();

      const routeStatsAfterReset = initializer.getRouteHandlerStats();
      expect(routeStatsAfterReset.eventsReceived).toBe(0);
      expect(routeStatsAfterReset.eventsProcessed).toBe(0);
      expect(routeStatsAfterReset.eventsDroppedNoHandler).toBe(0);
      expect(routeStatsAfterReset.eventsWithErrors).toBe(0);
    });
  });
});

// [REQ:BAS-RH-J22] The external transport is an independent disposal oracle.
describe('session-owned frame transport', () => {
  let configuration: jest.SpiedFunction<typeof driverConfig.loadConfig>;
  beforeEach(() => { configuration = jest.spyOn(driverConfig, 'loadConfig').mockReturnValue(createTestConfig()); });
  afterEach(() => { configuration.mockRestore(); });
  it.each(['close', 'reset'] as const)('disposes its preview on session %s', async (operation) => {
    const sockets = new WebSocketServer({ host: '127.0.0.1', port: 0 });
    await once(sockets, 'listening');
    const address = sockets.address();
    if (typeof address === 'string') throw new Error('Expected local socket port');
    const manager = new SessionManager(createTestConfig());
    const fixture = new PipelineTestServer();
    const port = await fixture.start();
    let sessionId: string | undefined;
    let timer: ReturnType<typeof setTimeout> | undefined;
    try {
      const session = await manager.startSession({
        execution_id: `frame-${operation}`, workflow_id: 'frame-owner', base_url: `http://127.0.0.1:${port}/`,
        viewport: { width: 640, height: 480 }, reuse_mode: 'fresh', required_capabilities: {},
      });
      sessionId = session.sessionId;
      const connected = once(sockets, 'connection');
      startFrameStreaming(sessionId, manager, { callbackUrl: `http://127.0.0.1:${address.port}/frames` });
      const [socket] = await connected;
      const painted = once(socket, 'message');
      await manager.peekSession(sessionId).page.evaluate(() => { document.body.style.background = 'tomato'; });
      await painted;
      const closed = once(socket, 'close').then(() => true);
      if (operation === 'close') await manager.closeSession(sessionId);
      else await manager.resetSession(sessionId);
      const observed = await Promise.race([
        closed,
        new Promise<boolean>((resolve) => { timer = setTimeout(() => resolve(false), 1000); }),
      ]);
      expect(observed).toBe(true);
      expect(getFrameStreamSettings(sessionId)).toBeNull();
    } finally {
      if (timer) clearTimeout(timer);
      if (sessionId) await stopFrameStreaming(sessionId);
      await manager.shutdown();
      await fixture.stop();
      for (const socket of sockets.clients) socket.terminate();
      await new Promise<void>((resolve) => sockets.close(() => resolve()));
    }
  });
});

// Native pixels are decoded in an independent page, not classified by the stream.
describe('native capture page identity', () => {
  it('reconnects with blue-tab pixels after buffering an old red tab', async () => {
    const browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
    let capture: StreamingHandle | undefined;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const bounded = async (event: Promise<void>): Promise<void> => {
      try {
        await Promise.race([event, new Promise<never>((_, reject) => {
          timer = setTimeout(() => reject(new Error('Native frame observation exceeded 1000ms')), 1000);
        })]);
      } finally { if (timer) clearTimeout(timer); }
    };
    try {
      const context = await browser.newContext({ viewport: { width: 640, height: 480 } });
      const red = await context.newPage();
      const blue = await context.newPage();
      const decoder = await browser.newPage();
      await red.goto('data:text/html,' + encodeURIComponent('<body style="margin:0;background:rgb(255,0,0);height:100vh"></body>'));
      await blue.goto('data:text/html,' + encodeURIComponent('<body style="margin:0;background:rgb(0,0,255);height:100vh"></body>'));
      let redSeen!: () => void;
      let blueSeen!: () => void;
      let frameSent!: () => void;
      const redFrame = new Promise<void>((resolve) => { redSeen = resolve; });
      const blueFrame = new Promise<void>((resolve) => { blueSeen = resolve; });
      const delivered = new Promise<void>((resolve) => { frameSent = resolve; });
      const acquire = context.newCDPSession.bind(context);
      jest.spyOn(context, 'newCDPSession').mockImplementation(async (target) => {
        const protocol = await acquire(target);
        protocol.on('Page.screencastFrame', () => {
          if (target === red) redSeen();
          if (target === blue) blueSeen();
        });
        return protocol;
      });
      let current = red;
      let ready = false;
      const frames: Buffer[] = [];
      capture = await new CdpScreencastStrategy().start(
        () => current,
        { sessionId: 'native-page-identity', sourceForPage: page => ({session_id:'native-page-identity',execution_id:'native-owner',lease_id:'native-lease',page_id:page===red?'red':'blue'}), quality: 65, targetFps: 30, scale: 'css', includePerfHeaders: false, cdp: { pageCheckIntervalMs: 25 } },
        { isReady: () => ready, getWebSocket: () => ({ readyState: 1, send: (bytes: Buffer) => { frames.push(Buffer.from(bytes.subarray(4 + bytes.readUInt32BE(0)))); frameSent(); } }) },
        { onFrameSent: () => {}, onFrameSkipped: () => {} },
      );
      await bounded(redFrame);
      current = blue;
      await bounded(blueFrame);
      ready = true;
      await bounded(delivered);
      await capture.stop();
      expect(frames.length).toBeGreaterThan(0);
      for (const frame of frames) {
        const rgb = await decoder.evaluate(async (encoded) => {
          const bytes = Uint8Array.from(atob(encoded), (character) => character.charCodeAt(0));
          const bitmap = await createImageBitmap(new Blob([bytes], { type: 'image/jpeg' }));
          const canvas = document.createElement('canvas');
          canvas.width = bitmap.width; canvas.height = bitmap.height;
          const context = canvas.getContext('2d')!;
          context.drawImage(bitmap, 0, 0);
          const color = Array.from(context.getImageData(canvas.width / 2, canvas.height / 2, 1, 1).data);
          bitmap.close();
          return color;
        }, frame.toString('base64'));
        expect(rgb[0]).toBeLessThan(40);
        expect(rgb[1]).toBeLessThan(40);
        expect(rgb[2]).toBeGreaterThan(210);
      }
    } finally {
      if (timer) clearTimeout(timer);
      await capture?.stop();
      await browser.close();
    }
  });
});


describe('native polling fallback [REQ:BAS-RH-J22]', () => {
  it.each(['css', 'device'] as const)('preserves blue pixels and %s dimensions without a public CDP session', async (scale) => {
    const browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
    let capture: StreamingHandle | undefined;
    let deadline: ReturnType<typeof setTimeout> | undefined;
    try {
      const context = await browser.newContext({ viewport: { width: 320, height: 240 }, deviceScaleFactor: 2 });
      const page = await context.newPage();
      await page.goto('data:text/html,' + encodeURIComponent('<body style="margin:0;background:rgb(0,0,255);height:100vh"></body>'));
      jest.spyOn(context, 'newCDPSession').mockRejectedValue(new Error('Public CDP unavailable'));
      let sent!: (frame: Buffer) => void;
      const delivered = new Promise<Buffer>((resolve) => { sent = resolve; });
      const socket = { readyState: 1, send: (frame: Buffer) => sent(Buffer.from(frame)) };
      capture = await new PollingStrategy().start(() => page,
        { sessionId: `native-polling-${scale}`, sourceForPage: () => ({session_id:`native-polling-${scale}`,execution_id:'native-owner',lease_id:'native-lease',page_id:'blue'}), quality: 65, targetFps: 10, scale, includePerfHeaders: false },
        { isReady: () => true, getWebSocket: () => socket },
        { onFrameSent: () => {}, onFrameSkipped: () => {} });
      const frame = await Promise.race([delivered, new Promise<never>((_, reject) => {
        deadline = setTimeout(() => reject(new Error('No native fallback frame within 2000ms')), 2000);
      })]);
      clearTimeout(deadline);
      await capture.stop();
      const observed = await page.evaluate(async (encoded) => {
        const bytes = Uint8Array.from(atob(encoded), (character) => character.charCodeAt(0));
        const bitmap = await createImageBitmap(new Blob([bytes], { type: 'image/jpeg' }));
        const canvas = document.createElement('canvas');
        canvas.width = bitmap.width; canvas.height = bitmap.height;
        const context = canvas.getContext('2d')!;
        context.drawImage(bitmap, 0, 0);
        const rgb = Array.from(context.getImageData(canvas.width / 2, canvas.height / 2, 1, 1).data);
        bitmap.close();
        return { width: canvas.width, height: canvas.height, rgb };
      }, frame.subarray(4 + frame.readUInt32BE(0)).toString('base64'));
      expect(observed.width).toBe(scale === 'css' ? 320 : 640);
      expect(observed.height).toBe(scale === 'css' ? 240 : 480);
      expect(observed.rgb[0]).toBeLessThan(40);
      expect(observed.rgb[1]).toBeLessThan(40);
      expect(observed.rgb[2]).toBeGreaterThan(210);
    } finally {
      if (deadline) clearTimeout(deadline);
      await capture?.stop();
      await browser.close();
    }
  });
});


describe('native recording tab ownership [REQ:BAS-RH-J03]', () => {
  it('uses one tab identity across route and callback and releases native page listeners', async () => {
    const browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
    let cleanup: (() => void) | undefined;
    let deadline: ReturnType<typeof setTimeout> | undefined;
    let deliver!: (event: { driverPageId: string }) => void;
    const created = new Promise<{ driverPageId: string }>((resolve) => { deliver = resolve; });
    const fetchMock = jest.spyOn(global, 'fetch').mockImplementation(async (_url, init) => {
      const event = JSON.parse(init!.body as string);
      if (event.eventType === 'created') deliver(event);
      return { ok: true, status: 200, statusText: 'OK' } as Response;
    });
    try {
      const context = await browser.newContext();
      const counts = new Map<Page, { navigation: number; close: number }>();
      context.on('page', (page) => counts.set(page, {
        navigation: page.listenerCount('framenavigated'), close: page.listenerCount('close'),
      }));
      const initial = await context.newPage();
      const session = {
        id: 'native-tab-owner', phase: 'ready', ownerExecutionId: 'owner', leaseId: 'lease', context, page: initial, pages: [initial],
        pageIdMap: new Map([['initial-id', initial]]),
        pageToIdMap: new WeakMap([[initial, 'initial-id']]),
        currentPageIndex: 0, frameStack: [],
      } as unknown as ReturnType<SessionManager['getSession']>;
      const config = createTestConfig({ history: { callbackUrl: '', thumbnailEnabled: false } });
      const pages = setupPageLifecycleListeners('native-tab-owner', session, 'http://fixture.invalid/callback', config);
      cleanup = pages.cleanup;
      await pages.ready;
      const response = createMockHttpResponse();
      await handleRecordNewPage(createMockHttpRequest({ method: 'POST', body: {
        execution_id: 'owner', lease_id: 'lease',
        url: 'data:text/html,<title>Independent second tab</title><body>Tab two</body>',
      } }), response, 'native-tab-owner', { getSession: () => session, peekSession: () => session, getSessionForLease: SessionManager.prototype.getSessionForLease, updateActivity: jest.fn() } as unknown as SessionManager, config);
      expect(response.statusCode).toBe(201);
      const event = await Promise.race([created, new Promise<never>((_, reject) => {
        deadline = setTimeout(() => reject(new Error('Native created callback exceeded2000ms')), 2000);
      })]);
      clearTimeout(deadline);
      expect(response.getJSON().driver_page_id).toBe(event.driverPageId);
      expect(session.pages).toHaveLength(2);
      expect(session.pageIdMap.size).toBe(2);
      expect(session.pageToIdMap.get(session.page)).toBe(event.driverPageId);
      expect(session.currentPageIndex).toBe(1);
      cleanup();
      for (const [page, before] of counts) {
        expect(page.listenerCount('framenavigated')).toBe(before.navigation);
        expect(page.listenerCount('close')).toBe(before.close);
      }
    } finally {
      if (deadline) clearTimeout(deadline);
      cleanup?.();
      await browser.close();
      fetchMock.mockRestore();
    }
  });
});


describe('native stream controls [REQ:BAS-RH-J23]', () => {
  it.each([
    { dpr: 1, scale: 'css' as const }, { dpr: 1, scale: 'device' as const },
    { dpr: 2, scale: 'css' as const }, { dpr: 2, scale: 'device' as const },
  ].flatMap(value => ['shell', 'regular'].map(mode => ({ ...value, mode }))))('delivers requested pixel dimensions at DPR$dpr scale=$scale mode=$mode', async ({ dpr, scale, mode }) => {
    const configuration = jest.spyOn(driverConfig, 'loadConfig').mockReturnValue(createTestConfig({
      frameStreaming: { useScreencast: true, fallbackToPolling: false },
      performance: { enabled: false, includeTimingHeaders: false },
    }));
    const browser = await chromium.launch({ headless: mode === 'shell', args: ['--no-sandbox', ...(mode === 'regular' ? ['--headless=new'] : [])] });
    const server = new WebSocketServer({ host: '127.0.0.1', port: 0 });
    const sessionId = `native-scale-${dpr}-${scale}`;
    let deadline: ReturnType<typeof setTimeout> | undefined;
    try {
      await once(server, 'listening');
      const address = server.address();
      if (typeof address === 'string') throw new Error('Expected socket port');
      const context = await browser.newContext({ viewport: { width: 320, height: 240 }, deviceScaleFactor: dpr });
      const page = await context.newPage();
      await page.goto('data:text/html,' + encodeURIComponent('<body style="margin:0;height:100vh;background:blue"></body>'));
      const received = new Promise<Buffer>((resolve) => {
        server.once('connection', (socket) => socket.once('message', (bytes: Buffer) => resolve(Buffer.from(bytes))));
      });
      startFrameStreaming(sessionId, { getSession: () => ({ id:sessionId,ownerExecutionId:'native-owner',leaseId:'native-lease',page,pageToIdMap:new WeakMap([[page,'native-page']]) }) }, {
        callbackUrl: `http://127.0.0.1:${address.port}/frames`, scale, quality: 65, fps: 30,
      });
      const packet = await Promise.race([received, new Promise<never>((_, reject) => {
        deadline = setTimeout(() => reject(new Error('Native scale frame exceeded5000ms')), 5000);
      })]);
      clearTimeout(deadline);
      await stopFrameStreaming(sessionId);
      // Both strategies preserve source identity before the JPEG bytes.
      const jpeg = packet.subarray(4 + packet.readUInt32BE(0));
      const decoder = await browser.newPage();
      const pixels = await decoder.evaluate(async (encoded) => {
        const bytes = Uint8Array.from(atob(encoded), (character) => character.charCodeAt(0));
        const image = await createImageBitmap(new Blob([bytes], { type: 'image/jpeg' }));
        const canvas = document.createElement('canvas');
        canvas.width = image.width; canvas.height = image.height;
        const drawing = canvas.getContext('2d')!;
        drawing.drawImage(image, 0, 0);
        const result = { width: image.width, height: image.height, blue: drawing.getImageData(10, 10, 1, 1).data[2] };
        image.close();
        return result;
      }, jpeg.toString('base64'));
      const factor = scale === 'device' ? dpr : 1;
      expect(pixels.width).toBe(320 * factor);
      expect(pixels.height).toBe(240 * factor);
      expect(pixels.blue).toBeGreaterThan(240);
      expect(getFrameStreamSettings(sessionId)).toBeNull();
    } finally {
      if (deadline) clearTimeout(deadline);
      await stopFrameStreaming(sessionId);
      await browser.close();
      for (const socket of server.clients) socket.terminate();
      await new Promise<void>((resolve) => server.close(() => resolve()));
      configuration.mockRestore();
    }
  });

  it.each([true, false])('applies quality, FPS and headers through the settings route (CDP=%s)', async (useScreencast) => {
    const config = createTestConfig({
      frameStreaming: { useScreencast },
      performance: { enabled: false, includeTimingHeaders: false },
    });
    const configuration = jest.spyOn(driverConfig, 'loadConfig').mockReturnValue(config);
    const browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
    const server = new WebSocketServer({ host: '127.0.0.1', port: 0 });
    const sessionId = `native-controls-${useScreencast}`;
    let deadline: ReturnType<typeof setTimeout> | undefined;
    try {
      await once(server, 'listening');
      const address = server.address();
      if (typeof address === 'string') throw new Error('Expected socket port');
      const context = await browser.newContext({ viewport: { width: 640, height: 480 } });
      const page = await context.newPage();
      await page.goto('data:text/html,' + encodeURIComponent('<body style="margin:0;height:100vh;background:blue"><h1>Native quality control</h1></body>'));
      const qualityCalls: number[] = [];
      const acquire = context.newCDPSession.bind(context);
      jest.spyOn(context, 'newCDPSession').mockImplementation(async (target) => {
        const protocol = await acquire(target);
        const send = protocol.send.bind(protocol);
        protocol.send = ((method: string, params: Record<string, unknown>) => {
          if (method === 'Page.startScreencast') qualityCalls.push(params.quality as number);
          return send(method as never, params as never);
        }) as typeof protocol.send;
        return protocol;
      });
      const screenshot = page.screenshot.bind(page);
      jest.spyOn(page, 'screenshot').mockImplementation(async (options) => {
        qualityCalls.push(options!.quality!);
        return screenshot(options);
      });
      const provider = { getSession: () => ({ id:sessionId,ownerExecutionId:'native-owner',leaseId:'native-lease',page,pageToIdMap:new WeakMap([[page,'native-page']]) }) } as unknown as SessionManager;
      const observations: { at: number; jpeg: Buffer; header: { frame_bytes: number } }[] = [];
      let observed!: () => void;
      const threeFrames = new Promise<void>((resolve) => { observed = resolve; });
      const connected = once(server, 'connection');
      startFrameStreaming(sessionId, provider, { callbackUrl: `http://127.0.0.1:${address.port}/frames`, quality: 65, fps: 30 });
      const [socket] = await connected;
      socket.on('message', (data: Buffer) => {
        if (data.length < 5) return;
        const length = data.readUInt32BE(0);
        if (length > data.length - 4 || data[4] !== 123) return;
        let envelope: { timing?: { frame_bytes: number } };
        try { envelope = JSON.parse(data.subarray(4, 4 + length).toString()); }
        catch { return; }
        if (!envelope.timing) return;
        observations.push({ at: performance.now(), jpeg: Buffer.from(data.subarray(4 + length)), header: envelope.timing });
        if (observations.length === 3) observed();
      });
      // Connection precedes capture readiness. First native frame is the oracle
      // that the strategy handle exists before submitting the settings request.
      const first = once(socket, 'message');
      await page.evaluate(() => {
        let frame = 0;
        const paint = () => {
          document.body.style.background = `rgb(${(frame++ * 23) % 256},20,180)`;
          requestAnimationFrame(paint);
        };
        requestAnimationFrame(paint);
      });
      await first;
      const response = createMockHttpResponse();
      await handleStreamSettings(createMockHttpRequest({ method: 'POST', body: { quality: 20, fps: 5, perfMode: true } }),
        response, sessionId, provider, config);
      expect(response.statusCode).toBe(200);
      expect(response.getJSON()).toMatchObject({ quality: 20, fps: 5, perf_mode: true, updated: true, is_streaming: true });
      await Promise.race([threeFrames, new Promise<never>((_, reject) => {
        deadline = setTimeout(() => reject(new Error('Native updated frames exceeded5000ms')), 5000);
      })]);
      clearTimeout(deadline);
      expect(qualityCalls).toContain(20);
      expect(observations[2].at - observations[0].at).toBeGreaterThanOrEqual(350);
      for (const observation of observations) {
        expect(observation.jpeg[0]).toBe(0xff); expect(observation.jpeg[1]).toBe(0xd8);
        expect(observation.header.frame_bytes).toBe(observation.jpeg.length);
      }
      expect(Number.isFinite(response.getJSON().current_fps)).toBe(true);
    } finally {
      if (deadline) clearTimeout(deadline);
      await stopFrameStreaming(sessionId);
      await browser.close();
      for (const socket of server.clients) socket.terminate();
      await new Promise<void>((resolve) => server.close(() => resolve()));
      configuration.mockRestore();
    }
  });
});
