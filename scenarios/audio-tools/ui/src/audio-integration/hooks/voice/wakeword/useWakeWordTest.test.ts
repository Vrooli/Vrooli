// The hook's startRecording wrapper returns synchronously while scheduling
// async work, so several `act(async …)` bodies have no inner await — they flush
// React effects rather than await a value.
/* eslint-disable @typescript-eslint/require-await */
// Tests for useWakeWordTest — the live wake-word test state machine
// (idle → recording → comparing → result). Browser audio APIs
// (MediaRecorder, AudioContext, getUserMedia) are unavailable in jsdom, so
// they are replaced with controllable fakes.
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";

import { useWakeWordTest, type UseWakeWordTestOpts } from "./useWakeWordTest";
import type { AudioFeatures, MatchResult, WakeWordEngine } from "./types";

// ── Browser fakes ──────────────────────────────────────────────────────────

let nowVal = 0;

class FakeMediaRecorder {
  static isTypeSupported = vi.fn(() => true);
  static instances: FakeMediaRecorder[] = [];
  state: "inactive" | "recording" = "inactive";
  ondataavailable: ((e: { data: Blob }) => void) | null = null;
  onerror: (() => void) | null = null;
  onstop: (() => void) | null = null;
  start = vi.fn((_timeslice?: number) => {
    this.state = "recording";
  });
  stop = vi.fn(() => {
    this.state = "inactive";
    this.onstop?.();
  });
  constructor(
    public stream: MediaStream,
    public opts?: MediaRecorderOptions,
  ) {
    FakeMediaRecorder.instances.push(this);
  }
}

let decodeShouldReject = false;

class FakeAudioContext {
  state = "running";
  sampleRate: number;
  constructor(opts?: { sampleRate?: number }) {
    this.sampleRate = opts?.sampleRate ?? 48_000;
  }
  decodeAudioData(_buf: ArrayBuffer): Promise<AudioBuffer> {
    if (decodeShouldReject) return Promise.reject(new Error("bad audio"));
    return Promise.resolve({
      getChannelData: () => new Float32Array(16),
    } as unknown as AudioBuffer);
  }
  close(): Promise<void> {
    return Promise.resolve();
  }
}

function fakeStream(): MediaStream {
  return {
    getTracks: () => [{ stop: vi.fn(), readyState: "live" }],
  } as unknown as MediaStream;
}

let getUserMediaImpl: () => Promise<MediaStream>;

function makeEngine(result: MatchResult): WakeWordEngine {
  const features: AudioFeatures = { kind: "mfcc-v1", data: [], sampleRate: 16_000, durationSec: 1 };
  return {
    extractFeatures: vi.fn(() => features),
    compare: vi.fn(() => result),
    compareBest: vi.fn(() => result),
  };
}

function baseOpts(over: Partial<UseWakeWordTestOpts> = {}): UseWakeWordTestOpts {
  const samples: AudioFeatures[] = [
    { kind: "mfcc-v1", data: [], sampleRate: 16_000, durationSec: 1 },
  ];
  return {
    engine: makeEngine({ score: 0.9, isMatch: true }),
    samples,
    threshold: 0.65,
    disabled: false,
    ...over,
  };
}

beforeEach(() => {
  nowVal = 0;
  decodeShouldReject = false;
  FakeMediaRecorder.instances = [];
  vi.spyOn(performance, "now").mockImplementation(() => nowVal);
  getUserMediaImpl = () => Promise.resolve(fakeStream());
  Object.defineProperty(navigator, "mediaDevices", {
    configurable: true,
    value: { getUserMedia: vi.fn(() => getUserMediaImpl()) },
  });
  (globalThis as unknown as { MediaRecorder: typeof FakeMediaRecorder }).MediaRecorder = FakeMediaRecorder;
  (globalThis as unknown as { AudioContext: typeof FakeAudioContext }).AudioContext = FakeAudioContext;
  // jsdom's Blob does not implement arrayBuffer(); define it so the decode path runs.
  (Blob.prototype as unknown as { arrayBuffer: () => Promise<ArrayBuffer> }).arrayBuffer = () =>
    Promise.resolve(new ArrayBuffer(8));
});

afterEach(() => {
  vi.restoreAllMocks();
});

/** Push a non-empty chunk through the active recorder's ondataavailable. */
function pushChunk(recorder: FakeMediaRecorder): void {
  recorder.ondataavailable?.({ data: new Blob(["abcd"]) });
}

