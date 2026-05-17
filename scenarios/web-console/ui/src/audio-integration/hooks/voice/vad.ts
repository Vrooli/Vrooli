// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// Voice Activity Detection (VAD) — pure functions with no external dependencies.
// All state is passed explicitly via VadRefs, making the state machine fully testable.

export type VadState = "idle" | "calibrating" | "waitingForSpeech" | "speechDetected" | "watchingSilence";

/** Action returned by vadTick. "segment-boundary" fires when silence exceeds
 *  segmentSilenceMs but hasn't yet reached the full stop threshold. */
export type VadAction = "stop" | "no-speech" | "segment-boundary";

export const VAD_CALIBRATION_MS = 500;
/**
 * Fallback silence timeout (ms). Used only for the initial paint before
 * the workspace store has been hydrated from audio-tools' stt_stream_config
 * (`vad_silence_ms`). At runtime, the hydrated `vadSilenceTimeoutMs`
 * always wins — see useHydrateVoiceConfig in the host scenario.
 */
export const VAD_FALLBACK_SILENCE_TIMEOUT_MS = 2000;
/**
 * Fallback segment-boundary silence (ms) for persistent mode. Same role
 * as VAD_FALLBACK_SILENCE_TIMEOUT_MS — overridden by hydrated server config.
 */
export const VAD_FALLBACK_SEGMENT_SILENCE_MS = 1500;
export const VAD_NO_SPEECH_TIMEOUT_MS = 15_000;
export const VAD_MIN_SILENCE_THRESHOLD = 0.02;
export const VAD_MIN_SPEECH_THRESHOLD = 0.06;
export const VAD_SLIDING_WINDOW_SIZE = 30;     // ~2s at 15Hz
export const VAD_NOISE_FLOOR_DECAY_RATE = 0.5; // max floor decrease per second

// ── Noise Floor Cache ──
// DOC: docs/internal/VOICE-LATENCY.md#persistent-noise-floor-cache
//
// Persisting the noise floor across sessions lets subsequent recordings skip
// the 500ms calibration phase entirely. The sliding window adaptation still
// runs and will correct the thresholds if the environment has changed.

/** Maximum age of a cached noise floor before it is considered stale. */
export const VAD_FLOOR_CACHE_MAX_AGE_MS = 86_400_000; // 24 hours

/** If the live noise floor diverges from the cached floor by more than this
 *  factor within the first 500ms, reset thresholds from live data. */
export const VAD_FLOOR_DRIFT_FACTOR = 3;

const NOISE_FLOOR_CACHE_KEY = "wc-noise-floor-cache";

/** Serializable snapshot of the VAD noise floor for localStorage persistence. */
export interface CachedNoiseFloor {
  silenceThreshold: number;
  speechThreshold: number;
  timestamp: number;
}

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
  /** When true, disables the no-speech timeout (passive mode waits indefinitely). */
  passiveMode: boolean;
  /** When set, the VAD was initialized from a cached floor. The drift guard
   *  in vadTick compares live noise floor against this value during the first
   *  VAD_CALIBRATION_MS to detect environment changes. Cleared after check. */
  cachedFloorBaseline: number;
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
    passiveMode: false,
    cachedFloorBaseline: 0,
  };
}

/** Create VAD refs configured for passive wake word listening.
 *  Disables no-speech timeout and uses shorter segment silence for quick detection. */
export const VAD_PASSIVE_SEGMENT_SILENCE_MS = 800;

export function createPassiveVadRefs(): VadRefs {
  return {
    ...createVadRefs(),
    segmentSilenceMs: VAD_PASSIVE_SEGMENT_SILENCE_MS,
    passiveMode: true,
  };
}

// ── Noise Floor Cache Helpers ──
// DOC: docs/internal/VOICE-LATENCY.md#persistent-noise-floor-cache

/** Load a previously saved noise floor from localStorage. Returns null if
 *  no cache exists, or the cache is malformed. Does NOT check age — callers
 *  should compare `cached.timestamp` against `VAD_FLOOR_CACHE_MAX_AGE_MS`. */
