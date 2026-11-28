import type { ConversationEvent } from "../../api/conversation";
import { strings } from "../../consts/strings";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#notices

const NOTICE_TITLE: Readonly<Record<string, string>> = {
  interrupted: strings.messagesPane.notice.interrupted,
  usage_limit_reset: strings.messagesPane.notice.usage_limit_reset,
  task_notification: strings.messagesPane.notice.task_notification,
  subagent_notification: strings.messagesPane.notice.subagent_notification,
  slash_command: strings.messagesPane.notice.slash_command,
  command_output: strings.messagesPane.notice.command_output,
  auto_continuation: strings.messagesPane.notice.auto_continuation,
  auto_review_verdict: strings.messagesPane.notice.auto_review_verdict,
};

/** The i18n key titling a notice (what the harness told the agent), from its recogniser label. */
export function noticeTitleKey(event: Pick<ConversationEvent, "kind" | "label">): string {
  if (event.kind === "attachment") return strings.messagesPane.notice.attachment;
  return (event.label ? NOTICE_TITLE[event.label] : undefined) ?? strings.messagesPane.notice.default;
}

const SUMMARY_LIMIT = 160;

/**
 * One readable line from a notice's raw text. Harnesses inject notices as
 * tagged markup or JSON; the line is the part a person would want — the task
 * summary, the command and its arguments, the reviewer's risk level — and
 * otherwise the text with its markup removed.
 */
export function noticeSummary(event: Pick<ConversationEvent, "label" | "text" | "authorName">): string {
  const text = event.text;
  switch (event.label) {
    case "task_notification":
      return clip(tagText(text, "summary") ?? tagText(text, "status") ?? plain(text));
    case "slash_command": {
      const command = tagText(text, "command-name") ?? event.authorName ?? "";
      const args = tagText(text, "command-args") ?? "";
      return clip(`${command} ${args}`.trim() || plain(text));
    }
    case "subagent_notification": {
      const status = jsonField(text, "status");
      return clip([event.authorName ?? jsonField(text, "agent_path"), status].filter(Boolean).join(" · ") || plain(text));
    }
    case "auto_review_verdict": {
      const risk = jsonField(text, "risk_level");
      const outcome = jsonField(text, "outcome") ?? jsonField(text, "decision");
      return clip([risk, outcome].filter(Boolean).join(" · ") || plain(text));
    }
    default:
      return clip(plain(text));
  }
}

function tagText(text: string, tag: string): string | null {
  const match = new RegExp(`<${tag}>([\\s\\S]*?)</${tag}>`).exec(text);
  const value = match?.[1]?.trim();
  return value ? value : null;
}

function jsonField(text: string, field: string): string | null {
  const match = new RegExp(`"${field}"\\s*:\\s*"([^"]*)"`).exec(text);
  return match?.[1] ?? null;
}

function plain(text: string): string {
  return text.replace(/<[^>]+>/g, " ").replace(/\s+/g, " ").trim();
}

function clip(text: string): string {
  return text.length > SUMMARY_LIMIT ? `${text.slice(0, SUMMARY_LIMIT - 1)}…` : text;
}
