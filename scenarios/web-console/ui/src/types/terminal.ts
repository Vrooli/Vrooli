/**
 * WebSocket JSON message protocol matching the Go backend
 * (api/terminal_ws.go). Single source of truth for message shape;
 * imported by the transport, session, and stdin-ack hooks.
 *
 * Message directions:
 *   Client → Server: stdin, control, resize, conversation_event_ack
 *   Server → Client: stdout, exit, error, pong, session_ready,
 *                    stdin_ack, history_end, sync_warning
 *
 * [REQ:P0-002b] WebSocket I/O Streaming
 */
export interface TerminalMessage {
  type:
    | "stdin"
    | "control"
    | "stdout"
    | "resize"
    | "resize_info"
	| "size_info"
	| "take_lease"
    | "exit"
    | "error"
    | "pong"
    | "sync_warning"
    | "history_end"
    | "conversation_event_ack"
    | "session_ready"
    | "stdin_ack";
  /** Terminal I/O payload (stdin input or stdout output). */
  data?: string;
  /** New terminal width for resize messages. */
  cols?: number;
  /** New terminal height for resize messages. */
  rows?: number;
  /** Process exit code (sent with "exit" messages). */
  code?: number;
  /** Cumulative coalesced frame count (sent with "sync_warning" messages). */
  coalesced_frames?: number;
  eventId?: string;
  source?: string;
  stage?: string;
  backend?: string;
  /** Client-assigned sequence number for stdin messages; echoed in stdin_ack. */
  seq?: number;
  /** Per-message success flag (used by stdin_ack). */
  ok?: boolean;
  /** Generation counter echoed in session_ready for the wsGen barrier. */
  gen?: number;
  /**
   * Stdin-frame discriminator. "keystroke" (default) routes through
   * `tmux send-keys -l --` on the persistent backend; "bulk_text" routes
   * through `tmux load-buffer` + `paste-buffer -d`, which auto-cancels
   * copy-mode and atomically delivers the payload. The standard
   * backend ignores this field but still accepts it.
   */
  intent?: "typing" | "bulk_text" | "named_key";
  /**
   * Typed error code on stdin_ack frames when ok=false. The UI maps
   * these to user-visible messages. Known values:
   *   - "tmux_write_failed"
   *   - "pty_closed"
   *   - "not_ready"
   */
  reason?: StdinAckReason;
	/** Per-connection size-lease state carried by size_info. */
	leader?: string;
	leaderDevice?: string;
	holdsLease?: boolean;
	viewerCount?: number;
}

/** Typed reason codes for stdin_ack.ok=false frames. */
export type StdinAckReason =
  | "tmux_write_failed"
  | "pty_closed"
  | "not_ready";
