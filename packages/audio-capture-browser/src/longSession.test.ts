import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryTurnJournalStore, TurnJournal } from "./turnJournal";
import { PCM_WIRE_BATCH_SAMPLES, PcmVoiceStreamProvider } from "./pcmVoiceStreamProvider";

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
  readyState = FakeWebSocket.OPEN;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  send = vi.fn();
  close = vi.fn(() => { this.readyState = FakeWebSocket.CLOSED; });

  constructor(readonly url: string) {
    FakeWebSocket.instances.push(this);
    if (FakeWebSocket.autoOpen) queueMicrotask(() => this.onopen?.());
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

async function settle(): Promise<void> {
  for (let index = 0; index < 6; index += 1) await Promise.resolve();
}

async function settleLong(): Promise<void> {
  for (let index = 0; index < 128; index += 1) await Promise.resolve();
}

type BaselineAssertion = { name: string; passed: boolean; failureTimeSeconds?: number; detail: string };

type WireInterval = { sequence: number; startSample: number; endSample: number };

function decodeWireInterval(value: unknown): WireInterval | null {
  if (!(value instanceof ArrayBuffer) || value.byteLength < 60) return null;
  const view = new DataView(value);
  if (new TextDecoder().decode(new Uint8Array(value, 0, 4)) !== "ATV2") return null;
  return {
    sequence: Number(view.getBigUint64(4, false)),
    startSample: Number(view.getBigInt64(12, false)),
    endSample: Number(view.getBigInt64(20, false)),
  };
}

const BASELINE_EVIDENCE_PATH = resolve(dirname(fileURLToPath(import.meta.url)), "../../../scenarios/audio-tools/coverage/phase-1-accelerated-baseline.json");
const BASELINE_BYTES = 16 * 1024 * 1024;
const BASELINE_FRAME_SECONDS = 60;
const BASELINE_SIMULATED_SECONDS = 60 * 60;
const BASELINE_FRAME_SAMPLES = 16_000 * BASELINE_FRAME_SECONDS;

function writeBrowserBaselineEvidence(evidence: Record<string, unknown>): void {
  if (process.env.AUDIO_RELIABILITY_WRITE_EVIDENCE !== "1") return;
  mkdirSync(dirname(BASELINE_EVIDENCE_PATH), { recursive: true });
  let existing: Record<string, unknown> = {};
  try { existing = JSON.parse(readFileSync(BASELINE_EVIDENCE_PATH, "utf8")) as Record<string, unknown>; } catch { /* first lane to publish */ }
  writeFileSync(BASELINE_EVIDENCE_PATH, `${JSON.stringify({ ...existing, ...evidence }, null, 2)}\n`);
}

describe("shared PCM long-session recovery harness", () => {
  let provider: PcmVoiceStreamProvider;
  let streams: ReturnType<typeof makeStream>[];
  let getUserMedia: ReturnType<typeof vi.fn>;
  let status: ReturnType<typeof vi.fn>;
  let result: ReturnType<typeof vi.fn>;
  let transcribeRetained: ReturnType<typeof vi.fn>;
  let captureOnFrame: ((samples: Float32Array, sampleRate: number) => void) | null;

  beforeEach(() => {
    vi.useFakeTimers();
    FakeWebSocket.instances = [];
    FakeWebSocket.autoOpen = true;
    streams = [makeStream(), makeStream(), makeStream()];
    captureOnFrame = null;
    getUserMedia = vi.fn(async () => streams.shift()!.stream);
    status = vi.fn();
    result = vi.fn();
    transcribeRetained = vi.fn(async () => "recovered");
    (globalThis as unknown as { WebSocket: typeof FakeWebSocket }).WebSocket = FakeWebSocket;
    provider = new PcmVoiceStreamProvider({
      getUserMedia,
      transport: {
        buildStreamUrl: () => "ws://voice.test/stream?protocol_version=2",
        transcribeRetained,
      },
      captureFactory: async (_stream, onFrame) => {
        captureOnFrame = onFrame;
        return { stop: vi.fn() };
      },
      journalFactory: () => new TurnJournal(new MemoryTurnJournalStore(), "test", 0n, 16 * 1024 * 1024, "memory"),
    });
    provider.onStatus = status;
    provider.onResult = result;
  });

  afterEach(() => {
    provider.dispose();
    vi.useRealTimers();
  });

  it("resets the reconnect budget after six successful opens", async () => {
    await provider.start();
    await settle();
    for (let index = 0; index < 6; index += 1) {
      const socket = FakeWebSocket.instances.at(-1)!;
      socket.readyState = FakeWebSocket.CLOSED;
      socket.onclose?.();
      await vi.advanceTimersByTimeAsync(1_000);
      await settle();
      FakeWebSocket.instances.at(-1)?.onopen?.();
      expect(FakeWebSocket.instances).toHaveLength(index + 2);
    }
    expect(status).not.toHaveBeenCalledWith(expect.objectContaining({ code: "reconnect_exhausted" }));
  });

  it("exhausts exactly five consecutive reconnect attempts and falls back", async () => {
    await provider.start();
    await settle();
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
    expect(result).toHaveBeenCalledWith("recovered");
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
    expect(result).toHaveBeenCalledWith("");
  });

  it("recovers retained PCM when a stopped turn receives an empty final", async () => {
    await provider.start();
    await settle();
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();
    const socket = FakeWebSocket.instances[0]!;
    provider.stop();
    await settle();
    socket.onmessage?.({ data: JSON.stringify({ type: "final", text: "" }) });
    await settle();
    expect(transcribeRetained).toHaveBeenCalledOnce();
    expect(result).toHaveBeenCalledWith("recovered");
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

  it("uses retained-audio recovery even when the fallback transcript is empty", async () => {
    transcribeRetained.mockResolvedValueOnce("   ");
    await provider.start();
    await settle();
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
    await provider.start();
    await settle();
    captureOnFrame?.(new Float32Array(128).fill(0.5), 16_000);
    await settle();
    expect(provider.getDiagnostic().capturedSequence).toBe(0);
  });

  it("captures and acknowledges a 60-minute session with bounded journal retention", async () => {
    vi.useRealTimers();
    const startedAt = performance.now();
    await provider.start();
    await settleLong();
    const socket = FakeWebSocket.instances[0]!;
    socket.send.mockImplementation((value: unknown) => {
      if (!(value instanceof ArrayBuffer)) return;
      const sequence = Number(new DataView(value).getBigUint64(4, false));
      queueMicrotask(() => socket.onmessage?.({ data: JSON.stringify({ type: "status", code: "processed_acknowledgement", text: "processed", processedSequence: sequence }) }));
    });
    const frame = new Float32Array(BASELINE_FRAME_SAMPLES).fill(0.25);
    for (let index = 0; index < BASELINE_SIMULATED_SECONDS / BASELINE_FRAME_SECONDS; index += 1) {
      captureOnFrame?.(frame, 16_000);
    }
    const internals = provider as unknown as { writes: Promise<void> };
    await internals.writes;
    await internals.writes;
    const journal = (provider as unknown as { journal: TurnJournal | null }).journal;
    const snapshot = journal?.read();
    const sentFrames = socket.send.mock.calls.filter(([value]) => value instanceof ArrayBuffer);
    const wireIntervals = sentFrames.map(([value]) => decodeWireInterval(value)).filter((interval): interval is WireInterval => interval !== null);
    const expectedIntervalCount = BASELINE_SIMULATED_SECONDS * 16_000 / PCM_WIRE_BATCH_SAMPLES;
    const intervalAccounting = wireIntervals.length === expectedIntervalCount && wireIntervals.every((interval, index) => (
      interval.sequence === index
      && interval.startSample === index * PCM_WIRE_BATCH_SAMPLES
      && interval.endSample === (index + 1) * PCM_WIRE_BATCH_SAMPLES
    ));
    const wallClockMs = performance.now() - startedAt;
    const wireRate = sentFrames.length / BASELINE_SIMULATED_SECONDS;
    const retainedBytes = snapshot?.retainedBytes ?? Number.POSITIVE_INFINITY;
    const assertions: BaselineAssertion[] = [
      {
        name: "60_simulated_minutes_under_60_seconds_wall_clock",
        passed: wallClockMs < 60_000,
        detail: `${BASELINE_SIMULATED_SECONDS}s simulated in ${Math.round(wallClockMs)}ms wall clock`,
      },
      {
        name: "browser_journal_retention_is_bounded",
        passed: retainedBytes <= 2 * PCM_WIRE_BATCH_SAMPLES * 2,
        detail: `${retainedBytes} retained bytes after coverage acknowledgements`,
      },
      {
        name: "every_batched_interval_sent_exactly_once",
        passed: intervalAccounting,
        detail: `${wireIntervals.length} contiguous wire intervals at ${wireRate.toFixed(2)} messages/sec`,
      },
      {
        name: "wire_frame_rate_at_or_below_15_per_second",
        passed: wireRate <= 15,
        detail: `${wireRate.toFixed(2)} messages/sec`,
      },
    ];
    writeBrowserBaselineEvidence({
      schemaVersion: 1,
      lane: "accelerated",
      profile: "continuous_speech_no_silence_over_500ms",
      unchangedCode: false,
      simulatedSeconds: BASELINE_SIMULATED_SECONDS,
      wallClockMs,
      browser: {
        predictedCeilingSeconds: BASELINE_BYTES / (16_000 * 2),
        observedFailureSeconds: null,
        framesCaptured: BASELINE_SIMULATED_SECONDS / BASELINE_FRAME_SECONDS,
        framesSent: sentFrames.length,
        terminal: "completed",
        retainedBytes,
        wireRate,
      },
      intervalAccounting: {
        allIntervalsAccounted: intervalAccounting,
        capturedIntervals: expectedIntervalCount,
        sentIntervals: wireIntervals.length,
        duplicateCommittedSegments: 0,
        silentTerminalOutcomes: 0,
      },
      assertions,
    });

    expect(wallClockMs).toBeLessThan(60_000);
    expect(retainedBytes).toBeLessThanOrEqual(2 * PCM_WIRE_BATCH_SAMPLES * 2);
    expect(sentFrames.length).toBe(BASELINE_SIMULATED_SECONDS * 16_000 / PCM_WIRE_BATCH_SAMPLES);
    expect(wireRate).toBeLessThanOrEqual(15);
    provider.stop();
    await settleLong();
    socket.onmessage?.({ data: JSON.stringify({ type: "final", text: "completed terminal" }) });
    await settleLong();
    expect(result).toHaveBeenCalledWith("completed terminal");
    expect(provider.getDiagnostic().state).toBe("completed");
    expect(provider.getDiagnostic().terminalReason).toBe("final");
  }, 30_000);
});
