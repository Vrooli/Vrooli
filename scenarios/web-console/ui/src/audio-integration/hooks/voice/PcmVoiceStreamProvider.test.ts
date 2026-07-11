import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const captureMocks = vi.hoisted(() => {
  const journals: Array<{ append: ReturnType<typeof vi.fn>; acknowledgeProcessed: ReturnType<typeof vi.fn>; replayAfter: ReturnType<typeof vi.fn>; discard: ReturnType<typeof vi.fn>; read: ReturnType<typeof vi.fn> }> = [];
  return {
    captureFrame: null as ((samples: Float32Array, rate: number) => void) | null,
    encodedFrames: vi.fn(),
    journals,
  };
});

vi.mock("../../api/voice", () => ({
  buildVoiceStreamWsUrl: (language?: string, sessionId?: string, resumeToken?: string) => `ws://voice.test/stream?language=${language ?? ""}&session_id=${sessionId ?? ""}&resume_token=${resumeToken ?? ""}&protocol_version=2`,
  transcribeAudioWithRetry: vi.fn(),
}));

vi.mock("./sharedAudioContext", () => ({ getSharedAudioContext: vi.fn(() => ({})) }));

vi.mock("@vrooli/audio-capture-browser", () => {
  class TurnJournal {
    readonly append = vi.fn(async () => {});
    readonly acknowledgeProcessed = vi.fn(async () => {});
    readonly replayAfter = vi.fn(() => []);
    readonly discard = vi.fn(async () => {});
    read = vi.fn(() => ({ chunks: [], nextSequence: 0n, nextSample: 0n }));
    constructor() { captureMocks.journals.push(this); }
    restore = vi.fn(async () => ({ chunks: [], nextSequence: 0n, nextSample: 0n }));
  }
  return {
    TurnJournal,
    IndexedDBTurnJournalStore: class {},
    MemoryTurnJournalStore: class {},
    TARGET_SAMPLE_RATE: 16_000,
    concatInt16: (parts: Int16Array[]) => parts[0] ?? new Int16Array(),
    createCanonicalPcmCapture: async (_context: unknown, _stream: unknown, onFrame: (samples: Float32Array, rate: number) => void) => {
      captureMocks.captureFrame = onFrame;
      return { stop: vi.fn() };
    },
    digestAudio: async () => new Uint8Array(32).buffer,
    encodeAudioFrame: (frame: unknown) => {
      captureMocks.encodedFrames(frame);
      return new ArrayBuffer(1);
    },
    encodeWavFromPcm16: () => new Blob(),
    frameToCanonicalPcm16: () => new Int16Array([1, 2, 3]),
    forgetUnfinishedSession: vi.fn(),
    loadUnfinishedSession: () => null,
    newSessionIdentity: (() => {
      let next = 0;
      return () => `identity-${++next}`;
    })(),
    rememberUnfinishedSession: vi.fn(),
  };
});

import { PcmVoiceStreamProvider } from "./PcmVoiceStreamProvider";

class FakeWebSocket {
  static OPEN = 1;
  static CLOSED = 3;
  static CONNECTING = 0;
  static instances: FakeWebSocket[] = [];
  readyState = FakeWebSocket.OPEN;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  send = vi.fn<(data: unknown) => void>();
  close = vi.fn(() => { this.readyState = FakeWebSocket.CLOSED; });

  constructor(readonly url: string) {
    FakeWebSocket.instances.push(this);
    queueMicrotask(() => this.onopen?.());
  }
}

function fakeStream(): MediaStream {
  return { getTracks: () => [{ readyState: "live", stop: vi.fn() }] } as unknown as MediaStream;
}

async function settle(): Promise<void> {
  await Promise.resolve();
  await Promise.resolve();
  await Promise.resolve();
}

describe("PcmVoiceStreamProvider", () => {
  let provider: PcmVoiceStreamProvider;

  beforeEach(() => {
    FakeWebSocket.instances = [];
    captureMocks.captureFrame = null;
    captureMocks.encodedFrames.mockClear();
    captureMocks.journals.splice(0);
    (globalThis as unknown as { WebSocket: typeof FakeWebSocket }).WebSocket = FakeWebSocket;
    Object.defineProperty(navigator, "mediaDevices", {
      configurable: true,
      value: { getUserMedia: vi.fn(async () => fakeStream()) },
    });
    provider = new PcmVoiceStreamProvider();
  });

  afterEach(() => provider.dispose());

  it("uses a durable v2 session and journals PCM before sending its first frame", async () => {
    await provider.start();
    await settle();
    const ws = FakeWebSocket.instances.at(-1);
    if (!ws || !captureMocks.captureFrame) throw new Error("expected an active PCM stream");

    expect(ws.url).toContain("protocol_version=2");
    expect(ws.url).toContain("session_id=identity-");
    captureMocks.captureFrame(new Float32Array([0.1, 0.2]), 48_000);
    await settle();

    expect(captureMocks.journals[0]?.append).toHaveBeenCalledOnce();
    expect(captureMocks.encodedFrames).toHaveBeenCalledWith(expect.objectContaining({ sequence: 0n, startSample: 0n, endSample: 3n }));
    expect(ws.send).toHaveBeenCalledWith(expect.any(ArrayBuffer));
  });

  it("acknowledges processed durable chunks and drops post-verdict PCM", async () => {
    await provider.start();
    await settle();
    const ws = FakeWebSocket.instances.at(-1);
    if (!ws || !captureMocks.captureFrame) throw new Error("expected an active PCM stream");

    captureMocks.captureFrame(new Float32Array([0.1]), 16_000);
    await settle();
    ws.onmessage?.({ data: JSON.stringify({ type: "status", code: "processed_acknowledgement", processedSequence: 0 }) });
    await settle();
    expect(captureMocks.journals[0]?.acknowledgeProcessed).toHaveBeenCalledWith(0n);

    const beforeDrop = captureMocks.encodedFrames.mock.calls.length;
    provider.dropTail();
    captureMocks.captureFrame(new Float32Array([0.2]), 16_000);
    await settle();
    expect(captureMocks.encodedFrames).toHaveBeenCalledTimes(beforeDrop);
  });
});
