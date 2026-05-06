// Tests for VoiceStreamProvider full-turn retention.

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { VoiceStreamProvider } from "../VoiceStreamProvider";

function defined<T>(val: T | null | undefined): T {
  if (val == null) throw new Error("Expected defined value");
  return val;
}

class MockMediaRecorder {
  state = "inactive" as "inactive" | "recording" | "paused";
  ondataavailable: ((e: { data: Blob }) => void) | null = null;
  onstop: (() => void) | null = null;

  constructor(_stream: MediaStream, _options?: MediaRecorderOptions) {
    (globalThis as { __lastRecorder?: MockMediaRecorder }).__lastRecorder = this;
  }

  start(_timeslice?: number) { this.state = "recording"; }
  stop() { this.state = "inactive"; this.onstop?.(); }

  /** Emit a chunk of the given byte length. */
  simulateChunk(bytes: number) {
    const data = new Uint8Array(bytes);
    const blob = new Blob([data]);
    blob.arrayBuffer = () => Promise.resolve(data.buffer);
    this.ondataavailable?.({ data: blob });
  }

  static isTypeSupported() { return true; }
}

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  readyState: number = WebSocket.CONNECTING;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  sentMessages: (string | ArrayBuffer)[] = [];

  constructor(_url: string) { MockWebSocket.instances.push(this); }
  send(data: string | ArrayBuffer) { this.sentMessages.push(data); }
  close() { this.readyState = WebSocket.CLOSED; }
  simulateOpen() { this.readyState = WebSocket.OPEN; this.onopen?.(); }
}

const mockStream = {
  getTracks: () => [{ stop: vi.fn(), readyState: "live" }],
} as unknown as MediaStream;

function installMocks() {
  Object.defineProperty(navigator, "mediaDevices", {
    value: { getUserMedia: vi.fn().mockResolvedValue(mockStream) },
    configurable: true,
  });
  (globalThis as Record<string, unknown>).MediaRecorder = MockMediaRecorder;
  (globalThis as Record<string, unknown>).WebSocket = MockWebSocket;
  MockWebSocket.instances = [];
}

describe("VoiceStreamProvider retention", () => {
  let origRecorder: unknown;
  let origWs: unknown;

  beforeEach(() => {
    origRecorder = (globalThis as Record<string, unknown>).MediaRecorder;
    origWs = (globalThis as Record<string, unknown>).WebSocket;
    (globalThis as { __lastRecorder?: MockMediaRecorder }).__lastRecorder = undefined;
    installMocks();
    vi.clearAllMocks();
  });

  afterEach(() => {
    (globalThis as Record<string, unknown>).MediaRecorder = origRecorder;
    (globalThis as Record<string, unknown>).WebSocket = origWs;
  });

  it("returns null before any recording starts", () => {
    const provider = new VoiceStreamProvider();
    expect(provider.getLastTurnAudio()).toBeNull();
  });

  it("concatenates every segment's audio (accepted + rejected) into one blob", async () => {
    const provider = new VoiceStreamProvider();
    provider.onResult = vi.fn();
    provider.onError = vi.fn();

    await provider.start();
    const ws = defined(MockWebSocket.instances[0]);
    ws.simulateOpen();

    // Simulate three segments of different sizes — one gets rejected,
    // two accepted. Retention must include all three.
    // Grab the underlying mediaRecorder via the recorded reference.
    // We rely on the provider having created a MockMediaRecorder and attached
    // its ondataavailable. Use the prototype hack: find the active mock via
    // its effects (we re-expose it through the only extant instance).
    // Simpler: provider created the MediaRecorder inside start(); capture
    // it by spying on the class constructor call order.
    // Since MockMediaRecorder doesn't expose a static registry, we drive
    // input through a fresh direct reference by calling into the
    // MediaRecorder constructor hook via the provider's getStream path.
    // Instead, we capture by re-importing: the provider's internal
    // recorder is the most-recently-constructed MockMediaRecorder.
    // To make this testable, we stash the last instance on globalThis.
    const recorder = (globalThis as { __lastRecorder?: MockMediaRecorder }).__lastRecorder;
    if (!recorder) throw new Error("no recorder captured — test mock didn't record instance");

    recorder.simulateChunk(100);
    recorder.simulateChunk(80); // imagine this one got rejected by the server
    recorder.simulateChunk(200);

    provider.stop();

    const retained = defined(provider.getLastTurnAudio());
    expect(retained.blob.size).toBe(380);
    expect(retained.mimeType).toMatch(/audio\/webm/);
    expect(retained.durationMs).toBeGreaterThanOrEqual(0);
  });

  it("replaces retained audio on a new start()", async () => {
    const provider = new VoiceStreamProvider();
    provider.onResult = vi.fn();
    provider.onError = vi.fn();

    await provider.start();
    defined(MockWebSocket.instances[0]).simulateOpen();
    const recorder1 = defined((globalThis as { __lastRecorder?: MockMediaRecorder }).__lastRecorder);
    recorder1.simulateChunk(500);
    provider.stop();

    const retained1 = defined(provider.getLastTurnAudio());
    expect(retained1.blob.size).toBe(500);

    // Second turn.
    await provider.start();
    // New turn resets retention immediately.
    expect(provider.getLastTurnAudio()).toBeNull();
    defined(MockWebSocket.instances[1]).simulateOpen();
    const recorder2 = defined((globalThis as { __lastRecorder?: MockMediaRecorder }).__lastRecorder);
    recorder2.simulateChunk(70);
    provider.stop();

    expect(defined(provider.getLastTurnAudio()).blob.size).toBe(70);
  });

  it("clears retained audio on disposeLastTurn()", async () => {
    const provider = new VoiceStreamProvider();
    provider.onResult = vi.fn();
    provider.onError = vi.fn();

    await provider.start();
    defined(MockWebSocket.instances[0]).simulateOpen();
    const recorder = defined((globalThis as { __lastRecorder?: MockMediaRecorder }).__lastRecorder);
    recorder.simulateChunk(10);
    provider.stop();

    expect(provider.getLastTurnAudio()).not.toBeNull();
    provider.disposeLastTurn();
    expect(provider.getLastTurnAudio()).toBeNull();
  });

  it("does not retain audio when no chunks arrived", async () => {
    const provider = new VoiceStreamProvider();
    provider.onResult = vi.fn();
    provider.onError = vi.fn();

    await provider.start();
    defined(MockWebSocket.instances[0]).simulateOpen();
    // No chunks.
    provider.stop();

    expect(provider.getLastTurnAudio()).toBeNull();
  });

  it("dispose() drops retained audio", async () => {
    const provider = new VoiceStreamProvider();
    provider.onResult = vi.fn();
    provider.onError = vi.fn();

    await provider.start();
    defined(MockWebSocket.instances[0]).simulateOpen();
    const recorder = defined((globalThis as { __lastRecorder?: MockMediaRecorder }).__lastRecorder);
    recorder.simulateChunk(40);
    provider.stop();
    expect(provider.getLastTurnAudio()).not.toBeNull();

    provider.dispose();
    expect(provider.getLastTurnAudio()).toBeNull();
  });
});
