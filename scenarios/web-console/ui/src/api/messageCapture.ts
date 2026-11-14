/**
 * The message-capture contract, kept in its own module deliberately.
 *
 * The store needs the `UNKNOWN_CAPTURE` value, not just the types, and the
 * conversation API client is mocked by a large number of tests. Importing a
 * runtime value from that client into the store would make every one of those
 * mocks responsible for re-exporting it. This module has no dependencies and
 * nothing worth mocking, so both sides can import it freely.
 *
 * Mirrors vrooli.web_console.v1.conversation.MessageCaptureStatus.
 */

/**
 * How the server describes its ability to record a session's messages.
 *
 * This exists because an empty event list is ambiguous on its own: a brand-new
 * session and a session whose transcript can never be read produce exactly the
 * same response. Callers must branch on the state, never on `events.length`.
 */
export type MessageCaptureState =
  | "capturing"
  | "not_applicable"
  | "pending"
  | "unavailable"
  | "unknown";

export interface MessageCaptureStatus {
  state: MessageCaptureState;
  /** Stable cause identifier, e.g. "hook_not_registered". Empty when healthy. */
  reasonCode: string;
  /** One sentence written for the person reading the Messages view. */
  summary: string;
  /** Operator-facing specifics — a path, a hook status. Shown on demand. */
  detail: string;
  /** The action that would fix this, when one exists. */
  remediation: string;
  transcriptPath: string;
  /** RFC3339 timestamp of the most recent captured message, or empty. */
  lastCapturedAt: string;
}

/**
 * The state before the server has told us anything. It is deliberately not
 * "capturing": assuming health we have not been told about is how an empty
 * pane came to claim there was simply nothing to show.
 */
export const UNKNOWN_CAPTURE: MessageCaptureStatus = {
  state: "unknown",
  reasonCode: "",
  summary: "",
  detail: "",
  remediation: "",
  transcriptPath: "",
  lastCapturedAt: "",
};

/**
 * The wire enum is numeric; map it explicitly rather than indexing an array so
 * a value this client predates degrades to "unknown" instead of undefined.
 */
const CAPTURE_STATE_BY_WIRE: Record<number, MessageCaptureState> = {
  0: "unknown",
  1: "capturing",
  2: "not_applicable",
  3: "pending",
  4: "unavailable",
};

export interface ProtoMessageCaptureStatus {
  state: number;
  reasonCode: string;
  summary: string;
  detail: string;
  remediation: string;
  transcriptPath: string;
  lastCapturedAt: string;
}

export function decodeCaptureStatus(c: ProtoMessageCaptureStatus | undefined): MessageCaptureStatus {
  if (!c) return UNKNOWN_CAPTURE;
  return {
    state: CAPTURE_STATE_BY_WIRE[c.state] ?? "unknown",
    reasonCode: c.reasonCode,
    summary: c.summary,
    detail: c.detail,
    remediation: c.remediation,
    transcriptPath: c.transcriptPath,
    lastCapturedAt: c.lastCapturedAt,
  };
}
