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

vi.mock("@vrooli/audio-capture-browser", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@vrooli/audio-capture-browser")>();
  class TurnJournal {
    readonly append = vi.fn(async () => {});
    readonly acknowledgeProcessed = vi.fn(async () => {});
    readonly replayAfter = vi.fn(() => []);
    readonly discard = vi.fn(async () => {});
    read = vi.fn(() => ({ chunks: [], nextSequence: 0n, nextSample: 0n }));
    constructor() { captureMocks.journals.push(this); }
    restore = vi.fn(async () => ({ chunks: [], nextSequence: 0n, nextSample: 0n }));
  }
  class StreamDiagnosticRecorder {
    private snapshot = { sessionId: "", generation: 0, durability: "reduced", state: "preparing", capturedSequence: -1, sentSequence: -1, processedSequence: -1, statusCodes: [] as string[], errorCodes: [] as string[], events: [] as Array<{ kind: string; code: string }> };
    constructor(sessionId = "", generation = 0, durability = "reduced") { this.reset(sessionId, generation, durability); }
    reset(sessionId: string, generation: number, durability: string) { this.snapshot = { sessionId, generation, durability, state: "preparing", capturedSequence: -1, sentSequence: -1, processedSequence: -1, statusCodes: [], errorCodes: [], events: [] }; }
    state(state: string, code = state) { this.snapshot.state = state; this.snapshot.events.push({ kind: "state", code }); }
    captured(sequence: bigint) { this.snapshot.capturedSequence = Number(sequence); }
    sent(sequence: bigint) { this.snapshot.sentSequence = Number(sequence); }
    processed(sequence: bigint) { this.snapshot.processedSequence = Number(sequence); }
    status(code: string) { this.snapshot.statusCodes.push(code); }
    error(code: string) { this.snapshot.errorCodes.push(code); }
    terminal(state: string, code: string) { this.snapshot.state = state; this.snapshot.events.push({ kind: "terminal", code }); }
    read() { return { ...this.snapshot, statusCodes: [...this.snapshot.statusCodes], errorCodes: [...this.snapshot.errorCodes], events: [...this.snapshot.events] }; }
    exportJSON() { return JSON.stringify(this.read()); }
  }
  return {
    ...actual,
    TurnJournal,
    StreamDiagnosticRecorder,
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
    dispatchStreamMessage: (raw: string, handlers: Record<string, (...args: unknown[]) => void>, delivered: Set<string>) => {
      const message = JSON.parse(raw) as { type: string; code?: string; text?: string; processedSequence?: number; segmentId?: string; segmentIndex?: number };
      if (message.type === "status") handlers.onStatus?.(message.code ?? "stream_status", message.text ?? "", message.processedSequence === undefined ? undefined : BigInt(message.processedSequence));
      else if (message.type === "final") handlers.onFinal?.(message.text?.trim() ?? "");
      else if (message.type === "error") handlers.onError?.(message.code ?? "provider_failure", message.text ?? "Streaming provider failed.");
      else if (message.type === "segment-final" && message.text !== undefined && (!message.segmentId || !delivered.has(message.segmentId))) {
        if (message.segmentId) delivered.add(message.segmentId);
        handlers.onSegmentFinal?.(message.text, message.segmentIndex ?? 0);
      }
    },
  };
});

import { buildVoiceStreamWsUrl } from "../../api/voice";
import { TurnJournal } from "@vrooli/audio-capture-browser";
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
    provider = new PcmVoiceStreamProvider({
      transport: {
        buildStreamUrl: (language, sessionId, resumeToken) => buildVoiceStreamWsUrl(language, sessionId, resumeToken),
        transcribeRetained: async () => "",
      },
      captureFactory: async (_stream, onFrame) => {
        captureMocks.captureFrame = onFrame;
        return { stop: vi.fn() };
      },
      journalFactory: () => new (TurnJournal as unknown as { new (): TurnJournal })(),
    });
  });

  afterEach(() => provider.dispose());

  it("uses a durable v2 session and journals PCM before sending its first frame", async () => {
    await provider.start();
    await settle();
    const ws = FakeWebSocket.instances.at(-1);
    if (!ws || !captureMocks.captureFrame) throw new Error("expected an active PCM stream");

    expect(ws.url).toContain("protocol_version=2");
    expect(ws.url).toMatch(/session_id=[^&]+/);
    captureMocks.captureFrame(new Float32Array([0.1, 0.2]), 48_000);
    // The shared provider batches short PCM writes to bound journal and wire
    // overhead. A sub-batch flushes on its 100 ms timer.
    await new Promise((resolve) => setTimeout(resolve, 120));
    await settle();

    expect(captureMocks.journals[0]?.append).toHaveBeenCalledOnce();
    expect(ws.send).toHaveBeenCalledWith(expect.any(ArrayBuffer));
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

  it("[REQ:ATD-P1-002] exposes a metadata-only terminal diagnostic with coverage", async () => {
    await provider.start();
    await settle();
    const ws = FakeWebSocket.instances.at(-1);
    if (!ws || !captureMocks.captureFrame) throw new Error("expected an active PCM stream");
    captureMocks.captureFrame(new Float32Array([0.1]), 16_000);
    await settle();
    ws.onmessage?.({ data: JSON.stringify({ type: "status", code: "processed_acknowledgement", processedSequence: 0 }) });
    provider.stop();
    ws.onmessage?.({ data: JSON.stringify({ type: "final", text: "private transcript" }) });

    expect(provider.getDiagnostic()).toMatchObject({ capturedSequence: 0, processedSequence: 0, state: "completed" });
    expect(provider.exportDiagnostic()).not.toContain("private transcript");
  });

  it("[REQ:ATD-P0-001] delivers a replayed durable segment identity once", async () => {
    const onSegmentFinal = vi.fn();
    provider.onSegmentFinal = onSegmentFinal;
    await provider.start();
    await settle();
    const ws = FakeWebSocket.instances.at(-1);
    if (!ws) throw new Error("expected an active PCM stream");

    const durableSegment = { type: "segment-final", text: "replayed once", segmentIndex: 0, segmentId: "turn-1:0:0:1" };
    ws.onmessage?.({ data: JSON.stringify(durableSegment) });
    ws.onmessage?.({ data: JSON.stringify(durableSegment) });

    expect(onSegmentFinal).toHaveBeenCalledTimes(1);
    expect(onSegmentFinal).toHaveBeenCalledWith("replayed once", 0);
  });

  it("[REQ:ATD-P0-004] retains the journal when a final follows incomplete coverage", async () => {
    const onResult = vi.fn();
    const onError = vi.fn();
    provider.onResult = onResult;
    provider.onError = onError;
    await provider.start();
    await settle();
    const ws = FakeWebSocket.instances.at(-1);
    if (!ws || !captureMocks.captureFrame) throw new Error("expected an active PCM stream");
    captureMocks.captureFrame(new Float32Array([0.1]), 16_000);
    await settle();

    ws.onmessage?.({ data: JSON.stringify({ type: "error", code: "incomplete_coverage", text: "coverage incomplete" }) });
    ws.onmessage?.({ data: JSON.stringify({ type: "final", text: "must not be delivered" }) });
    await settle();

    expect(onError).toHaveBeenCalledWith("coverage incomplete");
    expect(onResult).not.toHaveBeenCalled();
    expect(captureMocks.journals[0]?.discard).not.toHaveBeenCalled();
    expect(provider.getDiagnostic()).toMatchObject({ state: "failed" });
  });
});
