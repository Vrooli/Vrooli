import type { TerminalMessage } from "../types/terminal";

/**
 * State owned by the terminal wire protocol. This is deliberately separate
 * from xterm, React state, and transport effects so reconnect and multi-viewer
 * transitions can be tested as pure data transformations.
 */
export interface TerminalProtocolState {
  inSnapshot: boolean;
  outputCursor: number;
  echo: {
    known: boolean;
    enabled: boolean;
    inAltBuffer: boolean;
    cursorAtLineEnd: boolean;
  };
  serverSize: { cols: number; rows: number } | null;
  holdsLease: boolean;
  leaderDevice: string;
  viewerCount: number;
}

export const initialTerminalProtocolState: TerminalProtocolState = {
  inSnapshot: true,
  outputCursor: 0,
  echo: { known: false, enabled: false, inAltBuffer: false, cursorAtLineEnd: false },
  serverSize: null,
  holdsLease: true,
  leaderDevice: "",
  viewerCount: 1,
};

/** Pure wire-state transition. Rendering and transport effects stay outside. */
export function reduceTerminalMessage(
  state: TerminalProtocolState,
  msg: TerminalMessage,
): TerminalProtocolState {
  const next = {
    ...state,
    echo: { ...state.echo },
    serverSize: state.serverSize && { ...state.serverSize },
  };
  if (typeof msg.output_cursor === "number") next.outputCursor = msg.output_cursor;
  switch (msg.type) {
    case "stdout":
      return next;
    case "history_end":
      next.inSnapshot = false;
      return next;
    case "resync":
      next.inSnapshot = true;
      return next;
    case "echo_state":
      next.echo = {
        known: msg.echo_known === true,
        enabled: msg.echo_enabled === true,
        inAltBuffer: msg.in_alt_buffer === true,
        cursorAtLineEnd: msg.cursor_at_line_end === true,
      };
      return next;
    case "size_info":
    case "resize_info":
      if (typeof msg.cols === "number" && typeof msg.rows === "number") {
        next.serverSize = { cols: msg.cols, rows: msg.rows };
      }
      if (typeof msg.holdsLease === "boolean") next.holdsLease = msg.holdsLease;
      if (typeof msg.leaderDevice === "string") next.leaderDevice = msg.leaderDevice;
      if (typeof msg.viewerCount === "number") next.viewerCount = msg.viewerCount;
      return next;
    case "presence":
      if (typeof msg.holdsLease === "boolean") next.holdsLease = msg.holdsLease;
      if (typeof msg.leaderDevice === "string") next.leaderDevice = msg.leaderDevice;
      if (typeof msg.viewerCount === "number") next.viewerCount = msg.viewerCount;
      return next;
    default:
      return next;
  }
}
