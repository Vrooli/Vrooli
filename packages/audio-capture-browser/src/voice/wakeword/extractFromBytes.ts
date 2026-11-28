// DOC: docs/internal/SEAMS.md#wake-word-engine-seam
//
// Shared decode→extract path for the wake-word feature. The template persists
// RAW audio (proto WakeWordSample.audio); MFCC features are never persisted —
// they are re-derived from that audio both at enrollment and on load. Keeping a
// single helper guarantees the features produced on load are byte-for-byte
// identical to those produced while recording, and means an engine upgrade
// (mfcc-v1 → embedding-v1) needs no re-enrollment.

import type { AudioFeatures, WakeWordEngine } from "./types";

/** Sample rate features are extracted at — must match the streaming path. */
export const MFCC_SAMPLE_RATE = 16000;

/**
 * Decode encoded audio bytes (webm/opus/wav/…) to mono PCM at {@link MFCC_SAMPLE_RATE}
 * and extract features via the engine. Mirrors the record-time decode exactly.
 *
 * `decodeAudioData` sniffs the container/codec from the byte content, so no
 * explicit mime is required here.
 */
export async function bytesToFeatures(
  bytes: Uint8Array,
  engine: WakeWordEngine,
): Promise<AudioFeatures> {
  // decodeAudioData needs a standalone ArrayBuffer; a Uint8Array view may be a
  // subarray that does not start at byte 0, so copy the exact range.
  const buf = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer;
  const audioCtx = new AudioContext({ sampleRate: MFCC_SAMPLE_RATE });
  try {
    const decoded = await audioCtx.decodeAudioData(buf);
    const pcm = decoded.getChannelData(0);
    return engine.extractFeatures(pcm, MFCC_SAMPLE_RATE);
  } finally {
    await audioCtx.close();
  }
}
