// Unit tests for micReadiness.ts.
//
// Module state (module-level _stream and _state) is reset via
// _resetMicReadiness() in beforeEach. navigator.mediaDevices is
// patched per-test via Object.defineProperty.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  acquireStream,
  releaseStream,
  getStream,
  isStreamAlive,
  getMicReadinessState,
  installVisibilityHandler,
  _resetMicReadiness,
} from "./micReadiness";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeFakeTrack(readyState: "live" | "ended" = "live") {
  const listeners: Map<string, Array<(e: Event) => void>> = new Map();
  const track = {
    readyState,
    kind: "audio",
    stop: vi.fn(),
    addEventListener: vi.fn().mockImplementation(
      (event: string, fn: (e: Event) => void) => {
        const arr = listeners.get(event) ?? [];
        arr.push(fn);
        listeners.set(event, arr);
      },
    ),
    // Helper to fire an event on this track (used by tests)
    _emit: (event: string) => {
      for (const fn of listeners.get(event) ?? []) fn(new Event(event));
    },
  };
  return track;
}

function makeFakeStream(tracks: ReturnType<typeof makeFakeTrack>[] = [makeFakeTrack()]) {
  return {
    getTracks: () => tracks,
    getAudioTracks: () => tracks.filter((t) => t.kind === "audio"),
  } as unknown as MediaStream;
}

function stubGetUserMedia(impl: () => Promise<MediaStream>) {
  Object.defineProperty(navigator, "mediaDevices", {
    value: { getUserMedia: vi.fn().mockImplementation(impl) },
    configurable: true,
  });
}

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  _resetMicReadiness();
});

afterEach(() => {
  _resetMicReadiness();
});

// ---------------------------------------------------------------------------
// getStream / isStreamAlive / getMicReadinessState (initial state)
// ---------------------------------------------------------------------------

describe("initial state", () => {
  it("getStream returns null initially", () => {
    expect(getStream()).toBeNull();
  });

  it("isStreamAlive returns false initially (no stream)", () => {
    expect(isStreamAlive()).toBe(false);
  });

  it("getMicReadinessState is 'idle' initially", () => {
    expect(getMicReadinessState()).toBe("idle");
  });
});

// ---------------------------------------------------------------------------
// acquireStream — success paths
// ---------------------------------------------------------------------------

describe("acquireStream — success", () => {
  it("acquires a new stream when none exists", async () => {
    const stream = makeFakeStream();
    stubGetUserMedia(() => Promise.resolve(stream));
    const result = await acquireStream();
    expect(result).toBe(stream);
    expect(getMicReadinessState()).toBe("warm");
    expect(getStream()).toBe(stream);
  });

  it("transitions through 'acquiring' to 'warm'", async () => {
    let stateWhileAcquiring: string | undefined;
    stubGetUserMedia(() => {
      stateWhileAcquiring = getMicReadinessState();
      return Promise.resolve(makeFakeStream());
    });
    await acquireStream();
    expect(stateWhileAcquiring).toBe("acquiring");
    expect(getMicReadinessState()).toBe("warm");
  });

  it("reuses existing live stream without re-calling getUserMedia", async () => {
    const stream = makeFakeStream();
    const getUserMedia = vi.fn().mockResolvedValue(stream);
    Object.defineProperty(navigator, "mediaDevices", {
      value: { getUserMedia },
      configurable: true,
    });
    await acquireStream();
    await acquireStream(); // second call — should reuse
    expect(getUserMedia).toHaveBeenCalledTimes(1);
  });

  it("attaches 'ended' listeners to each track", async () => {
    const track = makeFakeTrack();
    const stream = makeFakeStream([track]);
    stubGetUserMedia(() => Promise.resolve(stream));
    await acquireStream();
    expect(track.addEventListener).toHaveBeenCalledWith("ended", expect.any(Function), { once: true });
  });
});

// ---------------------------------------------------------------------------
// acquireStream — failure path
// ---------------------------------------------------------------------------

describe("acquireStream — failure", () => {
  it("throws when getUserMedia is denied", async () => {
    stubGetUserMedia(() => Promise.reject(new Error("Permission denied")));
    await expect(acquireStream()).rejects.toThrow("Microphone access denied");
  });

  it("sets state to 'released' on failure", async () => {
    stubGetUserMedia(() => Promise.reject(new Error("denied")));
    try {
      await acquireStream();
    } catch {
      // expected
    }
    expect(getMicReadinessState()).toBe("released");
  });
});

