/**
 * Tail-drop arming for the auto-stop path. See plan
 * /home/matthalloran8/.vrooli/plans/audio-tools-auto-stop-tail-drop.md.
 *
 * The contract (now over the PCM capture seam rather than MediaRecorder):
 *  1. After dropTail(), subsequent captured PCM frames are NOT sent to the
 *     WebSocket (they would be words the user spoke after the visual stop).
 *  2. stop() while armed sends {type:"done"} and skips snapshotLastTurn
 *     (no tail retention).
 *  3. start() resets the flag — subsequent frames resume sending.
 */
import { beforeEach, afterEach, describe, expect, it, vi } from "vitest";

import { transcribeAudioWithRetry } from "../../api/voice";
import { VoiceStreamProvider } from "./VoiceStreamProvider";
import type { PcmCapture, PcmCaptureFactory } from "./pcmCapture";

vi.mock("../../api/voice", () => ({
  buildVoiceStreamWsUrl: vi.fn(() => "ws://audio-tools.test/api/v1/voice/stream?format=pcm_s16le"),
  transcribeAudioWithRetry: vi.fn(),
}));

class FakeWebSocket {
  static OPEN = 1;
  static CLOSED = 3;
  static CONNECTING = 0;
  static instances: FakeWebSocket[] = [];
  static autoOpen = true;
  readyState = FakeWebSocket.CONNECTING;
  url: string;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onmessage: ((e: { data: string }) => void) | null = null;
  send = vi.fn();
  close = vi.fn(() => {
    this.readyState = FakeWebSocket.CLOSED;
  });
  constructor(url: string) {
    this.url = url;
    FakeWebSocket.instances.push(this);
    // Open synchronously to match the prod fast-path
    if (FakeWebSocket.autoOpen) queueMicrotask(() => this.open());
  }

  open(): void {
    this.readyState = FakeWebSocket.OPEN;
    this.onopen?.();
  }
}

// Fake PCM capture seam: records the onFrame callback so the test can push
// synthetic frames, and tracks stop() calls. Replaces the real
// AudioContext/ScriptProcessor wiring (unavailable in jsdom).
let currentOnFrame: ((samples: Float32Array, sampleRate: number) => void) | null = null;
let captureStops = 0;
const fakeCaptureFactory: PcmCaptureFactory = (_stream, onFrame): PcmCapture => {
  currentOnFrame = onFrame;
  return {
    stop() {
      captureStops++;
    },
  };
};

/** Push a synthetic 16 kHz frame (identity path, no resampling). */
function pushFrame(samples = 256): void {
  currentOnFrame?.(new Float32Array(samples).fill(0.5), 16_000);
}

function fakeStream(): MediaStream {
  return {
    getTracks: () => [{ readyState: "live", stop: () => {} }],
  } as unknown as MediaStream;
}

