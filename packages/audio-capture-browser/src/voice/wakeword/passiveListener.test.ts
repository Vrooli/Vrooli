import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { _resetMicOwnershipForTesting, getActiveMicLeases } from "../micOwnership";
import { PassiveListener, type PassiveListenerSeams } from "./passiveListener";
import type { WakeWordEngine, WakeWordTemplate } from "./types";

vi.mock("../audioUtils", () => ({
  AudioRingBuffer: class {
    sampleRate: number;
    constructor(_seconds: number, sampleRate: number) { this.sampleRate = sampleRate; }
    mark() { return 0; }
    extractSinceMark() { return new Float32Array(0); }
  },
  createPassiveCapturePipeline: () => ({ analyser: { fftSize: 2048, getFloatTimeDomainData: () => {} }, nodes: [] }),
  downsample: (samples: Float32Array) => samples,
}));
vi.mock("../vad", () => ({
  createPassiveVadRefs: () => ({ state: "calibrating", recordingStart: 0 }),
  vadTick: () => null,
}));

function fakeAudioContext(throwOnSource = false) {
  return {
    state: "running" as AudioContextState,
    sampleRate: 48_000,
    resume: vi.fn().mockResolvedValue(undefined),
    close: vi.fn().mockResolvedValue(undefined),
    createMediaStreamSource: vi.fn(() => {
      if (throwOnSource) throw new Error("createMediaStreamSource failed");
      return { connect: vi.fn(), disconnect: vi.fn() };
    }),
  };
}

function installGetUserMedia() {
  const track = {
    readyState: "live" as MediaStreamTrackState,
    muted: false,
    kind: "audio",
    stop: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  };
  const stream = { getTracks: () => [track] } as unknown as MediaStream;
  const getUserMedia = vi.fn().mockResolvedValue(stream);
  Object.defineProperty(navigator, "mediaDevices", { configurable: true, value: { getUserMedia } });
  return { getUserMedia, stop: track.stop };
}

const seams: PassiveListenerSeams = {
  createRingBuffer: (_seconds, sampleRate) => ({ sampleRate, mark: () => 0, extractSinceMark: () => new Float32Array(0) } as never),
  createCapturePipeline: () => ({ analyser: { fftSize: 2048, getFloatTimeDomainData: () => {} } as never, nodes: [] }),
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
const template = { samples: [], label: "Hey Test", threshold: 0.5, updatedAt: 0, calibration: null } as unknown as WakeWordTemplate;

describe("package-owned PassiveListener mic lifecycle", () => {
  beforeEach(() => {
    vi.stubGlobal("AudioContext", vi.fn(() => fakeAudioContext()));
    vi.stubGlobal("requestAnimationFrame", vi.fn(() => 1));
    vi.stubGlobal("cancelAnimationFrame", vi.fn());
  });
  afterEach(() => {
    _resetMicOwnershipForTesting();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("acquires and releases its passive lease", async () => {
    installGetUserMedia();
    const onMicReleased = vi.fn();
    const listener = new PassiveListener({
      engine, template, onWakeWordDetected: vi.fn(), onError: vi.fn(), onMicReleased,
      audioContext: fakeAudioContext() as unknown as AudioContext, seams,
    });

    await listener.start();
    expect(getActiveMicLeases().map((lease) => lease.owner)).toEqual(["passive-wake-word"]);

    listener.dispose("toggle-off");
    expect(getActiveMicLeases()).toHaveLength(0);
    expect(onMicReleased).toHaveBeenCalledWith("toggle-off");
  });

  it("releases a stream if audio setup fails after getUserMedia", async () => {
    const { stop } = installGetUserMedia();
    const onError = vi.fn();
    const listener = new PassiveListener({
      engine, template, onWakeWordDetected: vi.fn(), onError, seams,
      audioContext: fakeAudioContext(true) as unknown as AudioContext,
    });

    await listener.start();
    expect(stop).toHaveBeenCalledTimes(1);
    expect(getActiveMicLeases()).toHaveLength(0);
    expect(onError).toHaveBeenCalledTimes(1);
  });

  it("reports denied microphone access without retaining a lease", async () => {
    const getUserMedia = vi.fn().mockRejectedValue(new Error("Permission denied"));
    Object.defineProperty(navigator, "mediaDevices", { configurable: true, value: { getUserMedia } });
    const onError = vi.fn();
    const listener = new PassiveListener({ engine, template, onWakeWordDetected: vi.fn(), onError, seams });

    await listener.start();
    expect(getActiveMicLeases()).toHaveLength(0);
    expect(onError).toHaveBeenCalledTimes(1);
  });

  it("keeps one passive lease across an accelerated hour of idle listening", async () => {
    installGetUserMedia();
    let now = 0;
    vi.spyOn(performance, "now").mockImplementation(() => {
      now += 1_000;
      return now;
    });
    const frames: FrameRequestCallback[] = [];
    vi.stubGlobal("requestAnimationFrame", vi.fn((callback: FrameRequestCallback) => {
      frames.push(callback);
      return frames.length;
    }));

    const listener = new PassiveListener({
      engine, template, onWakeWordDetected: vi.fn(), onError: vi.fn(), seams,
      audioContext: fakeAudioContext() as unknown as AudioContext,
    });
    await listener.start();

    for (let second = 0; second < 3_600; second += 1) {
      const frame = frames.shift();
      expect(frame).toBeDefined();
      frame?.(now);
    }

    expect(getActiveMicLeases()).toHaveLength(1);
    expect(getActiveMicLeases()[0]?.owner).toBe("passive-wake-word");
    listener.dispose("unmount");
    expect(getActiveMicLeases()).toHaveLength(0);
  });
});
