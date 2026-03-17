import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { WhisperProvider } from "../WhisperProvider";
import { WHISPER_FAILED_SENTINEL, AUDIO_BITRATE } from "../types";

// --- Mock infrastructure ---

const mockTrackStop = vi.fn();
const mockStream = {
  getTracks: () => [{ stop: mockTrackStop }],
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
    recorderInstance = this;
  }

  static isTypeSupported() { return true; }
}

function installMocks(micSuccess = true) {
  Object.defineProperty(navigator, "mediaDevices", {
    value: {
      getUserMedia: micSuccess
        ? vi.fn().mockResolvedValue(mockStream)
        : vi.fn().mockRejectedValue(new Error("Permission denied")),
    },
    configurable: true,
  });
  (globalThis as Record<string, unknown>).MediaRecorder = MockMediaRecorder;
}

// Mock the api module
vi.mock("../../../lib/api", () => ({
  transcribeAudioWithRetry: vi.fn(),
}));

describe("WhisperProvider", () => {
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

  it("acquires mic and starts MediaRecorder on start()", async () => {
    const provider = new WhisperProvider();
    provider.onResult = vi.fn();

    await provider.start();

    expect(provider.getStream()).toBe(mockStream);
    expect(recorderInstance).toBeTruthy();
    expect(recorderInstance!.start).toHaveBeenCalled();
  });

  it("uses AUDIO_BITRATE for MediaRecorder options", async () => {
    const provider = new WhisperProvider();
    provider.onResult = vi.fn();

    await provider.start();

    expect(recorderInstance!.options.audioBitsPerSecond).toBe(AUDIO_BITRATE);
  });

  it("calls onError when microphone access is denied", async () => {
    installMocks(false);
    const provider = new WhisperProvider();
    const onError = vi.fn();
    provider.onError = onError;

    await provider.start();

    expect(onError).toHaveBeenCalledWith("Microphone access denied");
    expect(provider.getStream()).toBeNull();
  });

  it("transcribes collected audio on stop", async () => {
    const { transcribeAudioWithRetry } = await import("../../../lib/api");
    (transcribeAudioWithRetry as ReturnType<typeof vi.fn>).mockResolvedValue("hello world");

    const provider = new WhisperProvider();
    const onResult = vi.fn();
    provider.onResult = onResult;

    await provider.start();

    // Simulate audio data
    const chunk = new Blob([new Uint8Array(100)], { type: "audio/webm" });
    recorderInstance!.ondataavailable?.({ data: chunk });

    // Stop triggers onstop which transcribes
    provider.stop();

    // Wait for async transcription
    await vi.waitFor(() => {
      expect(onResult).toHaveBeenCalledWith("hello world");
    });
  });

  it("calls onError with WHISPER_FAILED_SENTINEL when transcription fails", async () => {
    const { transcribeAudioWithRetry } = await import("../../../lib/api");
    (transcribeAudioWithRetry as ReturnType<typeof vi.fn>).mockRejectedValue(new Error("Network error"));

    const provider = new WhisperProvider();
    const onError = vi.fn();
    provider.onError = onError;

    await provider.start();

    const chunk = new Blob([new Uint8Array(100)], { type: "audio/webm" });
    recorderInstance!.ondataavailable?.({ data: chunk });

    provider.stop();

    await vi.waitFor(() => {
      expect(onError).toHaveBeenCalledWith(WHISPER_FAILED_SENTINEL);
    });
  });

  it("does not call onResult for empty audio blobs", async () => {
    const { transcribeAudioWithRetry } = await import("../../../lib/api");

    const provider = new WhisperProvider();
    const onResult = vi.fn();
    provider.onResult = onResult;

    await provider.start();

    // Stop without any audio data -> empty blob
    provider.stop();

    // Give async a chance to run
    await new Promise((r) => setTimeout(r, 50));

    expect(transcribeAudioWithRetry).not.toHaveBeenCalled();
    expect(onResult).not.toHaveBeenCalled();
  });

  it("releases mic tracks on stop", async () => {
    const provider = new WhisperProvider();
    provider.onResult = vi.fn();

    await provider.start();
    provider.stop();

    expect(mockTrackStop).toHaveBeenCalled();
  });

  it("dispose stops recording and releases mic", async () => {
    const provider = new WhisperProvider();
    provider.onResult = vi.fn();

    await provider.start();
    provider.dispose();

    expect(mockTrackStop).toHaveBeenCalled();
    expect(provider.getStream()).toBeNull();
  });

  it("passes language to transcribeAudioWithRetry", async () => {
    const { transcribeAudioWithRetry } = await import("../../../lib/api");
    (transcribeAudioWithRetry as ReturnType<typeof vi.fn>).mockResolvedValue("bonjour");

    const provider = new WhisperProvider();
    provider.language = "fr";
    provider.onResult = vi.fn();

    await provider.start();

    const chunk = new Blob([new Uint8Array(100)], { type: "audio/webm" });
    recorderInstance!.ondataavailable?.({ data: chunk });

    provider.stop();

    await vi.waitFor(() => {
      expect(transcribeAudioWithRetry).toHaveBeenCalledWith(expect.any(Blob), 2, "fr");
    });
  });
});
