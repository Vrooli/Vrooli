import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { forgetUnfinishedSession, loadUnfinishedSession } from "./sessionIdentity";
import { MemoryTurnJournalStore, TurnJournal, type JournalRecord } from "./turnJournal";
import { fallbackDelayMs, PcmVoiceStreamProvider } from "./pcmVoiceStreamProvider";
import {
  _resetMicOwnershipForTesting,
  acquireMicStream,
  getActiveMicLeases,
  installMicLifecycleCleanup,
  releaseAllMicLeases,
} from "./voice/micOwnership";

type FakeTrack = {
  readyState: "live" | "ended";
  muted: boolean;
  onended: (() => void) | null;
  onmute: (() => void) | null;
  onunmute: (() => void) | null;
  stop: ReturnType<typeof vi.fn>;
};

class FakeWebSocket {
  static OPEN = 1;
  static CLOSED = 3;
  static CONNECTING = 0;
  static instances: FakeWebSocket[] = [];
  static autoOpen = true;
  readyState = FakeWebSocket.CONNECTING;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  send = vi.fn();
  close = vi.fn(() => { this.readyState = FakeWebSocket.CLOSED; });

  constructor(readonly url: string) {
    FakeWebSocket.instances.push(this);
    if (FakeWebSocket.autoOpen) queueMicrotask(() => {
      this.readyState = FakeWebSocket.OPEN;
      this.onopen?.();
    });
  }
}

function makeStream(): { stream: MediaStream; track: FakeTrack } {
  const track: FakeTrack = {
    readyState: "live",
    muted: false,
    onended: null,
    onmute: null,
    onunmute: null,
    stop: vi.fn(),
  };
  return { stream: { getTracks: () => [track] } as unknown as MediaStream, track };
}

class FailOnceJournalStore extends MemoryTurnJournalStore {
  private failed = false;

  override async appendRecord(key: string, record: JournalRecord): Promise<void> {
    if (!this.failed) {
      this.failed = true;
      throw new Error("poisoned journal write chain");
    }
    await super.appendRecord(key, record);
  }
}

async function settle(): Promise<void> {
  for (let index = 0; index < 8; index += 1) await Promise.resolve();
}

