import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

vi.mock("@vrooli/proto-types/audio-tools/v1/common/common_pb", () => ({
  AudioFormat: { WEBM: 3 },
}));

// WakeWordSample type comes from services/wakeWord; no runtime value needed.
vi.mock("../../services/wakeWord", () => ({}));

// ---------------------------------------------------------------------------
// Fake MediaRecorder
// ---------------------------------------------------------------------------
class FakeMediaRecorder {
  static isTypeSupported() { return true; }
  state: "inactive" | "recording" = "inactive";
  mimeType = "audio/webm";
  ondataavailable: ((ev: { data: { size: number; arrayBuffer(): Promise<ArrayBuffer> } }) => void) | null = null;
  onstop: (() => void) | null = null;

  start() {
    this.state = "recording";
    void Promise.resolve().then(() => {
      const buf = new ArrayBuffer(2);
      new Uint8Array(buf).set([0xde, 0xad]);
      this.ondataavailable?.({ data: { size: 2, arrayBuffer: () => Promise.resolve(buf) } });
    });
  }

  stop() {
    this.state = "inactive";
    void Promise.resolve().then(() => this.onstop?.());
  }
}

vi.stubGlobal("MediaRecorder", FakeMediaRecorder);

// jsdom's Blob has no arrayBuffer(); the recorder calls it on the assembled clip.
if (typeof Blob.prototype.arrayBuffer !== "function") {
  Blob.prototype.arrayBuffer = function arrayBuffer() {
    return Promise.resolve(new ArrayBuffer(2));
  };
}

// ---------------------------------------------------------------------------
// Fake AudioContext
// ---------------------------------------------------------------------------
class FakeAudioContext {
  sampleRate = 16_000;
  close() { return Promise.resolve(); }
}

vi.stubGlobal("AudioContext", FakeAudioContext);

// Fake stream
const fakeTrack = { stop: vi.fn() };
const fakeStream = { getTracks: () => [fakeTrack] };

const getUserMediaMock = vi.fn().mockResolvedValue(fakeStream);
Object.defineProperty(navigator, "mediaDevices", {
  value: { getUserMedia: getUserMediaMock },
  configurable: true,
});

import { recordWakeWordSample } from "./wakeWordRecorder";

beforeEach(() => {
  vi.clearAllMocks();
  getUserMediaMock.mockResolvedValue(fakeStream);
  fakeTrack.stop.mockReset();
});

afterEach(() => {
  vi.useRealTimers();
});

describe("recordWakeWordSample", () => {
  it("resolves with audio bytes, WEBM format, and sampleRateHz after stop()", async () => {
    vi.useFakeTimers();
    const handle = await recordWakeWordSample();
    handle.stop();
    await vi.runAllTimersAsync();
    const sample = await handle.done;
    expect(sample.audio).toBeInstanceOf(Uint8Array);
    expect(sample.format).toBe(3); // AudioFormat.WEBM
    expect(sample.sampleRateHz).toBe(16_000);
  });

  it("auto-stops after the 3 second cap", async () => {
    vi.useFakeTimers();
    const handle = await recordWakeWordSample();
    await vi.advanceTimersByTimeAsync(4_000);
    const sample = await handle.done;
    expect(sample.audio).toBeInstanceOf(Uint8Array);
  });

  it("requests getUserMedia with audio:true", async () => {
    vi.useFakeTimers();
    const handle = await recordWakeWordSample();
    handle.stop();
    await vi.runAllTimersAsync();
    await handle.done;
    expect(getUserMediaMock).toHaveBeenCalledWith({ audio: true });
  });

  it("stops all stream tracks when recording ends", async () => {
    vi.useFakeTimers();
    const handle = await recordWakeWordSample();
    handle.stop();
    await vi.runAllTimersAsync();
    await handle.done;
    expect(fakeTrack.stop).toHaveBeenCalled();
  });

  it("calling stop() a second time is safe (state guard)", async () => {
    vi.useFakeTimers();
    const handle = await recordWakeWordSample();
    handle.stop();
    await vi.runAllTimersAsync();
    await handle.done;
    // Second call should not throw
    expect(() => handle.stop()).not.toThrow();
  });
});
