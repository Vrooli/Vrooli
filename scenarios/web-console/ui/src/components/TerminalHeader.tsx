import { useState, useRef, useCallback } from "react";
import type { PointerEvent as ReactPointerEvent, KeyboardEvent as ReactKeyboardEvent } from "react";
import { GripVertical, MessageSquareText, TerminalSquare, Palette, Send, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { PaneViewMode } from "../stores/useConversationStore";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import { paneAccentStyle } from "../lib/paneColor";
import { IconButton } from "@vrooli/react-component-library/IconButton";

interface TerminalHeaderProps {
  sessionId: string;
  name: string;
  headerColor: string;
  isActive: boolean;
  viewMode?: PaneViewMode;
  unreadCount?: number;
  onClose: () => void;
  onFocus: () => void;
  onToggleView?: () => void;
  /** True while the view is mid-switch; shows a spinner on the toggle button. */
  isViewSwitchPending?: boolean;
  /**
   * Open the handoff composer from this pane, carrying no payload.
   *
   * This is the always-available manual path: it needs no text analysis and
   * no detection, which is why the terminal view itself never grows a
   * suggestion overlay (decision D11).
   */
  onHandoff?: (sessionId: string) => void;
  onDragStart?: (sessionId: string, e: ReactPointerEvent) => void;
}

export default function TerminalHeader({
  sessionId,
  name,
  headerColor,
  isActive,
  viewMode = "terminal",
  unreadCount = 0,
  onClose,
  onFocus,
  onToggleView,
  isViewSwitchPending = false,
  onHandoff,
  onDragStart,
}: TerminalHeaderProps) {
  const { t } = useTranslation();
  const renamePaneById = useWorkspaceStore((s) => s.renamePaneById);
  const movePaneToIndex = useWorkspaceStore((s) => s.movePaneToIndex);
  const setAppearanceModalPane = useWorkspaceStore((s) => s.setAppearanceModalPane);
  const panes = useWorkspaceStore((s) => s.panes);
  // Group color, for panes grouped before joining a group started seeding the
  // pane's own color. Selecting the resolved color (a primitive) rather than
  // the group object keeps this subscription from re-rendering the header on
  // unrelated group edits.
  const groupColor = useWorkspaceStore((s) => {
    const groupId = s.panes.find((p) => p.sessionId === sessionId)?.groupId;
    return groupId ? s.groups.find((g) => g.id === groupId)?.color ?? null : null;
  });

  // Whether this pane's group holds another member. A primitive selector, so
  // an unrelated group edit never re-renders the header.
  const canHandoff = useWorkspaceStore((s) => {
    const groupId = s.panes.find((p) => p.sessionId === sessionId)?.groupId;
    if (!groupId) return false;
    const panes = s.panes.filter((p) => p.groupId === groupId).length;
    const roles = s.roles.filter((r) => r.groupId === groupId && r.sessionId === null).length;
    return panes + roles > 1;
  });

  const [editing, setEditing] = useState(false);
  const [editValue, setEditValue] = useState(name);
  const inputRef = useRef<HTMLInputElement>(null);

  const commitRename = useCallback(() => {
    const trimmed = editValue.trim();
    if (trimmed && trimmed !== name) {
      renamePaneById(sessionId, trimmed);
    } else {
      setEditValue(name);
    }
    setEditing(false);
  }, [editValue, name, renamePaneById, sessionId]);

  const startEditing = useCallback(() => {
    setEditValue(name);
    setEditing(true);
    requestAnimationFrame(() => inputRef.current?.select());
  }, [name]);

  const bgStyle = paneAccentStyle(headerColor, groupColor, "header");

  return (
    <div
      data-testid={`terminal-header-${sessionId}`}
      className={cn(
        "flex h-11 md:h-7 items-center gap-1 px-1.5 text-xs select-none",
        isActive ? "border-b-2 border-wc-accent" : "border-b border-wc-default",
      )}
      style={bgStyle ?? { backgroundColor: "rgb(var(--wc-surface-header))" }}
      onClick={onFocus}
    >
      {/* Drag handle */}
      <button
        type="button"
        data-testid={`terminal-drag-handle-${sessionId}`}
        className="flex h-11 w-11 md:h-5 md:w-5 items-center justify-center shrink-0 text-wc-text-faint hover:text-wc-text-secondary cursor-grab active:cursor-grabbing touch-none"
        onPointerDown={(e) => onDragStart?.(sessionId, e)}
        onKeyDown={(e: ReactKeyboardEvent) => {
          const idx = panes.findIndex((p) => p.sessionId === sessionId);
          if (idx === -1) return;
          if (e.key === "ArrowUp" && idx > 0) {
            e.preventDefault();
            movePaneToIndex(sessionId, idx - 1);
          } else if (e.key === "ArrowDown" && idx < panes.length - 1) {
            e.preventDefault();
            movePaneToIndex(sessionId, idx + 1);
          }
        }}
        aria-label={t(strings.terminalHeader.reorderAria, { name })}
        aria-roledescription="drag handle"
      >
        <GripVertical className="h-3 w-3" />
      </button>

      {/* Editable name */}
      {editing ? (
        <input
          ref={inputRef}
          data-testid={`terminal-header-name-input-${sessionId}`}
          className="min-w-0 flex-1 bg-transparent text-xs text-wc-text-primary outline-none"
          value={editValue}
          onChange={(e) => { setEditValue(e.target.value); }}
          onBlur={commitRename}
          onKeyDown={(e) => {
            if (e.key === "Enter") commitRename();
            if (e.key === "Escape") {
              setEditValue(name);
              setEditing(false);
            }
          }}
        />
      ) : (
        <span
          data-testid={`terminal-header-name-${sessionId}`}
          className="min-w-0 flex-1 truncate text-wc-text-secondary cursor-pointer"
          onClick={(e) => {
            e.stopPropagation();
            startEditing();
          }}
          title={t(strings.terminalHeader.renameTitle)}
        >
          {name}
        </span>
      )}

      {unreadCount > 0 && (
        <span className="rounded-full bg-wc-accent px-1.5 py-0.5 text-[10px] font-semibold text-wc-accent-fg">
          {unreadCount}
        </span>
      )}

      {onToggleView && (
        <IconButton
          data-testid={`terminal-header-toggle-view-${sessionId}`}
          // Per pane: the header survives a view switch today, but the pane
          // itself is unmounted whenever its tab is culled, and the toggle
          // should still animate when the pane comes back.
          swapIdentity={`pane-view-toggle-${sessionId}`}
          // The standing surface this control always had, now expressed once
          // rather than as six utility classes per call site.
          surface="soft"
          size="xs"
          denseTapTarget
          className="shrink-0"
          onClick={(e) => {
            e.stopPropagation();
            onToggleView();
          }}
          // Not `pending`: the view mode flips synchronously on click while
          // the pending window stays open through hydration, so the icon
          // changes *inside* that window. Dimming the control is enough
          // feedback and leaves the swap visible.
          disabled={isViewSwitchPending}
          aria-label={viewMode === "terminal" ? t(strings.terminalHeader.showMessages) : t(strings.terminalHeader.showTerminal)}
        >
          {viewMode === "terminal" ? <MessageSquareText /> : <TerminalSquare />}
        </IconButton>
      )}

      {/* The control is offered only when the pane's group holds someone to
          hand off TO — a composer with no targets is a dead end. */}
      {onHandoff && canHandoff && (
        <IconButton
          data-testid={`handoff-pane-header-${sessionId}`}
          size="sm"
          className="shrink-0"
          onClick={(e) => {
            e.stopPropagation();
            onHandoff(sessionId);
          }}
          aria-label={t(strings.terminalHeader.handOff)}
        >
          <Send />
        </IconButton>
      )}

      {/* Appearance button */}
      <IconButton
        data-testid={`terminal-header-appearance-${sessionId}`}
        size="sm"
        className="shrink-0"
        onClick={(e) => {
          e.stopPropagation();
          setAppearanceModalPane(sessionId);
        }}
        aria-label={t(strings.terminalHeader.appearanceTitle)}
      >
        <Palette />
      </IconButton>

      {/* Close button */}
      <IconButton
        data-testid={`terminal-close-${sessionId}`}
        size="sm"
        className="shrink-0"
        onClick={(e) => {
          e.stopPropagation();
          onClose();
        }}
        aria-label={t(strings.terminalHeader.closeTitle)}
      >
        <X />
      </IconButton>
    </div>
  );
}
