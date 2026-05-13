import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type CSSProperties,
} from "react";
import { createPortal } from "react-dom";
import { Pause, Play, X } from "lucide-react";
import type { ConversationEvent } from "../api/conversation";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { cn } from "../lib/classnames";
import { getScrubClasses } from "./tts/scrubStyles";
import {
  applyFilter,
  formatRelativeTime,
  groupEventsByTurn,
  statusGlyphFor,
  stripMarkdown,
  type FilterMode,
  type StatusGlyph,
} from "./MessageJumpList.helpers";

interface MessageJumpListProps {
  events: ConversationEvent[];
  focusedEventId: string | null;
  onSelect: (eventId: string) => void;
  onClose: () => void;
  desktopStyle?: CSSProperties;
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
}

function truncate(text: string, maxLen: number): string {
  const oneLine = text.replace(/\n+/g, " ").trim();
  return oneLine.length > maxLen ? oneLine.slice(0, maxLen) + "…" : oneLine;
}

function preview(text: string, maxLen: number): string {
  return truncate(stripMarkdown(text), maxLen);
}

function StatusIcon({ glyph, className }: { glyph: StatusGlyph; className?: string }) {
  if (glyph === "playing") {
    return (
      <span
        aria-hidden="true"
        className={cn("inline-flex h-3 w-3 items-center justify-center", className)}
      >
        <span className="block h-2 w-2 animate-pulse rounded-full bg-wc-accent" />
      </span>
    );
  }
  if (glyph === "played") {
    return (
      <span
        aria-hidden="true"
        className={cn(
          "inline-flex h-3 w-3 items-center justify-center text-emerald-400",
          className,
        )}
      >
        ✓
      </span>
    );
  }
  if (glyph === "failed") {
    return (
      <span
        aria-hidden="true"
        className={cn(
          "inline-flex h-3 w-3 items-center justify-center text-red-400",
          className,
        )}
      >
        ✗
      </span>
    );
  }
  return (
    <span
      aria-hidden="true"
      className={cn(
        "inline-flex h-3 w-3 items-center justify-center text-wc-text-faint/70",
        className,
      )}
    >
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
  const handleScrub = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      onSeek?.(Number(e.target.value));
    },
    [onSeek],
  );

  if (!event || duration === null) {
    return (
      <div
        data-testid="msg-jump-now-playing"
        data-state="idle"
        className="px-3 py-2 text-[11px] text-wc-text-faint"
      >
        No active playback
      </div>
    );
  }

  const desc = statusGlyphFor(event);
  const roleLabel = event.role === "user" ? "You" : event.source === "claude_hook" ? "Claude" : "Codex";
  const seekable = duration > 0 && !!onSeek;

  return (
    <div data-testid="msg-jump-now-playing" data-state="playing" className="px-3 pt-2 pb-2">
      <div className="mb-1.5 flex items-center gap-2">
        <button
          type="button"
          data-testid="msg-jump-now-playpause"
          onClick={() => (isPaused ? onResume?.() : onPause?.())}
          className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-wc-surface-input text-wc-text-primary transition hover:bg-wc-accent/20"
          aria-label={isPaused ? "Resume" : "Pause"}
        >
          {isPaused ? <Play className="h-4 w-4" /> : <Pause className="h-4 w-4" />}
        </button>
        <button
          type="button"
          data-testid="msg-jump-now-jump"
          onClick={onJumpToCurrent}
          className="flex min-w-0 flex-1 items-center gap-1.5 rounded px-1 py-0.5 text-left transition hover:bg-wc-surface-input/60"
          aria-label="Scroll to currently playing message"
        >
          <StatusIcon glyph={desc.glyph} />
          <span className="font-mono text-[11px] text-wc-text-faint">#{event.sequence}</span>
          <span className="text-[11px] font-medium text-wc-text-primary">{roleLabel}</span>
          <span className="ml-1 truncate text-[11px] text-wc-text-muted">
            {preview(event.text, 60)}
          </span>
          <span className="ml-auto shrink-0 pl-1 text-[10px] text-wc-text-faint">
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
        aria-label="Seek currently playing message"
        className={getScrubClasses({
          isSummarized,
          enabled: seekable,
          extra: "w-full",
        })}
      />
    </div>
  );
}

