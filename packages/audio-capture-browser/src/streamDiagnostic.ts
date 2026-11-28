const STREAM_PROTOCOL_VERSION = 2;
import type { DurabilityLevel } from "./turnJournal";
import { TARGET_SAMPLE_RATE } from "./pcm";

/**
 * Metadata-only record of one browser dictation turn.
 *
 * This deliberately has no transcript, audio bytes, URLs, or server error
 * message fields. It can therefore be rendered or exported as a user-facing
 * support bundle without turning recovery diagnostics into an audio store.
 */
export interface StreamTurnDiagnostic {
  schemaVersion: 1;
  sessionId: string;
  generation: number;
  protocolVersion: number;
  durability: DurabilityLevel;
  state: "preparing" | "recording" | "reconnecting" | "recovering" | "completed" | "failed" | "cancelled";
  capturedSequence: number;
  capturedSamples: number;
  /** Sample rate reported by the browser capture graph for this turn. */
  captureSampleRate?: number;
  /** Number of accepted capture callbacks with a valid processing-cost sample. */
  captureCallbackCount: number;
  /** Worst callback processing time divided by that callback's source-audio duration. */
  captureCallbackMaxCostRatio: number | null;
  /** Canonical stream rate used by captured/committed sample positions. */
  streamSampleRateHz: number;
  /** True once a captured frame contains non-trivial PCM amplitude. */
  signalObserved?: boolean;
  /** Metadata-only monotonic milestones used to decompose startup latency. */
  /** Timestamp when the consumer accepted the user's start action. */
  startRequestedAtMs?: number | null;
  /** First point capture graph and streaming session were both ready. */
  captureReadyAtMs?: number | null;
  /** startRequestedAtMs to captureReadyAtMs, when both were observed. */
  startToCaptureReadyMs?: number | null;
  captureStartedAtMs?: number | null;
  /** User/automatic stop action marker, before any capture settle delay. */
  stopRequestedAtMs?: number | null;
  /** Completion time of a real streaming final received after stop. */
  finalTailAtMs?: number | null;
  /** stopRequestedAtMs to finalTailAtMs; null means no qualifying final. */
  stopToFinalTailMs?: number | null;
  firstCaptureAtMs?: number | null;
  lastCaptureAtMs?: number | null;
  firstSignalAtMs?: number | null;
  /** Exact canonical sample cursor where the first above-threshold signal began. */
  firstSignalSample?: number | null;
  firstSentAtMs?: number | null;
  firstPartialAtMs?: number | null;
  /** Captured sample cursor when the first transcript revision became visible. */
  firstPartialCapturedSamples?: number | null;
  /** Post-onset audio duration already captured when the first partial became visible. */
  firstPartialAvailableAudioMs?: number | null;
  /** Wall latency minus available post-onset audio span; estimates lag beyond audio accumulation. */
  firstPartialExcessLatencyMs?: number | null;
  sentSequence: number;
  processedSequence: number;
  /** Highest end-sample position covered by committed text, when provided. */
  committedSamples: number | null;
  retainedBytes: number;
  /** Audio backlog from captured to committed coverage, in milliseconds. */
  firstPartialLatencyMs: number | null;
  /** Gaps between transcript revisions after React committed them to the DOM. */
  visibleUpdateGapsMs: number[];
  visibleRevisionCount: number;
  committedTextLagMs: number | null;
  providerId?: string;
  modelId?: string;
  providerTier?: string;
  doneSent: boolean;
  terminalReason?: string;
  statusCodes: string[];
  errorCodes: string[];
  events: StreamDiagnosticEvent[];
}

export interface StreamDiagnosticEvent {
  atMs: number;
  kind: "state" | "status" | "error" | "terminal";
  code: string;
}

/**
 * Metadata-only telemetry channel for automated product-path qualification.
 * The browser exposes only the latest bounded diagnostic snapshot; it never
 * exposes transcript text, audio bytes, URLs, or backend error payloads.
 *
 * BAS and other browser-owned harnesses read the global channel while a run is
 * active. Normal product code may subscribe without changing user-visible
 * state. Keeping this channel beside the diagnostic recorder makes the
 * machine-readable path available without making the recovery banner a test
 * control.
 */
export interface StreamDiagnosticTelemetry {
  schemaVersion: 1;
  updatedAtMs: number;
  latest: StreamTurnDiagnostic;
}

export const STREAM_DIAGNOSTIC_GLOBAL = "__VROOLI_AUDIO_STREAM_DIAGNOSTIC__" as const;

// Latency milestones are local durations/offsets. Keep them on the monotonic
// performance clock so NTP or manual wall-clock changes cannot skew latency.
function monotonicNowMs(): number {
  return typeof globalThis.performance?.now === "function" ? globalThis.performance.now() : Date.now();
}

