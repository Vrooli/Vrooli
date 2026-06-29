/**
 * Unit tests for WebSpeechProvider.
 *
 * jsdom does not implement SpeechRecognition. We stub both
 * window.SpeechRecognition and window.webkitSpeechRecognition with a fake
 * class that exposes the same handler slots (onresult/onerror/onend) that
 * WebSpeechProvider wires up, and spy methods (start/stop/abort) so tests
 * can verify calls and manually fire events.
 *
 * console.info is not tracked by test-setup, so no suppression is needed.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { WebSpeechProvider } from "./WebSpeechProvider";

// ─── Fake SpeechRecognition ───────────────────────────────────────────────────

class FakeSpeechRecognition {
  static lastInstance: FakeSpeechRecognition | null = null;

  continuous = false;
  interimResults = false;
  lang = "";
  onresult: ((event: FakeSpeechRecognitionEvent) => void) | null = null;
  onerror: ((event: { error: string; message: string }) => void) | null = null;
  onend: (() => void) | null = null;

  start = vi.fn();
  stop = vi.fn();
  abort = vi.fn();

  constructor() {
    FakeSpeechRecognition.lastInstance = this;
  }

  /** Build and dispatch a recognition result event. */
  fireResult(items: { transcript: string; isFinal: boolean }[]): void {
    // Build a cumulative results list matching the SpeechRecognitionResultList shape.
    const resultList = { length: items.length } as Record<string, unknown>;
    for (let i = 0; i < items.length; i++) {
      resultList[i] = {
        isFinal: items[i]!.isFinal,
        length: 1,
        0: { transcript: items[i]!.transcript, confidence: 1 },
      };
    }
    // The source reads event.results.length, so wrap the list in { results: ... }.
    this.onresult?.({ results: resultList } as unknown as FakeSpeechRecognitionEvent);
  }
}

interface FakeSpeechRecognitionEvent {
  results: Record<string, unknown> & { length: number };
}

// ─── Fake MediaStream / Track ─────────────────────────────────────────────────

function makeStream(readyState: "live" | "ended" = "live"): MediaStream {
  return {
    getTracks: () => [{ readyState, stop: vi.fn() }],
  } as unknown as MediaStream;
}

// ─── suite ────────────────────────────────────────────────────────────────────

