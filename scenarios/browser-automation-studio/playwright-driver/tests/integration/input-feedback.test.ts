import { createHash } from 'node:crypto';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, resolve, sep } from 'node:path';
import { chromium, type Browser, type Page } from 'rebrowser-playwright';
import { once } from 'node:events';
import WebSocket = require('ws');
import { handleRecordInput } from '../../src/routes/record-mode/recording-input';
import { startFrameStreaming, stopFrameStreaming } from '../../src/frame-streaming';
import * as driverConfig from '../../src/config';
import type { SessionManager } from '../../src/session';
import { createMockHttpRequest, createMockHttpResponse, createTestConfig } from '../helpers';

type ClockSample = { nodeMs: number; offsetMs: number; uncertaintyMs: number };
type FeedbackProbeEvent = {
  kind: 'recording_input' | 'recording_input_applied';
  inputId: string;
  atMs: number;
  appliedSequence?: number;
};
type FeedbackProbeWindow = Window & {
  __basPointerEventTimes?: number[];
  __basInputEvents?: FeedbackProbeEvent[];
};
type FrameSocket = {
  on: (event: 'message', listener: (message: Buffer) => void) => void;
  terminate: () => void;
};

function isFrameSocket(value: unknown): value is FrameSocket {
  if (typeof value !== 'object' || value === null) return false;
  const candidate = value as { on?: unknown; terminate?: unknown };
  return typeof candidate.on === 'function' && typeof candidate.terminate === 'function';
}

async function calibrateClock(page: Page): Promise<ClockSample> {
  let best: ClockSample | undefined;
  for (let attempt = 0; attempt < 8; attempt++) {
    const before = performance.now();
    const browserMs = await page.evaluate(() => performance.now());
    const after = performance.now();
    const uncertaintyMs = (after - before) / 2;
    if (!best || uncertaintyMs < best.uncertaintyMs) {
      best = { nodeMs: (before + after) / 2, offsetMs: browserMs - (before + after) / 2, uncertaintyMs };
    }
  }
  if (!best) throw new Error('Could not calibrate the driver/browser monotonic clocks');
  return best;
}

function percentile(values: number[], quantile: number): number {
  const sorted = [...values].sort((a, b) => a - b);
  return sorted[Math.ceil(quantile * sorted.length) - 1] ?? 0;
}

