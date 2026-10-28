import { StagedMessageScope, type StagedMessageSuggestion } from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";

export type DraftScope = "whole" | "limited" | "legacy_partial" | "legacy_complete" | "unknown";

/** Legacy coverage described the message. New scope describes the synthesis;
 * its coverage counts only detailed evidence and cannot narrow that scope. */
export function getDraftScope(suggestion: StagedMessageSuggestion): DraftScope {
  switch (suggestion.messageScope ?? StagedMessageScope.UNSPECIFIED) {
    case StagedMessageScope.WHOLE_SELECTION: return "whole";
    case StagedMessageScope.LIMITED: return "limited";
    case StagedMessageScope.UNSPECIFIED: return suggestion.coverage?.partial ? "legacy_partial" : "legacy_complete";
    default: return "unknown";
  }
}

export function isLimitedDraft(suggestion?: StagedMessageSuggestion): boolean {
  if (!suggestion) return false;
  const scope = getDraftScope(suggestion);
  return scope === "limited" || scope === "legacy_partial";
}

/** The ordinary composer edits one Git message. Unknown/folded trailer lines
 * stay verbatim; references describe context and never establish authorship. */
export interface MessageFields {
  title: string;
  description: string;
  trailers: string;
}

function isTrailerParagraph(lines: string[]): boolean {
  let trailers = 0;
  let other = 0;
  let recognized = false;
  let previous = false;
  for (const line of lines) {
    if (/^[ \t]/.test(line) && previous) continue;
    recognized ||= line.startsWith("Signed-off-by: ") || line.startsWith("(cherry picked from commit ");
    if (/^[a-zA-Z0-9-]+[ \t]*:/.test(line)) trailers++;
    else other++;
    previous = true;
  }
  return trailers > 0 && (other === 0 || (recognized && trailers * 3 >= other));
}

export function parseMessage(value: string): MessageFields {
  const lines = value.replace(/\r\n/g, "\n").trimEnd().split("\n");
  const title = lines.shift() ?? "";
  let rest = lines.join("\n").replace(/^\n+/, "");
  let trailers = "";
  const paragraphs = rest.split(/\n[ \t]*\n/);
  const last = paragraphs[paragraphs.length - 1] ?? "";
  // Git never interprets the title paragraph as trailers. The first empty
  // line is required even when there is no description.
  if (value.replace(/\r\n/g, "\n").includes("\n\n") && isTrailerParagraph(last.split("\n"))) {
    trailers = last;
    paragraphs.pop();
    rest = paragraphs.join("\n\n");
  }
  return { title, description: rest.trim(), trailers };
}

export function serializeMessage(fields: MessageFields): string {
  return [fields.title.trim(), ...[fields.description.trim(), fields.trailers.trim()].filter(Boolean)]
    .join("\n\n");
}

function parseTrailerValues(trailers: string): { key: string; value: string }[] {
  const lines = trailers ? trailers.split("\n") : [];
  const values: { key: string; value: string }[] = [];
  for (const line of lines) {
    const last = values[values.length - 1];
    if (/^[ \t]/.test(line) && last) {
      last.value += ` ${line.trim()}`;
      continue;
    }
    const parsed = /^([a-zA-Z0-9-]+)[ \t]*:[ \t]*(.*)$/.exec(line);
    if (parsed) values.push({ key: parsed[1] ?? "", value: parsed[2] ?? "" });
  }
  return values;
}

/** A history selection adds references; existing editor text keeps priority. */
export function mergeContinuationMessage(current: string, continuation: string): string {
  if (!current.trim()) return continuation;
  const fields = parseMessage(current);
  const merged = mergeSuggestionTrailers(fields.trailers, parseTrailerValues(parseMessage(continuation).trailers));
  const added = merged.slice(fields.trailers.length).replace(/^\n/, "");
  if (!added) return current;
  const newline = current.includes("\r\n") ? "\r\n" : "\n";
  const references = added.replace(/\n/g, newline);
  if (!fields.trailers) return `${current}${newline}${newline}${references}`;
  // Insert within the existing trailer paragraph, before its trailing space.
  // Never reserialize the title/body: Markdown indentation is meaningful.
  const end = current.trimEnd().length;
  return `${current.slice(0, end)}${newline}${references}${current.slice(end)}`;
}

export function mergeSuggestionTrailers(current: string, suggested: { key: string; value: string }[]): string {
  const lines = current ? current.split("\n") : [];
  const identity = (key: string, value: string) => `${key.trim().toLowerCase()}\u0000${value.trim()}`;
  const values = parseTrailerValues(current);
  const identities = new Set(values.map(entry => identity(entry.key, entry.value)));
  for (const entry of suggested) {
    const line = `${entry.key}: ${entry.value}`;
    const key = identity(entry.key, entry.value);
    if (!identities.has(key)) {
      lines.push(line);
      identities.add(key);
    }
  }
  return lines.join("\n");
}
