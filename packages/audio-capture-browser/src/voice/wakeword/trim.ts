// DOC: docs/internal/SEAMS.md#wake-word-engine-seam
//
// Endpoint silence trimming. Enrollment records the whole button-hold (leading
// + trailing silence) while the passive path captures from VAD onset; mismatched
// silence padding inflates the DTW path and the (n+m) normalization divisor,
// destabilizing scores. Trimming uniformly inside extractFeatures (every
// consumer goes through it) removes that mismatch with a single seam.

/** Frame size (ms) for the RMS energy gate. */
const TRIM_FRAME_MS = 10;
/** Head/tail margin (ms) preserved around detected speech to keep onsets/plosives. */
const TRIM_MARGIN_MS = 50;
/** Active-frame threshold = noise floor × this multiplier. */
const TRIM_FLOOR_MULTIPLIER = 3;
/** Absolute minimum RMS for an "active" frame — guards near-silent clips. */
const TRIM_ABS_MIN_RMS = 1e-4;
/** Below this many frames the clip is too short to trim meaningfully. */
const TRIM_MIN_FRAMES = 3;

/**
 * Trim leading/trailing silence from a mono PCM clip using a per-clip relative
 * noise-floor gate. Returns a copy spanning the first→last active frame plus a
 * small margin. If no frame exceeds the gate (the whole clip reads as silence),
 * the input is returned unchanged so downstream feature extraction never gets an
 * empty buffer.
 */
export function trimSilence(pcm: Float32Array, sampleRate: number): Float32Array {
  if (pcm.length === 0) return pcm;

  const frameLen = Math.max(1, Math.floor((TRIM_FRAME_MS / 1000) * sampleRate));
  const numFrames = Math.floor(pcm.length / frameLen);
  if (numFrames < TRIM_MIN_FRAMES) return pcm;

  const rms = new Float64Array(numFrames);
  for (let f = 0; f < numFrames; f++) {
    const start = f * frameLen;
    let sum = 0;
    for (let i = 0; i < frameLen; i++) {
      const v = pcm[start + i] ?? 0;
      sum += v * v;
    }
    rms[f] = Math.sqrt(sum / frameLen);
  }

  // Noise floor = low percentile of frame energies (robust to a few loud frames).
  const sorted = Array.from(rms).sort((a, b) => a - b);
  const floor = sorted[Math.floor(sorted.length * 0.1)] ?? 0;
  const threshold = Math.max(floor * TRIM_FLOOR_MULTIPLIER, TRIM_ABS_MIN_RMS);

  let firstActive = -1;
  let lastActive = -1;
  for (let f = 0; f < numFrames; f++) {
    if ((rms[f] ?? 0) >= threshold) {
      if (firstActive < 0) firstActive = f;
      lastActive = f;
    }
  }

  // Whole clip below the gate — cannot separate speech from silence; leave as-is
  // to avoid returning empty features.
  if (firstActive < 0) return pcm;

  const marginFrames = Math.ceil(TRIM_MARGIN_MS / TRIM_FRAME_MS);
  const startFrame = Math.max(0, firstActive - marginFrames);
  const endFrame = Math.min(numFrames - 1, lastActive + marginFrames);

  const startSample = startFrame * frameLen;
  const endSample = Math.min(pcm.length, (endFrame + 1) * frameLen);
  // Nothing to trim — avoid a needless copy.
  if (startSample <= 0 && endSample >= pcm.length) return pcm;

  return pcm.slice(startSample, endSample);
}
