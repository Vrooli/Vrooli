import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FakeVoiceTransport as FakeWebSocket, stubCryptoWithoutSubtle } from "./fakeVoiceTransport";
import { MemoryTurnJournalStore, TurnJournal } from "./turnJournal";
import { MAX_RETAINED_TURN_AUDIO_BYTES, PCM_WIRE_BATCH_SAMPLES, PcmVoiceStreamProvider } from "./pcmVoiceStreamProvider";
import { readStreamDiagnosticTelemetry } from "./streamDiagnostic";
import { forgetUnfinishedSession, rememberUnfinishedSession } from "./sessionIdentity";
import { getActiveMicLeases } from "./voice/micOwnership";

type FakeTrack = {
  readyState: "live" | "ended";
  muted: boolean;
  onended: (() => void) | null;
  onmute: (() => void) | null;
  onunmute: (() => void) | null;
  stop: ReturnType<typeof vi.fn>;
};

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

async function settle(): Promise<void> {
  for (let index = 0; index < 6; index += 1) await Promise.resolve();
}

async function settleLong(): Promise<void> {
  for (let index = 0; index < 128; index += 1) await Promise.resolve();
}

function identifyLocalRoute(socket: { onmessage: ((event: { data: string }) => void) | null }): void {
  socket.onmessage?.({ data: JSON.stringify({
    type: "status",
    code: "provider_identity",
    providerId: "whisper-local",
    modelId: "whisper-medium",
    providerTier: "local",
  }) });
}

const BASELINE_FRAME_SECONDS = 60;
const BASELINE_SIMULATED_SECONDS = 60 * 60;
const BASELINE_FRAME_SAMPLES = 16_000 * BASELINE_FRAME_SECONDS;

// This file used to publish conformance evidence into
// scenarios/audio-tools/coverage/phase-1-accelerated-baseline.json behind an
// environment variable, shallow-merging its keys over whatever two other
// processes had written there. The merged artifact read as one end-to-end
// proof; it was three partial observations with no shared run identity, and
// several of its fields were literals nothing computed.
//
// These are good unit tests and they stay. They no longer publish evidence:
// conformance evidence comes from a run that observed the whole chain in one
// process and reports `not_measured` for anything it could not see. See
// scenarios/audio-tools/api/internal/conformance.

