import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { apiBaseMock } from "../../test-utils";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { AUDIO_BITRATE, computeFinalTimeout, createAudioFilterChain, computeSlidingNoiseFloor } from "../useVoiceInput";

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