describe("ported capture-path behavior", () => {
  let provider: PcmVoiceStreamProvider;
  let streams: ReturnType<typeof makeStream>[];
  let getUserMedia: ReturnType<typeof vi.fn>;
  let transcribeRetained: ReturnType<typeof vi.fn>;
  let captureOnFrame: ((samples: Float32Array, sampleRate: number) => void) | null;

  beforeEach(() => {
    vi.useFakeTimers();
    FakeWebSocket.instances = [];
    FakeWebSocket.autoOpen = true;
    streams = [makeStream(), makeStream(), makeStream()];
    captureOnFrame = null;
    getUserMedia = vi.fn(async () => streams.shift()!.stream);
    transcribeRetained = vi.fn(async () => "recovered");
    (globalThis as unknown as { WebSocket: typeof FakeWebSocket }).WebSocket = FakeWebSocket;
    provider = new PcmVoiceStreamProvider({
      getUserMedia,
      transport: {
        buildStreamUrl: () => "ws://voice.test/stream",
        transcribeRetained,
      },
      captureFactory: async (_stream, onFrame) => {
        captureOnFrame = onFrame;
        return { stop: vi.fn() };
      },
      journalFactory: () => new TurnJournal(new MemoryTurnJournalStore(), "ported", 0n, 16 * 1024 * 1024, "memory"),
    });
  });

  afterEach(() => {
    provider.dispose();
    forgetUnfinishedSession();
    vi.useRealTimers();
  });

  async function start(): Promise<FakeWebSocket> {
    await provider.start();
    await settle();
    return FakeWebSocket.instances.at(-1)!;
  }

  function pushFrame(samples = 128): void {
    captureOnFrame?.(new Float32Array(samples).fill(0.5), 16_000);
  }

  it("surfaces a poisoned journal write and keeps the promise chain usable", async () => {
    provider.dispose();
    const onError = vi.fn();
    const onDiagnostic = vi.fn();
    const store = new FailOnceJournalStore();
    provider = new PcmVoiceStreamProvider({
      getUserMedia,
      transport: { buildStreamUrl: () => "ws://voice.test/stream", transcribeRetained },
      captureFactory: async (_stream, onFrame) => {
        captureOnFrame = onFrame;
        return { stop: vi.fn() };
      },
      journalFactory: () => new TurnJournal(store, "poisoned", 0n, 16 * 1024 * 1024, "memory"),
    });
    provider.onError = onError;
    provider.onDiagnostic = onDiagnostic;
    await start();
    pushFrame(1_600);
    await settle();

    expect(onError).toHaveBeenCalledWith(expect.stringContaining("could not be saved safely"));
    expect(provider.getDiagnostic()).toMatchObject({ state: "failed", terminalReason: "capture_write_failed" });
    expect(provider.getDiagnostic().errorCodes).toContain("capture_write_failed");
    expect(onDiagnostic).toHaveBeenCalled();

    pushFrame(1_600);
    await settle();
    expect(onError).toHaveBeenCalledOnce();
  });

  it("scales fallback time with captured duration and write backlog", () => {
    expect(fallbackDelayMs(0n, 0)).toBe(10_000);
    expect(fallbackDelayMs(16_000n * 60n, 3)).toBe(60_300);
    expect(fallbackDelayMs(16_000n * 60n * 60n, 0)).toBe(3_600_000);
  });

  it("promotes only the uncommitted fallback tail after committed segments", async () => {
    const { uncommittedRemainder } = await import("./voice/trailingPartial");
    expect(uncommittedRemainder({ committedText: "committed words", latestPartial: "committed words final tail" })).toBe(" final tail");
    expect(uncommittedRemainder({ committedText: "committed words", latestPartial: "final tail" })).toBe(" final tail");
    expect(uncommittedRemainder({ committedText: "committed words", latestPartial: "committed words" })).toBeNull();
  });

  it("sends versioned PCM frames in capture order", async () => {
    const socket = await start();
    pushFrame(128);
    pushFrame(64);
    await vi.advanceTimersByTimeAsync(100);
    await vi.waitFor(() => {
      const sentFrames = socket.send.mock.calls.filter(([payload]) => payload instanceof ArrayBuffer);
      expect(sentFrames).toHaveLength(1);
    });

    const frames = socket.send.mock.calls.map(([payload]) => payload).filter((payload): payload is ArrayBuffer => payload instanceof ArrayBuffer);
    expect(frames).toHaveLength(1);
    expect(new TextDecoder().decode(new Uint8Array(frames[0]).slice(0, 4))).toBe("ATV2");
    expect(new DataView(frames[0]).getBigUint64(4, false)).toBe(0n);
  });

  it("flushes buffered PCM before done when the socket opens after stop", async () => {
    FakeWebSocket.autoOpen = false;
    await provider.start();
    const socket = FakeWebSocket.instances.at(-1)!;
    pushFrame();
    provider.stop();
    expect(socket.send).not.toHaveBeenCalled();

    socket.readyState = FakeWebSocket.OPEN;
    socket.onopen?.();
    await vi.waitFor(() => expect(socket.send).toHaveBeenCalledWith(JSON.stringify({ type: "done" })));
    const payloads = socket.send.mock.calls.map(([payload]) => payload);
    expect(payloads[0]).toBeInstanceOf(ArrayBuffer);
    expect(payloads.at(-1)).toBe(JSON.stringify({ type: "done" }));
  });

  it("replays journaled frames after reconnect without changing their identity", async () => {
    const first = await start();
    pushFrame();
    await vi.advanceTimersByTimeAsync(100);
    await settle();
    const original = first.send.mock.calls.find(([payload]) => payload instanceof ArrayBuffer)?.[0] as ArrayBuffer;

    first.readyState = FakeWebSocket.CLOSED;
    first.onclose?.();
    await vi.advanceTimersByTimeAsync(1_000);
    await settle();
    const second = FakeWebSocket.instances.at(-1)!;
    const replay = second.send.mock.calls.find(([payload]) => payload instanceof ArrayBuffer)?.[0] as ArrayBuffer;
    expect(new Uint8Array(replay)).toEqual(new Uint8Array(original));
  });

  it("compacts acknowledged journal coverage on reconnect", async () => {
    const first = await start();
    pushFrame(1_600);
    pushFrame(1_600);
    await settle();
    first.onmessage?.({ data: JSON.stringify({ type: "status", code: "processed_acknowledgement", text: "processed", processedSequence: 0 }) });
    await settle();

    first.readyState = FakeWebSocket.CLOSED;
    first.onclose?.();
    await vi.advanceTimersByTimeAsync(1_000);
    await settle();
    const second = FakeWebSocket.instances.at(-1)!;
    const replay = second.send.mock.calls.find(([payload]) => payload instanceof ArrayBuffer)?.[0] as ArrayBuffer;
    expect(new DataView(replay).getBigUint64(4, false)).toBe(1n);
    expect(second.send.mock.calls.filter(([payload]) => payload instanceof ArrayBuffer)).toHaveLength(1);
  });

  it("retains the journal and fails visibly when coverage is incomplete", async () => {
    const socket = await start();
    pushFrame();
    await settle();
    const onError = vi.fn();
    provider.onError = onError;
    socket.onmessage?.({ data: JSON.stringify({ type: "error", code: "incomplete_coverage", text: "coverage intentionally withheld" }) });
    socket.onmessage?.({ data: JSON.stringify({ type: "final", text: "must not commit" }) });
    await settle();

    expect(onError).toHaveBeenCalledOnce();
    expect(provider.getDiagnostic()).toMatchObject({ state: "failed", terminalReason: "incomplete_coverage" });
    expect(loadUnfinishedSession()).toMatchObject({ sessionId: expect.any(String), resumeToken: expect.any(String) });
  });

  it("forgets recovery state only after a committed terminal final", async () => {
    const socket = await start();
    pushFrame();
    await settle();
    provider.stop();
    socket.onmessage?.({ data: JSON.stringify({ type: "final", text: "committed" }) });
    await settle();

    expect(provider.getDiagnostic()).toMatchObject({ state: "completed", terminalReason: "final" });
    expect(loadUnfinishedSession()).toBeNull();
    socket.readyState = FakeWebSocket.CLOSED;
    socket.onclose?.();
    expect(transcribeRetained).not.toHaveBeenCalled();
  });

  it("accepts an empty terminal final after durable segment text", async () => {
    const socket = await start();
    pushFrame();
    await settle();
    const result = vi.fn();
    provider.onResult = result;
    provider.stop();
    socket.onmessage?.({ data: JSON.stringify({ type: "segment-final", text: "committed segment", segmentId: "segment-1", segmentIndex: 0 }) });
    socket.onmessage?.({ data: JSON.stringify({ type: "final", text: "" }) });
    await settle();

    expect(result).toHaveBeenCalledWith("");
    expect(provider.getDiagnostic()).toMatchObject({ state: "completed", terminalReason: "final" });
    expect(transcribeRetained).not.toHaveBeenCalled();
  });

  it("retains a WAV turn snapshot until the consumer disposes it", async () => {
    await start();
    pushFrame(256);
    provider.stop();
    const retained = provider.getLastTurnAudio();
    expect(retained).toMatchObject({ mimeType: "audio/wav" });
    expect(retained?.blob.size).toBeGreaterThan(44);
    provider.disposeLastTurn();
    expect(provider.getLastTurnAudio()).toBeNull();
  });

  it("stops the capture seam exactly once when a turn stops", async () => {
    const captureStop = vi.fn();
    provider.dispose();
    provider = new PcmVoiceStreamProvider({
      getUserMedia,
      transport: { buildStreamUrl: () => "ws://voice.test/stream", transcribeRetained },
      captureFactory: async (_stream, onFrame) => {
        captureOnFrame = onFrame;
        return { stop: captureStop };
      },
      journalFactory: () => new TurnJournal(new MemoryTurnJournalStore(), "ported-stop", 0n, 16 * 1024 * 1024, "memory"),
    });
    await start();
    provider.stop();
    provider.stop();
    expect(captureStop).toHaveBeenCalledOnce();
  });

  it("reports acknowledgement and finalization watchdog states", async () => {
    const status = vi.fn();
    provider.onStatus = status;
    const socket = await start();
    await vi.advanceTimersByTimeAsync(1_000);
    expect(status).toHaveBeenCalledWith(expect.objectContaining({ code: "server_ack_pending" }));
    pushFrame();
    provider.stop();
    await vi.advanceTimersByTimeAsync(3_000);
    expect(status).toHaveBeenCalledWith(expect.objectContaining({ code: "final_pending" }));
    socket.onmessage?.({ data: JSON.stringify({ type: "final", text: "done" }) });
  });
});

