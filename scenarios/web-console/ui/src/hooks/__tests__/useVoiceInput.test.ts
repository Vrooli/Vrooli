import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { apiBaseMock } from "../../test-utils";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import {
  AUDIO_BITRATE,
  STREAM_CHUNK_INTERVAL_MS,
  VAD_DEFAULT_SILENCE_TIMEOUT_MS,
  computeFinalTimeout,
  createAudioFilterChain,
  computeSlidingNoiseFloor,
  vadTick,
  createVadRefs,
  type VadRefs,
} from "../useVoiceInput";

vi.mock("@vrooli/api-base", () => apiBaseMock());

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const mockCapabilities = (whisperAvailable: boolean) => {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () =>
      Promise.resolve({
        capabilities: [
          {
            id: "whisper-stt",
            status: whisperAvailable ? "available" : "unavailable",
          },
        ],
        timestamp: new Date().toISOString(),
      }),
  }) as typeof fetch;
};

const mockMediaDevices = (success: boolean) => {
  const mockStream = {
    getTracks: () => [{ stop: vi.fn() }],
  } as unknown as MediaStream;
  Object.defineProperty(navigator, "mediaDevices", {
    value: {
      getUserMedia: success
        ? vi.fn().mockResolvedValue(mockStream)
        : vi.fn().mockRejectedValue(new Error("Permission denied")),
    },
    configurable: true,
  });
};

/** Minimal SpeechRecognition stub */
function installSpeechRecognition() {
  window.SpeechRecognition = class {
    continuous = false;
    interimResults = false;
    lang = "";
    onresult: ((e: unknown) => void) | null = null;
    onerror: ((e: unknown) => void) | null = null;
    onend: (() => void) | null = null;
    start() {}
    stop() {}
    abort() {}
    addEventListener() {}
    removeEventListener() {}
    dispatchEvent() {
      return false;
    }
  } as unknown as typeof window.SpeechRecognition;
}

