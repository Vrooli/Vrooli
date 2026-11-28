import type { ConversationEvent } from "../../api/conversation";
import { strings } from "../../consts/strings";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#the-row-and-action-reveal

const SPEAKER_BY_SOURCE: Readonly<Record<string, string>> = {
  claude_hook: strings.messagesPane.speaker.claude,
  codex_tailer: strings.messagesPane.speaker.codex,
  codex_app_server: strings.messagesPane.speaker.codex,
  codex_voice: strings.agentVoice.voiceSpeaker,
  grok_tailer: strings.messagesPane.speaker.grok,
  opencode_api: strings.messagesPane.speaker.opencode,
};

type SpeakerShape = Pick<ConversationEvent, "role" | "source"> & Partial<Pick<ConversationEvent, "author" | "authorName">>;

/**
 * The i18n key naming who wrote a message, from its author (the role is only
 * the channel it arrived on): the person at the keyboard is "You"; the agent
 * is named by the harness whose capture recorded it, or "Agent" when unknown;
 * another agent, a program that launched the session, and the harness are
 * named as themselves, never as "You". A row from a server that predates
 * authors falls back to its role. Pair with {@link speakerParams}.
 */
export function speakerKey(event: SpeakerShape): string {
  switch (event.author ?? (event.role === "user" ? "human" : "agent")) {
    case "human":
      return strings.messagesPane.speaker.you;
    case "peer":
      return event.authorName ? strings.messagesPane.speaker.peerNamed : strings.messagesPane.speaker.peer;
    case "automation":
      return event.authorName ? strings.messagesPane.speaker.automationNamed : strings.messagesPane.speaker.automation;
    case "harness":
      return strings.messagesPane.speaker.harness;
    default:
      return SPEAKER_BY_SOURCE[event.source] ?? strings.messagesPane.speaker.agent;
  }
}

/** Interpolation for {@link speakerKey}: the author's name when the key names one. */
export function speakerParams(event: SpeakerShape): { name: string } | undefined {
  if ((event.author === "peer" || event.author === "automation") && event.authorName) return { name: event.authorName };
  return undefined;
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

/** Prefer native source time; label ingestion time when native source time is unknown. */
export function eventTime(event: Pick<ConversationEvent, "source" | "createdAt" | "occurredAt">, now: Date = new Date(), locale?: string): { timestamp: string; label: string } {
  if (event.occurredAt) return { timestamp: timeLabel(event.occurredAt, now, locale), label: "Message" };
  return {
    timestamp: timeLabel(event.createdAt, now, locale),
    label: event.source.endsWith("_archive") ? "Indexed" : "",
  };
}


/** Receipts come from capture provenance, never from matching caption text. */
export function voiceReceiptKey(event: Pick<ConversationEvent, "role" | "source">): string | null {
  if (event.role === "user" && event.source === "codex_voice_handoff") return strings.agentVoice.handoffSent;
  if (event.source === "codex_voice") return strings.agentVoice.voiceTranscript;
  return null;
}
