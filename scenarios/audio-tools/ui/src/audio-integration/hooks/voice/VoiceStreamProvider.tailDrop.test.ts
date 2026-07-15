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
import { loadUnfinishedSession } from "@vrooli/audio-capture-browser";
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

async function waitForSocketWrites(ws: FakeWebSocket, count = 1): Promise<void> {
  await vi.waitFor(() => expect(ws.send.mock.calls.length).toBeGreaterThanOrEqual(count));
}

  it("drops captured frames after dropTail() is armed", async () => {
    const ws = await startAndOpenWs();
    // Pre-arm: a frame should be forwarded.
    pushFrame();
    await waitForSocketWrites(ws);
    const sendsBeforeArm = ws.send.mock.calls.length;
    expect(sendsBeforeArm).toBeGreaterThanOrEqual(1);

    provider.dropTail();
    pushFrame();
    // No additional send after arming.
    expect(ws.send.mock.calls.length).toBe(sendsBeforeArm);
  });

  it("[REQ:ATD-P0-001] delivers a replayed durable segment identity once", async () => {
    const onSegmentFinal = vi.fn();
    provider.onSegmentFinal = onSegmentFinal;
    const ws = await startAndOpenWs();
    const durableSegment = { type: "segment-final", text: "replayed once", segmentIndex: 0, segmentId: "turn-1:0:0:1" };

    ws.onmessage?.({ data: JSON.stringify(durableSegment) });
    ws.onmessage?.({ data: JSON.stringify(durableSegment) });

    expect(onSegmentFinal).toHaveBeenCalledTimes(1);
    expect(onSegmentFinal).toHaveBeenCalledWith("replayed once", 0);
  });

  it("[REQ:ATD-P0-004] sends versioned binary frames with PCM coverage metadata", async () => {
    const ws = await startAndOpenWs();
    pushFrame(128);
    await waitForSocketWrites(ws);
    const lastCall = ws.send.mock.calls.at(-1);
    expect(lastCall).toBeDefined();
    const payload = lastCall![0] as ArrayBuffer;
    expect(payload).toBeInstanceOf(ArrayBuffer);
    const bytes = new Uint8Array(payload);
    expect(new TextDecoder().decode(bytes.slice(0, 4))).toBe("ATV2");
    // 60-byte v2 header (including SHA-256) + 128 samples * 2 bytes/sample.
    expect(payload.byteLength).toBe(316);
  });

  it("sends {type:done} and skips tail retention when armed", async () => {
    const ws = await startAndOpenWs();
    // Push one frame pre-arm so allPcm has something snapshotLastTurn
    // *could* retain — proving the skip actually skips.
    pushFrame();
    ws.send.mockClear();

    provider.dropTail();
    provider.stop();
    await waitForSocketWrites(ws);
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
    await waitForSocketWrites(ws, 2);

    const payloads = ws.send.mock.calls.map((args) => args[0]);
    expect(payloads[0]).toBeInstanceOf(ArrayBuffer);
    expect(payloads[1]).toBe(JSON.stringify({ type: "done" }));
    expect(payloads.filter((payload) => payload === JSON.stringify({ type: "done" })).length).toBe(1);
  });

  it("[REQ:ATD-P0-004] replays journaled v2 coverage after a reconnect", async () => {
    vi.useFakeTimers();
    try {
      const first = await startAndOpenWs();
      pushFrame(128);
      await waitForSocketWrites(first);
      const original = first.send.mock.calls[0]![0] as ArrayBuffer;

      first.readyState = FakeWebSocket.CLOSED;
      first.onclose?.();
      await vi.advanceTimersByTimeAsync(1_000);
      await Promise.resolve();

      const second = FakeWebSocket.instances.at(-1)!;
      await waitForSocketWrites(second);
      const replay = second.send.mock.calls[0]![0] as ArrayBuffer;
      expect(Array.from(new Uint8Array(replay))).toEqual(Array.from(new Uint8Array(original)));
    } finally {
      vi.useRealTimers();
    }
  });

  it("[REQ:ATD-P0-001] compacts processed coverage so reconnects replay only missing audio", async () => {
    vi.useFakeTimers();
    try {
      const first = await startAndOpenWs();
      pushFrame(128);
      pushFrame(128);
      await waitForSocketWrites(first, 2);

      first.onmessage?.({
        data: JSON.stringify({
          type: "status",
          code: "processed_acknowledgement",
          processedSequence: 0,
        }),
      });
      // The acknowledgement is serialized after the frame journal write.
      await Promise.resolve();
      await Promise.resolve();

      first.readyState = FakeWebSocket.CLOSED;
      first.onclose?.();
      await vi.advanceTimersByTimeAsync(1_000);
      const second = FakeWebSocket.instances.at(-1)!;
      await waitForSocketWrites(second);

      const replay = second.send.mock.calls[0]![0] as ArrayBuffer;
      expect(new DataView(replay).getBigUint64(4, false)).toBe(1n);
      expect(second.send.mock.calls).toHaveLength(1);
    } finally {
      vi.useRealTimers();
    }
  });

  it("[REQ:ATD-P0-004] retains captured audio when final follows incomplete coverage", async () => {
    const onResult = vi.fn();
    const onError = vi.fn();
    provider.onResult = onResult;
    provider.onError = onError;
    const ws = await startAndOpenWs();
    pushFrame(128);
    await waitForSocketWrites(ws);

    ws.onmessage?.({ data: JSON.stringify({ type: "error", code: "incomplete_coverage", text: "Audio processing acknowledgement was intentionally withheld." }) });
    ws.onmessage?.({ data: JSON.stringify({ type: "final", text: "must not be captured" }) });
    await Promise.resolve();
    await Promise.resolve();

    expect(onError).toHaveBeenCalledOnce();
    expect(onResult).not.toHaveBeenCalled();
    expect(loadUnfinishedSession()).toMatchObject({ sessionId: expect.any(String), resumeToken: expect.any(String) });
    expect(provider.getDiagnostic()).toMatchObject({ state: "failed", terminalReason: "incomplete_coverage" });
  });

  it("[REQ:ATD-P0-004] discards recovery state only after a terminal final", async () => {
    const onResult = vi.fn();
    provider.onResult = onResult;
    const ws = await startAndOpenWs();
    pushFrame(128);
    await waitForSocketWrites(ws);
    provider.stop();

    ws.onmessage?.({ data: JSON.stringify({ type: "final", text: "committed transcript" }) });
    await Promise.resolve();

    expect(onResult).toHaveBeenCalledWith("committed transcript");
    expect(loadUnfinishedSession()).toBeNull();
    ws.readyState = FakeWebSocket.CLOSED;
    ws.onclose?.();
    expect(vi.mocked(transcribeAudioWithRetry)).not.toHaveBeenCalled();
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
    await waitForSocketWrites(second);
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

  it("reports server status messages and clears the ack watchdog", async () => {
    vi.useFakeTimers();
    try {
      const onStatus = vi.fn();
      provider.onStatus = onStatus;
      const ws = await startAndOpenWs();

      ws.onmessage?.({
        data: JSON.stringify({
          type: "status",
          code: "stream_connected",
          text: "Streaming transcription connected.",
        }),
      });
      await vi.advanceTimersByTimeAsync(1500);

      expect(onStatus).toHaveBeenCalledWith({
        code: "stream_connected",
        message: "Streaming transcription connected.",
      });
      expect(onStatus).not.toHaveBeenCalledWith(expect.objectContaining({ code: "server_ack_pending" }));
    } finally {
      vi.useRealTimers();
    }
  });

  it("reports when the server does not acknowledge an opened stream", async () => {
    vi.useFakeTimers();
    try {
      const onStatus = vi.fn();
      provider.onStatus = onStatus;
      await startAndOpenWs();

      await vi.advanceTimersByTimeAsync(1500);

      expect(onStatus).toHaveBeenCalledWith({
        code: "server_ack_pending",
        message: "Waiting for the speech backend to acknowledge the stream.",
      });
    } finally {
      vi.useRealTimers();
    }
  });

  it("reports finalization progress before the last-resort final timeout", async () => {
    vi.useFakeTimers();
    try {
      const onStatus = vi.fn();
      provider.onStatus = onStatus;
      vi.mocked(transcribeAudioWithRetry).mockResolvedValue("timeout fallback transcript");
      await startAndOpenWs();
      pushFrame();
      provider.stop();

      await vi.advanceTimersByTimeAsync(3000);

      expect(onStatus).toHaveBeenCalledWith({
        code: "final_pending",
        message: "Speech audio was sent; waiting for the backend to finish transcription.",
      });
      expect(transcribeAudioWithRetry).not.toHaveBeenCalled();
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
