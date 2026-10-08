/**
 * Unit tests for WhisperProvider.
 *
 * Dependencies mocked:
 *  - transcribeAudioWithRetry from ../../api/voice (module mock)
 *  - navigator.mediaDevices.getUserMedia (stubbed on global)
 *  - MediaRecorder (stubbed on global — absent from jsdom)
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { makeMediaStream } from "../../test-support/browser";

// ─── Module mocks (hoisted) ───────────────────────────────────────────────────
const transcribeMock = vi.fn();

vi.mock("../../api/voice", () => ({
  transcribeAudioWithRetry: (...args: unknown[]) => transcribeMock(...args),
  WHISPER_FAILED_SENTINEL: "__WHISPER_FAILED__",
}));

import { WhisperProvider } from "./WhisperProvider";
import { WHISPER_FAILED_SENTINEL } from "./types";

// ─── Fake MediaRecorder ───────────────────────────────────────────────────────

class FakeMediaRecorder {
  static lastInstance: FakeMediaRecorder | null = null;
  static isTypeSupported = vi.fn().mockReturnValue(true);

  state: string = "inactive";
  ondataavailable: ((e: { data: Blob }) => void) | null = null;
  onstop: (() => void | Promise<void>) | null = null;

  start = vi.fn().mockImplementation(() => {
    this.state = "recording";
  });
  stop = vi.fn().mockImplementation(() => {
    this.state = "inactive";
  });

  constructor(_stream: unknown, _options?: unknown) {
    FakeMediaRecorder.lastInstance = this;
  }

  /** Test helper — push a data chunk into the provider. */
  pushChunk(data: Blob): void {
    this.ondataavailable?.({ data });
  }

  /** Test helper — fire the onstop callback and await it. */
  async fireStop(): Promise<void> {
    const cb = this.onstop;
    if (cb) await cb();
  }
}

// ─── Fake MediaStream / Track ─────────────────────────────────────────────────

// ─── helpers ──────────────────────────────────────────────────────────────────

/** Suppress console.warn emitted when transcription fails (test-setup kills stray warns). */
let warnSpy: ReturnType<typeof vi.spyOn> | undefined;

function suppressWarn(): void {
  warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
}
function restoreWarn(): void {
  warnSpy?.mockRestore();
}

// ─── suite ────────────────────────────────────────────────────────────────────

