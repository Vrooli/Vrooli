import type { TerminalMessage } from "../types/terminal";

/**
 * State owned by the terminal wire protocol. This is deliberately separate
 * from xterm, React state, and transport effects so reconnect and multi-viewer
 * transitions can be tested as pure data transformations.
 */
export interface TerminalProtocolState {
	/** This connection's browser-local device identity. */
	selfDeviceId: string;
	/** Device identity of the connection holding the session lease. */
	leaderDeviceId: string;
	followerMode: FollowerMode;
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
  /** Leader-declared device family; empty when the leader declared none. */
  leaderClass: string;
  /** The leader's virtual keyboard covers part of its viewport. */
  leaderKbOpen: boolean;
  viewerCount: number;
}

export type FollowerMode = "leader" | "follower" | "self-echo";

export function resolveFollowerMode(state: Pick<TerminalProtocolState, "holdsLease" | "selfDeviceId" | "leaderDeviceId">): FollowerMode {
	if (state.holdsLease) return "leader";
	if (state.selfDeviceId && state.selfDeviceId === state.leaderDeviceId) return "self-echo";
	return "follower";
}

export const initialTerminalProtocolState: TerminalProtocolState = {
	selfDeviceId: "",
	leaderDeviceId: "",
	followerMode: "leader",
  inSnapshot: true,
  outputCursor: 0,
  echo: { known: false, enabled: false, inAltBuffer: false, cursorAtLineEnd: false },
  serverSize: null,
  holdsLease: true,
  leaderDevice: "",
  leaderClass: "",
  leaderKbOpen: false,
  viewerCount: 1,
};

export function initialTerminalProtocolStateFor(selfDeviceId: string): TerminalProtocolState {
	return { ...initialTerminalProtocolState, selfDeviceId, followerMode: "leader" };
}

/**
 * size_info and presence carry the same lease and leader-presentation fields.
 * They are applied through one function so the two cases cannot drift.
 */
function applyLeaderPresentation(next: TerminalProtocolState, msg: TerminalMessage): void {
  if (typeof msg.holdsLease === "boolean") next.holdsLease = msg.holdsLease;
	if (typeof msg.leader === "string") next.leaderDeviceId = msg.leader;
  if (typeof msg.leaderDevice === "string") next.leaderDevice = msg.leaderDevice;
  if (typeof msg.deviceClass === "string") next.leaderClass = msg.deviceClass;
  if (typeof msg.kbOpen === "boolean") next.leaderKbOpen = msg.kbOpen;
  if (typeof msg.viewerCount === "number") next.viewerCount = msg.viewerCount;
}

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
	const finish = (): TerminalProtocolState => {
		next.followerMode = resolveFollowerMode(next);
		return next;
	};
	switch (msg.type) {
		case "stdout":
			return finish();
    case "history_end":
      next.inSnapshot = false;
			return finish();
    case "resync":
      next.inSnapshot = true;
			return finish();
    case "echo_state":
      next.echo = {
        known: msg.echo_known === true,
        enabled: msg.echo_enabled === true,
        inAltBuffer: msg.in_alt_buffer === true,
        cursorAtLineEnd: msg.cursor_at_line_end === true,
      };
			return finish();
    case "size_info":
    case "resize_info":
      if (typeof msg.cols === "number" && typeof msg.rows === "number") {
        next.serverSize = { cols: msg.cols, rows: msg.rows };
      }
      applyLeaderPresentation(next, msg);
			return finish();
    case "presence":
      applyLeaderPresentation(next, msg);
			return finish();
    default:
      return next;
  }
}
