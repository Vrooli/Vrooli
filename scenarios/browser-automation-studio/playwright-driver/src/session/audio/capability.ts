import type { Browser, Page } from 'rebrowser-playwright';

export type HostAudioCapabilityOutcome = 'device_available' | 'no_device' | 'detection_failed';
export type AudioStrategy = 'host_device' | 'synthetic_sink';

export interface HostAudioCapability {
  outcome: HostAudioCapabilityOutcome;
  currentTimeDelta: number;
  durationMs: number;
  reason: string;
}

export interface AudioCapabilityResult {
  available: boolean;
  currentTimeDelta: number;
  callbackCount: number;
  outputLatency: number | null;
  state: string;
  durationMs: number;
  finding?: string;
}

const detectionWindowMs = 1200;
const detectionClockAdvanceSeconds = 0.5;
const realtimeClockAdvanceSeconds = 1.5;

async function evaluateInMainWorld<T>(
  page: Page,
  expression: string,
  fallback: () => Promise<T>
): Promise<T> {
  if (!('context' in page) || typeof page.context !== 'function') return fallback();
  const context = page.context();
  if (!context || typeof context.newCDPSession !== 'function') return fallback();
  const session = await context.newCDPSession(page);
  try {
    const result = await session.send('Runtime.evaluate', {
      expression,
      awaitPromise: true,
      returnByValue: true,
    });
    if (result.exceptionDetails)
      throw new Error(result.exceptionDetails.text ?? 'main-world evaluation failed');
    return result.result.value as T;
  } finally {
    await session.detach().catch(() => undefined);
  }
}

/** Apply the context init script to the already-created initial document. */
export async function applySilentSinkToCurrentPage(page: Page, patch: string): Promise<void> {
  await evaluateInMainWorld(page, patch, async () => {
    await page.evaluate(patch);
  });
}

/** Select the only safe strategy when output detection is uncertain. */
export function selectAudioStrategy(capability: HostAudioCapability): AudioStrategy {
  return capability.outcome === 'device_available' ? 'host_device' : 'synthetic_sink';
}

/** Probe the browser itself, rather than host-specific sound-server state. */
export async function detectHostAudioCapability(browser: Browser): Promise<HostAudioCapability> {
  const startedAt = Date.now();
  let context: Awaited<ReturnType<Browser['newContext']>> | undefined;
  let page: Page | undefined;
  try {
    context = await browser.newContext();
    page = await context.newPage();
    await page.goto('about:blank');
    const probe = page.evaluate(async (waitMs) => {
      const Context =
        window.AudioContext ??
        (
          window as typeof window & {
            webkitAudioContext?: typeof AudioContext;
          }
        ).webkitAudioContext;
      if (!Context) return 0;
      const audioContext = new Context();
      const start = audioContext.currentTime;
      await new Promise((resolve) => window.setTimeout(resolve, waitMs));
      const delta = audioContext.currentTime - start;
      await audioContext.close();
      return delta;
    }, detectionWindowMs);
    const timedOut = Symbol('audio-probe-timeout');
    const currentTimeDelta = await Promise.race([
      probe,
      new Promise<typeof timedOut>((resolve) =>
        setTimeout(() => resolve(timedOut), detectionWindowMs)
      ),
    ]);
    const durationMs = Date.now() - startedAt;
    if (currentTimeDelta === timedOut) {
      // AudioContext construction itself can hang on a dead host sink. Do not
      // await its eventual completion: the process has enough evidence to use
      // the safe synthetic path immediately.
      void probe.catch(() => undefined);
      return {
        outcome: 'no_device',
        currentTimeDelta: 0,
        durationMs,
        reason: `host-output probe timed out after ${detectionWindowMs}ms`,
      };
    }
    if (currentTimeDelta >= detectionClockAdvanceSeconds) {
      return {
        outcome: 'device_available',
        currentTimeDelta,
        durationMs,
        reason: 'browser audio clock advanced during host-output probe',
      };
    }
    return {
      outcome: 'no_device',
      currentTimeDelta,
      durationMs,
      reason: `browser audio clock did not advance ${detectionClockAdvanceSeconds}s within ${detectionWindowMs}ms`,
    };
  } catch (error) {
    return {
      outcome: 'detection_failed',
      currentTimeDelta: 0,
      durationMs: Date.now() - startedAt,
      reason: `host-output probe failed: ${error instanceof Error ? error.message : String(error)}`,
    };
  } finally {
    // Closing a page with a blocked renderer can itself wait for the same
    // sound-server timeout. Request cleanup without extending detection.
    void page?.close().catch(() => undefined);
    void context?.close().catch(() => undefined);
  }
}

