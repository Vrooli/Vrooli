import type { ConversationEvent } from "../../api/conversation";
import { strings } from "../../consts/strings";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#the-row-and-action-reveal

const SPEAKER_BY_SOURCE: Readonly<Record<string, string>> = {
  claude_hook: strings.messagesPane.speaker.claude,
  codex_tailer: strings.messagesPane.speaker.codex,
  grok_tailer: strings.messagesPane.speaker.grok,
  opencode_api: strings.messagesPane.speaker.opencode,
};

/**
 * The i18n key naming who wrote a message: the user is "You"; an assistant is
 * named by the harness whose capture recorded it, or "Agent" when unknown.
 */
export function speakerKey(event: Pick<ConversationEvent, "role" | "source">): string {
  if (event.role === "user") return strings.messagesPane.speaker.you;
  return SPEAKER_BY_SOURCE[event.source] ?? strings.messagesPane.speaker.agent;
}

/**
 * A short time label: the clock for a message from today, a short date
 * otherwise. Empty when the timestamp cannot be parsed.
 */
export function timeLabel(createdAt: string, now: Date = new Date(), locale?: string): string {
  const at = new Date(createdAt);
  if (Number.isNaN(at.getTime())) return "";
  const sameDay = at.getFullYear() === now.getFullYear()
    && at.getMonth() === now.getMonth()
    && at.getDate() === now.getDate();
  const format = sameDay
    ? new Intl.DateTimeFormat(locale, { hour: "numeric", minute: "2-digit" })
    : new Intl.DateTimeFormat(locale, { month: "short", day: "numeric" });
  return format.format(at);
}