export function loadNoiseFloorCache(): CachedNoiseFloor | null {
  try {
    const raw = localStorage.getItem(NOISE_FLOOR_CACHE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as CachedNoiseFloor;
    if (
      typeof parsed.silenceThreshold !== "number" ||
      typeof parsed.speechThreshold !== "number" ||
      typeof parsed.timestamp !== "number"
    ) {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

/** Persist the current noise floor to localStorage for use in future sessions. */
export function saveNoiseFloorCache(floor: CachedNoiseFloor): void {
  try {
    localStorage.setItem(NOISE_FLOOR_CACHE_KEY, JSON.stringify(floor));
  } catch {
    // localStorage full or unavailable — silently skip.
  }
}

/** Extract the current thresholds from a VadRefs instance for persistence. */
export function extractCacheableFloor(vad: VadRefs): CachedNoiseFloor {
  return {
    silenceThreshold: vad.silenceThreshold,
    speechThreshold: vad.speechThreshold,
    timestamp: Date.now(),
  };
}

/**
 * Create VAD refs pre-seeded with cached thresholds, skipping the 500ms
 * calibration phase. The VAD starts directly in "waitingForSpeech" state.
 *
 * The sliding window noise floor adaptation still runs, so if the environment
 * has changed since the cache was written, thresholds will self-correct within
 * a few seconds.
 *
 * A drift guard is built into vadTick: if the live noise floor diverges from
 * the cached floor by more than VAD_FLOOR_DRIFT_FACTOR within the first
 * VAD_CALIBRATION_MS, thresholds are reset from live data immediately.
 *
 * DOC: docs/internal/VOICE-LATENCY.md#persistent-noise-floor-cache
 */
export function createVadRefsFromCache(cached: CachedNoiseFloor): VadRefs {
  return {
    ...createVadRefs(),
    state: "waitingForSpeech",
    silenceThreshold: cached.silenceThreshold,
    speechThreshold: cached.speechThreshold,
    // Store the implied noise floor as baseline for the drift guard.
    // vadTick compares live sliding-window floor against this during
    // the first VAD_CALIBRATION_MS to detect environment changes.
    cachedFloorBaseline: cached.silenceThreshold / 1.5,
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
export function vadTick(vad: VadRefs, rms: number, now: number, silenceTimeoutMs: number = VAD_FALLBACK_SILENCE_TIMEOUT_MS): VadAction | null {
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

    // ── Drift guard for cached floor ──
    // DOC: docs/internal/VOICE-LATENCY.md#persistent-noise-floor-cache
    //
    // When the VAD was initialized from a cached noise floor (cachedFloorBaseline > 0),
    // compare the live floor against the cached baseline during the first
    // VAD_CALIBRATION_MS. If the live floor diverges by more than VAD_FLOOR_DRIFT_FACTOR,
    // the environment has changed significantly (e.g., user moved from quiet room to
    // coffee shop). Reset thresholds from live data immediately — no pause needed.
    if (vad.cachedFloorBaseline > 0 && now - vad.recordingStart <= VAD_CALIBRATION_MS) {
      const ratio = newFloor / vad.cachedFloorBaseline;
      if (ratio > VAD_FLOOR_DRIFT_FACTOR || (vad.cachedFloorBaseline > 0 && ratio < 1 / VAD_FLOOR_DRIFT_FACTOR)) {
        console.info(
          "[voice] Noise floor drift detected (cached=%.4f, live=%.4f, ratio=%.1f), resetting from live data",
          vad.cachedFloorBaseline, newFloor, ratio,
        );
        vad.cachedFloorBaseline = 0; // Clear so this only fires once
      }
    }
    // Clear the drift guard after the calibration window regardless
    if (vad.cachedFloorBaseline > 0 && now - vad.recordingStart > VAD_CALIBRATION_MS) {
      vad.cachedFloorBaseline = 0;
    }
  }

  if (vad.state === "waitingForSpeech") {
    if (rms > vad.speechThreshold) {
      vad.state = "speechDetected";
      return null;
    }
    if (!vad.passiveMode && now - vad.recordingStart > VAD_NO_SPEECH_TIMEOUT_MS) {
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

  // vad.state === "watchingSilence" (only remaining branch in the state machine)
  {
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
