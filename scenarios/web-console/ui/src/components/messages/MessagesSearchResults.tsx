import { useEffect, useLayoutEffect, useRef, useState, type KeyboardEvent as ReactKeyboardEvent, type RefObject } from "react";
import { createPortal } from "react-dom";
import { CornerDownLeft, Loader2, Search } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { eventTime } from "./speaker";
import { assistantRoleLabelKey, statusGlyphFor, type NavigatorResult } from "../MessageJumpList.helpers";
import { Skeleton } from "@vrooli/react-component-library/Skeleton/1";
import { isHumanMessage } from "../../lib/conversationRows";
import { workKindLabel } from "./workSummary";

interface MessagesSearchResultsProps {
  open: boolean;
  anchorRef: RefObject<HTMLElement | null>;
  onClose: () => void;
  query: string;
  results: readonly NavigatorResult[];
  /** The whole-history search is still in flight. */
  loading: boolean;
  /** Local matches are shown while the whole-history search finishes. */
  searchingAll?: boolean;
  error?: string;
  truncated: boolean;
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => void;
  activeIndex: number;
  onActiveIndexChange: (index: number) => void;
  onSelect: (eventId: string) => void;
  /** Total messages in the conversation, for the empty/query framing. */
  totalCount: number;
}

const PANEL_PLACEMENT_GAP = 6;

function Highlight({ result }: { result: NavigatorResult }) {
  return (
    <>
      {result.excerpt.map((segment, index) => segment.match ? (
        <mark key={index} className="rounded-[2px] bg-wc-accent/30 text-wc-text-primary">{segment.text}</mark>
      ) : (
        <span key={index}>{segment.text}</span>
      ))}
    </>
  );
}

function StatusGlyph({ glyph }: { glyph: ReturnType<typeof statusGlyphFor>["glyph"] }) {
  if (glyph === "playing") return <span aria-hidden className="h-2 w-2 shrink-0 animate-pulse rounded-full bg-wc-accent" />;
  if (glyph === "played") return <span aria-hidden className="shrink-0 text-emerald-400">✓</span>;
  if (glyph === "failed") return <span aria-hidden className="shrink-0 text-red-400">✗</span>;
  return <span aria-hidden className="shrink-0 text-wc-text-faint/70">○</span>;
}

/**
 * The messages toolbar's search results.
 *
 * This is a standard anchored result list, not a second navigation surface:
 * the field keeps the query, the list shows matches with their server-reported
 * highlight ranges, and choosing one closes the list and moves the conversation
 * to that message. Filters and sort are the toolbar's, so what is listed here
 * is exactly what the list itself would show.
 */