let latestTelemetry: StreamDiagnosticTelemetry | null = null;
const telemetrySubscribers = new Set<(telemetry: StreamDiagnosticTelemetry) => void>();

export function publishStreamDiagnostic(diagnostic: StreamTurnDiagnostic): void {
  const telemetry: StreamDiagnosticTelemetry = {
    schemaVersion: 1,
    // This is an export/update timestamp, so keep its epoch semantics separate
    // from the monotonic clock used for within-turn latency measurements.
    updatedAtMs: Date.now(),
    latest: {
      ...diagnostic,
      visibleUpdateGapsMs: [...diagnostic.visibleUpdateGapsMs],
      statusCodes: [...diagnostic.statusCodes],
      errorCodes: [...diagnostic.errorCodes],
      events: diagnostic.events.map((event) => ({ ...event })),
    },
  };
  latestTelemetry = telemetry;

  // The global is intentionally a plain metadata snapshot so browser
  // automation can collect it through page.evaluate without a test-only
  // bundle or a product-side network endpoint.
  if (typeof globalThis === "object") {
    (globalThis as typeof globalThis & { [STREAM_DIAGNOSTIC_GLOBAL]?: StreamDiagnosticTelemetry })[
      STREAM_DIAGNOSTIC_GLOBAL
    ] = telemetry;
  }
  for (const subscriber of telemetrySubscribers) subscriber(telemetry);
}

export function readStreamDiagnosticTelemetry(): StreamDiagnosticTelemetry | null {
  return latestTelemetry ? {
    ...latestTelemetry,
    latest: {
      ...latestTelemetry.latest,
      visibleUpdateGapsMs: [...latestTelemetry.latest.visibleUpdateGapsMs],
      statusCodes: [...latestTelemetry.latest.statusCodes],
      errorCodes: [...latestTelemetry.latest.errorCodes],
      events: latestTelemetry.latest.events.map((event) => ({ ...event })),
    },
  } : null;
}

export function subscribeStreamDiagnosticTelemetry(
  subscriber: (telemetry: StreamDiagnosticTelemetry) => void,
): () => void {
  telemetrySubscribers.add(subscriber);
  return () => telemetrySubscribers.delete(subscriber);
}

/** Test-only reset so one browser test cannot credit another run. */
export function _resetStreamDiagnosticTelemetryForTesting(): void {
  latestTelemetry = null;
  if (typeof globalThis === "object") {
    delete (globalThis as typeof globalThis & { [STREAM_DIAGNOSTIC_GLOBAL]?: StreamDiagnosticTelemetry })[
      STREAM_DIAGNOSTIC_GLOBAL
    ];
  }
  telemetrySubscribers.clear();
}

const MAX_EVENTS = 32;
const MAX_CODES = 12;

/** A bounded in-memory recorder; persistence remains the host's policy. */
export class StreamDiagnosticRecorder {
  private snapshot: StreamTurnDiagnostic;
  private firstCaptureAtMs: number | null = null;
  private firstSignalAtMs: number | null = null;
  private committedSample: number | null = null;
  private lastVisibleRevisionAtMs: number | null = null;
  private readonly now: () => number;

  constructor(sessionId = "", generation = 0, durability: DurabilityLevel = "reduced", now = monotonicNowMs) {
    this.now = now;
    this.snapshot = {
      schemaVersion: 1,
      sessionId,
      generation,
      protocolVersion: STREAM_PROTOCOL_VERSION,
      durability,
      state: "preparing",
      capturedSequence: -1,
      capturedSamples: 0,
      captureSampleRate: undefined,
      captureCallbackCount: 0,
      captureCallbackMaxCostRatio: null,
      streamSampleRateHz: TARGET_SAMPLE_RATE,
      signalObserved: false,
      startRequestedAtMs: null,
      captureReadyAtMs: null,
      startToCaptureReadyMs: null,
      captureStartedAtMs: null,
      stopRequestedAtMs: null,
      finalTailAtMs: null,
      stopToFinalTailMs: null,
      firstCaptureAtMs: null,
      lastCaptureAtMs: null,
      firstSignalAtMs: null,
      firstSignalSample: null,
      firstSentAtMs: null,
      firstPartialAtMs: null,
      firstPartialCapturedSamples: null,
      firstPartialAvailableAudioMs: null,
      firstPartialExcessLatencyMs: null,
      sentSequence: -1,
      processedSequence: -1,
      committedSamples: null,
      retainedBytes: 0,
      firstPartialLatencyMs: null,
      visibleUpdateGapsMs: [],
      visibleRevisionCount: 0,
      committedTextLagMs: null,
      doneSent: false,
      statusCodes: [],
      errorCodes: [],
      events: [],
    };
  }

