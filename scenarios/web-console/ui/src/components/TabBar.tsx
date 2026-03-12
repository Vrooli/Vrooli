import { useCallback, useEffect, useRef, useState } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";
import { Plus, X } from "lucide-react";
import { useWorkspaceStore, type PaneMetadata } from "../stores/useWorkspaceStore";
import { cn } from "../lib/classnames";
import { Button } from "./ui/button";
import { useLongPress } from "../hooks/useLongPress";

interface TabBarProps {
  panes: PaneMetadata[];
  activePane: string | null;
  onNewTerminal: () => void;
  onOpenLauncher: () => void;
  onClosePane: (sessionId: string) => void;
  isCreating?: boolean;
}

export default function TabBar({
  panes,
  activePane,
  onNewTerminal,
  onOpenLauncher,
  onClosePane,
  isCreating,
}: TabBarProps) {
  const setActivePane = useWorkspaceStore((s) => s.setActivePane);
  const movePaneToIndex = useWorkspaceStore((s) => s.movePaneToIndex);
  const setAppearanceModalPane = useWorkspaceStore((s) => s.setAppearanceModalPane);
  const displayMode = useWorkspaceStore((s) => s.displayMode);

  // Long-press detection for opening appearance modal on tabs
  const longPressTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const longPressFired = useRef(false);

  const plusHandlers = useLongPress({
    onPress: onNewTerminal,
    onLongPress: onOpenLauncher,
  });

  const tabContainerRef = useRef<HTMLDivElement>(null);
  const activeTabRef = useRef<HTMLButtonElement>(null);

  // Drag state for tab reordering
  const [dragState, setDragState] = useState<{
    paneId: string;
    dropIndex: number;
  } | null>(null);

  // Auto-scroll active tab into view
  useEffect(() => {
    if (activeTabRef.current && tabContainerRef.current) {
      activeTabRef.current.scrollIntoView({
        behavior: "smooth",
        block: "nearest",
        inline: "nearest",
      });
    }
  }, [activePane]);

  // Keyboard shortcuts for tab navigation (only in tabs mode)
  useEffect(() => {
    if (displayMode !== "tabs") return;

    const handleKeyDown = (e: KeyboardEvent) => {
      const activeIdx = panes.findIndex((p) => p.sessionId === activePane);

      // Ctrl+Tab / Ctrl+Shift+Tab - cycle tabs
      if (e.ctrlKey && e.key === "Tab") {
        e.preventDefault();
        if (panes.length === 0) return;

        const direction = e.shiftKey ? -1 : 1;
        const nextIdx = (activeIdx + direction + panes.length) % panes.length;
        const nextPane = panes[nextIdx];
        if (nextPane) {
          setActivePane(nextPane.sessionId);
        }
        return;
      }

      // Ctrl+1-9 - jump to tab by index
      if (e.ctrlKey && !e.shiftKey && !e.altKey && /^[1-9]$/.test(e.key)) {
        const idx = parseInt(e.key, 10) - 1;
        if (idx < panes.length) {
          e.preventDefault();
          const targetPane = panes[idx];
          if (targetPane) {
            setActivePane(targetPane.sessionId);
          }
        }
        return;
      }

      // Ctrl+W - close active tab
      if (e.ctrlKey && !e.shiftKey && !e.altKey && e.key === "w") {
        e.preventDefault();
        if (activePane) {
          onClosePane(activePane);
        }
        return;
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [displayMode, panes, activePane, setActivePane, onClosePane]);

  // Tab drag-to-reorder using pointer capture
  const startTabDrag = useCallback(
    (paneId: string, e: ReactPointerEvent) => {
      const idx = panes.findIndex((p) => p.sessionId === paneId);
      if (idx === -1) return;
      e.preventDefault();
      e.stopPropagation();
      (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
      setDragState({ paneId, dropIndex: idx });
    },
    [panes],
  );

  useEffect(() => {
    if (dragState === null) return;

    const handleMove = (e: PointerEvent) => {
      const el = document.elementFromPoint(e.clientX, e.clientY);
      const tabEl = el?.closest("[data-tab-index]");
      if (tabEl) {
        const idx = Number(tabEl.getAttribute("data-tab-index"));
        if (Number.isFinite(idx)) {
          setDragState((prev) =>
            prev ? { ...prev, dropIndex: idx } : null,
          );
        }
      }
    };

    const handleUp = () => {
      setDragState((prev) => {
        if (prev) movePaneToIndex(prev.paneId, prev.dropIndex);
        return null;
      });
    };

    window.addEventListener("pointermove", handleMove);
    window.addEventListener("pointerup", handleUp);
    window.addEventListener("pointercancel", handleUp);
    return () => {
      window.removeEventListener("pointermove", handleMove);
      window.removeEventListener("pointerup", handleUp);
      window.removeEventListener("pointercancel", handleUp);
    };
  }, [dragState, movePaneToIndex]);

  const isDragging = dragState !== null;

  return (
    <div
      data-testid="tab-bar"
      className="flex items-stretch h-9 border-b border-wc-default bg-wc-surface-header shrink-0"
      role="tablist"
    >
      <div
        ref={tabContainerRef}
        className="flex-1 flex items-stretch overflow-x-auto wc-hide-scrollbar"
      >
        {panes.map((pane, idx) => {
          const isActive = pane.sessionId === activePane;
          const isBeingDragged = dragState?.paneId === pane.sessionId;
          const isDropTarget =
            isDragging && !isBeingDragged && dragState?.dropIndex === idx;

          return (
            <button
              key={pane.sessionId}
              ref={isActive ? activeTabRef : undefined}
              data-testid={`tab-${pane.sessionId}`}
              data-tab-index={idx}
              role="tab"
              aria-selected={isActive}
              className={cn(
                "group relative flex items-center gap-1.5 h-full px-3 text-xs shrink-0 border-r border-wc-default transition-colors",
                "hover:bg-wc-surface-raised focus:outline-none focus-visible:ring-1 focus-visible:ring-wc-accent",
                isActive
                  ? "bg-wc-surface-base text-wc-text-primary font-medium shadow-[inset_0_-2px_0_0_rgb(var(--wc-accent))]"
                  : "bg-wc-surface-header text-wc-text-secondary",
                isBeingDragged && "opacity-40",
                isDropTarget && "ring-2 ring-blue-400/60 ring-inset",
              )}
              onClick={() => setActivePane(pane.sessionId)}
              onContextMenu={(e) => {
                e.preventDefault();
                if (longPressTimer.current) {
                  clearTimeout(longPressTimer.current);
                  longPressTimer.current = null;
                }
                longPressFired.current = true;
                setAppearanceModalPane(pane.sessionId);
              }}
              onPointerDown={(e) => {
                // Ctrl+left-click starts drag reorder
                if (e.button === 0 && e.ctrlKey) {
                  startTabDrag(pane.sessionId, e);
                  return;
                }
                // Start long-press timer for touch/pen
                if (e.pointerType !== "mouse" && e.button === 0) {
                  longPressFired.current = false;
                  longPressTimer.current = setTimeout(() => {
                    longPressFired.current = true;
                    setAppearanceModalPane(pane.sessionId);
                  }, 500);
                }
              }}
              onPointerUp={() => {
                if (longPressTimer.current) {
                  clearTimeout(longPressTimer.current);
                  longPressTimer.current = null;
                }
                // Activate tab immediately on pointer-up rather than waiting
                // for onClick, which mobile browsers may delay or suppress
                // when the element is inside a scrollable container.
                if (!longPressFired.current && !isDragging) {
                  setActivePane(pane.sessionId);
                }
              }}
              onPointerCancel={() => {
                if (longPressTimer.current) {
                  clearTimeout(longPressTimer.current);
                  longPressTimer.current = null;
                }
              }}
            >
              {/* Color indicator */}
              {pane.headerColor !== "transparent" && (
                <span
                  className="absolute left-0 top-0 bottom-0 w-0.5"
                  style={{ backgroundColor: pane.headerColor }}
                />
              )}

              {/* Tab name */}
              <span className="truncate max-w-[120px]">{pane.name}</span>

              {/* Close button - visible on hover or when active */}
              <button
                type="button"
                className={cn(
                  "flex items-center justify-center h-4 w-4 rounded hover:bg-wc-surface-input",
                  "opacity-0 group-hover:opacity-100 focus:opacity-100",
                  isActive && "opacity-100",
                )}
                onClick={(e) => {
                  e.stopPropagation();
                  onClosePane(pane.sessionId);
                }}
                title="Close tab"
                aria-label={`Close ${pane.name}`}
              >
                <X className="h-3 w-3" />
              </button>
            </button>
          );
        })}
      </div>

      {/* New tab button */}
      <Button
        data-testid="tab-bar-new"
        variant="ghost"
        size="icon"
        className="h-7 w-7 shrink-0 mx-1 self-center"
        disabled={isCreating}
        title="New terminal (long-press for launcher)"
        onPointerDown={plusHandlers.onPointerDown}
        onPointerUp={plusHandlers.onPointerUp}
        onPointerCancel={plusHandlers.onPointerCancel}
        onContextMenu={plusHandlers.onContextMenu}
      >
        <Plus className="h-4 w-4" />
      </Button>
    </div>
  );
}
