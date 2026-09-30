import { describe, it, expect, vi } from "vitest";

import {
  createVadRefs,
  createPassiveVadRefs,
  loadNoiseFloorCache,
  saveNoiseFloorCache,
  extractCacheableFloor,
  createVadRefsFromCache,
  computeSlidingNoiseFloor,
  vadTick,
  VAD_CALIBRATION_MS,
  VAD_NO_SPEECH_TIMEOUT_MS,
  VAD_FALLBACK_SILENCE_TIMEOUT_MS,
  VAD_FALLBACK_SEGMENT_SILENCE_MS,
  VAD_MIN_SILENCE_THRESHOLD,
  VAD_MIN_SPEECH_THRESHOLD,
  VAD_SLIDING_WINDOW_SIZE,
  VAD_NOISE_FLOOR_DECAY_RATE,
  VAD_FLOOR_CACHE_MAX_AGE_MS,
  VAD_FLOOR_DRIFT_FACTOR,
  VAD_PASSIVE_SEGMENT_SILENCE_MS,
} from "./vad";
import type { VadRefs, CachedNoiseFloor } from "./vad";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Returns a freshly created VadRefs in "calibrating" state at t=0. */
function makeCalibrating(now = 0): VadRefs {
  const vad = createVadRefs();
  vad.state = "calibrating";
  vad.recordingStart = now;
  return vad;
}

/** Returns a VadRefs in "waitingForSpeech" with low thresholds. */
function makeWaiting(recordingStart = 0): VadRefs {
  const vad = createVadRefs();
  vad.state = "waitingForSpeech";
  vad.recordingStart = recordingStart;
  vad.speechThreshold = 0.06;
  vad.silenceThreshold = 0.02;
  return vad;
}

/** Returns a VadRefs in "speechDetected" state. */
function makeSpeechDetected(): VadRefs {
  const vad = makeWaiting();
  vad.state = "speechDetected";
  return vad;
}

/** Returns a VadRefs in "watchingSilence" state. */
function makeWatchingSilence(silenceStart = 0): VadRefs {
  const vad = makeWaiting();
  vad.state = "watchingSilence";
  vad.silenceStart = silenceStart;
  return vad;
}

/**
 * Push exactly `n` samples directly into the sliding window without calling vadTick.
 * This avoids triggering state transitions while still populating the window buffer.
 */
function fillSlidingWindow(vad: VadRefs, n: number, rms: number): void {
  for (let i = 0; i < n; i++) {
    if (vad.slidingWindow.length < VAD_SLIDING_WINDOW_SIZE) {
      vad.slidingWindow.push(rms);
    } else {
      vad.slidingWindow[vad.slidingWindowIdx % VAD_SLIDING_WINDOW_SIZE] = rms;
    }
    vad.slidingWindowIdx++;
  }
}

// ---------------------------------------------------------------------------
// Exported constants
// ---------------------------------------------------------------------------

describe("VAD constants", () => {
  it("calibration is 500ms", () => expect(VAD_CALIBRATION_MS).toBe(500));
  it("no-speech timeout is 15s", () => expect(VAD_NO_SPEECH_TIMEOUT_MS).toBe(15_000));
  // Parity tripwire: these MUST equal audio-tools internal/stt.DefaultVADSilenceMs
  // (Go const, currently 1200) so the client fallback fires at the same instant
  // the server cuts the segment. The Go TestVADSilenceDefaultsSingleSource guards
  // the same number from the server side; changing one without the other fails a
  // build. See vad.ts for why this matters ("button off but still transcribing").
  it("fallback silence timeout matches server DefaultVADSilenceMs (1200ms)", () => expect(VAD_FALLBACK_SILENCE_TIMEOUT_MS).toBe(1200));
  it("fallback segment silence matches server DefaultVADSilenceMs (1200ms)", () => expect(VAD_FALLBACK_SEGMENT_SILENCE_MS).toBe(1200));
  it("passive segment silence is 800ms", () => expect(VAD_PASSIVE_SEGMENT_SILENCE_MS).toBe(800));
  it("sliding window size is 30", () => expect(VAD_SLIDING_WINDOW_SIZE).toBe(30));
  it("drift factor is 3", () => expect(VAD_FLOOR_DRIFT_FACTOR).toBe(3));
  it("floor cache max age is 24 hours", () => expect(VAD_FLOOR_CACHE_MAX_AGE_MS).toBe(86_400_000));
});

