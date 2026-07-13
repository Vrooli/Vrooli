/**
 * Silent-loss contract for the streaming + HTTP providers.
 *
 * The bug these guard against: a long voice turn that ends with no text and no
 * error, leaving the UI wedged on "transcribing" or silently dropping the
 * message. The contract (see TranscriptionProvider.onResult):
 *
 *   1. An empty `final` resolves the turn via onResult("") (already true; pinned
 *      here so a future refactor can't re-introduce the silent drop).
 *   2. An empty HTTP-fallback / batch result resolves the turn via onResult("")
 *      instead of returning early and wedging the UI.
 *   3. When the streaming `final` never arrives, the timeout falls back to a
 *      one-shot HTTP transcription of the FULL retained turn — recovering a
 *      message the streaming path lost — and resolves the turn either way.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { VoiceStreamProvider } from "./VoiceStreamProvider";
import { WhisperProvider } from "./WhisperProvider";

const transcribeMock = vi.fn<(blob: Blob, max?: number, lang?: string) => Promise<string>>();

vi.mock("../../api/voice", () => ({
  transcribeAudioWithRetry: (blob: Blob, max?: number, lang?: string) => transcribeMock(blob, max, lang),
  buildVoiceStreamWsUrl: () => "ws://test/stream",
}));

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
    this.onclose?.();
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

function fakeBlob(size: number): Blob {
  return {
    size,
    type: "audio/webm",
    arrayBuffer: () => Promise.resolve(new ArrayBuffer(size)),
  } as unknown as Blob;
}

function fakeStream(): MediaStream {
  return {
    getTracks: () => [{ readyState: "live", stop: () => {}, enabled: true, muted: false }],
  } as unknown as MediaStream;
}

let infoSpy: ReturnType<typeof vi.spyOn>;
let warnSpy: ReturnType<typeof vi.spyOn>;

beforeEach(() => {
  FakeWebSocket.instances = [];
  FakeMediaRecorder.instances = [];
  transcribeMock.mockReset();
  (globalThis as unknown as { WebSocket: typeof FakeWebSocket }).WebSocket = FakeWebSocket;
  (globalThis as unknown as { MediaRecorder: typeof FakeMediaRecorder }).MediaRecorder = FakeMediaRecorder;
  Object.defineProperty(navigator, "mediaDevices", {
    configurable: true,
    value: { getUserMedia: vi.fn(async () => fakeStream()) },
  });
  infoSpy = vi.spyOn(console, "info").mockImplementation(() => {});
  warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
});

afterEach(() => {
  infoSpy.mockRestore();
  warnSpy.mockRestore();
  vi.useRealTimers();
});

describe("VoiceStreamProvider empty-result contract", () => {
  it("resolves the turn via onResult('') when the streaming final is empty", async () => {
    const provider = new VoiceStreamProvider();
    const onResult = vi.fn();
    const onError = vi.fn();
    provider.onResult = onResult;
    provider.onError = onError;

    await provider.start();
    await Promise.resolve();
    const ws = FakeWebSocket.instances.at(-1);
    if (!ws) throw new Error("expected a WebSocket");

    // A final that arrives while the recorder is still actively recording is
    // treated as backend loss (reconnect/fallback), so end the turn first —
    // the empty-final contract applies to finals after an intentional stop.
    provider.stop();
    ws.onmessage?.({ data: JSON.stringify({ type: "final", text: "   " }) });

    expect(onResult).toHaveBeenCalledTimes(1);
    expect(onResult).toHaveBeenCalledWith("");
    expect(onError).not.toHaveBeenCalled();
    provider.dispose();
  });

  it("falls back to HTTP transcription of the retained turn when the final times out (recovery)", async () => {
    vi.useFakeTimers();
    transcribeMock.mockResolvedValue("the message the stream lost");
    const provider = new VoiceStreamProvider();
    const onResult = vi.fn();
    const onError = vi.fn();
    const statuses: string[] = [];
    provider.onResult = onResult;
    provider.onError = onError;
    provider.onStatus = (status) => statuses.push(status.code);

    await provider.start();
    await vi.advanceTimersByTimeAsync(0); // flush the WS onopen microtask
    const rec = FakeMediaRecorder.instances.at(-1);
    if (!rec) throw new Error("expected a MediaRecorder");

    // Buffer a chunk so the retained turn (allChunks) is non-empty.
    rec.ondataavailable?.({ data: fakeBlob(2048) });
    await vi.advanceTimersByTimeAsync(0);

    provider.stop(); // arms the final-timeout; no `final` will arrive
    await vi.advanceTimersByTimeAsync(3001);
    expect(statuses).toContain("final_pending");
    // Advance past the max final timeout (cap is 60s).
    await vi.advanceTimersByTimeAsync(61_000);

    expect(transcribeMock).toHaveBeenCalledTimes(1);
    expect(onResult).toHaveBeenCalledWith("the message the stream lost");
    expect(statuses).toContain("transcription_complete");
    expect(onError).not.toHaveBeenCalled();
    provider.dispose();
  });

  it("clears the final-pending status when the websocket final eventually arrives", async () => {
    vi.useFakeTimers();
    const provider = new VoiceStreamProvider();
    const onResult = vi.fn();
    const statuses: string[] = [];
    provider.onResult = onResult;
    provider.onError = vi.fn();
    provider.onStatus = (status) => statuses.push(status.code);

    await provider.start();
    await vi.advanceTimersByTimeAsync(0);
    const rec = FakeMediaRecorder.instances.at(-1);
    const ws = FakeWebSocket.instances.at(-1);
    if (!rec || !ws) throw new Error("expected recorder and websocket");
    rec.ondataavailable?.({ data: fakeBlob(2048) });
    await vi.advanceTimersByTimeAsync(0);

    provider.stop();
    await vi.advanceTimersByTimeAsync(3001);
    expect(statuses).toContain("final_pending");

    ws.onmessage?.({ data: JSON.stringify({ type: "final", text: "eventual transcript" }) });

    expect(statuses).toContain("transcription_complete");
    expect(onResult).toHaveBeenCalledWith("eventual transcript");
    provider.dispose();
  });

  it("still resolves the turn via onResult('') when the timeout fallback is also empty", async () => {
    vi.useFakeTimers();
    transcribeMock.mockResolvedValue("");
    const provider = new VoiceStreamProvider();
    const onResult = vi.fn();
    provider.onResult = onResult;
    provider.onError = vi.fn();

    await provider.start();
    await vi.advanceTimersByTimeAsync(0);
    const rec = FakeMediaRecorder.instances.at(-1);
    if (!rec) throw new Error("expected a MediaRecorder");
    rec.ondataavailable?.({ data: fakeBlob(2048) });
    await vi.advanceTimersByTimeAsync(0);

    provider.stop();
    await vi.advanceTimersByTimeAsync(61_000);

    expect(transcribeMock).toHaveBeenCalledTimes(1);
    expect(onResult).toHaveBeenCalledWith("");
    provider.dispose();
  });
});

describe("WhisperProvider empty-result contract", () => {
  it("resolves the turn via onResult('') when batch transcription is empty", async () => {
    transcribeMock.mockResolvedValue("   ");
    const provider = new WhisperProvider();
    const onResult = vi.fn();
    const onError = vi.fn();
    provider.onResult = onResult;
    provider.onError = onError;

    await provider.start();
    const rec = FakeMediaRecorder.instances.at(-1);
    if (!rec) throw new Error("expected a MediaRecorder");
    rec.ondataavailable?.({ data: fakeBlob(4096) });

    provider.stop(); // fake recorder fires onstop synchronously
    // onstop is async (awaits transcribe); flush microtasks.
    await Promise.resolve();
    await Promise.resolve();

    expect(transcribeMock).toHaveBeenCalledTimes(1);
    expect(onResult).toHaveBeenCalledTimes(1);
    expect(onResult).toHaveBeenCalledWith("");
    expect(onError).not.toHaveBeenCalled();
    provider.dispose();
  });
});
