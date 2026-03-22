import { describe, it, expect } from "vitest";
import {
  createVadRefs,
  vadTick,
  computeSlidingNoiseFloor,
  VAD_DEFAULT_SILENCE_TIMEOUT_MS,
  VAD_DEFAULT_SEGMENT_SILENCE_MS,
  type VadRefs,
} from "../vad";

// --- computeSlidingNoiseFloor ---

describe("computeSlidingNoiseFloor", () => {
  const fill = (n: number, v: number): number[] => Array.from({ length: n }, () => v);

  it("returns current floor for empty samples", () => {
    expect(computeSlidingNoiseFloor([], 0.05, 1, 0.5)).toBe(0.05);
  });

  it("adopts higher noise floor immediately", () => {
    expect(computeSlidingNoiseFloor(fill(30, 0.05), 0.02, 1, 0.5)).toBe(0.05);
  });

  it("decays lower noise floor gradually", () => {
    // currentFloor=0.04, elapsed=1s, rate=0.005 -> maxDrop=0.005, new=0.035
    expect(computeSlidingNoiseFloor(fill(30, 0.01), 0.04, 1, 0.005)).toBeCloseTo(0.035, 3);
  });

  it("limits decay rate per second", () => {
    // currentFloor=0.5, elapsed=0.1s, rate=0.5 -> maxDrop=0.05 -> 0.45
    expect(computeSlidingNoiseFloor(fill(4, 0.01), 0.5, 0.1, 0.5)).toBeCloseTo(0.45);
  });

  it("uses 25th percentile to ignore speech spikes", () => {
    const samples = [...fill(25, 0.02), ...fill(5, 0.5)];
    // sorted[floor(30*0.25)] = sorted[7] = 0.02 -> same as current floor
    expect(computeSlidingNoiseFloor(samples, 0.02, 1, 0.5)).toBe(0.02);
  });

  it("prevents decay with zero elapsed time", () => {
    expect(computeSlidingNoiseFloor(fill(30, 0.01), 0.04, 0, 0.5)).toBe(0.04);
  });

  it("stable noise remains unchanged", () => {
    expect(computeSlidingNoiseFloor(fill(30, 0.02), 0.02, 1, 0.5)).toBe(0.02);
  });
});

// --- createVadRefs ---

describe("createVadRefs", () => {
  it("returns idle state with default thresholds", () => {
    const refs = createVadRefs();
    expect(refs.state).toBe("idle");
    expect(refs.recordingStart).toBe(0);
    expect(refs.silenceStart).toBe(0);
    expect(refs.noiseFloorSamples).toEqual([]);
    expect(refs.slidingWindow).toEqual([]);
    expect(refs.slidingWindowIdx).toBe(0);
    expect(refs.speechThreshold).toBeGreaterThan(0);
    expect(refs.silenceThreshold).toBeGreaterThan(0);
    expect(refs.speechThreshold).toBeGreaterThan(refs.silenceThreshold);
  });
});

// --- vadTick ---