// ---------------------------------------------------------------------------
// createVadRefs / createPassiveVadRefs
// ---------------------------------------------------------------------------

describe("createVadRefs", () => {
  it("starts in idle state", () => {
    const vad = createVadRefs();
    expect(vad.state).toBe("idle");
  });

  it("sets sensible default thresholds", () => {
    const vad = createVadRefs();
    expect(vad.speechThreshold).toBe(VAD_MIN_SPEECH_THRESHOLD);
    expect(vad.silenceThreshold).toBe(VAD_MIN_SILENCE_THRESHOLD);
  });

  it("starts with passiveMode = false", () => {
    expect(createVadRefs().passiveMode).toBe(false);
  });
});

describe("createPassiveVadRefs", () => {
  it("sets passiveMode = true", () => {
    expect(createPassiveVadRefs().passiveMode).toBe(true);
  });

  it("sets segmentSilenceMs to VAD_PASSIVE_SEGMENT_SILENCE_MS", () => {
    expect(createPassiveVadRefs().segmentSilenceMs).toBe(VAD_PASSIVE_SEGMENT_SILENCE_MS);
  });

  it("starts in idle state", () => {
    expect(createPassiveVadRefs().state).toBe("idle");
  });
});

// ---------------------------------------------------------------------------
// loadNoiseFloorCache / saveNoiseFloorCache
// ---------------------------------------------------------------------------

describe("loadNoiseFloorCache", () => {
  it("returns null when localStorage has no entry", () => {
    expect(loadNoiseFloorCache()).toBeNull();
  });

  it("returns the saved floor after saveNoiseFloorCache", () => {
    const floor: CachedNoiseFloor = { silenceThreshold: 0.03, speechThreshold: 0.06, timestamp: 1000 };
    saveNoiseFloorCache(floor);
    const loaded = loadNoiseFloorCache();
    expect(loaded).not.toBeNull();
    expect(loaded!.silenceThreshold).toBe(0.03);
    expect(loaded!.speechThreshold).toBe(0.06);
    expect(loaded!.timestamp).toBe(1000);
  });

  it("returns null for malformed JSON", () => {
    localStorage.setItem("wc-noise-floor-cache", "not-json{");
    expect(loadNoiseFloorCache()).toBeNull();
  });

  it("returns null when cached object is missing required fields", () => {
    localStorage.setItem("wc-noise-floor-cache", JSON.stringify({ silenceThreshold: 0.03 }));
    expect(loadNoiseFloorCache()).toBeNull();
  });

  it("returns null when a required field has the wrong type", () => {
    localStorage.setItem(
      "wc-noise-floor-cache",
      JSON.stringify({ silenceThreshold: "bad", speechThreshold: 0.06, timestamp: 1000 }),
    );
    expect(loadNoiseFloorCache()).toBeNull();
  });
});

describe("saveNoiseFloorCache", () => {
  it("stores the floor in localStorage so it can be reloaded", () => {
    const floor: CachedNoiseFloor = { silenceThreshold: 0.05, speechThreshold: 0.1, timestamp: 9999 };
    saveNoiseFloorCache(floor);
    const raw = localStorage.getItem("wc-noise-floor-cache");
    expect(raw).not.toBeNull();
    const parsed = JSON.parse(raw!) as CachedNoiseFloor;
    expect(parsed.silenceThreshold).toBe(0.05);
  });

  it("overwrites an existing entry", () => {
    saveNoiseFloorCache({ silenceThreshold: 0.01, speechThreshold: 0.02, timestamp: 1 });
    saveNoiseFloorCache({ silenceThreshold: 0.09, speechThreshold: 0.18, timestamp: 2 });
    const loaded = loadNoiseFloorCache();
    expect(loaded!.silenceThreshold).toBe(0.09);
  });

  it("does not throw when localStorage is unavailable", () => {
    const setItemSpy = vi.spyOn(Storage.prototype, "setItem").mockImplementationOnce(() => {
      throw new Error("QuotaExceededError");
    });
    expect(() => saveNoiseFloorCache({ silenceThreshold: 0.03, speechThreshold: 0.06, timestamp: 0 })).not.toThrow();
    setItemSpy.mockRestore();
  });
});

