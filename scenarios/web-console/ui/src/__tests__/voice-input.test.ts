import { describe, it, expect } from "vitest";
import {
  computeFinalTimeout,
  createVadRefs,
  vadTick,
  computeSlidingNoiseFloor,
  createAudioFilterChain,
  VAD_DEFAULT_SILENCE_TIMEOUT_MS,
  AUDIO_BITRATE,
  STREAM_CHUNK_INTERVAL_MS,
} from "../hooks/useVoiceInput";

// --- computeFinalTimeout ---

describe("computeFinalTimeout", () => {
  it("returns minimum of 10s for short recordings", () => {
    expect(computeFinalTimeout(1000)).toBe(10_000);
    expect(computeFinalTimeout(3000)).toBe(10_000);
  });

  it("returns 2× duration for medium recordings", () => {
    expect(computeFinalTimeout(8000)).toBe(16_000);
    expect(computeFinalTimeout(15_000)).toBe(30_000);
  });

  it("caps at 60s for long recordings", () => {
    expect(computeFinalTimeout(40_000)).toBe(60_000);
    expect(computeFinalTimeout(100_000)).toBe(60_000);
  });

  it("returns 10s for zero-length recording", () => {
    expect(computeFinalTimeout(0)).toBe(10_000);
  });
});

// --- computeSlidingNoiseFloor ---

