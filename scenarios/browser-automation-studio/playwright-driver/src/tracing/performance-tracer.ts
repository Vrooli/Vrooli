/**
 * Performance Tracer
 *
 * Captures a Chrome DevTools performance trace (Tier 0) for a session,
 * plus a web-vitals summary, and writes both as JSON to a target
 * directory. This is the driver-side mechanism behind BAS's
 * CAPTURE_TYPE_PERFORMANCE artifact.
 *
 * DESIGN CHOICES (P2 of the performance-health buildout):
 * - Raw CDP, not a Playwright high-level tracing API. rebrowser-playwright
 *   (the driver's fork) exposes Playwright's own `context.tracing` (a .zip
 *   workflow-trace), NOT a DevTools-timeline trace. The DevTools timeline +
 *   CPU profile + `blink.user_timing` (the ⚛ React marks) are only
 *   reachable via the CDP `Tracing` domain. The hacky
 *   `scenario-performance-audit` skill proved this exact raw-CDP path.
 * - Streamed trace via `Tracing.end` → `tracingComplete{stream}` → `IO.read`
 *   loop, so even a multi-MB trace never has to fit in one CDP message.
 * - `performance.json` is the DevTools-loadable trace ({ traceEvents: [...],
 *   metadata }). `performance.web-vitals.json` is the injected-observer
 *   summary. Both are plain JSON the Go PerformanceProducer surfaces.
 * - NEVER throws on the capture path: a tracing failure degrades to "no
 *   files written" (Tier 0 absent), never a failed session. The web-vitals
 *   global may be absent (page emitted nothing) — that is a valid Tier 0
 *   result, not an error.
 *
 * Tier 1 (⚛ React component-commit marks) rides along automatically:
 * the `blink.user_timing` category carries any `performance.measure("⚛ …")`
 * the page emits. The tracer is agnostic — it captures whatever the page
 * produced. performance-health does the Tier-1 interpretation downstream.
 */

import type { CDPSession, Page } from 'rebrowser-playwright';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { logger, scopedLog, LogContext } from '../utils';
import { createCDPSession, detachCDPSession } from '../session/cdp-session';
import { WEB_VITALS_GLOBAL, WEB_VITALS_INIT_SCRIPT } from './web-vitals-script';

/** Canonical artifact filenames the Go side reads back. */
export const PERF_TRACE_FILE = 'performance.json';
export const PERF_WEB_VITALS_FILE = 'performance.web-vitals.json';

/**
 * The DevTools-timeline categories to record. This is the canonical set:
 * - devtools.timeline(+disabled-by-default*) : the flame chart / timeline.
 * - disabled-by-default-v8.cpu_profiler      : the CPU sampling profile.
 * - blink.user_timing                        : performance.mark/measure,
 *                                              including the ⚛ React marks
 *                                              (Tier 1 pass-through).
 * - loading / v8.execute                     : navigation + script timing.
 */
export const PERF_TRACE_CATEGORIES = [
  'devtools.timeline',
  'disabled-by-default-devtools.timeline',
  'disabled-by-default-devtools.timeline.frame',
  'disabled-by-default-v8.cpu_profiler',
  'blink.user_timing',
  'loading',
  'v8.execute',
];

/**
 * Minimal CDP command surface the tracer needs. Declaring it lets the unit
 * test drive the tracer with a fake CDP session (no real browser), and
 * documents exactly which CDP calls are made.
 */
export interface TracingCDP {
  send(method: 'Tracing.start', params: Record<string, unknown>): Promise<unknown>;
  send(method: 'Tracing.end'): Promise<unknown>;
  send(method: 'IO.read', params: { handle: string; size?: number }): Promise<{ data: string; eof: boolean; base64Encoded?: boolean }>;
  send(method: 'IO.close', params: { handle: string }): Promise<unknown>;
  send(method: string, params?: Record<string, unknown>): Promise<unknown>;
  on(event: 'Tracing.tracingComplete', cb: (payload: { stream?: string }) => void): void;
  off?(event: 'Tracing.tracingComplete', cb: (payload: { stream?: string }) => void): void;
}

/** A page surface the tracer can read the web-vitals global back from. */
export interface VitalsPage {
  evaluate<T>(fn: (key: string) => T, arg: string): Promise<T>;
}

/**
 * Inject the web-vitals observer into a browser context so it runs in every
 * document before the page's own scripts. Best-effort: a failure here means
 * the trace still captures Tier 0 timeline data, just without the
 * observer-derived web-vitals summary.
 *
 * @param context anything with addInitScript (BrowserContext or Page)
 */
export async function injectWebVitalsObserver(
  context: { addInitScript(script: { content: string }): Promise<void> }
): Promise<void> {
  try {
    await context.addInitScript({ content: WEB_VITALS_INIT_SCRIPT });
  } catch (error) {
    logger.warn(scopedLog(LogContext.TELEMETRY, 'web-vitals observer injection failed'), {
      error: error instanceof Error ? error.message : String(error),
      hint: 'CDP trace will still capture Tier 0 timeline data without web-vitals summary',
    });
  }
}

/**
 * PerformanceTracer owns the CDP tracing lifecycle for one session.
 *
 * Usage:
 *   const tracer = new PerformanceTracer(perfDir);
 *   await tracer.start(page);   // at session start, before navigation
 *   // ... session runs instructions ...
 *   await tracer.stop(page);    // at session close, before context teardown
 */
