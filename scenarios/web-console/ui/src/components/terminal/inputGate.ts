/**
 * TerminalInputGate is the single authority that decides whether an
 * input payload is sent to the PTY, queued for later, or rejected.
 * Every caller that would otherwise call `ws.send({type: "stdin"})`
 * directly routes through this gate instead.
 *
 * Three-state result:
 *  - sent: payload was handed to the WebSocket stack (end offset included).
 *  - queued: payload is in the pending queue; will be sent on resolve.
 *  - rejected: the gate refused the payload (empty, disposed).
 *
 * The gate does NOT own transport state; it consults read-only
 * accessors supplied by the session hook. This keeps the gate a pure
 * decision layer and leaves transport concerns to the hook.
 *
 */
export const INPUT_INTENTS = ["typing", "bulk_text", "named_key", "control"] as const;
export type InputIntent = (typeof INPUT_INTENTS)[number];

export type GateResult =
  | { status: "sent"; offset: number }
  | { status: "queued"; reason: QueuedReason }
  | { status: "rejected"; reason: RejectedReason };

export type QueuedReason = "not-ready" | "ws-closed" | "paused";
export type RejectedReason = "empty" | "disposed";

export interface RawSendResult {
  /** True if the payload was accepted by the WebSocket stack. */
  sent: boolean;
  /** Present iff sent === true. */
  offset?: number;
  /** Present iff sent === false; hints at the reason. */
  reason?: QueuedReason;
}

/**
 * The semantic intent carried through the reliable stdin lane. Control is
 * represented by the separate control frame and is not accepted by the gate.
 */

/**
 * Transport seam consumed by the gate. Implementations must:
 *  - return {sent: true, offset} when the frame reached the WebSocket.
 *  - return {sent: false, reason} when the frame was not sent. The
 *    reason is exposed to callers unchanged; common values:
 *    "not-ready" (session_ready not yet received), "ws-closed"
 *    (WebSocket not OPEN or back-pressure high-water breached).
 *
 * The intent argument is load-bearing for the persistent (tmux) backend
 * and cosmetic for the standard backend.
 */
export interface GateTransport {
  send(data: string, intent: Exclude<InputIntent, "control">): RawSendResult;
  enqueue(data: string, intent: Exclude<InputIntent, "control">): void;
}

/** Options for createInputGate. */
export interface InputGateOptions {
  transport: GateTransport;
  /** Optional external paused flag (e.g. tests or higher layers). */
  isPaused?: () => boolean;
}

export interface TerminalInputGate {
  submit(data: string, intent: Exclude<InputIntent, "control">): GateResult;
  /**
   * Dispose the gate. After dispose, every submit returns
   * {status: "rejected", reason: "disposed"}.
   */
  dispose(): void;
}

export function createInputGate(opts: InputGateOptions): TerminalInputGate {
  let disposed = false;

  const submit = (data: string, intent: Exclude<InputIntent, "control">): GateResult => {
    if (disposed) return { status: "rejected", reason: "disposed" };
    if (!data) return { status: "rejected", reason: "empty" };

    // External pause (voice mode etc.) blocks every source uniformly.
    if (opts.isPaused?.() === true) {
      opts.transport.enqueue(data, intent);
      return { status: "queued", reason: "paused" };
    }

    const res = opts.transport.send(data, intent);
    if (res.sent && typeof res.offset === "number") {
      return { status: "sent", offset: res.offset };
    }
    opts.transport.enqueue(data, intent);
    return { status: "queued", reason: res.reason ?? "not-ready" };
  };

  const dispose = (): void => {
    disposed = true;
  };

  return { submit, dispose };
}