// ---------------------------------------------------------------------------
// Track ended handler
// ---------------------------------------------------------------------------

describe("track ended handler", () => {
  it("sets state to 'released' and clears stream when track ends", async () => {
    const track = makeFakeTrack();
    const stream = makeFakeStream([track]);
    stubGetUserMedia(() => Promise.resolve(stream));
    await acquireStream();

    expect(getMicReadinessState()).toBe("warm");

    // Silence the expected console.warn
    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    track._emit("ended");
    warnSpy.mockRestore();

    expect(getMicReadinessState()).toBe("released");
    expect(getStream()).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// releaseStream
// ---------------------------------------------------------------------------

describe("releaseStream", () => {
  it("stops all tracks and nulls the stream", async () => {
    const track = makeFakeTrack();
    const stream = makeFakeStream([track]);
    stubGetUserMedia(() => Promise.resolve(stream));
    await acquireStream();

    releaseStream();

    expect(track.stop).toHaveBeenCalledOnce();
    expect(getStream()).toBeNull();
    expect(getMicReadinessState()).toBe("released");
  });

  it("is safe to call when no stream is held", () => {
    expect(() => releaseStream()).not.toThrow();
    expect(getMicReadinessState()).toBe("released");
  });
});

// ---------------------------------------------------------------------------
// isStreamAlive
// ---------------------------------------------------------------------------

describe("isStreamAlive", () => {
  it("returns false when no stream exists", () => {
    expect(isStreamAlive()).toBe(false);
  });

  it("returns true when all tracks are live", async () => {
    const stream = makeFakeStream([makeFakeTrack("live"), makeFakeTrack("live")]);
    stubGetUserMedia(() => Promise.resolve(stream));
    await acquireStream();
    expect(isStreamAlive()).toBe(true);
  });

  it("returns false when any track is ended", async () => {
    const stream = makeFakeStream([makeFakeTrack("live"), makeFakeTrack("ended")]);
    stubGetUserMedia(() => Promise.resolve(stream));
    await acquireStream();
    expect(isStreamAlive()).toBe(false);
  });

  it("returns false when all tracks are ended", async () => {
    const stream = makeFakeStream([makeFakeTrack("ended")]);
    stubGetUserMedia(() => Promise.resolve(stream));
    await acquireStream();
    expect(isStreamAlive()).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// installVisibilityHandler
// ---------------------------------------------------------------------------

describe("installVisibilityHandler", () => {
  // Track cleanup functions so we always remove handlers between tests
  let cleanupFns: Array<() => void> = [];

  function installHandler(opts: Parameters<typeof installVisibilityHandler>[0]) {
    const cleanup = installVisibilityHandler(opts);
    cleanupFns.push(cleanup);
    return cleanup;
  }

  beforeEach(() => {
    cleanupFns = [];
  });

  afterEach(() => {
    // Remove ALL installed handlers so they don't bleed into subsequent tests
    for (const fn of cleanupFns) fn();
    cleanupFns = [];
    // Reset jsdom's visibilityState property
    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      configurable: true,
    });
  });

  function simulateVisibility(state: "hidden" | "visible") {
    Object.defineProperty(document, "visibilityState", {
      value: state,
      configurable: true,
    });
    document.dispatchEvent(new Event("visibilitychange"));
  }

  it("returns a cleanup function that removes the listener", () => {
    const cleanup = installHandler({
      isRecordingActive: () => false,
      isLowLatencyEnabled: () => false,
    });
    expect(cleanup).toBeTypeOf("function");
  });

  it("releases stream when tab is hidden and not recording", async () => {
    const track = makeFakeTrack();
    const stream = makeFakeStream([track]);
    stubGetUserMedia(() => Promise.resolve(stream));
    await acquireStream();

    installHandler({
      isRecordingActive: () => false,
      isLowLatencyEnabled: () => false,
    });

    simulateVisibility("hidden");

    expect(track.stop).toHaveBeenCalledOnce();
    expect(getMicReadinessState()).toBe("released");
  });

  it("keeps stream when tab is hidden during active recording", async () => {
    const track = makeFakeTrack();
    const stream = makeFakeStream([track]);
    stubGetUserMedia(() => Promise.resolve(stream));
    await acquireStream();

    installHandler({
      isRecordingActive: () => true,
      isLowLatencyEnabled: () => false,
    });

    simulateVisibility("hidden");

    // Stream should NOT be released
    expect(track.stop).not.toHaveBeenCalled();
    expect(getMicReadinessState()).toBe("warm");
  });

  it("does not release stream when hidden and no stream exists", () => {
    // No stream acquired — should not throw
    installHandler({
      isRecordingActive: () => false,
      isLowLatencyEnabled: () => false,
    });
    expect(() => simulateVisibility("hidden")).not.toThrow();
  });

  it("re-acquires stream when visible and low-latency enabled and not recording", async () => {
    const stream = makeFakeStream();
    const getUserMedia = vi.fn().mockResolvedValue(stream);
    Object.defineProperty(navigator, "mediaDevices", {
      value: { getUserMedia },
      configurable: true,
    });

    installHandler({
      isRecordingActive: () => false,
      isLowLatencyEnabled: () => true,
    });

    simulateVisibility("visible");

    // Wait for the promise to resolve
    await vi.waitFor(() => expect(getUserMedia).toHaveBeenCalledTimes(1));
  });

  it("does not re-acquire when visible but low-latency disabled", async () => {
    const getUserMedia = vi.fn();
    Object.defineProperty(navigator, "mediaDevices", {
      value: { getUserMedia },
      configurable: true,
    });

    installHandler({
      isRecordingActive: () => false,
      isLowLatencyEnabled: () => false,
    });

    simulateVisibility("visible");
    await Promise.resolve();
    expect(getUserMedia).not.toHaveBeenCalled();
  });

  it("does not re-acquire when visible but recording is active", async () => {
    const getUserMedia = vi.fn();
    Object.defineProperty(navigator, "mediaDevices", {
      value: { getUserMedia },
      configurable: true,
    });

    installHandler({
      isRecordingActive: () => true,
      isLowLatencyEnabled: () => true,
    });

    simulateVisibility("visible");
    await Promise.resolve();
    expect(getUserMedia).not.toHaveBeenCalled();
  });

  it("logs a warning when re-acquire fails on visible", async () => {
    const getUserMedia = vi.fn().mockRejectedValue(new Error("denied"));
    Object.defineProperty(navigator, "mediaDevices", {
      value: { getUserMedia },
      configurable: true,
    });

    installHandler({
      isRecordingActive: () => false,
      isLowLatencyEnabled: () => true,
    });

    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    simulateVisibility("visible");
    await vi.waitFor(() => expect(getUserMedia).toHaveBeenCalled());
    await new Promise((r) => setTimeout(r, 0));
    expect(warnSpy).toHaveBeenCalled();
    warnSpy.mockRestore();
  });

  it("cleanup removes the visibility listener", async () => {
    const track = makeFakeTrack();
    const stream = makeFakeStream([track]);
    stubGetUserMedia(() => Promise.resolve(stream));
    await acquireStream();

    const cleanup = installVisibilityHandler({
      isRecordingActive: () => false,
      isLowLatencyEnabled: () => false,
    });
    cleanup(); // remove listener manually (NOT via cleanupFns)

    simulateVisibility("hidden");
    // Stream should NOT be released (listener was removed before dispatch)
    expect(track.stop).not.toHaveBeenCalled();
  });
});

// ---------------------------------------------------------------------------
// _resetMicReadiness
// ---------------------------------------------------------------------------

describe("_resetMicReadiness", () => {
  it("stops tracks and resets state to idle", async () => {
    const track = makeFakeTrack();
    const stream = makeFakeStream([track]);
    stubGetUserMedia(() => Promise.resolve(stream));
    await acquireStream();

    _resetMicReadiness();

    expect(track.stop).toHaveBeenCalledOnce();
    expect(getStream()).toBeNull();
    expect(getMicReadinessState()).toBe("idle");
  });

  it("is safe to call when no stream is held", () => {
    expect(() => _resetMicReadiness()).not.toThrow();
    expect(getMicReadinessState()).toBe("idle");
  });
});
