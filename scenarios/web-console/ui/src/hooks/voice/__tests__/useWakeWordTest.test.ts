import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useWakeWordTest } from "../wakeword/useWakeWordTest";
import type { AudioFeatures, MatchResult, WakeWordEngine } from "../wakeword/types";

// ---------------------------------------------------------------------------
// Mock engine
// ---------------------------------------------------------------------------
function createMockEngine(score = 0.8, isMatch = true): WakeWordEngine {
  return {
    extractFeatures: vi.fn((): AudioFeatures => ({
      kind: "mfcc-v1",
      data: [[1, 2, 3]],
      sampleRate: 16000,
      durationSec: 1,
    })),
    compare: vi.fn((): MatchResult => ({ score, isMatch })),
    compareBest: vi.fn((): MatchResult => ({ score, isMatch })),
  };
}

const sampleFeatures: AudioFeatures = {
  kind: "mfcc-v1",
  data: [[1, 2, 3]],
  sampleRate: 16000,
  durationSec: 1,
};

// ---------------------------------------------------------------------------
// Controllable clock for performance.now()
// ---------------------------------------------------------------------------
let nowMs = 1000;
function advanceNow(ms: number) { nowMs += ms; }

// ---------------------------------------------------------------------------
// Mock MediaRecorder + getUserMedia + AudioContext
// ---------------------------------------------------------------------------
let mockRecorderInstance: {
  start: ReturnType<typeof vi.fn>;
  stop: ReturnType<typeof vi.fn>;
  state: string;
  ondataavailable: ((e: { data: Blob }) => void) | null;
  onstop: (() => void) | null;
  onerror: (() => void) | null;
};

function setupMediaMocks() {
  mockRecorderInstance = {
    start: vi.fn(() => { mockRecorderInstance.state = "recording"; }),
    stop: vi.fn(() => {
      mockRecorderInstance.state = "inactive";
      // Provide a chunk with a working arrayBuffer method
      const fakeBlob = new Blob(["audio-data"], { type: "audio/webm" });
      mockRecorderInstance.ondataavailable?.({ data: fakeBlob });
      mockRecorderInstance.onstop?.();
    }),
    state: "inactive",
    ondataavailable: null,
    onstop: null,
    onerror: null,
  };

  vi.stubGlobal("MediaRecorder", vi.fn(() => mockRecorderInstance));
  (MediaRecorder as unknown as { isTypeSupported: (t: string) => boolean }).isTypeSupported = () => true;

  const mockStream = {
    getTracks: () => [{ stop: vi.fn() }],
  };
  vi.stubGlobal("navigator", {
    ...navigator,
    mediaDevices: {
      getUserMedia: vi.fn(() => Promise.resolve(mockStream)),
    },
  });

  // Mock AudioContext so decodeAudioData returns a valid buffer
  const mockAudioBuffer = {
    getChannelData: () => new Float32Array(16000),
  };
  vi.stubGlobal("AudioContext", vi.fn(() => ({
    decodeAudioData: vi.fn(() => Promise.resolve(mockAudioBuffer)),
    close: vi.fn(() => Promise.resolve()),
    sampleRate: 16000,
  })));
}

/** Flush multiple rounds of microtasks (the decode pipeline has ~5 awaits). */
async function flushMicrotasks(rounds = 10) {
  for (let i = 0; i < rounds; i++) {
    // Advance fake timers to fire any pending setTimeout(0) calls
    vi.advanceTimersByTime(1);
    // Yield to let promise continuations run
    await Promise.resolve();
  }
}

// Polyfill Blob.arrayBuffer for jsdom which doesn't support it
if (!Blob.prototype.arrayBuffer) {
  Blob.prototype.arrayBuffer = function () {
    return new Promise((resolve) => {
      const reader = new FileReader();
      reader.onload = () => resolve(reader.result as ArrayBuffer);
      reader.readAsArrayBuffer(this);
    });
  };
}

