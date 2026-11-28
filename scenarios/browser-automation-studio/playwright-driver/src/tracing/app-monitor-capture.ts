import type { CDPSession, Page } from 'rebrowser-playwright';
import { createCDPSession, detachCDPSession } from '../session/cdp-session';

export const APP_MONITOR_TRACE_CONTRACT = 'app-monitor.preview-workspace.trace.v1';
export const APP_MONITOR_TRACE_CATEGORIES = [
  'devtools.timeline',
  'blink.user_timing',
  'v8.execute',
  'disabled-by-default-v8.cpu_profiler',
] as const;

type TraceCDP = CDPSession & {
  send(method: 'Tracing.end'): Promise<unknown>;
  send(
    method: 'IO.read',
    params: { handle: string; size?: number }
  ): Promise<{ data: string; eof: boolean; base64Encoded?: boolean }>;
  send(method: 'IO.close', params: { handle: string }): Promise<unknown>;
  on(event: 'Tracing.tracingComplete', cb: (payload: { stream?: string }) => void): void;
  off?(event: 'Tracing.tracingComplete', cb: (payload: { stream?: string }) => void): void;
};

export interface AppMonitorTraceArtifact {
  contract: typeof APP_MONITOR_TRACE_CONTRACT;
  captureId: string;
  sha256: string;
  byteLength: number;
  traceJson: string;
}

const safeEventNames = new Set([
  'RunTask',
  'Program',
  'EvaluateScript',
  'FunctionCall',
  'EventDispatch',
  'Layout',
  'UpdateLayoutTree',
  'Paint',
  'CompositeLayers',
  'Commit',
  'FireAnimationFrame',
  'TimerFire',
  'TimerInstall',
  'ParseHTML',
  'ResourceSendRequest',
  'ResourceReceiveResponse',
  'ResourceReceivedData',
  'ResourceFinish',
  'FrameStartedLoading',
  'firstContentfulPaint',
  'firstPaint',
  'navigationStart',
  'largestContentfulPaint::Candidate',
]);
const safeCategories = new Set(APP_MONITOR_TRACE_CATEGORIES);
const safePhases = new Set([
  'B',
  'E',
  'X',
  'I',
  'C',
  'M',
  's',
  't',
  'f',
  'b',
  'e',
  'n',
  'O',
  'D',
  'R',
  'N',
  'T',
  'p',
  'F',
]);

/** A one-shot tracer for one authorized App Monitor capture. It stores no
 * caller data and produces only whitelisted trace fields. */
export class AppMonitorTraceCapture {
  private cdp?: TraceCDP;
  private readonly startedAt: number;
  private durationTimer?: NodeJS.Timeout;

  private constructor(
    readonly captureId: string,
    private readonly maxDurationMs: number,
    private readonly maxBytes: number
  ) {
    this.startedAt = Date.now();
  }

  static async start(
    captureId: string,
    maxDurationMs: number,
    maxBytes: number,
    page: Page,
    cdpFactory: (page: Page) => Promise<CDPSession> = createCDPSession
  ): Promise<AppMonitorTraceCapture> {
    if (!/^[0-9a-f-]{36}$/i.test(captureId)) throw new Error('invalid capture ID');
    if (!Number.isSafeInteger(maxDurationMs) || maxDurationMs < 1 || maxDurationMs > 60_000)
      throw new Error('invalid trace duration cap');
    if (!Number.isSafeInteger(maxBytes) || maxBytes < 1 || maxBytes > 16 * 1024 * 1024)
      throw new Error('invalid trace byte cap');
    const capture = new AppMonitorTraceCapture(captureId, maxDurationMs, maxBytes);
    capture.cdp = (await cdpFactory(page)) as TraceCDP;
    try {
      await capture.cdp.send('Tracing.start', {
        traceConfig: {
          includedCategories: [...APP_MONITOR_TRACE_CATEGORIES],
          recordMode: 'recordAsMuchAsPossible',
        },
        transferMode: 'ReturnAsStream',
        streamFormat: 'json',
      });
      capture.durationTimer = setTimeout(() => {
        void capture.stopAndDiscard();
      }, maxDurationMs);
      capture.durationTimer.unref?.();
    } catch (error) {
      await detachCDPSession(capture.cdp);
      capture.cdp = undefined;
      throw error;
    }
    return capture;
  }