describe("vadTick", () => {
  function makeVad(overrides: Partial<VadRefs> = {}): VadRefs {
    return {
      ...createVadRefs(),
      speechThreshold: 0.06,
      silenceThreshold: 0.02,
      ...overrides,
    };
  }

  it("returns null in idle state", () => {
    const vad = makeVad({ state: "idle" });
    expect(vadTick(vad, 0.5, 1000, 2000)).toBeNull();
    expect(vad.state).toBe("idle");
  });

  it("collects samples during calibration", () => {
    const vad = makeVad({ state: "calibrating", recordingStart: 0 });
    vadTick(vad, 0.03, 200, 2000);
    expect(vad.noiseFloorSamples).toContain(0.03);
    expect(vad.state).toBe("calibrating");
  });

  it("transitions from calibrating to waitingForSpeech after 500ms", () => {
    const vad = makeVad({ state: "calibrating", recordingStart: 0, noiseFloorSamples: [0.02, 0.02, 0.02] });
    vadTick(vad, 0.02, 600, 2000);
    expect(vad.state).toBe("waitingForSpeech");
  });

  it("sets adaptive thresholds from calibration noise floor", () => {
    const vad = createVadRefs();
    vad.state = "calibrating";
    vad.recordingStart = 0;
    for (let t = 0; t < 500; t += 66) vadTick(vad, 0.01, t);
    vadTick(vad, 0.01, 500);
    expect(vad.state).toBe("waitingForSpeech");
    expect(vad.silenceThreshold).toBe(0.02);
    expect(vad.speechThreshold).toBe(0.06);
  });

  it("transitions to speechDetected when RMS exceeds threshold", () => {
    const vad = makeVad({ state: "waitingForSpeech", recordingStart: 0 });
    vadTick(vad, 0.01, 1000);
    expect(vad.state).toBe("waitingForSpeech");
    vadTick(vad, 0.5, 1100);
    expect(vad.state).toBe("speechDetected");
  });

  it("returns no-speech after 15s timeout with no speech", () => {
    const vad = makeVad({ state: "waitingForSpeech", recordingStart: 0 });
    expect(vadTick(vad, 0.01, 14_999)).toBeNull();
    expect(vadTick(vad, 0.01, 15_001)).toBe("no-speech");
  });

  it("transitions speechDetected to watchingSilence on low RMS", () => {
    const vad = makeVad({ state: "speechDetected" });
    vadTick(vad, 0.001, 2000);
    expect(vad.state).toBe("watchingSilence");
    expect(vad.silenceStart).toBe(2000);
  });

  it("returns stop after silence timeout in watchingSilence", () => {
    const vad = makeVad({ state: "watchingSilence", silenceStart: 1000 });
    expect(vadTick(vad, 0.001, 2999)).toBeNull();
    expect(vadTick(vad, 0.001, 3000)).toBe("stop");
  });

  it("returns to speechDetected from watchingSilence on loud RMS", () => {
    const vad = makeVad({ state: "watchingSilence", silenceStart: 1000 });
    vadTick(vad, 0.5, 1500);
    expect(vad.state).toBe("speechDetected");
  });

  it("uses custom silence timeout", () => {
    const vad = makeVad({ state: "watchingSilence", silenceStart: 1000 });
    expect(vadTick(vad, 0.001, 1499, 500)).toBeNull();
    expect(vadTick(vad, 0.001, 1500, 500)).toBe("stop");
  });

  it("uses default silence timeout when not provided", () => {
    const vad = makeVad({ state: "watchingSilence", silenceStart: 0 });
    expect(vadTick(vad, 0.001, VAD_DEFAULT_SILENCE_TIMEOUT_MS - 1)).toBeNull();
    expect(vadTick(vad, 0.001, VAD_DEFAULT_SILENCE_TIMEOUT_MS)).toBe("stop");
  });
});

// --- Full VAD lifecycle ---

describe("VAD lifecycle", () => {
  it("calibrate -> detect speech -> detect silence -> stop", () => {
    const vad = createVadRefs();
    vad.state = "calibrating";
    vad.recordingStart = 0;

    for (let t = 0; t < 500; t += 66) vadTick(vad, 0.005, t);
    vadTick(vad, 0.005, 500);
    expect(vad.state).toBe("waitingForSpeech");

    vadTick(vad, 0.005, 600);
    expect(vad.state).toBe("waitingForSpeech");

    vadTick(vad, 0.3, 700);
    expect(vad.state).toBe("speechDetected");

    vadTick(vad, 0.25, 800);
    expect(vad.state).toBe("speechDetected");

    vadTick(vad, 0.001, 1000);
    expect(vad.state).toBe("watchingSilence");

    vadTick(vad, 0.001, 2500);
    expect(vad.state).toBe("watchingSilence");

    expect(vadTick(vad, 0.001, 3000)).toBe("stop");
  });
});

// --- Segment boundary ---

