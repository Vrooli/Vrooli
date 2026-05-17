/**
 * Tail-drop arming for the auto-stop path. See plan
 * /home/matthalloran8/.vrooli/plans/audio-tools-auto-stop-tail-drop.md.
 *
 * The contract:
 *  1. After dropTail(), subsequent ondataavailable blobs are NOT sent to the
 *     WebSocket (they would be words the user spoke after the visual stop).
 *  2. stop() while armed sends {type:"done"} synchronously (no waiting for
 *     the encoder's onstop) and skips snapshotLastTurn (no tail retention).
 *  3. start() resets the flag — subsequent ondataavailable resumes sending.
 */
import { beforeEach, afterEach, describe, expect, it, vi } from "vitest";

import { VoiceStreamProvider } from "./VoiceStreamProvider";

class FakeWebSocket {
  static OPEN = 1;
  static CLOSED = 3;
  static CONNECTING = 0;
  static instances: FakeWebSocket[] = [];
  readyState = FakeWebSocket.OPEN;
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
    queueMicrotask(() => {
      this.readyState = FakeWebSocket.OPEN;
      this.onopen?.();
    });
  }
}

class FakeMediaRecorder {
  static instances: FakeMediaRecorder[] = [];
  static isTypeSupported = vi.fn(() => true);
  state: "inactive" | "recording" = "inactive";
  ondataavailable: ((e: { data: Blob }) => void) | null = null;
  onstop: (() => void) | null = null;
  start = vi.fn(() => {
    this.state = "recording";
  });
  stop = vi.fn(() => {
    this.state = "inactive";
    this.onstop?.();
  });
  constructor(_stream: MediaStream, _opts?: unknown) {
    FakeMediaRecorder.instances.push(this);
  }
}

function fakeBlob(size: number): Blob {
  return {
    size,
    type: "audio/webm",
    // Provide arrayBuffer() so the prod-path microtask resolves.
    arrayBuffer: () => Promise.resolve(new ArrayBuffer(size)),
  } as unknown as Blob;
}

function fakeStream(): MediaStream {
  return {
    getTracks: () => [{ readyState: "live", stop: () => {} }],
  } as unknown as MediaStream;
}

describe("VoiceStreamProvider tail-drop", () => {
  let provider: VoiceStreamProvider;
  let infoSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    FakeWebSocket.instances = [];
    FakeMediaRecorder.instances = [];
    (globalThis as unknown as { WebSocket: typeof FakeWebSocket }).WebSocket = FakeWebSocket;
    (globalThis as unknown as { MediaRecorder: typeof FakeMediaRecorder }).MediaRecorder =
      FakeMediaRecorder;
    Object.defineProperty(navigator, "mediaDevices", {
      configurable: true,
      value: { getUserMedia: vi.fn(async () => fakeStream()) },
    });
    // console.info is allowed by test-setup; silence it here so the
    // tail-drop log line doesn't clutter test output.
    infoSpy = vi.spyOn(console, "info").mockImplementation(() => {});
    provider = new VoiceStreamProvider();
  });

  afterEach(() => {
    provider.dispose();
    infoSpy.mockRestore();
  });

  async function startAndOpenWs(): Promise<{ ws: FakeWebSocket; rec: FakeMediaRecorder }> {
    await provider.start();
    // Drain microtask queue so the FakeWebSocket onopen fires.
    await Promise.resolve();
    const ws = FakeWebSocket.instances.at(-1)!;
    const rec = FakeMediaRecorder.instances.at(-1)!;
    return { ws, rec };
  }

  it("drops ondataavailable blobs after dropTail() is armed", async () => {
    const { ws, rec } = await startAndOpenWs();
    // Pre-arm: a chunk should be forwarded.
    rec.ondataavailable?.({ data: fakeBlob(1024) });
    await Promise.resolve(); // resolve the arrayBuffer().then microtask
    const sendsBeforeArm = ws.send.mock.calls.length;
    expect(sendsBeforeArm).toBeGreaterThanOrEqual(1);

    provider.dropTail();
    rec.ondataavailable?.({ data: fakeBlob(2048) });
    await Promise.resolve();
    // No additional send after arming.
    expect(ws.send.mock.calls.length).toBe(sendsBeforeArm);
  });

  it("sends {type:done} synchronously and skips tail retention when armed", async () => {
    const { ws, rec } = await startAndOpenWs();
    // Push one chunk pre-arm so allChunks has something snapshotLastTurn
    // *could* retain — proving the skip actually skips.
    rec.ondataavailable?.({ data: fakeBlob(1024) });
    await Promise.resolve();
    ws.send.mockClear();

    provider.dropTail();
    provider.stop();
    // done message went out before any onstop callback fired.
    const doneCalls = ws.send.mock.calls.filter((args) => {
      const payload = args[0];
      return typeof payload === "string" && payload.includes('"done"');
    });
    expect(doneCalls.length).toBe(1);
    // The encoder may still emit a final ondataavailable; gate must drop it.
    rec.ondataavailable?.({ data: fakeBlob(4096) });
    await Promise.resolve();
    expect(ws.send.mock.calls.length).toBe(doneCalls.length);
    // No retained tail.
    expect(provider.getLastTurnAudio()).toBeNull();
  });

  it("clears the tail-drop flag on the next start()", async () => {
    const first = await startAndOpenWs();
    provider.dropTail();
    provider.stop();
    // Fresh session.
    const second = await startAndOpenWs();
    expect(second.rec).not.toBe(first.rec);
    second.ws.send.mockClear();
    second.rec.ondataavailable?.({ data: fakeBlob(512) });
    await Promise.resolve();
    // Send happens again; the new session is not still armed.
    expect(second.ws.send.mock.calls.length).toBeGreaterThanOrEqual(1);
  });
});
