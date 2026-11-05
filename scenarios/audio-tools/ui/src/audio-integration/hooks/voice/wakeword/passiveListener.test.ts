// vi.fn() spies are accessed by reference for `toHaveBeenCalled` assertions;
// the unbound-method rule's `this`-scoping concern does not apply to them.
/* eslint-disable @typescript-eslint/unbound-method */
// Tests for PassiveListener — the background wake-word listening loop.
// The audio plumbing (ring buffer, capture pipeline, downsample) and VAD are
// mocked so the listener's own control flow (start/stop/dispose, the RAF tick,
// speech onset, capture completion, match vs no-match) is exercised directly.
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { makeMediaStream } from "../../../test-support/browser";

import { PassiveListener } from "./passiveListener";
import type { PassiveListenerOpts, PassiveListenerSeams } from "./passiveListener";
import type { AudioFeatures, MatchResult, WakeWordEngine, WakeWordTemplate } from "./types";

// ── Mock the audio + VAD seams ──────────────────────────────────────────────

interface FakeVad {
  state: string;
  recordingStart: number;
  segmentBoundaryEmitted: boolean;
}

const vadState: FakeVad = { state: "calibrating", recordingStart: 0, segmentBoundaryEmitted: false };
// Programmable per-tick behaviour: sets vad.state and returns the VAD action.
let vadTickImpl: (vad: FakeVad) => string | null = () => null;

vi.mock("../vad", () => ({
  createPassiveVadRefs: () => {
    vadState.state = "calibrating";
    vadState.recordingStart = 0;
    vadState.segmentBoundaryEmitted = false;
    return vadState;
  },
  vadTick: vi.fn((vad: FakeVad) => vadTickImpl(vad)),
}));

let ringMark = 1_000;
let extractResult = new Float32Array(800);

vi.mock("../audioUtils", () => ({
  AudioRingBuffer: class {
    sampleRate: number;
    constructor(_seconds: number, sampleRate: number) {
      this.sampleRate = sampleRate;
    }
    mark() {
      return ringMark;
    }
    extractSinceMark(_from: number) {
      return extractResult;
    }
  },
  createPassiveCapturePipeline: () => ({
    analyser: {
      fftSize: 2048,
      getFloatTimeDomainData: (arr: Float32Array) => arr.fill(0.5),
    },
    nodes: [{ disconnect: vi.fn() }],
  }),
  downsample: (buf: Float32Array) => buf,
}));

// ── Browser fakes ───────────────────────────────────────────────────────────

let rafCb: (() => void) | null = null;
let nowVal = 0;

