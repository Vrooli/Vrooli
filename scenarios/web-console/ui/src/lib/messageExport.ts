import type { ConversationEvent } from "../api/conversation";
import { strings } from "../consts/strings";

/**
 * Message export model — pure, deterministic formatting of an explicit set of
 * conversation events into paste-ready coding-agent context. React-free by
 * design: the navigator and drawer both consume this module so ordering,
 * escaping, and token estimation live in exactly one tested place.
 *
 * Contract: only the given messages are exported, once each, in ascending
 * `sequence` order with their original roles. No purpose wrapper, timestamps,
 * summaries, or auto-included neighbors.
 */

export type MessageExportFormat = "agentXml" | "markdown" | "quote" | "plain";

export interface MessageExportFormatDescriptor {
  id: MessageExportFormat;
  /** i18n key for the format's short label. */
  labelKey: string;
  /** i18n key for the one-line description shown in the format picker. */
  descriptionKey: string;
}

/** Picker order. The first entry is the default (Agent XML). */
export const MESSAGE_EXPORT_FORMATS = [
  {
    id: "agentXml",
    labelKey: strings.messageExport.formatAgentXmlLabel,
    descriptionKey: strings.messageExport.formatAgentXmlDescription,
  },
  {
    id: "markdown",
    labelKey: strings.messageExport.formatMarkdownLabel,
    descriptionKey: strings.messageExport.formatMarkdownDescription,
  },
  {
    id: "quote",
    labelKey: strings.messageExport.formatQuoteLabel,
    descriptionKey: strings.messageExport.formatQuoteDescription,
  },
  {
    id: "plain",
    labelKey: strings.messageExport.formatPlainLabel,
    descriptionKey: strings.messageExport.formatPlainDescription,
  },
] as const satisfies readonly MessageExportFormatDescriptor[];

export const DEFAULT_MESSAGE_EXPORT_FORMAT: MessageExportFormat = "agentXml";

export interface MessageExportResult {
  format: MessageExportFormat;
  /** Rendered export text. Empty string when no messages are selected. */
  text: string;
  /** Distinct messages included after deduplication. */
  messageCount: number;
  /** Approximate token count of `text` (see estimateTokens). */
  tokenEstimate: number;
}

/**
 * Approximates the token count of rendered text as ceil(length / 4) — the
 * common ~4-characters-per-token heuristic for English/code. The same
 * estimator backs the selection footer and the drawer preview so both always
 * report the same number for the same text.
 */
export function estimateTokens(text: string): number {
  if (text.length === 0) return 0;
  return Math.ceil(text.length / 4);
}

/**
 * Normalizes an explicit selection: drops duplicate ids (first occurrence
 * wins), then orders strictly by ascending `sequence` regardless of the order
 * the navigator handed them over in. Never mutates the input array.
 */
export function normalizeExportEvents(events: readonly ConversationEvent[]): ConversationEvent[] {
  const seen = new Set<string>();
  const unique: ConversationEvent[] = [];
  for (const event of events) {
    if (seen.has(event.id)) continue;
    seen.add(event.id);
    unique.push(event);
  }
  unique.sort((a, b) => a.sequence - b.sequence);
  return unique;
}

function escapeXml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function renderAgentXml(events: ConversationEvent[]): string {
  const messages = events.map(
    (event) =>
      `  <message number="${event.sequence}" role="${escapeXml(event.role)}">\n` +
      `${escapeXml(event.text)}\n` +
      `  </message>`,
  );
  return `<conversation>\n${messages.join("\n")}\n</conversation>`;
}

function renderMarkdown(events: ConversationEvent[]): string {
  return events
    .map((event) => `**#${event.sequence} · ${event.role}**\n\n${event.text}`)
    .join("\n\n---\n\n");
}

function renderQuote(events: ConversationEvent[]): string {
  return events
    .map((event) => {
      const quoted = event.text
        .split("\n")
        .map((line) => (line.length > 0 ? `> ${line}` : ">"))
        .join("\n");
      return `**#${event.sequence} · ${event.role}**\n${quoted}`;
    })
    .join("\n\n");
}

function renderPlain(events: ConversationEvent[]): string {
  return events
    .map((event) => `[#${event.sequence}] ${event.role}:\n${event.text}`)
    .join("\n\n");
}

/**
 * Renders the selected events in the requested format. An empty selection
 * yields an empty result (no wrapper markup) so callers can disable copy.
 */
export function buildMessageExport(
  events: readonly ConversationEvent[],
  format: MessageExportFormat,
): MessageExportResult {
  const ordered = normalizeExportEvents(events);
  if (ordered.length === 0) {
    return { format, text: "", messageCount: 0, tokenEstimate: 0 };
  }

  let text: string;
  switch (format) {
    case "agentXml":
      text = renderAgentXml(ordered);
      break;
    case "markdown":
      text = renderMarkdown(ordered);
      break;
    case "quote":
      text = renderQuote(ordered);
      break;
    case "plain":
      text = renderPlain(ordered);
      break;
  }

  return {
    format,
    text,
    messageCount: ordered.length,
    tokenEstimate: estimateTokens(text),
  };
}
