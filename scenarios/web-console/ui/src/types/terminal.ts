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
    | "stdin_ack"
    | "hello"
    | "resync"
    | "snapshot_notice"
    | "echo_state"
    | "mouse_mode"
    | "scroll"
    | "presence"
    | "device_state";

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
	/** UTF-8 byte offset of a reliable stdin frame. */
  offset?: number;
  /** Highest contiguous reliable stdin byte offset accepted by the server. */
  accepted_through?: number;
  /** Highest reliable stdin byte offset known by the reconnecting client. */
  have_through?: number;
	/** Highest PTY-output cursor already rendered by this client. */
	rendered_through?: number;
	/** Requests cursor-based output replay during reconnect. */
	want_resume?: boolean;
	/** End cursor of a stdout/history replay frame. */
	output_cursor?: number;
  /** Per-message success flag (used by stdin_ack). */
  ok?: boolean;
  /** Generation counter echoed in session_ready for the wsGen barrier. */
  gen?: number;
  /** Version of the terminal WebSocket contract. */
  protocol_version?: string;
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
	 *   - "offset_gap"
	 *   - "unreconcilable"
   */
  reason?: StdinAckReason;
	/** Per-connection size-lease state carried by size_info. */
	leader?: string;
	leaderDevice?: string;
	/**
	 * Leader-declared device family, used only to choose a follower's
	 * decorative frame. Never an authorization signal, and never a hardware
	 * identity claim — an operator can edit it.
	 */
	deviceClass?: string;
	/**
	 * The leader's virtual keyboard covers part of its viewport. Followers
	 * draw this rather than inferring it from the grid, which shrinks for many
	 * reasons besides a keyboard.
	 */
	kbOpen?: boolean;
	holdsLease?: boolean;
	viewerCount?: number;
	/** Server-owned predictive-input authorization. Unknown is fail-closed. */
	echo_known?: boolean;
	echo_enabled?: boolean;
	in_alt_buffer?: boolean;
	cursor_at_line_end?: boolean;
	/** Persistent-pane tmux mouse capture state, when the backend supports it. */
	mouse_mode?: boolean;
	mouse_mode_known?: boolean;
	/**
	 * Scroll request in terminal rows: negative scrolls back toward older
	 * output, positive scrolls forward toward live output. Only meaningful on
	 * a `scroll` frame.
	 */
	lines?: number;
}

/** Typed reason codes for stdin_ack.ok=false frames. */
export type StdinAckReason =
  | "tmux_write_failed"
  | "pty_closed"
  | "offset_gap"
  | "unreconcilable"
  | "input_queue_full";