describe("VoiceStreamProvider tail-drop", () => {
  let provider: VoiceStreamProvider;
  let infoSpy: ReturnType<typeof vi.spyOn>;
  let warnSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    FakeWebSocket.instances = [];
    FakeWebSocket.autoOpen = true;
    currentOnFrame = null;
    captureStops = 0;
    (globalThis as unknown as { WebSocket: typeof FakeWebSocket }).WebSocket = FakeWebSocket;
    Object.defineProperty(navigator, "mediaDevices", {
      configurable: true,
      value: { getUserMedia: vi.fn(() => Promise.resolve(fakeStream())) },
    });
    // console.info is allowed by test-setup; silence it here so the
    // tail-drop log line doesn't clutter test output.
    infoSpy = vi.spyOn(console, "info").mockImplementation(() => {});
    warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    provider = new VoiceStreamProvider();
    provider.captureFactory = fakeCaptureFactory;
  });

  afterEach(() => {
    provider.dispose();
    infoSpy.mockRestore();
    warnSpy.mockRestore();
    vi.mocked(transcribeAudioWithRetry).mockReset();
  });

  async function startAndOpenWs(): Promise<FakeWebSocket> {
    await provider.start();
    // Drain microtask queue so the FakeWebSocket onopen fires.
    await Promise.resolve();
    return FakeWebSocket.instances[FakeWebSocket.instances.length - 1]!;
  }

  it("drops captured frames after dropTail() is armed", async () => {
    const ws = await startAndOpenWs();
    // Pre-arm: a frame should be forwarded.
    pushFrame();
    const sendsBeforeArm = ws.send.mock.calls.length;
    expect(sendsBeforeArm).toBeGreaterThanOrEqual(1);

    provider.dropTail();
    pushFrame();
    // No additional send after arming.
    expect(ws.send.mock.calls.length).toBe(sendsBeforeArm);
  });

  it("sends binary PCM frames (s16le bytes) over the socket", async () => {
    const ws = await startAndOpenWs();
    pushFrame(128);
    const lastCall = ws.send.mock.calls.at(-1);
    expect(lastCall).toBeDefined();
    const payload = lastCall![0] as ArrayBufferView;
    expect(ArrayBuffer.isView(payload)).toBe(true);
    // 128 samples * 2 bytes/sample of s16le PCM.
    expect(payload.byteLength).toBe(256);
  });

  it("sends {type:done} and skips tail retention when armed", async () => {
    const ws = await startAndOpenWs();
    // Push one frame pre-arm so allPcm has something snapshotLastTurn
    // *could* retain — proving the skip actually skips.
    pushFrame();
    ws.send.mockClear();

    provider.dropTail();
    provider.stop();
    const doneCalls = ws.send.mock.calls.filter((args) => {
      const payload = args[0];
      return typeof payload === "string" && payload.includes('"done"');
    });
    expect(doneCalls.length).toBe(1);
    // Capture is stopped, so no more frames arrive; no retained tail.
    expect(provider.getLastTurnAudio()).toBeNull();
  });

  it("sends buffered PCM followed by done when stop happens before WebSocket open", async () => {
    FakeWebSocket.autoOpen = false;
    await provider.start();
    const ws = FakeWebSocket.instances.at(-1)!;
    expect(ws.readyState).toBe(FakeWebSocket.CONNECTING);

    pushFrame();
    provider.stop();
    expect(ws.send).not.toHaveBeenCalled();

    ws.open();

    const payloads = ws.send.mock.calls.map((args) => args[0]);
    expect(ArrayBuffer.isView(payloads[0])).toBe(true);
    expect(payloads[1]).toBe(JSON.stringify({ type: "done" }));
    expect(payloads.filter((payload) => payload === JSON.stringify({ type: "done" })).length).toBe(1);
  });

  it("retains the turn audio as a WAV blob when not armed", async () => {
    const ws = await startAndOpenWs();
    pushFrame();
    provider.stop();
    void ws;
    const retained = provider.getLastTurnAudio();
    expect(retained).not.toBeNull();
    expect(retained!.mimeType).toBe("audio/wav");
    expect(retained!.blob.size).toBeGreaterThan(44); // header + samples
  });

  it("clears the tail-drop flag on the next start()", async () => {
    await startAndOpenWs();
    provider.dropTail();
    provider.stop();
    // Fresh session.
    const second = await startAndOpenWs();
    second.send.mockClear();
    pushFrame();
    // Send happens again; the new session is not still armed.
    expect(second.send.mock.calls.length).toBeGreaterThanOrEqual(1);
  });

  it("stops the capture on stop()", async () => {
    await startAndOpenWs();
    const before = captureStops;
    provider.stop();
    expect(captureStops).toBe(before + 1);
  });

  it("falls back to HTTP transcription when audio streamed but no final arrives", async () => {
    vi.useFakeTimers();
    try {
      const onError = vi.fn();
      const onResult = vi.fn();
      provider.onError = onError;
      provider.onResult = onResult;
      vi.mocked(transcribeAudioWithRetry).mockResolvedValue("timeout fallback transcript");
      await startAndOpenWs();
      pushFrame();
      provider.stop();

      await vi.advanceTimersByTimeAsync(10_000);
      await Promise.resolve();

      expect(transcribeAudioWithRetry).toHaveBeenCalledWith(expect.any(Blob), 2, "en");
      expect(onResult).toHaveBeenCalledWith("timeout fallback transcript");
      expect(onError).not.toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
    }
  });

  it("falls back to HTTP transcription after WebSocket reconnects are exhausted", async () => {
    vi.useFakeTimers();
    try {
      const onResult = vi.fn();
      provider.onResult = onResult;
      vi.mocked(transcribeAudioWithRetry).mockResolvedValue("fallback transcript");
      const first = await startAndOpenWs();
      pushFrame();

      first.readyState = FakeWebSocket.CLOSED;
      first.onclose?.();
      await vi.advanceTimersByTimeAsync(1_000);
      const second = FakeWebSocket.instances.at(-1)!;
      await Promise.resolve();
      pushFrame();

      second.readyState = FakeWebSocket.CLOSED;
      second.onclose?.();
      await vi.advanceTimersByTimeAsync(3_000);
      const third = FakeWebSocket.instances.at(-1)!;
      await Promise.resolve();

      third.readyState = FakeWebSocket.CLOSED;
      third.onclose?.();
      await Promise.resolve();

      expect(transcribeAudioWithRetry).toHaveBeenCalledWith(expect.any(Blob), 2, "en");
      await expect(vi.mocked(transcribeAudioWithRetry).mock.results[0]!.value).resolves.toBe("fallback transcript");
      expect(onResult).toHaveBeenCalledWith("fallback transcript");
    } finally {
      vi.useRealTimers();
    }
  });
});