type LifecycleTrack = MediaStreamTrack & { emit(type: string): void };

function makeLifecycleStream(): { stream: MediaStream; track: LifecycleTrack } {
  const listeners = new Map<string, Set<() => void>>();
  const track = {
    readyState: "live",
    stop: vi.fn(),
    addEventListener: (type: string, listener: EventListenerOrEventListenerObject) => {
      const callback = typeof listener === "function" ? listener : () => listener.handleEvent(new Event(type));
      const set = listeners.get(type) ?? new Set<() => void>();
      set.add(callback as () => void);
      listeners.set(type, set);
    },
    removeEventListener: (type: string, listener: EventListenerOrEventListenerObject) => {
      listeners.get(type)?.delete(listener as () => void);
    },
    emit: (type: string) => { for (const listener of listeners.get(type) ?? []) listener(); },
  } as unknown as LifecycleTrack;
  return { stream: { getTracks: () => [track] } as unknown as MediaStream, track };
}

describe("ported microphone readiness behavior", () => {
  beforeEach(() => {
    _resetMicOwnershipForTesting();
  });

  afterEach(() => {
    _resetMicOwnershipForTesting();
  });

  it("acquires through getUserMedia, exposes metadata-only readiness, and releases idempotently", async () => {
    const { stream, track } = makeLifecycleStream();
    const getUserMedia = vi.fn(async () => stream);
    Object.defineProperty(navigator, "mediaDevices", { configurable: true, value: { getUserMedia } });
    const lease = await acquireMicStream("voice-stream", { audio: true });
    expect(getUserMedia).toHaveBeenCalledOnce();
    expect(getActiveMicLeases()).toMatchObject([{ owner: "voice-stream", trackCount: 1, liveTrackCount: 1 }]);
    lease.release("manual-stop");
    lease.release("manual-stop");
    expect(track.stop).toHaveBeenCalledOnce();
    expect(getActiveMicLeases()).toEqual([]);
  });

  it("releases a passive lease on hidden while preserving an active recording lease", async () => {
    const passive = makeLifecycleStream();
    const active = makeLifecycleStream();
    let index = 0;
    Object.defineProperty(navigator, "mediaDevices", {
      configurable: true,
      value: { getUserMedia: vi.fn(async () => (index++ === 0 ? passive.stream : active.stream)) },
    });
    await acquireMicStream("passive-wake-word", { audio: true });
    await acquireMicStream("voice-stream", { audio: true });
    const uninstall = installMicLifecycleCleanup();
    Object.defineProperty(document, "visibilityState", { configurable: true, value: "hidden" });
    document.dispatchEvent(new Event("visibilitychange"));
    uninstall();
    expect(passive.track.stop).toHaveBeenCalledOnce();
    expect(active.track.stop).not.toHaveBeenCalled();
  });

  it("releases every lease on pagehide and cleanup removes future lifecycle listeners", async () => {
    const first = makeLifecycleStream();
    Object.defineProperty(navigator, "mediaDevices", { configurable: true, value: { getUserMedia: vi.fn(async () => first.stream) } });
    await acquireMicStream("whisper", { audio: true });
    const uninstall = installMicLifecycleCleanup();
    window.dispatchEvent(new Event("pagehide"));
    expect(first.track.stop).toHaveBeenCalledOnce();
    uninstall();
    const second = makeLifecycleStream();
    const lease = await import("./voice/micOwnership").then(({ registerMicStream }) => registerMicStream("passive-wake-word", second.stream));
    window.dispatchEvent(new Event("pagehide"));
    expect(second.track.stop).not.toHaveBeenCalled();
    lease.release("test-reset");
  });
});
