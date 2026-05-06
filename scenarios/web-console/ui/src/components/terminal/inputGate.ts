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
 * The gate is mode-aware: certain input sources (pastes) are held back
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
export type InputSource =
  | "xterm"
  | "toolbar-key"
  | "toolbar-submit"
  | "paste"
  | "voice"
  | "upload";

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
 * Wire-level input-kind discriminator. Mirrors the server's StdinKind*.
 * "keystroke" is the default for ordinary input (xterm keystrokes,
 * toolbar keys, voice); "paste" routes through the persistent
 * backend's mode-safe paste-buffer path.
 */
export type WireInputKind = "keystroke" | "paste";

/**
 * Transport seam consumed by the gate. Implementations must:
 *  - return {sent: true, seq} when the frame reached the WebSocket.
 *  - return {sent: false, reason} when the frame was not sent. The
 *    reason is exposed to callers unchanged; common values:
 *    "not-ready" (session_ready not yet received), "ws-closed"
 *    (WebSocket not OPEN or back-pressure high-water breached).
 *
 * The kind argument is load-bearing for the persistent (tmux) backend
 * and cosmetic for the standard backend.
 */
export interface GateTransport {
  send(data: string, kind: WireInputKind): RawSendResult;
  enqueue(data: string, kind: WireInputKind): void;
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
  submit(data: string, source: InputSource): GateResult;
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
 * interpret pasted bytes as mouse events rather than text. Only
 * pasted multi-byte input is blocked — individual keystrokes from
 * xterm.onData already pass through the terminal's own event handling
 * and are not affected.
 */
export function terminalIsInMouseTrackingMode(t: Terminal | null): boolean {
  if (!t) return false;
  const modes = t.modes;
  if (!modes) return false;
  return modes.mouseTrackingMode !== "none";
}

/**
 * wireKindFor maps the UI-level input source to the wire-level kind
 * discriminator. Only "paste" goes on the paste path; everything else
 * (xterm typing, toolbar keys/submit, voice, upload-triggered stdin)
 * is delivered as keystrokes.
 */
export function wireKindFor(source: InputSource): WireInputKind {
  return source === "paste" ? "paste" : "keystroke";
}

export function createInputGate(opts: InputGateOptions): TerminalInputGate {
  let disposed = false;

  const canAcceptPaste = (): boolean => !terminalIsInMouseTrackingMode(opts.getTerminal());

  const submit = (data: string, source: InputSource): GateResult => {
    if (disposed) return { status: "rejected", reason: "disposed" };
    if (!data) return { status: "rejected", reason: "empty" };

    const kind = wireKindFor(source);

    // External pause (voice mode etc.) blocks every source uniformly.
    if (opts.isPaused?.() === true) {
      opts.transport.enqueue(data, kind);
      return { status: "queued", reason: "paused" };
    }

    // Paste-specific client-side mode gating: mouse-tracking TUIs
    // running INSIDE xterm consume bytes as mouse events at the
    // browser layer (before the WS frame is sent). Hold paste
    // payloads until the TUI exits that mode. Other sources
    // (keystrokes, toolbar keys, voice) are one byte at a time and
    // don't trigger the same misinterpretation. Tmux-side modes
    // (copy-mode, command-prompt, menu) are handled server-side via
    // paste-buffer and need no gating here.
    if (source === "paste" && terminalIsInMouseTrackingMode(opts.getTerminal())) {
      opts.transport.enqueue(data, kind);
      return { status: "queued", reason: "paused" };
    }

    const res = opts.transport.send(data, kind);
    if (res.sent && typeof res.seq === "number") {
      return { status: "sent", seq: res.seq };
    }
    opts.transport.enqueue(data, kind);
    return { status: "queued", reason: res.reason ?? "not-ready" };
  };

  const dispose = (): void => {
    disposed = true;
  };

  return { submit, dispose, canAcceptPaste };
}
