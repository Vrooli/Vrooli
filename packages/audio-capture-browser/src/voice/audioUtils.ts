// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// Audio filter chain for speech-band isolation and level monitoring.
// Also provides AudioRingBuffer, downsampling, and a passive capture pipeline
// for wake word detection.

export { downsample } from "../pcm";

/**
 * Time-domain window the level meter analyses, in samples.
 *
 * This was 128 (and only half of it was ever read), which is ~1.3ms at a 48kHz
 * AudioContext — a fraction of one glottal period, since a speaking voice runs
 * 85-255Hz and so pulses every 3.9-11.8ms. Successive reads therefore landed at
 * effectively random points in the waveform, and because speech is peaky most
 * of them landed between pulses: the meter reported the gaps, not the phrase.
 * 1024 samples covers ~21ms at 48kHz, long enough to be an envelope rather than
 * an instantaneous sample.
 */
export const LEVEL_ANALYSER_FFT_SIZE = 1024;

/**
 * Release time constant of the meter's envelope follower. Attack is immediate:
 * a rising level is adopted on the tick it appears, a falling one decays over
 * this window so the bar traces the shape of a phrase.
 */
export const LEVEL_METER_RELEASE_MS = 300;

/** Decibels above the adaptive noise floor that map to a full bar. */
export const LEVEL_METER_RANGE_DB = 26;

/** Level-monitor tick interval (~15Hz). Audio analysis does not need 60fps. */
export const LEVEL_TICK_MS = 66;

/**
 * Advance the meter's envelope by one tick. Pure so the curve is testable
 * without an AudioContext.
 */
export function advanceMeterEnvelope(
  previous: number,
  rms: number,
  elapsedMs: number,
  releaseMs: number = LEVEL_METER_RELEASE_MS,
): number {
  if (!Number.isFinite(rms) || rms < 0) return previous;
  if (rms >= previous) return rms;
  const decay = Math.exp(-Math.max(0, elapsedMs) / Math.max(1, releaseMs));
  return Math.max(rms, previous * decay);
}

/**
 * Map an envelope onto a 0-1 bar height, measured in dB above the VAD's own
 * adaptive noise floor rather than on a fixed absolute scale.
 *
 * The previous mapping was `min(1, rms * 4)`, which assumes one microphone gain
 * and one room. The VAD does not make that assumption — it compares the same
 * RMS against thresholds it derives from the running noise floor, which is why
 * speech detection kept working on a meter that read flat. This mapping shares
 * that floor, so the bar self-calibrates the same way.
 */
export function meterLevelFromEnvelope(
  envelope: number,
  noiseFloor: number,
  rangeDb: number = LEVEL_METER_RANGE_DB,
): number {
  if (!Number.isFinite(envelope) || !Number.isFinite(noiseFloor)) return 0;
  if (envelope <= 0 || noiseFloor <= 0 || envelope <= noiseFloor) return 0;
  const db = 20 * Math.log10(envelope / noiseFloor);
  if (!Number.isFinite(db) || db <= 0) return 0;
  return Math.min(1, db / Math.max(1, rangeDb));
}

/**
 * Build a bandpass filter chain targeting the speech band (80Hz-8kHz).
 * Returns an AnalyserNode (for level monitoring) and a filtered MediaStream
 * suitable for MediaRecorder input.
 */
export function createAudioFilterChain(
  ctx: AudioContext,
  source: MediaStreamAudioSourceNode,
): { analyser: AnalyserNode; filteredStream: MediaStream; nodes: AudioNode[] } {
  const highpass = ctx.createBiquadFilter();
  highpass.type = "highpass";
  highpass.frequency.value = 80;
  highpass.Q.value = 0.707; // Butterworth

  const lowpass = ctx.createBiquadFilter();
  lowpass.type = "lowpass";
  lowpass.frequency.value = 8000;
  lowpass.Q.value = 0.707;

  const destination = ctx.createMediaStreamDestination();
  const analyser = ctx.createAnalyser();
  analyser.fftSize = LEVEL_ANALYSER_FFT_SIZE;

  // Chain: source -> highpass -> lowpass -> destination
  //                                     +-> analyser (for level monitoring)
  //                                     +-> silentGain(0) -> ctx.destination
  source.connect(highpass);
  highpass.connect(lowpass);
  lowpass.connect(destination);
  lowpass.connect(analyser);

  // Connect to ctx.destination via a silent gain node so Chrome's Web Audio
  // renderer keeps processing this subgraph. Without a path to the primary
  // destination, Chrome may stop rendering as a power-saving optimisation,
  // causing the AnalyserNode to return stale/silent data. This is the same
  // pattern used in createPassiveCapturePipeline.
  const silentGain = ctx.createGain();
  silentGain.gain.value = 0;
  lowpass.connect(silentGain);
  silentGain.connect(ctx.destination);

  return { analyser, filteredStream: destination.stream, nodes: [highpass, lowpass, destination, analyser, silentGain] };
}

