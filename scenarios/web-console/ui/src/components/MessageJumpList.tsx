import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import type { ConversationEvent } from "../api/conversation";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useVirtualList } from "../hooks/useVirtualList";
import { cn } from "../lib/classnames";

interface MessageJumpListProps {
  events: ConversationEvent[];
  focusedEventId: string | null;
  onSelect: (eventId: string) => void;
  onClose: () => void;
  desktopStyle?: CSSProperties;
}

/** Truncates text to maxLen characters, adding ellipsis if needed. */
function truncate(text: string, maxLen: number): string {
  const oneLine = text.replace(/\n/g, " ").trim();
  return oneLine.length > maxLen ? oneLine.slice(0, maxLen) + "…" : oneLine;
}

/**
 * Dropdown (desktop) or bottom sheet (mobile) listing all messages
 * for random-access navigation.
 */
export default function MessageJumpList({
  events,
  focusedEventId,
  onSelect,
  onClose,
  desktopStyle,
}: MessageJumpListProps) {
  const isMobile = useMediaQuery("(max-width: 767px)");
  const listRef = useRef<HTMLDivElement>(null);
  const focusedIndex = useMemo(
    () => (focusedEventId ? events.findIndex((event) => event.id === focusedEventId) : -1),
    [events, focusedEventId],
  );
  const [activeIndex, setActiveIndex] = useState(focusedIndex >= 0 ? focusedIndex : 0);
  const { registerItem, scrollToIndex, totalSize, virtualItems } = useVirtualList({
    count: events.length,
    estimateSize: () => 38,
    overscan: 8,
    scrollElementRef: listRef,
    enabled: events.length > 40,
  });

  useEffect(() => {
    setActiveIndex(focusedIndex >= 0 ? focusedIndex : 0);
  }, [focusedIndex]);

  useEffect(() => {
    if (focusedIndex >= 0) {
      scrollToIndex(focusedIndex, "auto", "center");
    }
  }, [focusedIndex, scrollToIndex]);

  // Keyboard navigation
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        onClose();
        return;
      }
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        e.preventDefault();
        if (events.length === 0) return;
        let nextIdx: number;
        if (e.key === "ArrowDown") {
          nextIdx = activeIndex < events.length - 1 ? activeIndex + 1 : 0;
        } else {
          nextIdx = activeIndex > 0 ? activeIndex - 1 : events.length - 1;
        }
        setActiveIndex(nextIdx);
        scrollToIndex(nextIdx, "auto", "center");
        return;
      }
      if (e.key === "Enter" && events[activeIndex]) {
        onSelect(events[activeIndex].id);
        onClose();
      }
    },
    [activeIndex, events, onClose, onSelect, scrollToIndex],
  );

  const content = (
    <div
      ref={listRef}
      data-testid="msg-jump-list"
      tabIndex={0}
      className={cn(
        "flex flex-col overflow-y-auto",
        isMobile
          ? "max-h-[60vh] rounded-t-[20px] border-t border-wc-default bg-wc-surface-raised p-3 pb-[max(0.75rem,var(--wc-safe-bottom))] shadow-2xl"
          : "max-h-80 w-72 rounded-xl border border-wc-default bg-wc-surface-raised p-2 shadow-lg",
      )}
      onKeyDown={handleKeyDown}
    >
      {/* Header */}
      <div className="flex items-center justify-between px-2 pb-2">
        <span className="text-xs font-medium uppercase tracking-wider text-wc-text-faint">
          Jump to message
        </span>
        <button
          onClick={onClose}
          className="rounded p-1 text-wc-text-secondary hover:bg-wc-surface-input hover:text-wc-text-primary transition"
          aria-label="Close jump list"
          type="button"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>

      {/* Message list */}
      <div className="relative" style={{ height: totalSize > 0 ? `${totalSize}px` : undefined }}>
      {virtualItems.map(({ index, start }) => {
        const event = events[index];
        if (!event) return null;
        const isUser = event.role === "user";
        const isFocused = event.id === focusedEventId;
        const roleLabel = isUser ? "You" : event.source === "claude_hook" ? "Claude" : "Codex";

        return (
          <button
            key={event.id}
            ref={(node) => registerItem(index, node)}
            data-testid={`msg-jump-item-${event.id}`}
            data-jump-item
            onClick={() => { onSelect(event.id); onClose(); }}
            className={cn(
              "absolute left-0 right-0 flex items-start gap-2 rounded-lg px-2 py-1.5 text-left text-xs transition",
              isFocused
                ? "bg-wc-accent/15 text-wc-text-primary"
                : "text-wc-text-secondary hover:bg-wc-surface-input hover:text-wc-text-primary",
              index === activeIndex && !isFocused && "ring-1 ring-wc-accent/30",
            )}
            style={{ top: `${start}px` }}
            type="button"
          >
            <span className="shrink-0 font-mono text-wc-text-faint">#{event.sequence}</span>
            <span className="shrink-0 font-medium" style={{ minWidth: "3rem" }}>{roleLabel}</span>
            <span className="min-w-0 truncate text-wc-text-muted">{truncate(event.text, 50)}</span>
          </button>
        );
      })}
      </div>

      {events.length === 0 && (
        <div className="px-2 py-4 text-center text-xs text-wc-text-faint">No messages</div>
      )}
    </div>
  );

  return createPortal(
    <div className="fixed inset-0 z-40" onMouseDown={(e) => e.preventDefault()}>
      <div className="absolute inset-0 bg-wc-backdrop" onClick={onClose} />
      {isMobile ? (
        <div className="absolute bottom-0 left-0 right-0 z-50">
          {/* Drag handle */}
          <div className="flex justify-center pt-2 pb-1 rounded-t-[20px] bg-wc-surface-raised">
            <div className="h-1 w-8 rounded-full bg-wc-text-muted/40" />
          </div>
          {content}
        </div>
      ) : (
        <div
          className="absolute z-50"
          style={desktopStyle ?? { top: 48, right: 16 }}
        >
          {content}
        </div>
      )}
    </div>,
    document.body,
  );
}
