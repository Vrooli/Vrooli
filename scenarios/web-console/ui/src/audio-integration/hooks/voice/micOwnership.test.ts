import { afterEach, describe, expect, it, vi } from "vitest";
import {
  _resetMicOwnershipForTesting,
  acquireMicStream,
  getActiveMicLeases,
  installMicLifecycleCleanup,
  registerMicStream,
  releaseAllMicLeases,
  releaseMicLease,
} from "./micOwnership";

interface FakeTrack {
  readyState: "live" | "ended";
  muted: boolean;
  kind: string;
  stop: ReturnType<typeof vi.fn>;
  listeners: Record<string, Array<(ev?: unknown) => void>>;
  addEventListener: (type: string, cb: (ev?: unknown) => void) => void;
  removeEventListener: (type: string, cb: (ev?: unknown) => void) => void;
  fire: (type: string) => void;
}

function fakeTrack(): FakeTrack {
  const listeners: FakeTrack["listeners"] = {};
  const t: FakeTrack = {
    readyState: "live",
    muted: false,
    kind: "audio",
    stop: vi.fn(() => { t.readyState = "ended"; }),
    listeners,
    addEventListener(type, cb) {
      (listeners[type] ??= []).push(cb);
    },
    removeEventListener(type, cb) {
      listeners[type] = (listeners[type] ?? []).filter((c) => c !== cb);
    },
    fire(type) {
      for (const cb of listeners[type] ?? []) cb();
    },
  };
  return t;
}

function fakeStream(tracks: FakeTrack[]): MediaStream {
  return { getTracks: () => tracks as unknown as MediaStreamTrack[] } as unknown as MediaStream;
}

describe("micOwnership", () => {
  afterEach(() => {
    _resetMicOwnershipForTesting();
    vi.restoreAllMocks();
  });

  it("registers a stream and exposes a metadata-only snapshot", () => {
    const track = fakeTrack();
    const lease = registerMicStream("passive-wake-word", fakeStream([track]), { metadata: { mode: "test" } });

    const snaps = getActiveMicLeases();
    expect(snaps).toHaveLength(1);
    expect(snaps[0]).toMatchObject({ owner: "passive-wake-word", trackCount: 1, liveTrackCount: 1, metadata: { mode: "test" } });
    // The snapshot must not leak the raw stream.
    expect(snaps[0]).not.toHaveProperty("stream");
    expect(lease.released).toBe(false);
  });

  it("acquireMicStream calls getUserMedia and registers the result", async () => {
    const track = fakeTrack();
    const stream = fakeStream([track]);
    const getUserMedia = vi.fn().mockResolvedValue(stream);
    Object.defineProperty(navigator, "mediaDevices", { configurable: true, value: { getUserMedia } });

    const lease = await acquireMicStream("voice-stream", { audio: true });
    expect(getUserMedia).toHaveBeenCalledWith({ audio: true });
    expect(lease.stream).toBe(stream);
    expect(getActiveMicLeases()).toHaveLength(1);
  });

  it("release stops all tracks exactly once, is idempotent, and fires onRelease once", () => {
    const a = fakeTrack();
    const b = fakeTrack();
    const onRelease = vi.fn();
    const lease = registerMicStream("speaker-enrollment", fakeStream([a, b]), { onRelease });

    releaseMicLease(lease, "manual-stop");
    releaseMicLease(lease, "unmount"); // second call is a no-op

    expect(a.stop).toHaveBeenCalledTimes(1);
    expect(b.stop).toHaveBeenCalledTimes(1);
    expect(onRelease).toHaveBeenCalledTimes(1);
    expect(onRelease).toHaveBeenCalledWith("manual-stop");
    expect(lease.released).toBe(true);
    expect(getActiveMicLeases()).toHaveLength(0);
  });

  it("an OS track 'ended' event releases the lease and fires onRelease", () => {
    const track = fakeTrack();
    const onRelease = vi.fn();
    registerMicStream("low-latency-prewarm", fakeStream([track]), { onRelease });

    track.fire("ended");

    expect(onRelease).toHaveBeenCalledTimes(1);
    expect(onRelease).toHaveBeenCalledWith("ended");
    expect(getActiveMicLeases()).toHaveLength(0);
  });

  it("releaseAllMicLeases honors a predicate", () => {
    const passive = registerMicStream("passive-wake-word", fakeStream([fakeTrack()]));
    const active = registerMicStream("voice-stream", fakeStream([fakeTrack()]));

    releaseAllMicLeases("hidden", (l) => l.owner === "passive-wake-word");

    expect(passive.released).toBe(true);
    expect(active.released).toBe(false);
    expect(getActiveMicLeases().map((l) => l.owner)).toEqual(["voice-stream"]);
  });

  describe("installMicLifecycleCleanup", () => {
    it("releases non-active leases on visibility hidden and all leases on pagehide", () => {
      const uninstall = installMicLifecycleCleanup();
      const passive = registerMicStream("passive-wake-word", fakeStream([fakeTrack()]));
      const prewarm = registerMicStream("low-latency-prewarm", fakeStream([fakeTrack()]));
      const recording = registerMicStream("voice-stream", fakeStream([fakeTrack()]));

      Object.defineProperty(document, "visibilityState", { configurable: true, value: "hidden" });
      document.dispatchEvent(new Event("visibilitychange"));

      expect(passive.released).toBe(true);
      expect(prewarm.released).toBe(true);
      // Active recording is owned by its provider, not the backstop, on hidden.
      expect(recording.released).toBe(false);

      window.dispatchEvent(new Event("pagehide"));
      expect(recording.released).toBe(true);

      uninstall();
    });

    it("ref-counts so a single uninstall does not detach a still-needed handler", () => {
      const uninstall1 = installMicLifecycleCleanup();
      const uninstall2 = installMicLifecycleCleanup();
      uninstall1();

      const passive = registerMicStream("passive-wake-word", fakeStream([fakeTrack()]));
      Object.defineProperty(document, "visibilityState", { configurable: true, value: "hidden" });
      document.dispatchEvent(new Event("visibilitychange"));
      expect(passive.released).toBe(true);

      uninstall2();
    });
  });
});
