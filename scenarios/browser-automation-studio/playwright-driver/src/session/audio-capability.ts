import type { Page } from 'rebrowser-playwright';

/** Measured state of the browser's real-time Web Audio output path. */
export interface AudioCapabilityResult {
  available: boolean;
  currentTimeDelta: number;
  callbackCount: number;
  outputLatency: number | null;
  state: string;
  durationMs: number;
  finding?: string;
}

const minimumClockAdvanceSeconds = 1.5;

/**
 * Proves that a driver-managed page can advance a real-time audio graph.
 * OfflineAudioContext is intentionally not used: it exercises DSP but cannot
 * establish that Chromium started an output stream for this session.
 */
export async function measureRealtimeAudio(
  page: Page,
  durationMs = 2000,
): Promise<AudioCapabilityResult> {
  const sample = await page.evaluate(async (waitMs) => {
    const Context = window.AudioContext ?? (window as typeof window & {
      webkitAudioContext?: typeof AudioContext;
    }).webkitAudioContext;
    if (!Context) {
      return { supported: false, currentTimeDelta: 0, callbackCount: 0, outputLatency: null, state: 'unsupported' };
    }

    const context = new Context();
    const start = context.currentTime;
    let callbackCount = 0;
    const processor = context.createScriptProcessor(1024, 1, 1);
    processor.onaudioprocess = () => { callbackCount += 1; };
    const oscillator = context.createOscillator();
    oscillator.connect(processor);
    processor.connect(context.destination);
    oscillator.start();
    await new Promise((resolve) => window.setTimeout(resolve, waitMs));
    const currentTimeDelta = context.currentTime - start;
    const outputLatency = typeof context.outputLatency === 'number' ? context.outputLatency : null;
    const state = context.state;
    oscillator.stop();
    oscillator.disconnect();
    processor.disconnect();
    await context.close();
    return { supported: true, currentTimeDelta, callbackCount, outputLatency, state };
  }, durationMs);

  const available = sample.supported
    && sample.currentTimeDelta >= minimumClockAdvanceSeconds
    && sample.callbackCount > 0;
  return {
    available,
    currentTimeDelta: sample.currentTimeDelta,
    callbackCount: sample.callbackCount,
    outputLatency: sample.outputLatency,
    state: sample.state,
    durationMs,
    ...(available ? {} : {
      finding: sample.supported
        ? 'realtime_audio_unavailable: audio clock or render callbacks did not advance'
        : 'realtime_audio_unavailable: AudioContext is not supported by this browser session',
    }),
  };
}
