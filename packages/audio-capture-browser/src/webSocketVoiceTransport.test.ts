import { afterEach, describe, expect, it, vi } from "vitest";
import { openWebSocketVoiceStream, type VoiceWebSocketCodec } from "./webSocketVoiceTransport";
import type { VoiceAudioChunk, VoiceStreamHandlers, VoiceStreamStart } from "./transport";

type Request = { type: string; sequence?: bigint };
type Response = { type: "accepted" | "partial" | "done"; text?: string };

const start: VoiceStreamStart = {
  language: "en",
  sessionId: "ws-session",
  resumeToken: "resume-token",
  protocolVersion: 2,
  generation: 3n,
  inputFormat: "pcm_s16le",
  sampleRateHz: 16000,
};

class FakeWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;
  static instances: FakeWebSocket[] = [];

  readonly url: string;
  readyState = FakeWebSocket.CONNECTING;
  bufferedAmount = 0;
  binaryType: BinaryType = "blob";
  sent: Uint8Array[] = [];
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;

  constructor(url: string) {
    this.url = url;
    FakeWebSocket.instances.push(this);
  }

  send(data: ArrayBufferLike | ArrayBufferView): void {
    if (data instanceof ArrayBuffer) this.sent.push(new Uint8Array(data));
    else this.sent.push(new Uint8Array(data.buffer, data.byteOffset, data.byteLength));
  }

  open(): void {
    this.readyState = FakeWebSocket.OPEN;
    this.onopen?.({} as Event);
  }

  receive(value: Response): void {
    const bytes = new TextEncoder().encode(JSON.stringify(value));
    this.onmessage?.({ data: bytes.buffer } as MessageEvent);
  }

  close(code = 1000): void {
    this.readyState = FakeWebSocket.CLOSED;
    this.onclose?.({ code } as CloseEvent);
  }
}

const codec: VoiceWebSocketCodec<Request, Response> = {
  start: () => ({ type: "start" }),
  audio: (chunk: VoiceAudioChunk) => ({ type: "audio", sequence: chunk.sequence }),
  end: () => ({ type: "end" }),
  encode: (value) => new TextEncoder().encode(JSON.stringify(value, (_key, field) => typeof field === "bigint" ? field.toString() : field)),
  decode: (value) => JSON.parse(new TextDecoder().decode(value)) as Response,
  isReady: (value) => value.type === "accepted",
  event: (value) => value.type === "accepted" ? null : value.type === "done" ? { type: "final", text: value.text } : { type: value.type, text: value.text },
};

let originalWebSocket: typeof WebSocket;

afterEach(() => {
  if (originalWebSocket) Object.defineProperty(globalThis, "WebSocket", { configurable: true, value: originalWebSocket });
  FakeWebSocket.instances = [];
});

describe("openWebSocketVoiceStream", () => {
  it("sends binary start, gates audio on admission, and preserves end/event order", () => {
    originalWebSocket = globalThis.WebSocket;
    Object.defineProperty(globalThis, "WebSocket", { configurable: true, value: FakeWebSocket });
    const onReady = vi.fn();
    const onEvent = vi.fn();
    const onClose = vi.fn();
    const handlers: VoiceStreamHandlers = { onReady, onEvent, onClose };
    const connection = openWebSocketVoiceStream("ws://same-origin/ws/stt/transcribe-stream-v2", codec, start, handlers);
    const socket = FakeWebSocket.instances[0]!;

    expect(socket.binaryType).toBe("arraybuffer");
    expect(connection.isOpen()).toBe(false);
    socket.open();
    expect(JSON.parse(new TextDecoder().decode(socket.sent[0]!))).toEqual({ type: "start" });
    connection.sendAudio({ sequence: 1n, startSample: 0n, endSample: 2n, audio: new Uint8Array([1, 2]), sha256: new Uint8Array() });
    expect(socket.sent).toHaveLength(1);

    socket.receive({ type: "accepted" });
    expect(onReady).toHaveBeenCalledOnce();
    expect(connection.isOpen()).toBe(true);
    connection.sendAudio({ sequence: 1n, startSample: 0n, endSample: 2n, audio: new Uint8Array([1, 2]), sha256: new Uint8Array() });
    connection.end();
    expect(socket.sent.map((bytes) => JSON.parse(new TextDecoder().decode(bytes)).type)).toEqual(["start", "audio", "end"]);
    expect(connection.isOpen()).toBe(false);

    socket.receive({ type: "partial", text: "hello" });
    expect(onEvent).toHaveBeenCalledWith({ type: "partial", text: "hello" });
    socket.receive({ type: "done", text: "hello" });
    expect(onEvent).toHaveBeenCalledWith({ type: "final", text: "hello" });
    socket.close(1000);
    expect(onClose).toHaveBeenCalledOnce();
    expect(onClose).toHaveBeenCalledWith(undefined);
  });

  it("closes on bounded-buffer pressure so uncommitted audio can be recovered", () => {
    originalWebSocket = globalThis.WebSocket;
    Object.defineProperty(globalThis, "WebSocket", { configurable: true, value: FakeWebSocket });
    const onClose = vi.fn();
    const connection = openWebSocketVoiceStream("ws://same-origin/ws/stt/transcribe-stream-v2", codec, start, { onReady: vi.fn(), onEvent: vi.fn(), onClose });
    const socket = FakeWebSocket.instances[0]!;
    socket.open();
    socket.receive({ type: "accepted" });
    socket.bufferedAmount = 2 * 1024 * 1024;
    connection.sendAudio({ sequence: 1n, startSample: 0n, endSample: 2n, audio: new Uint8Array([1, 2]), sha256: new Uint8Array() });
    expect(onClose).toHaveBeenCalledOnce();
    expect(onClose.mock.calls[0]?.[0]).toBeInstanceOf(Error);
    expect(socket.sent).toHaveLength(1);
  });

  it("reports non-normal closes without surfacing transport details", () => {
    originalWebSocket = globalThis.WebSocket;
    Object.defineProperty(globalThis, "WebSocket", { configurable: true, value: FakeWebSocket });
    const onClose = vi.fn();
    openWebSocketVoiceStream("ws://same-origin/ws/stt/transcribe-stream-v2", codec, start, { onReady: vi.fn(), onEvent: vi.fn(), onClose });
    FakeWebSocket.instances[0]!.close(1006);
    expect(onClose).toHaveBeenCalledOnce();
    expect(onClose.mock.calls[0]?.[0]).toEqual(new Error("voice stream WebSocket closed before successful completion"));
  });

  it("treats normal close before a final event as incomplete", () => {
    originalWebSocket = globalThis.WebSocket;
    Object.defineProperty(globalThis, "WebSocket", { configurable: true, value: FakeWebSocket });
    const onClose = vi.fn();
    openWebSocketVoiceStream("ws://same-origin/ws/stt/transcribe-stream-v2", codec, start, { onReady: vi.fn(), onEvent: vi.fn(), onClose });
    FakeWebSocket.instances[0]!.close(1000);
    expect(onClose).toHaveBeenCalledOnce();
    expect(onClose.mock.calls[0]?.[0]).toEqual(new Error("voice stream WebSocket closed before successful completion"));
  });
});
