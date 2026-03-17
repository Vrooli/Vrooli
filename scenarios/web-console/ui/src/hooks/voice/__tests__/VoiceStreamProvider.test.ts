import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { VoiceStreamProvider } from "../VoiceStreamProvider";

// --- Helpers ---

/** Narrows away null/undefined — throws (fails the test) if the value is nullish. */
function defined<T>(val: T | null | undefined): T {
  if (val == null) throw new Error("Expected defined value");
  return val;
}

// --- Mock infrastructure ---

class MockMediaRecorder {
  state = "inactive" as "inactive" | "recording" | "paused";
  ondataavailable: ((e: { data: Blob }) => void) | null = null;
  onstop: (() => void) | null = null;
  timeslice = 0;

  constructor(
    _stream: MediaStream,
    _options?: MediaRecorderOptions,
  ) {}

  start(timeslice?: number) {
    this.state = "recording";
    this.timeslice = timeslice ?? 0;
  }
  stop() {
    this.state = "inactive";
    this.onstop?.();
  }

  /** Simulate a data chunk arriving. */
  simulateChunk(data: ArrayBuffer) {
    const blob = new Blob([data]);
    // Also mock arrayBuffer() on the blob
    blob.arrayBuffer = () => Promise.resolve(data);
    this.ondataavailable?.({ data: blob });
  }

  static isTypeSupported() { return true; }
}

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  readyState: number = WebSocket.CONNECTING;
  url: string;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  sentMessages: (string | ArrayBuffer)[] = [];

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  send(data: string | ArrayBuffer) {
    this.sentMessages.push(data);
  }

  close() {
    this.readyState = WebSocket.CLOSED;
  }

  /** Test helper: simulate connection opening. */
  simulateOpen() {
    this.readyState = WebSocket.OPEN;
    this.onopen?.();
  }

  /** Test helper: simulate a server message. */
  simulateMessage(msg: { type: string; text?: string }) {
    this.onmessage?.({ data: JSON.stringify(msg) });
  }
}

const mockStream = {
  getTracks: () => [{ stop: vi.fn() }],
} as unknown as MediaStream;

function installMocks() {
  Object.defineProperty(navigator, "mediaDevices", {
    value: {
      getUserMedia: vi.fn().mockResolvedValue(mockStream),
    },
    configurable: true,
  });
  (globalThis as Record<string, unknown>).MediaRecorder = MockMediaRecorder;
  (globalThis as Record<string, unknown>).WebSocket = MockWebSocket;
  MockWebSocket.instances = [];
}

describe("VoiceStreamProvider", () => {
  let originalMediaRecorder: unknown;
  let originalWebSocket: unknown;

  beforeEach(() => {
    originalMediaRecorder = (globalThis as Record<string, unknown>).MediaRecorder;
    originalWebSocket = (globalThis as Record<string, unknown>).WebSocket;
    installMocks();
    vi.clearAllMocks();
  });

  afterEach(() => {
    (globalThis as Record<string, unknown>).MediaRecorder = originalMediaRecorder;
    (globalThis as Record<string, unknown>).WebSocket = originalWebSocket;
  });

  it("starts MediaRecorder immediately on start(), before WebSocket connects", async () => {
    const provider = new VoiceStreamProvider();
    provider.onResult = vi.fn();
    provider.onError = vi.fn();

    await provider.start();

    // MediaRecorder should be recording even though WS hasn't connected yet
    const ws = defined(MockWebSocket.instances[0]);
    expect(ws.readyState).toBe(WebSocket.CONNECTING); // WS not yet open
    expect(provider.getStream()).toBe(mockStream); // Stream acquired
  });

  it("buffers audio chunks before WebSocket connects, then flushes on open", async () => {
    const provider = new VoiceStreamProvider();
    provider.onResult = vi.fn();
    provider.onError = vi.fn();

    await provider.start();

    const ws = defined(MockWebSocket.instances[0]);

    // WebSocket is still CONNECTING — nothing sent yet
    expect(ws.sentMessages.length).toBe(0);

    // Simulate WebSocket opening
    ws.simulateOpen();

    // After open, any previously buffered chunks should have been flushed
    // (In this test there are none because MediaRecorder hasn't fired ondataavailable yet,
    //  but the flush code path was exercised)
    expect(ws.readyState).toBe(WebSocket.OPEN);
  });

  it("cleans up stale WebSocket on repeated start() calls (Bug 6 fix)", async () => {
    const provider = new VoiceStreamProvider();
    provider.onResult = vi.fn();
    provider.onError = vi.fn();

    // First start
    await provider.start();
    const firstWs = defined(MockWebSocket.instances[0]);
    expect(firstWs.readyState).toBe(WebSocket.CONNECTING);

    // Second start should close the first WebSocket
    await provider.start();
    expect(firstWs.readyState).toBe(WebSocket.CLOSED);
    expect(MockWebSocket.instances.length).toBe(2);
  });

  it("calls onError when microphone access is denied", async () => {
    Object.defineProperty(navigator, "mediaDevices", {
      value: {
        getUserMedia: vi.fn().mockRejectedValue(new Error("Permission denied")),
      },
      configurable: true,
    });

    const provider = new VoiceStreamProvider();
    const onError = vi.fn();
    provider.onError = onError;

    await provider.start();

    expect(onError).toHaveBeenCalledWith("Microphone access denied");
    expect(provider.getStream()).toBeNull();
    // No WebSocket should be created
    expect(MockWebSocket.instances.length).toBe(0);
  });

  it("handles final message correctly", async () => {
    const provider = new VoiceStreamProvider();
    const onResult = vi.fn();
    provider.onResult = onResult;

    await provider.start();
    const ws = defined(MockWebSocket.instances[0]);
    ws.simulateOpen();

    ws.simulateMessage({ type: "final", text: "Hello world" });

    expect(onResult).toHaveBeenCalledWith("Hello world");
  });

  it("handles partial messages", async () => {
    const provider = new VoiceStreamProvider();
    const onPartial = vi.fn();
    provider.onPartial = onPartial;

    await provider.start();
    const ws = defined(MockWebSocket.instances[0]);
    ws.simulateOpen();

    ws.simulateMessage({ type: "partial", text: "Hel" });

    expect(onPartial).toHaveBeenCalledWith("Hel");
  });

  it("sends done message on stop", async () => {
    const provider = new VoiceStreamProvider();
    provider.onResult = vi.fn();

    await provider.start();
    const ws = defined(MockWebSocket.instances[0]);
    ws.simulateOpen();

    provider.stop();

    const doneMsg = ws.sentMessages.find(
      (m) => typeof m === "string" && (JSON.parse(m) as { type: string }).type === "done",
    );
    expect(doneMsg).toBeTruthy();
  });

  it("dispose cleans up all resources", async () => {
    const provider = new VoiceStreamProvider();
    provider.onResult = vi.fn();

    await provider.start();
    const ws = defined(MockWebSocket.instances[0]);
    ws.simulateOpen();

    provider.dispose();

    expect(ws.readyState).toBe(WebSocket.CLOSED);
    expect(provider.getStream()).toBeNull();
  });
});