  async finish(now = Date.now()): Promise<AppMonitorTraceArtifact> {
    if (!this.cdp) throw new Error('trace capture is not active');
    const durationExceeded = now - this.startedAt > this.maxDurationMs;
    if (this.durationTimer) clearTimeout(this.durationTimer);
    this.durationTimer = undefined;
    const cdp = this.cdp;
    this.cdp = undefined;
    let streamHandle: string | undefined;
    let complete: { stream?: string } | undefined;
    const done = new Promise<{ stream?: string }>((resolve) => {
      const listener = (value: { stream?: string }) => {
        cdp.off?.('Tracing.tracingComplete', listener);
        resolve(value);
      };
      cdp.on('Tracing.tracingComplete', listener);
    });
    try {
      await cdp.send('Tracing.end');
      complete = await done;
      if (!complete.stream) throw new Error('trace stream unavailable');
      streamHandle = complete.stream;
      let raw = '';
      let rawBytes = 0;
      for (;;) {
        const chunk = await cdp.send('IO.read', { handle: streamHandle, size: 64 * 1024 });
        raw += chunk.data ?? '';
        rawBytes += Buffer.byteLength(chunk.data ?? '');
        if (rawBytes > this.maxBytes)
          throw new Error('raw App Monitor trace exceeds the authorized byte limit');
        if (chunk.eof) break;
      }
      await cdp.send('IO.close', { handle: streamHandle });
      streamHandle = undefined;
      if (durationExceeded) throw new Error('App Monitor trace duration limit exceeded');
      const parsed = JSON.parse(raw) as { traceEvents?: unknown[] };
      const events = Array.isArray(parsed.traceEvents)
        ? parsed.traceEvents.map(sanitizeEvent).filter(Boolean)
        : [];
      const traceJson = JSON.stringify({
        traceEvents: events,
        metadata: { source: 'browser-automation-studio', contract: APP_MONITOR_TRACE_CONTRACT },
      });
      const byteLength = Buffer.byteLength(traceJson);
      if (byteLength > this.maxBytes)
        throw new Error('sanitized App Monitor trace exceeds the authorized byte limit');
      const { createHash } = await import('node:crypto');
      return {
        contract: APP_MONITOR_TRACE_CONTRACT,
        captureId: this.captureId,
        sha256: createHash('sha256').update(traceJson).digest('hex'),
        byteLength,
        traceJson,
      };
    } finally {
      if (streamHandle) {
        try {
          await cdp.send('IO.close', { handle: streamHandle });
        } catch {
          /* best-effort stream cleanup */
        }
      }
      await detachCDPSession(cdp);
    }
  }

  async abort(): Promise<void> {
    if (this.durationTimer) clearTimeout(this.durationTimer);
    this.durationTimer = undefined;
    await this.stopAndDiscard();
  }

  private async stopAndDiscard(): Promise<void> {
    const cdp = this.cdp;
    if (!cdp) return;
    this.cdp = undefined;
    this.durationTimer = undefined;
    let stream: string | undefined;
    const done = new Promise<{ stream?: string }>((resolve) => {
      const listener = (value: { stream?: string }) => {
        cdp.off?.('Tracing.tracingComplete', listener);
        resolve(value);
      };
      cdp.on('Tracing.tracingComplete', listener);
    });
    try {
      await cdp.send('Tracing.end');
      stream = (
        await Promise.race([
          done,
          new Promise<undefined>((resolve) => setTimeout(() => resolve(undefined), 2_000)),
        ])
      )?.stream;
      if (stream) await cdp.send('IO.close', { handle: stream });
    } catch {
      // Expired captures are discarded; cleanup cannot turn expiry into output.
    } finally {
      await detachCDPSession(cdp);
    }
  }
}

function sanitizeEvent(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== 'object') return null;
  const event = value as Record<string, unknown>;
  if (
    typeof event.name !== 'string' ||
    typeof event.cat !== 'string' ||
    typeof event.ph !== 'string'
  )
    return null;
  const categories = event.cat
    .split(',')
    .filter((category) =>
      safeCategories.has(category as (typeof APP_MONITOR_TRACE_CATEGORIES)[number])
    );
  if (categories.length === 0 || !safePhases.has(event.ph)) return null;
  if (typeof event.ts !== 'number' || !Number.isFinite(event.ts)) return null;
  const sanitized: Record<string, unknown> = {
    name: safeEventNames.has(event.name) ? event.name : 'OtherEvent',
    cat: categories.join(','),
    ph: event.ph,
    ts: event.ts,
  };
  if (typeof event.dur === 'number' && Number.isFinite(event.dur) && event.dur >= 0)
    sanitized.dur = event.dur;
  if (Number.isSafeInteger(event.pid)) sanitized.pid = event.pid;
  if (Number.isSafeInteger(event.tid)) sanitized.tid = event.tid;
  if (event.args && typeof event.args === 'object') {
    // Intentionally omitted: args may include page URLs, user timing labels,
    // script URLs, or content-derived values.
  }
  return sanitized;
}