export function MessagesSearchResults({
  open,
  anchorRef,
  onClose,
  query,
  results,
  loading,
  searchingAll = false,
  error,
  truncated,
  hasMore,
  loadingMore,
  onLoadMore,
  activeIndex,
  onActiveIndexChange,
  onSelect,
  totalCount,
}: MessagesSearchResultsProps) {
  const { t } = useTranslation();
  const panelRef = useRef<HTMLDivElement | null>(null);
  const listRef = useRef<HTMLDivElement | null>(null);
  const [box, setBox] = useState<{ left: number; top?: number; bottom?: number; width: number; maxHeight: number } | null>(null);

  const trimmedQuery = query.trim();
  const heading = trimmedQuery
    ? t("messagesPane.searchResultsHeading", { query: trimmedQuery, defaultValue: "Results for ‘{{query}}’" })
    : t("messagesPane.searchRecentHeading", { defaultValue: "Recent messages" });

  useLayoutEffect(() => {
    if (!open) {
      setBox(null);
      return;
    }
    const measure = () => {
      const anchor = anchorRef.current;
      if (!anchor) return;
      const rect = anchor.getBoundingClientRect();
      const available = Math.max(240, window.innerWidth - 24);
      const width = Math.max(240, Math.min(Math.max(rect.width, 320), 672, available));
      const left = Math.min(Math.max(rect.left, 12), Math.max(12, window.innerWidth - width - 12));
      const spaceBelow = window.innerHeight - rect.bottom - 12;
      const spaceAbove = rect.top - 12;
      const above = spaceBelow < 240 && spaceAbove > spaceBelow;
      const maxHeight = Math.max(180, Math.min(448, above ? spaceAbove : spaceBelow));
      setBox(above
        ? { left, bottom: window.innerHeight - rect.top + PANEL_PLACEMENT_GAP, width, maxHeight }
        : { left, top: rect.bottom + PANEL_PLACEMENT_GAP, width, maxHeight });
    };
    measure();
    window.addEventListener("resize", measure);
    window.addEventListener("scroll", measure, true);
    return () => {
      window.removeEventListener("resize", measure);
      window.removeEventListener("scroll", measure, true);
    };
  }, [anchorRef, open]);

  useEffect(() => {
    if (!open) return undefined;
    const onPointerDown = (event: PointerEvent) => {
      const target = event.target as Node | null;
      if (!target) return;
      if (anchorRef.current?.contains(target) || panelRef.current?.contains(target)) return;
      onClose();
    };
    document.addEventListener("pointerdown", onPointerDown, true);
    return () => { document.removeEventListener("pointerdown", onPointerDown, true); };
  }, [anchorRef, onClose, open]);

  const safeActive = results.length === 0 ? -1 : Math.min(Math.max(activeIndex, 0), results.length - 1);
  useEffect(() => {
    if (!open || safeActive < 0) return;
    const list = listRef.current;
    const row = list?.querySelector<HTMLElement>('[data-search-result-active="true"]');
    if (!list || !row) return;
    const listRect = list.getBoundingClientRect();
    const rowRect = row.getBoundingClientRect();
    if (rowRect.top < listRect.top) list.scrollTop -= listRect.top - rowRect.top;
    else if (rowRect.bottom > listRect.bottom) list.scrollTop += rowRect.bottom - listRect.bottom;
  }, [open, safeActive]);

  if (!open || !box) return null;

  const onPanelKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    if (event.key === "Escape") {
      event.preventDefault();
      onClose();
      return;
    }
    if (event.key === "ArrowDown") {
      event.preventDefault();
      if (results.length > 0) onActiveIndexChange((safeActive + 1) % results.length);
      return;
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      if (results.length > 0) onActiveIndexChange((safeActive - 1 + results.length) % results.length);
      return;
    }
    if (event.key === "Enter" && safeActive >= 0) {
      event.preventDefault();
      const result = results[safeActive];
      if (result) onSelect(result.event.id);
    }
  };

  const showEmpty = !loading && !error && results.length === 0;

  return createPortal(
    <div
      ref={panelRef}
      id="messages-search-results"
      data-testid="messages-search-results"
      role="listbox"
      aria-label={t(strings.messageJumpList.searchAriaLabel)}
      aria-busy={loading || undefined}
      tabIndex={-1}
      onKeyDown={onPanelKeyDown}
      className="wc-stable-theme fixed z-wc-popover flex flex-col overflow-hidden rounded-xl border bg-wc-surface-raised shadow-2xl"
      style={{
        left: box.left,
        top: box.top,
        bottom: box.bottom,
        width: box.width,
        maxHeight: box.maxHeight,
        borderColor: "color-mix(in srgb, var(--color-border) 55%, transparent)",
      }}
    >
      <div className="flex shrink-0 items-center gap-2 border-b border-wc-default/60 px-3 py-2">
        <Search className="h-3.5 w-3.5 shrink-0 text-wc-text-faint" aria-hidden />
        <span className="min-w-0 flex-1 truncate text-[11px] font-semibold uppercase tracking-wider text-wc-text-muted">
          {heading}
        </span>
        <span data-testid="messages-search-count" className="shrink-0 text-[11px] tabular-nums text-wc-text-faint">
          {loading ? <Loader2 className="h-3.5 w-3.5 animate-spin motion-reduce:animate-none" aria-label={t("messagesPane.searching", { defaultValue: "Searching" })} /> : `${String(results.length)} ${results.length === 1 ? "result" : "results"}`}
        </span>
      </div>

      {error ? (
        <div role="alert" data-testid="messages-search-error" className="px-4 py-6 text-center text-xs text-red-300">
          {error}
        </div>
      ) : loading ? (
        <div role="status" aria-live="polite" className="space-y-2 p-3">
          {[0, 1, 2].map((row) => (
            <div key={row} className="rounded-lg border border-wc-default/50 p-3">
              <Skeleton aria-hidden="true" className="min-h-0 animate-pulse motion-reduce:animate-none h-2 w-24 rounded" />
              <Skeleton aria-hidden="true" className="min-h-0 animate-pulse motion-reduce:animate-none mt-2 h-2 w-full rounded" />
              <Skeleton aria-hidden="true" className="min-h-0 animate-pulse motion-reduce:animate-none mt-1.5 h-2 w-4/5 rounded" />
            </div>
          ))}
        </div>
      ) : showEmpty ? (
        <div data-testid="messages-search-empty" className="px-5 py-8 text-center">
          <p className="text-sm font-medium text-wc-text-primary">
            {trimmedQuery
              ? t("messagesPane.searchNoMatchTitle", { defaultValue: "No messages match this search" })
              : t("messagesPane.searchNoMessagesTitle", { defaultValue: "No messages to search yet" })}
          </p>
          <p className="mx-auto mt-1 max-w-xs text-xs leading-relaxed text-wc-text-faint">
            {trimmedQuery
              ? t("messagesPane.searchNoMatchBody", { defaultValue: "Try a different term, or clear a filter to widen the results." })
              : t("messagesPane.searchNoMessagesBody", { defaultValue: "Messages appear here as the agent conversation is captured." })}
          </p>
        </div>
      ) : (
        <div ref={listRef} className="min-h-0 flex-1 overflow-y-auto p-1" role="none">
          {results.map((result, index) => {
            const { event } = result;
            const active = index === safeActive;
            // A hit inside work (searching tool calls and context) is named
            // by what it is; otherwise by who wrote it.
            const workLabel = workKindLabel(event.kind);
            const isUser = workLabel == null && isHumanMessage(event);
            const clock = eventTime(event);
            const glyph = statusGlyphFor(event);
            return (
              <button
                key={event.id}
                id={`messages-search-result-${event.id}`}
                type="button"
                role="option"
                aria-selected={active}
                data-search-result-active={active || undefined}
                data-testid={`messages-search-result-${event.id}`}
                onMouseEnter={() => { onActiveIndexChange(index); }}
                onClick={() => { onSelect(event.id); }}
                className={cn(
                  "flex w-full flex-col items-start gap-0.5 rounded-lg px-3 py-2 text-start transition-colors",
                  active ? "bg-wc-accent/15 text-wc-text-primary" : "text-wc-text-secondary hover:bg-wc-surface-input/70",
                )}
              >
                <span className="flex w-full min-w-0 items-center gap-1.5 text-[11px]">
                  {!isUser && <StatusGlyph glyph={glyph.glyph} />}
                  <span className={cn("truncate font-medium", isUser ? "text-wc-text-primary" : "text-wc-text-secondary")}>
                    {workLabel ? t(workLabel as never) : isUser ? t(strings.messageJumpList.roleYou) : t(assistantRoleLabelKey(event.source))}
                  </span>
                  <span className="truncate text-wc-text-faint">· {clock.label ? `${clock.label} · ` : ""}{clock.timestamp}</span>
                  <span className="ms-auto shrink-0 font-mono text-[10px] text-wc-text-faint">#{String(event.sequence)}</span>
                </span>
                <span className="line-clamp-2 w-full text-[12px] leading-snug text-wc-text-muted">
                  <Highlight result={result} />
                </span>
              </button>
            );
          })}
        </div>
      )}

      {(truncated || hasMore) && !loading && !error && (
        <div data-testid="messages-search-truncated" className="flex shrink-0 items-center justify-between gap-2 border-t border-wc-default/60 px-3 py-1.5 text-[10px] text-wc-text-faint">
          <span>{t(strings.messageJumpList.truncated)}</span>
          {hasMore && (
            <button
              type="button"
              data-testid="messages-search-load-more"
              onClick={onLoadMore}
              disabled={loadingMore}
              className="min-h-9 shrink-0 rounded-md px-2 font-medium text-wc-accent transition hover:bg-wc-surface-input disabled:opacity-50"
            >
              {loadingMore ? "…" : t(strings.messageJumpList.loadMore)}
            </button>
          )}
        </div>
      )}

      {searchingAll && (
        <div data-testid="messages-search-searching-all" role="status" className="flex shrink-0 items-center gap-2 border-t border-wc-default/60 px-3 py-1.5 text-[10px] text-wc-text-faint">
          <Loader2 className="h-3 w-3 animate-spin motion-reduce:animate-none" aria-hidden />
          {t("messagesPane.searchingAll", { defaultValue: "Searching the full conversation…" })}
        </div>
      )}

      {!loading && !error && results.length > 0 && (
        <div className="hidden shrink-0 items-center gap-3 border-t border-wc-default/60 px-3 py-1.5 text-[10px] text-wc-text-faint sm:flex">
          <span className="inline-flex items-center gap-1"><CornerDownLeft className="h-3 w-3" aria-hidden /> {t("messagesPane.searchOpenHint", { defaultValue: "Open" })}</span>
          <span>↑ ↓ {t("messagesPane.searchNavigateHint", { defaultValue: "Navigate" })}</span>
          <span className="ms-auto">{t("messagesPane.searchScopeHint", { count: totalCount, defaultValue: "Searching {{count}} messages" })}</span>
        </div>
      )}

      {showEmpty && !trimmedQuery && totalCount > 0 && (
        <div className="shrink-0 border-t border-wc-default/60 px-3 py-1.5 text-[10px] text-wc-text-faint">
          {t("messagesPane.searchScopeHint", { count: totalCount, defaultValue: "Searching {{count}} messages" })}
        </div>
      )}
    </div>,
    document.body,
  );
}

export default MessagesSearchResults;
