import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { createServer } from 'node:http';
import { dirname, resolve } from 'node:path';
import { chromium, type Browser, type Page } from 'rebrowser-playwright';

type MotionSample = {
  capturedAt: string;
  frameAgeMs: number;
  frameBytes: number;
  decodeMs: number;
  marker: number;
};

type FixtureSchedulerSample = {
  observedAt: string;
  durationMs: number;
  rafTicks: number;
  paintUpdates: number;
};

type MotionWindow = {
  startedAt: string;
  durationMs: number;
  receivedFrames: number;
  decodedFrames: number;
  renderedFrames: number;
  uniqueFixtureFrames: number;
  maxConcurrentDecodes: number;
  stallCount: number;
  samples: MotionSample[];
};

type MotionProbeWindow = Window & {
  __basMotionProbe?: {
    collecting: boolean;
    observedFrames: number;
    receivedFrames: number;
    decodedFrames: number;
    renderedFrames: number;
    activeDecodes: number;
    inWindowDecodes: number;
    maxConcurrentDecodes: number;
    stallCount: number;
    stallEnabled: boolean;
    decodeDelayMs: number;
    startedAt: number;
    endedAt: number;
    endTimer?: number;
    samples: MotionSample[];
  };
};

function percentile(values: number[], quantile: number): number {
  const ordered = [...values].sort((left, right) => left - right);
  return ordered[Math.ceil(quantile * ordered.length) - 1] ?? 0;
}

function summarize(window: MotionWindow) {
  const ages = window.samples.map((sample) => sample.frameAgeMs);
  const decodeTimes = window.samples.map((sample) => sample.decodeMs);
  const uniqueFixtureFrames = new Set(
    window.samples.map((sample) => sample.marker).filter((marker) => marker >= 0)
  ).size;
  return {
    ...window,
    uniqueFixtureFrames,
    renderedFps: window.renderedFrames / (window.durationMs / 1000),
    p95FrameAgeMs: percentile(ages, 0.95),
    maxFrameAgeMs: Math.max(...ages),
    maxFrameBytes: Math.max(...window.samples.map((sample) => sample.frameBytes)),
    p95DecodeMs: percentile(decodeTimes, 0.95),
    maxDecodeMs: Math.max(...decodeTimes),
  };
}

function expectBaselineInBand(baseline: ReturnType<typeof summarize>) {
  // A finite capture window can start and stop between frame boundaries. Allow
  // one frame of endpoint uncertainty while still requiring 9,000 distinct
  // rendered frames and the full five-minute observation.
  const boundaryFps = 1000 / baseline.durationMs;
  expect(baseline.durationMs).toBeGreaterThanOrEqual(300_000);
  expect(baseline.renderedFps + boundaryFps).toBeGreaterThanOrEqual(30);
  expect(baseline.uniqueFixtureFrames).toBeGreaterThanOrEqual(9_000);
  expect(baseline.p95FrameAgeMs).toBeLessThanOrEqual(100);
  expect(baseline.maxFrameBytes).toBeLessThanOrEqual(12 * 1024 * 1024 + 4 * 1024);
  expect(baseline.p95DecodeMs).toBeLessThanOrEqual(100);
}