describe("shared PCM long-session recovery harness", () => {
  let provider: PcmVoiceStreamProvider;
  let streams: ReturnType<typeof makeStream>[];
  let getUserMedia: ReturnType<typeof vi.fn>;
  let status: ReturnType<typeof vi.fn>;
  let result: ReturnType<typeof vi.fn>;
  let transcribeRetained: ReturnType<typeof vi.fn>;
  let journal: TurnJournal;
  let captureOnFrame: ((samples: Float32Array, sampleRate: number) => void) | null;

  beforeEach(() => {
    vi.useFakeTimers();
    stubCryptoWithoutSubtle();
    FakeWebSocket.instances = [];
    FakeWebSocket.autoOpen = true;
    streams = [makeStream(), makeStream(), makeStream()];
    captureOnFrame = null;
    getUserMedia = vi.fn(async () => streams.shift()!.stream);
    status = vi.fn();
    result = vi.fn();
    transcribeRetained = vi.fn(async () => "recovered");
    journal = new TurnJournal(new MemoryTurnJournalStore(), "test", 0n, 16 * 1024 * 1024, "memory");

    provider = new PcmVoiceStreamProvider({
      getUserMedia,
      transport: new FakeWebSocket(transcribeRetained),
      captureFactory: async (_stream, onFrame) => {
        captureOnFrame = onFrame;
        return { stop: vi.fn() };
      },
      journalFactory: () => journal,
    });
    provider.onStatus = status;
    provider.onResult = result;
  });

  afterEach(() => {
    provider.dispose();
    forgetUnfinishedSession();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  it("publishes recording state once capture is ready", async () => {
    provider.markStartRequested(1);
    await provider.start();
    await settle();

    expect(provider.getDiagnostic().state).toBe("recording");
    expect(provider.getDiagnostic()).toMatchObject({ startRequestedAtMs: 1 });
    expect(provider.getDiagnostic().captureReadyAtMs).not.toBeNull();
    expect(provider.getDiagnostic().startToCaptureReadyMs).not.toBeNull();
    expect(readStreamDiagnosticTelemetry()?.latest.state).toBe("recording");
  });

  it("reconnects when admission fails before an asynchronous capture graph is installed", async () => {
    provider.dispose();
    let openCalls = 0;
    let finishCapture!: (capture: { stop: ReturnType<typeof vi.fn> }) => void;
    const captureStop = vi.fn();
    provider = new PcmVoiceStreamProvider({
      getUserMedia,
      transport: {
        retainedRoutePolicy: "exact-model",
        transcribeRetained: async () => "recovered",
        openStream: (_start, handlers) => {
          openCalls += 1;
          let open = false;
          const connection = {
            isOpen: () => open,
            sendAudio: vi.fn(),
            end: vi.fn(),
            close: vi.fn(() => { open = false; }),
          };
          queueMicrotask(() => {
            if (openCalls === 1) {
              handlers.onClose(new Error("admission failed before capture"));
            } else {
              open = true;
              handlers.onReady();
            }
          });
          return connection;
        },
      },
      captureFactory: () => new Promise((resolve) => {
        finishCapture = resolve;
      }),
      journalFactory: () => journal,
    });

    const starting = provider.start();
    await settleLong();
    // Admission retries as soon as storage recovery is ready, even while the
    // asynchronous capture graph is still being installed.
    expect(openCalls).toBe(2);
    expect(finishCapture).toBeTypeOf("function");

    finishCapture({ stop: captureStop });
    await starting;
    await settleLong();

    expect(openCalls).toBe(2);
    expect(provider.isStreamConnected()).toBe(true);
    expect(provider.getDiagnostic().statusCodes).toContain("stream_admission_failed");
    provider.stop();
    expect(captureStop).toHaveBeenCalledOnce();
  });

  it("stops a capture graph that resolves after the turn was stopped", async () => {
    let finishCapture!: (capture: { stop: ReturnType<typeof vi.fn> }) => void;
    const lateStop = vi.fn();
    provider.captureFactory = () => new Promise((resolve) => {
      finishCapture = resolve;
    });

    const starting = provider.start();
    await settleLong();
    expect(finishCapture).toBeTypeOf("function");

    provider.stop();
    finishCapture({ stop: lateStop });
    await starting;

    expect(lateStop).toHaveBeenCalledOnce();
  });

  it("releases a microphone permission result that resolves after start was stopped", async () => {
    let finishPermission!: (stream: MediaStream) => void;
    getUserMedia.mockImplementationOnce(() => new Promise((resolve) => {
      finishPermission = resolve;
    }));

    const starting = provider.start();
    await settleLong();
    expect(finishPermission).toBeTypeOf("function");

    provider.stop();
    const late = makeStream();
    finishPermission(late.stream);
    await starting;

    expect(late.track.stop).toHaveBeenCalledOnce();
    expect(provider.getStream()).toBeNull();
    expect(FakeWebSocket.instances).toHaveLength(0);
  });

  it("stops a reacquired capture graph that resolves after the turn was stopped", async () => {
    await provider.start();
    await settleLong();
    let finishCapture!: (capture: { stop: ReturnType<typeof vi.fn> }) => void;
    const lateStop = vi.fn();
    provider.captureFactory = () => new Promise((resolve) => {
      finishCapture = resolve;
    });

    (provider.getStream()!.getTracks()[0] as unknown as FakeTrack).onended?.();
    await settleLong();
    expect(finishCapture).toBeTypeOf("function");

    provider.stop();
    finishCapture({ stop: lateStop });
    await settleLong();

    expect(lateStop).toHaveBeenCalledOnce();
  });

  it("does not let a stale microphone reacquisition replace a newer turn", async () => {
    await provider.start();
    await settleLong();
    let finishRecovery!: (stream: MediaStream) => void;
    getUserMedia.mockImplementationOnce(() => new Promise((resolve) => {
      finishRecovery = resolve;
    }));

    (provider.getStream()!.getTracks()[0] as unknown as FakeTrack).onended?.();
    await settleLong();
    expect(finishRecovery).toBeTypeOf("function");

    provider.stop();
    FakeWebSocket.instances.at(-1)?.onmessage?.({ data: JSON.stringify({ type: "final", text: "previous turn" }) });
    await settleLong();
    await provider.start();
    await settleLong();
    const newerStream = provider.getStream();
    expect(newerStream).not.toBeNull();

    const stale = makeStream();
    finishRecovery(stale.stream);
    await settleLong();

    expect(stale.track.stop).toHaveBeenCalledOnce();
    expect(provider.getStream()).toBe(newerStream);
    expect(provider.getDiagnostic().state).toBe("recording");
  });

  it("ignores late frames from the capture graph replaced by mic reacquisition", async () => {
    await provider.start();
    await settleLong();
    const staleCapture = captureOnFrame!;
    expect(staleCapture).toBeTypeOf("function");

    (provider.getStream()!.getTracks()[0] as unknown as FakeTrack).onended?.();
    await settleLong();
    const activeCapture = captureOnFrame!;
    expect(activeCapture).toBeTypeOf("function");
    expect(activeCapture).not.toBe(staleCapture);

    staleCapture(new Float32Array(160), 16_000);
    expect(provider.getDiagnostic().capturedSamples).toBe(0);
    activeCapture(new Float32Array(160), 16_000);
    expect(provider.getDiagnostic().capturedSamples).toBe(160);
  });

  it("exhausts reconnects when sockets repeatedly become ready without processing audio", async () => {
    await provider.start();
    await settleLong();
    const firstSocket = FakeWebSocket.instances.at(-1)!;
    identifyLocalRoute(firstSocket);
    captureOnFrame?.(new Float32Array(PCM_WIRE_BATCH_SAMPLES).fill(0.5), 16_000);
    await settleLong();
    FakeWebSocket.autoOpen = false;

    firstSocket.readyState = FakeWebSocket.CLOSED;
    firstSocket.onclose?.();
    for (const delay of [1_000, 2_000, 4_000, 8_000, 8_000]) {
      await vi.advanceTimersByTimeAsync(delay);
      await settleLong();
      const socket = FakeWebSocket.instances.at(-1)!;
      socket.onopen?.();
      await settleLong();
      socket.readyState = FakeWebSocket.CLOSED;
      socket.onclose?.();
      await settleLong();
    }

    expect(status).toHaveBeenCalledWith(expect.objectContaining({ code: "reconnect_exhausted" }));
    await vi.waitFor(() => expect(transcribeRetained).toHaveBeenCalledOnce());
    await vi.waitFor(() => expect(result).toHaveBeenCalledWith("recovered"));
  });

  it("resets reconnect backoff only after a newly processed cursor is durable", async () => {
    await provider.start();
    await settleLong();
    const firstSocket = FakeWebSocket.instances.at(-1)!;
    identifyLocalRoute(firstSocket);
    captureOnFrame?.(new Float32Array(PCM_WIRE_BATCH_SAMPLES).fill(0.5), 16_000);
    await settleLong();
    FakeWebSocket.autoOpen = false;

    firstSocket.readyState = FakeWebSocket.CLOSED;
    firstSocket.onclose?.();
    await vi.advanceTimersByTimeAsync(1_000);
    await settleLong();
    const secondSocket = FakeWebSocket.instances.at(-1)!;
    secondSocket.onopen?.();
    await settleLong();
    secondSocket.readyState = FakeWebSocket.CLOSED;
    secondSocket.onclose?.();

    await vi.advanceTimersByTimeAsync(2_000);
    await settleLong();
    const thirdSocket = FakeWebSocket.instances.at(-1)!;
    thirdSocket.onopen?.();
    await settleLong();
    thirdSocket.onmessage?.({ data: JSON.stringify({ type: "status", code: "processed_acknowledgement", processedSequence: 0 }) });
    await settleLong();
    thirdSocket.readyState = FakeWebSocket.CLOSED;
    thirdSocket.onclose?.();

    const beforeNextReconnect = FakeWebSocket.instances.length;
    await vi.advanceTimersByTimeAsync(999);
    expect(FakeWebSocket.instances).toHaveLength(beforeNextReconnect);
    await vi.advanceTimersByTimeAsync(1);
    await settleLong();
    expect(FakeWebSocket.instances).toHaveLength(beforeNextReconnect + 1);
  });

  it("reuses a preconnected socket when capture starts", async () => {
    provider.preConnect("en");
    await settle();
    expect(FakeWebSocket.instances).toHaveLength(1);

    await vi.advanceTimersByTimeAsync(4_000);

    await provider.start();
    await settle();

    expect(FakeWebSocket.instances).toHaveLength(1);
    expect(getUserMedia).toHaveBeenCalledOnce();
  });

  it("publishes a terminal diagnostic when microphone startup fails", async () => {
    const startupError = new Error("capture device disappeared");
    getUserMedia.mockRejectedValueOnce(startupError);
    const diagnostics = vi.fn();
    provider.onDiagnostic = diagnostics;

    await expect(provider.start()).rejects.toBe(startupError);

    const diagnostic = readStreamDiagnosticTelemetry()?.latest;
    expect(diagnostic?.state).toBe("failed");
    expect(diagnostic?.terminalReason).toBe("capture_start_failed");
    expect(diagnostic?.errorCodes).toContain("capture_start_failed");
    expect(diagnostic?.statusCodes).toContain("capture_starting");
    expect(diagnostics).toHaveBeenCalled();
  });

  it("releases the microphone and transport when capture graph startup fails", async () => {
    const captureError = new Error("audio graph unavailable");
    const acquiredStream = streams[0]!;
    provider.captureFactory = vi.fn(async () => { throw captureError; });
    const diagnostics = vi.fn();
    provider.onDiagnostic = diagnostics;

    await expect(provider.start()).rejects.toBe(captureError);

    expect(streams).toHaveLength(2);
    expect(FakeWebSocket.instances).toHaveLength(1);
    expect(provider.getStream()).toBeNull();
    expect(acquiredStream.track.stop).toHaveBeenCalledOnce();
    expect(FakeWebSocket.instances[0]!.readyState).toBe(FakeWebSocket.CLOSED);
    expect(provider.getDiagnostic()).toMatchObject({ state: "failed", terminalReason: "capture_start_failed" });
    expect(diagnostics).toHaveBeenCalled();
  });

  it("exhausts exactly five consecutive reconnect attempts and falls back", async () => {
    await provider.start();
    await settle();
    identifyLocalRoute(FakeWebSocket.instances.at(-1)!);
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();
    FakeWebSocket.autoOpen = false;
    for (const delay of [1_000, 2_000, 4_000, 8_000, 8_000]) {
      const socket = FakeWebSocket.instances.at(-1)!;
      socket.readyState = FakeWebSocket.CLOSED;
      socket.onclose?.();
      await vi.advanceTimersByTimeAsync(delay);
      await settle();
    }
    const exhausted = FakeWebSocket.instances.at(-1)!;
    exhausted.readyState = FakeWebSocket.CLOSED;
    exhausted.onclose?.();
    await settle();
    expect(status).toHaveBeenCalledWith(expect.objectContaining({ code: "reconnect_exhausted" }));
    expect(transcribeRetained).toHaveBeenCalledOnce();
    await vi.waitFor(() => expect(result).toHaveBeenCalledWith("recovered"));
  });

  it("recovers the unacknowledged journal tail after the bounded full-turn copy overflows", async () => {
    transcribeRetained.mockImplementation(async (audio: Blob) => {
      // Two 100 ms frames are present in the durable journal. The explicit
      // 3,200-byte whole-turn budget below overflows on the second frame, so
      // this proves fallback uses the journal rather than silently dropping
      // the long-session recovery tail.
      expect(audio.size).toBe(44 + (2 * PCM_WIRE_BATCH_SAMPLES * 2));
      return "recovered from journal tail";
    });
    provider = new PcmVoiceStreamProvider({
      getUserMedia,
      maxRetainedTurnAudioBytes: PCM_WIRE_BATCH_SAMPLES * 2,
      transport: new FakeWebSocket(transcribeRetained),
      captureFactory: async (_stream, onFrame) => {
        captureOnFrame = onFrame;
        return { stop: vi.fn() };
      },
      // Keep the durable journal larger than the short-turn UX copy so the
      // test isolates the recovery-source boundary.
      journalFactory: () => new TurnJournal(new MemoryTurnJournalStore(), "journal-tail", 0n, 16 * 1024 * 1024, "memory"),
    });
    provider.onStatus = status;
    provider.onResult = result;
    const partial = vi.fn();
    provider.onPartial = partial;

    await provider.start();
    await settle();
    const firstSocket = FakeWebSocket.instances.at(-1)!;
    // This is the server's early ready status. The browser must retain the
    // selected catalog identity even if the connection dies before a segment.
    firstSocket.onmessage?.({ data: JSON.stringify({
      type: "status", code: "session_ready", providerId: "whisper-local",
      modelId: "whisper-medium", providerTier: "local",
    }) });
    const frame = new Float32Array(PCM_WIRE_BATCH_SAMPLES).fill(0.5);
    captureOnFrame?.(frame, 16_000);
    captureOnFrame?.(frame, 16_000);
    await settleLong();
    firstSocket.onmessage?.({ data: JSON.stringify({ type: "partial", text: "first visible partial" }) });
    await settle();
    expect(partial).toHaveBeenCalledWith("first visible partial");

    FakeWebSocket.autoOpen = false;
    for (const delay of [1_000, 2_000, 4_000, 8_000, 8_000]) {
      const socket = FakeWebSocket.instances.at(-1)!;
      socket.readyState = FakeWebSocket.CLOSED;
      socket.onclose?.();
      await vi.advanceTimersByTimeAsync(delay);
      await settle();
    }
    const exhausted = FakeWebSocket.instances.at(-1)!;
    exhausted.readyState = FakeWebSocket.CLOSED;
    exhausted.onclose?.();
    await settleLong();

    expect(status).toHaveBeenCalledWith(expect.objectContaining({ code: "reconnect_exhausted" }));
    expect(transcribeRetained).toHaveBeenCalledOnce();
    expect(transcribeRetained.mock.calls[0]?.[2]).toMatchObject({
      providerTier: "local",
      providerId: "whisper-local",
      modelId: "whisper-medium",
    });
    expect(result).toHaveBeenCalledWith("recovered from journal tail");
  });

  it("refuses retained-audio recovery when the selected stream route is not local", async () => {
    await provider.start();
    await settle();
    const socket = FakeWebSocket.instances.at(-1)!;
    socket.onmessage?.({ data: JSON.stringify({
      type: "status",
      code: "provider_identity",
      providerId: "openai-gpt-live",
      modelId: "gpt-live-1.1",
      providerTier: "byok",
    }) });
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();

    provider.stop();
    FakeWebSocket.autoOpen = false;
    for (const delay of [1_000, 2_000, 4_000, 8_000, 8_000]) {
      FakeWebSocket.instances.at(-1)!.onclose?.();
      await vi.advanceTimersByTimeAsync(delay);
      await settleLong();
    }
    FakeWebSocket.instances.at(-1)!.onclose?.();
    await settleLong();

    expect(transcribeRetained).not.toHaveBeenCalled();
    expect(result).not.toHaveBeenCalled();
    expect(provider.getDiagnostic()).toMatchObject({
      state: "failed",
      errorCodes: expect.arrayContaining(["retained_route_unavailable"]),
    });
    expect(journal.read().retainedBytes).toBeGreaterThan(0);
  });

  it("keeps the primary streaming provider identity when fallback uses another provider", async () => {
    provider.dispose();
    const transcribeRetainedWithIdentity = vi.fn(async () => ({
      text: "recovered by fallback",
      providerIdentity: { providerId: "whisper-local", modelId: "whisper-medium" },
    }));
    provider = new PcmVoiceStreamProvider({
      getUserMedia,
      transport: new FakeWebSocket(transcribeRetained, transcribeRetainedWithIdentity),
      captureFactory: async (_stream, onFrame) => {
        captureOnFrame = onFrame;
        return { stop: vi.fn() };
      },
      journalFactory: () => new TurnJournal(new MemoryTurnJournalStore(), "provider-identity", 0n, 16 * 1024 * 1024, "memory"),
    });
    provider.onResult = result;

    await provider.start();
    await settle();
    const socket = FakeWebSocket.instances.at(-1)!;
    socket.onmessage?.({ data: JSON.stringify({
      type: "status",
      code: "provider_identity",
      providerId: "kyutai",
      modelId: "kyutai/stt-1b-en_fr",
      providerTier: "local",
    }) });
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();

    FakeWebSocket.autoOpen = false;
    for (const delay of [1_000, 2_000, 4_000, 8_000, 8_000]) {
      const active = FakeWebSocket.instances.at(-1)!;
      active.readyState = FakeWebSocket.CLOSED;
      active.onclose?.();
      await vi.advanceTimersByTimeAsync(delay);
      await settle();
    }
    const exhausted = FakeWebSocket.instances.at(-1)!;
    exhausted.readyState = FakeWebSocket.CLOSED;
    exhausted.onclose?.();
    await settleLong();

    expect(transcribeRetainedWithIdentity).toHaveBeenCalledOnce();
    expect(result).toHaveBeenCalledWith("recovered by fallback");
    expect(provider.getDiagnostic()).toMatchObject({
      providerId: "kyutai",
      modelId: "kyutai/stt-1b-en_fr",
    });
  });

  it("measures terminal-tail lag for a non-segmenting streaming route", async () => {
    await provider.start();
    await settle();
    const socket = FakeWebSocket.instances.at(-1)!;
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();
    provider.stop();
    socket.onmessage?.({ data: JSON.stringify({ type: "final", text: "terminal tail" }) });
    await settle();

    await vi.waitFor(() => expect(provider.getDiagnostic().committedTextLagMs).not.toBeNull());
    expect(result).toHaveBeenCalledWith("terminal tail");
  });

  it("deletes durable journal data after a committed terminal final", async () => {
    const store = new MemoryTurnJournalStore();
    provider = new PcmVoiceStreamProvider({
      getUserMedia,
      transport: new FakeWebSocket(transcribeRetained),
      captureFactory: async (_stream, onFrame) => {
        captureOnFrame = onFrame;
        return { stop: vi.fn() };
      },
      journalFactory: () => new TurnJournal(store, "cleanup", 0n, 16 * 1024 * 1024, "memory"),
    });
    provider.onResult = result;

    await provider.start();
    await settle();
    const socket = FakeWebSocket.instances.at(-1)!;
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settleLong();
    provider.stop();
    socket.onmessage?.({ data: JSON.stringify({ type: "final", text: "committed" }) });
    await settleLong();

    expect(result).toHaveBeenCalledWith("committed");
    expect(await store.load(TurnJournal.key("cleanup", 0n))).toBeUndefined();
  });

  it("starts with a fresh session identity when retained audio has expired", async () => {
    class ExpiringStore extends MemoryTurnJournalStore {
      async prepareRestore(key: string): Promise<boolean> {
        await this.remove(key);
        return false;
      }
    }
    const store = new ExpiringStore();
    const oldSessionId = "expired-session";
    const oldJournal = new TurnJournal(store, oldSessionId, 0n, 16 * 1024 * 1024, "persistent");
    await oldJournal.append({ sequence: 0n, startSample: 0n, endSample: 2n, audio: new Uint8Array([1, 2]).buffer, sha256: new ArrayBuffer(0) });
    rememberUnfinishedSession({ sessionId: oldSessionId, resumeToken: "expired-token" });
    provider = new PcmVoiceStreamProvider({
      getUserMedia,
      transport: new FakeWebSocket(transcribeRetained),
      captureFactory: async (_stream, onFrame) => {
        captureOnFrame = onFrame;
        return { stop: vi.fn() };
      },
      journalFactory: () => new TurnJournal(store, oldSessionId, 0n, 16 * 1024 * 1024, "persistent"),
    });
    provider.onStatus = status;

    await provider.start();
    await settle();

    expect(await store.load(TurnJournal.key(oldSessionId, 0n))).toBeUndefined();
    expect(FakeWebSocket.instances.at(-1)?.start.sessionId).not.toBe(oldSessionId);
    expect(status).toHaveBeenCalledWith(expect.objectContaining({ code: "retention_expired" }));
  });

  it("treats a mid-recording final as backend loss and reconnects", async () => {
    await provider.start();
    await settle();
    const socket = FakeWebSocket.instances[0]!;
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();

    socket.onmessage?.({ data: JSON.stringify({ type: "final", text: "" }) });
    await settle();
    expect(result).not.toHaveBeenCalled();
    expect(status).toHaveBeenCalledWith(expect.objectContaining({ code: "backend_degraded" }));

    socket.readyState = FakeWebSocket.CLOSED;
    socket.onclose?.();
    await vi.advanceTimersByTimeAsync(1_000);
    await settle();
    expect(FakeWebSocket.instances).toHaveLength(2);
  });

  it("keeps a recoverable backend restart in the reconnect path", async () => {
    await provider.start();
    await settle();
    const socket = FakeWebSocket.instances[0]!;
    const onError = vi.fn();
    provider.onError = onError;
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();

    socket.onmessage?.({ data: JSON.stringify({ type: "error", code: "backend_restart", text: "backend restarted" }) });
    await settle();

    expect(onError).not.toHaveBeenCalled();
    expect(status).toHaveBeenCalledWith(expect.objectContaining({ code: "backend_restart" }));
    expect(provider.getDiagnostic().state).not.toBe("failed");

    socket.readyState = FakeWebSocket.CLOSED;
    socket.onclose?.();
    await vi.advanceTimersByTimeAsync(1_000);
    await settle();
    expect(FakeWebSocket.instances).toHaveLength(2);
  });

  it("ignores a late close from a stale socket after reconnect ownership moves", async () => {
    await provider.start();
    await settle();
    const first = FakeWebSocket.instances[0]!;
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();

    first.readyState = FakeWebSocket.CLOSED;
    first.onclose?.();
    await vi.advanceTimersByTimeAsync(1_000);
    await settle();

    expect(FakeWebSocket.instances).toHaveLength(2);
    const second = FakeWebSocket.instances[1]!;
    expect(second.readyState).toBe(FakeWebSocket.OPEN);

    // A browser can deliver the old socket's close callback after the
    // replacement has opened. It must not schedule a third replay socket.
    first.onclose?.();
    await vi.advanceTimersByTimeAsync(1_000);
    await settle();
    expect(FakeWebSocket.instances).toHaveLength(2);
  });

  it("does not reacquire a track after an intentional final", async () => {
    await provider.start();
    await settle();
    const stream = provider.getStream()!;
    const track = stream.getTracks()[0] as unknown as FakeTrack;
    provider.stop();
    track.readyState = "ended";
    track.onended?.();
    await settle();
    expect(getUserMedia).toHaveBeenCalledOnce();
  });

  it("clears the pending acknowledgement notice when the server reports progress", async () => {
    await provider.start();
    await settle();
    const socket = FakeWebSocket.instances[0]!;
    await vi.advanceTimersByTimeAsync(1_000);
    expect(status).toHaveBeenCalledWith(expect.objectContaining({ code: "server_ack_pending" }));

    socket.onmessage?.({ data: JSON.stringify({ type: "status", code: "processed_acknowledgement", text: "processed", processedSequence: 0 }) });
    await vi.advanceTimersByTimeAsync(1_000);
    expect(status.mock.calls.filter(([value]) => value.code === "server_ack_pending")).toHaveLength(1);
  });

  it("surfaces a clean backend error without leaking a dial string", async () => {
    await provider.start();
    await settle();
    const socket = FakeWebSocket.instances[0]!;
    const onError = vi.fn();
    provider.onError = onError;
    socket.onmessage?.({ data: JSON.stringify({ type: "error", code: "backend_unavailable", text: "dial tcp 127.0.0.1:19630: connect: connection refused" }) });
    expect(onError).toHaveBeenCalledWith("The speech backend is unavailable; retry the turn.");
    expect(onError.mock.calls[0]?.[0]).not.toContain("127.0.0.1:19630");
  });

	it("delivers an empty final after an intentional stop", async () => {
    await provider.start();
    await settle();
    const socket = FakeWebSocket.instances[0]!;
    provider.stop();
    await settle();
    socket.onmessage?.({ data: JSON.stringify({ type: "final", text: "   " }) });
		await vi.waitFor(() => expect(result).toHaveBeenCalledWith(""));
	});

  it("recovers retained PCM when the socket closes after stop", async () => {
		await provider.start();
		await settle();
		identifyLocalRoute(FakeWebSocket.instances[0]!);
		captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
		await settle();
		FakeWebSocket.autoOpen = false;
		provider.stop();
		for (const delay of [1_000, 2_000, 4_000, 8_000, 8_000]) {
			FakeWebSocket.instances.at(-1)!.onclose?.();
			await vi.advanceTimersByTimeAsync(delay);
			await settleLong();
		}
		FakeWebSocket.instances.at(-1)!.onclose?.();
		await settleLong();
		expect(transcribeRetained).toHaveBeenCalledOnce();
		expect(result).toHaveBeenCalledWith("recovered");
	});

	it("replays a stopped stream before trying unary recovery", async () => {
		await provider.start();
		await settle();
		const first = FakeWebSocket.instances.at(-1)!;
		first.onmessage?.({ data: JSON.stringify({
			type: "status", code: "provider_identity", providerId: "sherpa-stt",
			modelId: "sherpa-onnx/streaming-zipformer-en-2023-06-26", providerTier: "local",
		}) });
		captureOnFrame?.(new Float32Array(1_600).fill(0.5), 16_000);
		await settle();
		provider.stop();
		first.readyState = FakeWebSocket.CLOSED;
		first.onclose?.();
		await settle();

		// Stop fences further microphone capture, but the bounded reconnect
		// budget still owns the retained, unacknowledged journal tail.
		await vi.advanceTimersByTimeAsync(1_000);
		await settleLong();
		const replay = FakeWebSocket.instances.at(-1)!;
		expect(FakeWebSocket.instances).toHaveLength(2);
		expect(replay.sendAudio).toHaveBeenCalledOnce();
		expect(replay.end).toHaveBeenCalledOnce();
		expect(transcribeRetained).not.toHaveBeenCalled();

		replay.onmessage?.({ data: JSON.stringify({ type: "final", text: "complete recovered tail" }) });
		await settleLong();
		expect(result).toHaveBeenCalledWith("complete recovered tail");
		expect(provider.getDiagnostic()).toMatchObject({ state: "completed", committedSamples: 1_600 });
	});

	it("recovers retained PCM when a stopped turn receives an empty final", async () => {
    await provider.start();
    await settle();
    identifyLocalRoute(FakeWebSocket.instances[0]!);
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();
    const socket = FakeWebSocket.instances[0]!;
    provider.stop();
    await settle();
    socket.onmessage?.({ data: JSON.stringify({ type: "final", text: "" }) });
    await settle();
    expect(transcribeRetained).toHaveBeenCalledOnce();
    await vi.waitFor(() => expect(result).toHaveBeenCalledWith("recovered"));
  });

  it("announces bounded recovery before using the retained-audio path", async () => {
    await provider.start();
    await settle();
    identifyLocalRoute(FakeWebSocket.instances[0]!);
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();
    const socket = FakeWebSocket.instances[0]!;
    provider.stop();
    FakeWebSocket.autoOpen = false;
    socket.readyState = FakeWebSocket.CLOSED;
    socket.onclose?.();
    for (const delay of [1_000, 2_000, 4_000, 8_000, 8_000]) {
      await vi.advanceTimersByTimeAsync(delay);
      FakeWebSocket.instances.at(-1)!.onclose?.();
      await settleLong();
    }
    await settleLong();
    expect(status).toHaveBeenCalledWith(expect.objectContaining({
      code: "buffered_recovery",
      message: "Streaming degraded; recovering retained audio through bounded batch transcription.",
    }));
    await vi.waitFor(() => expect(result).toHaveBeenCalledWith("recovered"));
  });

  it("clears the final-pending notice when the final arrives", async () => {
    await provider.start();
    await settle();
    const socket = FakeWebSocket.instances[0]!;
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();
    provider.stop();
    await vi.advanceTimersByTimeAsync(3_000);
    const pendingCount = status.mock.calls.filter(([value]) => value.code === "final_pending").length;
    expect(pendingCount).toBe(1);
    socket.onmessage?.({ data: JSON.stringify({ type: "final", text: "done" }) });
    await vi.advanceTimersByTimeAsync(3_000);
    expect(status.mock.calls.filter(([value]) => value.code === "final_pending")).toHaveLength(pendingCount);
  });

  it("measures the shared stop marker through the 120 ms settle window to a real stream final", async () => {
    // Vitest 2 (the Audio Tools UI runner) advances timers but does not fake
    // performance.now by default. Drive the monotonic clock explicitly so
    // this assertion verifies the 120 ms settle plus the 80 ms final wait in
    // both the package's Vitest 3 runner and consumer Vitest 2 bridges.
    let monotonicMs = 0;
    vi.spyOn(globalThis.performance, "now").mockImplementation(() => monotonicMs);
    await provider.start();
    await settle();
    const socket = FakeWebSocket.instances[0]!;
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();

    // The shared capture action marks stop before its user-tap encoder settle.
    provider.markStopRequested();
    monotonicMs += 120;
    await vi.advanceTimersByTimeAsync(120);
    provider.stop();
    await settle();
    monotonicMs += 80;
    await vi.advanceTimersByTimeAsync(80);
    socket.onmessage?.({ data: JSON.stringify({ type: "final", text: "done" }) });
    await settle();

    expect(provider.getDiagnostic()).toMatchObject({
      stopToFinalTailMs: 200,
      terminalReason: "final",
    });
  });

  it("uses retained-audio recovery even when the fallback transcript is empty", async () => {
    transcribeRetained.mockResolvedValueOnce("   ");
    await provider.start();
    await settle();
    identifyLocalRoute(FakeWebSocket.instances.at(-1)!);
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();
    FakeWebSocket.autoOpen = false;
    for (const delay of [1_000, 2_000, 4_000, 8_000, 8_000]) {
      const socket = FakeWebSocket.instances.at(-1)!;
      socket.readyState = FakeWebSocket.CLOSED;
      socket.onclose?.();
      await vi.advanceTimersByTimeAsync(delay);
      await settle();
    }
    const exhausted = FakeWebSocket.instances.at(-1)!;
    exhausted.readyState = FakeWebSocket.CLOSED;
    exhausted.onclose?.();
    await settle();
    expect(result).toHaveBeenCalledWith("");
  });

  it("reacquires an ended track without replacing the socket or journal", async () => {
    await provider.start();
    await settle();
    const active = provider.getStream()!;
    const track = active.getTracks()[0] as unknown as FakeTrack;
    track.readyState = "ended";
    track.onended?.();
    await settle();
    expect(getUserMedia).toHaveBeenCalledTimes(2);
    expect(provider.getStream()).not.toBe(active);
    expect(FakeWebSocket.instances).toHaveLength(1);
    expect(status).toHaveBeenCalledWith(expect.objectContaining({ code: "mic_reacquired" }));
  });

  it("reports mute and unmute without closing the active socket", async () => {
    await provider.start();
    await settle();
    const socket = FakeWebSocket.instances[0]!;
    const track = provider.getStream()!.getTracks()[0] as unknown as FakeTrack;
    track.onmute?.();
    track.onunmute?.();
    expect(socket.readyState).toBe(FakeWebSocket.OPEN);
    expect(status).toHaveBeenCalledWith(expect.objectContaining({ code: "mic_muted" }));
    expect(status).toHaveBeenCalledWith(expect.objectContaining({ code: "mic_unmuted" }));
  });

  it("terminates with mic_source_lost while retaining the recovery journal when reacquisition fails", async () => {
    await provider.start();
    await settle();
    getUserMedia.mockRejectedValueOnce(new Error("permission revoked"));
    const track = provider.getStream()!.getTracks()[0] as unknown as FakeTrack;
    const onError = vi.fn();
    provider.onError = onError;
    track.readyState = "ended";
    track.onended?.();
    await settle();
    expect(status).toHaveBeenCalledWith(expect.objectContaining({ code: "mic_reacquiring" }));
    expect(status).toHaveBeenCalledWith(expect.objectContaining({ code: "mic_source_lost" }));
    expect(onError).toHaveBeenCalledOnce();
    expect(provider.getDiagnostic()).toMatchObject({ state: "failed", terminalReason: "mic_source_lost" });
  });

  it("releases a reacquired stream when its capture graph cannot be created", async () => {
    await provider.start();
    await settle();
    const originalTrack = provider.getStream()!.getTracks()[0] as unknown as FakeTrack;
    const reacquired = streams[0]!;
    const onError = vi.fn();
    provider.onError = onError;
    provider.captureFactory = vi.fn(async () => {
      throw new Error("AudioWorklet and ScriptProcessor setup failed");
    });

    originalTrack.readyState = "ended";
    originalTrack.onended?.();
    await settle();

    expect(getUserMedia).toHaveBeenCalledTimes(2);
    expect(reacquired.track.stop).toHaveBeenCalledOnce();
    expect(provider.getStream()).toBeNull();
    expect(getActiveMicLeases().some((lease) => lease.owner === "voice-stream")).toBe(false);
    expect(status).toHaveBeenCalledWith(expect.objectContaining({ code: "mic_source_lost" }));
    expect(onError).toHaveBeenCalledOnce();
    expect(provider.getDiagnostic()).toMatchObject({ state: "failed", terminalReason: "mic_source_lost" });
  });

  it("drops PCM frames after tail-drop is armed", async () => {
    await provider.start();
    await settle();
    const socket = FakeWebSocket.instances[0]!;
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();
    const beforeDrop = socket.send.mock.calls.length;
    provider.dropTail();
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();
    expect(socket.send.mock.calls.length).toBe(beforeDrop);
  });

  it("clears tail-drop on the next start", async () => {
    await provider.start();
    await settle();
    provider.dropTail();
    provider.stop();
    FakeWebSocket.instances.at(-1)?.onmessage?.({ data: JSON.stringify({ type: "final", text: "" }) });
    await settleLong();
    await provider.start();
    await settle();
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();
    expect(provider.getDiagnostic().capturedSequence).toBe(0);
  });

  it("drops the whole-turn retry copy at its fixed recovery budget", async () => {
    provider.dispose();
    provider = new PcmVoiceStreamProvider({
      getUserMedia,
      maxRetainedTurnAudioBytes: 512,
      transport: new FakeWebSocket(transcribeRetained),
      captureFactory: async (_stream, onFrame) => {
        captureOnFrame = onFrame;
        return { stop: vi.fn() };
      },
      journalFactory: () => new TurnJournal(new MemoryTurnJournalStore(), "bounded-retry", 0n, 16 * 1024 * 1024, "memory"),
    });
    await provider.start();
    await settle();
    captureOnFrame?.(new Float32Array(1_600).fill(0.5), 16_000);
    await settle();
    provider.stop();

    expect(provider.getLastTurnAudio()).toBeNull();
    expect(MAX_RETAINED_TURN_AUDIO_BYTES).toBe(16 * 1024 * 1024);
  });

  it("captures and acknowledges a 60-minute session with bounded journal retention", async () => {
    vi.useRealTimers();
    const startedAt = performance.now();
    await provider.start();
    await settleLong();
    const socket = FakeWebSocket.instances[0]!;
    socket.sendAudio.mockImplementation((value) => {
      const sequence = Number(value.sequence);
      queueMicrotask(() => socket.onmessage?.({ data: JSON.stringify({ type: "status", code: "processed_acknowledgement", text: "processed", processedSequence: sequence }) }));
    });
    const frame = new Float32Array(BASELINE_FRAME_SAMPLES).fill(0.25);
    for (let index = 0; index < BASELINE_SIMULATED_SECONDS / BASELINE_FRAME_SECONDS; index += 1) {
      captureOnFrame?.(frame, 16_000);
      // Pace each virtual minute through its real journal/transport chain.
      // A synchronous 115 MiB burst is outside bounded capture support.
      await (provider as unknown as { writes: Promise<void> }).writes;
    }
    const internals = provider as unknown as { writes: Promise<void> };
    await internals.writes;
    await internals.writes;
    const journal = (provider as unknown as { journal: TurnJournal | null }).journal;
    const snapshot = journal?.read();
    const wireIntervals = socket.sendAudio.mock.calls.map(([value]) => ({
      sequence: Number(value.sequence),
      startSample: Number(value.startSample),
      endSample: Number(value.endSample),
    }));
    const expectedIntervalCount = BASELINE_SIMULATED_SECONDS * 16_000 / PCM_WIRE_BATCH_SAMPLES;
    const intervalAccounting = wireIntervals.length === expectedIntervalCount && wireIntervals.every((interval, index) => (
      interval.sequence === index
      && interval.startSample === index * PCM_WIRE_BATCH_SAMPLES
      && interval.endSample === (index + 1) * PCM_WIRE_BATCH_SAMPLES
    ));
    const wallClockMs = performance.now() - startedAt;
    const wireRate = wireIntervals.length / BASELINE_SIMULATED_SECONDS;
    const retainedBytes = snapshot?.retainedBytes ?? Number.POSITIVE_INFINITY;
    expect(intervalAccounting).toBe(true);
    expect(wireIntervals).toHaveLength(expectedIntervalCount);
    expect(wallClockMs).toBeLessThan(60_000);
    expect(retainedBytes).toBeLessThanOrEqual(2 * PCM_WIRE_BATCH_SAMPLES * 2);
    expect(wireIntervals.length).toBe(BASELINE_SIMULATED_SECONDS * 16_000 / PCM_WIRE_BATCH_SAMPLES);
    expect(wireRate).toBeLessThanOrEqual(15);
    const telemetry = readStreamDiagnosticTelemetry()?.latest;
    expect(telemetry?.capturedSamples).toBe(BASELINE_SIMULATED_SECONDS * 16_000);
    expect(telemetry?.retainedBytes).toBeLessThanOrEqual(2 * PCM_WIRE_BATCH_SAMPLES * 2);
    provider.stop();
    await settleLong();
    socket.onmessage?.({ data: JSON.stringify({ type: "final", text: "completed terminal" }) });
    await settleLong();
    expect(result).toHaveBeenCalledWith("completed terminal");
    expect(provider.getDiagnostic().state).toBe("completed");
    expect(provider.getDiagnostic().terminalReason).toBe("final");
    // The budget for a simulated hour is 60 s of wall clock, asserted above.
    // The runner timeout sits above that budget on purpose: when it sat below,
    // a run that exceeded the budget under parallel load reported "timed out"
    // instead of the measurement, which is the difference between a signal and
    // a shrug. Let the assertion be the thing that fails.
  }, 90_000);

  // ── Browser-edge interval accounting ────────────────────────────────────
  //
  // `interval_accounting_exactly_once` in the soak harness reads
  // `sentSequence` and `processedSequence` off the published telemetry global,
  // not off the provider. Nothing asserted that those two fields ever leave
  // their -1 sentinel, so every published soak artifact has reported
  // `sent=-1 processed=-1` and the assertion has failed as "not collected"
  // while reading like "frames were dropped". These tests hold the seam the
  // soak depends on.
  it("advances the published sent cursor as batches reach the wire", async () => {
    await provider.start();
    await settle();
    const frame = new Float32Array(PCM_WIRE_BATCH_SAMPLES).fill(0.5);
    for (let index = 0; index < 4; index += 1) captureOnFrame?.(frame, 16_000);
    await settleLong();

    const published = readStreamDiagnosticTelemetry()?.latest;
    expect(published?.capturedSequence).toBe(3);
    expect(published?.sentSequence).toBe(3);
  });

  it("advances the published processed cursor when the server acknowledges", async () => {
    await provider.start();
    await settle();
    const socket = FakeWebSocket.instances.at(-1)!;
    const frame = new Float32Array(PCM_WIRE_BATCH_SAMPLES).fill(0.5);
    for (let index = 0; index < 3; index += 1) captureOnFrame?.(frame, 16_000);
    await settleLong();

    socket.onmessage?.({ data: JSON.stringify({ type: "status", code: "processed_acknowledgement", text: "processed", processedSequence: 2 }) });
    await settleLong();

    expect(readStreamDiagnosticTelemetry()?.latest.processedSequence).toBe(2);
  });

  it("still reports a sent cursor when frames are captured before the socket opens", async () => {
    FakeWebSocket.autoOpen = false;
    await provider.start();
    await settle();
    const frame = new Float32Array(PCM_WIRE_BATCH_SAMPLES).fill(0.5);
    for (let index = 0; index < 3; index += 1) captureOnFrame?.(frame, 16_000);
    await settleLong();
    // Nothing has reached the wire yet; the sentinel is still honest here.
    expect(readStreamDiagnosticTelemetry()?.latest.sentSequence).toBe(-1);

    FakeWebSocket.instances.at(-1)!.onopen?.();
    await settleLong();

    expect(readStreamDiagnosticTelemetry()?.latest.sentSequence).toBe(2);
  });
});
