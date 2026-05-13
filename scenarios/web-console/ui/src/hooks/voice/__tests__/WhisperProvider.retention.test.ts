// Tests for WhisperProvider retention of the last completed turn's audio.

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { WhisperProvider } from "../WhisperProvider";

function defined<T>(val: T | null | undefined): T {
  if (val == null) throw new Error("Expected defined value");
  return val;
}

const mockTrackStop = vi.fn();
const mockStream = {
  getTracks: () => [{ stop: mockTrackStop, readyState: "live" }],
} as unknown as MediaStream;

let recorderInstance: {
  state: string;
  ondataavailable: ((e: { data: Blob }) => void) | null;
  onstop: (() => void) | null;
  start: ReturnType<typeof vi.fn>;
  stop: ReturnType<typeof vi.fn>;
  options: MediaRecorderOptions;
} | null = null;

class MockMediaRecorder {
  state = "inactive";
  ondataavailable: ((e: { data: Blob }) => void) | null = null;
  onstop: (() => void) | null = null;
  start = vi.fn(() => { this.state = "recording"; });
  stop = vi.fn(() => {
    this.state = "inactive";
    this.onstop?.();
  });
  options: MediaRecorderOptions;

  constructor(_stream: MediaStream, options?: MediaRecorderOptions) {
    this.options = options ?? {};
    recorderInstance = this as unknown as typeof recorderInstance;
  }

  static isTypeSupported() { return true; }
}

function installMocks() {
  Object.defineProperty(navigator, "mediaDevices", {
    value: { getUserMedia: vi.fn().mockResolvedValue(mockStream) },
    configurable: true,
  });
  (globalThis as Record<string, unknown>).MediaRecorder = MockMediaRecorder;
}

vi.mock("../../../api/voice", () => ({
  transcribeAudioWithRetry: vi.fn().mockResolvedValue(""),
}));

describe("WhisperProvider retention", () => {
  let originalMediaRecorder: unknown;

  beforeEach(() => {
    originalMediaRecorder = (globalThis as Record<string, unknown>).MediaRecorder;
    recorderInstance = null;
    mockTrackStop.mockClear();
    vi.clearAllMocks();
    installMocks();
  });

  afterEach(() => {
    (globalThis as Record<string, unknown>).MediaRecorder = originalMediaRecorder;
  });

  it("returns null before any recording starts", () => {
    const provider = new WhisperProvider();
    expect(provider.getLastTurnAudio()).toBeNull();
  });

  it("retains the combined blob after stop()", async () => {
    const provider = new WhisperProvider();
    provider.onResult = vi.fn();

    await provider.start();

    // Two chunks of known size.
    const recorder = defined(recorderInstance);
    const chunkA = new Blob([new Uint8Array(100)], { type: "audio/webm" });
    const chunkB = new Blob([new Uint8Array(50)], { type: "audio/webm" });
    recorder.ondataavailable?.({ data: chunkA });
    recorder.ondataavailable?.({ data: chunkB });

    provider.stop();
    await vi.waitFor(() => {
      const retained = provider.getLastTurnAudio();
      expect(retained).not.toBeNull();
    });

    const retained = defined(provider.getLastTurnAudio());
    expect(retained.blob.size).toBe(150);
    expect(retained.mimeType).toMatch(/audio\/webm/);
    expect(retained.durationMs).toBeGreaterThanOrEqual(0);
    expect(retained.capturedAt).toBeGreaterThan(0);
  });

  it("clears retained audio on disposeLastTurn()", async () => {
    const provider = new WhisperProvider();
    provider.onResult = vi.fn();

    await provider.start();
    defined(recorderInstance).ondataavailable?.({ data: new Blob([new Uint8Array(80)]) });
    provider.stop();

    await vi.waitFor(() => expect(provider.getLastTurnAudio()).not.toBeNull());

    provider.disposeLastTurn();
    expect(provider.getLastTurnAudio()).toBeNull();
  });

  it("replaces retained audio on the next start() (single-slot retention)", async () => {
    const provider = new WhisperProvider();
    provider.onResult = vi.fn();

    await provider.start();
    defined(recorderInstance).ondataavailable?.({ data: new Blob([new Uint8Array(200)]) });
    provider.stop();
    await vi.waitFor(() => expect(provider.getLastTurnAudio()).not.toBeNull());
    const first = defined(provider.getLastTurnAudio());
    expect(first.blob.size).toBe(200);

    // Second turn. The MockMediaRecorder constructor replaces
    // recorderInstance with the fresh instance, so we read it again after
    // provider.start() to avoid racing with the previous turn's reference.
    await provider.start();
    // Between start() beginning and the new blob arriving, the previous
    // retention must already be dropped — no leak across turns.
    expect(provider.getLastTurnAudio()).toBeNull();

    const recorder2 = defined(recorderInstance);
    recorder2.ondataavailable?.({ data: new Blob([new Uint8Array(30)]) });
    provider.stop();

    await vi.waitFor(() => expect(provider.getLastTurnAudio()?.blob.size).toBe(30));
  });

  it("does not retain audio for an empty recording", async () => {
    const provider = new WhisperProvider();
    provider.onResult = vi.fn();

    await provider.start();
    // No ondataavailable calls — the user released the mic without speaking.
    provider.stop();

    // Give the onstop handler a chance to run.
    await new Promise((r) => setTimeout(r, 20));
    expect(provider.getLastTurnAudio()).toBeNull();
  });

  it("dispose() drops retained audio", async () => {
    const provider = new WhisperProvider();
    provider.onResult = vi.fn();

    await provider.start();
    defined(recorderInstance).ondataavailable?.({ data: new Blob([new Uint8Array(50)]) });
    provider.stop();
    await vi.waitFor(() => expect(provider.getLastTurnAudio()).not.toBeNull());

    provider.dispose();
    expect(provider.getLastTurnAudio()).toBeNull();
  });
});