describe("WebSpeechProvider", () => {
  let getUserMediaMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    FakeSpeechRecognition.lastInstance = null;
    vi.clearAllMocks();
    getUserMediaMock = vi.fn().mockResolvedValue(makeStream());
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    vi.stubGlobal("webkitSpeechRecognition", undefined);
    vi.stubGlobal("navigator", {
      mediaDevices: { getUserMedia: getUserMediaMock },
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  // ── static interface ─────────────────────────────────────────────────────────

  it("getLastTurnAudio() always returns null (no audio bytes from Web Speech)", async () => {
    const p = new WebSpeechProvider();
    await p.start(makeStream());
    expect(p.getLastTurnAudio()).toBeNull();
    p.stop();
  });

  it("disposeLastTurn() is a no-op", () => {
    const p = new WebSpeechProvider();
    expect(() => p.disposeLastTurn()).not.toThrow();
  });

  it("dropTail() is a no-op", () => {
    const p = new WebSpeechProvider();
    expect(() => p.dropTail()).not.toThrow();
  });

  it("getStream() returns null before start", () => {
    const p = new WebSpeechProvider();
    expect(p.getStream()).toBeNull();
  });

  // ── start — API unavailable ──────────────────────────────────────────────────

  it("start() fires onError when no Speech Recognition API is available", async () => {
    vi.stubGlobal("SpeechRecognition", undefined);
    vi.stubGlobal("webkitSpeechRecognition", undefined);

    const p = new WebSpeechProvider();
    const onError = vi.fn();
    p.onError = onError;
    await p.start();

    expect(onError).toHaveBeenCalledWith("Web Speech API not available");
    expect(FakeSpeechRecognition.lastInstance).toBeNull();
  });

  it("start() uses webkitSpeechRecognition as fallback", async () => {
    vi.stubGlobal("SpeechRecognition", undefined);
    vi.stubGlobal("webkitSpeechRecognition", FakeSpeechRecognition);

    const p = new WebSpeechProvider();
    await p.start(makeStream());
    expect(FakeSpeechRecognition.lastInstance).not.toBeNull();
    p.stop();
  });

  // ── start — pre-warmed stream ─────────────────────────────────────────────────

  it("start() uses pre-warmed stream when all tracks are live", async () => {
    const preWarmed = makeStream("live");
    const p = new WebSpeechProvider();
    await p.start(preWarmed);
    expect(getUserMediaMock).not.toHaveBeenCalled();
    expect(p.getStream()).toBe(preWarmed);
    p.stop();
  });

  it("start() falls back to getUserMedia when pre-warmed stream has ended tracks", async () => {
    const endedStream = makeStream("ended");
    const p = new WebSpeechProvider();
    await p.start(endedStream);
    expect(getUserMediaMock).toHaveBeenCalled();
    p.stop();
  });

  // ── start — getUserMedia ──────────────────────────────────────────────────────

  it("start() acquires fresh stream when no pre-warmed stream", async () => {
    const p = new WebSpeechProvider();
    await p.start();
    expect(getUserMediaMock).toHaveBeenCalledWith({ audio: true });
    expect(FakeSpeechRecognition.lastInstance!.start).toHaveBeenCalled();
    p.stop();
  });

  it("start() fires onError when getUserMedia fails", async () => {
    getUserMediaMock.mockRejectedValueOnce(new Error("denied"));
    const p = new WebSpeechProvider();
    const onError = vi.fn();
    p.onError = onError;
    await p.start();
    expect(onError).toHaveBeenCalledWith("Microphone access denied");
    expect(FakeSpeechRecognition.lastInstance).toBeNull();
  });

  // ── start — recognition configuration ────────────────────────────────────────

  it("start() configures recognition with correct settings", async () => {
    const p = new WebSpeechProvider();
    p.lang = "fr-FR";
    await p.start(makeStream());
    const r = FakeSpeechRecognition.lastInstance!;
    expect(r.continuous).toBe(true);
    expect(r.interimResults).toBe(true);
    expect(r.lang).toBe("fr-FR");
    expect(r.start).toHaveBeenCalled();
    p.stop();
  });

  // ── onresult — final results ──────────────────────────────────────────────────

  it("onresult fires onResult with final transcript text", async () => {
    const p = new WebSpeechProvider();
    const onResult = vi.fn();
    p.onResult = onResult;
    await p.start(makeStream());
    const r = FakeSpeechRecognition.lastInstance!;
    r.fireResult([{ transcript: "  hello world  ", isFinal: true }]);
    expect(onResult).toHaveBeenCalledWith("hello world");
    p.stop();
  });

  it("onresult does NOT call onResult when final text is whitespace", async () => {
    const p = new WebSpeechProvider();
    const onResult = vi.fn();
    p.onResult = onResult;
    await p.start(makeStream());
    FakeSpeechRecognition.lastInstance!.fireResult([{ transcript: "   ", isFinal: true }]);
    expect(onResult).not.toHaveBeenCalled();
    p.stop();
  });

  // ── onresult — interim results ────────────────────────────────────────────────

  it("onresult fires onPartial for non-final results", async () => {
    const p = new WebSpeechProvider();
    const onPartial = vi.fn();
    p.onPartial = onPartial;
    await p.start(makeStream());
    FakeSpeechRecognition.lastInstance!.fireResult([{ transcript: "in progress", isFinal: false }]);
    expect(onPartial).toHaveBeenCalledWith("in progress");
    p.stop();
  });

  it("onresult handles mixed final and interim results", async () => {
    const p = new WebSpeechProvider();
    const onResult = vi.fn();
    const onPartial = vi.fn();
    p.onResult = onResult;
    p.onPartial = onPartial;
    await p.start(makeStream());
    FakeSpeechRecognition.lastInstance!.fireResult([
      { transcript: "hello", isFinal: true },
      { transcript: "world", isFinal: false },
    ]);
    expect(onResult).toHaveBeenCalledWith("hello");
    expect(onPartial).toHaveBeenCalledWith("world");
    p.stop();
  });

  // ── onresult — cumulative processing ─────────────────────────────────────────

  it("onresult skips previously finalized results on subsequent events", async () => {
    const p = new WebSpeechProvider();
    const onResult = vi.fn();
    p.onResult = onResult;
    await p.start(makeStream());
    const r = FakeSpeechRecognition.lastInstance!;

    // First event: one final result at index 0
    r.fireResult([{ transcript: "first", isFinal: true }]);
    expect(onResult).toHaveBeenCalledTimes(1);
    expect(onResult).toHaveBeenLastCalledWith("first");

    // Second event: cumulative — index 0 is already processed, index 1 is new
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion
    r.onresult?.({
      results: {
        length: 2,
        0: { isFinal: true, length: 1, 0: { transcript: "first", confidence: 1 } },
        1: { isFinal: true, length: 1, 0: { transcript: "second", confidence: 1 } },
      },
    } as unknown as FakeSpeechRecognitionEvent);

    // Only "second" should be dispatched on second call
    expect(onResult).toHaveBeenCalledTimes(2);
    expect(onResult).toHaveBeenLastCalledWith("second");
    p.stop();
  });

  // ── onerror ───────────────────────────────────────────────────────────────────

  it("onerror fires onError for non-aborted errors", async () => {
    const p = new WebSpeechProvider();
    const onError = vi.fn();
    p.onError = onError;
    await p.start(makeStream());
    FakeSpeechRecognition.lastInstance!.onerror?.({ error: "network", message: "" });
    expect(onError).toHaveBeenCalledWith("Speech recognition error: network");
    p.stop();
  });

  it("onerror ignores 'aborted' error", async () => {
    const p = new WebSpeechProvider();
    const onError = vi.fn();
    p.onError = onError;
    await p.start(makeStream());
    FakeSpeechRecognition.lastInstance!.onerror?.({ error: "aborted", message: "" });
    expect(onError).not.toHaveBeenCalled();
    p.stop();
  });

  // ── onend — auto-restart ──────────────────────────────────────────────────────

  it("onend auto-restarts recognition when not stopped", async () => {
    const p = new WebSpeechProvider();
    await p.start(makeStream());
    const r = FakeSpeechRecognition.lastInstance!;
    const startCallsBefore = r.start.mock.calls.length;
    r.onend?.();
    expect(r.start.mock.calls.length).toBeGreaterThan(startCallsBefore);
    p.stop();
  });

  it("onend does NOT restart when provider is stopped", async () => {
    const p = new WebSpeechProvider();
    await p.start(makeStream());
    const r = FakeSpeechRecognition.lastInstance!;
    p.stop();
    const startCallsAfterStop = r.start.mock.calls.length;
    r.onend?.(); // fire onend after stop() has set stopped=true
    expect(r.start.mock.calls.length).toBe(startCallsAfterStop);
  });

  it("onend does NOT restart when recognition is null (after dispose)", async () => {
    const p = new WebSpeechProvider();
    await p.start(makeStream());
    const r = FakeSpeechRecognition.lastInstance!;
    p.dispose();
    const startCalls = r.start.mock.calls.length;
    // manually invoke onend handler — recognition is null after stop/dispose
    // should not throw
    r.onend?.();
    expect(r.start.mock.calls.length).toBe(startCalls);
  });

  // ── stop ─────────────────────────────────────────────────────────────────────

  it("stop() calls recognition.stop() and releases stream", async () => {
    const fakeTrack = { readyState: "live", stop: vi.fn() };
    const stream = { getTracks: () => [fakeTrack] } as unknown as MediaStream;
    const p = new WebSpeechProvider();
    await p.start(stream);
    p.stop();
    expect(FakeSpeechRecognition.lastInstance!.stop).toHaveBeenCalled();
    expect(fakeTrack.stop).toHaveBeenCalled();
    expect(p.getStream()).toBeNull();
  });

  it("stop() is safe to call when not started", () => {
    const p = new WebSpeechProvider();
    expect(() => p.stop()).not.toThrow();
  });

  // ── dispose ──────────────────────────────────────────────────────────────────

  it("dispose() delegates to stop()", async () => {
    const p = new WebSpeechProvider();
    await p.start(makeStream());
    p.dispose();
    expect(FakeSpeechRecognition.lastInstance!.stop).toHaveBeenCalled();
    expect(p.getStream()).toBeNull();
  });

  // ── branch coverage: nullish fallbacks and start() throw in onend ─────────────

  it("onresult ?? fallback: final result with undefined transcript emits empty string (not dispatched)", async () => {
    // Hits the `?? ""` branch on result[0]?.transcript when transcript is absent.
    const p = new WebSpeechProvider();
    const onResult = vi.fn();
    p.onResult = onResult;
    await p.start(makeStream());
    const r = FakeSpeechRecognition.lastInstance!;
    // isFinal=true but no transcript property → `?? ""` → trim → empty → not dispatched
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion
    r.onresult?.({
      results: {
        length: 1,
        0: { isFinal: true, length: 0 },  // no [0] item → undefined
      },
    } as unknown as FakeSpeechRecognitionEvent);
    expect(onResult).not.toHaveBeenCalled();
    p.stop();
  });

  it("onresult ?? fallback: interim result with undefined transcript fires onPartial with empty string not dispatched", async () => {
    // Hits the `?? ""` branch on result?.[0]?.transcript for non-final results.
    const p = new WebSpeechProvider();
    const onPartial = vi.fn();
    p.onPartial = onPartial;
    await p.start(makeStream());
    const r = FakeSpeechRecognition.lastInstance!;
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion
    r.onresult?.({
      results: {
        length: 1,
        0: { isFinal: false, length: 0 },  // no [0] item
      },
    } as unknown as FakeSpeechRecognitionEvent);
    // interimText is "" so onPartial should NOT be called (falsy guard)
    expect(onPartial).not.toHaveBeenCalled();
    p.stop();
  });

  it("onend catches and silently ignores start() throws during auto-restart", async () => {
    const p = new WebSpeechProvider();
    await p.start(makeStream());
    const r = FakeSpeechRecognition.lastInstance!;
    // Make next start() throw to exercise the catch block in onend
    r.start.mockImplementationOnce(() => {
      throw new Error("already started");
    });
    expect(() => r.onend?.()).not.toThrow();
    p.stop();
  });
});