// ---------------------------------------------------------------------------
// extractCacheableFloor
// ---------------------------------------------------------------------------

describe("extractCacheableFloor", () => {
  it("extracts silenceThreshold and speechThreshold from VadRefs", () => {
    const vad = createVadRefs();
    vad.silenceThreshold = 0.04;
    vad.speechThreshold = 0.08;
    const floor = extractCacheableFloor(vad);
    expect(floor.silenceThreshold).toBe(0.04);
    expect(floor.speechThreshold).toBe(0.08);
  });

  it("timestamp is approximately now", () => {
    const before = Date.now();
    const floor = extractCacheableFloor(createVadRefs());
    const after = Date.now();
    expect(floor.timestamp).toBeGreaterThanOrEqual(before);
    expect(floor.timestamp).toBeLessThanOrEqual(after);
  });
});

// ---------------------------------------------------------------------------
// createVadRefsFromCache
// ---------------------------------------------------------------------------

describe("createVadRefsFromCache", () => {
  it("starts in waitingForSpeech state (skips calibration)", () => {
    const cached: CachedNoiseFloor = { silenceThreshold: 0.03, speechThreshold: 0.06, timestamp: 1000 };
    const vad = createVadRefsFromCache(cached);
    expect(vad.state).toBe("waitingForSpeech");
  });

  it("seeds silenceThreshold and speechThreshold from the cache", () => {
    const cached: CachedNoiseFloor = { silenceThreshold: 0.05, speechThreshold: 0.10, timestamp: 1000 };
    const vad = createVadRefsFromCache(cached);
    expect(vad.silenceThreshold).toBe(0.05);
    expect(vad.speechThreshold).toBe(0.10);
  });

  it("sets cachedFloorBaseline = silenceThreshold / 1.5", () => {
    const cached: CachedNoiseFloor = { silenceThreshold: 0.03, speechThreshold: 0.06, timestamp: 1000 };
    const vad = createVadRefsFromCache(cached);
    expect(vad.cachedFloorBaseline).toBeCloseTo(0.03 / 1.5, 10);
  });
});

// ---------------------------------------------------------------------------
// computeSlidingNoiseFloor
// ---------------------------------------------------------------------------

describe("computeSlidingNoiseFloor", () => {
  it("returns currentFloor for empty samples", () => {
    expect(computeSlidingNoiseFloor([], 0.05, 1, 0.5)).toBe(0.05);
  });

  it("adopts a rising noise floor immediately", () => {
    const samples = [0.5, 0.5, 0.5, 0.5]; // all 0.5 > currentFloor 0.05
    const result = computeSlidingNoiseFloor(samples, 0.05, 1, 0.5);
    // 25th percentile of [0.5, 0.5, 0.5, 0.5] = 0.5 → adopted
    expect(result).toBeCloseTo(0.5, 10);
  });

  it("decays a falling noise floor gradually (hysteresis)", () => {
    const samples = [0.01, 0.01, 0.01, 0.01]; // all below currentFloor 0.1
    const elapsedSec = 0.05;
    const maxDrop = VAD_NOISE_FLOOR_DECAY_RATE * elapsedSec; // 0.5 * 0.05 = 0.025
    const result = computeSlidingNoiseFloor(samples, 0.1, elapsedSec, VAD_NOISE_FLOOR_DECAY_RATE);
    // percentile < currentFloor → max(percentile, currentFloor - maxDrop)
    expect(result).toBeCloseTo(0.1 - maxDrop, 10);
  });

  it("never drops below the measured percentile", () => {
    const samples = [0.05, 0.05, 0.05, 0.05];
    // Large elapsed time → large maxDrop, but floor cannot drop below percentile
    const result = computeSlidingNoiseFloor(samples, 0.1, 100, 10);
    expect(result).toBeCloseTo(0.05, 10);
  });

  it("uses the 25th percentile (ignores speech spikes)", () => {
    // 4 samples: sorted = [0.01, 0.02, 0.5, 0.8]; pctIdx = floor(4*0.25) = 1 → 0.02
    const samples = [0.8, 0.01, 0.5, 0.02];
    const result = computeSlidingNoiseFloor(samples, 0.5, 1, 0.5);
    expect(result).toBeCloseTo(0.02, 10);
  });
});

