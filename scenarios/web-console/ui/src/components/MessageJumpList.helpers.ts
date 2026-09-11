import type { ConversationEvent, ConversationSearchMatch, SearchTextRange } from "../api/conversation";
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
export type SortMode = "oldest" | "newest";
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

export interface DerivedEvent {
  preview: string;
  badges: ContentBadge[];
}

const derivedEvents = new WeakMap<ConversationEvent, DerivedEvent>();

export function getDerived(event: ConversationEvent): DerivedEvent {
  const cached = derivedEvents.get(event);
  if (cached) return cached;
  const preview = normalizePreview(event.text);
  const derived = {
    preview,
    badges: detectBadges(event),
  };
  derivedEvents.set(event, derived);
  return derived;
}

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
  /** The row's excerpt as segments; server search marks its matches. */
  excerpt: MatchSegment[];
  badges: ContentBadge[];
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

/** The leading slice of a preview, as one plain segment. */
export function previewSegments(preview: string): MatchSegment[] {
  const head = preview.length > EXCERPT_MAX ? preview.slice(0, EXCERPT_MAX) + "…" : preview;
  return head ? [{ text: head, match: false }] : [];
}

/**
 * Splits a server excerpt at the server's match ranges (UTF-16 offsets, the
 * units JS strings index by). A range that runs past the excerpt, overlaps an
 * earlier one, or is empty is dropped: the navigator never re-matches a query
 * itself.
 */
export function excerptSegments(excerpt: string, ranges: readonly SearchTextRange[]): MatchSegment[] {
  const segments: MatchSegment[] = [];
  let cursor = 0;
  for (const range of [...ranges].sort((a, b) => a.start - b.start)) {
    if (range.start < cursor || range.end > excerpt.length || range.start >= range.end) continue;
    if (range.start > cursor) segments.push({ text: excerpt.slice(cursor, range.start), match: false });
    segments.push({ text: excerpt.slice(range.start, range.end), match: true });
    cursor = range.end;
  }
  if (cursor < excerpt.length) segments.push({ text: excerpt.slice(cursor), match: false });
  return segments;
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
 * The loaded events under the role, status, and content filters, sorted. A
 * query is not applied here: search runs on the server over the whole history
 * (buildSearchResults lists its hits). Grouping is a presentation concern
 * handled by groupResults so keyboard indexing always refers to this flat list.
 */
export function buildResults(
  events: ConversationEvent[],
  state: NavigatorState,
): NavigatorResult[] {
  const filtered: NavigatorResult[] = [];
  for (const event of events) {
    if (!matchesRole(event, state.role)) continue;
    if (!matchesStatus(event, state.status)) continue;
    const { badges, preview } = getDerived(event);
    if (!matchesContent(badges, state.content)) continue;
    filtered.push({ event, preview, excerpt: previewSegments(preview), badges });
  }
  return sortResults(filtered, state.sort);
}

/** A hit outside the loaded window, listed from what the server reported. */
function eventFromHit(hit: ConversationSearchMatch, assistantSource: string): ConversationEvent {
  return {
    id: hit.eventId,
    sessionId: "",
    source: hit.role === "user" ? "" : assistantSource,
    role: hit.role,
    text: hit.excerpt,
    speechParagraphs: [hit.excerpt],
    summarized: false,
    createdAt: hit.createdAt,
    sequence: hit.sequence,
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "seen",
  };
}

/**
 * The server's search hits as navigator rows, in sequence order, highlighted
 * with the server's ranges. A hit that is loaded is shown as its event; one
 * outside the loaded window keeps the server's role and time. Status and
 * content filters need the event itself, so they keep loaded hits only.
 */
export function buildSearchResults(
  hits: readonly ConversationSearchMatch[],
  events: readonly ConversationEvent[],
  state: NavigatorState,
): NavigatorResult[] {
  const loaded = new Map(events.map((event) => [event.id, event]));
  const assistantSource = events.find((event) => event.role === "assistant")?.source ?? "";
  const needsEvent = state.status !== "all" || state.content !== "all";
  const results: NavigatorResult[] = [];
  for (const hit of hits) {
    const known = loaded.get(hit.eventId);
    if (!known && needsEvent) continue;
    const event = known ?? eventFromHit(hit, assistantSource);
    if (!matchesRole(event, state.role)) continue;
    if (known && !matchesStatus(known, state.status)) continue;
    const badges = known ? getDerived(known).badges : [];
    if (known && !matchesContent(badges, state.content)) continue;
    results.push({ event, preview: hit.excerpt, excerpt: excerptSegments(hit.excerpt, hit.ranges), badges });
  }
  return sortResults(results, state.sort);
}

function sortResults(results: NavigatorResult[], sort: SortMode): NavigatorResult[] {
  const sorted = [...results];
  sorted.sort(sort === "newest"
    ? (a, b) => b.event.sequence - a.event.sequence
    : (a, b) => a.event.sequence - b.event.sequence);
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

/**
 * A navigator row's expected height before it is measured: the header line
 * plus one or two clamped preview lines. The virtualizer replaces it with the
 * measured height as soon as the row renders.
 */
export function estimateNavRowHeight(result: NavigatorResult): number {
  const lines = result.preview.length > 48 ? 2 : 1;
  return (result.event.role === "user" ? 30 : 26) + lines * 17;
}
