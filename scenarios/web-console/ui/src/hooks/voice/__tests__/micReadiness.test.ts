import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  acquireStream,
  releaseStream,
  getStream,
  isStreamAlive,
  getMicReadinessState,
  installVisibilityHandler,
  _resetMicReadiness,
} from "../micReadiness";

// --- Mock getUserMedia ---

function createMockStream() {
  const track = {
    readyState: "live" as MediaStreamTrack["readyState"],
    stop: vi.fn(() => {
      track.readyState = "ended";
    }),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  };
  return {
    stream: { getTracks: () => [track] } as unknown as MediaStream,
    track,
  };
}

let mockStreamFactory: ReturnType<typeof createMockStream>;

beforeEach(() => {
  mockStreamFactory = createMockStream();
  Object.defineProperty(navigator, "mediaDevices", {
    value: {
      getUserMedia: vi.fn().mockResolvedValue(mockStreamFactory.stream),
    },
    configurable: true,
    writable: true,
  });
});

afterEach(() => {
  _resetMicReadiness();
  vi.restoreAllMocks();
});

describe("micReadiness", () => {
  it("acquireStream calls getUserMedia and returns a stream", async () => {
    const stream = await acquireStream();
    expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalledWith({ audio: true });
    expect(stream).toBe(mockStreamFactory.stream);
  });

  it("second call to acquireStream reuses the existing stream (no second getUserMedia)", async () => {
    await acquireStream();
    await acquireStream();
    expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalledTimes(1);
  });

  it("releaseStream stops all tracks and getStream returns null", async () => {
    await acquireStream();
    releaseStream();
    expect(mockStreamFactory.track.stop).toHaveBeenCalled();
    expect(getStream()).toBeNull();
  });

  it("isStreamAlive returns true for live tracks, false after release", async () => {
    await acquireStream();
    expect(isStreamAlive()).toBe(true);

    releaseStream();
    expect(isStreamAlive()).toBe(false);
  });

  it("getMicReadinessState transitions: idle -> acquiring -> warm -> released", async () => {
    expect(getMicReadinessState()).toBe("idle");

    // Capture state during acquisition by making getUserMedia controllable
    let resolveGUM!: (stream: MediaStream) => void;
    const gumPromise = new Promise<MediaStream>((resolve) => {
      resolveGUM = resolve;
    });
    vi.mocked(navigator.mediaDevices.getUserMedia).mockReturnValue(gumPromise);

    const acquirePromise = acquireStream();
    expect(getMicReadinessState()).toBe("acquiring");

    resolveGUM(mockStreamFactory.stream);
    await acquirePromise;
    expect(getMicReadinessState()).toBe("warm");

    releaseStream();
    expect(getMicReadinessState()).toBe("released");
  });

  it("acquireStream after releaseStream calls getUserMedia again (re-acquisition)", async () => {
    await acquireStream();
    releaseStream();

    // Create a fresh mock stream for re-acquisition
    const newMock = createMockStream();
    vi.mocked(navigator.mediaDevices.getUserMedia).mockResolvedValue(newMock.stream);

    const stream = await acquireStream();
    expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalledTimes(2);
    expect(stream).toBe(newMock.stream);
  });

  it("track ended event transitions state to released", async () => {
    await acquireStream();
    expect(getMicReadinessState()).toBe("warm");

    // The module registers an "ended" listener on each track.
    // Simulate the track ending by invoking the registered callback.
    const addEventCall = mockStreamFactory.track.addEventListener.mock.calls.find(
      (call: unknown[]) => call[0] === "ended",
    );
    expect(addEventCall).toBeDefined();
    if (!addEventCall) throw new Error("ended listener not registered");

    // Invoke the handler
    const endedHandler = addEventCall[1] as () => void;
    mockStreamFactory.track.readyState = "ended";
    endedHandler();

    expect(getMicReadinessState()).toBe("released");
    expect(getStream()).toBeNull();
  });
});

describe("installVisibilityHandler", () => {
  it("hidden releases stream when not recording", async () => {
    await acquireStream();
    expect(getStream()).not.toBeNull();

    const cleanup = installVisibilityHandler({
      isRecordingActive: () => false,
      isLowLatencyEnabled: () => false,
    });

    Object.defineProperty(document, "visibilityState", { value: "hidden", configurable: true });
    document.dispatchEvent(new Event("visibilitychange"));

    expect(getStream()).toBeNull();
    expect(getMicReadinessState()).toBe("released");

    cleanup();
    Object.defineProperty(document, "visibilityState", { value: "visible", configurable: true });
  });

  it("hidden does NOT release during active recording", async () => {
    await acquireStream();

    const cleanup = installVisibilityHandler({
      isRecordingActive: () => true,
      isLowLatencyEnabled: () => false,
    });

    Object.defineProperty(document, "visibilityState", { value: "hidden", configurable: true });
    document.dispatchEvent(new Event("visibilitychange"));

    // Stream should still be alive since recording is active
    expect(getStream()).not.toBeNull();
    expect(getMicReadinessState()).toBe("warm");

    cleanup();
    Object.defineProperty(document, "visibilityState", { value: "visible", configurable: true });
  });

  it("visible re-acquires when low-latency enabled", async () => {
    await acquireStream();
    releaseStream();
    expect(getStream()).toBeNull();

    // Prepare a new stream for re-acquisition
    const newMock = createMockStream();
    vi.mocked(navigator.mediaDevices.getUserMedia).mockResolvedValue(newMock.stream);

    const cleanup = installVisibilityHandler({
      isRecordingActive: () => false,
      isLowLatencyEnabled: () => true,
    });

    Object.defineProperty(document, "visibilityState", { value: "visible", configurable: true });
    document.dispatchEvent(new Event("visibilitychange"));

    // acquireStream is async, so wait for it
    await vi.waitFor(() => {
      expect(getStream()).not.toBeNull();
    });

    cleanup();
  });

  it("cleanup function removes the visibility listener", async () => {
    const removeSpy = vi.spyOn(document, "removeEventListener");

    const cleanup = installVisibilityHandler({
      isRecordingActive: () => false,
      isLowLatencyEnabled: () => false,
    });

    cleanup();

    const removeCall = removeSpy.mock.calls.find(
      ([evt]) => evt === "visibilitychange",
    );
    expect(removeCall).toBeDefined();

    removeSpy.mockRestore();
  });
});