export class PerformanceTracer {
  private readonly outDir: string;
  private cdp?: CDPSession;
  private started = false;
  private stopped = false;

  /** Test seam: override CDP creation (defaults to a real per-page session). */
  constructor(outDir: string, private readonly cdpFactory: (page: Page) => Promise<CDPSession> = createCDPSession) {
    this.outDir = outDir;
  }

  /** True once tracing has been started and not yet stopped. */
  isActive(): boolean {
    return this.started && !this.stopped;
  }

  /**
   * Start CDP tracing for the page. Best-effort: a failure leaves the tracer
   * inert (isActive() === false) and is logged, never thrown.
   */
  async start(page: Page): Promise<void> {
    if (this.started) {
      return;
    }
    try {
      this.cdp = await this.cdpFactory(page);
      await (this.cdp as unknown as TracingCDP).send('Tracing.start', {
        traceConfig: {
          includedCategories: PERF_TRACE_CATEGORIES,
          recordMode: 'recordAsMuchAsPossible',
        },
        transferMode: 'ReturnAsStream',
        streamFormat: 'json',
      });
      this.started = true;
      logger.debug(scopedLog(LogContext.TELEMETRY, 'performance tracing started'), {
        categories: PERF_TRACE_CATEGORIES.length,
        outDir: this.outDir,
      });
    } catch (error) {
      this.started = false;
      this.cdp = undefined;
      logger.warn(scopedLog(LogContext.TELEMETRY, 'performance tracing start failed'), {
        error: error instanceof Error ? error.message : String(error),
        hint: 'session continues without a performance trace (Tier 0 absent)',
      });
    }
  }

  /**
   * Stop tracing, stream the trace to performance.json, and read the
   * web-vitals global into performance.web-vitals.json. Best-effort and
   * idempotent: safe to call when start() failed or was never called.
   */
  async stop(page: VitalsPage): Promise<void> {
    if (this.stopped) {
      return;
    }
    this.stopped = true;

    // Read the web-vitals global first — it must be read while the page is
    // still alive (before context teardown).
    await this.writeWebVitals(page);

    if (!this.started || !this.cdp) {
      return;
    }
    const cdp = this.cdp as unknown as TracingCDP;
    try {
      const traceJSON = await collectTrace(cdp);
      await mkdir(this.outDir, { recursive: true });
      await writeFile(path.join(this.outDir, PERF_TRACE_FILE), traceJSON, 'utf8');
      logger.info(scopedLog(LogContext.TELEMETRY, 'performance trace written'), {
        bytes: traceJSON.length,
        file: PERF_TRACE_FILE,
      });
    } catch (error) {
      logger.warn(scopedLog(LogContext.TELEMETRY, 'performance trace stop failed'), {
        error: error instanceof Error ? error.message : String(error),
      });
    } finally {
      if (this.cdp) {
        await detachCDPSession(this.cdp);
        this.cdp = undefined;
      }
    }
  }

  private async writeWebVitals(page: VitalsPage): Promise<void> {
    try {
      const vitals = await page.evaluate(
        (key: string) => (window as unknown as Record<string, unknown>)[key] ?? null,
        WEB_VITALS_GLOBAL
      );
      if (vitals == null) {
        // No observable metrics — a valid Tier 0 result; skip the file.
        return;
      }
      await mkdir(this.outDir, { recursive: true });
      await writeFile(
        path.join(this.outDir, PERF_WEB_VITALS_FILE),
        JSON.stringify(vitals, null, 2),
        'utf8'
      );
    } catch (error) {
      logger.warn(scopedLog(LogContext.TELEMETRY, 'web-vitals read failed'), {
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }
}

/**
 * Drive the CDP Tracing.end → tracingComplete{stream} → IO.read loop and
 * return the full trace as a JSON string. The browser writes a JSON stream
 * (an object with a `traceEvents` array); we concatenate the IO chunks
 * verbatim, yielding a DevTools-loadable document.
 */
export async function collectTrace(cdp: TracingCDP): Promise<string> {
  const streamHandle = await new Promise<string | undefined>((resolve, reject) => {
    const onComplete = (payload: { stream?: string }): void => {
      cdp.off?.('Tracing.tracingComplete', onComplete);
      resolve(payload?.stream);
    };
    cdp.on('Tracing.tracingComplete', onComplete);
    cdp.send('Tracing.end').catch((err: unknown) => {
      cdp.off?.('Tracing.tracingComplete', onComplete);
      reject(err);
    });
  });

  if (!streamHandle) {
    // No stream handle: the browser returned the trace some other way or
    // produced nothing. Emit a minimal valid trace document.
    return JSON.stringify({ traceEvents: [], metadata: { source: 'vrooli-bas', note: 'empty-stream' } });
  }

  let out = '';
  for (;;) {
    const chunk = await cdp.send('IO.read', { handle: streamHandle, size: 1024 * 1024 });
    if (chunk?.data) {
      out += chunk.base64Encoded ? Buffer.from(chunk.data, 'base64').toString('utf8') : chunk.data;
    }
    if (chunk?.eof) {
      break;
    }
  }
  await cdp.send('IO.close', { handle: streamHandle }).catch(() => undefined);
  return out;
}
