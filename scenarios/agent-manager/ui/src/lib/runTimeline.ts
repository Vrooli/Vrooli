import type { RunEvent } from "../types.js";

export type TimelineDisplayMode = "conversation" | "combined" | "events";

export type TimelineCategory =
  | "messages"
  | "reasoning"
  | "tools"
  | "errors"
  | "status"
  | "logs"
  | "artifacts"
  | "metrics"
  | "compaction"
  | "redactions";

export interface TimelineFilterState {
  mode: TimelineDisplayMode;
  categories: Record<TimelineCategory, boolean>;
}

export interface TimelineMessageEntry {
  id: string;
  kind: "message";
  category: "messages";
  event: RunEvent;
  role: "user" | "assistant" | "system";
  content: string;
  attachments: Array<{ id: string; fileName: string; url: string }>;
  deleted: boolean;
}

export interface TimelineEventEntry {
  id: string;
  kind: "event";
  category: Exclude<TimelineCategory, "messages">;
  event: RunEvent;
}

export type TimelineEntry = TimelineMessageEntry | TimelineEventEntry;

export const TIMELINE_CATEGORY_ORDER: TimelineCategory[] = [
  "messages",
  "reasoning",
  "tools",
  "errors",
  "compaction",
  "status",
  "logs",
  "artifacts",
  "metrics",
  "redactions",
];

export function createDefaultTimelineFilterState(): TimelineFilterState {
  return {
    mode: "combined",
    categories: {
      messages: true,
      reasoning: true,
      tools: true,
      errors: true,
      status: false,
      logs: false,
      artifacts: false,
      metrics: false,
      compaction: true,
      redactions: false,
    },
  };
}

export function createShowAllTimelineFilterState(): TimelineFilterState {
  return {
    mode: "events",
    categories: TIMELINE_CATEGORY_ORDER.reduce<Record<TimelineCategory, boolean>>((acc, category) => {
      acc[category] = true;
      return acc;
    }, {} as Record<TimelineCategory, boolean>),
  };
}

export function getTimelineModeLabel(mode: TimelineDisplayMode): string {
  switch (mode) {
    case "conversation":
      return "Conversation";
    case "events":
      return "Events";
    default:
      return "Combined";
  }
}

export function getTimelineCategoryLabel(category: TimelineCategory): string {
  switch (category) {
    case "messages":
      return "Messages";
    case "reasoning":
      return "Reasoning";
    case "tools":
      return "Tool Use";
    case "errors":
      return "Errors";
    case "status":
      return "Status";
    case "logs":
      return "Logs";
    case "artifacts":
      return "Artifacts";
    case "metrics":
      return "Metrics";
    case "compaction":
      return "Compaction";
    case "redactions":
      return "Redactions";
  }
}

export function stripImageContextTags(content: string): string {
  return content.replace(/<context\s+[^>]*type="image"[^>]*>\s*<\/context>/g, "").trim();
}

export function isReasoningEvent(event: RunEvent): boolean {
  if (event.data.case !== "log") return false;
  const message = String(event.data.value.message ?? "");
  return /^reasoning:\s*/i.test(message) || /^thinking:\s*/i.test(message);
}

export function getTimelineCategory(event: RunEvent): TimelineCategory {
  switch (event.data.case) {
    case "message":
      return "messages";
    case "messageDeleted":
      return "redactions";
    case "compaction":
      return "compaction";
    case "toolCall":
    case "toolResult":
      return "tools";
    case "error":
    case "rateLimit":
      return "errors";
    case "status":
    case "progress":
      return "status";
    case "artifact":
      return "artifacts";
    case "metric":
    case "cost":
      return "metrics";
    case "log":
      return isReasoningEvent(event) ? "reasoning" : "logs";
    default:
      return "logs";
  }
}

function toSequenceComparable(event: RunEvent): number {
  return typeof event.sequence === "bigint" ? Number(event.sequence) : Number(event.sequence ?? 0);
}

function toTimestampComparable(event: RunEvent): number {
  if (!event.timestamp) return 0;
  const seconds = Number(event.timestamp.seconds ?? 0);
  const nanos = Number(event.timestamp.nanos ?? 0);
  return (seconds * 1000) + Math.floor(nanos / 1_000_000);
}

export function sortRunEvents(events: RunEvent[]): RunEvent[] {
  return [...events].sort((a, b) => {
    const bySequence = toSequenceComparable(a) - toSequenceComparable(b);
    if (bySequence !== 0) return bySequence;
    return toTimestampComparable(a) - toTimestampComparable(b);
  });
}

