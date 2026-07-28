import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const core = vi.hoisted(() => ({
  frame: null as ((samples: Float32Array, rate: number) => void) | null,
  append: vi.fn(async () => {}),
  acknowledge: vi.fn(async () => {}),
  discard: vi.fn(async () => {}),
}));

vi.mock("../../api/voice", () => ({
  buildVoiceStreamWsUrl: (_language?: string, sessionId?: string) => `ws://voice.test/stream?protocol_version=2&session_id=${sessionId}`,
  transcribeAudioWithRetry: vi.fn(),
}));
vi.mock("./sharedAudioContext", () => ({ getSharedAudioContext: vi.fn(() => ({})) }));
vi.mock("@vrooli/audio-capture-browser", () => ({
  TARGET_SAMPLE_RATE: 16_000,
  concatInt16: (parts: Int16Array[]) => parts[0] ?? new Int16Array(),
  createCanonicalPcmCapture: async (_context: unknown, _stream: unknown, onFrame: (samples: Float32Array, rate: number) => void) => {
    core.frame = onFrame;
    return { stop: vi.fn() };
  },
  digestAudio: async () => new ArrayBuffer(32),
  encodeAudioFrame: () => new ArrayBuffer(1),
  encodeWavFromPcm16: () => new Blob(),
  forgetUnfinishedSession: vi.fn(),
  frameToCanonicalPcm16: () => new Int16Array([1, 2]),
  IndexedDBTurnJournalStore: class IndexedDBTurnJournalStore { readonly kind = "indexeddb"; },
  loadUnfinishedSession: () => null,
  MemoryTurnJournalStore: class MemoryTurnJournalStore { readonly kind = "memory"; },
  newSessionIdentity: (() => { let next = 0; return () => `session-${++next}`; })(),
  rememberUnfinishedSession: vi.fn(),
  dispatchStreamMessage: (raw: string, handlers: {
    onStatus?: (code: string, text: string, processedSequence?: bigint) => void;
    onFinal?: (text: string) => void;
    onError?: (code: string, text: string) => void;
  }) => {
    const message = JSON.parse(raw) as { type: string; code?: string; text?: string; processedSequence?: number };
    if (message.type === "status") handlers.onStatus?.(message.code ?? "stream_status", message.text ?? "Streaming transcription status updated.", message.processedSequence === undefined ? undefined : BigInt(message.processedSequence));
    else if (message.type === "final") handlers.onFinal?.(message.text?.trim() ?? "");
    else if (message.type === "error") handlers.onError?.(message.code ?? "provider_failure", message.text ?? "Streaming provider failed.");
  },
  TurnJournal: class TurnJournal {
    restore = async () => ({ nextSequence: 0n, nextSample: 0n });
    append = core.append;
    acknowledgeProcessed = core.acknowledge;
    replayAfter = () => [];
    discard = core.discard;
  },
}));

import { PcmVoiceStreamProvider } from "./PcmVoiceStreamProvider";

class FakeWebSocket {
  static OPEN = 1;
  static instances: FakeWebSocket[] = [];
  readyState = FakeWebSocket.OPEN;
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  send = vi.fn();
  close = vi.fn();

  constructor(readonly url: string) {
    FakeWebSocket.instances.push(this);
    queueMicrotask(() => this.onopen?.());
  }
}

function stream(): MediaStream {
  return { getTracks: () => [{ readyState: "live", stop: vi.fn() }] } as unknown as MediaStream;
}

describe("PcmVoiceStreamProvider", () => {
  let provider: PcmVoiceStreamProvider;

  beforeEach(() => {
    FakeWebSocket.instances = [];
    core.frame = null;
    core.append.mockClear();
    core.acknowledge.mockClear();
    core.discard.mockClear();
    (globalThis as unknown as { WebSocket: typeof FakeWebSocket }).WebSocket = FakeWebSocket;
    provider = new PcmVoiceStreamProvider();
  });

  afterEach(() => provider.dispose());

  it("[REQ:ATD-P0-006] journals canonical audio before sending a v2 frame through a pre-warmed stream", async () => {
    await provider.start(stream());
    await Promise.resolve();
    await Promise.resolve();
    const ws = FakeWebSocket.instances[0];
    expect(ws?.url).toContain("protocol_version=2");

    core.frame?.(new Float32Array([0.1]), 48_000);
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(core.append).toHaveBeenCalledOnce();
    expect(ws?.send).toHaveBeenCalledWith(expect.any(ArrayBuffer));
  });

  it("compacts only after a processed acknowledgement", async () => {
    await provider.start(stream());
    await Promise.resolve();
    const ws = FakeWebSocket.instances[0];
    ws?.onmessage?.({ data: JSON.stringify({ type: "status", code: "processed_acknowledgement", processedSequence: 0 }) });
    await Promise.resolve();

    expect(core.acknowledge).toHaveBeenCalledWith(0n);
  });

  it("[REQ:ATD-P0-004] preserves recovery audio when final follows an incomplete-coverage error", async () => {
    const onResult = vi.fn();
    const onError = vi.fn();
    provider.onResult = onResult;
    provider.onError = onError;
    await provider.start(stream());
    await Promise.resolve();
    const ws = FakeWebSocket.instances[0];
    core.frame?.(new Float32Array([0.1]), 16_000);
    await Promise.resolve();

    ws?.onmessage?.({ data: JSON.stringify({ type: "error", code: "incomplete_coverage", text: "coverage incomplete" }) });
    ws?.onmessage?.({ data: JSON.stringify({ type: "final", text: "must not be delivered" }) });
    await Promise.resolve();

    expect(onError).toHaveBeenCalledWith("coverage incomplete");
    expect(onResult).not.toHaveBeenCalled();
    expect(core.discard).not.toHaveBeenCalled();
  });
});