  reset(sessionId: string, generation: number, durability: DurabilityLevel): void {
    const fresh = new StreamDiagnosticRecorder(sessionId, generation, durability, this.now);
    this.snapshot = fresh.read();
    this.firstCaptureAtMs = null;
    this.firstSignalAtMs = null;
    this.committedSample = null;
    this.lastVisibleRevisionAtMs = null;
  }

  state(state: StreamTurnDiagnostic["state"], code: string = state): void {
    this.snapshot.state = state;
    this.event("state", code);
  }

  captured(sequence: bigint): void {
    this.snapshot.capturedSequence = Number(sequence);
  }

  capturedSamples(samples: bigint): void {
    this.snapshot.capturedSamples = Number(samples);
    this.updateCommittedBacklog();
  }

  captureRate(rate: number): void {
    if (Number.isFinite(rate) && rate > 0) this.snapshot.captureSampleRate = rate;
  }

  startRequested(atMs = this.now()): void {
    if (Number.isFinite(atMs) && atMs >= 0 && this.snapshot.startRequestedAtMs == null) {
      this.snapshot.startRequestedAtMs = atMs;
      this.event("state", "start_requested");
    }
  }

  captureReady(): void {
    if (this.snapshot.captureReadyAtMs != null) return;
    const now = this.now();
    this.snapshot.captureReadyAtMs = now;
    const requestedAt = this.snapshot.startRequestedAtMs;
    if (requestedAt != null) this.snapshot.startToCaptureReadyMs = Math.max(0, now - requestedAt);
    this.event("state", "capture_ready");
  }

  captureCallbackCost(processingMs: number, audioDurationMs: number): void {
    if (!Number.isFinite(processingMs) || processingMs < 0 || !Number.isFinite(audioDurationMs) || audioDurationMs <= 0) return;
    const ratio = processingMs / audioDurationMs;
    this.snapshot.captureCallbackCount += 1;
    this.snapshot.captureCallbackMaxCostRatio = Math.max(this.snapshot.captureCallbackMaxCostRatio ?? 0, ratio);
  }

  signalObserved(samplesFromSignalToCallbackEnd = 0, signalSample?: bigint): void {
    this.snapshot.signalObserved = true;
    if (this.firstSignalAtMs === null) {
      // Capture callbacks deliver a block after its samples were recorded.
      // Backdate the first threshold-crossing sample within that block so
      // leading silence in the same callback does not disappear from latency.
      const sampleOffsetMs = Number.isFinite(samplesFromSignalToCallbackEnd)
        ? Math.max(0, samplesFromSignalToCallbackEnd)
        : 0;
      this.firstSignalAtMs = this.now() - sampleOffsetMs;
      this.snapshot.firstSignalAtMs = this.firstSignalAtMs;
      if (signalSample !== undefined && signalSample <= BigInt(Number.MAX_SAFE_INTEGER)) {
        this.snapshot.firstSignalSample = Number(signalSample);
      }
    }
  }

  sent(sequence: bigint): void {
    this.snapshot.sentSequence = Math.max(this.snapshot.sentSequence, Number(sequence));
    if (this.snapshot.firstSentAtMs === null) this.snapshot.firstSentAtMs = this.now();
  }

  processed(sequence: bigint): void {
    this.snapshot.processedSequence = Math.max(this.snapshot.processedSequence, Number(sequence));
  }

  retained(bytes: number): void {
    this.snapshot.retainedBytes = Math.max(0, bytes);
  }

  durability(level: DurabilityLevel): void {
    this.snapshot.durability = level;
  }

  partial(): void {
    if (this.snapshot.firstPartialLatencyMs !== null) return;
    // Interactive latency is speech-to-visible-partial latency. Capture can
    // begin with device warm-up or leading silence, so do not substitute the
    // first PCM callback when the signal threshold was never observed.
    const speechStartedAt = this.firstSignalAtMs;
    const now = this.now();
    this.snapshot.firstPartialLatencyMs = speechStartedAt !== null
      ? Math.max(0, now - speechStartedAt)
      : null;
    this.snapshot.firstPartialAtMs = now;
    const firstSignalSample = this.snapshot.firstSignalSample;
    if (speechStartedAt !== null && typeof firstSignalSample === "number") {
      const capturedSamples = this.snapshot.capturedSamples;
      this.snapshot.firstPartialCapturedSamples = capturedSamples;
      const availableAudioMs = Math.max(0, capturedSamples - firstSignalSample)
        * 1_000 / this.snapshot.streamSampleRateHz;
      this.snapshot.firstPartialAvailableAudioMs = availableAudioMs;
      if (this.snapshot.firstPartialLatencyMs !== null) {
        this.snapshot.firstPartialExcessLatencyMs = this.snapshot.firstPartialLatencyMs - availableAudioMs;
      }
    }
  }

