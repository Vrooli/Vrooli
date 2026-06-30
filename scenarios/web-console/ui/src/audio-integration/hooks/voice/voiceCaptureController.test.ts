import { afterEach, describe, expect, it, vi } from "vitest";
import { VoiceCaptureController, type ProviderRef } from "./voiceCaptureController";
import {
  _resetMicOwnershipForTesting,
  getActiveMicLeases,
  registerMicStream,
} from "./micOwnership";
import type { LastTurnAudio, TranscriptionProvider } from "./types";

/** Minimal TranscriptionProvider that records dispose/stop calls. */
function fakeProvider(): TranscriptionProvider & { disposeCount: number; stopCount: number } {
  return {
    disposeCount: 0,
    stopCount: 0,
    start() {},
    stop() { this.stopCount++; },
    dispose() { this.disposeCount++; },
    getStream(): MediaStream | null { return null; },
    getLastTurnAudio(): LastTurnAudio | null { return null; },
    disposeLastTurn() {},
    dropTail() {},
    onResult: null,
    onError: null,
    onPartial: null,
  };
}

function fakeTrack() {
  const t = {
    readyState: "live" as "live" | "ended",
    muted: false,
    kind: "audio",
    stop: vi.fn(() => { t.readyState = "ended"; }),
    addEventListener() {},
    removeEventListener() {},
  };
  return t;
}

function fakeStream() {
  const tracks = [fakeTrack()];
  return { tracks, stream: { getTracks: () => tracks } as unknown as MediaStream };
}

describe("VoiceCaptureController", () => {
  afterEach(() => {
    _resetMicOwnershipForTesting();
    vi.restoreAllMocks();
  });

  it("replace() disposes the previous provider BEFORE installing the next (atomic)", () => {
    const ref: ProviderRef = { current: null };
    const controller = new VoiceCaptureController(ref);
    const a = fakeProvider();
    const b = fakeProvider();
    ref.current = a;

    controller.replace(b, "provider-error");

    expect(a.disposeCount).toBe(1);
    expect(ref.current).toBe(b);
    // Idempotent: replacing with the already-current provider is a no-op.
    controller.replace(b, "provider-error");
    expect(b.disposeCount).toBe(0);
    expect(ref.current).toBe(b);
  });

  it("shutdown() disposes the provider exactly once and runs teardown, idempotently", () => {
    const ref: ProviderRef = { current: null };
    const teardown = vi.fn();
    const controller = new VoiceCaptureController(ref, { onCaptureTeardown: teardown });
    const a = fakeProvider();
    ref.current = a;

    controller.shutdown("unmount");
    controller.shutdown("unmount"); // second call must be safe

    expect(a.disposeCount).toBe(1);
    expect(ref.current).toBeNull();
    // Teardown runs each call (it is itself idempotent) but the provider is
    // disposed only once.
    expect(teardown).toHaveBeenCalled();
  });

  it("generation token invalidates after cancelStarts (late-resolve detection)", () => {
    const controller = new VoiceCaptureController({ current: null });
    const token = controller.beginStart();
    expect(controller.isCurrentStart(token)).toBe(true);

    controller.cancelStarts(); // e.g. tab hidden during preparing
    expect(controller.isCurrentStart(token)).toBe(false);

    const next = controller.beginStart();
    expect(controller.isCurrentStart(next)).toBe(true);
    expect(controller.isCurrentStart(token)).toBe(false);
  });

  it("recoverStaleLeases releases an orphaned recording lease, disposes the provider, and logs", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const ref: ProviderRef = { current: null };
    const controller = new VoiceCaptureController(ref);
    const provider = fakeProvider();
    ref.current = provider;

    // Simulate the bug: a live "voice-stream" lease while the UI is idle.
    const { tracks } = fakeStream();
    registerMicStream("voice-stream", { getTracks: () => tracks } as unknown as MediaStream);
    expect(getActiveMicLeases()).toHaveLength(1);

    const released = controller.recoverStaleLeases({
      voiceState: "idle",
      lowLatencyVoice: false,
      passiveListenerActive: false,
    });

    expect(released).toHaveLength(1);
    expect(released[0]?.owner).toBe("voice-stream");
    expect(tracks[0]?.stop).toHaveBeenCalledTimes(1);
    expect(getActiveMicLeases()).toHaveLength(0);
    // The dangling provider is also disposed (it escaped normal cleanup).
    expect(provider.disposeCount).toBe(1);
    expect(ref.current).toBeNull();
    expect(warn).toHaveBeenCalled();
  });

  it("recoverStaleLeases is a no-op when the live lease is legitimately expected", () => {
    const ref: ProviderRef = { current: fakeProvider() };
    const controller = new VoiceCaptureController(ref);
    const { tracks } = fakeStream();
    // Active recording owner while the workflow IS recording — expected.
    registerMicStream("voice-stream", { getTracks: () => tracks } as unknown as MediaStream);

    const released = controller.recoverStaleLeases({
      voiceState: "recording",
      lowLatencyVoice: false,
      passiveListenerActive: false,
    });

    expect(released).toHaveLength(0);
    expect(tracks[0]?.stop).not.toHaveBeenCalled();
    expect(getActiveMicLeases()).toHaveLength(1);
  });
});
