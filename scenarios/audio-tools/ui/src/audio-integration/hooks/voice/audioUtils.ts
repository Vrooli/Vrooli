// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// Audio filter chain for speech-band isolation and level monitoring.
// Also provides AudioRingBuffer, downsampling, and a passive capture pipeline
// for wake word detection.

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
  analyser.fftSize = 128;

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
// Downsampling — linear interpolation for MFCC input
// ---------------------------------------------------------------------------

/**
 * Downsample audio from `fromRate` to `toRate` using linear interpolation.
 * Used to convert AudioContext's native rate (typically 48kHz) to 16kHz for MFCC.
 */
export function downsample(buffer: Float32Array<ArrayBufferLike>, fromRate: number, toRate: number): Float32Array {
  if (fromRate === toRate) return buffer;
  if (toRate > fromRate) throw new Error(`Cannot upsample: ${fromRate} -> ${toRate}`);

  const ratio = fromRate / toRate;
  const outputLength = Math.ceil(buffer.length / ratio);
  const output = new Float32Array(outputLength);

  for (let i = 0; i < outputLength; i++) {
    const srcIndex = i * ratio;
    const srcFloor = Math.floor(srcIndex);
    const srcCeil = Math.min(srcFloor + 1, buffer.length - 1);
    const frac = srcIndex - srcFloor;
    output[i] = (buffer[srcFloor] ?? 0) * (1 - frac) + (buffer[srcCeil] ?? 0) * frac;
  }

  return output;
}

// ---------------------------------------------------------------------------
// Passive capture pipeline — for wake word detection without MediaRecorder
// ---------------------------------------------------------------------------

/**
 * Build a passive audio capture pipeline that writes raw PCM into a ring buffer.
 * Unlike `createAudioFilterChain`, this does NOT create a MediaStreamDestination
 * (no MediaRecorder needed in passive mode). Instead, a ScriptProcessorNode
 * captures PCM samples directly into the ring buffer.
 *
 * Returns an analyser for VAD RMS monitoring and the processor node for cleanup.
 */
export function createPassiveCapturePipeline(
  ctx: AudioContext,
  source: MediaStreamAudioSourceNode,
  ringBuffer: AudioRingBuffer,
): { analyser: AnalyserNode; nodes: AudioNode[] } {
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

  // ScriptProcessorNode (deprecated but universally supported) for PCM capture.
  // Buffer size 4096 at 48kHz = ~85ms per callback — low overhead.
  const processor = ctx.createScriptProcessor(4096, 1, 1);
  processor.onaudioprocess = (e: AudioProcessingEvent) => {
    const input = e.inputBuffer.getChannelData(0);
    ringBuffer.write(input);
    // Pass-through to keep the node alive (output must be connected)
    const output = e.outputBuffer.getChannelData(0);
    output.set(input);
  };

  // Chain: source -> highpass -> lowpass -> analyser
  //                                     +-> processor -> ctx.destination (silent pass-through)
  source.connect(highpass);
  highpass.connect(lowpass);
  lowpass.connect(analyser);
  lowpass.connect(processor);
  // Connect processor to destination to keep it alive (ScriptProcessor quirk).
  // The pass-through audio is at the same level, so we need a silent gain node.
  const silentGain = ctx.createGain();
  silentGain.gain.value = 0;
  processor.connect(silentGain);
  silentGain.connect(ctx.destination);

  return { analyser, nodes: [highpass, lowpass, analyser, processor, silentGain] };
}