class FakeAudioContext {
  state = "running";
  sampleRate = 16_000;
  createMediaStreamSource = vi.fn(() => ({ connect: vi.fn(), disconnect: vi.fn() }));
  resume = vi.fn(() => Promise.resolve());
  close = vi.fn(() => Promise.resolve());
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

function makeTemplate(): WakeWordTemplate {
  return {
    samples: [{ kind: "mfcc-v1", data: [], sampleRate: 16_000, durationSec: 1 }],
    label: "hey",
    threshold: 0.65,
    updatedAt: new Date(0).toISOString(),
  };
}

const testSeams: PassiveListenerSeams = {
  createRingBuffer: (_seconds, sampleRate) => ({
    sampleRate,
    mark: () => ringMark,
    extractSinceMark: () => extractResult,
  } as unknown as ReturnType<PassiveListenerSeams["createRingBuffer"]>),
  createCapturePipeline: (() => ({
    analyser: {
      fftSize: 2048,
      getFloatTimeDomainData: (arr: Float32Array) => arr.fill(0.5),
    },
    nodes: [{ disconnect: vi.fn() }],
  })) as unknown as PassiveListenerSeams["createCapturePipeline"],
  downsample: (buf) => buf,
  createVadRefs: () => vadState as unknown as ReturnType<PassiveListenerSeams["createVadRefs"]>,
  vadTick: (vad) => vadTickImpl(vad as unknown as FakeVad) as unknown as ReturnType<PassiveListenerSeams["vadTick"]>,
};

function makeListener(options: Omit<PassiveListenerOpts, "seams">): PassiveListener {
  return new PassiveListener({ ...options, seams: testSeams });
}

beforeEach(() => {
  rafCb = null;
  nowVal = 0;
  ringMark = 1_000;
  extractResult = new Float32Array(800);
  vadTickImpl = () => null;
  vadState.state = "calibrating";
  getUserMediaImpl = () => Promise.resolve(makeMediaStream());
  vi.spyOn(performance, "now").mockImplementation(() => nowVal);
  (globalThis as unknown as { requestAnimationFrame: (cb: () => void) => number }).requestAnimationFrame = (cb) => {
    rafCb = cb;
    return 1;
  };
  (globalThis as unknown as { cancelAnimationFrame: (id: number) => void }).cancelAnimationFrame = vi.fn();
  (globalThis as unknown as { AudioContext: typeof FakeAudioContext }).AudioContext = FakeAudioContext;
  Object.defineProperty(navigator, "mediaDevices", {
    configurable: true,
    value: { getUserMedia: vi.fn(() => getUserMediaImpl()) },
  });
});

afterEach(() => {
  vi.restoreAllMocks();
});

/** Run the most recently scheduled RAF callback once. */
function runTick(): void {
  const cb = rafCb;
  rafCb = null;
  cb?.();
}

describe("PassiveListener", () => {
  it("acquires the mic and starts the loop on start()", async () => {
    const listener = makeListener({
      engine: makeEngine({ score: 0.9, isMatch: true }),
      template: makeTemplate(),
      onWakeWordDetected: vi.fn(),
      onError: vi.fn(),
    });
    await listener.start();
    expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalled();
    expect(listener.getStream()).not.toBeNull();
    expect(listener.getAudioContext()).not.toBeNull();
    listener.dispose();
  });

  it("reuses an injected AudioContext instead of creating one", async () => {
    const ctx = new FakeAudioContext() as unknown as AudioContext;
    const listener = makeListener({
      engine: makeEngine({ score: 0.9, isMatch: true }),
      template: makeTemplate(),
      onWakeWordDetected: vi.fn(),
      onError: vi.fn(),
      audioContext: ctx,
    });
    await listener.start();
    expect(listener.getAudioContext()).toBe(ctx);
    // Injected context is not owned → dispose must not close it.
    listener.dispose();
    expect((ctx as unknown as FakeAudioContext).close).not.toHaveBeenCalled();
  });

  it("resumes a suspended AudioContext", async () => {
    const ctx = new FakeAudioContext();
    ctx.state = "suspended";
    const listener = makeListener({
      engine: makeEngine({ score: 0.9, isMatch: true }),
      template: makeTemplate(),
      onWakeWordDetected: vi.fn(),
      onError: vi.fn(),
      audioContext: ctx as unknown as AudioContext,
    });
    await listener.start();
    expect(ctx.resume).toHaveBeenCalled();
    listener.dispose();
  });

  it("reports an error when getUserMedia fails", async () => {
    getUserMediaImpl = () => Promise.reject(new Error("denied"));
    const onError = vi.fn();
    const listener = makeListener({
      engine: makeEngine({ score: 0.9, isMatch: true }),
      template: makeTemplate(),
      onWakeWordDetected: vi.fn(),
      onError,
    });
    await listener.start();
    expect(onError).toHaveBeenCalledWith(expect.stringContaining("Passive listener failed to start"));
    expect(listener.getStream()).toBeNull();
  });

  it("start() is idempotent while already running", async () => {
    const listener = makeListener({
      engine: makeEngine({ score: 0.9, isMatch: true }),
      template: makeTemplate(),
      onWakeWordDetected: vi.fn(),
      onError: vi.fn(),
    });
    await listener.start();
    const calls = (navigator.mediaDevices.getUserMedia as ReturnType<typeof vi.fn>).mock.calls.length;
    await listener.start();
    expect((navigator.mediaDevices.getUserMedia as ReturnType<typeof vi.fn>).mock.calls.length).toBe(calls);
    listener.dispose();
  });

  it("fires onWakeWordDetected when a captured segment matches", async () => {
    const onWakeWordDetected = vi.fn();
    const engine = makeEngine({ score: 0.95, isMatch: true });
    const listener = makeListener({
      engine,
      template: makeTemplate(),
      onWakeWordDetected,
      onError: vi.fn(),
    });
    await listener.start();

    // Tick 1: speech onset (waitingForSpeech → speechDetected).
    nowVal = 1_000;
    vadState.state = "waitingForSpeech";
    vadTickImpl = (vad) => {
      vad.state = "speechDetected";
      return null;
    };
    runTick();

    // Tick 2: VAD says stop with a long-enough speech duration → capture+match.
    nowVal = 2_000; // 1000ms speech > MIN_SPEECH_DURATION_MS
    vadTickImpl = () => "stop";
    runTick();

    expect(engine.extractFeatures).toHaveBeenCalled();
    expect(engine.compareBest).toHaveBeenCalled();
    expect(onWakeWordDetected).toHaveBeenCalledTimes(1);
    listener.dispose();
  });

  it("does not fire on a failed match and records a debounce time", async () => {
    const onWakeWordDetected = vi.fn();
    const engine = makeEngine({ score: 0.1, isMatch: false });
    const listener = makeListener({
      engine,
      template: makeTemplate(),
      onWakeWordDetected,
      onError: vi.fn(),
    });
    await listener.start();

    nowVal = 1_000;
    vadState.state = "waitingForSpeech";
    vadTickImpl = (vad) => {
      vad.state = "speechDetected";
      return null;
    };
    runTick();

    nowVal = 2_000;
    vadTickImpl = () => "stop";
    runTick();

    expect(engine.compareBest).toHaveBeenCalled();
    expect(onWakeWordDetected).not.toHaveBeenCalled();
    listener.dispose();
  });

  it("skips capture when the speech segment is too short", async () => {
    const engine = makeEngine({ score: 0.95, isMatch: true });
    const onWakeWordDetected = vi.fn();
    const listener = makeListener({
      engine,
      template: makeTemplate(),
      onWakeWordDetected,
      onError: vi.fn(),
    });
    await listener.start();

    nowVal = 1_000;
    vadState.state = "waitingForSpeech";
    vadTickImpl = (vad) => {
      vad.state = "speechDetected";
      return null;
    };
    runTick();

    // Only 100ms of speech (< MIN_SPEECH_DURATION_MS) → no feature extraction.
    nowVal = 1_100;
    vadTickImpl = () => "stop";
    runTick();

    expect(engine.extractFeatures).not.toHaveBeenCalled();
    expect(onWakeWordDetected).not.toHaveBeenCalled();
    listener.dispose();
  });

  it("throttles ticks that arrive within the throttle window", async () => {
    const engine = makeEngine({ score: 0.95, isMatch: true });
    const listener = makeListener({
      engine,
      template: makeTemplate(),
      onWakeWordDetected: vi.fn(),
      onError: vi.fn(),
    });
    await listener.start();
    // The very first tick set lastTickTime; an immediate follow-up within
    // TICK_THROTTLE_MS must early-return without running VAD.
    const tickFn = vi.mocked((await import("../vad")).vadTick);
    const before = tickFn.mock.calls.length;
    nowVal = 1; // < 66ms since last
    runTick();
    expect(tickFn.mock.calls.length).toBe(before);
    listener.dispose();
  });

  it("stop() halts the loop and dispose() tears down stream + owned context", async () => {
    const trackStop = vi.fn();
    getUserMediaImpl = () => Promise.resolve(makeMediaStream("live", trackStop));
    const listener = makeListener({
      engine: makeEngine({ score: 0.9, isMatch: true }),
      template: makeTemplate(),
      onWakeWordDetected: vi.fn(),
      onError: vi.fn(),
    });
    await listener.start();
    const ownedCtx = listener.getAudioContext() as unknown as FakeAudioContext;

    listener.stop();
    // After stop, a scheduled tick must no-op (running=false).
    expect(() => runTick()).not.toThrow();

    listener.dispose();
    expect(trackStop).toHaveBeenCalled();
    expect(ownedCtx.close).toHaveBeenCalled(); // owned context is closed
    expect(listener.getStream()).toBeNull();
  });

  it("ignores empty captured audio", async () => {
    extractResult = new Float32Array(0);
    const engine = makeEngine({ score: 0.95, isMatch: true });
    const onWakeWordDetected = vi.fn();
    const listener = makeListener({
      engine,
      template: makeTemplate(),
      onWakeWordDetected,
      onError: vi.fn(),
    });
    await listener.start();

    nowVal = 1_000;
    vadState.state = "waitingForSpeech";
    vadTickImpl = (vad) => {
      vad.state = "speechDetected";
      return null;
    };
    runTick();

    nowVal = 2_000;
    vadTickImpl = () => "stop";
    runTick();

    expect(engine.extractFeatures).not.toHaveBeenCalled();
    listener.dispose();
  });
});
