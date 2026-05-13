import type { ConversationEvent } from "../api/conversation";

export type StatusGlyph = "played" | "playing" | "unseen" | "failed";

export interface Turn {
  /** The user event that initiated this turn, or null for a leading run of assistant events. */
  user: ConversationEvent | null;
  assistants: ConversationEvent[];
}

export interface StatusDescriptor {
  glyph: StatusGlyph;
  label: string;
}

export function groupEventsByTurn(events: ConversationEvent[]): Turn[] {
  const turns: Turn[] = [];
  let current: Turn | null = null;
  for (const event of events) {
    if (event.role === "user") {
      if (current) turns.push(current);
      current = { user: event, assistants: [] };
    } else {
      if (!current) current = { user: null, assistants: [] };
      current.assistants.push(event);
    }
  }
  if (current) turns.push(current);
  return turns;
}

export function formatRelativeTime(iso: string, now: Date = new Date()): string {
  const created = new Date(iso).getTime();
  if (Number.isNaN(created)) return "";
  const diffSec = Math.max(0, Math.floor((now.getTime() - created) / 1000));
  if (diffSec < 45) return "just now";
  if (diffSec < 60 * 60) {
    const minutes = Math.max(1, Math.round(diffSec / 60));
    return `${minutes}m`;
  }
  if (diffSec < 60 * 60 * 24) {
    const hours = Math.max(1, Math.round(diffSec / 3600));
    return `${hours}h`;
  }
  return new Date(iso).toLocaleDateString();
}

export function statusGlyphFor(event: ConversationEvent): StatusDescriptor {
  switch (event.ttsState) {
    case "playing":
      return { glyph: "playing", label: "Playing" };
    case "played":
      return { glyph: "played", label: "Played" };
    case "failed":
      return { glyph: "failed", label: "Failed" };
    case "rejected":
      return { glyph: "failed", label: "Rejected" };
    case "idle":
    default:
      if (event.consumptionState === "listened") return { glyph: "played", label: "Listened" };
      return { glyph: "unseen", label: "Unseen" };
  }
}

/**
 * Strips the most common Markdown decorations so an event preview reads as
 * prose. Intentionally minimal — full markdown rendering would be wrong for
 * a one-line preview.
 */
export function stripMarkdown(text: string): string {
  return text
    .replace(/```[\s\S]*?```/g, " ")        // fenced code
    .replace(/`([^`]+)`/g, "$1")              // inline code
    .replace(/^\s*#{1,6}\s+/gm, "")          // ATX headings
    .replace(/\*\*([^*]+)\*\*/g, "$1")         // bold
    .replace(/__([^_]+)__/g, "$1")              // bold-alt
    .replace(/(?<!\*)\*([^*\n]+)\*(?!\*)/g, "$1") // italic
    .replace(/!\[([^\]]*)\]\([^)]+\)/g, "$1")   // images → alt
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1");    // links → text
}

export type FilterMode = "all" | "unheard" | "failed";

export function applyFilter(events: ConversationEvent[], filter: FilterMode): ConversationEvent[] {
  if (filter === "all") return events;
  if (filter === "failed") {
    return events.filter((e) => e.ttsState === "failed" || e.ttsState === "rejected");
  }
  return events.filter(
    (e) => e.ttsState !== "played" && e.consumptionState !== "listened",
  );
}