// ---------------------------------------------------------------------------
// vadTick state machine: idle
// ---------------------------------------------------------------------------

describe("vadTick — idle state", () => {
  it("always returns null in idle state", () => {
    const vad = createVadRefs(); // state = "idle"
    expect(vadTick(vad, 0.5, 1000)).toBeNull();
    expect(vad.state).toBe("idle");
  });
});

// ---------------------------------------------------------------------------
// vadTick state machine: calibrating
// ---------------------------------------------------------------------------

describe("vadTick — calibrating state", () => {
  it("collects noise floor samples during calibration", () => {
    const vad = makeCalibrating(0);
    vadTick(vad, 0.01, 100);
    vadTick(vad, 0.02, 200);
    expect(vad.noiseFloorSamples).toEqual([0.01, 0.02]);
    expect(vad.state).toBe("calibrating");
  });

  it("transitions to waitingForSpeech after VAD_CALIBRATION_MS", () => {
    const vad = makeCalibrating(0);
    vadTick(vad, 0.01, 100);
    vadTick(vad, 0.02, 400);
    expect(vad.state).toBe("calibrating"); // not yet
    const result = vadTick(vad, 0.03, VAD_CALIBRATION_MS); // exactly 500ms
    expect(result).toBeNull();
    expect(vad.state).toBe("waitingForSpeech");
  });

  it("sets adaptive thresholds from noise floor", () => {
    const vad = makeCalibrating(0);
    vadTick(vad, 0.02, 100);
    vadTick(vad, 0.04, 200);
    vadTick(vad, 0.06, VAD_CALIBRATION_MS);
    // avg = 0.04; silenceThreshold = max(0.02, 0.04*1.5=0.06) = 0.06
    // speechThreshold = max(0.06, 0.04*3=0.12) = 0.12
    expect(vad.silenceThreshold).toBeCloseTo(0.06, 5);
    expect(vad.speechThreshold).toBeCloseTo(0.12, 5);
  });

  it("respects VAD_MIN thresholds as lower bounds", () => {
    const vad = makeCalibrating(0);
    // Very quiet environment → thresholds should not go below minimums
    vadTick(vad, 0.001, 100);
    vadTick(vad, 0.001, VAD_CALIBRATION_MS);
    expect(vad.silenceThreshold).toBeGreaterThanOrEqual(VAD_MIN_SILENCE_THRESHOLD);
    expect(vad.speechThreshold).toBeGreaterThanOrEqual(VAD_MIN_SPEECH_THRESHOLD);
  });
});

// ---------------------------------------------------------------------------
// vadTick state machine: waitingForSpeech
// ---------------------------------------------------------------------------