describe("useWakeWordTest", () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: false });
    nowMs = 1000;
    vi.spyOn(performance, "now").mockImplementation(() => nowMs);
    setupMediaMocks();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("starts in idle state with empty history", () => {
    const engine = createMockEngine();
    const { result } = renderHook(() =>
      useWakeWordTest({ engine, samples: [sampleFeatures], threshold: 0.65, disabled: false }),
    );
    expect(result.current.state.status).toBe("idle");
    expect(result.current.state.history).toEqual([]);
    expect(result.current.state.currentResult).toBeNull();
    expect(result.current.state.error).toBeNull();
  });

  it("does nothing when disabled", async () => {
    const engine = createMockEngine();
    const { result } = renderHook(() =>
      useWakeWordTest({ engine, samples: [sampleFeatures], threshold: 0.65, disabled: true }),
    );

    await act(async () => { result.current.startRecording(); });

    expect(result.current.state.status).toBe("idle");
    expect(navigator.mediaDevices.getUserMedia).not.toHaveBeenCalled();
  });

  it("does nothing when no samples provided", async () => {
    const engine = createMockEngine();
    const { result } = renderHook(() =>
      useWakeWordTest({ engine, samples: [], threshold: 0.65, disabled: false }),
    );

    await act(async () => { result.current.startRecording(); });

    expect(result.current.state.status).toBe("idle");
  });

  it("transitions idle → recording on startRecording", async () => {
    const engine = createMockEngine();
    const { result } = renderHook(() =>
      useWakeWordTest({ engine, samples: [sampleFeatures], threshold: 0.65, disabled: false }),
    );

    await act(async () => { result.current.startRecording(); });

    expect(result.current.state.status).toBe("recording");
  });

  it("transitions recording → comparing → result on stopRecording after sufficient duration", async () => {
    const engine = createMockEngine(0.8, true);
    const { result } = renderHook(() =>
      useWakeWordTest({ engine, samples: [sampleFeatures], threshold: 0.65, disabled: false }),
    );

    await act(async () => { result.current.startRecording(); });
    expect(result.current.state.status).toBe("recording");

    // Simulate 1s of elapsed time
    advanceNow(1000);

    await act(async () => { result.current.stopRecording(); });
    // Flush async decode + compare
    await act(async () => { await flushMicrotasks(); });

    expect(result.current.state.status).toBe("result");
    const matchResult = result.current.state.currentResult;
    expect(matchResult).not.toBeNull();
    expect(matchResult?.score).toBe(0.8);
    expect(matchResult?.isMatch).toBe(true);
  });

  it("shows error for too-short recordings", async () => {
    const engine = createMockEngine();
    const { result } = renderHook(() =>
      useWakeWordTest({ engine, samples: [sampleFeatures], threshold: 0.65, disabled: false }),
    );

    await act(async () => { result.current.startRecording(); });

    // Only 200ms elapsed — below 500ms minimum
    advanceNow(200);

    await act(async () => { result.current.stopRecording(); });
    await act(async () => { await flushMicrotasks(); });

    expect(result.current.state.status).toBe("idle");
    expect(result.current.state.error).toMatch(/hold.*longer/i);
  });

  it("caps history at 10 entries", async () => {
    const engine = createMockEngine(0.75, true);
    const { result } = renderHook(() =>
      useWakeWordTest({ engine, samples: [sampleFeatures], threshold: 0.65, disabled: false }),
    );

    for (let i = 0; i < 12; i++) {
      await act(async () => { result.current.startRecording(); });
      advanceNow(1000);
      await act(async () => { result.current.stopRecording(); });
      await act(async () => { await flushMicrotasks(); });
    }

    expect(result.current.state.history.length).toBeLessThanOrEqual(10);
  });

  it("clearHistory resets history and current result", async () => {
    const engine = createMockEngine(0.8, true);
    const { result } = renderHook(() =>
      useWakeWordTest({ engine, samples: [sampleFeatures], threshold: 0.65, disabled: false }),
    );

    await act(async () => { result.current.startRecording(); });
    advanceNow(1000);
    await act(async () => { result.current.stopRecording(); });
    await act(async () => { await flushMicrotasks(); });

    expect(result.current.state.history.length).toBe(1);

    act(() => { result.current.clearHistory(); });

    expect(result.current.state.history).toEqual([]);
    expect(result.current.state.currentResult).toBeNull();
    expect(result.current.state.error).toBeNull();
  });

  it("auto-stops at 3s max duration", async () => {
    const engine = createMockEngine(0.7, true);
    const { result } = renderHook(() =>
      useWakeWordTest({ engine, samples: [sampleFeatures], threshold: 0.65, disabled: false }),
    );

    await act(async () => { result.current.startRecording(); });
    expect(result.current.state.status).toBe("recording");

    // Advance past the 3s auto-stop timeout
    await act(async () => { vi.advanceTimersByTime(3100); });

    expect(mockRecorderInstance.stop).toHaveBeenCalled();
  });

  it("increments recordingSeconds while recording", async () => {
    const engine = createMockEngine();
    const { result } = renderHook(() =>
      useWakeWordTest({ engine, samples: [sampleFeatures], threshold: 0.65, disabled: false }),
    );

    await act(async () => { result.current.startRecording(); });
    expect(result.current.state.recordingSeconds).toBe(0);

    await act(async () => { vi.advanceTimersByTime(1000); });
    expect(result.current.state.recordingSeconds).toBe(1);

    await act(async () => { vi.advanceTimersByTime(1000); });
    expect(result.current.state.recordingSeconds).toBe(2);
  });

  it("reports reject result when score is below threshold", async () => {
    const engine = createMockEngine(0.3, false);
    const { result } = renderHook(() =>
      useWakeWordTest({ engine, samples: [sampleFeatures], threshold: 0.65, disabled: false }),
    );

    await act(async () => { result.current.startRecording(); });
    advanceNow(1000);
    await act(async () => { result.current.stopRecording(); });
    await act(async () => { await flushMicrotasks(); });

    const noMatch = result.current.state.currentResult;
    expect(noMatch).not.toBeNull();
    expect(noMatch?.isMatch).toBe(false);
    expect(noMatch?.score).toBe(0.3);
  });

  it("sets error when getUserMedia fails", async () => {
    const engine = createMockEngine();
    vi.mocked(navigator.mediaDevices.getUserMedia).mockRejectedValueOnce(new Error("Permission denied"));

    const { result } = renderHook(() =>
      useWakeWordTest({ engine, samples: [sampleFeatures], threshold: 0.65, disabled: false }),
    );

    await act(async () => { result.current.startRecording(); });
    await act(async () => { await flushMicrotasks(); });

    expect(result.current.state.status).toBe("idle");
    expect(result.current.state.error).toMatch(/mic access failed/i);
  });
});
