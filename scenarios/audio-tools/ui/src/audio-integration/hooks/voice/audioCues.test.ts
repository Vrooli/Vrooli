// Unit tests for audioCues.ts (playRecordingStartCue / playRecordingStopCue).
//
// Both exported functions call `void playChime(freq1, freq2)` — fire-and-forget async.
// We mock `./sharedAudioContext` to return a fake AudioContext and verify
// that the oscillator nodes are created with the correct frequencies and that
// the gain automation API is invoked.

import { beforeEach, afterEach, describe, expect, it, vi } from "vitest";

// Mock sharedAudioContext before importing audioCues
vi.mock("./sharedAudioContext", () => ({
  getSharedAudioContext: vi.fn(),
}));

import { getSharedAudioContext } from "./sharedAudioContext";
import { playRecordingStartCue, playRecordingStopCue } from "./audioCues";

// ---------------------------------------------------------------------------
// Fake AudioContext
// ---------------------------------------------------------------------------

function makeGainNode() {
  return {
    gain: {
      value: 1,
      setValueAtTime: vi.fn(),
      linearRampToValueAtTime: vi.fn(),
      exponentialRampToValueAtTime: vi.fn(),
    },
    connect: vi.fn().mockReturnThis(),
    disconnect: vi.fn(),
  };
}

function makeOscillatorNode() {
  return {
    type: "",
    frequency: { value: 0, setValueAtTime: vi.fn() },
    connect: vi.fn().mockReturnThis(),
    disconnect: vi.fn(),
    start: vi.fn(),
    stop: vi.fn(),
  };
}

interface FakeContext {
  state: AudioContextState;
  currentTime: number;
  destination: object;
  createGain: ReturnType<typeof vi.fn>;
  createOscillator: ReturnType<typeof vi.fn>;
  resume: ReturnType<typeof vi.fn>;
  close: ReturnType<typeof vi.fn>;
}

function makeFakeContext(state: AudioContextState = "running"): FakeContext {
  return {
    state,
    currentTime: 0,
    destination: {},
    createGain: vi.fn().mockImplementation(makeGainNode),
    createOscillator: vi.fn().mockImplementation(makeOscillatorNode),
    resume: vi.fn().mockResolvedValue(undefined),
    close: vi.fn().mockResolvedValue(undefined),
  };
}

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

let fakeCtx: FakeContext;

beforeEach(() => {
  fakeCtx = makeFakeContext("running");
  vi.mocked(getSharedAudioContext).mockReturnValue(fakeCtx as unknown as AudioContext);
});

afterEach(() => {
  vi.clearAllMocks();
});

// ---------------------------------------------------------------------------
// playRecordingStartCue
// ---------------------------------------------------------------------------

