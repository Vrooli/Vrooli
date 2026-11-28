/** Canonical capture format shared by browser dictation transports. */
export const TARGET_SAMPLE_RATE = 16_000;

/** Convert normalized Float32 samples to signed 16-bit PCM. */
export function floatTo16BitPCM(input: Float32Array): Int16Array {
  const out = new Int16Array(input.length);
  for (let i = 0; i < input.length; i++) {
    out[i] = floatSampleTo16Bit(input[i] ?? 0);
  }
  return out;
}

function floatSampleTo16Bit(value: number): number {
  let sample = value;
  if (sample > 1) sample = 1;
  else if (sample < -1) sample = -1;
  return sample < 0 ? Math.round(sample * 0x8000) : Math.round(sample * 0x7fff);
}

/** Linear downsampling only; callers must not silently upsample capture. */
export function downsample(frame: Float32Array, fromRate: number, toRate: number): Float32Array {
  if (fromRate === toRate) return frame;
  if (toRate > fromRate) throw new Error(`Cannot upsample: ${fromRate} -> ${toRate}`);
  const ratio = fromRate / toRate;
  const output = new Float32Array(Math.ceil(frame.length / ratio));
  for (let i = 0; i < output.length; i++) {
    const source = i * ratio;
    const floor = Math.floor(source);
    const ceil = Math.min(floor + 1, frame.length - 1);
    const fraction = source - floor;
    output[i] = (frame[floor] ?? 0) * (1 - fraction) + (frame[ceil] ?? 0) * fraction;
  }
  return output;
}

/** Normalize a browser capture frame to the wire's 16 kHz mono s16le format. */
export function frameToCanonicalPcm16(frame: Float32Array, captureRate: number): Int16Array {
  return floatTo16BitPCM(captureRate === TARGET_SAMPLE_RATE ? frame : downsample(frame, captureRate, TARGET_SAMPLE_RATE));
}

/**
 * Stateful linear downsampler for browser callback frames. Audio callbacks do
 * not generally end on a resampling phase boundary; resetting `downsample()`
 * for every callback duplicates/omits samples and introduces discontinuities
 * at each boundary. The stream transport owns one instance per turn so the
 * wire remains a continuous canonical PCM sequence.
 */
export class StreamingPcm16Resampler {
  private captureRate = TARGET_SAMPLE_RATE;
  private ratio = 1;
  private pending = new Float32Array(0);
  private position = 0;

  reset(): void {
    this.captureRate = TARGET_SAMPLE_RATE;
    this.ratio = 1;
    this.pending = new Float32Array(0);
    this.position = 0;
  }

  process(frame: Float32Array, captureRate: number): Int16Array {
    if (!Number.isFinite(captureRate) || captureRate <= 0 || captureRate < TARGET_SAMPLE_RATE) {
      throw new Error(`Unsupported capture sample rate: ${captureRate}`);
    }
    if (captureRate === TARGET_SAMPLE_RATE) {
      const tail = this.flushPendingTail();
      this.reset();
      const canonical = floatTo16BitPCM(frame);
      return concatInt16([tail, canonical]);
    }
    if (captureRate !== this.captureRate) {
      const tail = this.flushPendingTail();
      this.reset();
      this.captureRate = captureRate;
      this.ratio = captureRate / TARGET_SAMPLE_RATE;
      const output = this.processResampledFrame(frame);
      return concatInt16([tail, output]);
    }

    return this.processResampledFrame(frame);
  }

  /**
   * Finish the old rate's pending interval on a rate boundary. We use bounded
   * zero-order hold: emit canonical grid points strictly before the end of the
   * captured interval, holding the final captured value when interpolation
   * would otherwise need a sample from the next (different-rate) stream. This
   * emits no point beyond the old interval's boundary and avoids dropping its
   * final fraction of a sample period.
   */
  private flushPendingTail(): Int16Array {
    if (this.pending.length === 0) return new Int16Array(0);
    const output: number[] = [];
    while (this.position < this.pending.length) {
      const floor = Math.floor(this.position);
      const fraction = this.position - floor;
      const left = this.pending[floor] ?? this.pending[this.pending.length - 1]!;
      const right = this.pending[floor + 1] ?? left;
      output.push(floatSampleTo16Bit(Math.fround(left * (1 - fraction) + right * fraction)));
      this.position += this.ratio;
    }
    this.pending = new Float32Array(0);
    this.position = 0;
    return Int16Array.from(output);
  }