describe("vadTick segment-boundary", () => {
  function makeVad(overrides: Partial<VadRefs> = {}): VadRefs {
    return {
      ...createVadRefs(),
      speechThreshold: 0.06,
      silenceThreshold: 0.02,
      ...overrides,
    };
  }

  it("emits segment-boundary when silence exceeds segmentSilenceMs", () => {
    const vad = makeVad({
      state: "watchingSilence",
      silenceStart: 1000,
      segmentSilenceMs: 1500,
    });
    expect(vadTick(vad, 0.001, 2499)).toBeNull();
    expect(vadTick(vad, 0.001, 2500)).toBe("segment-boundary");
    expect(vad.segmentBoundaryEmitted).toBe(true);
  });

  it("does not emit segment-boundary twice for same silence gap", () => {
    const vad = makeVad({
      state: "watchingSilence",
      silenceStart: 1000,
      segmentSilenceMs: 1500,
    });
    expect(vadTick(vad, 0.001, 2500)).toBe("segment-boundary");
    expect(vadTick(vad, 0.001, 2600)).toBeNull(); // not emitted again
  });

  it("emits stop after segment-boundary when silence continues", () => {
    const vad = makeVad({
      state: "watchingSilence",
      silenceStart: 1000,
      segmentSilenceMs: 1500,
    });
    expect(vadTick(vad, 0.001, 2500)).toBe("segment-boundary");
    expect(vadTick(vad, 0.001, 3000, 2000)).toBe("stop");
  });

  it("does not emit segment-boundary when segmentSilenceMs is 0", () => {
    const vad = makeVad({
      state: "watchingSilence",
      silenceStart: 1000,
      segmentSilenceMs: 0,
    });
    expect(vadTick(vad, 0.001, 2500)).toBeNull();
    expect(vadTick(vad, 0.001, 3000)).toBe("stop");
  });

  it("resets segmentBoundaryEmitted when speech resumes", () => {
    const vad = makeVad({
      state: "watchingSilence",
      silenceStart: 1000,
      segmentSilenceMs: 1500,
    });
    vadTick(vad, 0.001, 2500); // segment-boundary fires
    expect(vad.segmentBoundaryEmitted).toBe(true);

    // Speech resumes
    vadTick(vad, 0.5, 2600);
    expect(vad.state).toBe("speechDetected");

    // Silence again
    vadTick(vad, 0.001, 2700);
    expect(vad.state).toBe("watchingSilence");
    expect(vad.segmentBoundaryEmitted).toBe(false); // reset!
  });

  it("segment-boundary fires before stop at correct thresholds", () => {
    const vad = makeVad({
      state: "watchingSilence",
      silenceStart: 0,
      segmentSilenceMs: 1500,
    });
    // At 1500ms: segment-boundary
    expect(vadTick(vad, 0.001, 1500)).toBe("segment-boundary");
    // At 2000ms: stop
    expect(vadTick(vad, 0.001, 2000)).toBe("stop");
  });
});

// --- Persistent mode lifecycle ---

describe("VAD persistent mode lifecycle", () => {
  it("calibrate -> speech -> silence -> segment-boundary -> speech -> silence -> segment-boundary", () => {
    const vad = createVadRefs();
    vad.state = "calibrating";
    vad.recordingStart = 0;
    vad.segmentSilenceMs = 1500;

    // Calibrate
    for (let t = 0; t < 500; t += 66) vadTick(vad, 0.005, t);
    vadTick(vad, 0.005, 500);
    expect(vad.state).toBe("waitingForSpeech");

    // Speech detected
    vadTick(vad, 0.3, 700);
    expect(vad.state).toBe("speechDetected");

    // Silence
    vadTick(vad, 0.001, 1000);
    expect(vad.state).toBe("watchingSilence");

    // Segment boundary at 1000 + 1500 = 2500
    expect(vadTick(vad, 0.001, 2500)).toBe("segment-boundary");

    // Speech resumes
    vadTick(vad, 0.3, 3000);
    expect(vad.state).toBe("speechDetected");

    // Another silence period
    vadTick(vad, 0.001, 4000);
    expect(vad.state).toBe("watchingSilence");

    // Another segment boundary at 4000 + 1500 = 5500
    expect(vadTick(vad, 0.001, 5500)).toBe("segment-boundary");
  });
});

// --- Constants ---

describe("VAD constants", () => {
  it("exports sensible defaults", () => {
    expect(VAD_DEFAULT_SILENCE_TIMEOUT_MS).toBe(2000);
    expect(VAD_DEFAULT_SEGMENT_SILENCE_MS).toBe(1500);
    expect(VAD_DEFAULT_SEGMENT_SILENCE_MS).toBeLessThan(VAD_DEFAULT_SILENCE_TIMEOUT_MS);
  });
});
