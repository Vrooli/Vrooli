/**
 * WebSocket JSON message protocol matching the Go backend
 * (api/terminal_ws.go). Single source of truth for message shape;
 * imported by the transport, session, and stdin-ack hooks.
 *
 * Message directions:
 *   Client → Server: stdin, resize, ping, conversation_event_ack
 *   Server → Client: stdout, exit, error, pong, session_ready,
 *                    stdin_ack, pty_state, history_end, sync_warning,
 *                    conversation_event, conversation_event_update
 *
 * [REQ:P0-002b] WebSocket I/O Streaming
 */
export interface TerminalMessage {
  type:
    | "stdin"
    | "stdout"
    | "resize"
    | "resize_info"
    | "exit"
    | "error"
    | "ping"
    | "pong"
    | "sync_warning"
    | "history_end"
    | "conversation_event"
    | "conversation_event_ack"
    | "conversation_event_update"
    | "session_ready"
    | "stdin_ack"
    | "pty_state";
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
  /** Server's monotonic output byte count (sent with "history_end"). */
  total_bytes?: number;
  /** True when the server honored the client's resume offset (delta-only). */
  resumed?: boolean;
  eventId?: string;
  source?: string;
  stage?: string;
  backend?: string;
  role?: string;
  createdAt?: string;
  sequence?: number;
  speechParagraphs?: string[];
  originalSpeechParagraphs?: string[];
  summarized?: boolean;
  /** Client-assigned sequence number for stdin messages; echoed in stdin_ack. */
  seq?: number;
  /** Per-message success flag (used by stdin_ack). */
  ok?: boolean;
  /** Alternate-screen-buffer flag on pty_state messages. */
  altBuffer?: boolean;
  /** Generation counter echoed in session_ready for the wsGen barrier. */
  gen?: number;
  /**
   * Stdin-frame discriminator. "keystroke" (default) routes through
   * `tmux send-keys -l --` on the persistent backend; "paste" routes
   * through `tmux load-buffer` + `paste-buffer -d`, which auto-cancels
   * copy-mode and atomically delivers the payload. The standard
   * backend ignores this field but still accepts it.
   */
  kind?: "keystroke" | "paste";
  /**
   * Typed error code on stdin_ack frames when ok=false. The UI maps
   * these to user-visible messages. Known values:
   *   - "tmux_write_failed"
   *   - "pty_closed"
   *   - "not_ready"
   *   - "invalid_input"
   */
  reason?: StdinAckReason;
}

/** Typed reason codes for stdin_ack.ok=false frames. */
export type StdinAckReason =
  | "tmux_write_failed"
  | "pty_closed"
  | "not_ready"
  | "invalid_input";

export interface ConversationEventMessage {
  id: string;
  source: string;
  role: "assistant" | "user";
  text: string;
  speechParagraphs?: string[];
  originalSpeechParagraphs?: string[];
  summarized?: boolean;
  createdAt?: string;
  sequence: number;
}