function FilterChips({
  filter,
  onChange,
}: {
  filter: FilterMode;
  onChange: (next: FilterMode) => void;
}) {
  const chips: { id: FilterMode; label: string }[] = [
    { id: "all", label: "All" },
    { id: "unheard", label: "Unheard" },
    { id: "failed", label: "Failed" },
  ];
  return (
    <div
      data-testid="msg-jump-filters"
      role="radiogroup"
      aria-label="Filter messages"
      className="flex items-center gap-1 px-3 pb-2"
    >
      {chips.map((chip) => {
        const active = chip.id === filter;
        return (
          <button
            key={chip.id}
            type="button"
            role="radio"
            aria-checked={active}
            data-testid={`msg-jump-filter-${chip.id}`}
            data-active={active}
            onClick={() => onChange(chip.id)}
            className={cn(
              "rounded-full px-3 py-1 text-[11px] font-medium transition",
              active
                ? "bg-wc-accent/25 text-wc-text-primary"
                : "bg-wc-surface-input/40 text-wc-text-muted hover:bg-wc-surface-input hover:text-wc-text-primary",
            )}
          >
            {chip.label}
          </button>
        );
      })}
    </div>
  );
}

function UserTurnHeader({
  event,
  isFocused,
  isActive,
  onSelect,
  now,
}: {
  event: ConversationEvent;
  isFocused: boolean;
  isActive: boolean;
  onSelect: () => void;
  now: Date;
}) {
  return (
    <button
      type="button"
      data-testid={`msg-jump-item-${event.id}`}
      data-jump-item
      data-role="user"
      onClick={onSelect}
      className={cn(
        "flex min-h-[48px] w-full flex-col items-start gap-0.5 rounded-lg border px-3 py-2.5 text-left transition",
        isFocused
          ? "border-wc-accent/50 bg-wc-accent/15 text-wc-text-primary"
          : "border-wc-default/60 bg-wc-surface-input/40 text-wc-text-secondary hover:bg-wc-surface-input",
        isActive && !isFocused && "ring-1 ring-wc-accent/30",
      )}
    >
      <span className="flex w-full items-center gap-2 text-[11px]">
        <span className="font-semibold text-wc-text-primary">You</span>
        <span className="text-wc-text-faint">{formatRelativeTime(event.createdAt, now)}</span>
        <span className="ml-auto font-mono text-wc-text-faint">#{event.sequence}</span>
      </span>
      <span className="line-clamp-2 w-full text-[12px] leading-snug text-wc-text-muted">
        {preview(event.text, 160)}
      </span>
    </button>
  );
}

function AssistantRow({
  event,
  isFocused,
  isActive,
  isNext,
  onSelect,
  now,
}: {
  event: ConversationEvent;
  isFocused: boolean;
  isActive: boolean;
  isNext: boolean;
  onSelect: () => void;
  now: Date;
}) {
  const desc = statusGlyphFor(event);
  const roleLabel = event.source === "claude_hook" ? "Claude" : "Codex";
  return (
    <button
      type="button"
      data-testid={`msg-jump-item-${event.id}`}
      data-jump-item
      data-role="assistant"
      data-glyph={desc.glyph}
      aria-current={desc.glyph === "playing" ? "true" : undefined}
      onClick={onSelect}
      className={cn(
        "ml-4 flex min-h-[44px] w-auto flex-col items-start gap-0.5 rounded-lg px-2.5 py-2 text-left transition",
        isFocused
          ? "bg-wc-accent/15 text-wc-text-primary"
          : "text-wc-text-secondary hover:bg-wc-surface-input/70 hover:text-wc-text-primary",
        isActive && !isFocused && "ring-1 ring-wc-accent/30",
      )}
    >
      <span className="flex w-full items-center gap-1.5 text-[11px]">
        <StatusIcon glyph={desc.glyph} />
        <span className="font-mono text-wc-text-faint">#{event.sequence}</span>
        <span className="font-medium">{roleLabel}</span>
        <span className="text-wc-text-faint">· {formatRelativeTime(event.createdAt, now)}</span>
        {event.summarized && (
          <span
            data-testid={`msg-jump-summarized-${event.id}`}
            className="ml-1 rounded bg-amber-400/15 px-1 py-0.5 text-[9px] font-semibold uppercase text-amber-300"
            title="Summarized version"
          >
            S
          </span>
        )}
        {isNext && (
          <span
            data-testid={`msg-jump-next-${event.id}`}
            className="ml-auto inline-flex items-center gap-0.5 text-[10px] font-medium text-amber-300"
          >
            ⏭ next
          </span>
        )}
      </span>
      <span className="line-clamp-2 w-full pl-[18px] text-[12px] leading-snug text-wc-text-muted">
        {preview(event.text, 160)}
      </span>
    </button>
  );
}