describe('interactive input feedback diagnostic (real Chromium)', () => {
  let browser: Browser;

  beforeAll(async () => {
    browser = await chromium.launch({ headless: true });
  });

  afterAll(async () => {
    await browser?.close();
  });

  it('correlates 1000 driver inputs with the fixture’s affected paint marker', async () => {
    const context = await browser.newContext({ viewport: { width: 800, height: 600 } });
    const page = await context.newPage();
    const session = {
      phase: 'ready', ownerExecutionId: 'feedback-owner', leaseId: 'feedback-lease', page,
      pageToIdMap: new WeakMap([[page, 'feedback-page']]),
    } as ReturnType<SessionManager['getSession']>;
    const sessionManager = {
      getSessionForLease: () => session,
      updateActivity: jest.fn(),
    } as unknown as SessionManager;
    const config = createTestConfig();
    const count = 1000;

    try {
      const fixture = '<!doctype html><html><body><div id="affected" style="width:30px;height:30px;background:rgb(1,2,3)"></div></body></html>';
      await page.goto(`data:text/html,${encodeURIComponent(fixture)}`, { waitUntil: 'domcontentloaded', timeout: 10000 });
      await page.evaluate(() => {
        type FeedbackFixture = Window & {
          __nextInputId?: string;
          __paintPromises?: Record<string, Promise<number>>;
          __paintWaiters?: Record<string, (paintedAt: number) => void>;
          __paintIds?: string[];
        };
        const fixture = window as FeedbackFixture;
        fixture.__paintPromises = {};
        fixture.__paintWaiters = {};
        fixture.__paintIds = [];
        window.addEventListener('pointermove', (event) => {
          const inputId = fixture.__nextInputId;
          if (!inputId) return;
          const target = document.getElementById('affected');
          if (!target) return;
          target.dataset.inputId = inputId;
          target.style.transform = `translateX(${Math.round(event.clientX)}px)`;
          requestAnimationFrame(() => requestAnimationFrame(() => {
            // The second animation frame observes the DOM update after one
            // intervening paint opportunity, using this browser's monotonic clock.
            const paintedAt = performance.now();
            fixture.__paintIds?.push(inputId);
            fixture.__paintWaiters?.[inputId]?.(paintedAt);
            delete fixture.__paintWaiters?.[inputId];
          }));
        });
      });

      const clockStart = await calibrateClock(page);
      const inputPaintPairs: Array<{ inputId: string; nodeStartedAt: number; browserPaintedAt: number }> = [];
      const receiptSequences: number[] = [];
      let maxClockUncertaintyMs = clockStart.uncertaintyMs;

      for (let index = 0; index < count; index++) {
        const inputId = `feedback-${index}`;
        await page.evaluate((id) => {
          type FeedbackFixture = Window & {
            __nextInputId?: string;
            __paintPromises?: Record<string, Promise<number>>;
            __paintWaiters?: Record<string, (paintedAt: number) => void>;
          };
          const fixture = window as FeedbackFixture;
          fixture.__nextInputId = id;
          const promises = fixture.__paintPromises ?? (fixture.__paintPromises = {});
          const waiters = fixture.__paintWaiters ?? (fixture.__paintWaiters = {});
          promises[id] = new Promise((resolve) => { waiters[id] = resolve; });
        }, inputId);

        const nodeStartedAt = performance.now();
        const response = createMockHttpResponse();
        await handleRecordInput(createMockHttpRequest({ method: 'POST', body: {
          execution_id: 'feedback-owner', lease_id: 'feedback-lease', input_id: inputId,
          type: 'pointer', action: 'move', x: 20 + (index % 700), y: 20 + Math.floor(index / 700) * 30,
        } }), response, 'feedback-session', sessionManager, config);
        expect(response.statusCode).toBe(200);
        const receipt = JSON.parse(response.getBody()) as { input_id: string; applied_sequence: number };
        expect(receipt.input_id).toBe(inputId);
        receiptSequences.push(receipt.applied_sequence);

        const paintedAt = await page.evaluate((id) => {
          type FeedbackFixture = Window & { __paintPromises?: Record<string, Promise<number>> };
          return (window as FeedbackFixture).__paintPromises?.[id];
        }, inputId);
        if (typeof paintedAt !== 'number' || !Number.isFinite(paintedAt)) {
          throw new Error(`Input ${index} produced no browser paint timestamp`);
        }
        inputPaintPairs.push({ inputId, nodeStartedAt, browserPaintedAt: paintedAt });
      }

      const clockEnd = await calibrateClock(page);
      maxClockUncertaintyMs = Math.max(maxClockUncertaintyMs, clockEnd.uncertaintyMs);
      const clockOffsetAt = (nodeMs: number): number => {
        const fraction = (nodeMs - clockStart.nodeMs) / Math.max(1, clockEnd.nodeMs - clockStart.nodeMs);
        return clockStart.offsetMs + (clockEnd.offsetMs - clockStart.offsetMs) * fraction;
      };
      const samples = inputPaintPairs.map(({ nodeStartedAt, browserPaintedAt }) =>
        browserPaintedAt - (nodeStartedAt + clockOffsetAt(nodeStartedAt)));
      if (samples.some((elapsed) => !Number.isFinite(elapsed) || elapsed < 0)) {
        throw new Error('Clock-calibrated input-to-paint samples must be finite and nonnegative');
      }
      const paints = await page.evaluate(() => {
        type FeedbackFixture = Window & { __paintIds?: string[] };
        return (window as FeedbackFixture).__paintIds ?? [];
      });
      const report = {
        producer: 'playwright-driver/tests/integration/input-feedback.test.ts',
        scope: 'local driver route to fixture DOM update and two-animation-frame paint observation; excludes API, UI frame decode/draw, and network transit',
        runtimeVersion: process.version,
        osPlatform: process.platform,
        architecture: process.arch,
        browserVersion: browser.version(),
        viewport: { width: 800, height: 600, deviceScaleFactor: 1 },
        sampleCount: samples.length,
        correlatedPaintCount: paints.length,
        correlationComplete: inputPaintPairs.every(({ inputId }, index) => inputId === paints[index]),
        receiptSequencesMonotonic: receiptSequences.slice(0, -1).every((sequence, index) => receiptSequences[index + 1] > sequence),
        p50Ms: percentile(samples, 0.50),
        p95Ms: percentile(samples, 0.95),
        p99Ms: percentile(samples, 0.99),
        maxClockUncertaintyMs,
        minMs: Math.min(...samples),
        maxMs: Math.max(...samples),
        samplesMs: samples,
      };
      // Structured output lets the controlled cohort be retained without a broad suite.
      // The local fixture is diagnostic evidence only, not a full contract-row receipt.
      // eslint-disable-next-line no-console
      console.log(`BAS_INTERACTIVE_FEEDBACK_DIAGNOSTIC ${JSON.stringify(report)}`);
      expect(samples).toHaveLength(count);
      expect(paints).toEqual(Array.from({ length: count }, (_, index) => `feedback-${index}`));
      expect(report.receiptSequencesMonotonic).toBe(true);
      expect(report.correlationComplete).toBe(true);
      expect(maxClockUncertaintyMs).toBeLessThan(10);
    } finally {
      await context.close();
    }
  }, 120000);

  it('correlates input receipts with pixels in native streamed frames', async () => {
    const configuration = jest.spyOn(driverConfig, 'loadConfig').mockReturnValue(createTestConfig({
      frameStreaming: { useScreencast: true, fallbackToPolling: false },
      performance: { enabled: false, includeTimingHeaders: false },
    }));
    const context = await browser.newContext({ viewport: { width: 800, height: 600 } });
    const page = await context.newPage();
    const decoder = await context.newPage();
    const sessionId = 'feedback-frame-session';
    const session = {
      id: sessionId, phase: 'ready', ownerExecutionId: 'feedback-owner', leaseId: 'feedback-lease',
      leaseReleasedAt: null, page, pageToIdMap: new WeakMap([[page, 'feedback-page']]),
    } as ReturnType<SessionManager['getSession']>;
    const sessionManager = {
      getSessionForLease: () => session,
      updateActivity: jest.fn(),
    } as unknown as SessionManager;
    const server = new WebSocket.Server({ host: '127.0.0.1', port: 0 });
    const frameBuffers: Buffer[] = [];
    const frameWaiters: Array<(frame: Buffer) => void> = [];
    let socket: FrameSocket | undefined;
    const count = 1000;

    try {
      await once(server, 'listening');
      const address = server.address();
      if (typeof address === 'string') throw new Error('Expected an ephemeral WebSocket port');
      await page.goto(`data:text/html,${encodeURIComponent('<!doctype html><html><body style="margin:0;background:#777"><div id="marker" style="position:fixed;left:30px;top:30px;display:flex;gap:4px"></div></body></html>')}`, {
        waitUntil: 'domcontentloaded', timeout: 10000,
      });
      await page.evaluate(() => {
        type MarkerFixture = Window & { __nextInputId?: string };
        const marker = document.getElementById('marker');
        if (!marker) throw new Error('Frame marker element is missing');
        const cells = Array.from({ length: 10 }, () => {
          const cell = document.createElement('div');
          cell.style.cssText = 'width:24px;height:24px;background:white';
          marker.appendChild(cell);
          return cell;
        });
        (window as MarkerFixture).__nextInputId = '';
        window.addEventListener('pointermove', () => {
          const id = (window as MarkerFixture).__nextInputId ?? '';
          if (!id) return;
          const value = Number(id.slice('feedback-frame-'.length)) + 1;
          cells.forEach((cell, index) => {
            const bit = (value >> (9 - index)) & 1;
            cell.style.backgroundColor = bit ? 'white' : 'black';
          });
        });
      });
      const connected = once(server, 'connection');
      startFrameStreaming(sessionId, { getSession: () => session }, {
        callbackUrl: `http://127.0.0.1:${address.port}/frames`, quality: 65, fps: 30,
      });
      const connectionEvent: unknown = await connected;
      if (!Array.isArray(connectionEvent) || !isFrameSocket(connectionEvent[0])) {
        throw new Error('Frame stream did not establish a usable WebSocket');
      }
      socket = connectionEvent[0];
      socket.on('message', (message) => {
        const frame = Buffer.from(message);
        const waiter = frameWaiters.shift();
        if (waiter) waiter(frame);
        else frameBuffers.push(frame);
      });
      const nextFrame = (): Promise<Buffer> => {
        const buffered = frameBuffers.shift();
        return buffered ? Promise.resolve(buffered) : new Promise((resolve) => frameWaiters.push(resolve));
      };

      const samples: number[] = [];
      const appliedSequences: number[] = [];
      for (let index = 0; index < count; index++) {
        const inputId = `feedback-frame-${index}`;
        await page.evaluate((id) => {
          (window as Window & { __nextInputId?: string }).__nextInputId = id;
        }, inputId);
        const startedAt = performance.now();
        const response = createMockHttpResponse();
        await handleRecordInput(createMockHttpRequest({ method: 'POST', body: {
          execution_id: 'feedback-owner', lease_id: 'feedback-lease', input_id: inputId,
          type: 'pointer', action: 'move', x: 350 + (index % 50), y: 350 + Math.floor(index / 50),
        } }), response, sessionId, sessionManager, createTestConfig());
        expect(response.statusCode).toBe(200);
        const receipt = JSON.parse(response.getBody()) as { input_id: string; applied_sequence: number };
        expect(receipt.input_id).toBe(inputId);
        appliedSequences.push(receipt.applied_sequence);

        let observedValue = -1;
        while (observedValue !== index + 1) {
          const packet = await nextFrame();
          if (packet.length < 6) throw new Error('Streamed frame envelope was truncated');
          const headerLength = packet.readUInt32BE(0);
          if (!headerLength || headerLength > packet.length - 6) throw new Error('Invalid streamed frame header');
          const header = JSON.parse(packet.subarray(4, 4 + headerLength).toString('utf8')) as {
            version?: number; source?: { session_id?: string; page_id?: string };
          };
          expect(header.version).toBe(1);
          expect(header.source?.session_id).toBe(sessionId);
          expect(header.source?.page_id).toBe('feedback-page');
          const jpegBase64 = packet.subarray(4 + headerLength).toString('base64');
          observedValue = await decoder.evaluate(async (encoded) => {
            const bytes = Uint8Array.from(atob(encoded), (character) => character.charCodeAt(0));
            const bitmap = await createImageBitmap(new Blob([bytes], { type: 'image/jpeg' }));
            const canvas = document.createElement('canvas');
            canvas.width = bitmap.width;
            canvas.height = bitmap.height;
            const context = canvas.getContext('2d');
            if (!context) throw new Error('Could not create streamed-frame decoder canvas');
            context.drawImage(bitmap, 0, 0);
            const value = Array.from({ length: 10 }, (_, index) => {
              const pixel = context.getImageData(30 + index * 28 + 12, 42, 1, 1).data;
              return pixel[0] > 160 && pixel[1] > 160 && pixel[2] > 160 ? 1 : 0;
            }).reduce((number, bit) => (number << 1) | bit, 0);
            bitmap.close();
            return value;
          }, jpegBase64);
        }
        samples.push(performance.now() - startedAt);
      }

      const report = {
        producer: 'playwright-driver/tests/integration/input-feedback.test.ts',
        scope: 'driver input receipt to matching pixels in a real Chromium CDP-screencast JPEG frame; excludes Go API/WebSocket relay and viewer canvas decode/draw',
        browserVersion: browser.version(),
        viewport: { width: 800, height: 600, deviceScaleFactor: 1 },
        sampleCount: samples.length,
        correlatedFrameCount: samples.length,
        correlationComplete: true,
        receiptSequencesMonotonic: appliedSequences.slice(0, -1).every((sequence, index) => {
          const nextSequence = appliedSequences[index + 1];
          return typeof nextSequence === 'number' && nextSequence > sequence;
        }),
        p50Ms: percentile(samples, 0.50), p95Ms: percentile(samples, 0.95), p99Ms: percentile(samples, 0.99),
        minMs: Math.min(...samples), maxMs: Math.max(...samples), samplesMs: samples,
      };
      // eslint-disable-next-line no-console
      console.log(`BAS_INTERACTIVE_FEEDBACK_FRAME_DIAGNOSTIC ${JSON.stringify(report)}`);
      expect(samples).toHaveLength(count);
      expect(report.receiptSequencesMonotonic).toBe(true);
    } finally {
      await stopFrameStreaming(sessionId);
      socket?.terminate();
      for (const client of server.clients) client.terminate();
      await new Promise<void>((resolve) => server.close(() => resolve()));
      await context.close();
      configuration.mockRestore();
    }
  }, 120000);

  const runAgainstLiveBAS = process.env.BAS_REHAB_LIVE_API_BASE && process.env.BAS_REHAB_LIVE_UI_BASE;
  const liveFeedbackTest = runAgainstLiveBAS ? it : it.skip;

  liveFeedbackTest('correlates live UI inputs with applied receipts and viewer-canvas pixels', async () => {
    const apiBase = process.env.BAS_REHAB_LIVE_API_BASE as string;
    const uiBase = process.env.BAS_REHAB_LIVE_UI_BASE as string;
    const count = Number(process.env.BAS_REHAB_LIVE_SAMPLE_COUNT ?? 1000);
    if (!Number.isSafeInteger(count) || count < 1 || count > 1000) {
      throw new Error('BAS_REHAB_LIVE_SAMPLE_COUNT must be an integer from 1 through 1000');
    }
    const cellCount = 10;
    const cells = Array.from({ length: cellCount }, (_, index) =>
      `<i data-bit="${index}" style="display:block;width:24px;height:24px;background:white"></i>`
    ).join('');
    const fixture = `<!doctype html><html><body style="margin:0;background:rgb(119,119,119)"><div style="position:fixed;left:30px;top:30px;display:flex;gap:4px">${cells}</div><script>window.__inputCount=0;addEventListener('pointermove',()=>{const id=++window.__inputCount;document.querySelectorAll('[data-bit]').forEach((cell,index)=>cell.style.backgroundColor=((id>>(9-index))&1)?'white':'black')})</script></body></html>`;
    let sessionId: string | undefined;
    const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
    await page.addInitScript(() => {
      const probeWindow = window as FeedbackProbeWindow;
      probeWindow.__basPointerEventTimes = [];
      probeWindow.__basInputEvents = [];
      window.addEventListener('pointermove', () => {
        probeWindow.__basPointerEventTimes?.push(performance.now());
      }, true);

      const nativeSend = WebSocket.prototype.send;
      WebSocket.prototype.send = function sendWithProbe(data): void {
        if (typeof data === 'string') {
          try {
            const message = JSON.parse(data) as { type?: string; input?: { input_id?: string } };
            const inputId = message.input?.input_id;
            if (message.type === 'recording_input' && typeof inputId === 'string') {
              probeWindow.__basInputEvents?.push({ kind: 'recording_input', inputId, atMs: performance.now() });
            }
          } catch {
            // Other application WebSocket messages do not carry input IDs.
          }
        }
        nativeSend.call(this, data);
      };

      const messageHandler = Object.getOwnPropertyDescriptor(WebSocket.prototype, 'onmessage');
      if (!messageHandler?.get || !messageHandler.set) throw new Error('WebSocket onmessage instrumentation is unavailable');
      Object.defineProperty(WebSocket.prototype, 'onmessage', {
        configurable: true,
        enumerable: messageHandler.enumerable,
        get() { return messageHandler.get?.call(this); },
        set(handler: ((event: MessageEvent<unknown>) => void) | null) {
          if (!handler) {
            messageHandler.set?.call(this, handler);
            return;
          }
          messageHandler.set?.call(this, function recordAppliedInput(event: MessageEvent<unknown>) {
            if (typeof event.data === 'string') {
              try {
                const message = JSON.parse(event.data) as { type?: string; input_id?: string; applied_sequence?: number };
                if (message.type === 'recording_input_applied' && typeof message.input_id === 'string') {
                  probeWindow.__basInputEvents?.push({
                    kind: 'recording_input_applied',
                    inputId: message.input_id,
                    appliedSequence: message.applied_sequence,
                    atMs: performance.now(),
                  });
                }
              } catch {
                // Invalid and non-input events are handled by the application callback.
              }
            }
            return handler.call(this, event);
          });
        },
      });
    });

    try {
      const created = await fetch(`${apiBase}/recordings/live/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          viewport_width: 1080,
          viewport_height: 836,
          initial_url: `data:text/html,${encodeURIComponent(fixture)}`,
          restore_tabs: false,
          stream_fps: 30,
        }),
      });
      if (!created.ok) throw new Error(`Session creation failed (${created.status}): ${await created.text()}`);
      const createdSession = await created.json() as { session_id?: string };
      if (!createdSession.session_id) throw new Error('Session creation returned no session_id');
      sessionId = createdSession.session_id;

      const started = await fetch(`${apiBase}/recordings/live/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: sessionId, frame_fps: 30 }),
      });
      if (!started.ok) throw new Error(`Recording start failed (${started.status}): ${await started.text()}`);

      await page.goto(`${uiBase}/record/${sessionId}`, { waitUntil: 'domcontentloaded' });
      await page.waitForFunction(() => {
        const canvas = document.querySelector('canvas');
        if (!canvas || canvas.width < 322 || canvas.height < 60 || getComputedStyle(canvas).display === 'none') return false;
        const context = canvas.getContext('2d');
        if (!context) return false;
        const background = context.getImageData(5, 5, 1, 1).data;
        const firstCell = context.getImageData(42, 42, 1, 1).data;
        return background[0] > 90 && background[0] < 150 && firstCell[0] > 180;
      }, { timeout: 30000 });
      const canvas = page.locator('canvas').first();
      const box = await canvas.boundingBox();
      if (!box) throw new Error('The BAS viewer canvas has no visible bounds');
      const dimensions = await canvas.evaluate((element) => ({
        width: (element as HTMLCanvasElement).width,
        height: (element as HTMLCanvasElement).height,
      }));
      const correlatedTimings: Array<{
        pointerEventMs: number;
        socketSendMs: number;
        appliedAckMs: number;
        canvasObservedMs: number;
        appliedSequence: number;
      }> = [];
      const appliedSequences: number[] = [];

      const observeSample = async (pointerIndex: number, inputSendIndex: number, wantedMarker: number) => page.evaluate(({ pointerIndex: expectedPointer, inputSendIndex: expectedSend, marker: expectedMarker }) => {
        const probeWindow = window as FeedbackProbeWindow;
        const pointerTimes = probeWindow.__basPointerEventTimes ?? [];
        const inputEvents = probeWindow.__basInputEvents ?? [];
        const sends = inputEvents.filter(event => event.kind === 'recording_input');
        const pointerEventMs = pointerTimes[expectedPointer];
        const sent = sends[expectedSend];
        const applied = sent
          ? inputEvents.find(event => event.kind === 'recording_input_applied' && event.inputId === sent.inputId)
          : undefined;
        const canvas = document.querySelector('canvas');
        const context = canvas?.getContext('2d');
        if (!context) return { pointerEventCount: pointerTimes.length, pointerEventMs, sent, applied, marker: -1, canvasObservedMs: undefined };
        let marker = 0;
        for (let index = 0; index < 10; index++) {
          const pixel = context.getImageData(42 + index * 28, 42, 1, 1).data;
          if (pixel[0] > 128) marker |= 1 << (9 - index);
        }
        return {
          pointerEventCount: pointerTimes.length,
          pointerEventMs,
          sent,
          applied,
          marker,
          canvasObservedMs: marker === expectedMarker ? performance.now() : undefined,
        };
      }, { pointerIndex, inputSendIndex, marker: wantedMarker });

      for (let index = 0; index < count; index++) {
        const wantedMarker = index + 1;
        const counts = await page.evaluate(() => {
          const probeWindow = window as FeedbackProbeWindow;
          return {
            pointer: probeWindow.__basPointerEventTimes?.length ?? 0,
            sends: probeWindow.__basInputEvents?.filter(event => event.kind === 'recording_input').length ?? 0,
          };
        });
        const x = 200 + (index % Math.min(600, Math.max(1, dimensions.width - 300)));
        const y = 120 + Math.floor(index / 500);
        await page.mouse.move(
          box.x + box.width * x / dimensions.width,
          box.y + box.height * y / dimensions.height,
          { steps: 1 },
        );

        const deadline = performance.now() + 5000;
        let observation = await observeSample(counts.pointer, counts.sends, wantedMarker);
        while ((!observation.sent || !observation.applied || observation.marker !== wantedMarker) && performance.now() < deadline) {
          await new Promise(resolve => setTimeout(resolve, 5));
          observation = await observeSample(counts.pointer, counts.sends, wantedMarker);
        }
        if (!observation.sent || !observation.applied || observation.marker !== wantedMarker || observation.canvasObservedMs === undefined) {
          throw new Error(`Input ${index + 1} did not correlate with its receipt and canvas pixels: ${JSON.stringify({ sent: Boolean(observation.sent), receipt: Boolean(observation.applied), marker: observation.marker, wantedMarker })}`);
        }
        if (typeof observation.applied.appliedSequence !== 'number') {
          throw new Error(`Input ${index + 1} receipt omitted applied_sequence`);
        }
        correlatedTimings.push({
          pointerEventMs: observation.pointerEventMs as number,
          socketSendMs: observation.sent.atMs,
          appliedAckMs: observation.applied.atMs,
          canvasObservedMs: observation.canvasObservedMs,
          appliedSequence: observation.applied.appliedSequence,
        });
        appliedSequences.push(observation.applied.appliedSequence);
      }

      const samplesMs = correlatedTimings.map(sample =>
        sample.canvasObservedMs - sample.pointerEventMs
      );
      const inputToSocketSendMs = correlatedTimings.map(sample =>
        sample.socketSendMs - sample.pointerEventMs
      );
      const socketSendToAckMs = correlatedTimings.map(sample =>
        sample.appliedAckMs - sample.socketSendMs
      );
      const ackToCanvasMs = correlatedTimings.map(sample =>
        sample.canvasObservedMs - sample.appliedAckMs
      );

      const summarize = (values: number[]) => ({
        p50Ms: percentile(values, 0.50),
        p95Ms: percentile(values, 0.95),
        p99Ms: percentile(values, 0.99),
        minMs: Math.min(...values),
        maxMs: Math.max(...values),
      });

      const report = {
        producer: 'playwright-driver/tests/integration/input-feedback.test.ts',
        scope: 'managed BAS recording input over WebSocket through Go input forwarding, driver page mutation, Go frame relay and BAS useFrameStream canvas draw; local loopback only',
        browserVersion: browser.version(),
        viewport: dimensions,
        sampleCount: samplesMs.length,
        correlatedReceiptCount: appliedSequences.length,
        correlatedCanvasPaintCount: samplesMs.length,
        correlationComplete: samplesMs.length === count && appliedSequences.length === count,
        receiptSequencesMonotonic: appliedSequences.slice(0, -1).every((sequence, index) => appliedSequences[index + 1] > sequence),
        inputClock: 'single BAS workspace browser performance.now clock for capture-phase pointer event, WebSocket send/ack handlers and viewer-canvas marker scan',
        p50Ms: percentile(samplesMs, 0.50),
        p95Ms: percentile(samplesMs, 0.95),
        p99Ms: percentile(samplesMs, 0.99),
        minMs: Math.min(...samplesMs),
        maxMs: Math.max(...samplesMs),
        components: {
          inputToSocketSend: summarize(inputToSocketSendMs),
          socketSendToAppliedAck: summarize(socketSendToAckMs),
          appliedAckToCanvasPixels: summarize(ackToCanvasMs),
        },
        samplesMs,
      };
      const receiptPath = process.env.BAS_REHAB_RECEIPT_PATH;
      if (receiptPath) {
        const scenarioRoot = resolve(process.cwd(), '..');
        const evidenceRoot = resolve(scenarioRoot, '.vrooli/runtime/rehabilitation-evidence');
        const outputPath = resolve(process.cwd(), receiptPath);
        if (!outputPath.startsWith(`${evidenceRoot}${sep}`)) {
          throw new Error('BAS_REHAB_RECEIPT_PATH must be inside .vrooli/runtime/rehabilitation-evidence');
        }
        const contractPath = resolve(scenarioRoot, 'docs/internal/REFRACTOR_CONTRACT.json');
        const testPath = resolve(process.cwd(), 'tests/integration/input-feedback.test.ts');
        const [contractBytes, testBytes] = await Promise.all([readFile(contractPath), readFile(testPath)]);
        const healthUrl = new URL(apiBase);
        healthUrl.pathname = healthUrl.pathname.replace(/\/api\/v1\/?$/, '/health');
        const healthResponse = await fetch(healthUrl);
        if (!healthResponse.ok) throw new Error(`Managed API health read failed (${healthResponse.status})`);
        const health = await healthResponse.json() as { build_identity?: string };
        if (!health.build_identity) throw new Error('Managed API health omitted build_identity');
        const sha256 = (bytes: Buffer) => createHash('sha256').update(bytes).digest('hex');
        const receipt = {
          schema_version: 1,
          evidence_kind: 'diagnostic_partial_remote_cohort_pending',
          outcome_id: 'interactive-feedback',
          contract_row: 'bas-rehabilitation-v1#interactive-feedback',
          observed_at: new Date().toISOString(),
          status: 'local_cohort_measured_remote_unmeasured',
          managed_build_identity: health.build_identity,
          source_sha256: {
            'docs/internal/REFRACTOR_CONTRACT.json': sha256(contractBytes),
            'playwright-driver/tests/integration/input-feedback.test.ts': sha256(testBytes),
          },
          producer: {
            owner: 'playwright-driver live BAS integration test',
            test: 'correlates live UI inputs with applied receipts and viewer-canvas pixels',
            result: 'passed',
          },
          measurement: report,
          limitations: [
            'local loopback only; contract remote p95 cohort remains unmeasured',
            'no governed sensor consumes this receipt, so it cannot award setpoint credit',
          ],
        };
        await mkdir(dirname(outputPath), { recursive: true });
        await writeFile(outputPath, `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
      }
      if (!receiptPath) {
        // eslint-disable-next-line no-console
        console.log(`BAS_INTERACTIVE_FEEDBACK_LIVE_DIAGNOSTIC ${JSON.stringify(report)}`);
      } else {
        // eslint-disable-next-line no-console
        console.log(`BAS_INTERACTIVE_FEEDBACK_RECEIPT ${resolve(process.cwd(), receiptPath)}`);
      }
      expect(samplesMs).toHaveLength(count);
      expect(appliedSequences).toHaveLength(count);
      expect(report.receiptSequencesMonotonic).toBe(true);
      expect(report.correlationComplete).toBe(true);
    } finally {
      if (sessionId) {
        await fetch(`${apiBase}/recordings/live/${sessionId}/stop`, { method: 'POST' }).catch(() => undefined);
        await fetch(`${apiBase}/recordings/live/session/${sessionId}/close`, { method: 'POST' }).catch(() => undefined);
      }
      await page.close();
    }
  }, 180000);
});
