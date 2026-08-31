import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";

import { _resetMicOwnershipForTesting, getActiveMicLeases, installMicLifecycleCleanup } from "../micOwnership";
import { useWakeWordTest } from "./useWakeWordTest";
import type { AudioFeatures, WakeWordEngine } from "./types";

class FakeMediaRecorder {
  state: "inactive" | "recording" = "inactive";
  ondataavailable: ((event: { data: { size: number } }) => void) | null = null;
  onstop: (() => void) | null = null;
  onerror: (() => void) | null = null;
  constructor(public stream: MediaStream) {}
  start(): void { this.state = "recording"; }
  stop(): void { this.state = "inactive"; this.onstop?.(); }
  static isTypeSupported(): boolean { return true; }
}

const engine: WakeWordEngine = { extractFeatures: vi.fn(), compare: vi.fn(), compareBest: vi.fn(), calibrate: vi.fn(() => null) } as unknown as WakeWordEngine;
const samples = [{} as AudioFeatures];

function installMicrophone() {
  const track = { readyState: "live" as MediaStreamTrackState, muted: false, kind: "audio", stop: vi.fn(), addEventListener: vi.fn(), removeEventListener: vi.fn() };
  const stream = { getTracks: () => [track] } as unknown as MediaStream;
  Object.defineProperty(navigator, "mediaDevices", { configurable: true, value: { getUserMedia: vi.fn().mockResolvedValue(stream) } });
  return track.stop;
}

describe("useWakeWordTest lifecycle mic ownership", () => {
  beforeEach(() => {
    vi.stubGlobal("MediaRecorder", FakeMediaRecorder);
    vi.stubGlobal("requestAnimationFrame", vi.fn(() => 1));
    vi.stubGlobal("cancelAnimationFrame", vi.fn());
  });
  afterEach(() => {
    _resetMicOwnershipForTesting();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    Object.defineProperty(document, "visibilityState", { configurable: true, value: "visible" });
  });

  it("cancels a recording and releases the lease when the tab is hidden", async () => {
    const stop = installMicrophone();
    const uninstall = installMicLifecycleCleanup();
    const { result } = renderHook(() => useWakeWordTest({ engine, samples, threshold: 0.5, disabled: false }));
    await act(async () => { result.current.startRecording(); await Promise.resolve(); });
    expect(result.current.state.status).toBe("recording");
    expect(getActiveMicLeases().map((lease) => lease.owner)).toEqual(["wake-word-test"]);
    Object.defineProperty(document, "visibilityState", { configurable: true, value: "hidden" });
    await act(async () => { document.dispatchEvent(new Event("visibilitychange")); await Promise.resolve(); });
    expect(stop).toHaveBeenCalledTimes(1);
    expect(getActiveMicLeases()).toHaveLength(0);
    expect(result.current.state.currentResult).toBeNull();
    uninstall();
  });

  it("releases the mic lease on unmount", async () => {
    const stop = installMicrophone();
    const { result, unmount } = renderHook(() => useWakeWordTest({ engine, samples, threshold: 0.5, disabled: false }));
    await act(async () => { result.current.startRecording(); await Promise.resolve(); });
    expect(getActiveMicLeases()).toHaveLength(1);
    unmount();
    expect(getActiveMicLeases()).toHaveLength(0);
    expect(stop).toHaveBeenCalled();
  });
});
