import { afterEach, describe, expect, it, vi } from "vitest";
import {
  _resetMicReadiness,
  acquireStream,
  getMicReadinessState,
  getStream,
  isStreamUsable,
  isTrackUsable,
  releaseStream,
} from "./micReadiness";

function track(readyState: "live" | "ended", muted: boolean): MediaStreamTrack {
  return { readyState, muted, kind: "audio" } as unknown as MediaStreamTrack;
}

function streamOf(...tracks: MediaStreamTrack[]): MediaStream {
  return { getTracks: () => tracks } as unknown as MediaStream;
}

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

  // Regression: the "mic stuck until page reload" bug. A live-but-muted track
  // flows no audio (sleep/wake, device change, another app holding the mic) yet
  // the "ended" event never fires, so a retained stream sat muted forever and
  // recorded silence. Muted tracks must be treated as NOT usable so the next
  // session re-acquires a fresh stream.
  describe("isStreamUsable / muted-track rejection", () => {
    it("accepts a live, unmuted track", () => {
      expect(isTrackUsable(track("live", false))).toBe(true);
      expect(isStreamUsable(streamOf(track("live", false)))).toBe(true);
    });

    it("rejects a live but MUTED track (the stuck-mic case)", () => {
      expect(isTrackUsable(track("live", true))).toBe(false);
      expect(isStreamUsable(streamOf(track("live", true)))).toBe(false);
    });

    it("rejects an ended track", () => {
      expect(isTrackUsable(track("ended", false))).toBe(false);
      expect(isStreamUsable(streamOf(track("ended", false)))).toBe(false);
    });

    it("rejects when ANY track is muted", () => {
      expect(isStreamUsable(streamOf(track("live", false), track("live", true)))).toBe(false);
    });

    it("rejects null/empty streams", () => {
      expect(isStreamUsable(null)).toBe(false);
      expect(isStreamUsable(streamOf())).toBe(false);
    });
  });
});