describe('managed BAS live motion qualification', () => {
  let browser: Browser;

  beforeAll(async () => {
    browser = await chromium.launch({ headless: true });
  });

  afterAll(async () => {
    await browser?.close();
  });

  it('renders five minutes of changing 30 FPS motion and stays bounded during a slow-reader cohort', async () => {
    const apiBase = process.env.BAS_REHAB_LIVE_API_BASE;
    const uiBase = process.env.BAS_REHAB_LIVE_UI_BASE;
    const observationPath = process.env.BAS_MOTION_OBSERVATION_PATH;
    const viewportWidth = Number(process.env.BAS_MOTION_VIEWPORT_WIDTH || 1080);
    const viewportHeight = Number(process.env.BAS_MOTION_VIEWPORT_HEIGHT || 836);
    if (!apiBase || !uiBase || !observationPath) {
      throw new Error(
        'BAS_REHAB_LIVE_API_BASE, BAS_REHAB_LIVE_UI_BASE and BAS_MOTION_OBSERVATION_PATH are required'
      );
    }

    const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
    const fixtureCells = Array.from(
      { length: 16 },
      (_, index) =>
        `<i style="display:block;width:22px;height:22px;background:black" data-motion-bit="${index}"></i>`
    ).join('');
    const fixtureScheduler: FixtureSchedulerSample[] = [];
    const fixtureSchedulerServer = createServer((request, response) => {
      response.setHeader('Access-Control-Allow-Origin', '*');
      response.setHeader('Access-Control-Allow-Methods', 'POST, OPTIONS');
      if (request.method === 'OPTIONS') {
        response.writeHead(204).end();
        return;
      }
      if (request.method !== 'POST' || request.url !== '/scheduler') {
        response.writeHead(404).end();
        return;
      }
      let body = '';
      request.setEncoding('utf8');
      request.on('data', (chunk: string) => {
        body += chunk;
      });
      request.on('end', () => {
        try {
          fixtureScheduler.push(JSON.parse(body) as FixtureSchedulerSample);
          response.writeHead(204).end();
        } catch {
          response.writeHead(400).end();
        }
      });
    });
    await new Promise<void>((resolveListen, rejectListen) => {
      fixtureSchedulerServer.once('error', rejectListen);
      fixtureSchedulerServer.listen(0, '127.0.0.1', resolveListen);
    });
    const fixtureSchedulerAddress = fixtureSchedulerServer.address();
    if (!fixtureSchedulerAddress || typeof fixtureSchedulerAddress === 'string')
      throw new Error('Fixture scheduler observer did not bind a TCP port');
    const fixtureSchedulerUrl = `http://127.0.0.1:${fixtureSchedulerAddress.port}/scheduler`;
    const fixture = `<!doctype html><html><body style="margin:0;background:rgb(119,119,119)"><div style="position:fixed;left:20px;top:20px;display:flex;flex-wrap:wrap;width:204px;gap:4px">${fixtureCells}</div><script>
      const schedulerURL=${JSON.stringify(fixtureSchedulerUrl)};
      const paintIntervalMs=1000/30;let marker=0,previousRafAt=null,paintTimeRemaining=0,rafTicks=0,paintUpdates=0,windowStarted=performance.now();const bits=[...document.querySelectorAll('[data-motion-bit]')];
      const reportScheduler=()=>{const durationMs=performance.now()-windowStarted;fetch(schedulerURL,{method:'POST',mode:'cors',headers:{'Content-Type':'text/plain'},body:JSON.stringify({observedAt:new Date().toISOString(),durationMs,rafTicks,paintUpdates})}).catch(()=>{});rafTicks=0;paintUpdates=0;windowStarted=performance.now();};
      setInterval(reportScheduler,1000);
      const paint=now=>{rafTicks++;if(previousRafAt!==null)paintTimeRemaining+=now-previousRafAt;previousRafAt=now;if(paintTimeRemaining>=paintIntervalMs-1){paintTimeRemaining-=paintIntervalMs;marker=(marker+1)&65535;paintUpdates++;bits.forEach((cell,index)=>{cell.style.backgroundColor=((marker>>(15-index))&1)?'white':'black'});}requestAnimationFrame(paint)};
      requestAnimationFrame(paint);
    </script></body></html>`;

    await page.addInitScript(() => {
      const probeWindow = window as MotionProbeWindow;
      const probe = {
        collecting: false,
        observedFrames: 0,
        receivedFrames: 0,
        decodedFrames: 0,
        renderedFrames: 0,
        activeDecodes: 0,
        inWindowDecodes: 0,
        maxConcurrentDecodes: 0,
        stallCount: 0,
        startedAt: 0,
        stallEnabled: false,
        decodeDelayMs: 0,
        samples: [] as MotionSample[],
        endedAt: 0,
      };
      probeWindow.__basMotionProbe = probe;
      const blobFrames = new WeakMap<Blob, { capturedAt: string; frameBytes: number; inWindow: boolean }>();
      const bitmapFrames = new WeakMap<
        ImageBitmap,
        { capturedAt: string; frameBytes: number; decodeMs: number; inWindow: boolean }
      >();
      let activeFrame: { capturedAt: string; frameBytes: number; inWindow: boolean } | null = null;

      const blobConstructor = new Proxy(window.Blob, {
        construct(target, args) {
          const blob = Reflect.construct(target, args) as Blob;
          if (activeFrame) blobFrames.set(blob, activeFrame);
          return blob;
        },
      });
      Object.defineProperty(window, 'Blob', {
        configurable: true,
        writable: true,
        value: blobConstructor,
      });

      const messageProperty = Object.getOwnPropertyDescriptor(WebSocket.prototype, 'onmessage');
      if (!messageProperty?.get || !messageProperty.set)
        throw new Error('WebSocket frame observation is unavailable');
      Object.defineProperty(WebSocket.prototype, 'onmessage', {
        configurable: true,
        enumerable: messageProperty.enumerable,
        get() {
          return messageProperty.get?.call(this);
        },
        set(handler: ((event: MessageEvent<unknown>) => void) | null) {
          if (!handler) {
            messageProperty.set?.call(this, handler);
            return;
          }
          messageProperty.set?.call(this, function observeFrame(event: MessageEvent<unknown>) {
            const previous = activeFrame;
            if (event.data instanceof ArrayBuffer && event.data.byteLength >= 6) {
              try {
                const packet = new DataView(event.data);
                const headerSize = packet.getUint32(0);
                if (headerSize > 0 && headerSize <= event.data.byteLength - 6) {
                  const header = JSON.parse(
                    new TextDecoder().decode(new Uint8Array(event.data, 4, headerSize))
                  ) as { captured_at?: string };
                  if (typeof header.captured_at === 'string') {
                    activeFrame = {
                      capturedAt: header.captured_at,
                      frameBytes: event.data.byteLength - 4 - headerSize,
                      inWindow: probe.collecting,
                    };
                    probe.observedFrames++;
                    if (activeFrame.inWindow) probe.receivedFrames++;
                  }
                }
              } catch {
                // Non-frame WebSocket messages remain owned by the BAS handler.
              }
            }
            try {
              return handler.call(this, event);
            } finally {
              activeFrame = previous;
            }
          });
        },
      });

      const nativeCreateImageBitmap = window.createImageBitmap.bind(window);
      window.createImageBitmap = (async (...args: Parameters<typeof createImageBitmap>) => {
        const source = args[0];
        const frame = source instanceof Blob ? blobFrames.get(source) : undefined;
        if (!frame) return nativeCreateImageBitmap(...args);
        const startedAt = performance.now();
        probe.activeDecodes++;
        probe.maxConcurrentDecodes = Math.max(probe.maxConcurrentDecodes, probe.activeDecodes);
        if (frame?.inWindow) probe.inWindowDecodes++;
        try {
          if (probe.decodeDelayMs > 0) {
            await new Promise((resolve) => setTimeout(resolve, probe.decodeDelayMs));
          }
          const bitmap = await nativeCreateImageBitmap(...args);
          if (frame?.inWindow) probe.decodedFrames++;
          if (frame) bitmapFrames.set(bitmap, { ...frame, decodeMs: performance.now() - startedAt });
          return bitmap;
        } finally {
          probe.activeDecodes--;
          if (frame?.inWindow) probe.inWindowDecodes--;
        }
      }) as typeof createImageBitmap;

      const nativeDrawImage = CanvasRenderingContext2D.prototype.drawImage;
      CanvasRenderingContext2D.prototype.drawImage = function observeDrawImage(
        ...args: Parameters<typeof nativeDrawImage>
      ) {
        const result = nativeDrawImage.apply(this, args);
        const frame = bitmapFrames.get(args[0] as ImageBitmap);
        if (frame?.inWindow && this.canvas.width >= 450 && this.canvas.height >= 450) {
          // Sample one pixel at each cell center with two narrow row reads.
          // Reading the full 183x27 marker box adds avoidable synchronous work
          // to every viewer paint and can itself drop frames in a five-minute
          // qualification window.
          const markerRows = [
            this.getImageData(31, 31, 183, 1).data,
            this.getImageData(31, 57, 183, 1).data,
          ];
          let marker = 0;
          for (let index = 0; index < 16; index++) {
            const row = Math.floor(index / 8);
            const x = (index % 8) * 26;
            const red = markerRows[row][x * 4];
            if (red > 128) marker |= 1 << (15 - index);
          }
          probe.renderedFrames++;
          probe.samples.push({
            capturedAt: frame.capturedAt,
            frameAgeMs: Date.now() - Date.parse(frame.capturedAt),
            frameBytes: frame.frameBytes,
            decodeMs: frame.decodeMs,
            marker,
          });
        }
        return result;
      };

      setInterval(() => {
        if (!probe.collecting || !probe.stallEnabled) return;
        const start = performance.now();
        while (performance.now() - start < 250) {
          /* intentionally slow the viewer event loop */
        }
        probe.stallCount++;
      }, 5000);
    });

    const openSession = async () => {
      const created = await fetch(`${apiBase}/recordings/live/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          viewport_width: viewportWidth,
          viewport_height: viewportHeight,
          initial_url: `data:text/html,${encodeURIComponent(fixture)}`,
          restore_tabs: false,
          stream_fps: 30,
        }),
      });
      if (!created.ok)
        throw new Error(`Session creation failed (${created.status}): ${await created.text()}`);
      const value = (await created.json()) as { session_id?: string };
      if (!value.session_id) throw new Error('Session creation omitted session_id');
      const started = await fetch(`${apiBase}/recordings/live/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: value.session_id, frame_fps: 30 }),
      });
      if (!started.ok)
        throw new Error(`Recording start failed (${started.status}): ${await started.text()}`);
      await page.goto(`${uiBase}/record/${value.session_id}`, { waitUntil: 'domcontentloaded' });
      await page.waitForFunction(
        () => {
          const canvas = document.querySelector('canvas');
          const probe = (window as MotionProbeWindow).__basMotionProbe;
          return Boolean(
            canvas &&
            canvas.width >= 450 &&
            canvas.height >= 450 &&
            probe &&
            probe.observedFrames >= 5
          );
        },
        { timeout: 30000 }
      );
      return value.session_id;
    };
    const closeSession = async (sessionId: string) => {
      await fetch(`${apiBase}/recordings/live/${sessionId}/stop`, { method: 'POST' }).catch(
        () => undefined
      );
      const closed = await fetch(`${apiBase}/recordings/live/session/${sessionId}/close`, {
        method: 'POST',
      });
      if (!closed.ok) throw new Error(`Session ${sessionId} cleanup failed (${closed.status})`);
    };
    const startWindow = async (slowReader: boolean, durationMs: number) => {
      await page.evaluate(({ isSlowReader, durationMs: windowDurationMs }) => {
        const probe = (window as MotionProbeWindow).__basMotionProbe;
        if (!probe) throw new Error('Motion frame probe was not installed');
        if (probe.endTimer !== undefined) clearTimeout(probe.endTimer);
        probe.samples = [];
        probe.receivedFrames = 0;
        probe.decodedFrames = 0;
        probe.renderedFrames = 0;
        probe.inWindowDecodes = 0;
        probe.stallCount = 0;
        probe.maxConcurrentDecodes = probe.activeDecodes;
        probe.startedAt = performance.now();
        probe.endedAt = 0;
        probe.stallEnabled = isSlowReader;
        probe.decodeDelayMs = isSlowReader ? 80 : 0;
        probe.collecting = true;
        probe.endTimer = window.setTimeout(() => {
          probe.endedAt = performance.now();
          probe.collecting = false;
        }, windowDurationMs);
      }, { isSlowReader: slowReader, durationMs });
    };
    const finishWindow = async (): Promise<MotionWindow> =>
      page.evaluate(async () => {
        const probe = (window as MotionProbeWindow).__basMotionProbe;
        if (!probe) throw new Error('Motion frame probe was not installed');
        const endedAt = probe.endedAt || performance.now();
        if (probe.endTimer !== undefined) clearTimeout(probe.endTimer);
        probe.collecting = false;
        const drainDeadline = performance.now() + 1_000;
        while (probe.inWindowDecodes > 0 && performance.now() < drainDeadline) {
          await new Promise((resolve) => setTimeout(resolve, 1));
        }
        if (probe.inWindowDecodes > 0) {
          throw new Error(`Timed out draining ${probe.inWindowDecodes} in-window frame decode(s)`);
        }
        await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
        await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
        const measuredDurationMs = endedAt - probe.startedAt;
        return {
          startedAt: new Date(Date.now() - measuredDurationMs).toISOString(),
          durationMs: measuredDurationMs,
          receivedFrames: probe.receivedFrames,
          decodedFrames: probe.decodedFrames,
          renderedFrames: probe.renderedFrames,
          uniqueFixtureFrames: new Set(probe.samples.map((sample) => sample.marker)).size,
          maxConcurrentDecodes: probe.maxConcurrentDecodes,
          stallCount: probe.stallCount,
          samples: probe.samples,
        };
      });

    const observations: {
      baseline?: ReturnType<typeof summarize>;
      slowReader?: MotionWindow;
      driverSettings?: unknown;
      fixtureScheduler?: FixtureSchedulerSample[];
    } = {};
    let sessionId: string | undefined;
    try {
      sessionId = await openSession();
      const rateSmokeMs = Number(process.env.BAS_MOTION_RATE_SMOKE_MS || 0);
      const rateSmokeFps = Number(process.env.BAS_MOTION_RATE_SMOKE_FPS || 0);
      if (rateSmokeMs > 0 && rateSmokeFps > 0) {
        const streamResponse = await fetch(
          `${apiBase}/recordings/live/${sessionId}/stream-settings`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ fps: rateSmokeFps }),
          }
        );
        if (!streamResponse.ok) {
          throw new Error(
            `Motion rate-smoke FPS update failed (${streamResponse.status}): ${await streamResponse.text()}`
          );
        }
      }
      if (process.env.BAS_MOTION_STARTUP_ONLY === '1') {
        // Fast focused check for the managed session/viewer path before the
        // five-minute performance cohort is run.
        await closeSession(sessionId);
        sessionId = undefined;
        return;
      }
      const priorObservationPath = process.env.BAS_MOTION_BASELINE_OBSERVATION_PATH;
      if (priorObservationPath) {
        const prior = JSON.parse(await readFile(priorObservationPath, 'utf8')) as {
          baseline?: ReturnType<typeof summarize>;
          fixtureScheduler?: FixtureSchedulerSample[];
        };
        if (!prior.baseline) throw new Error('Prior motion observation has no baseline');
        observations.baseline = prior.baseline;
        observations.fixtureScheduler = prior.fixtureScheduler ?? [];
        expectBaselineInBand(observations.baseline);
      } else {
        const baselineMs = rateSmokeMs > 0 ? rateSmokeMs : 300_000;
        await startWindow(false, baselineMs);
        await page.waitForFunction(
          () => ((window as MotionProbeWindow).__basMotionProbe?.endedAt ?? 0) > 0,
          undefined,
          { timeout: baselineMs + 10_000 }
        );
        observations.baseline = summarize(await finishWindow());
        observations.fixtureScheduler = [...fixtureScheduler];
        expect(observations.fixtureScheduler.length).toBeGreaterThan(0);
      }
      if (!priorObservationPath && rateSmokeMs > 0) {
        expect(observations.baseline.renderedFrames).toBeGreaterThan(0);
        const fixtureDurationMs = observations.fixtureScheduler.reduce(
          (total, sample) => total + sample.durationMs,
          0
        );
        const fixturePaintUpdates = observations.fixtureScheduler.reduce(
          (total, sample) => total + sample.paintUpdates,
          0
        );
        expect(fixturePaintUpdates / (fixtureDurationMs / 1000)).toBeGreaterThanOrEqual(29);
        const streamResponse = await fetch(
          `${apiBase}/recordings/live/${sessionId}/stream-settings`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({}),
          }
        );
        const streamBody = await streamResponse.text();
        observations.driverSettings = streamResponse.ok
          ? JSON.parse(streamBody)
          : { status: streamResponse.status, body: streamBody };
        expect(streamResponse.ok).toBe(true);
        return;
      }
      expectBaselineInBand(observations.baseline!);
      await closeSession(sessionId);
      sessionId = undefined;

      sessionId = await openSession();
      await startWindow(true, 18_000);
      await page.waitForFunction(
        () => ((window as MotionProbeWindow).__basMotionProbe?.endedAt ?? 0) > 0,
        undefined,
        { timeout: 23_000 }
      );
      observations.slowReader = summarize(await finishWindow());
      expect(observations.slowReader.stallCount).toBeGreaterThanOrEqual(2);
      expect(observations.slowReader.receivedFrames).toBeGreaterThan(
        observations.slowReader.renderedFrames
      );
      expect(observations.slowReader.maxConcurrentDecodes).toBe(1);
      expect(
        Math.max(...observations.slowReader.samples.map((sample) => sample.frameAgeMs))
      ).toBeLessThanOrEqual(1000);
      expect(
        Math.max(...observations.slowReader.samples.map((sample) => sample.frameBytes))
      ).toBeLessThanOrEqual(12 * 1024 * 1024 + 4 * 1024);

      // eslint-disable-next-line no-console
      console.log(
        `BAS_MOTION_QUALIFICATION ${JSON.stringify({ baseline: observations.baseline, slowReader: observations.slowReader })}`
      );
    } finally {
      // Keep any completed cohort when a later threshold assertion fails. This
      // gives the owner actionable measurements without counting them as proof.
      if (observations.baseline || observations.slowReader) {
        const output = resolve(observationPath);
        await mkdir(dirname(output), { recursive: true });
        await writeFile(output, `${JSON.stringify(observations, null, 2)}\n`, {
          mode: 0o600,
          flag: 'wx',
        });
      }
      if (sessionId) await closeSession(sessionId).catch(() => undefined);
      await new Promise<void>((resolveClose) => fixtureSchedulerServer.close(() => resolveClose()));
      await page.close();
    }
  }, 390_000);
});
