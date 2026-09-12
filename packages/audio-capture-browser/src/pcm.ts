/** Canonical capture format shared by browser dictation transports. */
export const TARGET_SAMPLE_RATE = 16_000;

/** Convert normalized Float32 samples to signed 16-bit PCM. */
export function floatTo16BitPCM(input: Float32Array): Int16Array {
  const out = new Int16Array(input.length);
  for (let i = 0; i < input.length; i++) {
    let sample = input[i] ?? 0;
    if (sample > 1) sample = 1;
    else if (sample < -1) sample = -1;
    out[i] = sample < 0 ? Math.round(sample * 0x8000) : Math.round(sample * 0x7fff);
  }
  return out;
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
