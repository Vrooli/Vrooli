import { useEffect, useRef } from "react";
import { createPortal } from "react-dom";
import { ChevronDown, ChevronUp, Search, X } from "lucide-react";

interface MessagesSearchDrawerProps {
  open: boolean;
  onClose: () => void;
  query: string;
  onQueryChange: (query: string) => void;
  matchCount: number;
  /** 0-based index of the currently focused match, or -1 if no matches. */
  currentMatchIndex: number;
  onPrevMatch: () => void;
  onNextMatch: () => void;
}

/**
 * Search drawer for the Messages view. Renders as a bottom sheet via portal
 * (consistent with KeyComboPicker) so it floats above the message list.
 * The search input auto-focuses when opened after dismissing any virtual
 * keyboard (same 50ms delay pattern as KeyComboPicker).
 */
export default function MessagesSearchDrawer({
  open,
  onClose,
  query,
  onQueryChange,
  matchCount,
  currentMatchIndex,
  onPrevMatch,
  onNextMatch,
}: MessagesSearchDrawerProps) {
  const searchRef = useRef<HTMLInputElement>(null);

  // Auto-focus search input when opened, with keyboard dismiss delay
  useEffect(() => {
    if (open) {
      if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur();
      }
      const id = setTimeout(() => searchRef.current?.focus(), 50);
      return () => clearTimeout(id);
    }
  }, [open]);

  if (!open) return null;

  const matchLabel = query
    ? matchCount === 0 ? "No matches" : `${currentMatchIndex + 1} of ${matchCount}`
    : "Type to search";

  return createPortal(
    <div
      className="fixed inset-0 z-40"
      onMouseDown={(e) => e.preventDefault()}
    >
      {/* Backdrop */}
      <div
        data-testid="messages-search-backdrop"
        className="absolute inset-0 bg-wc-backdrop"
        onClick={onClose}
      />

      {/* Bottom sheet panel */}
      <div
        data-testid="messages-search-panel"
        className="absolute bottom-0 left-0 right-0 z-50 rounded-t-xl border-t border-wc-default bg-wc-surface-raised pb-[var(--wc-safe-bottom)] shadow-2xl"
      >
        {/* Drag handle */}
        <div className="flex justify-center py-2">
          <div className="h-1 w-8 rounded-full bg-wc-text-muted/40" />
        </div>

        {/* Search bar */}
        <div className="flex items-center gap-2 px-3 pb-3">
          <Search className="h-4 w-4 shrink-0 text-wc-text-muted" />
          <input
            ref={searchRef}
            data-testid="messages-search-input"
            type="text"
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
            placeholder="Search messages…"
            className="min-w-0 flex-1 bg-transparent text-sm text-wc-text-primary placeholder:text-wc-text-muted outline-none"
          />

          {/* Match count / empty state */}
          <span
            data-testid="messages-search-match-count"
            className="shrink-0 text-xs text-wc-text-muted"
          >
            {matchLabel}
          </span>

          {/* Prev / Next / Close */}
          <button
            data-testid="messages-search-prev"
            onClick={onPrevMatch}
            disabled={!query || matchCount === 0}
            className="rounded p-1 text-wc-text-secondary transition hover:bg-wc-surface-input hover:text-wc-text-primary disabled:opacity-30 disabled:pointer-events-none"
            title="Previous match"
          >
            <ChevronUp className="h-4 w-4" />
          </button>
          <button
            data-testid="messages-search-next"
            onClick={onNextMatch}
            disabled={!query || matchCount === 0}
            className="rounded p-1 text-wc-text-secondary transition hover:bg-wc-surface-input hover:text-wc-text-primary disabled:opacity-30 disabled:pointer-events-none"
            title="Next match"
          >
            <ChevronDown className="h-4 w-4" />
          </button>
          <button
            data-testid="messages-search-close"
            onClick={onClose}
            className="rounded p-1 text-wc-text-secondary transition hover:bg-wc-surface-input hover:text-wc-text-primary"
            title="Close search"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>,
    document.body,
  );
}