// ---------------------------------------------------------------------------
// AudioRingBuffer — circular PCM buffer for passive wake word capture
// ---------------------------------------------------------------------------

/**
 * Fixed-capacity circular buffer for raw PCM Float32 audio.
 * Writes wrap around when full; reads extract contiguous slices.
 * Typical capacity: 3 seconds at AudioContext sampleRate (~576KB at 48kHz).
 */
export class AudioRingBuffer {
  private buffer: Float32Array;
  private writePos = 0;
  private _totalWritten = 0;
  readonly capacity: number;
  readonly sampleRate: number;

  constructor(durationSeconds: number, sampleRate: number) {
    this.sampleRate = sampleRate;
    this.capacity = Math.ceil(durationSeconds * sampleRate);
    this.buffer = new Float32Array(this.capacity);
  }

  /** Total samples written since construction/reset. Monotonically increasing. */
  get totalWritten(): number {
    return this._totalWritten;
  }

  /** Append PCM samples. Wraps around when full. */
  write(samples: Float32Array): void {
    const len = samples.length;
    if (len >= this.capacity) {
      // Input larger than buffer: just keep the tail
      this.buffer.set(samples.subarray(len - this.capacity));
      this.writePos = 0;
      this._totalWritten += len;
      return;
    }
    const firstPart = Math.min(len, this.capacity - this.writePos);
    this.buffer.set(samples.subarray(0, firstPart), this.writePos);
    if (firstPart < len) {
      this.buffer.set(samples.subarray(firstPart), 0);
    }
    this.writePos = (this.writePos + len) % this.capacity;
    this._totalWritten += len;
  }

  /**
   * Extract the last N samples as a contiguous Float32Array.
   * Returns fewer samples if fewer have been written.
   */
  extractLast(numSamples: number): Float32Array {
    const n = Math.min(numSamples, this.capacity, this._totalWritten);
    if (n === 0) return new Float32Array(0);
    const result = new Float32Array(n);
    const start = (this.writePos - n + this.capacity) % this.capacity;
    if (start + n <= this.capacity) {
      result.set(this.buffer.subarray(start, start + n));
    } else {
      const firstPart = this.capacity - start;
      result.set(this.buffer.subarray(start, this.capacity), 0);
      result.set(this.buffer.subarray(0, n - firstPart), firstPart);
    }
    return result;
  }

  /**
   * Return a mark (totalWritten snapshot) for later range extraction.
   * Use with `extractSinceMark()` to get audio captured after the mark.
   */
  mark(): number {
    return this._totalWritten;
  }

  /**
   * Extract samples written between `fromMark` and now.
   * Returns at most `capacity` samples (oldest data may be overwritten).
   */
  extractSinceMark(fromMark: number): Float32Array {
    const available = this._totalWritten - fromMark;
    if (available <= 0) return new Float32Array(0);
    return this.extractLast(Math.min(available, this.capacity));
  }

  /** Reset the buffer to empty state. */
  reset(): void {
    this.writePos = 0;
    this._totalWritten = 0;
    this.buffer.fill(0);
  }
}

// ---------------------------------------------------------------------------
// Passive capture pipeline — for wake word detection without MediaRecorder
// ---------------------------------------------------------------------------