describe("vadTick — waitingForSpeech state", () => {
  it("returns null while rms is below speechThreshold", () => {
    const vad = makeWaiting(0);
    const result = vadTick(vad, 0.01, 1000, 2000);
    expect(result).toBeNull();
    expect(vad.state).toBe("waitingForSpeech");
  });

  it("transitions to speechDetected when rms exceeds speechThreshold", () => {
    const vad = makeWaiting(0);
    const result = vadTick(vad, 1.0, 1000, 2000); // rms well above 0.06
    expect(result).toBeNull();
    expect(vad.state).toBe("speechDetected");
  });

  it("returns 'no-speech' after VAD_NO_SPEECH_TIMEOUT_MS in non-passive mode", () => {
    const vad = makeWaiting(0);
    const result = vadTick(vad, 0.01, VAD_NO_SPEECH_TIMEOUT_MS + 1, 2000);
    expect(result).toBe("no-speech");
    expect(vad.state).toBe("waitingForSpeech");
  });

  it("does NOT return 'no-speech' in passive mode regardless of time", () => {
    const vad = makeWaiting(0);
    vad.passiveMode = true;
    const result = vadTick(vad, 0.01, VAD_NO_SPEECH_TIMEOUT_MS * 10, 2000);
    expect(result).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// vadTick state machine: speechDetected
// ---------------------------------------------------------------------------

describe("vadTick — speechDetected state", () => {
  it("returns null while rms is above silenceThreshold", () => {
    const vad = makeSpeechDetected();
    const result = vadTick(vad, 1.0, 2000, 2000); // high rms — still speaking
    expect(result).toBeNull();
    expect(vad.state).toBe("speechDetected");
  });

  it("transitions to watchingSilence when rms drops below silenceThreshold", () => {
    const vad = makeSpeechDetected();
    const now = 2000;
    const result = vadTick(vad, 0.001, now, 2000); // very low rms — silence
    expect(result).toBeNull();
    expect(vad.state).toBe("watchingSilence");
    expect(vad.silenceStart).toBe(now);
    expect(vad.segmentBoundaryEmitted).toBe(false);
  });

  it("does NOT update the sliding window while speaking", () => {
    const vad = makeSpeechDetected();
    const winBefore = vad.slidingWindow.length;
    vadTick(vad, 1.0, 1000, 2000);
    expect(vad.slidingWindow.length).toBe(winBefore);
  });
});

// ---------------------------------------------------------------------------
// vadTick state machine: watchingSilence
// ---------------------------------------------------------------------------

describe("vadTick — watchingSilence state", () => {
  it("returns null while elapsed time is below all thresholds", () => {
    const vad = makeWatchingSilence(0);
    // Low rms, elapsed = 100ms — no action yet
    expect(vadTick(vad, 0.001, 100, 2000)).toBeNull();
    expect(vad.state).toBe("watchingSilence");
  });

  it("returns 'stop' after silence exceeds silenceTimeoutMs", () => {
    const vad = makeWatchingSilence(0);
    const result = vadTick(vad, 0.001, 2000, 2000); // elapsed = 2000 >= 2000
    expect(result).toBe("stop");
  });

  it("transitions back to speechDetected when rms exceeds speechThreshold", () => {
    const vad = makeWatchingSilence(0);
    const result = vadTick(vad, 1.0, 500, 2000); // high rms while in silence
    expect(result).toBeNull();
    expect(vad.state).toBe("speechDetected");
  });

  it("emits 'segment-boundary' once when segmentSilenceMs threshold is crossed", () => {
    const vad = makeWatchingSilence(0);
    vad.segmentSilenceMs = 500;

    // Not enough elapsed yet
    expect(vadTick(vad, 0.001, 400, 2000)).toBeNull();

    // Threshold reached — first emission
    const r2 = vadTick(vad, 0.001, 500, 2000);
    expect(r2).toBe("segment-boundary");
    expect(vad.segmentBoundaryEmitted).toBe(true);

    // Subsequent ticks do NOT re-emit segment-boundary
    const r3 = vadTick(vad, 0.001, 600, 2000);
    expect(r3).toBeNull(); // elapsed 600 < silenceTimeoutMs 2000

    // Eventually reaches full stop
    const r4 = vadTick(vad, 0.001, 2000, 2000);
    expect(r4).toBe("stop");
  });

  it("does NOT emit segment-boundary when segmentSilenceMs is 0", () => {
    const vad = makeWatchingSilence(0);
    vad.segmentSilenceMs = 0;
    expect(vadTick(vad, 0.001, 1000, 2000)).toBeNull();
    expect(vad.state).toBe("watchingSilence");
  });

  it("resets segmentBoundaryEmitted when speech resumes then silence starts again", () => {
    const vad = makeWatchingSilence(0);
    vad.segmentSilenceMs = 200;

    // Emit segment-boundary
    expect(vadTick(vad, 0.001, 200, 2000)).toBe("segment-boundary");
    expect(vad.segmentBoundaryEmitted).toBe(true);

    // Speech resumes — back to speechDetected, segmentBoundaryEmitted reset
    expect(vadTick(vad, 1.0, 300, 2000)).toBeNull();
    expect(vad.state).toBe("speechDetected");

    // Silence again — segmentBoundaryEmitted = false
    expect(vadTick(vad, 0.001, 400, 2000)).toBeNull();
    expect(vad.state).toBe("watchingSilence");
    expect(vad.segmentBoundaryEmitted).toBe(false);

    // Second segment-boundary can fire
    expect(vadTick(vad, 0.001, 600, 2000)).toBe("segment-boundary");
  });
});

// ---------------------------------------------------------------------------
// vadTick: sliding window and noise floor recomputation
// ---------------------------------------------------------------------------

describe("vadTick — sliding window noise floor updates", () => {
  it("recomputes noise floor once the window reaches VAD_SLIDING_WINDOW_SIZE", () => {
    const vad = makeWaiting(0);
    vad.silenceThreshold = 0.02; // start low
    const LOW_RMS = 0.001; // much lower than current floor

    // Fill 29 samples (below full)
    for (let i = 0; i < 29; i++) {
      vadTick(vad, LOW_RMS, 100, 5000);
    }

    // 30th sample — triggers recomputation
    vadTick(vad, LOW_RMS, 200, 5000);
    expect(vad.lastFloorUpdateTime).toBe(200);
    expect(Number.isFinite(vad.silenceThreshold)).toBe(true);
    expect(Number.isFinite(vad.speechThreshold)).toBe(true);
    expect(vad.silenceThreshold).toBeGreaterThanOrEqual(VAD_MIN_SILENCE_THRESHOLD);
  });

  it("uses the elapsed time between floor updates (lastFloorUpdateTime)", () => {
    const vad = makeWaiting(0);
    fillSlidingWindow(vad, VAD_SLIDING_WINDOW_SIZE, 0.01);
    vad.lastFloorUpdateTime = 1000;
    // Trigger recomputation at t=2000 — elapsed = 1s
    vadTick(vad, 0.01, 2000, 5000);
    expect(vad.lastFloorUpdateTime).toBe(2000);
  });

  it("does not update sliding window while in speechDetected state", () => {
    const vad = makeSpeechDetected();
    const before = vad.slidingWindow.length;
    for (let i = 0; i < 5; i++) {
      vadTick(vad, 1.0, i * 100, 5000);
    }
    expect(vad.slidingWindow.length).toBe(before);
  });
});

// ---------------------------------------------------------------------------
// vadTick: drift guard for cached noise floor
// ---------------------------------------------------------------------------

describe("vadTick — drift guard", () => {
  it("clears cachedFloorBaseline when live floor diverges during calibration window", () => {
    const cached: CachedNoiseFloor = { silenceThreshold: 0.03, speechThreshold: 0.06, timestamp: Date.now() };
    const vad = createVadRefsFromCache(cached);
    // cachedFloorBaseline = 0.03 / 1.5 = 0.02
    expect(vad.cachedFloorBaseline).toBeCloseTo(0.02, 5);

    // Pre-fill sliding window with HIGH values (0.5).
    // 25th percentile = 0.5, ratio = 0.5/0.02 = 25 > VAD_FLOOR_DRIFT_FACTOR(3) → drift.
    fillSlidingWindow(vad, VAD_SLIDING_WINDOW_SIZE, 0.5);
    expect(vad.slidingWindow.length).toBe(VAD_SLIDING_WINDOW_SIZE);

    const infoSpy = vi.spyOn(console, "info").mockImplementation(() => {});

    // One tick within calibration window (now=100 < recordingStart+500=500).
    // rms=0.01 < speechThreshold=0.06 so state stays waitingForSpeech.
    vadTick(vad, 0.01, 100, 5000);

    expect(vad.cachedFloorBaseline).toBe(0);
    expect(infoSpy).toHaveBeenCalled();
    infoSpy.mockRestore();
  });

  it("clears cachedFloorBaseline after VAD_CALIBRATION_MS even without drift", () => {
    const cached: CachedNoiseFloor = { silenceThreshold: 0.03, speechThreshold: 0.06, timestamp: Date.now() };
    const vad = createVadRefsFromCache(cached);
    vad.recordingStart = 0;

    // Pre-fill with moderate rms so the live floor matches the baseline (no drift).
    fillSlidingWindow(vad, VAD_SLIDING_WINDOW_SIZE, 0.02);

    // Tick PAST the calibration window (now=601 > 500) triggers the
    // "clear baseline after calibration window" guard, regardless of drift.
    vadTick(vad, 0.01, VAD_CALIBRATION_MS + 101, 5000);

    expect(vad.cachedFloorBaseline).toBe(0);
  });

  it("does not clear cachedFloorBaseline when floor is within drift tolerance", () => {
    const cached: CachedNoiseFloor = { silenceThreshold: 0.03, speechThreshold: 0.06, timestamp: Date.now() };
    const vad = createVadRefsFromCache(cached);
    // cachedFloorBaseline = 0.03/1.5 = 0.02

    // Fill window with rms = 0.02 so 25th percentile = 0.02.
    // ratio = 0.02 / 0.02 = 1 — well within [1/3, 3] tolerance.
    fillSlidingWindow(vad, VAD_SLIDING_WINDOW_SIZE, 0.02);

    // Tick within calibration window (now=100 < 500): no drift, no clearance.
    vadTick(vad, 0.01, 100, 5000);

    // Baseline should still be set (non-zero).
    expect(vad.cachedFloorBaseline).toBeGreaterThan(0);
  });

  it("clears cachedFloorBaseline on DOWNWARD drift (live floor much lower than cached)", () => {
    // Use a higher cached baseline so cachedFloorBaseline = 0.09/1.5 = 0.06.
    const cached: CachedNoiseFloor = { silenceThreshold: 0.09, speechThreshold: 0.18, timestamp: Date.now() };
    const vad = createVadRefsFromCache(cached);
    expect(vad.cachedFloorBaseline).toBeCloseTo(0.06, 5);

    // Fill window with very low rms=0.001.
    // 25th percentile = 0.001 (all values the same).
    fillSlidingWindow(vad, VAD_SLIDING_WINDOW_SIZE, 0.001);

    // Set lastFloorUpdateTime to 1 (non-zero) so elapsed = (100-1)/1000 = 0.099s.
    // maxDrop = 0.5 * 0.099 ≈ 0.0495; newFloor = max(0.001, 0.06-0.0495) = 0.0105.
    // ratio = 0.0105/0.06 ≈ 0.175 < 1/VAD_FLOOR_DRIFT_FACTOR(1/3≈0.333) → downward drift.
    vad.lastFloorUpdateTime = 1;

    const infoSpy = vi.spyOn(console, "info").mockImplementation(() => {});

    // Tick within calibration window (now=100 < 500).
    // rms=0.001 < speechThreshold=0.18, so state stays in waitingForSpeech.
    vadTick(vad, 0.001, 100, 5000);

    // Downward drift detected: baseline cleared.
    expect(vad.cachedFloorBaseline).toBe(0);
    expect(infoSpy).toHaveBeenCalled();
    infoSpy.mockRestore();
  });
});

// ---------------------------------------------------------------------------
// vadTick: full lifecycle round-trip
// ---------------------------------------------------------------------------

describe("vadTick — full lifecycle", () => {
  it("drives calibrating → waitingForSpeech → speechDetected → watchingSilence → stop", () => {
    const vad = createVadRefs();
    vad.state = "calibrating";
    vad.recordingStart = 0;

    // Calibration phase
    for (let t = 100; t <= VAD_CALIBRATION_MS; t += 100) {
      const r = vadTick(vad, 0.02, t, 2000);
      expect(r).toBeNull();
    }
    expect(vad.state).toBe("waitingForSpeech");

    // Speech detected
    const r2 = vadTick(vad, 1.0, 600, 2000);
    expect(r2).toBeNull();
    expect(vad.state).toBe("speechDetected");

    // Silence begins
    const r3 = vadTick(vad, 0.001, 700, 2000);
    expect(r3).toBeNull();
    expect(vad.state).toBe("watchingSilence");

    // Silence accumulates to stop (silenceStart=700, elapsed=2000 at t=2700)
    const r4 = vadTick(vad, 0.001, 2700, 2000);
    expect(r4).toBe("stop");
  });
});
