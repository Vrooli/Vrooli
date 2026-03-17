// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// Voice Activity Detection (VAD) — pure functions with no external dependencies.
// All state is passed explicitly via VadRefs, making the state machine fully testable.

export type VadState = "idle" | "calibrating" | "waitingForSpeech" | "speechDetected" | "watchingSilence";

export const VAD_CALIBRATION_MS = 500;
/** Default silence timeout (ms). Configurable via workspace store `vadSilenceTimeoutMs`. */
export const VAD_DEFAULT_SILENCE_TIMEOUT_MS = 2000;
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
 * Run one VAD tick. Returns "stop" if recording should auto-stop,
 * "no-speech" if the no-speech timeout expired, or null to continue.
 *
 * Pure function -- all inputs are explicit parameters with no external dependencies.
 */
export function vadTick(vad: VadRefs, rms: number, now: number, silenceTimeoutMs: number = VAD_DEFAULT_SILENCE_TIMEOUT_MS): "stop" | "no-speech" | null {
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

  // --- Sliding window noise floor update (active in all post-calibration states) ---
  // Push RMS into circular buffer
  if (vad.slidingWindow.length < VAD_SLIDING_WINDOW_SIZE) {
    vad.slidingWindow.push(rms);
  } else {
    vad.slidingWindow[vad.slidingWindowIdx % VAD_SLIDING_WINDOW_SIZE] = rms;
  }
  vad.slidingWindowIdx++;

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
    }
    return null;
  }

  if (vad.state === "watchingSilence") {
    if (rms > vad.speechThreshold) {
      vad.state = "speechDetected";
      return null;
    }
    if (now - vad.silenceStart >= silenceTimeoutMs) {
      return "stop";
    }
    return null;
  }

  return null;
}
