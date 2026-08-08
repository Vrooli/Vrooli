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
  sentSequence: number;
  processedSequence: number;
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

const MAX_EVENTS = 32;
const MAX_CODES = 12;

/** A bounded in-memory recorder; persistence remains the host's policy. */
export class StreamDiagnosticRecorder {
  private snapshot: StreamTurnDiagnostic;

  constructor(sessionId = "", generation = 0, durability: DurabilityLevel = "reduced") {
    this.snapshot = {
      schemaVersion: 1,
      sessionId,
      generation,
      protocolVersion: STREAM_PROTOCOL_VERSION,
      durability,
      state: "preparing",
      capturedSequence: -1,
      sentSequence: -1,
      processedSequence: -1,
      doneSent: false,
      statusCodes: [],
      errorCodes: [],
      events: [],
    };
  }

  reset(sessionId: string, generation: number, durability: DurabilityLevel): void {
    this.snapshot = new StreamDiagnosticRecorder(sessionId, generation, durability).read();
  }

  state(state: StreamTurnDiagnostic["state"], code: string = state): void {
    this.snapshot.state = state;
    this.event("state", code);
  }

  captured(sequence: bigint): void {
    this.snapshot.capturedSequence = Number(sequence);
  }

  sent(sequence: bigint): void {
    this.snapshot.sentSequence = Math.max(this.snapshot.sentSequence, Number(sequence));
  }

  processed(sequence: bigint): void {
    this.snapshot.processedSequence = Math.max(this.snapshot.processedSequence, Number(sequence));
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
