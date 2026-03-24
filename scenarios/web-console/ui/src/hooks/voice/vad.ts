// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// Voice Activity Detection (VAD) — pure functions with no external dependencies.
// All state is passed explicitly via VadRefs, making the state machine fully testable.

export type VadState = "idle" | "calibrating" | "waitingForSpeech" | "speechDetected" | "watchingSilence";

/** Action returned by vadTick. "segment-boundary" fires when silence exceeds
 *  segmentSilenceMs but hasn't yet reached the full stop threshold. */
export type VadAction = "stop" | "no-speech" | "segment-boundary";

export const VAD_CALIBRATION_MS = 500;
/** Default silence timeout (ms). Configurable via workspace store `vadSilenceTimeoutMs`. */
export const VAD_DEFAULT_SILENCE_TIMEOUT_MS = 2000;
/** Default silence duration (ms) that triggers a segment boundary in persistent mode. */
export const VAD_DEFAULT_SEGMENT_SILENCE_MS = 1500;
export const VAD_NO_SPEECH_TIMEOUT_MS = 15_000;
export const VAD_MIN_SILENCE_THRESHOLD = 0.02;
export const VAD_MIN_SPEECH_THRESHOLD = 0.06;
export const VAD_SLIDING_WINDOW_SIZE = 30;     // ~2s at 15Hz
export const VAD_NOISE_FLOOR_DECAY_RATE = 0.5; // max floor decrease per second

export interface VadRefs {
  state: VadState;
  recordingStart: number;
  silenceStart: number;
  noiseFloorSamples: number[];
  speechThreshold: number;
  silenceThreshold: number;
  slidingWindow: number[];
  slidingWindowIdx: number;
  lastFloorUpdateTime: number;
  /** When set, enables segment-boundary detection. Silence ≥ this threshold
   *  emits "segment-boundary" before the full stop timeout fires. */
  segmentSilenceMs: number;
  /** Whether segment-boundary was already emitted for the current silence gap.
   *  Reset when speech resumes. */
  segmentBoundaryEmitted: boolean;
}

export function createVadRefs(): VadRefs {
  return {
    state: "idle",
    recordingStart: 0,
    silenceStart: 0,
    noiseFloorSamples: [],
    speechThreshold: VAD_MIN_SPEECH_THRESHOLD,
    silenceThreshold: VAD_MIN_SILENCE_THRESHOLD,
    slidingWindow: [],
    slidingWindowIdx: 0,
    lastFloorUpdateTime: 0,
    segmentSilenceMs: 0,
    segmentBoundaryEmitted: false,
  };
}

/**
 * Compute an updated noise floor from a sliding window of RMS samples.
 * Uses the 25th percentile to ignore speech spikes. Rising noise is adopted
 * immediately; falling noise decays at most `decayRate` per second (hysteresis).
 */
export function computeSlidingNoiseFloor(
  samples: number[],
  currentFloor: number,
  elapsedSec: number,
  decayRate: number,
): number {
  if (samples.length === 0) return currentFloor;
  const sorted = [...samples].sort((a, b) => a - b);
  const pctIdx = Math.floor(sorted.length * 0.25);
  const percentile = sorted[pctIdx] ?? currentFloor;

  if (percentile >= currentFloor) {
    // Noise rose -- adopt immediately
    return percentile;
  }
  // Noise fell -- decay gradually
  const maxDrop = decayRate * elapsedSec;
  return Math.max(percentile, currentFloor - maxDrop);
}

/**
 * Run one VAD tick. Returns a VadAction if an event occurred, or null to continue.
 *
 * - `"stop"`: silence exceeded the full stop timeout — recording should end.
 * - `"segment-boundary"`: silence exceeded `segmentSilenceMs` — a segment
 *   boundary was detected (only when `vad.segmentSilenceMs > 0`).
 * - `"no-speech"`: no speech detected within the no-speech timeout.
 *
 * Pure function -- all inputs are explicit parameters with no external dependencies.
 */
export function vadTick(vad: VadRefs, rms: number, now: number, silenceTimeoutMs: number = VAD_DEFAULT_SILENCE_TIMEOUT_MS): VadAction | null {
  if (vad.state === "idle") return null;

  if (vad.state === "calibrating") {
    vad.noiseFloorSamples.push(rms);
    if (now - vad.recordingStart >= VAD_CALIBRATION_MS) {
      // Compute adaptive thresholds from noise floor
      const avg = vad.noiseFloorSamples.reduce((a, b) => a + b, 0) / (vad.noiseFloorSamples.length || 1);
      vad.silenceThreshold = Math.max(VAD_MIN_SILENCE_THRESHOLD, avg * 1.5);
      vad.speechThreshold = Math.max(VAD_MIN_SPEECH_THRESHOLD, avg * 3);
      vad.state = "waitingForSpeech";
    }
    return null;
  }

  // --- Sliding window noise floor update ---
  // Only collect samples when NOT in speechDetected state. During speech,
  // high RMS values inflate the 25th percentile, pushing the noise floor
  // (and thus speechThreshold) up to levels that normal speech can't exceed.
  // This caused premature VAD stops during sustained speech.
  if (vad.state !== "speechDetected") {
    if (vad.slidingWindow.length < VAD_SLIDING_WINDOW_SIZE) {
      vad.slidingWindow.push(rms);
    } else {
      vad.slidingWindow[vad.slidingWindowIdx % VAD_SLIDING_WINDOW_SIZE] = rms;
    }
    vad.slidingWindowIdx++;
  }

  // Recompute noise floor when buffer is full
  if (vad.slidingWindow.length >= VAD_SLIDING_WINDOW_SIZE) {
    const elapsed = vad.lastFloorUpdateTime > 0
      ? (now - vad.lastFloorUpdateTime) / 1000
      : 0;
    const currentFloor = vad.silenceThreshold / 1.5; // reverse from threshold
    const newFloor = computeSlidingNoiseFloor(
      vad.slidingWindow,
      currentFloor,
      elapsed,
      VAD_NOISE_FLOOR_DECAY_RATE,
    );
    vad.silenceThreshold = Math.max(VAD_MIN_SILENCE_THRESHOLD, newFloor * 1.5);
    vad.speechThreshold = Math.max(VAD_MIN_SPEECH_THRESHOLD, newFloor * 3);
    vad.lastFloorUpdateTime = now;
  }

  if (vad.state === "waitingForSpeech") {
    if (rms > vad.speechThreshold) {
      vad.state = "speechDetected";
      return null;
    }
    if (now - vad.recordingStart > VAD_NO_SPEECH_TIMEOUT_MS) {
      return "no-speech";
    }
    return null;
  }

  if (vad.state === "speechDetected") {
    if (rms < vad.silenceThreshold) {
      vad.state = "watchingSilence";
      vad.silenceStart = now;
      vad.segmentBoundaryEmitted = false;
    }
    return null;
  }

  if (vad.state === "watchingSilence") {
    if (rms > vad.speechThreshold) {
      vad.state = "speechDetected";
      return null;
    }
    const elapsed = now - vad.silenceStart;
    // Segment boundary fires once per silence gap, before the full stop timeout
    if (vad.segmentSilenceMs > 0 && !vad.segmentBoundaryEmitted && elapsed >= vad.segmentSilenceMs) {
      vad.segmentBoundaryEmitted = true;
      return "segment-boundary";
    }
    if (elapsed >= silenceTimeoutMs) {
      return "stop";
    }
    return null;
  }

  return null;
}
