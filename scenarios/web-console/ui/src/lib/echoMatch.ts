import type { ConversationEvent } from "../api/conversation";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#echo-rows-and-history

/** How far apart a send and the harness's record of it may be and still match. */
export const ECHO_MATCH_WINDOW_MS = 60_000;

/**
 * An outbox entry's lifecycle, as far as the console can tell.
 *
 *  - "staged"    — typed into the agent's input but not yet submitted. For a
 *                  coding agent this is the text sitting at the prompt waiting
 *                  for an Enter; it is the default after a send is acknowledged.
 *  - "submitted" — the user submitted it (pressed Enter) while the agent was
 *                  free, or the agent was observed to start working right after
 *                  it was staged. It is on its way into the turn.
 *  - "queued"    — submitted while the agent was already working, so it waits
 *                  behind the current turn instead of starting one.
 *
 * Entries leave the outbox when the harness records the matching user turn (see
 * {@link echoMatches}) or, failing that, when they expire.
 */
export type OutboxState = "staged" | "submitted" | "queued";

/**
 * A send shown in Messages before the harness records it: local, transient,
 * never persisted and never merged into events. One entry can represent several
 * sends that were typed into the same unsubmitted input (they are coalesced),
 * which is why `text`/`payload` can carry newlines the user entered as separate
 * sends.
 */
export interface SentEcho {
  id: string;
  sessionId: string;
  /** What the user sees as the message — attachment paths excluded. */
  text: string;
  /** The exact bytes sent to the terminal, including resolved attachment paths. */
  payload: string;
  /** How many attachments the payload carries, for a "+N attachment" hint. */
  attachmentCount: number;
  /** Epoch ms of the most recent send folded into this entry. */
  sentAt: number;
  state: OutboxState;
}

export type EchoMatchEvent = Pick<ConversationEvent, "sessionId" | "role" | "text" | "createdAt">;

/** Sent text and recorded text compared alike: trimmed, whitespace collapsed, no trailing CR. */
export function normalizeSentText(text: string): string {
  return text.replace(/\r$/u, "").replace(/\s+/gu, " ").trim();
}

/** Short strings must match exactly; longer ones may match as a substring either way. */
const SUBSTRING_MATCH_FLOOR = 8;

/**
 * True when a recorded turn `candidate` is (plausibly) this send `sent`.
 *
 * Attachments are why this is forgiving rather than an equality check: the
 * console sends the resolved file paths, but the harness records the message as
 * the agent rendered it (a chip, "[Image #1]", "+1 attachment"), so the two
 * texts never match verbatim once a file is attached. A long send is accepted
 * when either string contains the other; a short one must match exactly so a
 * stray "y" or "ls" cannot sweep up an unrelated turn.
 */
function textsAlign(candidate: string, sent: string): boolean {
  if (sent === "") return false;
  if (candidate === sent) return true;
  if (sent.length < SUBSTRING_MATCH_FLOOR) return false;
  return candidate.includes(sent) || sent.includes(candidate);
}

/** True when a recorded user turn is this send. */
export function echoMatches(echo: SentEcho, event: EchoMatchEvent): boolean {
  if (event.role !== "user" || event.sessionId !== echo.sessionId) return false;
  const recorded = normalizeSentText(event.text);
  // Match on the user text first, then the full payload (paths and all): a send
  // with no attachments has payload === text, so this costs nothing there.
  if (!textsAlign(recorded, normalizeSentText(echo.text)) && !textsAlign(recorded, normalizeSentText(echo.payload))) {
    return false;
  }
  const recordedAt = Date.parse(event.createdAt);
  // A turn with no usable time is judged by text alone.
  return !Number.isFinite(recordedAt) || Math.abs(recordedAt - echo.sentAt) <= ECHO_MATCH_WINDOW_MS;
}