const PASSIVE_CAPTURE_WORKLET = `class VrooliPassiveCapture extends AudioWorkletProcessor {
  process(inputs) {
    const input = inputs[0] && inputs[0][0];
    if (input) { const copy = input.slice(); this.port.postMessage(copy, [copy.buffer]); }
    return true;
  }
}
registerProcessor("vrooli-passive-capture", VrooliPassiveCapture);`;

// Processor names are registered once per AudioWorkletGlobalScope. Cache both
// in-flight and successful module loads per context to avoid duplicate
// registration when passive listening is stopped and restarted. Failed loads
// are evicted so transient module/CSP failures can recover on a later start.
const passiveWorkletLoads = new WeakMap<AudioContext, Promise<void>>();

function loadPassiveCaptureWorklet(ctx: AudioContext): Promise<void> {
  const cached = passiveWorkletLoads.get(ctx);
  if (cached) return cached;

  const url = URL.createObjectURL(new Blob([PASSIVE_CAPTURE_WORKLET], { type: "application/javascript" }));
  const load = ctx.audioWorklet!.addModule(url).finally(() => URL.revokeObjectURL(url));
  passiveWorkletLoads.set(ctx, load);
  void load.catch(() => {
    if (passiveWorkletLoads.get(ctx) === load) passiveWorkletLoads.delete(ctx);
  });
  return load;
}

/**
 * Build a passive audio capture pipeline that writes raw PCM into a ring buffer.
 * Unlike `createAudioFilterChain`, this does NOT create a MediaStreamDestination
 * (no MediaRecorder needed in passive mode). AudioWorklet captures PCM off the
 * UI thread where available; ScriptProcessor remains the compatibility path.
 *
 * Returns an analyser for VAD RMS monitoring and the processor node for cleanup.
 */
export async function createPassiveCapturePipeline(
  ctx: AudioContext,
  source: MediaStreamAudioSourceNode,
  ringBuffer: AudioRingBuffer,
): Promise<{ analyser: AnalyserNode; nodes: AudioNode[] }> {
  const highpass = ctx.createBiquadFilter();
  highpass.type = "highpass";
  highpass.frequency.value = 80;
  highpass.Q.value = 0.707;

  const lowpass = ctx.createBiquadFilter();
  lowpass.type = "lowpass";
  lowpass.frequency.value = 8000;
  lowpass.Q.value = 0.707;

  const analyser = ctx.createAnalyser();
  analyser.fftSize = 128;

  // Chain: source -> highpass -> lowpass -> analyser and PCM capture.
  source.connect(highpass);
  highpass.connect(lowpass);
  lowpass.connect(analyser);
  const silentGain = ctx.createGain();
  silentGain.gain.value = 0;

  let processor: AudioNode;
  if (ctx.audioWorklet && typeof AudioWorkletNode !== "undefined") {
    try {
      await loadPassiveCaptureWorklet(ctx);
      const worklet = new AudioWorkletNode(ctx, "vrooli-passive-capture", { numberOfInputs: 1, numberOfOutputs: 1, outputChannelCount: [1] });
      worklet.port.onmessage = (event: MessageEvent<Float32Array>) => {
        if (event.data instanceof Float32Array) ringBuffer.write(event.data);
      };
      lowpass.connect(worklet);
      processor = worklet;
    } catch {
      // Unsupported or blocked worklet modules retain the historical fallback.
      const fallback = ctx.createScriptProcessor(4096, 1, 1);
      fallback.onaudioprocess = (event: AudioProcessingEvent) => {
        const input = event.inputBuffer.getChannelData(0);
        ringBuffer.write(input);
        event.outputBuffer.getChannelData(0).set(input);
      };
      lowpass.connect(fallback);
      processor = fallback;
    }
  } else {
    const fallback = ctx.createScriptProcessor(4096, 1, 1);
    fallback.onaudioprocess = (event: AudioProcessingEvent) => {
      const input = event.inputBuffer.getChannelData(0);
      ringBuffer.write(input);
      event.outputBuffer.getChannelData(0).set(input);
    };
    lowpass.connect(fallback);
    processor = fallback;
  }
  silentGain.connect(ctx.destination);
  processor.connect(silentGain);

  return { analyser, nodes: [highpass, lowpass, analyser, processor, silentGain] };
}
