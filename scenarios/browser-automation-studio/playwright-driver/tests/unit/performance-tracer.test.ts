/**
 * Performance Tracer unit tests (no real browser).
 *
 * Drives PerformanceTracer with a fake CDP session and a fake page so the
 * full start → collect-stream → write-files lifecycle is exercised
 * deterministically. A real-browser Tier 0 + ⚛ pass-through proof lives in
 * tests/integration/performance-capture.test.ts.
 */

import { mkdtemp, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { EventEmitter } from 'node:events';
import {
  PerformanceTracer,
  collectTrace,
  PERF_TRACE_FILE,
  PERF_WEB_VITALS_FILE,
  PERF_TRACE_CATEGORIES,
  WEB_VITALS_GLOBAL,
  type TracingCDP,
} from '../../src/tracing';
import type { CDPSession, Page } from 'rebrowser-playwright';

interface TraceEvent {
  cat?: string;
  name?: string;
  ph?: string;
  ts?: number;
  dur?: number;
}
interface TraceDoc {
  traceEvents: TraceEvent[];
  metadata?: Record<string, unknown>;
}
interface IOReadResult {
  data: string;
  eof: boolean;
  base64Encoded?: boolean;
}

/** A fake CDP session that records sends and replays a streamed trace. */
class FakeCDP extends EventEmitter {
  public sends: Array<{ method: string; params?: Record<string, unknown> }> = [];
  private readonly streamChunks: string[];
  private readCount = 0;
  public detached = false;

  constructor(traceDocument: object, chunkCount = 2) {
    super();
    const json = JSON.stringify(traceDocument);
    const size = Math.max(1, Math.ceil(json.length / chunkCount));
    const chunks: string[] = [];
    for (let i = 0; i < json.length; i += size) {
      chunks.push(json.slice(i, i + size));
    }
    this.streamChunks = chunks.length > 0 ? chunks : [json];
  }

  send(method: string, params?: Record<string, unknown>): Promise<unknown> {
    this.sends.push({ method, params });
    if (method === 'Tracing.end') {
      setImmediate(() => this.emit('Tracing.tracingComplete', { stream: 'handle-1' }));
      return Promise.resolve(undefined);
    }
    if (method === 'IO.read') {
      const data = this.streamChunks[this.readCount] ?? '';
      const eof = this.readCount >= this.streamChunks.length - 1;
      this.readCount += 1;
      const result: IOReadResult = { data, eof };
      return Promise.resolve(result);
    }
    return Promise.resolve(undefined);
  }

  detach(): Promise<void> {
    this.detached = true;
    return Promise.resolve();
  }
}

/** Build a fake CDP typed as a real CDPSession for the tracer's factory. */
function asCDP(fake: FakeCDP): CDPSession {
  return fake as unknown as CDPSession;
}

/** A fake page returning a fixed web-vitals global from evaluate(). */
function fakePage(vitals: unknown): Page {
  const page = {
    evaluate: (): Promise<unknown> => Promise.resolve(vitals),
  };
  return page as unknown as Page;
}

describe('PerformanceTracer', () => {
  let dir: string;

  beforeEach(async () => {
    dir = await mkdtemp(path.join(tmpdir(), 'perf-tracer-'));
  });

  afterEach(async () => {
    await rm(dir, { recursive: true, force: true });
  });

  it('starts CDP tracing with the canonical category set', async () => {
    const cdp = new FakeCDP({ traceEvents: [] });
    const tracer = new PerformanceTracer(dir, () => Promise.resolve(asCDP(cdp)));
    await tracer.start({} as Page);

    expect(tracer.isActive()).toBe(true);
    const startCall = cdp.sends.find((s) => s.method === 'Tracing.start');
    expect(startCall).toBeDefined();
    const cfg = startCall?.params?.traceConfig as { includedCategories: string[] };
    expect(cfg.includedCategories).toEqual(PERF_TRACE_CATEGORIES);
    expect(cfg.includedCategories).toContain('blink.user_timing'); // Tier 1 ⚛ marks
    expect(cfg.includedCategories).toContain('disabled-by-default-v8.cpu_profiler');
  });

  it('writes a DevTools-loadable trace and web-vitals on stop', async () => {
    const traceDoc: TraceDoc = {
      traceEvents: [
        { cat: 'devtools.timeline', name: 'RunTask', ph: 'X', ts: 1, dur: 5 },
        { cat: 'disabled-by-default-v8.cpu_profiler', name: 'Profile', ph: 'P', ts: 2 },
      ],
      metadata: { source: 'test' },
    };
    const cdp = new FakeCDP(traceDoc, 3);
    const tracer = new PerformanceTracer(dir, () => Promise.resolve(asCDP(cdp)));
    await tracer.start({} as Page);

    const vitals = { lcp: { value: 123 }, fcp: 50, longTasks: [], cls: { value: 0 } };
    await tracer.stop(fakePage(vitals));

    const traceRaw = await readFile(path.join(dir, PERF_TRACE_FILE), 'utf8');
    const parsed = JSON.parse(traceRaw) as TraceDoc;
    expect(Array.isArray(parsed.traceEvents)).toBe(true);
    expect(parsed.traceEvents).toHaveLength(2);
    expect(parsed.traceEvents[0].name).toBe('RunTask');

    const vitalsRaw = await readFile(path.join(dir, PERF_WEB_VITALS_FILE), 'utf8');
    const vitalsParsed = JSON.parse(vitalsRaw) as { lcp: { value: number } };
    expect(vitalsParsed.lcp.value).toBe(123);

    expect(cdp.detached).toBe(true);
    expect(cdp.sends.some((s) => s.method === 'IO.close')).toBe(true);
  });

  it('omits the web-vitals file when the page emitted no metrics', async () => {
    const cdp = new FakeCDP({ traceEvents: [] });
    const tracer = new PerformanceTracer(dir, () => Promise.resolve(asCDP(cdp)));
    await tracer.start({} as Page);
    await tracer.stop(fakePage(null));

    await expect(readFile(path.join(dir, PERF_WEB_VITALS_FILE), 'utf8')).rejects.toBeDefined();
    await expect(readFile(path.join(dir, PERF_TRACE_FILE), 'utf8')).resolves.toBeDefined();
  });

  it('is inert when CDP start fails (no throw, no files, session unaffected)', async () => {
    const tracer = new PerformanceTracer(dir, () => Promise.reject(new Error('cdp unavailable')));
    await tracer.start({} as Page);
    expect(tracer.isActive()).toBe(false);
    await tracer.stop(fakePage(null));
    await expect(readFile(path.join(dir, PERF_TRACE_FILE), 'utf8')).rejects.toBeDefined();
  });

  it('stop is idempotent', async () => {
    const cdp = new FakeCDP({ traceEvents: [] });
    const tracer = new PerformanceTracer(dir, () => Promise.resolve(asCDP(cdp)));
    await tracer.start({} as Page);
    await tracer.stop(fakePage(null));
    const endCount = cdp.sends.filter((s) => s.method === 'Tracing.end').length;
    await tracer.stop(fakePage(null));
    expect(cdp.sends.filter((s) => s.method === 'Tracing.end').length).toBe(endCount);
  });
});

describe('collectTrace', () => {
  it('concatenates streamed IO chunks into one document', async () => {
    const doc: TraceDoc = { traceEvents: [{ name: 'A' }, { name: 'B' }] };
    const cdp = new FakeCDP(doc, 4);
    const out = await collectTrace(cdp as unknown as TracingCDP);
    expect(JSON.parse(out) as TraceDoc).toEqual(doc);
  });

  it('returns a minimal valid trace when no stream handle is provided', async () => {
    const cdp = new (class extends EventEmitter {
      send(method: string): Promise<unknown> {
        if (method === 'Tracing.end') {
          setImmediate(() => this.emit('Tracing.tracingComplete', {}));
        }
        return Promise.resolve(undefined);
      }
    })();
    const out = await collectTrace(cdp as unknown as TracingCDP);
    const parsed = JSON.parse(out) as TraceDoc;
    expect(parsed.traceEvents).toEqual([]);
  });

  it('decodes base64-encoded chunks', async () => {
    const payload = JSON.stringify({ traceEvents: [{ name: 'Z' }] });
    const cdp = new (class extends EventEmitter {
      send(method: string): Promise<unknown> {
        if (method === 'Tracing.end') {
          setImmediate(() => this.emit('Tracing.tracingComplete', { stream: 'h' }));
          return Promise.resolve(undefined);
        }
        if (method === 'IO.read') {
          const result: IOReadResult = {
            data: Buffer.from(payload, 'utf8').toString('base64'),
            eof: true,
            base64Encoded: true,
          };
          return Promise.resolve(result);
        }
        return Promise.resolve(undefined);
      }
    })();
    const out = await collectTrace(cdp as unknown as TracingCDP);
    const parsed = JSON.parse(out) as TraceDoc;
    expect(parsed.traceEvents[0].name).toBe('Z');
  });
});

describe('web-vitals init script', () => {
  it('references the global the tracer reads back', () => {
    expect(WEB_VITALS_GLOBAL).toBe('__vrooliWebVitals');
  });
});