describe("useWakeWordTest", () => {
  it("starts idle", () => {
    const { result } = renderHook(() => useWakeWordTest(baseOpts()));
    expect(result.current.state.status).toBe("idle");
    expect(result.current.state.history).toEqual([]);
  });

  it("is a no-op when disabled", async () => {
    const gum = vi.spyOn(navigator.mediaDevices, "getUserMedia");
    const { result } = renderHook(() => useWakeWordTest(baseOpts({ disabled: true })));
    await act(async () => {
      result.current.startRecording();
    });
    expect(result.current.state.status).toBe("idle");
    expect(gum).not.toHaveBeenCalled();
  });

  it("is a no-op when there are no enrolled samples", async () => {
    const gum = vi.spyOn(navigator.mediaDevices, "getUserMedia");
    const { result } = renderHook(() => useWakeWordTest(baseOpts({ samples: [] })));
    await act(async () => {
      result.current.startRecording();
    });
    expect(result.current.state.status).toBe("idle");
    expect(gum).not.toHaveBeenCalled();
  });

  it("records, compares, and reports a match", async () => {
    const { result } = renderHook(() => useWakeWordTest(baseOpts()));
    await act(async () => {
      result.current.startRecording();
    });
    await waitFor(() => expect(result.current.state.status).toBe("recording"));

    // Push a chunk and advance the clock past MIN_DURATION before stopping.
    pushChunk(lastRecorder());
    nowVal = 1_000;
    await act(async () => {
      result.current.stopRecording();
    });

    await waitFor(() => expect(result.current.state.status).toBe("result"));
    expect(result.current.state.currentResult).toMatchObject({ score: 0.9, isMatch: true });
    expect(result.current.state.history).toHaveLength(1);
  });

  it("reports a non-match result", async () => {
    const { result } = renderHook(() =>
      useWakeWordTest(baseOpts({ engine: makeEngine({ score: 0.2, isMatch: false }) })),
    );
    await act(async () => {
      result.current.startRecording();
    });
    await waitFor(() => expect(result.current.state.status).toBe("recording"));
    pushChunk(lastRecorder());
    nowVal = 1_000;
    await act(async () => {
      result.current.stopRecording();
    });
    await waitFor(() => expect(result.current.state.status).toBe("result"));
    expect(result.current.state.currentResult).toMatchObject({ isMatch: false });
  });

  it("rejects recordings that are too short", async () => {
    const { result } = renderHook(() => useWakeWordTest(baseOpts()));
    await act(async () => {
      result.current.startRecording();
    });
    await waitFor(() => expect(result.current.state.status).toBe("recording"));
    pushChunk(lastRecorder());
    nowVal = 100; // < MIN_DURATION_MS (500)
    await act(async () => {
      result.current.stopRecording();
    });
    await waitFor(() => expect(result.current.state.error).toMatch(/at least 0.5s/));
    expect(result.current.state.status).toBe("idle");
  });

  it("rejects empty recordings", async () => {
    const { result } = renderHook(() => useWakeWordTest(baseOpts()));
    await act(async () => {
      result.current.startRecording();
    });
    await waitFor(() => expect(result.current.state.status).toBe("recording"));
    // No chunks pushed → blob.size === 0.
    nowVal = 1_000;
    await act(async () => {
      result.current.stopRecording();
    });
    await waitFor(() => expect(result.current.state.error).toMatch(/empty/));
    expect(result.current.state.status).toBe("idle");
  });

  it("surfaces a comparison failure", async () => {
    decodeShouldReject = true;
    const { result } = renderHook(() => useWakeWordTest(baseOpts()));
    await act(async () => {
      result.current.startRecording();
    });
    await waitFor(() => expect(result.current.state.status).toBe("recording"));
    pushChunk(lastRecorder());
    nowVal = 1_000;
    await act(async () => {
      result.current.stopRecording();
    });
    await waitFor(() => expect(result.current.state.error).toMatch(/Comparison failed/));
    expect(result.current.state.status).toBe("idle");
  });

  it("surfaces a mic-access failure", async () => {
    getUserMediaImpl = () => Promise.reject(new Error("denied"));
    const { result } = renderHook(() => useWakeWordTest(baseOpts()));
    await act(async () => {
      result.current.startRecording();
    });
    await waitFor(() => expect(result.current.state.error).toMatch(/Mic access failed/));
    expect(result.current.state.status).toBe("idle");
  });

  it("stopRecording is a no-op when not recording", () => {
    const { result } = renderHook(() => useWakeWordTest(baseOpts()));
    expect(() => {
      act(() => result.current.stopRecording());
    }).not.toThrow();
    expect(result.current.state.status).toBe("idle");
  });

  it("clears history and current result", async () => {
    const { result } = renderHook(() => useWakeWordTest(baseOpts()));
    await act(async () => {
      result.current.startRecording();
    });
    await waitFor(() => expect(result.current.state.status).toBe("recording"));
    pushChunk(lastRecorder());
    nowVal = 1_000;
    await act(async () => {
      result.current.stopRecording();
    });
    await waitFor(() => expect(result.current.state.history).toHaveLength(1));

    act(() => result.current.clearHistory());
    expect(result.current.state.history).toEqual([]);
    expect(result.current.state.currentResult).toBeNull();
    expect(result.current.state.error).toBeNull();
  });

  it("ticks the recording-seconds counter and auto-stops", async () => {
    vi.useFakeTimers();
    try {
      const { result } = renderHook(() => useWakeWordTest(baseOpts()));
      await act(async () => {
        result.current.startRecording();
        await vi.advanceTimersByTimeAsync(0);
      });
      expect(result.current.state.status).toBe("recording");

      // Ticker increments recordingSeconds every 1s.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(1_000);
      });
      expect(result.current.state.recordingSeconds).toBe(1);

      pushChunk(lastRecorder());
      nowVal = 4_000;
      // Auto-stop fires at MAX_DURATION_MS (3000ms total).
      await act(async () => {
        await vi.advanceTimersByTimeAsync(3_000);
      });
      await vi.waitFor(() => expect(result.current.state.status).toBe("result"));
    } finally {
      vi.useRealTimers();
    }
  });
});

/** Return the most recently constructed FakeMediaRecorder. */
function lastRecorder(): FakeMediaRecorder {
  const all = FakeMediaRecorder.instances;
  return all[all.length - 1]!;
}
