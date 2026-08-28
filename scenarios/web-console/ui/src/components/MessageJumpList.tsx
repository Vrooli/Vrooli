import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type RefObject,
} from "react";
import { createPortal } from "react-dom";
import { BottomSheet } from "@vrooli/react-component-library/BottomSheet/1";
import { AlignLeft, CheckSquare, ClipboardCopy, Code, FileText, Pause, Play, Search, Square, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { ConversationEvent } from "../api/conversation";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useVirtualList } from "../hooks/useVirtualList";
import { useAnchoredPopoverPosition, type FloatingPlacement } from "../hooks/useFloatingPosition";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import { buildMessageExport, DEFAULT_MESSAGE_EXPORT_FORMAT } from "../lib/messageExport";
import { getScrubClasses } from "./tts/scrubStyles";
import {
  assistantRoleLabelKey,
  availableSources,
  buildResults,
  formatRelativeTime,
  groupResults,
  noResultReason,
  statusGlyphFor,
  type ContentBadge,
  type ContentFilter,
  type GroupMode,
  type NavigatorResult,
  type NavigatorState,
  type RoleFilter,
  type SortMode,
  type SourceId,
  type StatusFilter,
  type StatusGlyph,
} from "./MessageJumpList.helpers";

/**
 * Whether the navigator jumps to a message (Messages view) or selects a
 * playback start (AudioPlayerBar). Drives copy and which surface is emphasized
 * so the same component never silently means two different things.
 */
export type MessageNavigatorMode = "jump" | "playback-select";

/**
 * Export-selection contract. Selected IDs live in MessagesPane (the single
 * source of truth shared with the export drawer); the navigator only renders
 * selection affordances and reports intent through these callbacks.
 */
export interface MessageExportSelection {
  selectedIds: ReadonlySet<string>;
  onToggle: (eventId: string) => void;
  /** Select the entire conversation (every event id). */
  onSelectAll: () => void;
  /** Select exactly the currently visible (filtered) result ids. */
  onSelectVisible: (eventIds: string[]) => void;
  onClear: () => void;
  /** Proceed to the export drawer with the current selection. */
  onContinue: () => void;
}

interface MessageJumpListProps {
  events: ConversationEvent[];
  focusedEventId: string | null;
  onSelect: (eventId: string) => void;
  onClose: () => void;
  mode?: MessageNavigatorMode;
  /** Where focus lands on open. Defaults to search for jump, list for playback. */
  initialFocus?: "search" | "list";
  /** Controlled search query — lifted by MessagesPane so dimming/nav persist. */
  query?: string;
  onQueryChange?: (query: string) => void;
  /** Whole-session server search metadata; local events remain the rendered window. */
  searchMatchCount?: number;
  searchTruncated?: boolean;
  /** Desktop anchor: when set, the panel positions itself against this
   *  element via the shared anchored-floating math (above, end-aligned). */
  desktopAnchorRef?: RefObject<HTMLElement | null>;
  /** Mini playback header — playback state from the parent AudioPlayerBar. */
  currentTime?: number;
  duration?: number | null;
  isPaused?: boolean;
  /** Whether the currently playing event is the summarized version. */
  isSummarized?: boolean;
  onPause?: () => void;
  onResume?: () => void;
  onSeek?: (seconds: number) => void;
  /** Whether the next event is queued and will auto-play after the current one. */
  hasQueuedNext?: boolean;
  /**
   * Enables the Export header action (jump mode only). Activating it switches
   * the navigator into an explicit selection mode where rows toggle instead of
   * jumping; normal jump and playback-select semantics are unchanged.
   */
  exportSelection?: MessageExportSelection;
}

const SOURCE_LABEL_KEY = {
  claude: strings.messageJumpList.roleClaude,
  codex: strings.messageJumpList.roleCodex,
  opencode: strings.messageJumpList.roleOpenCode,
  grok: strings.messageJumpList.roleGrok,
} satisfies Record<SourceId, string>;

const STATUS_OPTIONS: StatusFilter[] = ["all", "unheard", "played", "failed", "summarized"];
const STATUS_LABEL_KEY = {
  all: strings.messageJumpList.statusAll,
  unheard: strings.messageJumpList.statusUnheard,
  played: strings.messageJumpList.statusPlayed,
  failed: strings.messageJumpList.statusFailed,
  summarized: strings.messageJumpList.statusSummarized,
} satisfies Record<StatusFilter, string>;

const CONTENT_OPTIONS: ContentFilter[] = ["all", "code", "fileReference", "long"];
const CONTENT_LABEL_KEY = {
  all: strings.messageJumpList.contentAll,
  code: strings.messageJumpList.contentCode,
  fileReference: strings.messageJumpList.contentFile,
  long: strings.messageJumpList.contentLong,
} satisfies Record<ContentFilter, string>;

const SORT_OPTIONS: SortMode[] = ["oldest", "newest", "relevance"];
const SORT_LABEL_KEY = {
  oldest: strings.messageJumpList.sortOldest,
  newest: strings.messageJumpList.sortNewest,
  relevance: strings.messageJumpList.sortRelevance,
} satisfies Record<SortMode, string>;

const GROUP_OPTIONS: GroupMode[] = ["turn", "flat", "role"];
const GROUP_LABEL_KEY = {
  turn: strings.messageJumpList.groupTurn,
  flat: strings.messageJumpList.groupFlat,
  role: strings.messageJumpList.groupRole,
} satisfies Record<GroupMode, string>;