export function audioFailureFinding(
  capability: HostAudioCapability,
  strategy: AudioStrategy,
  measurement: Pick<AudioCapabilityResult, 'currentTimeDelta' | 'durationMs'>
): string {
  const prefix =
    strategy === 'synthetic_sink'
      ? 'realtime_audio_driver_failure'
      : 'realtime_audio_host_output_unavailable';
  return `${prefix}: host_audio=${capability.outcome}; strategy=${strategy}; clock_delta=${measurement.currentTimeDelta}s; window=${measurement.durationMs}ms; detection=${capability.reason}`;
}

export async function measureRealtimeAudio(
  page: Page,
  durationMs = 2000,
  capability?: HostAudioCapability,
  strategy?: AudioStrategy
): Promise<AudioCapabilityResult> {
  const expression = `(async () => {
    const Context = window.AudioContext ?? window.webkitAudioContext;
    if (!Context) return { supported: false, currentTimeDelta: 0, callbackCount: 0, outputLatency: null, state: 'unsupported' };
    const context = new Context();
    await Promise.race([
      (async () => { if (${strategy === 'synthetic_sink'} && typeof context.setSinkId === 'function') await context.setSinkId({ type: 'none' }); await context.resume(); })(),
      new Promise((resolve) => window.setTimeout(resolve, 1000)),
    ]);
    const start = context.currentTime;
    let callbackCount = 0;
    const processor = context.createScriptProcessor(1024, 1, 1);
    processor.onaudioprocess = () => { callbackCount += 1; };
    const oscillator = context.createOscillator();
    oscillator.connect(processor);
    processor.connect(context.destination);
    oscillator.start();
    await new Promise((resolve) => window.setTimeout(resolve, ${durationMs}));
    const currentTimeDelta = context.currentTime - start;
    const outputLatency = typeof context.outputLatency === 'number' ? context.outputLatency : null;
    const state = context.state;
    oscillator.stop(); oscillator.disconnect(); processor.disconnect(); await context.close();
    return { supported: true, currentTimeDelta, callbackCount, outputLatency, state };
  })()`;
  const sample = await evaluateInMainWorld(page, expression, () =>
    page.evaluate(async (waitMs) => {
      const Context =
        window.AudioContext ??
        (window as typeof window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
      if (!Context)
        return {
          supported: false,
          currentTimeDelta: 0,
          callbackCount: 0,
          outputLatency: null,
          state: 'unsupported',
        };
      const context = new Context();
      await context.resume();
      const start = context.currentTime;
      let callbackCount = 0;
      const processor = context.createScriptProcessor(1024, 1, 1);
      processor.onaudioprocess = (): void => {
        callbackCount += 1;
      };
      const oscillator = context.createOscillator();
      oscillator.connect(processor);
      processor.connect(context.destination);
      oscillator.start();
      await new Promise((resolve) => window.setTimeout(resolve, waitMs));
      const currentTimeDelta = context.currentTime - start;
      const outputLatency =
        typeof context.outputLatency === 'number' ? context.outputLatency : null;
      const state = context.state;
      oscillator.stop();
      oscillator.disconnect();
      processor.disconnect();
      await context.close();
      return { supported: true, currentTimeDelta, callbackCount, outputLatency, state };
    }, durationMs)
  );
  const available =
    sample.supported &&
    sample.currentTimeDelta >= realtimeClockAdvanceSeconds &&
    sample.callbackCount > 0;
  const result = {
    available,
    currentTimeDelta: sample.currentTimeDelta,
    callbackCount: sample.callbackCount,
    outputLatency: sample.outputLatency,
    state: sample.state,
    durationMs,
  };
  if (available) return result;
  if (capability && strategy)
    return { ...result, finding: audioFailureFinding(capability, strategy, result) };
  return {
    ...result,
    finding: sample.supported
      ? 'realtime_audio_host_output_unavailable: host audio capability was not recorded'
      : 'realtime_audio_host_output_unavailable: AudioContext is not supported by this browser session',
  };
}

/**
 * Measure audio in a fresh context owned by the managed browser. Keeping this
 * here makes the audio diagnostic lifecycle (probe, strategy, measurement)
 * explicit and prevents routes from creating unpatched ad-hoc contexts.
 */
export async function measureBareRealtimeAudio(
  browser: Browser,
  capability: HostAudioCapability,
  strategy: AudioStrategy,
  durationMs = 2000
): Promise<AudioCapabilityResult> {
  const context = await browser.newContext();
  const page = await context.newPage();
  try {
    await page.goto('about:blank');
    return await measureRealtimeAudio(page, durationMs, capability, strategy);
  } finally {
    await page.close().catch(() => undefined);
    await context.close().catch(() => undefined);
  }
}
