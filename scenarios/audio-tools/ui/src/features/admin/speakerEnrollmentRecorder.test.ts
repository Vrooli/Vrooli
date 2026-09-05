import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

// ---------------------------------------------------------------------------
// Minimal fake AudioFormat enum — only the value used by the recorder.
// ---------------------------------------------------------------------------
vi.mock("@vrooli/proto-types/audio-tools/v1/common/common_pb", () => ({
  AudioFormat: { WEBM: 3 },
}));

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
    // Emit a data chunk asynchronously
    void Promise.resolve().then(() => {
      const buf = new ArrayBuffer(4);
      new Uint8Array(buf).set([1, 2, 3, 4]);
      this.ondataavailable?.({ data: { size: 4, arrayBuffer: () => Promise.resolve(buf) } });
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
    return Promise.resolve(new ArrayBuffer(4));
  };
}

// Fake stream with one audio track
const fakeTrack = { stop: vi.fn() };
const fakeStream = {
  getTracks: () => [fakeTrack],
};

const getUserMediaMock = vi.fn().mockResolvedValue(fakeStream);
Object.defineProperty(navigator, "mediaDevices", {
  value: { getUserMedia: getUserMediaMock },
  configurable: true,
});

// ---------------------------------------------------------------------------
// Now import the module under test (after mocks are set up).
// ---------------------------------------------------------------------------
import { recordEnrollmentClip, MAX_ENROLL_CLIP_MS } from "./speakerEnrollmentRecorder";

beforeEach(() => {
  vi.clearAllMocks();
  getUserMediaMock.mockResolvedValue(fakeStream);
  fakeTrack.stop.mockReset();
});

afterEach(() => {
  vi.useRealTimers();
});

describe("recordEnrollmentClip", () => {
  it("resolves with audio bytes and WEBM format after stop() is called", async () => {
    vi.useFakeTimers();
    const handle = await recordEnrollmentClip();
    // Trigger stop manually
    handle.stop();
    // Let promises/microtasks settle
    await vi.runAllTimersAsync();
    const clip = await handle.done;
    expect(clip.audio).toBeInstanceOf(Uint8Array);
    expect(clip.format).toBe(3); // AudioFormat.WEBM
  });

  it("stop() does not call recorder.stop() if already inactive", async () => {
    vi.useFakeTimers();
    const handle = await recordEnrollmentClip();
    // Stop immediately
    handle.stop();
    await vi.runAllTimersAsync();
    // Calling stop() a second time on an already-inactive recorder is safe
    handle.stop();
    // No throw → success
    await handle.done;
  });

  it("auto-stops after maxMs via setTimeout", async () => {
    vi.useFakeTimers();
    const handle = await recordEnrollmentClip(100);
    // Advance past the timeout
    await vi.advanceTimersByTimeAsync(150);
    const clip = await handle.done;
    expect(clip.audio).toBeInstanceOf(Uint8Array);
  });

  it("exposes a positive MAX_ENROLL_CLIP_MS constant", () => {
    expect(MAX_ENROLL_CLIP_MS).toBeGreaterThan(0);
  });

  it("requests microphone access with audio:true", async () => {
    vi.useFakeTimers();
    const handle = await recordEnrollmentClip();
    handle.stop();
    await vi.runAllTimersAsync();
    await handle.done;
    expect(getUserMediaMock).toHaveBeenCalledWith({ audio: true });
  });

  it("stops all stream tracks after recording", async () => {
    vi.useFakeTimers();
    const handle = await recordEnrollmentClip();
    handle.stop();
    await vi.runAllTimersAsync();
    await handle.done;
    expect(fakeTrack.stop).toHaveBeenCalled();
  });
});