const BADGE_META = {
  code: { Icon: Code, labelKey: strings.messageJumpList.badgeCode },
  fileReference: { Icon: FileText, labelKey: strings.messageJumpList.badgeFile },
  long: { Icon: AlignLeft, labelKey: strings.messageJumpList.badgeLong },
} satisfies Record<ContentBadge, { Icon: typeof Code; labelKey: string }>;

function StatusIcon({ glyph, className }: { glyph: StatusGlyph; className?: string }) {
  if (glyph === "playing") {
    return (
      <span aria-hidden="true" className={cn("inline-flex h-3 w-3 items-center justify-center", className)}>
        <span className="block h-2 w-2 animate-pulse rounded-full bg-wc-accent" />
      </span>
    );
  }
  if (glyph === "played") {
    return (
      <span aria-hidden="true" className={cn("inline-flex h-3 w-3 items-center justify-center text-emerald-400", className)}>
        ✓
      </span>
    );
  }
  if (glyph === "failed") {
    return (
      <span aria-hidden="true" className={cn("inline-flex h-3 w-3 items-center justify-center text-red-400", className)}>
        ✗
      </span>
    );
  }
  return (
    <span aria-hidden="true" className={cn("inline-flex h-3 w-3 items-center justify-center text-wc-text-faint/70", className)}>
      ○
    </span>
  );
}

function NowPlayingHeader({
  event,
  currentTime,
  duration,
  isPaused,
  isSummarized,
  onPause,
  onResume,
  onSeek,
  now,
  onJumpToCurrent,
}: {
  event: ConversationEvent | null;
  currentTime: number;
  duration: number | null;
  isPaused: boolean;
  isSummarized: boolean;
  onPause?: () => void;
  onResume?: () => void;
  onSeek?: (seconds: number) => void;
  now: Date;
  onJumpToCurrent: () => void;
}) {
  const { t } = useTranslation();
  const handleScrub = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      onSeek?.(Number(e.target.value));
    },
    [onSeek],
  );

  if (!event || duration === null) {
    return (
      <div data-testid="msg-jump-now-playing" data-state="idle" className="px-3 py-2 text-[11px] text-wc-text-faint">
        {t(strings.messageJumpList.noActivePlayback)}
      </div>
    );
  }

  const desc = statusGlyphFor(event);
  const roleLabel = event.role === "user"
    ? t(strings.messageJumpList.roleYou)
    : t(assistantRoleLabelKey(event.source));
  const seekable = duration > 0 && !!onSeek;

  return (
    <div data-testid="msg-jump-now-playing" data-state="playing" className="border-b border-wc-default/60 px-3 pt-2 pb-2">
      <div className="mb-1.5 flex items-center gap-2">
        <button
          type="button"
          data-testid="msg-jump-now-playpause"
          onClick={() => (isPaused ? onResume?.() : onPause?.())}
          className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-wc-surface-input text-wc-text-primary transition hover:bg-wc-accent/20"
          aria-label={isPaused ? t(strings.messageJumpList.resume) : t(strings.messageJumpList.pause)}
        >
          {isPaused ? <Play className="h-4 w-4" /> : <Pause className="h-4 w-4" />}
        </button>
        <button
          type="button"
          data-testid="msg-jump-now-jump"
          onClick={onJumpToCurrent}
          className="flex min-w-0 flex-1 items-center gap-1.5 rounded px-1 py-0.5 text-start transition hover:bg-wc-surface-input/60"
          aria-label={t(strings.messageJumpList.scrollToCurrent)}
        >
          <StatusIcon glyph={desc.glyph} />
          <span className="font-mono text-[11px] text-wc-text-faint">#{event.sequence}</span>
          <span className="text-[11px] font-medium text-wc-text-primary">{roleLabel}</span>
          <span className="ml-auto shrink-0 ps-1 text-[10px] text-wc-text-faint">
            {formatRelativeTime(event.createdAt, now)}
          </span>
        </button>
      </div>
      <input
        data-testid="msg-jump-now-scrub"
        type="range"
        min={0}
        max={seekable ? duration : 0}
        value={Math.min(currentTime, duration)}
        step={0.1}
        disabled={!seekable}
        onChange={handleScrub}
        aria-label={t(strings.messageJumpList.seekCurrent)}
        className={getScrubClasses({ isSummarized, enabled: seekable, extra: "w-full" })}
      />
    </div>
  );
}

function Highlight({ result }: { result: NavigatorResult }) {
  return (
    <>
      {result.excerpt.map((seg, i) =>
        seg.match ? (
          <span key={i} data-match="true" className="rounded-[2px] bg-wc-accent/30 text-wc-text-primary">
            {seg.text}
          </span>
        ) : (
          <span key={i}>{seg.text}</span>
        ),
      )}
    </>
  );
}

function Badges({ result }: { result: NavigatorResult }) {
  const { t } = useTranslation();
  if (result.badges.length === 0) return null;
  return (
    <span className="inline-flex items-center gap-1">
      {result.badges.map((badge) => {
        const meta = BADGE_META[badge];
        const Icon = meta.Icon;
        return (
          <span
            key={badge}
            data-testid={`msg-nav-badge-${badge}-${result.event.id}`}
            className="inline-flex h-4 w-4 items-center justify-center rounded bg-wc-surface-input/70 text-wc-text-faint"
            title={t(meta.labelKey)}
            aria-label={t(meta.labelKey)}
          >
            <Icon className="h-2.5 w-2.5" aria-hidden="true" />
          </span>
        );
      })}
    </span>
  );
}

