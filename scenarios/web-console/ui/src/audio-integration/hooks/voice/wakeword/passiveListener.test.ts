import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { _resetMicOwnershipForTesting, getActiveMicLeases } from "../micOwnership";
import { PassiveListener, type PassiveListenerSeams } from "./passiveListener";
import type { WakeWordEngine, WakeWordTemplate } from "./types";

// Keep the audio pipeline + VAD trivial — this suite is about mic-stream
// ownership, not detection math.
vi.mock("../audioUtils", () => ({
  AudioRingBuffer: class {
    sampleRate: number;
    constructor(_seconds: number, sampleRate: number) { this.sampleRate = sampleRate; }
    mark() { return 0; }
    extractSinceMark() { return new Float32Array(0); }
  },
  createPassiveCapturePipeline: () => ({
    analyser: { fftSize: 2048, getFloatTimeDomainData: () => {} },
    nodes: [],
  }),
  downsample: (x: Float32Array) => x,
}));
vi.mock("../vad", () => ({
  createPassiveVadRefs: () => ({ state: "calibrating", recordingStart: 0 }),
  vadTick: () => null,
}));

let throwOnCreateSource = false;

function fakeAudioContext() {
  return {
    state: "running" as AudioContextState,
    sampleRate: 48000,
    resume: vi.fn().mockResolvedValue(undefined),
    close: vi.fn().mockResolvedValue(undefined),
    createMediaStreamSource: vi.fn(() => {
      if (throwOnCreateSource) throw new Error("createMediaStreamSource failed");
      return { connect: vi.fn(), disconnect: vi.fn() };
    }),
  };
}

function fakeTrack() {
  return {
    readyState: "live" as MediaStreamTrackState,
    muted: false,
    kind: "audio",
    stop: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  };
}

function installGetUserMedia(): { getUserMedia: ReturnType<typeof vi.fn>; stop: ReturnType<typeof vi.fn> } {
  const track = fakeTrack();
  const stream = { getTracks: () => [track] } as unknown as MediaStream;
  const getUserMedia = vi.fn().mockResolvedValue(stream);
  Object.defineProperty(navigator, "mediaDevices", { configurable: true, value: { getUserMedia } });
  return { getUserMedia, stop: track.stop };
}

const testSeams: PassiveListenerSeams = {
  createRingBuffer: (_seconds, sampleRate) => ({
    sampleRate,
    mark: () => 0,
    extractSinceMark: () => new Float32Array(0),
  } as never),
  createCapturePipeline: () => ({
    analyser: { fftSize: 2048, getFloatTimeDomainData: () => {} } as never,
    nodes: [],
  }),
  downsample: (samples) => samples,
  createVadRefs: () => ({ state: "calibrating", recordingStart: 0 } as never),
  vadTick: () => null,
};

const engine: WakeWordEngine = {
  extractFeatures: vi.fn(),
  compare: vi.fn(),
  compareBest: vi.fn(),
  calibrate: vi.fn(),
} as unknown as WakeWordEngine;

const template: WakeWordTemplate = {
  samples: [],
  label: "Hey Test",
  threshold: 0.5,
  updatedAt: 0,
  calibration: null,
} as unknown as WakeWordTemplate;

describe("PassiveListener mic ownership", () => {
  beforeEach(() => {
    throwOnCreateSource = false;
    vi.stubGlobal("AudioContext", vi.fn(fakeAudioContext));
    vi.stubGlobal("requestAnimationFrame", vi.fn(() => 1));
    vi.stubGlobal("cancelAnimationFrame", vi.fn());
  });
  afterEach(() => {
    _resetMicOwnershipForTesting();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("registers a mic lease on successful start and releases it on dispose", async () => {
    installGetUserMedia();
    const onMicReleased = vi.fn();
    const listener = new PassiveListener({
      engine, template, onWakeWordDetected: vi.fn(), onError: vi.fn(),
      audioContext: fakeAudioContext() as unknown as AudioContext, onMicReleased, seams: testSeams,
    });

    await listener.start();
    const leases = getActiveMicLeases();
    expect(leases).toHaveLength(1);
    expect(leases[0]?.owner).toBe("passive-wake-word");

    listener.dispose("toggle-off");
    expect(getActiveMicLeases()).toHaveLength(0);
    expect(onMicReleased).toHaveBeenCalledWith("toggle-off");
  });

  it("releases the acquired stream when audio setup fails AFTER getUserMedia (no leak)", async () => {
    const { stop } = installGetUserMedia();
    throwOnCreateSource = true;
    const onError = vi.fn();
    const onMicReleased = vi.fn();
    // Force the listener to create its OWN context (so the throwing createMediaStreamSource runs).
    const listener = new PassiveListener({
      engine, template, onWakeWordDetected: vi.fn(), onError, onMicReleased, seams: testSeams,
    });

    await listener.start();

    // The historical leak: getUserMedia succeeded but setup threw, leaving the
    // mic track live forever. It must be stopped and no lease left active.
    expect(stop).toHaveBeenCalledTimes(1);
    expect(getActiveMicLeases()).toHaveLength(0);
    expect(onError).toHaveBeenCalledTimes(1);
    expect(onMicReleased).toHaveBeenCalledWith("setup-error");
  });

  it("reports an error and leaves no lease when getUserMedia itself is denied", async () => {
    const getUserMedia = vi.fn().mockRejectedValue(new Error("Permission denied"));
    Object.defineProperty(navigator, "mediaDevices", { configurable: true, value: { getUserMedia } });
    const onError = vi.fn();
    const listener = new PassiveListener({
      engine, template, onWakeWordDetected: vi.fn(), onError,
      audioContext: fakeAudioContext() as unknown as AudioContext, seams: testSeams,
    });

    await listener.start();
    expect(getActiveMicLeases()).toHaveLength(0);
    expect(onError).toHaveBeenCalledTimes(1);
  });
});