export default function MessageJumpList({
  events,
  focusedEventId,
  onSelect,
  onClose,
  desktopStyle,
  currentTime = 0,
  duration = null,
  isPaused = true,
  isSummarized = false,
  onPause,
  onResume,
  onSeek,
  hasQueuedNext = false,
}: MessageJumpListProps) {
  const isMobile = useMediaQuery("(max-width: 767px)");
  const listRef = useRef<HTMLDivElement>(null);
  const itemRefs = useRef(new Map<string, HTMLElement>());
  const [filter, setFilter] = useState<FilterMode>("all");

  const visibleEvents = useMemo(() => applyFilter(events, filter), [events, filter]);

  // Snapshot "now" once per visible-event change; relative-time labels are
  // stable while the sheet is open. No live tick needed.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const now = useMemo(() => new Date(), [visibleEvents]);

  const focusedIndex = useMemo(
    () => (focusedEventId ? visibleEvents.findIndex((event) => event.id === focusedEventId) : -1),
    [visibleEvents, focusedEventId],
  );
  const [activeIndex, setActiveIndex] = useState(focusedIndex >= 0 ? focusedIndex : 0);

  useEffect(() => {
    setActiveIndex(focusedIndex >= 0 ? focusedIndex : 0);
  }, [focusedIndex]);

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

  const turns = useMemo(() => groupEventsByTurn(visibleEvents), [visibleEvents]);

  const registerItemRef = useCallback((eventId: string) => (node: HTMLElement | null) => {
    if (node) itemRefs.current.set(eventId, node);
    else itemRefs.current.delete(eventId);
  }, []);

  const scrollToEvent = useCallback((eventId: string, smooth: boolean) => {
    const node = itemRefs.current.get(eventId);
    if (!node || !listRef.current) return;
    node.scrollIntoView({ behavior: smooth ? "smooth" : "auto", block: "center" });
  }, []);

  // Scroll focused event into view when it changes (not on every render).
  const lastScrolledId = useRef<string | null>(null);
  useEffect(() => {
    if (focusedEventId && focusedEventId !== lastScrolledId.current) {
      lastScrolledId.current = focusedEventId;
      // Defer to allow refs to register on initial mount.
      const id = focusedEventId;
      requestAnimationFrame(() => scrollToEvent(id, false));
    }
  }, [focusedEventId, scrollToEvent, visibleEvents]);

  const jumpToCurrent = useCallback(() => {
    if (focusedEventId) scrollToEvent(focusedEventId, true);
  }, [focusedEventId, scrollToEvent]);

  const handleSelect = useCallback(
    (eventId: string) => {
      onSelect(eventId);
      onClose();
    },
    [onSelect, onClose],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        onClose();
        return;
      }
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        e.preventDefault();
        if (visibleEvents.length === 0) return;
        let nextIdx: number;
        if (e.key === "ArrowDown") {
          nextIdx = activeIndex < visibleEvents.length - 1 ? activeIndex + 1 : 0;
        } else {
          nextIdx = activeIndex > 0 ? activeIndex - 1 : visibleEvents.length - 1;
        }
        setActiveIndex(nextIdx);
        const targetId = visibleEvents[nextIdx]?.id;
        if (targetId) scrollToEvent(targetId, false);
        return;
      }
      if (e.key === "Enter" && visibleEvents[activeIndex]) {
        const ev = visibleEvents[activeIndex];
        if (ev) onSelect(ev.id);
        onClose();
      }
    },
    [activeIndex, visibleEvents, onClose, onSelect, scrollToEvent],
  );

  const content = (
    <div
      data-testid="msg-jump-list"
      tabIndex={0}
      onKeyDown={handleKeyDown}
      className={cn(
        "flex flex-col overflow-hidden rounded-t-[20px] border-t border-wc-default bg-wc-surface-raised shadow-2xl",
        isMobile ? "max-h-[75vh]" : "max-h-[28rem] w-80 rounded-xl border",
      )}
    >
      {/* Drag handle (mobile only) — visually part of the same card. */}
      {isMobile && (
        <div className="flex shrink-0 justify-center pt-2 pb-1">
          <div className="h-1 w-9 rounded-full bg-wc-text-muted/40" />
        </div>
      )}

      {/* Title + close */}
      <div className="flex shrink-0 items-center justify-between px-3 pt-1 pb-1">
        <span className="text-[11px] font-medium uppercase tracking-wider text-wc-text-faint">
          Jump to message
        </span>
        <button
          onClick={onClose}
          className="rounded p-1 text-wc-text-secondary transition hover:bg-wc-surface-input hover:text-wc-text-primary"
          aria-label="Close jump list"
          type="button"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>

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

      <FilterChips filter={filter} onChange={setFilter} />

      {visibleEvents.length === 0 ? (
        <div className="px-3 py-8 text-center text-xs text-wc-text-faint">
          {filter === "all" ? "No messages" : "No matching messages"}
        </div>
      ) : (
        <div
          ref={listRef}
          data-testid="msg-jump-scroll"
          className="flex-1 space-y-1 overflow-y-auto px-2 pb-[max(0.5rem,var(--wc-safe-bottom,0px))] pt-1"
        >
          {turns.map((turn, turnIdx) => (
            <div key={turn.user?.id ?? `lead-${turnIdx}`} className="space-y-1">
              {turn.user && (
                <div ref={registerItemRef(turn.user.id)}>
                  <UserTurnHeader
                    event={turn.user}
                    isFocused={turn.user.id === focusedEventId}
                    isActive={visibleEvents[activeIndex]?.id === turn.user.id}
                    onSelect={() => turn.user && handleSelect(turn.user.id)}
                    now={now}
                  />
                </div>
              )}
              {turn.assistants.map((event) => (
                <div key={event.id} ref={registerItemRef(event.id)}>
                  <AssistantRow
                    event={event}
                    isFocused={event.id === focusedEventId}
                    isActive={visibleEvents[activeIndex]?.id === event.id}
                    isNext={event.id === nextEventId}
                    onSelect={() => handleSelect(event.id)}
                    now={now}
                  />
                </div>
              ))}
            </div>
          ))}
          <div
            data-testid="msg-jump-safe-spacer"
            aria-hidden="true"
            style={{ height: "var(--wc-safe-bottom, 0px)" }}
          />
        </div>
      )}
    </div>
  );

  return createPortal(
    <div className="fixed inset-0 z-40" onMouseDown={(e) => e.preventDefault()}>
      <div className="absolute inset-0 bg-wc-backdrop" onClick={onClose} />
      {isMobile ? (
        <div className="absolute bottom-0 left-0 right-0 z-50">{content}</div>
      ) : (
        <div className="absolute z-50" style={desktopStyle ?? { top: 48, right: 16 }}>
          {content}
        </div>
      )}
    </div>,
    document.body,
  );
}