function NavRow({
  result,
  isFocused,
  isActive,
  isNext,
  onSelect,
  now,
  selectMode = false,
  isSelected = false,
}: {
  result: NavigatorResult;
  isFocused: boolean;
  isActive: boolean;
  isNext: boolean;
  onSelect: () => void;
  now: Date;
  /** Export-selection mode: the row toggles a checkbox instead of jumping. */
  selectMode?: boolean;
  isSelected?: boolean;
}) {
  const { t } = useTranslation();
  const { event } = result;
  const isUser = event.role === "user";
  const desc = statusGlyphFor(event);
  const roleLabel = isUser ? t(strings.messageJumpList.roleYou) : t(assistantRoleLabelKey(event.source));

  return (
    <button
      type="button"
      data-testid={`msg-jump-item-${event.id}`}
      data-jump-item
      data-role={isUser ? "user" : "assistant"}
      data-glyph={isUser ? undefined : desc.glyph}
      data-selected={selectMode ? isSelected : undefined}
      role={selectMode ? "checkbox" : undefined}
      aria-checked={selectMode ? isSelected : undefined}
      aria-label={selectMode ? t(strings.messageExport.selectMessageAria, { sequence: event.sequence }) : undefined}
      aria-current={!isUser && !selectMode && desc.glyph === "playing" ? "true" : undefined}
      onClick={onSelect}
      className={cn(
        "flex w-full flex-col items-start gap-0.5 rounded-lg px-3 py-2 text-start transition",
        isUser ? "min-h-[48px] border" : "min-h-[44px]",
        isFocused
          ? isUser
            ? "border-wc-accent/50 bg-wc-accent/15 text-wc-text-primary"
            : "bg-wc-accent/15 text-wc-text-primary"
          : isUser
            ? "border-wc-default/60 bg-wc-surface-input/40 text-wc-text-secondary hover:bg-wc-surface-input"
            : "text-wc-text-secondary hover:bg-wc-surface-input/70 hover:text-wc-text-primary",
        isActive && !isFocused && "ring-1 ring-wc-accent/30",
      )}
    >
      <span className="flex w-full items-center gap-1.5 text-[11px]">
        {selectMode && (
          <span
            aria-hidden="true"
            data-testid={`msg-export-check-${event.id}`}
            className={cn("inline-flex shrink-0 items-center justify-center", isSelected ? "text-wc-accent" : "text-wc-text-faint")}
          >
            {isSelected ? <CheckSquare className="h-4 w-4" /> : <Square className="h-4 w-4" />}
          </span>
        )}
        {!isUser && <StatusIcon glyph={desc.glyph} />}
        <span className="font-mono text-wc-text-faint">#{event.sequence}</span>
        <span className={cn("font-medium", isUser && "text-wc-text-primary")}>{roleLabel}</span>
        <span className="text-wc-text-faint">· {formatRelativeTime(event.createdAt, now)}</span>
        <Badges result={result} />
        {event.summarized && (
          <span
            data-testid={`msg-jump-summarized-${event.id}`}
            className="ms-1 rounded bg-amber-400/15 px-1 py-0.5 text-[9px] font-semibold uppercase text-amber-300"
            title={t(strings.messageJumpList.summarizedBadge)}
          >
            S
          </span>
        )}
        {isNext && (
          <span
            data-testid={`msg-jump-next-${event.id}`}
            className="ml-auto inline-flex items-center gap-0.5 text-[10px] font-medium text-amber-300"
          >
            {t(strings.messageJumpList.nextBadge)}
          </span>
        )}
      </span>
      <span className="line-clamp-2 w-full text-[12px] leading-snug text-wc-text-muted">
        <Highlight result={result} />
      </span>
    </button>
  );
}

function FilterChip({
  id,
  label,
  active,
  onClick,
  ariaRole,
}: {
  id: string;
  label: string;
  active: boolean;
  onClick: () => void;
  ariaRole?: "radio";
}) {
  return (
    <button
      key={id}
      type="button"
      role={ariaRole}
      aria-checked={ariaRole === "radio" ? active : undefined}
      aria-pressed={ariaRole ? undefined : active}
      data-testid={`msg-nav-chip-${id}`}
      data-active={active}
      onClick={onClick}
      className={cn(
        "rounded-full px-3 py-1 text-[11px] font-medium transition",
        active
          ? "bg-wc-accent/25 text-wc-text-primary"
          : "bg-wc-surface-input/40 text-wc-text-muted hover:bg-wc-surface-input hover:text-wc-text-primary",
      )}
    >
      {label}
    </button>
  );
}

function OptionButton({
  testId,
  label,
  active,
  disabled,
  onClick,
}: {
  testId: string;
  label: string;
  active: boolean;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={active}
      data-testid={testId}
      data-active={active}
      disabled={disabled}
      onClick={onClick}
      className={cn(
        "rounded-md px-2 py-1 text-[11px] font-medium transition",
        active
          ? "bg-wc-accent/25 text-wc-text-primary"
          : "bg-wc-surface-input/40 text-wc-text-muted hover:bg-wc-surface-input hover:text-wc-text-primary",
        disabled && "cursor-not-allowed opacity-40 hover:bg-wc-surface-input/40",
      )}
    >
      {label}
    </button>
  );
}

