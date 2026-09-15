import { STREAM_PROTOCOL_VERSION } from "./protocol";
import type { DurabilityLevel } from "./turnJournal";

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
  /** True once a captured frame contains non-trivial PCM amplitude. */
  signalObserved?: boolean;
  sentSequence: number;
  processedSequence: number;
  retainedBytes: number;
  firstPartialLatencyMs: number | null;
  committedTextLagMs: number | null;
  providerId?: string;
  modelId?: string;
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

let latestTelemetry: StreamDiagnosticTelemetry | null = null;
const telemetrySubscribers = new Set<(telemetry: StreamDiagnosticTelemetry) => void>();

export function publishStreamDiagnostic(diagnostic: StreamTurnDiagnostic): void {
  const telemetry: StreamDiagnosticTelemetry = {
    schemaVersion: 1,
    updatedAtMs: Date.now(),
    latest: {
      ...diagnostic,
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
  private firstCaptureAtMs = 0;
  private lastCaptureAtMs = 0;

  constructor(sessionId = "", generation = 0, durability: DurabilityLevel = "reduced") {
    this.snapshot = {
      schemaVersion: 1,
      sessionId,
      generation,
      protocolVersion: STREAM_PROTOCOL_VERSION,
      durability,
      state: "preparing",
      capturedSequence: -1,
      capturedSamples: 0,
      signalObserved: false,
      sentSequence: -1,
      processedSequence: -1,
      retainedBytes: 0,
      firstPartialLatencyMs: null,
      committedTextLagMs: null,
      doneSent: false,
      statusCodes: [],
      errorCodes: [],
      events: [],
    };
  }

  reset(sessionId: string, generation: number, durability: DurabilityLevel): void {
    const fresh = new StreamDiagnosticRecorder(sessionId, generation, durability);
    this.snapshot = fresh.read();
    this.firstCaptureAtMs = 0;
    this.lastCaptureAtMs = 0;
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
  }

  signalObserved(): void {
    this.snapshot.signalObserved = true;
  }

  sent(sequence: bigint): void {
    this.snapshot.sentSequence = Math.max(this.snapshot.sentSequence, Number(sequence));
  }

  processed(sequence: bigint): void {
    this.snapshot.processedSequence = Math.max(this.snapshot.processedSequence, Number(sequence));
  }

  retained(bytes: number): void {
    this.snapshot.retainedBytes = Math.max(0, bytes);
  }

  partial(): void {
    if (this.snapshot.firstPartialLatencyMs !== null) return;
    this.snapshot.firstPartialLatencyMs = this.firstCaptureAtMs > 0
      ? Math.max(0, Date.now() - this.firstCaptureAtMs)
      : null;
  }

  committed(): void {
    this.snapshot.committedTextLagMs = this.lastCaptureAtMs > 0
      ? Math.max(0, Date.now() - this.lastCaptureAtMs)
      : null;
  }

  providerIdentity(providerId?: string, modelId?: string): void {
    if (providerId) this.snapshot.providerId = providerId;
    if (modelId) this.snapshot.modelId = modelId;
  }

  captureStarted(): void {
    this.event("state", "capture_started");
  }

  captureObserved(): void {
    const now = Date.now();
    if (this.firstCaptureAtMs === 0) this.firstCaptureAtMs = now;
    this.lastCaptureAtMs = now;
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
    this.snapshot.events.push({ atMs: Date.now(), kind, code });
    if (this.snapshot.events.length > MAX_EVENTS) {
      this.snapshot.events.splice(0, this.snapshot.events.length - MAX_EVENTS);
    }
  }
}
