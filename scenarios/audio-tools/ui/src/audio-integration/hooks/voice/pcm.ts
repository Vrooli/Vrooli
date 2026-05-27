// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// Pure PCM conversion helpers for the streaming voice path. The embed
// captures Float32 PCM at the AudioContext's native rate, downsamples to
// the canonical STT rate, and converts to signed 16-bit little-endian PCM
// — the exact representation the audio-tools `pcm_s16le` fast-path expects
// (no server-side ffmpeg). These functions are deliberately
// transport-free and DOM-free so they unit-test without a browser.

import { downsample } from "./audioUtils";

/**
 * Canonical STT capture rate. The audio-tools substrate's PCM fast-path is
 * an identity decoder: declaring `pcm_s16le` means the bytes are already
 * 16 kHz mono s16le, so the client must downsample to exactly this rate.
 */
export const TARGET_SAMPLE_RATE = 16_000;

/**
 * Convert normalized Float32 samples (range [-1, 1]) to signed 16-bit PCM.
 * Out-of-range samples are clamped. Uses the asymmetric int16 scale
 * (0x7fff for positive, 0x8000 for negative) so full-scale audio maps to
 * the full int16 range without overflow.
 */
export function floatTo16BitPCM(input: Float32Array): Int16Array {
  const out = new Int16Array(input.length);
  for (let i = 0; i < input.length; i++) {
    let s = input[i] ?? 0;
    if (s > 1) s = 1;
    else if (s < -1) s = -1;
    out[i] = s < 0 ? Math.round(s * 0x8000) : Math.round(s * 0x7fff);
  }
  return out;
}

/**
 * Downsample a Float32 frame from its capture rate to the canonical STT
 * rate and return signed 16-bit PCM. When the capture rate already equals
 * the target, no resampling occurs. Upsampling is rejected by `downsample`.
 */
export function frameToCanonicalPcm16(frame: Float32Array, captureRate: number): Int16Array {
  const resampled = captureRate === TARGET_SAMPLE_RATE ? frame : downsample(frame, captureRate, TARGET_SAMPLE_RATE);
  return floatTo16BitPCM(resampled);
}

/** Concatenate Int16 PCM chunks into one contiguous buffer. */
export function concatInt16(chunks: Int16Array[]): Int16Array {
  let total = 0;
  for (const c of chunks) total += c.length;
  const out = new Int16Array(total);
  let off = 0;
  for (const c of chunks) {
    out.set(c, off);
    off += c.length;
  }
  return out;
}

function writeAscii(view: DataView, offset: number, s: string): void {
  for (let i = 0; i < s.length; i++) view.setUint8(offset + i, s.charCodeAt(i));
}

/**
 * Build a canonical mono s16le WAV (44-byte RIFF header + samples) as a
 * raw ArrayBuffer. Pure and DOM-free so it unit-tests without a Blob.
 */
export function pcm16ToWavBuffer(pcm: Int16Array, sampleRate: number): ArrayBuffer {
  const dataLen = pcm.length * 2;
  const buf = new ArrayBuffer(44 + dataLen);
  const view = new DataView(buf);
  const byteRate = sampleRate * 2; // mono * 2 bytes/sample
  writeAscii(view, 0, "RIFF");
  view.setUint32(4, 36 + dataLen, true);
  writeAscii(view, 8, "WAVE");
  writeAscii(view, 12, "fmt ");
  view.setUint32(16, 16, true); // PCM fmt chunk size
  view.setUint16(20, 1, true); // audio format: PCM
  view.setUint16(22, 1, true); // channels: mono
  view.setUint32(24, sampleRate, true);
  view.setUint32(28, byteRate, true);
  view.setUint16(32, 2, true); // block align (mono * 2 bytes)
  view.setUint16(34, 16, true); // bits per sample
  writeAscii(view, 36, "data");
  view.setUint32(40, dataLen, true);
  let off = 44;
  for (let i = 0; i < pcm.length; i++, off += 2) {
    view.setInt16(off, pcm[i] ?? 0, true);
  }
  return buf;
}

/**
 * Wrap signed 16-bit mono PCM in a canonical WAV container as a Blob.
 * Used for the HTTP batch-transcription fallback and the speaker-rejection
 * retry path, which post a whole-file container rather than a live stream.
 * The Blob's type is `audio/wav` so the API's format sniffing (blobFormat)
 * maps it to WAV.
 */
export function encodeWavFromPcm16(pcm: Int16Array, sampleRate: number): Blob {
  return new Blob([pcm16ToWavBuffer(pcm, sampleRate)], { type: "audio/wav" });
}