/** Anchored placement order for the desktop panel opening above its trigger. */
const ABOVE_ANCHOR_PLACEMENTS: FloatingPlacement[] = ["top-end", "top-start", "bottom-end", "bottom-start"];

/** Stable stand-in so the position hook can run unconditionally without an anchor. */
const NULL_ANCHOR_REF: RefObject<HTMLElement | null> = { current: null };

export default function MessageJumpList({
  events,
  focusedEventId,
  onSelect,
  onClose,
  mode = "jump",
  initialFocus,
  query,
  onQueryChange,
  searchMatchCount,
  searchTruncated = false,
  desktopAnchorRef,
  currentTime = 0,
  duration = null,
  isPaused = true,
  isSummarized = false,
  onPause,
  onResume,
  onSeek,
  hasQueuedNext = false,
  exportSelection,
}: MessageJumpListProps) {
  const { t } = useTranslation();
  const isMobile = useMediaQuery("(max-width: 767px)");
  const listRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const desktopPanelRef = useRef<HTMLDivElement>(null);
  const anchoredStyle = useAnchoredPopoverPosition(
    Boolean(desktopAnchorRef) && !isMobile,
    desktopAnchorRef ?? NULL_ANCHOR_REF,
    desktopPanelRef,
    ABOVE_ANCHOR_PLACEMENTS,
  );

  const isControlledQuery = onQueryChange !== undefined;
  const [internalQuery, setInternalQuery] = useState(query ?? "");
  const q = isControlledQuery ? (query ?? "") : internalQuery;
  const setQuery = useCallback(
    (next: string) => {
      if (isControlledQuery) onQueryChange?.(next);
      else setInternalQuery(next);
    },
    [isControlledQuery, onQueryChange],
  );

  const [role, setRole] = useState<RoleFilter>("all");
  const [status, setStatus] = useState<StatusFilter>("all");
  const [content, setContent] = useState<ContentFilter>("all");
  const [sort, setSort] = useState<SortMode>("oldest");
  const [groupMode, setGroupMode] = useState<GroupMode>("turn");
  const [showAdvanced, setShowAdvanced] = useState(false);

  // Export selection is only reachable from normal jump mode; the selected-ID
  // set itself lives in MessagesPane so it survives filter changes and close.
  const canExport = mode === "jump" && exportSelection !== undefined;
  const [isExportSelecting, setIsExportSelecting] = useState(false);
  const exportActive = canExport && isExportSelecting;

  const navState = useMemo<NavigatorState>(
    () => ({ query: q, role, status, content, sort, group: groupMode }),
    [q, role, status, content, sort, groupMode],
  );

  const results = useMemo(() => buildResults(events, navState), [events, navState]);
  const groups = useMemo(() => groupResults(results, groupMode), [results, groupMode]);
  const navigatorRows = useMemo(() => groups.flatMap((group) => {
    const rows: Array<{ type: "header"; id: string; label: "user" | "assistant" } | { type: "event"; result: NavigatorResult }> = [];
    if (group.roleLabel) rows.push({ type: "header", id: group.id, label: group.roleLabel });
    if (group.leadUser) rows.push({ type: "event", result: group.leadUser });
    for (const result of group.items) rows.push({ type: "event", result });
    return rows;
  }), [groups]);
  const navigatorIndexByEventId = useMemo(() => new Map(
    navigatorRows.flatMap((row, index) => row.type === "event" ? [[row.result.event.id, index] as const] : []),
  ), [navigatorRows]);
  const { registerItem: registerNavigatorItem, totalSize: navigatorTotalSize, virtualItems: navigatorVirtualItems, scrollToIndex: scrollNavigatorToIndex } = useVirtualList({
    count: navigatorRows.length,
    estimateSize: (index) => navigatorRows[index]?.type === "header" ? 26 : 64,
    overscan: 6,
    scrollElementRef: listRef,
    enabled: navigatorRows.length > 40,
  });
  const sources = useMemo(() => availableSources(events), [events]);
  const reason = useMemo(() => noResultReason(events.length, navState), [events.length, navState]);

  // Snapshot "now" once per result change; relative-time labels are stable
  // while the panel is open. No live tick needed.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const now = useMemo(() => new Date(), [results]);

  const focusedIndex = useMemo(
    () => (focusedEventId ? results.findIndex((r) => r.event.id === focusedEventId) : -1),
    [results, focusedEventId],
  );
  const [activeIndex, setActiveIndex] = useState(0);
  useEffect(() => {
    setActiveIndex(focusedIndex >= 0 ? focusedIndex : 0);
  }, [focusedIndex, results.length]);
  const safeActive = results.length === 0 ? -1 : Math.min(activeIndex, results.length - 1);
  const activeId = safeActive >= 0 ? results[safeActive]?.event.id ?? null : null;

  const focusedEvent = useMemo(() => {
    if (!focusedEventId) return null;
    return events.find((e) => e.id === focusedEventId) ?? null;
  }, [events, focusedEventId]);

  const nextEventId = useMemo<string | null>(() => {
    if (!hasQueuedNext || !focusedEventId) return null;
    const idx = events.findIndex((e) => e.id === focusedEventId);
    if (idx < 0 || idx >= events.length - 1) return null;
    return events[idx + 1]?.id ?? null;
  }, [events, focusedEventId, hasQueuedNext]);

  // Scroll the active row into view by adjusting ONLY the navigator's own
  // scroll container — never scrollIntoView, which can scroll the host
  // document/window in iframe/proxy embeddings (spatial-navigation hazard).
  const scrollToEvent = useCallback((eventId: string, smooth: boolean) => {
    const index = navigatorIndexByEventId.get(eventId);
    if (index != null) scrollNavigatorToIndex(index, smooth ? "smooth" : "auto", "center");
  }, [navigatorIndexByEventId, scrollNavigatorToIndex]);

  // Scroll focused event into view when it changes (not on every render).
  const lastScrolledId = useRef<string | null>(null);
  useEffect(() => {
    if (focusedEventId && focusedEventId !== lastScrolledId.current) {
      lastScrolledId.current = focusedEventId;
      const id = focusedEventId;
      requestAnimationFrame(() => scrollToEvent(id, false));
    }
  }, [focusedEventId, scrollToEvent, results]);

  // Auto-focus the search input on open (desktop only — on mobile the virtual
  // keyboard would cover results before the user has chosen to search).
  const wantsSearchFocus = (initialFocus ?? (mode === "playback-select" ? "list" : "search")) === "search";
  useEffect(() => {
    if (!wantsSearchFocus || isMobile) return;
    const id = setTimeout(() => searchRef.current?.focus(), 50);
    return () => clearTimeout(id);
  }, [wantsSearchFocus, isMobile]);

  const jumpToCurrent = useCallback(() => {
    if (focusedEventId) scrollToEvent(focusedEventId, true);
  }, [focusedEventId, scrollToEvent]);

  const handleSelect = useCallback(
    (eventId: string) => {
      if (exportActive) {
        exportSelection?.onToggle(eventId);
        return;
      }
      onSelect(eventId);
      onClose();
    },
    [exportActive, exportSelection, onSelect, onClose],
  );

  // --- Export selection derived state -------------------------------------
  const selectedIds = exportSelection?.selectedIds;
  const selectedEvents = useMemo(
    () => (selectedIds ? events.filter((e) => selectedIds.has(e.id)) : []),
    [events, selectedIds],
  );
  // Shared formatter is the single token-estimate authority; the footer shows
  // the default-format estimate the drawer will open with.
  const exportTokenEstimate = useMemo(
    () => (exportActive ? buildMessageExport(selectedEvents, DEFAULT_MESSAGE_EXPORT_FORMAT).tokenEstimate : 0),
    [exportActive, selectedEvents],
  );
  const visibleSelectedCount = useMemo(() => {
    if (!selectedIds) return 0;
    let count = 0;
    for (const r of results) {
      if (selectedIds.has(r.event.id)) count += 1;
    }
    return count;
  }, [results, selectedIds]);
  const hiddenSelectedCount = (selectedIds?.size ?? 0) - visibleSelectedCount;

  const exitExportSelection = useCallback(() => setIsExportSelecting(false), []);

  const moveActive = useCallback(
    (delta: number) => {
      if (results.length === 0) return;
      const base = safeActive < 0 ? 0 : safeActive;
      let next = base + delta;
      if (next < 0) next = results.length - 1;
      if (next > results.length - 1) next = 0;
      setActiveIndex(next);
      const targetId = results[next]?.event.id;
      if (targetId) scrollToEvent(targetId, false);
    },
    [results, safeActive, scrollToEvent],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        // In selection mode, Escape steps back to the normal navigator
        // (selection is retained by the parent) instead of closing outright.
        if (exportActive) exitExportSelection();
        else onClose();
        return;
      }
      if (e.key === "ArrowDown") {
        e.preventDefault();
        moveActive(1);
        return;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        moveActive(-1);
        return;
      }
      if ((e.key === "Enter" || (e.key === " " && exportActive)) && safeActive >= 0) {
        e.preventDefault();
        const result = results[safeActive];
        if (result) handleSelect(result.event.id);
      }
    },
    [moveActive, onClose, results, safeActive, handleSelect, exportActive, exitExportSelection],
  );

  const handleSearchKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      // Handled keys must not bubble to the container's onKeyDown, which would
      // otherwise double-process them (e.g. close on Escape while clearing).
      if (e.key === "ArrowDown") {
        e.preventDefault();
        e.stopPropagation();
        if (results.length > 0) {
          setActiveIndex(0);
          const id = results[0]?.event.id;
          if (id) scrollToEvent(id, false);
          listRef.current?.focus();
        }
        return;
      }
      if (e.key === "Enter" && safeActive >= 0) {
        e.preventDefault();
        e.stopPropagation();
        const result = results[safeActive];
        if (result) handleSelect(result.event.id);
        return;
      }
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopPropagation();
        if (q) setQuery("");
        else if (exportActive) exitExportSelection();
        else onClose();
      }
    },
    [results, safeActive, scrollToEvent, handleSelect, q, setQuery, onClose, exportActive, exitExportSelection],
  );

  const resetPrimaryFilters = useCallback(() => {
    setRole("all");
    setStatus("all");
    setContent("all");
  }, []);

  const allActive = role === "all" && status === "all" && content === "all";
  const title = exportActive
    ? t(strings.messageExport.selectionTitle)
    : mode === "playback-select"
      ? t(strings.messageJumpList.titlePlayback)
      : t(strings.messageJumpList.titleJump);

  const renderRow = (result: NavigatorResult, index: number) => (
    <div key={result.event.id} ref={(node) => registerNavigatorItem(index, node)}>
      <NavRow
        result={result}
        isFocused={!exportActive && result.event.id === focusedEventId}
        isActive={result.event.id === activeId}
        isNext={!exportActive && result.event.id === nextEventId}
        onSelect={() => handleSelect(result.event.id)}
        now={now}
        selectMode={exportActive}
        isSelected={exportActive && (selectedIds?.has(result.event.id) ?? false)}
      />
    </div>
  );

  // The export affordance is chrome, not content: on mobile it rides the
  // sheet's header slot, on desktop the anchored panel's own title row.
  const exportAction =
    canExport && !exportActive ? (
      <button
        type="button"
        data-testid="msg-export-enter"
        onClick={() => setIsExportSelecting(true)}
        className="inline-flex min-h-[32px] items-center gap-1 rounded-full bg-wc-surface-input/40 px-2.5 py-1 text-[11px] font-medium text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary"
      >
        <ClipboardCopy className="h-3.5 w-3.5" aria-hidden="true" />
        {t(strings.messageExport.exportAction)}
      </button>
    ) : null;

  const body_node = (
    <>

      {(mode === "playback-select" || duration !== null) && !exportActive && (
        <NowPlayingHeader
          event={focusedEvent}
          currentTime={currentTime}
          duration={duration}
          isPaused={isPaused}
          isSummarized={isSummarized}
          onPause={onPause}
          onResume={onResume}
          onSeek={onSeek}
          now={now}
          onJumpToCurrent={jumpToCurrent}
        />
      )}

      {/* Search */}
      <div className="flex shrink-0 items-center gap-2 px-3 pt-2 pb-1.5">
        <Search className="h-4 w-4 shrink-0 text-wc-text-muted" aria-hidden="true" />
        <input
          ref={searchRef}
          data-testid="msg-nav-search"
          type="text"
          value={q}
          onChange={(e) => setQuery(e.target.value)}
          // The overlay wrapper calls preventDefault on mousedown to keep focus
          // off the host (terminal); that also blocks the browser from focusing
          // this input on click. Stop the event here so the input focuses
          // normally while the rest of the overlay keeps its behavior.
          onMouseDown={(e) => e.stopPropagation()}
          onKeyDown={handleSearchKeyDown}
          placeholder={t(strings.messageJumpList.searchPlaceholder)}
          aria-label={t(strings.messageJumpList.searchAriaLabel)}
          className="min-w-0 flex-1 bg-transparent text-sm text-wc-text-primary placeholder:text-wc-text-muted outline-none"
        />
        {q && (
          <button
            type="button"
            data-testid="msg-nav-clear"
            onClick={() => {
              setQuery("");
              searchRef.current?.focus();
            }}
            className="rounded p-1 text-wc-text-secondary transition hover:bg-wc-surface-input hover:text-wc-text-primary"
            aria-label={t(strings.messageJumpList.clearSearch)}
          >
            <X className="h-3.5 w-3.5" />
          </button>
        )}
      </div>

      {/* Result count */}
      <div data-testid="msg-nav-count" className="shrink-0 px-3 pb-1 text-[10px] text-wc-text-faint">
        {t(strings.messageJumpList.resultCount, { count: results.length })}
      </div>

      {/* Primary chips */}
      <div
        data-testid="msg-jump-filters"
        role="group"
        aria-label={t(strings.messageJumpList.filterAriaLabel)}
        className="flex shrink-0 flex-wrap items-center gap-1 px-3 pb-2"
      >
        <FilterChip id="all" label={t(strings.messageJumpList.filterAll)} active={allActive} onClick={resetPrimaryFilters} />
        <FilterChip id="user" label={t(strings.messageJumpList.filterUser)} active={role === "user"} onClick={() => setRole(role === "user" ? "all" : "user")} />
        <FilterChip id="assistant" label={t(strings.messageJumpList.filterAssistant)} active={role === "assistant"} onClick={() => setRole(role === "assistant" ? "all" : "assistant")} />
        <FilterChip id="failed" label={t(strings.messageJumpList.filterFailed)} active={status === "failed"} onClick={() => setStatus(status === "failed" ? "all" : "failed")} />
        <FilterChip id="unheard" label={t(strings.messageJumpList.filterUnheard)} active={status === "unheard"} onClick={() => setStatus(status === "unheard" ? "all" : "unheard")} />
        <button
          type="button"
          data-testid="msg-nav-more"
          data-active={showAdvanced}
          aria-expanded={showAdvanced}
          onClick={() => setShowAdvanced((v) => !v)}
          className={cn(
            "rounded-full px-3 py-1 text-[11px] font-medium transition",
            showAdvanced ? "bg-wc-accent/25 text-wc-text-primary" : "bg-wc-surface-input/40 text-wc-text-muted hover:bg-wc-surface-input hover:text-wc-text-primary",
          )}
        >
          {showAdvanced ? t(strings.messageJumpList.filterLess) : t(strings.messageJumpList.filterMore)}
        </button>
      </div>

      {/* Advanced panel */}
      {showAdvanced && (
        <div
          data-testid="msg-nav-advanced"
          className="shrink-0 space-y-2 border-y border-wc-default/60 bg-wc-surface-base/40 px-3 py-2"
        >
          {sources.length > 0 && (
            <div>
              <div className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-wc-text-faint">{t(strings.messageJumpList.sourceHeading)}</div>
              <div role="radiogroup" aria-label={t(strings.messageJumpList.sourceHeading)} className="flex flex-wrap gap-1">
                {sources.map((src) => {
                  const value: RoleFilter = `source:${src}`;
                  return (
                    <OptionButton
                      key={src}
                      testId={`msg-nav-source-${src}`}
                      label={t(SOURCE_LABEL_KEY[src])}
                      active={role === value}
                      onClick={() => setRole(role === value ? "all" : value)}
                    />
                  );
                })}
              </div>
            </div>
          )}

          <div>
            <div className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-wc-text-faint">{t(strings.messageJumpList.statusHeading)}</div>
            <div role="radiogroup" aria-label={t(strings.messageJumpList.statusHeading)} className="flex flex-wrap gap-1">
              {STATUS_OPTIONS.map((opt) => (
                <OptionButton key={opt} testId={`msg-nav-status-${opt}`} label={t(STATUS_LABEL_KEY[opt])} active={status === opt} onClick={() => setStatus(opt)} />
              ))}
            </div>
          </div>

          <div>
            <div className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-wc-text-faint">{t(strings.messageJumpList.contentHeading)}</div>
            <div role="radiogroup" aria-label={t(strings.messageJumpList.contentHeading)} className="flex flex-wrap gap-1">
              {CONTENT_OPTIONS.map((opt) => (
                <OptionButton key={opt} testId={`msg-nav-content-${opt}`} label={t(CONTENT_LABEL_KEY[opt])} active={content === opt} onClick={() => setContent(opt)} />
              ))}
            </div>
          </div>

          <div>
            <div className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-wc-text-faint">{t(strings.messageJumpList.sortHeading)}</div>
            <div role="radiogroup" aria-label={t(strings.messageJumpList.sortHeading)} className="flex flex-wrap gap-1">
              {SORT_OPTIONS.map((opt) => (
                <OptionButton
                  key={opt}
                  testId={`msg-nav-sort-${opt}`}
                  label={t(SORT_LABEL_KEY[opt])}
                  active={sort === opt}
                  disabled={opt === "relevance" && !q}
                  onClick={() => setSort(opt)}
                />
              ))}
            </div>
          </div>

          <div>
            <div className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-wc-text-faint">{t(strings.messageJumpList.groupHeading)}</div>
            <div role="radiogroup" aria-label={t(strings.messageJumpList.groupHeading)} className="flex flex-wrap gap-1">
              {GROUP_OPTIONS.map((opt) => (
                <OptionButton key={opt} testId={`msg-nav-group-${opt}`} label={t(GROUP_LABEL_KEY[opt])} active={groupMode === opt} onClick={() => setGroupMode(opt)} />
              ))}
            </div>
          </div>
        </div>
      )}

      {results.length === 0 ? (
        <div data-testid="msg-nav-empty" data-reason={reason} className="px-3 py-8 text-center text-xs text-wc-text-faint">
          {t(strings.messageJumpList[reason])}
        </div>
      ) : (
        <div
          ref={listRef}
          data-testid="msg-jump-scroll"
          tabIndex={-1}
          className="flex-1 space-y-1 overflow-y-auto px-2 pb-[max(0.5rem,var(--wc-safe-bottom,0px))] pt-1 outline-none"
        >
          <div style={{ height: navigatorTotalSize, position: "relative" }}>
            {navigatorVirtualItems.map((item) => {
              const row = navigatorRows[item.index];
              if (!row) return null;
              return (
                <div key={row.type === "header" ? row.id : row.result.event.id} style={{ position: "absolute", top: item.start, left: 0, right: 0 }}>
                  {row.type === "header" ? (
                    <div ref={(node) => registerNavigatorItem(item.index, node)} className="px-2 pt-1 text-[10px] font-semibold uppercase tracking-wider text-wc-text-faint">
                      {row.label === "user" ? t(strings.messageJumpList.roleYou) : t(strings.messageJumpList.roleAssistant)}
                    </div>
                  ) : renderRow(row.result, item.index)}
                </div>
              );
            })}
          </div>
          <div data-testid="msg-jump-safe-spacer" aria-hidden="true" style={{ height: "var(--wc-safe-bottom, 0px)" }} />
        </div>
      )}

      {q && searchMatchCount !== undefined && (
        <div data-testid="msg-nav-server-search-count" className="shrink-0 border-t border-wc-default/60 px-3 py-1 text-[10px] text-wc-text-faint">
          {searchMatchCount} {searchTruncated ? "+" : ""} {t(strings.messageJumpList.noSearchResults)}
        </div>
      )}

      {exportActive && exportSelection && (
        <div
          data-testid="msg-export-footer"
          className="shrink-0 border-t border-wc-default/60 bg-wc-surface-raised px-3 pt-2 pb-[max(0.75rem,var(--wc-safe-bottom,0px))]"
        >
          <div className="flex flex-wrap items-center gap-1 pb-2">
            <button
              type="button"
              data-testid="msg-export-select-all"
              onClick={exportSelection.onSelectAll}
              className="min-h-[32px] rounded-md bg-wc-surface-input/40 px-2 py-1 text-[11px] font-medium text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary"
            >
              {t(strings.messageExport.selectAll)}
            </button>
            <button
              type="button"
              data-testid="msg-export-select-visible"
              onClick={() => exportSelection.onSelectVisible(results.map((r) => r.event.id))}
              className="min-h-[32px] rounded-md bg-wc-surface-input/40 px-2 py-1 text-[11px] font-medium text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary"
            >
              {t(strings.messageExport.selectVisible)}
            </button>
            <button
              type="button"
              data-testid="msg-export-clear"
              onClick={exportSelection.onClear}
              disabled={(selectedIds?.size ?? 0) === 0}
              className="min-h-[32px] rounded-md bg-wc-surface-input/40 px-2 py-1 text-[11px] font-medium text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary disabled:cursor-not-allowed disabled:opacity-40"
            >
              {t(strings.messageExport.clearSelection)}
            </button>
          </div>
          {hiddenSelectedCount > 0 && (
            <div data-testid="msg-export-hidden-hint" className="pb-2 text-[10px] text-wc-text-faint">
              {t(strings.messageExport.hiddenSelected, { count: hiddenSelectedCount })}
            </div>
          )}
          <div className="flex items-center gap-2">
            <span data-testid="msg-export-count" className="text-[11px] font-medium text-wc-text-primary">
              {t(strings.messageExport.selectedCount, { count: selectedIds?.size ?? 0 })}
            </span>
            <span data-testid="msg-export-tokens" className="text-[10px] text-wc-text-faint">
              {t(strings.messageExport.approxTokens, { count: exportTokenEstimate })}
            </span>
            <span className="flex-1" />
            <button
              type="button"
              data-testid="msg-export-cancel"
              onClick={exitExportSelection}
              className="min-h-[36px] rounded-lg px-3 py-1.5 text-xs font-medium text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary"
            >
              {t(strings.messageExport.cancelAction)}
            </button>
            <button
              type="button"
              data-testid="msg-export-continue"
              onClick={exportSelection.onContinue}
              disabled={(selectedIds?.size ?? 0) === 0}
              className="min-h-[36px] rounded-lg bg-wc-accent/25 px-3 py-1.5 text-xs font-semibold text-wc-text-primary transition hover:bg-wc-accent/35 disabled:cursor-not-allowed disabled:opacity-40"
            >
              {t(strings.messageExport.continueAction)}
            </button>
          </div>
        </div>
      )}
    </>
  );

  // On mobile the sheet is the library's: grabber, swipe-to-dismiss, backdrop,
  // Escape, focus containment, scroll lock and safe-area all arrive with it,
  // replacing the decorative bar this file used to draw. The close button
  // stays off — the grabber is the dismiss affordance, and offering both says
  // the gesture might not work.
  if (isMobile) {
    return (
      <BottomSheet
        open
        onClose={onClose}
        title={title}
        headerActions={exportAction}
        closeLabel={t(strings.messageJumpList.closeAriaLabel)}
        testId="msg-jump-list"
        avoidKeyboard
      >
        <div tabIndex={0} onKeyDown={handleKeyDown} className="flex flex-col">
          {body_node}
        </div>
      </BottomSheet>
    );
  }

  // Desktop stays an anchored panel: it has no dismiss gesture, so it keeps an
  // explicit close control.
  const content_node = (
    <div
      data-testid="msg-jump-list"
      tabIndex={0}
      onKeyDown={handleKeyDown}
      className="wc-stable-theme flex max-h-[32rem] w-[22rem] flex-col overflow-hidden rounded-xl border border-wc-default bg-wc-surface-raised shadow-2xl"
    >
      {/* Title + export + close */}
      <div className="flex shrink-0 items-center justify-between gap-2 px-3 pt-1 pb-1">
        <span className="text-[11px] font-medium uppercase tracking-wider text-wc-text-faint">{title}</span>
        <span className="flex items-center gap-1">
          {canExport && !exportActive && (
            <button
              type="button"
              data-testid="msg-export-enter"
              onClick={() => setIsExportSelecting(true)}
              className="inline-flex min-h-[32px] items-center gap-1 rounded-full bg-wc-surface-input/40 px-2.5 py-1 text-[11px] font-medium text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary"
            >
              <ClipboardCopy className="h-3.5 w-3.5" aria-hidden="true" />
              {t(strings.messageExport.exportAction)}
            </button>
          )}
          <button
            onClick={onClose}
            className="rounded p-1 text-wc-text-secondary transition hover:bg-wc-surface-input hover:text-wc-text-primary"
            aria-label={t(strings.messageJumpList.closeAriaLabel)}
            type="button"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </span>
      </div>
      {body_node}
    </div>
  );


  return createPortal(
    <div className="fixed inset-0 z-wc-popover-backdrop" onMouseDown={(e) => e.preventDefault()}>
      <div className="absolute inset-0 bg-wc-backdrop" onClick={onClose} />
      <div
        ref={desktopPanelRef}
        className="absolute z-wc-popover"
        style={desktopAnchorRef ? anchoredStyle : { top: 48, right: 16 }}
      >
        {content_node}
      </div>
    </div>,
    document.body,
  );
}
