import { afterEach, describe, expect, it, vi } from "vitest";
import {
  _resetMicReadiness,
  acquireStream,
  getMicReadinessState,
  getStream,
  releaseStream,
} from "./micReadiness";

function installGetUserMediaMock() {
  const stop = vi.fn();
  const track = {
    readyState: "live",
    stop,
    addEventListener: vi.fn(),
  };
  const stream = {
    getTracks: () => [track],
  } as unknown as MediaStream;

  const controls: {
    resolveGetUserMedia?: (stream: MediaStream) => void;
  } = {};
  const getUserMedia = vi.fn(() => new Promise<MediaStream>((resolve) => {
    controls.resolveGetUserMedia = resolve;
  }));

  Object.defineProperty(navigator, "mediaDevices", {
    configurable: true,
    value: { getUserMedia },
  });

  return { getUserMedia, controls, stream, stop };
}

describe("micReadiness", () => {
  afterEach(() => {
    _resetMicReadiness();
    vi.restoreAllMocks();
  });

  it("stores a pre-warmed stream when acquisition completes", async () => {
    const { getUserMedia, controls, stream } = installGetUserMediaMock();

    const pending = acquireStream();
    expect(getUserMedia).toHaveBeenCalledWith({ audio: true });

    controls.resolveGetUserMedia?.(stream);

    await expect(pending).resolves.toBe(stream);
    expect(getStream()).toBe(stream);
    expect(getMicReadinessState()).toBe("warm");
  });

  it("stops and discards a stream when release wins the getUserMedia race", async () => {
    const { controls, stream, stop } = installGetUserMediaMock();

    const pending = acquireStream();
    releaseStream();
    controls.resolveGetUserMedia?.(stream);

    await expect(pending).rejects.toThrow("Microphone pre-warm cancelled");
    expect(stop).toHaveBeenCalledTimes(1);
    expect(getStream()).toBeNull();
    expect(getMicReadinessState()).toBe("released");
  });
});