function removeSpeechRecognition() {
  delete window.SpeechRecognition;
  delete window.webkitSpeechRecognition;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("useVoiceInput", () => {
  let originalFetch: typeof globalThis.fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    vi.clearAllMocks();
    removeSpeechRecognition();
    useWorkspaceStore.setState({ voiceEnabled: true });
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("falls back to web-speech when whisper unavailable", async () => {
    mockCapabilities(false);
    installSpeechRecognition();
    mockMediaDevices(true);

    const onTranscript = vi.fn();
    // Dynamic import so vi.mock is applied
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    // Wait for the async capability detection to settle
    await act(async () => {
      await vi.dynamicImportSettled?.();
      // Allow microtasks from the effect to flush
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(result.current.backend).toBe("web-speech");
    expect(result.current.supported).toBe(true);
  });

  it("uses whisper when available", async () => {
    mockCapabilities(true);
    mockMediaDevices(true);

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(result.current.backend).toBe("whisper");
    expect(result.current.supported).toBe(true);
  });

  it("reports unsupported when no backend available", async () => {
    mockCapabilities(false);
    removeSpeechRecognition();

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(result.current.supported).toBe(false);
    expect(result.current.backend).toBe("none");
  });

  it("disables when voiceEnabled is false", async () => {
    useWorkspaceStore.setState({ voiceEnabled: false });
    mockCapabilities(true);
    installSpeechRecognition();

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(result.current.supported).toBe(false);
    expect(result.current.backend).toBe("none");
  });

  it("reports error when capabilities fetch fails", async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(
      new Error("Network error"),
    ) as typeof fetch;
    removeSpeechRecognition();

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    // Should gracefully fall through to unsupported
    expect(result.current.supported).toBe(false);
    expect(result.current.backend).toBe("none");
  });
});

describe("AUDIO_BITRATE", () => {
  it("is 48kbps for optimal Whisper accuracy on localhost", () => {
    expect(AUDIO_BITRATE).toBe(48_000);
  });
});

describe("STREAM_CHUNK_INTERVAL_MS", () => {
  it("is 250ms for low-latency streaming", () => {
    expect(STREAM_CHUNK_INTERVAL_MS).toBe(250);
  });
});

describe("computeFinalTimeout", () => {
  const cases: [string, number, number][] = [
    ["zero duration → floor", 0, 10_000],
    ["short recording → floor", 3_000, 10_000],
    ["exactly at floor boundary", 5_000, 10_000],
    ["medium recording → 2× scaling", 15_000, 30_000],
    ["long recording → capped at 60s", 30_000, 60_000],
    ["very long recording → capped at 60s", 120_000, 60_000],
  ];

  it.each(cases)("%s (input=%d → expected=%d)", (_label, input, expected) => {
    expect(computeFinalTimeout(input)).toBe(expected);
  });
});

describe("createAudioFilterChain", () => {
  function createMockAudioContext() {
    const connectCalls: Array<{ from: string; to: string }> = [];

    const makeNode = (name: string) => ({
      _name: name,
      connect(target: { _name: string }) {
        connectCalls.push({ from: name, to: target._name });
        return target;
      },
      type: "" as string,
      frequency: { value: 0 },
      Q: { value: 0 },
      fftSize: 0,
      frequencyBinCount: 64,
      stream: { id: "filtered-stream" } as unknown as MediaStream,
    });

    let filterIdx = 0;
    const ctx = {
      createBiquadFilter: () => makeNode(`filter-${filterIdx++}`),
      createMediaStreamDestination: () => makeNode("destination"),
      createAnalyser: () => makeNode("analyser"),
    } as unknown as AudioContext;

    const source = makeNode("source") as unknown as MediaStreamAudioSourceNode;

    return { ctx, source, connectCalls };
  }

  it("creates highpass filter at 80Hz and lowpass at 8kHz", () => {
    const { ctx, source } = createMockAudioContext();
    // We need to track the created filters
    const filters: Array<{ type: string; frequency: { value: number }; Q: { value: number } }> = [];
    const origCreate = ctx.createBiquadFilter.bind(ctx);
    ctx.createBiquadFilter = () => {
      const node = origCreate();
      filters.push(node as typeof filters[0]);
      return node as unknown as BiquadFilterNode;
    };

    createAudioFilterChain(ctx, source);

    expect(filters).toHaveLength(2);
    const [hp, lp] = filters;
    expect(hp?.type).toBe("highpass");
    expect(hp?.frequency.value).toBe(80);
    expect(hp?.Q.value).toBeCloseTo(0.707);
    expect(lp?.type).toBe("lowpass");
    expect(lp?.frequency.value).toBe(8000);
    expect(lp?.Q.value).toBeCloseTo(0.707);
  });

  it("chains nodes: source → highpass → lowpass → destination + analyser", () => {
    const { ctx, source, connectCalls } = createMockAudioContext();
    createAudioFilterChain(ctx, source);

    expect(connectCalls).toEqual([
      { from: "source", to: "filter-0" },
      { from: "filter-0", to: "filter-1" },
      { from: "filter-1", to: "destination" },
      { from: "filter-1", to: "analyser" },
    ]);
  });

  it("returns filteredStream from destination node", () => {
    const { ctx, source } = createMockAudioContext();
    const result = createAudioFilterChain(ctx, source);

    expect(result.filteredStream).toBeDefined();
    expect(result.analyser).toBeDefined();
  });

  it("sets analyser fftSize to 128", () => {
    const { ctx, source } = createMockAudioContext();
    const result = createAudioFilterChain(ctx, source);
    expect((result.analyser as unknown as { fftSize: number }).fftSize).toBe(128);
  });
});

describe("computeSlidingNoiseFloor", () => {
  const fill = (n: number, v: number): number[] => Array.from({ length: n }, () => v);

  const cases: [string, number[], number, number, number, number][] = [
    ["stable noise remains unchanged", fill(30, 0.02), 0.02, 1, 0.5, 0.02],
    ["rising noise adopted immediately", fill(30, 0.05), 0.02, 1, 0.5, 0.05],
    [
      "falling noise decays gradually",
      fill(30, 0.01),
      0.04,
      1,
      0.005,
      0.035, // 0.04 - 0.005*1 = 0.035, > 0.01 so clamped to 0.035
    ],
    [
      "spike ignored via 25th percentile",
      [...fill(25, 0.02), ...fill(5, 0.5)],
      0.02,
      1,
      0.5,
      0.02,
    ],
    ["zero elapsed prevents decay", fill(30, 0.01), 0.04, 0, 0.5, 0.04],
    ["empty samples returns current floor", [], 0.04, 1, 0.5, 0.04],
  ];

  it.each(cases)(
    "%s",
    (_label, samples, currentFloor, elapsed, decayRate, expected) => {
      const result = computeSlidingNoiseFloor(samples, currentFloor, elapsed, decayRate);
      expect(result).toBeCloseTo(expected, 3);
    },
  );
});

// ---------------------------------------------------------------------------
// Phase 1: WebSpeech Deduplication Tests
// ---------------------------------------------------------------------------

/**
 * Controllable SpeechRecognition stub. Unlike the minimal stub, this one
 * captures the instance so tests can fire synthetic `onresult` events with
 * controlled cumulative results — exercising the `processedResultCount` fix.
 */
function installControllableSpeechRecognition() {
  type SRInstance = {
    continuous: boolean;
    interimResults: boolean;
    lang: string;
    onresult: ((e: unknown) => void) | null;
    onerror: ((e: unknown) => void) | null;
    onend: (() => void) | null;
    start(): void;
    stop(): void;
    abort(): void;
    addEventListener(): void;
    removeEventListener(): void;
    dispatchEvent(): boolean;
  };

  let instance: SRInstance | null = null;

  window.SpeechRecognition = class {
    continuous = false;
    interimResults = false;
    lang = "";
    onresult: ((e: unknown) => void) | null = null;
    onerror: ((e: unknown) => void) | null = null;
    onend: (() => void) | null = null;
    start() { instance = this as unknown as SRInstance; }
    stop() { this.onend?.(); }
    abort() {}
    addEventListener() {}
    removeEventListener() {}
    dispatchEvent() { return false; }
  } as unknown as typeof window.SpeechRecognition;

  /**
   * Build a synthetic SpeechRecognitionEvent from an array of result descriptors.
   * The results list is cumulative, matching browser behavior.
   */
  function fireResult(results: Array<{ transcript: string; isFinal: boolean }>) {
    if (!instance?.onresult) return;
    const resultList = results.map((r) => {
      const item = { transcript: r.transcript, confidence: 0.95 };
      return Object.assign([item], { isFinal: r.isFinal, length: 1, item: () => item });
    });
    const event = {
      results: Object.assign(resultList, {
        length: resultList.length,
        item: (i: number) => resultList[i],
      }),
    };
    instance.onresult(event);
  }

  return {
    getInstance: () => instance,
    fireResult,
    triggerEnd: () => instance?.onend?.(),
  };
}

describe("WebSpeechProvider deduplication", () => {
  let originalFetch: typeof globalThis.fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    vi.clearAllMocks();
    removeSpeechRecognition();
    useWorkspaceStore.setState({ voiceEnabled: true });
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("dispatches only new final results, not cumulative duplicates", async () => {
    mockCapabilities(false);
    mockMediaDevices(true);
    const ctrl = installControllableSpeechRecognition();

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    // Wait for backend detection
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    expect(result.current.backend).toBe("web-speech");

    // Start recording
    await act(async () => { await result.current.startRecording(); });

    // First final result: "hello"
    act(() => {
      ctrl.fireResult([{ transcript: "hello", isFinal: true }]);
    });
    expect(onTranscript).toHaveBeenCalledTimes(1);
    expect(onTranscript).toHaveBeenLastCalledWith("hello");

    // Second cumulative event: old "hello" + new " world"
    // Without the fix, this would dispatch "hello world" (including the old "hello")
    act(() => {
      ctrl.fireResult([
        { transcript: "hello", isFinal: true },
        { transcript: " world", isFinal: true },
      ]);
    });
    expect(onTranscript).toHaveBeenCalledTimes(2);
    expect(onTranscript).toHaveBeenLastCalledWith("world");
  });

  it("interim results update partialTranscript but do not dispatch as final", async () => {
    mockCapabilities(false);
    mockMediaDevices(true);
    const ctrl = installControllableSpeechRecognition();

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    await act(async () => { await result.current.startRecording(); });

    // Fire an interim (non-final) result
    act(() => {
      ctrl.fireResult([{ transcript: "hel", isFinal: false }]);
    });

    // onTranscript should NOT have been called for interim results
    expect(onTranscript).not.toHaveBeenCalled();
    // partialTranscript should be updated
    expect(result.current.partialTranscript).toBe("hel");
  });

  it("processedResultCount persists across spontaneous recognition restarts", async () => {
    mockCapabilities(false);
    mockMediaDevices(true);
    const ctrl = installControllableSpeechRecognition();

    const onTranscript = vi.fn();
    const { useVoiceInput } = await import("../useVoiceInput");
    const { result } = renderHook(() => useVoiceInput(onTranscript));

    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    await act(async () => { await result.current.startRecording(); });

    // Fire first final result
    act(() => {
      ctrl.fireResult([{ transcript: "hello", isFinal: true }]);
    });
    expect(onTranscript).toHaveBeenCalledTimes(1);

    // Simulate browser spontaneously ending recognition (triggers onend → restart)
    act(() => { ctrl.triggerEnd(); });

    // Fire cumulative results after restart — old "hello" + new " world"
    // processedResultCount should still be 1, so only " world" is new
    act(() => {
      ctrl.fireResult([
        { transcript: "hello", isFinal: true },
        { transcript: " world", isFinal: true },
      ]);
    });
    expect(onTranscript).toHaveBeenCalledTimes(2);
    expect(onTranscript).toHaveBeenLastCalledWith("world");
  });
});

// ---------------------------------------------------------------------------
// Phase 2: vadTick Tests
// ---------------------------------------------------------------------------

describe("VAD_DEFAULT_SILENCE_TIMEOUT_MS", () => {
  it("defaults to 2000ms", () => {
    expect(VAD_DEFAULT_SILENCE_TIMEOUT_MS).toBe(2000);
  });
});

describe("vadTick", () => {
  /** Helper: create VadRefs in a specific state with pre-set thresholds. */
  function makeVad(overrides: Partial<VadRefs> = {}): VadRefs {
    return {
      ...createVadRefs(),
      // Set reasonable post-calibration thresholds by default
      speechThreshold: 0.06,
      silenceThreshold: 0.02,
      ...overrides,
    };
  }

  it("returns null when idle regardless of RMS", () => {
    const vad = makeVad({ state: "idle" });
    expect(vadTick(vad, 0.5, 1000, 2000)).toBeNull();
  });

  it("collects samples during calibration", () => {
    const vad = makeVad({ state: "calibrating", recordingStart: 0 });
    vadTick(vad, 0.03, 200, 2000);
    expect(vad.noiseFloorSamples).toContain(0.03);
    expect(vad.state).toBe("calibrating");
  });

  it("transitions from calibrating to waitingForSpeech after 500ms", () => {
    const vad = makeVad({ state: "calibrating", recordingStart: 0, noiseFloorSamples: [0.02, 0.02, 0.02] });
    const result = vadTick(vad, 0.02, 600, 2000);
    expect(result).toBeNull();
    expect(vad.state).toBe("waitingForSpeech");
  });

  it("returns no-speech after 15s of waiting", () => {
    const vad = makeVad({ state: "waitingForSpeech", recordingStart: 0 });
    expect(vadTick(vad, 0.01, 16_000, 2000)).toBe("no-speech");
  });

  it("transitions from waitingForSpeech to speechDetected on loud RMS", () => {
    const vad = makeVad({ state: "waitingForSpeech", recordingStart: 0 });
    vadTick(vad, 0.5, 1000, 2000);
    expect(vad.state).toBe("speechDetected");
  });

  it("transitions from speechDetected to watchingSilence on quiet RMS", () => {
    const vad = makeVad({ state: "speechDetected" });
    vadTick(vad, 0.001, 5000, 2000);
    expect(vad.state).toBe("watchingSilence");
    expect(vad.silenceStart).toBe(5000);
  });

  it("returns stop after default 2000ms of silence", () => {
    const vad = makeVad({ state: "watchingSilence", silenceStart: 5000 });
    // Just before timeout
    expect(vadTick(vad, 0.001, 6999, 2000)).toBeNull();
    // At timeout
    expect(vadTick(vad, 0.001, 7000, 2000)).toBe("stop");
  });

  it("respects custom silence timeout of 3000ms", () => {
    const vad = makeVad({ state: "watchingSilence", silenceStart: 5000 });
    // At 2000ms (would trigger with default, but not with 3000ms)
    expect(vadTick(vad, 0.001, 7000, 3000)).toBeNull();
    // At 3000ms
    expect(vadTick(vad, 0.001, 8000, 3000)).toBe("stop");
  });

  it("does not stop before custom timeout elapses", () => {
    const vad = makeVad({ state: "watchingSilence", silenceStart: 5000 });
    expect(vadTick(vad, 0.001, 6500, 3000)).toBeNull();
    expect(vad.state).toBe("watchingSilence");
  });

  it("transitions back to speechDetected from watchingSilence on loud RMS", () => {
    const vad = makeVad({ state: "watchingSilence", silenceStart: 5000 });
    vadTick(vad, 0.5, 5500, 2000);
    expect(vad.state).toBe("speechDetected");
  });

  it("uses default timeout when parameter omitted", () => {
    const vad = makeVad({ state: "watchingSilence", silenceStart: 5000 });
    // Default is 2000ms — should stop at 7000
    expect(vadTick(vad, 0.001, 7000)).toBe("stop");
  });
});
