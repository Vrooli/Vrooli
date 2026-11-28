import type { ConversationEvent, ConversationKind, ConversationThreadSummary, ConversationWorkSpan } from "../../api/conversation";
import { strings } from "../../consts/strings";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#work-between-messages

/** The translate function rows already hold, narrowed to what these helpers need. */
export type Translate = (key: string, params?: Record<string, number | string>) => string;

/**
 * The secondary line a work row reads as: "Worked for 20s · 10 tool calls",
 * "Thought for 15s", or just "2 context updates". Empty when the span holds
 * nothing of this thread (it only keeps the list contiguous).
 *
 * The duration runs from the conversation row before the span to the one
 * after it, so it is how long the agent took to answer, not just the time
 * between its first and last tool call. The span's own times stand in for a
 * missing neighbour (the tail of a live turn, the edge of the loaded page).
 */
export function workSummary(work: ConversationWorkSpan, t: Translate, previousAt?: string, nextAt?: string): string[] {
  const parts: string[] = [];
  const acted = work.toolCalls + work.toolResults > 0;
  if (acted || work.reasoning > 0) {
    const duration = durationText(elapsedMs(previousAt ?? work.startedAt, nextAt ?? work.endedAt), t);
    const verb = acted ? strings.messagesPane.work.worked : strings.messagesPane.work.thought;
    parts.push(duration ? t(verb, { duration }) : t(strings.messagesPane.work.working));
  }
  if (work.toolCalls > 0) parts.push(t(strings.messagesPane.work.toolCalls, { count: work.toolCalls }));
  if (work.context > 0) parts.push(t(strings.messagesPane.work.context, { count: work.context }));
  return parts;
}

/** A compact duration ("42s", "3m 12s", "1h 4m"), or null when unknown. */
export function durationText(ms: number | null, t: Translate): string | null {
  if (ms == null || ms < 0) return null;
  const total = Math.max(1, Math.round(ms / 1000));
  if (total < 60) return t(strings.messagesPane.duration.seconds, { seconds: total });
  if (total < 3600) return t(strings.messagesPane.duration.minutes, { minutes: Math.floor(total / 60), seconds: total % 60 });
  return t(strings.messagesPane.duration.hours, { hours: Math.floor(total / 3600), minutes: Math.floor((total % 3600) / 60) });
}

function elapsedMs(from?: string, to?: string): number | null {
  if (!from || !to) return null;
  const start = Date.parse(from);
  const end = Date.parse(to);
  if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) return null;
  return end - start;
}

/** How long a thread ran, for its group header; null under a second or unknown. */
export function threadDuration(thread: Pick<ConversationThreadSummary, "startedAt" | "endedAt">, t: Translate): string | null {
  const ms = elapsedMs(thread.startedAt, thread.endedAt);
  return ms == null || ms < 1000 ? null : durationText(ms, t);
}

const WORK_KIND_LABEL: Readonly<Partial<Record<ConversationKind, string>>> = {
  tool_call: strings.messagesPane.work.toolCall,
  tool_result: strings.messagesPane.work.toolResult,
  reasoning: strings.messagesPane.work.reasoning,
  context: strings.messagesPane.work.contextRow,
};

/** The label of one row inside an expanded work span, or of a search hit inside work. */
export function workKindLabel(kind: ConversationKind | undefined): string | null {
  return kind ? WORK_KIND_LABEL[kind] ?? null : null;
}

/**
 * One line naming what a work row did. Tool rows are stored as
 * "[Tool call: name]\n<arguments>\nCall ID: …"; the line is the tool and the
 * first meaningful argument line, never the call id.
 */
export function workRowLine(event: Pick<ConversationEvent, "label" | "text">): { title: string | null; detail: string } {
  const lines = event.text.split("\n").map((line) => line.trim());
  const body = lines.filter((line, index) => line !== "" && !(index === 0 && /^\[Tool (call|result)/.test(line)) && !line.startsWith("Call ID:"));
  return { title: event.label ?? null, detail: (body[0] ?? "").replace(/^"+|"+$/g, "").slice(0, 160) };
}
