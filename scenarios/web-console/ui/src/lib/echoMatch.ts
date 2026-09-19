import type { ConversationEvent } from "../api/conversation";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#echo-rows-and-history

/** How far apart a send and the harness's record of it may be and still match. */
export const ECHO_MATCH_WINDOW_MS = 60_000;

/**
 * A send shown in Messages before the harness records it: local, transient,
 * never persisted and never merged into events.
 */
export interface SentEcho {
  id: string;
  sessionId: string;
  text: string;
  /** Epoch ms when the send was acknowledged. */
  sentAt: number;
}

export type EchoMatchEvent = Pick<ConversationEvent, "sessionId" | "role" | "text" | "createdAt">;

/** Sent text and recorded text compared alike: trimmed, whitespace collapsed, no trailing CR. */
export function normalizeSentText(text: string): string {
  return text.replace(/\r$/u, "").replace(/\s+/gu, " ").trim();
}

/** True when a recorded user turn is this send. */
export function echoMatches(echo: SentEcho, event: EchoMatchEvent): boolean {
  if (event.role !== "user" || event.sessionId !== echo.sessionId) return false;
  if (normalizeSentText(event.text) !== normalizeSentText(echo.text)) return false;
  const recordedAt = Date.parse(event.createdAt);
  // A turn with no usable time is judged by text alone.
  return !Number.isFinite(recordedAt) || Math.abs(recordedAt - echo.sentAt) <= ECHO_MATCH_WINDOW_MS;
}