  /** Call only after a non-empty transcript revision has committed visibly. */
  visibleRevisionRendered(): void {
    const now = this.now();
    if (this.lastVisibleRevisionAtMs !== null) {
      const gap = now - this.lastVisibleRevisionAtMs;
      if (Number.isFinite(gap) && gap >= 0) {
        this.snapshot.visibleUpdateGapsMs.push(gap);
        // Keep the export bounded even for very long dictation turns.
        if (this.snapshot.visibleUpdateGapsMs.length > 10_000) this.snapshot.visibleUpdateGapsMs.shift();
      }
    }
    this.lastVisibleRevisionAtMs = now;
    this.snapshot.visibleRevisionCount += 1;
  }

  committed(endSample?: number): void {
    if (endSample === undefined || !Number.isSafeInteger(endSample) || endSample <= 0) return;
    this.committedSample = Math.max(this.committedSample ?? 0, endSample);
    this.snapshot.committedSamples = this.committedSample;
    this.updateCommittedBacklog();
  }

  providerIdentity(providerId?: string, modelId?: string, providerTier?: string): void {
    if (providerId) this.snapshot.providerId = providerId;
    if (modelId) this.snapshot.modelId = modelId;
    if (providerTier === "local" || providerTier === "byok" || providerTier === "vrooli") {
      this.snapshot.providerTier = providerTier;
    }
  }

  captureStarted(): void {
    this.snapshot.captureStartedAtMs = this.now();
    this.event("state", "capture_started");
  }

  stopRequested(): void {
    if (this.snapshot.stopRequestedAtMs != null) return;
    this.snapshot.stopRequestedAtMs = this.now();
    this.event("state", "stop_requested");
  }

  streamingFinalReceived(): void {
    const stoppedAtMs = this.snapshot.stopRequestedAtMs;
    if (stoppedAtMs == null || this.snapshot.finalTailAtMs != null) return;
    const now = this.now();
    this.snapshot.finalTailAtMs = now;
    this.snapshot.stopToFinalTailMs = Math.max(0, now - stoppedAtMs);
    this.event("state", "stream_final_tail");
  }

  captureObserved(): void {
    const now = this.now();
    if (this.firstCaptureAtMs === null) {
      this.firstCaptureAtMs = now;
      this.snapshot.firstCaptureAtMs = now;
    }
    this.snapshot.lastCaptureAtMs = now;
    this.event("state", "captured");
  }

  done(): void {
    this.snapshot.doneSent = true;
  }

  status(code: string): void {
    this.remember(this.snapshot.statusCodes, code);
    this.event("status", code);
  }

  error(code: string): void {
    this.remember(this.snapshot.errorCodes, code);
    this.event("error", code);
  }

  terminal(state: Extract<StreamTurnDiagnostic["state"], "completed" | "failed" | "cancelled">, reason: string): void {
    this.snapshot.state = state;
    this.snapshot.terminalReason = reason;
    this.event("terminal", reason);
  }

  read(): StreamTurnDiagnostic {
    return {
      ...this.snapshot,
      visibleUpdateGapsMs: [...this.snapshot.visibleUpdateGapsMs],
      statusCodes: [...this.snapshot.statusCodes],
      errorCodes: [...this.snapshot.errorCodes],
      events: this.snapshot.events.map((event) => ({ ...event })),
    };
  }

  /** Stable JSON export with only the allow-listed metadata above. */
  exportJSON(): string {
    return JSON.stringify(this.read(), null, 2);
  }

  private remember(target: string[], code: string): void {
    if (!code || target.includes(code)) return;
    target.push(code);
    if (target.length > MAX_CODES) target.splice(0, target.length - MAX_CODES);
  }

  private event(kind: StreamDiagnosticEvent["kind"], code: string): void {
    if (!code) return;
    this.snapshot.events.push({ atMs: this.now(), kind, code });
    if (this.snapshot.events.length > MAX_EVENTS) {
      this.snapshot.events.splice(0, this.snapshot.events.length - MAX_EVENTS);
    }
  }

  private updateCommittedBacklog(): void {
    if (this.committedSample === null) {
      this.snapshot.committedTextLagMs = null;
      return;
    }
    const uncoveredSamples = Math.max(0, this.snapshot.capturedSamples - this.committedSample);
    this.snapshot.committedTextLagMs = uncoveredSamples * 1_000 / this.snapshot.streamSampleRateHz;
  }
}
