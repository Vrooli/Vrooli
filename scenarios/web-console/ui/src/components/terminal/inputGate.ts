import type { Terminal } from "@xterm/xterm";

/**
 * TerminalInputGate is the single authority that decides whether an
 * input payload is sent to the PTY, queued for later, or rejected.
 * Every caller that would otherwise call `ws.send({type: "stdin"})`
 * directly routes through this gate instead.
 *
 * Three-state result:
 *  - sent: payload was handed to the WebSocket stack (seq included).
 *  - queued: payload is in the pending queue; will be sent on resolve.
 *  - rejected: the gate refused the payload (empty, disposed).
 *
 * The gate is mode-aware: bulk text is held back
 * when xterm is in a mouse-tracking mode that would misinterpret bytes
 * as mouse events. This prevents the "paste disappears into Claude
 * Code" class of bugs where paste bytes silently became unintended
 * keystrokes inside a TUI.
 *
 * The gate does NOT own transport state; it consults read-only
 * accessors supplied by the session hook. This keeps the gate a pure
 * decision layer and leaves transport concerns to the hook.
 *
 */
export const INPUT_INTENTS = ["typing", "bulk_text", "named_key", "control"] as const;
export type InputIntent = (typeof INPUT_INTENTS)[number];

export type GateResult =
  | { status: "sent"; seq: number }
  | { status: "queued"; reason: QueuedReason }
  | { status: "rejected"; reason: RejectedReason };

export type QueuedReason = "not-ready" | "ws-closed" | "paused";
export type RejectedReason = "empty" | "disposed";

export interface RawSendResult {
  /** True if the payload was accepted by the WebSocket stack. */
  sent: boolean;
  /** Present iff sent === true. */
  seq?: number;
  /** Present iff sent === false; hints at the reason. */
  reason?: QueuedReason;
}

/**
 * The semantic intent carried through the reliable stdin lane. Control is
 * represented by the separate control frame and is not accepted by the gate.
 */

/**
 * Transport seam consumed by the gate. Implementations must:
 *  - return {sent: true, seq} when the frame reached the WebSocket.
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
  /** Read-only accessor for the live xterm instance. Can return null. */
  getTerminal: () => Terminal | null;
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
  /** Pure accessor used by tests and observability. */
  canAcceptPaste(): boolean;
}

/**
 * Returns true when xterm is in any mouse-tracking mode that would
 * interpret bulk text bytes as mouse events rather than text. Only
 * bulk text input is blocked — individual keystrokes from
 * xterm.onData already pass through the terminal's own event handling
 * and are not affected.
 */
export function terminalIsInMouseTrackingMode(t: Terminal | null): boolean {
  if (!t) return false;
  const modes = t.modes;
  if (!modes) return false;
  return modes.mouseTrackingMode !== "none";
}

export function createInputGate(opts: InputGateOptions): TerminalInputGate {
  let disposed = false;

  const canAcceptPaste = (): boolean => !terminalIsInMouseTrackingMode(opts.getTerminal());

  const submit = (data: string, intent: Exclude<InputIntent, "control">): GateResult => {
    if (disposed) return { status: "rejected", reason: "disposed" };
    if (!data) return { status: "rejected", reason: "empty" };

    // External pause (voice mode etc.) blocks every source uniformly.
    if (opts.isPaused?.() === true) {
      opts.transport.enqueue(data, intent);
      return { status: "queued", reason: "paused" };
    }

    // Bulk-text-specific client-side mode gating: mouse-tracking TUIs
    // running INSIDE xterm consume bytes as mouse events at the
    // browser layer (before the WS frame is sent). Hold bulk-text
    // payloads until the TUI exits that mode. Typing and named keys
    // are one event at a time and
    // don't trigger the same misinterpretation. Tmux-side modes
    // (copy-mode, command-prompt, menu) are handled server-side via
    // paste-buffer and need no gating here.
    if (intent === "bulk_text" && terminalIsInMouseTrackingMode(opts.getTerminal())) {
      opts.transport.enqueue(data, intent);
      return { status: "queued", reason: "paused" };
    }

    const res = opts.transport.send(data, intent);
    if (res.sent && typeof res.seq === "number") {
      return { status: "sent", seq: res.seq };
    }
    opts.transport.enqueue(data, intent);
    return { status: "queued", reason: res.reason ?? "not-ready" };
  };

  const dispose = (): void => {
    disposed = true;
  };

  return { submit, dispose, canAcceptPaste };
}