export function buildTimelineEntries(events: RunEvent[]): TimelineEntry[] {
  const ordered = sortRunEvents(events);
  const deletedMessageIds = new Set<string>();

  for (const event of ordered) {
    if (event.data.case !== "messageDeleted") continue;
    const targetEventId = event.data.value.targetEventId;
    if (targetEventId) deletedMessageIds.add(targetEventId);
  }

  const entries: TimelineEntry[] = [];
  for (const event of ordered) {
    if (event.data.case === "message") {
      const role = String(event.data.value.role ?? "").toLowerCase();
      if (role !== "user" && role !== "assistant" && role !== "system") continue;

      const content = stripImageContextTags(String(event.data.value.content ?? ""));
      const attachments = (event.data.value.attachments ?? []).map((attachment: {
        id: string;
        fileName: string;
        url: string;
      }) => ({
        id: attachment.id,
        fileName: attachment.fileName,
        url: attachment.url,
      }));

      if (!content && attachments.length === 0) continue;

      entries.push({
        id: event.id,
        kind: "message",
        category: "messages",
        event,
        role,
        content,
        attachments,
        deleted: deletedMessageIds.has(event.id),
      });
      continue;
    }

    const category = getTimelineCategory(event);
    if (category === "messages") continue;

    entries.push({
      id: event.id,
      kind: "event",
      category,
      event,
    });
  }

  return entries;
}

export function filterTimelineEntries(
  entries: TimelineEntry[],
  filters: TimelineFilterState
): TimelineEntry[] {
  return entries.filter((entry) => {
    if (entry.kind === "message") {
      return filters.mode !== "events" && filters.categories.messages;
    }

    if (filters.mode === "conversation") return false;
    return filters.categories[entry.category];
  });
}

export function countTimelineEntriesByCategory(entries: TimelineEntry[]): Record<TimelineCategory, number> {
  const counts = TIMELINE_CATEGORY_ORDER.reduce<Record<TimelineCategory, number>>((acc, category) => {
    acc[category] = 0;
    return acc;
  }, {} as Record<TimelineCategory, number>);

  for (const entry of entries) {
    counts[entry.category] += 1;
  }

  return counts;
}

export function getTimelineEventLabel(entry: TimelineEventEntry): string {
  switch (entry.category) {
    case "reasoning":
      return "Reasoning";
    case "tools":
      return "Tool";
    case "errors":
      return "Error";
    case "status":
      return "Status";
    case "artifacts":
      return "Artifact";
    case "metrics":
      return "Metric";
    case "compaction":
      return "Compaction";
    case "redactions":
      return "Redaction";
    default:
      return "Log";
  }
}

export function getTimelineEventSummary(entry: TimelineEventEntry): string {
  const { event } = entry;
  const payload = event.data.value as Record<string, unknown>;

  switch (event.data.case) {
    case "log": {
      const message = String(payload.message ?? "Log entry");
      if (entry.category === "reasoning") {
        return message.replace(/^reasoning:\s*/i, "").replace(/^thinking:\s*/i, "");
      }
      return message.replace(/^phase:\s*/i, "");
    }
    case "toolCall":
      return String(payload.toolName ?? "Unknown tool");
    case "toolResult":
      return `${payload.success ? "Completed" : "Failed"} ${String(payload.toolName ?? "tool")}`;
    case "status":
      return `${String(payload.oldStatus ?? "unknown")} -> ${String(payload.newStatus ?? "unknown")}`;
    case "progress":
      return `Progress ${String(payload.percentComplete ?? 0)}%`;
    case "artifact":
      return String(payload.path ?? payload.type ?? "Artifact created");
    case "metric":
      return `${String(payload.name ?? "metric")}: ${String(payload.value ?? 0)}`;
    case "cost":
      return payload.totalCostUsd != null
        ? `Cost update $${Number(payload.totalCostUsd).toFixed(4)}`
        : "Cost update";
    case "error":
      return String(payload.message ?? payload.code ?? "Error");
    case "rateLimit":
      return String(payload.message ?? "Rate limited");
    case "compaction":
      return String(payload.summary ?? payload.trigger ?? "Context compacted");
    case "messageDeleted":
      return `Message ${String(payload.targetEventId ?? "").slice(0, 8)} redacted`;
    default:
      return getTimelineEventLabel(entry);
  }
}