describe("playRecordingStartCue", () => {
  it("does not throw", () => {
    expect(() => playRecordingStartCue()).not.toThrow();
  });

  it("creates two oscillators (one per note)", async () => {
    playRecordingStartCue();
    // Allow microtask queue to drain so playChime's async body runs
    await new Promise((r) => setTimeout(r, 0));
    expect(fakeCtx.createOscillator).toHaveBeenCalledTimes(2);
  });

  it("creates two gain nodes (one per note)", async () => {
    playRecordingStartCue();
    await new Promise((r) => setTimeout(r, 0));
    expect(fakeCtx.createGain).toHaveBeenCalledTimes(2);
  });

  it("sets oscillator frequencies to 523 Hz (C5) and 659 Hz (E5) — rising interval", async () => {
    playRecordingStartCue();
    await new Promise((r) => setTimeout(r, 0));
    const oscCalls = fakeCtx.createOscillator.mock.results;
    const freqs = oscCalls.map((r) => (r.value as ReturnType<typeof makeOscillatorNode>).frequency.value);
    expect(freqs).toContain(523);
    expect(freqs).toContain(659);
  });

  it("starts and schedules stop for each oscillator", async () => {
    playRecordingStartCue();
    await new Promise((r) => setTimeout(r, 0));
    for (const result of fakeCtx.createOscillator.mock.results) {
      const osc = result.value as ReturnType<typeof makeOscillatorNode>;
      expect(osc.start).toHaveBeenCalled();
      expect(osc.stop).toHaveBeenCalled();
    }
  });

  it("applies gain envelope (setValueAtTime called on each gain node)", async () => {
    playRecordingStartCue();
    await new Promise((r) => setTimeout(r, 0));
    for (const result of fakeCtx.createGain.mock.results) {
      const gain = result.value as ReturnType<typeof makeGainNode>;
      expect(gain.gain.setValueAtTime).toHaveBeenCalled();
    }
  });

  it("resumes context if suspended before playing", async () => {
    fakeCtx.state = "suspended";
    playRecordingStartCue();
    await new Promise((r) => setTimeout(r, 0));
    expect(fakeCtx.resume).toHaveBeenCalledOnce();
  });

  it("does not call resume when context is already running", async () => {
    fakeCtx.state = "running";
    playRecordingStartCue();
    await new Promise((r) => setTimeout(r, 0));
    expect(fakeCtx.resume).not.toHaveBeenCalled();
  });

  it("silently ignores errors from the AudioContext", async () => {
    vi.mocked(getSharedAudioContext).mockImplementation(() => {
      throw new Error("no audio");
    });
    // Should not throw (errors are swallowed by playChime's catch block)
    expect(() => playRecordingStartCue()).not.toThrow();
    await new Promise((r) => setTimeout(r, 0));
  });
});

// ---------------------------------------------------------------------------
// playRecordingStopCue
// ---------------------------------------------------------------------------

describe("playRecordingStopCue", () => {
  it("does not throw", () => {
    expect(() => playRecordingStopCue()).not.toThrow();
  });

  it("creates two oscillators", async () => {
    playRecordingStopCue();
    await new Promise((r) => setTimeout(r, 0));
    expect(fakeCtx.createOscillator).toHaveBeenCalledTimes(2);
  });

  it("sets frequencies to 659 Hz (E5) and 523 Hz (C5) — falling interval", async () => {
    playRecordingStopCue();
    await new Promise((r) => setTimeout(r, 0));
    const oscCalls = fakeCtx.createOscillator.mock.results;
    const freqs = oscCalls.map((r) => (r.value as ReturnType<typeof makeOscillatorNode>).frequency.value);
    expect(freqs).toContain(659);
    expect(freqs).toContain(523);
  });

  it("starts and stops each oscillator", async () => {
    playRecordingStopCue();
    await new Promise((r) => setTimeout(r, 0));
    for (const result of fakeCtx.createOscillator.mock.results) {
      const osc = result.value as ReturnType<typeof makeOscillatorNode>;
      expect(osc.start).toHaveBeenCalled();
      expect(osc.stop).toHaveBeenCalled();
    }
  });

  it("start cue uses reversed order (stop note first = 659 then 523)", async () => {
    // start = 523→659 (rising), stop = 659→523 (falling)
    // Verify stop cue: first oscillator freq=659, second=523
    playRecordingStopCue();
    await new Promise((r) => setTimeout(r, 0));
    const results = fakeCtx.createOscillator.mock.results;
    expect((results[0]!.value as ReturnType<typeof makeOscillatorNode>).frequency.value).toBe(659);
    expect((results[1]!.value as ReturnType<typeof makeOscillatorNode>).frequency.value).toBe(523);
  });
});

describe("start vs stop cue frequency ordering", () => {
  it("start cue: first note is 523 (C5), second is 659 (E5)", async () => {
    playRecordingStartCue();
    await new Promise((r) => setTimeout(r, 0));
    const results = fakeCtx.createOscillator.mock.results;
    expect((results[0]!.value as ReturnType<typeof makeOscillatorNode>).frequency.value).toBe(523);
    expect((results[1]!.value as ReturnType<typeof makeOscillatorNode>).frequency.value).toBe(659);
  });
});