  private processResampledFrame(frame: Float32Array): Int16Array {

    const pendingLength = this.pending.length;
    const sourceLength = pendingLength + frame.length;
    const sampleAt = (index: number) => index < pendingLength ? this.pending[index]! : frame[index - pendingLength]!;

    // Count first so callback output has an exact-sized buffer. This avoids
    // growing a number[] and then copying it through Float32Array.from().
    let outputLength = 0;
    let position = this.position;
    while (position + 1 < sourceLength) {
      outputLength++;
      position += this.ratio;
    }

    const output = new Int16Array(outputLength);
    let outputIndex = 0;
    while (this.position + 1 < sourceLength) {
      const floor = Math.floor(this.position);
      const fraction = this.position - floor;
      // Float32Array.from() in the previous implementation rounded each
      // interpolation to float32 before PCM quantization. Keep that boundary.
      const interpolated = Math.fround(sampleAt(floor) * (1 - fraction) + sampleAt(floor + 1) * fraction);
      output[outputIndex++] = floatSampleTo16Bit(interpolated);
      this.position += this.ratio;
    }
    // Keep the final source sample used by the next interpolation. The raw
    // phase can advance beyond the current callback after emitting its last
    // complete pair, but the next callback must begin from that retained
    // endpoint rather than from a reset phase.
    const consumed = Math.min(Math.floor(this.position), Math.max(0, sourceLength - 1));
    if (consumed > 0) {
      const remainingLength = sourceLength - consumed;
      const remaining = new Float32Array(remainingLength);
      for (let index = 0; index < remainingLength; index++) remaining[index] = sampleAt(consumed + index);
      this.pending = remaining;
      this.position -= consumed;
    } else {
      const remaining = new Float32Array(sourceLength);
      remaining.set(this.pending);
      remaining.set(frame, pendingLength);
      this.pending = remaining;
    }
    return output;
  }
}

export function concatInt16(chunks: Int16Array[]): Int16Array {
  let length = 0;
  for (const chunk of chunks) length += chunk.length;
  const output = new Int16Array(length);
  let offset = 0;
  for (const chunk of chunks) {
    output.set(chunk, offset);
    offset += chunk.length;
  }
  return output;
}

function writeASCII(view: DataView, offset: number, text: string): void {
  for (let i = 0; i < text.length; i++) view.setUint8(offset + i, text.charCodeAt(i));
}

/** Build a standard mono s16le WAV container for declared unary recovery. */
export function pcm16ToWavBuffer(pcm: Int16Array, sampleRate: number): ArrayBuffer {
  const dataLength = pcm.length * 2;
  const output = new ArrayBuffer(44 + dataLength);
  const view = new DataView(output);
  writeASCII(view, 0, "RIFF");
  view.setUint32(4, 36 + dataLength, true);
  writeASCII(view, 8, "WAVE");
  writeASCII(view, 12, "fmt ");
  view.setUint32(16, 16, true);
  view.setUint16(20, 1, true);
  view.setUint16(22, 1, true);
  view.setUint32(24, sampleRate, true);
  view.setUint32(28, sampleRate * 2, true);
  view.setUint16(32, 2, true);
  view.setUint16(34, 16, true);
  writeASCII(view, 36, "data");
  view.setUint32(40, dataLength, true);
  let offset = 44;
  for (let i = 0; i < pcm.length; i++, offset += 2) view.setInt16(offset, pcm[i] ?? 0, true);
  return output;
}

export function encodeWavFromPcm16(pcm: Int16Array, sampleRate: number): Blob {
  return new Blob([pcm16ToWavBuffer(pcm, sampleRate)], { type: "audio/wav" });
}