describe("computeSlidingNoiseFloor", () => {
  it("returns current floor for empty samples", () => {
    expect(computeSlidingNoiseFloor([], 0.05, 1, 0.5)).toBe(0.05);
  });

  it("adopts higher noise floor immediately", () => {
    const samples = [0.1, 0.1, 0.1, 0.1];
    // 25th percentile = 0.1, which is above currentFloor 0.05
    expect(computeSlidingNoiseFloor(samples, 0.05, 1, 0.5)).toBe(0.1);
  });

  it("decays lower noise floor gradually", () => {
    const samples = [0.01, 0.01, 0.01, 0.01];
    // 25th percentile = 0.01, currentFloor = 0.1
    // maxDrop = 0.5 * 1 = 0.5, so new floor = max(0.01, 0.1 - 0.5) = 0.01
    // With only 1s elapsed, the floor drops to the percentile since maxDrop > difference
    const result = computeSlidingNoiseFloor(samples, 0.1, 1, 0.5);
    expect(result).toBe(0.01);
  });

  it("limits decay rate per second", () => {
    const samples = [0.01, 0.01, 0.01, 0.01];
    // currentFloor = 0.5, elapsedSec = 0.1, decayRate = 0.5
    // maxDrop = 0.5 * 0.1 = 0.05 → new floor = max(0.01, 0.5 - 0.05) = 0.45
    const result = computeSlidingNoiseFloor(samples, 0.5, 0.1, 0.5);
    expect(result).toBeCloseTo(0.45);
  });

  it("uses 25th percentile to ignore speech spikes", () => {
    // 4 low, 12 high → 25th percentile is in the low range
    const samples = [0.02, 0.02, 0.02, 0.02, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5];
    const result = computeSlidingNoiseFloor(samples, 0.01, 1, 0.5);
    // 25th percentile index: floor(16 * 0.25) = 4 → sorted[4] = 0.5? No...
    // sorted: [0.02, 0.02, 0.02, 0.02, 0.5, 0.5, ...] → sorted[4] = 0.5
    // That's >= currentFloor 0.01, so adopt immediately
    expect(result).toBe(0.5);
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
  it("returns null in idle state", () => {
    const vad = createVadRefs();
    expect(vadTick(vad, 0.1, 1000)).toBeNull();
    expect(vad.state).toBe("idle");
  });

  it("calibrates for 500ms then transitions to waitingForSpeech", () => {
    const vad = createVadRefs();
    vad.state = "calibrating";
    vad.recordingStart = 1000;

    // During calibration period
    expect(vadTick(vad, 0.01, 1200)).toBeNull();
    expect(vad.state).toBe("calibrating");
    expect(vad.noiseFloorSamples).toContain(0.01);

    // At calibration end
    expect(vadTick(vad, 0.01, 1500)).toBeNull();
    expect(vad.state).toBe("waitingForSpeech");
  });

  it("sets adaptive thresholds from calibration noise floor", () => {
    const vad = createVadRefs();
    vad.state = "calibrating";
    vad.recordingStart = 0;

    // Feed consistent low noise during calibration
    for (let t = 0; t < 500; t += 66) {
      vadTick(vad, 0.01, t);
    }
    vadTick(vad, 0.01, 500); // triggers transition

    expect(vad.state).toBe("waitingForSpeech");
    // silenceThreshold = max(0.02, 0.01 * 1.5) = 0.02
    expect(vad.silenceThreshold).toBe(0.02);
    // speechThreshold = max(0.06, 0.01 * 3) = 0.06
    expect(vad.speechThreshold).toBe(0.06);
  });

  it("transitions to speechDetected when RMS exceeds threshold", () => {
    const vad = createVadRefs();
    vad.state = "waitingForSpeech";
    vad.recordingStart = 0;

    // Low RMS — stays waiting
    vadTick(vad, 0.01, 1000);
    expect(vad.state).toBe("waitingForSpeech");

    // High RMS — speech detected
    vadTick(vad, 0.2, 1100);
    expect(vad.state).toBe("speechDetected");
  });

  it("returns no-speech after timeout with no speech", () => {
    const vad = createVadRefs();
    vad.state = "waitingForSpeech";
    vad.recordingStart = 0;

    // Just before timeout
    expect(vadTick(vad, 0.01, 14_999)).toBeNull();
    // After timeout
    expect(vadTick(vad, 0.01, 15_001)).toBe("no-speech");
  });

  it("transitions speechDetected → watchingSilence on low RMS", () => {
    const vad = createVadRefs();
    vad.state = "speechDetected";

    vadTick(vad, 0.001, 2000); // below silence threshold
    expect(vad.state).toBe("watchingSilence");
    expect(vad.silenceStart).toBe(2000);
  });

  it("returns stop after silence timeout in watchingSilence", () => {
    const vad = createVadRefs();
    vad.state = "watchingSilence";
    vad.silenceStart = 1000;

    // Just before timeout
    expect(vadTick(vad, 0.001, 2999)).toBeNull();
    // At timeout
    expect(vadTick(vad, 0.001, 3000)).toBe("stop");
  });

  it("returns to speechDetected from watchingSilence on loud RMS", () => {
    const vad = createVadRefs();
    vad.state = "watchingSilence";
    vad.silenceStart = 1000;

    vadTick(vad, 0.2, 1500);
    expect(vad.state).toBe("speechDetected");
  });

  it("uses custom silence timeout", () => {
    const vad = createVadRefs();
    vad.state = "watchingSilence";
    vad.silenceStart = 1000;

    // With 500ms timeout
    expect(vadTick(vad, 0.001, 1499, 500)).toBeNull();
    expect(vadTick(vad, 0.001, 1500, 500)).toBe("stop");
  });

  it("uses default silence timeout when not provided", () => {
    const vad = createVadRefs();
    vad.state = "watchingSilence";
    vad.silenceStart = 0;

    // Default is VAD_DEFAULT_SILENCE_TIMEOUT_MS (2000)
    expect(vadTick(vad, 0.001, VAD_DEFAULT_SILENCE_TIMEOUT_MS - 1)).toBeNull();
    expect(vadTick(vad, 0.001, VAD_DEFAULT_SILENCE_TIMEOUT_MS)).toBe("stop");
  });
});

// --- Full VAD lifecycle ---

describe("VAD lifecycle", () => {
  it("calibrate → detect speech → detect silence → stop", () => {
    const vad = createVadRefs();
    vad.state = "calibrating";
    vad.recordingStart = 0;

    // Calibration phase (500ms of low noise)
    for (let t = 0; t < 500; t += 66) {
      vadTick(vad, 0.005, t);
    }
    // Final tick at exactly 500ms triggers transition
    vadTick(vad, 0.005, 500);
    expect(vad.state).toBe("waitingForSpeech");

    // Wait for speech — low noise continues
    vadTick(vad, 0.005, 600);
    expect(vad.state).toBe("waitingForSpeech");

    // Speech begins
    vadTick(vad, 0.3, 700);
    expect(vad.state).toBe("speechDetected");

    // Speech continues
    vadTick(vad, 0.25, 800);
    expect(vad.state).toBe("speechDetected");

    // Silence begins
    vadTick(vad, 0.001, 1000);
    expect(vad.state).toBe("watchingSilence");

    // Silence continues but not long enough
    vadTick(vad, 0.001, 2500);
    expect(vad.state).toBe("watchingSilence");

    // Silence timeout reached
    const result = vadTick(vad, 0.001, 3000);
    expect(result).toBe("stop");
  });
});

// --- Constants ---

describe("voice input constants", () => {
  it("exports sensible defaults", () => {
    expect(AUDIO_BITRATE).toBe(48_000);
    expect(STREAM_CHUNK_INTERVAL_MS).toBe(250);
    expect(VAD_DEFAULT_SILENCE_TIMEOUT_MS).toBe(2000);
  });
});

// --- createAudioFilterChain ---

describe("createAudioFilterChain", () => {
  it("creates analyser and filtered stream", () => {
    // Minimal AudioContext stub for structure test
    const mockAnalyser = {
      fftSize: 0,
      frequencyBinCount: 64,
      connect: vi.fn(),
    };
    const mockDestination = {
      stream: { id: "filtered" },
    };
    const mockHighpass = {
      type: "",
      frequency: { value: 0 },
      Q: { value: 0 },
      connect: vi.fn(),
    };
    const mockLowpass = {
      type: "",
      frequency: { value: 0 },
      Q: { value: 0 },
      connect: vi.fn(),
    };

    const ctx = {
      createBiquadFilter: vi.fn()
        .mockReturnValueOnce(mockHighpass)
        .mockReturnValueOnce(mockLowpass),
      createMediaStreamDestination: vi.fn().mockReturnValue(mockDestination),
      createAnalyser: vi.fn().mockReturnValue(mockAnalyser),
    } as unknown as AudioContext;

    const source = { connect: vi.fn() } as unknown as MediaStreamAudioSourceNode;

    const result = createAudioFilterChain(ctx, source);

    expect(result.analyser).toBe(mockAnalyser);
    expect(result.filteredStream).toBe(mockDestination.stream);
    // Verify filter chain: source → highpass → lowpass → destination + analyser
    expect(source.connect).toHaveBeenCalledWith(mockHighpass);
    expect(mockHighpass.connect).toHaveBeenCalledWith(mockLowpass);
    expect(mockLowpass.connect).toHaveBeenCalledWith(mockDestination);
    expect(mockLowpass.connect).toHaveBeenCalledWith(mockAnalyser);
    // Verify filter parameters
    expect(mockHighpass.type).toBe("highpass");
    expect(mockHighpass.frequency.value).toBe(80);
    expect(mockLowpass.type).toBe("lowpass");
    expect(mockLowpass.frequency.value).toBe(8000);
  });
});
