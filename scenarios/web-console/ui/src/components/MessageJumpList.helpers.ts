import type { ConversationEvent } from "../api/conversation";
import { looksLikeFileReference } from "../lib/fileReferences";
import { strings } from "../consts/strings";

export type StatusGlyph = "played" | "playing" | "unseen" | "failed";

export interface StatusDescriptor {
  glyph: StatusGlyph;
  label: string;
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

/**
 * Collapses runs of whitespace into single spaces and trims, after stripping
 * Markdown. This is the canonical preview/search text for an event — used for
 * both matching and excerpt rendering. Never mutates the event's own `text`.
 */
export function normalizePreview(text: string): string {
  return stripMarkdown(text).replace(/\s+/g, " ").trim();
}

// assistantRoleLabelKey maps a conversation event source to the i18n key for
// the assistant's display name. Centralized so adding an agent runtime touches
// one place instead of every component that renders a source label.
export function assistantRoleLabelKey(source: string) {
  switch (source) {
    case "claude_hook":
      return strings.messageJumpList.roleClaude;
    case "opencode_api":
      return strings.messageJumpList.roleOpenCode;
    case "grok_tailer":
      return strings.messageJumpList.roleGrok;
    default:
      return strings.messageJumpList.roleCodex;
  }
}

// ---------------------------------------------------------------------------
// Navigator model — pure derivation of search/filter/sort/group results.
// Everything below is deterministic and React-free so it can be unit-tested
// against the contract independently of rendering.
// ---------------------------------------------------------------------------

/** A known assistant runtime source. `source:<id>` role filters key off this. */
export type SourceId = "claude" | "codex" | "opencode" | "grok";

/** Maps a raw event `source` string to the navigator's stable source id. */
export function sourceIdFor(source: string): SourceId {
  switch (source) {
    case "claude_hook":
      return "claude";
    case "opencode_api":
      return "opencode";
    case "grok_tailer":
      return "grok";
    default:
      return "codex";
  }
}

export type RoleFilter = "all" | "user" | "assistant" | `source:${SourceId}`;
export type StatusFilter = "all" | "unheard" | "played" | "failed" | "summarized";
export type ContentFilter = "all" | "code" | "fileReference" | "long";
export type SortMode = "oldest" | "newest" | "relevance";
export type GroupMode = "turn" | "flat" | "role";

export interface NavigatorState {
  query: string;
  role: RoleFilter;
  status: StatusFilter;
  content: ContentFilter;
  sort: SortMode;
  group: GroupMode;
}

export const DEFAULT_NAVIGATOR_STATE: NavigatorState = {
  query: "",
  role: "all",
  status: "all",
  content: "all",
  sort: "oldest",
  group: "turn",
};

export type ContentBadge = "code" | "fileReference" | "long";

/**
 * Messages whose normalized preview is at least this many characters are
 * considered "long" landmarks worth filtering to in dense sessions.
 */
export const LONG_MESSAGE_THRESHOLD = 600;

/** A run of preview text, flagged as a query match or plain text. */
export interface MatchSegment {
  text: string;
  match: boolean;
}

export interface NavigatorResult {
  event: ConversationEvent;
  /** Markdown-stripped, whitespace-collapsed preview (original case preserved). */
  preview: string;
  /** Excerpt centered on the first query match, split into highlight segments. */
  excerpt: MatchSegment[];
  badges: ContentBadge[];
  /** Number of query matches in the preview. 0 when no query. */
  score: number;
}

export type NoResultReason =
  | "noMessages"
  | "noSearchResults"
  | "noFilterResults"
  | "noResultsNarrow";

// --- Content badge detection -----------------------------------------------

const CODE_MARKER = /```|`[^`]+`|^( {4,}|\t)\S/m;

function extractFileRefCandidates(text: string): string[] {
  const candidates: string[] = [];
  for (const m of text.matchAll(/`([^`]+)`/g)) {
    if (m[1]) candidates.push(m[1].trim());
  }
  for (const m of text.matchAll(/\]\(([^)\s]+)\)/g)) {
    if (m[1]) candidates.push(m[1].trim());
  }
  for (const raw of text.split(/\s+/)) {
    const token = raw.replace(/[),.;:!?]+$/, "");
    if (!token) continue;
    if (token.includes("/") || /\.[a-z]{2,5}(:\d+)?$/i.test(token)) {
      candidates.push(token);
    }
  }
  return candidates;
}

/** Conservatively detects code/file-reference/long content landmarks. */
export function detectBadges(event: ConversationEvent): ContentBadge[] {
  const badges: ContentBadge[] = [];
  if (CODE_MARKER.test(event.text)) badges.push("code");
  if (extractFileRefCandidates(event.text).some((c) => looksLikeFileReference(c))) {
    badges.push("fileReference");
  }
  if (normalizePreview(event.text).length >= LONG_MESSAGE_THRESHOLD) badges.push("long");
  return badges;
}

// --- Excerpt + matching -----------------------------------------------------

const EXCERPT_MAX = 160;
const EXCERPT_LEAD = 32;

/**
 * Produces a highlight-segment excerpt centered on the first occurrence of
 * `query` within `preview`. With no query (or no match) it returns a leading
 * slice as a single non-match segment. Never emits HTML — callers render each
 * segment as a span.
 */
export function computeExcerpt(preview: string, query: string): MatchSegment[] {
  const trimmedQuery = query.trim();
  if (!trimmedQuery) {
    const head = preview.length > EXCERPT_MAX ? preview.slice(0, EXCERPT_MAX) + "…" : preview;
    return head ? [{ text: head, match: false }] : [];
  }

  const lowerPreview = preview.toLowerCase();
  const lowerQuery = trimmedQuery.toLowerCase();
  const first = lowerPreview.indexOf(lowerQuery);
  if (first < 0) {
    const head = preview.length > EXCERPT_MAX ? preview.slice(0, EXCERPT_MAX) + "…" : preview;
    return head ? [{ text: head, match: false }] : [];
  }

  const windowStart = Math.max(0, first - EXCERPT_LEAD);
  const windowEnd = Math.min(preview.length, windowStart + EXCERPT_MAX);
  const slice = preview.slice(windowStart, windowEnd);
  const sliceLower = slice.toLowerCase();

  const segments: MatchSegment[] = [];
  if (windowStart > 0) segments.push({ text: "…", match: false });

  let cursor = 0;
  let idx = sliceLower.indexOf(lowerQuery);
  while (idx >= 0) {
    if (idx > cursor) segments.push({ text: slice.slice(cursor, idx), match: false });
    segments.push({ text: slice.slice(idx, idx + lowerQuery.length), match: true });
    cursor = idx + lowerQuery.length;
    idx = sliceLower.indexOf(lowerQuery, cursor);
  }
  if (cursor < slice.length) segments.push({ text: slice.slice(cursor), match: false });
  if (windowEnd < preview.length) segments.push({ text: "…", match: false });

  return segments;
}

function countOccurrences(haystack: string, needle: string): number {
  if (!needle) return 0;
  let count = 0;
  let idx = haystack.indexOf(needle);
  while (idx >= 0) {
    count += 1;
    idx = haystack.indexOf(needle, idx + needle.length);
  }
  return count;
}

/**
 * Lowercased search tokens for an event's metadata: role words, source label,
 * and sequence number. Locale-independent on purpose — these are stable
 * identifiers, not display copy.
 */
function metaTokens(event: ConversationEvent): string {
  if (event.role === "user") return `user you #${event.sequence} ${event.sequence}`;
  return `assistant ${sourceIdFor(event.source)} #${event.sequence} ${event.sequence}`;
}

// --- Filters ----------------------------------------------------------------

function matchesRole(event: ConversationEvent, role: RoleFilter): boolean {
  if (role === "all") return true;
  if (role === "user") return event.role === "user";
  if (role === "assistant") return event.role === "assistant";
  // source:<id>
  const wanted = role.slice("source:".length) as SourceId;
  return event.role === "assistant" && sourceIdFor(event.source) === wanted;
}

function matchesStatus(event: ConversationEvent, status: StatusFilter): boolean {
  switch (status) {
    case "all":
      return true;
    case "failed":
      return event.ttsState === "failed" || event.ttsState === "rejected";
    case "played":
      return event.ttsState === "played" || event.consumptionState === "listened";
    case "unheard":
      return event.ttsState !== "played" && event.consumptionState !== "listened";
    case "summarized":
      return event.summarized;
    default:
      return true;
  }
}

function matchesContent(badges: ContentBadge[], content: ContentFilter): boolean {
  if (content === "all") return true;
  return badges.includes(content);
}

/**
 * The set of assistant runtime sources actually present in `events`. Used to
 * hide source filter chips that would match nothing.
 */
export function availableSources(events: ConversationEvent[]): SourceId[] {
  const seen = new Set<SourceId>();
  for (const e of events) {
    if (e.role === "assistant") seen.add(sourceIdFor(e.source));
  }
  const order: SourceId[] = ["claude", "codex", "opencode", "grok"];
  return order.filter((id) => seen.has(id));
}

// --- The one pure derivation -------------------------------------------------

/**
 * Applies query, role/status/content filters, and sort to produce the flat,
 * ordered result list. Grouping is a presentation concern handled separately
 * by groupResults so keyboard indexing always refers to this flat list.
 */
export function buildResults(
  events: ConversationEvent[],
  state: NavigatorState,
): NavigatorResult[] {
  const query = state.query.trim().toLowerCase();

  const filtered: NavigatorResult[] = [];
  for (let i = 0; i < events.length; i += 1) {
    const event = events[i];
    if (!event) continue;
    if (!matchesRole(event, state.role)) continue;
    if (!matchesStatus(event, state.status)) continue;

    const badges = detectBadges(event);
    if (!matchesContent(badges, state.content)) continue;

    const preview = normalizePreview(event.text);

    let score = 0;
    if (query) {
      const haystack = `${preview.toLowerCase()} ${metaTokens(event)}`;
      score = countOccurrences(haystack, query);
      if (score === 0) continue; // query present but no match anywhere
    }

    filtered.push({
      event,
      preview,
      excerpt: computeExcerpt(preview, state.query),
      badges,
      score,
    });
  }

  return sortResults(filtered, state.sort, Boolean(query));
}

function sortResults(
  results: NavigatorResult[],
  sort: SortMode,
  hasQuery: boolean,
): NavigatorResult[] {
  // Relevance has no meaning without a query — fall back to conversation order.
  const effective: SortMode = sort === "relevance" && !hasQuery ? "oldest" : sort;
  const sorted = [...results];
  switch (effective) {
    case "newest":
      sorted.sort((a, b) => b.event.sequence - a.event.sequence);
      break;
    case "relevance":
      sorted.sort(
        (a, b) => b.score - a.score || a.event.sequence - b.event.sequence,
      );
      break;
    case "oldest":
    default:
      sorted.sort((a, b) => a.event.sequence - b.event.sequence);
      break;
  }
  return sorted;
}

/**
 * Explains why a result list is empty so the UI can show a precise empty
 * state. `totalEvents` is the unfiltered session size.
 */
export function noResultReason(
  totalEvents: number,
  state: NavigatorState,
): NoResultReason {
  if (totalEvents === 0) return "noMessages";
  const hasQuery = state.query.trim().length > 0;
  const hasFilters =
    state.role !== "all" || state.status !== "all" || state.content !== "all";
  if (hasQuery && hasFilters) return "noResultsNarrow";
  if (hasQuery) return "noSearchResults";
  return "noFilterResults";
}

// --- Grouping (presentation only) -------------------------------------------

export type GroupRole = "user" | "assistant";

export interface NavigatorGroup {
  id: string;
  /** Turn mode: the user result that opens the turn (rendered as a header). */
  leadUser: NavigatorResult | null;
  /** Role mode: which role this group collects. */
  roleLabel: GroupRole | null;
  /** Rows to render under this group (excludes leadUser). */
  items: NavigatorResult[];
}

/**
 * Arranges the flat result list into display groups without reordering across
 * the flat order. Keyboard indexing must continue to use the flat list.
 */
export function groupResults(
  results: NavigatorResult[],
  group: GroupMode,
): NavigatorGroup[] {
  if (results.length === 0) return [];

  if (group === "flat") {
    return [{ id: "flat", leadUser: null, roleLabel: null, items: results }];
  }

  if (group === "role") {
    const users = results.filter((r) => r.event.role === "user");
    const assistants = results.filter((r) => r.event.role === "assistant");
    const groups: NavigatorGroup[] = [];
    if (users.length > 0) {
      groups.push({ id: "role-user", leadUser: null, roleLabel: "user", items: users });
    }
    if (assistants.length > 0) {
      groups.push({
        id: "role-assistant",
        leadUser: null,
        roleLabel: "assistant",
        items: assistants,
      });
    }
    return groups;
  }

  // turn
  const groups: NavigatorGroup[] = [];
  let current: NavigatorGroup | null = null;
  for (const result of results) {
    if (result.event.role === "user") {
      if (current) groups.push(current);
      current = { id: `turn-${result.event.id}`, leadUser: result, roleLabel: null, items: [] };
    } else {
      if (!current) {
        current = { id: `turn-lead-${result.event.id}`, leadUser: null, roleLabel: null, items: [] };
      }
      current.items.push(result);
    }
  }
  if (current) groups.push(current);
  return groups;
}
