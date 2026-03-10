import { useState, useRef, useCallback } from "react";
import type { PointerEvent as ReactPointerEvent, KeyboardEvent as ReactKeyboardEvent } from "react";
import { GripVertical, Palette, X } from "lucide-react";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { cn } from "../lib/classnames";
import { Button } from "./ui/button";

interface TerminalHeaderProps {
  sessionId: string;
  name: string;
  headerColor: string;
  isActive: boolean;
  onClose: () => void;
  onFocus: () => void;
  onDragStart?: (sessionId: string, e: ReactPointerEvent) => void;
}

export default function TerminalHeader({
  sessionId,
  name,
  headerColor,
  isActive,
  onClose,
  onFocus,
  onDragStart,
}: TerminalHeaderProps) {
  const renamePaneById = useWorkspaceStore((s) => s.renamePaneById);
  const movePaneToIndex = useWorkspaceStore((s) => s.movePaneToIndex);
  const setAppearanceModalPane = useWorkspaceStore((s) => s.setAppearanceModalPane);
  const panes = useWorkspaceStore((s) => s.panes);

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

  const bgStyle =
    headerColor !== "transparent" ? { backgroundColor: headerColor } : undefined;

  return (
    <div
      data-testid={`terminal-header-${sessionId}`}
      className={cn(
        "flex h-7 items-center gap-1 px-1.5 text-xs select-none",
        isActive ? "border-b-2 border-wc-accent" : "border-b border-wc-default",
      )}
      style={bgStyle ?? { backgroundColor: "rgb(var(--wc-surface-header))" }}
      onClick={onFocus}
    >
      {/* Drag handle */}
      <button
        type="button"
        data-testid={`terminal-drag-handle-${sessionId}`}
        className="flex h-5 w-5 items-center justify-center shrink-0 text-wc-text-faint hover:text-wc-text-secondary cursor-grab active:cursor-grabbing touch-none"
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
        aria-label={`Reorder ${name}`}
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
          onChange={(e) => setEditValue(e.target.value)}
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
          title="Click to rename"
        >
          {name}
        </span>
      )}

      {/* Appearance button */}
      <button
        data-testid={`terminal-header-appearance-${sessionId}`}
        type="button"
        className="flex h-5 w-5 items-center justify-center rounded shrink-0 text-wc-text-faint hover:text-wc-text-secondary"
        onClick={(e) => {
          e.stopPropagation();
          setAppearanceModalPane(sessionId);
        }}
        title="Appearance settings"
      >
        <Palette className="h-3 w-3" />
      </button>

      {/* Close button */}
      <Button
        data-testid={`terminal-close-${sessionId}`}
        variant="ghost"
        size="icon"
        className="h-5 w-5 shrink-0 text-wc-text-faint"
        onClick={(e) => {
          e.stopPropagation();
          onClose();
        }}
        title="Close terminal"
      >
        <X className="h-3 w-3" />
      </Button>
    </div>
  );
}