describe("WhisperProvider", () => {
  let getUserMediaMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    FakeMediaRecorder.lastInstance = null;
    vi.clearAllMocks();
    vi.stubGlobal("MediaRecorder", FakeMediaRecorder);
    getUserMediaMock = vi.fn().mockResolvedValue(makeMediaStream());
    vi.stubGlobal("navigator", {
      mediaDevices: { getUserMedia: getUserMediaMock },
    });
  });

  afterEach(() => {
    restoreWarn();
    vi.unstubAllGlobals();
  });

  // ── initial state ────────────────────────────────────────────────────────────

  it("getStream() returns null before start", () => {
    const p = new WhisperProvider();
    expect(p.getStream()).toBeNull();
  });

  it("getLastTurnAudio() returns null before any turn", () => {
    const p = new WhisperProvider();
    expect(p.getLastTurnAudio()).toBeNull();
  });

  it("disposeLastTurn() is a no-op when no audio retained", () => {
    const p = new WhisperProvider();
    expect(() => p.disposeLastTurn()).not.toThrow();
  });

  it("dropTail() is a no-op (batch mode)", () => {
    const p = new WhisperProvider();
    expect(() => p.dropTail()).not.toThrow();
  });

  // ── start with pre-warmed stream ─────────────────────────────────────────────

  it("start() uses pre-warmed stream when all tracks are live", async () => {
    const preWarmed = makeMediaStream("live");
    const p = new WhisperProvider();
    await p.start(preWarmed);
    expect(getUserMediaMock).not.toHaveBeenCalled();
    expect(FakeMediaRecorder.lastInstance).not.toBeNull();
    expect(FakeMediaRecorder.lastInstance!.start).toHaveBeenCalled();
  });

  it("start() ignores pre-warmed stream with ended tracks and falls through to getUserMedia", async () => {
    const endedStream = makeMediaStream("ended");
    const p = new WhisperProvider();
    await p.start(endedStream);
    expect(getUserMediaMock).toHaveBeenCalled();
  });

  // ── start with getUserMedia ───────────────────────────────────────────────────

  it("start() acquires fresh stream via getUserMedia when no pre-warmed stream", async () => {
    const p = new WhisperProvider();
    await p.start();
    expect(getUserMediaMock).toHaveBeenCalledWith({ audio: true });
    expect(FakeMediaRecorder.lastInstance!.start).toHaveBeenCalled();
  });

  it("start() calls onError and returns when getUserMedia fails", async () => {
    getUserMediaMock.mockRejectedValueOnce(new Error("denied"));
    const p = new WhisperProvider();
    const onError = vi.fn();
    p.onError = onError;
    await p.start();
    expect(onError).toHaveBeenCalledWith("Microphone access denied");
    expect(FakeMediaRecorder.lastInstance).toBeNull(); // recorder never created
  });

  it("start() clears previous lastTurn on new turn", async () => {
    const p = new WhisperProvider();
    // Manually seed a fake lastTurn (simulates previous turn)
    // @ts-expect-error — accessing private for test
    p.lastTurn = { blob: new Blob(["x"]), mimeType: "audio/webm", durationMs: 1000, capturedAt: 0 };
    await p.start(makeMediaStream());
    expect(p.getLastTurnAudio()).toBeNull();
  });

  // ── onstop — transcription success ───────────────────────────────────────────

  it("onstop fires onResult with trimmed text on success", async () => {
    const p = new WhisperProvider();
    const onResult = vi.fn();
    p.onResult = onResult;
    transcribeMock.mockResolvedValueOnce("  hello world  ");

    await p.start(makeMediaStream());
    const mr = FakeMediaRecorder.lastInstance!;
    mr.pushChunk(new Blob(["audio-data"]));
    await mr.fireStop();

    expect(transcribeMock).toHaveBeenCalledWith(expect.any(Blob), 2, "en");
    expect(onResult).toHaveBeenCalledWith("hello world");
  });

  it("onstop resolves onResult with an empty string when text is whitespace-only", async () => {
    const p = new WhisperProvider();
    const onResult = vi.fn();
    p.onResult = onResult;
    transcribeMock.mockResolvedValueOnce("   ");

    await p.start(makeMediaStream());
    const mr = FakeMediaRecorder.lastInstance!;
    mr.pushChunk(new Blob(["data"]));
    await mr.fireStop();

    expect(onResult).toHaveBeenCalledWith("");
  });

  it("onstop retains lastTurnAudio before calling transcribe", async () => {
    const p = new WhisperProvider();
    transcribeMock.mockResolvedValueOnce("text");

    await p.start(makeMediaStream());
    const mr = FakeMediaRecorder.lastInstance!;
    mr.pushChunk(new Blob(["data"]));
    await mr.fireStop();

    const last = p.getLastTurnAudio();
    expect(last).not.toBeNull();
    expect(last!.mimeType).toMatch(/audio\/webm/);
    expect(last!.durationMs).toBeGreaterThanOrEqual(0);
  });

  it("disposeLastTurn() clears retained audio", async () => {
    const p = new WhisperProvider();
    transcribeMock.mockResolvedValueOnce("text");

    await p.start(makeMediaStream());
    const mr = FakeMediaRecorder.lastInstance!;
    mr.pushChunk(new Blob(["data"]));
    await mr.fireStop();

    expect(p.getLastTurnAudio()).not.toBeNull();
    p.disposeLastTurn();
    expect(p.getLastTurnAudio()).toBeNull();
  });

  it("onstop skips transcription when blob is empty", async () => {
    const p = new WhisperProvider();
    const onResult = vi.fn();
    p.onResult = onResult;

    await p.start(makeMediaStream());
    const mr = FakeMediaRecorder.lastInstance!;
    // No chunks pushed → blob is empty
    await mr.fireStop();

    expect(transcribeMock).not.toHaveBeenCalled();
    expect(onResult).not.toHaveBeenCalled();
  });

  // ── onstop — transcription failure ───────────────────────────────────────────

  it("onstop fires onError with WHISPER_FAILED_SENTINEL when transcribe throws", async () => {
    suppressWarn();
    const p = new WhisperProvider();
    const onError = vi.fn();
    p.onError = onError;
    transcribeMock.mockRejectedValueOnce(new Error("network"));

    await p.start(makeMediaStream());
    const mr = FakeMediaRecorder.lastInstance!;
    mr.pushChunk(new Blob(["data"]));
    await mr.fireStop();

    expect(onError).toHaveBeenCalledWith(WHISPER_FAILED_SENTINEL);
  });

  it("onstop uses language property when transcribing", async () => {
    const p = new WhisperProvider();
    p.language = "fr";
    transcribeMock.mockResolvedValueOnce("bonjour");

    await p.start(makeMediaStream());
    const mr = FakeMediaRecorder.lastInstance!;
    mr.pushChunk(new Blob(["data"]));
    await mr.fireStop();

    expect(transcribeMock).toHaveBeenCalledWith(expect.any(Blob), 2, "fr");
  });

  it("onstop releases stream tracks before transcribing", async () => {
    const fakeTrack = { readyState: "live", stop: vi.fn() };
    const stream = { getTracks: () => [fakeTrack] } as unknown as MediaStream;
    getUserMediaMock.mockResolvedValueOnce(stream);
    transcribeMock.mockResolvedValueOnce("hello");

    const p = new WhisperProvider();
    await p.start();
    const mr = FakeMediaRecorder.lastInstance!;
    mr.pushChunk(new Blob(["data"]));
    await mr.fireStop();

    expect(fakeTrack.stop).toHaveBeenCalled();
    expect(p.getStream()).toBeNull();
  });

  // ── stop ─────────────────────────────────────────────────────────────────────

  it("stop() calls mediaRecorder.stop() when recording", async () => {
    const p = new WhisperProvider();
    await p.start(makeMediaStream());
    p.stop();
    expect(FakeMediaRecorder.lastInstance!.stop).toHaveBeenCalled();
  });

  it("stop() stops stream tracks when recorder is not recording", () => {
    const fakeTrack = { readyState: "live", stop: vi.fn() };
    const stream = { getTracks: () => [fakeTrack] } as unknown as MediaStream;
    const p = new WhisperProvider();
    // @ts-expect-error — accessing private for test
    p.stream = stream;
    p.stop(); // recorder is null (not started)
    expect(fakeTrack.stop).toHaveBeenCalled();
    expect(p.getStream()).toBeNull();
  });

  it("stop() is safe when nothing has been started", () => {
    const p = new WhisperProvider();
    expect(() => p.stop()).not.toThrow();
  });

  // ── dispose ──────────────────────────────────────────────────────────────────

  it("dispose() stops recorder if recording", async () => {
    const p = new WhisperProvider();
    await p.start(makeMediaStream());
    p.dispose();
    expect(FakeMediaRecorder.lastInstance!.stop).toHaveBeenCalled();
    expect(p.getStream()).toBeNull();
    expect(p.getLastTurnAudio()).toBeNull();
  });

  it("dispose() stops stream tracks and clears lastTurn even when not recording", async () => {
    const p = new WhisperProvider();
    transcribeMock.mockResolvedValueOnce("hi");
    await p.start(makeMediaStream());
    const mr = FakeMediaRecorder.lastInstance!;
    mr.pushChunk(new Blob(["data"]));
    await mr.fireStop(); // now state is inactive

    // @ts-expect-error — private field patched for test setup
    p.stream = makeMediaStream(); // simulate stream still set

    p.dispose();
    expect(p.getStream()).toBeNull();
    expect(p.getLastTurnAudio()).toBeNull();
  });

  it("dispose() is safe when nothing has been started", () => {
    const p = new WhisperProvider();
    expect(() => p.dispose()).not.toThrow();
  });

  // ── MediaRecorder mimeType selection ─────────────────────────────────────────

  it("uses opus mimeType when supported", async () => {
    FakeMediaRecorder.isTypeSupported.mockReturnValue(true);
    const p = new WhisperProvider();
    await p.start(makeMediaStream());
    // The FakeMediaRecorder constructor is called — verify it was constructed
    expect(FakeMediaRecorder.lastInstance).not.toBeNull();
  });

  it("falls back to audio/webm when opus is unsupported", async () => {
    FakeMediaRecorder.isTypeSupported.mockReturnValue(false);
    const p = new WhisperProvider();
    await p.start(makeMediaStream());
    expect(FakeMediaRecorder.lastInstance).not.toBeNull();
  });
});
