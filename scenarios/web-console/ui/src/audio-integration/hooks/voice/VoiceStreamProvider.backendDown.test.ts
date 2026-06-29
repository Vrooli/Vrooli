/**
 * Backend-down error rendering (plan L2). When the speech backend (whisper) is
 * down, on-demand recovery runs server-side and the stream emits a typed, USER-
 * SAFE error: a clean, actionable message ("Speech backend (whisper) is
 * starting…") with a machine-readable `code` — never the raw
 * `dial tcp 127.0.0.1:8090: connect: connection refused` transport string.
 *
 * This pins the client contract: the server message reaches onError verbatim and
 * carries no transport detail.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

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
  send = vi.fn<(data: unknown) => void>();
  close = vi.fn(() => {
    this.readyState = FakeWebSocket.CLOSED;
  });
  constructor(url: string) {
    this.url = url;
    FakeWebSocket.instances.push(this);
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

function fakeStream(): MediaStream {
  return { getTracks: () => [{ readyState: "live", stop: () => {} }] } as unknown as MediaStream;
}

describe("VoiceStreamProvider backend-down error", () => {
  let provider: VoiceStreamProvider;
  let infoSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    FakeWebSocket.instances = [];
    FakeMediaRecorder.instances = [];
    (globalThis as unknown as { WebSocket: typeof FakeWebSocket }).WebSocket = FakeWebSocket;
    (globalThis as unknown as { MediaRecorder: typeof FakeMediaRecorder }).MediaRecorder = FakeMediaRecorder;
    Object.defineProperty(navigator, "mediaDevices", {
      configurable: true,
      value: { getUserMedia: vi.fn(async () => fakeStream()) },
    });
    infoSpy = vi.spyOn(console, "info").mockImplementation(() => {});
    provider = new VoiceStreamProvider();
  });

  afterEach(() => {
    provider.dispose();
    infoSpy.mockRestore();
  });

  it("surfaces the clean server message and never the raw dial string", async () => {
    const seen: string[] = [];
    provider.onError = (e) => seen.push(e);

    await provider.start();
    await Promise.resolve();
    const ws = FakeWebSocket.instances.at(-1);
    if (!ws) throw new Error("expected a WebSocket instance");

    const serverMessage = "Speech backend (whisper) is starting — please try again in a moment.";
    ws.onmessage?.({
      data: JSON.stringify({ type: "error", code: "backend_starting", text: serverMessage }),
    });

    expect(seen).toContain(serverMessage);
    for (const msg of seen) {
      expect(msg).not.toContain("dial tcp");
      expect(msg).not.toContain("connection refused");
    }
  });
});
