import { useState, useRef, useCallback } from "react";
import { GripVertical, PaintBucket, X } from "lucide-react";
import { HEADER_COLORS } from "../consts/config";
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
}

export default function TerminalHeader({
  sessionId,
  name,
  headerColor,
  isActive,
  onClose,
  onFocus,
}: TerminalHeaderProps) {
  const renamePaneById = useWorkspaceStore((s) => s.renamePaneById);
  const setPaneColor = useWorkspaceStore((s) => s.setPaneColor);

  const [editing, setEditing] = useState(false);
  const [editValue, setEditValue] = useState(name);
  const [colorPickerOpen, setColorPickerOpen] = useState(false);
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
      {/* Drag indicator */}
      <GripVertical className="h-3 w-3 shrink-0 text-wc-text-faint" />

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

      {/* Color picker */}
      <div className="relative">
        <button
          data-testid={`terminal-header-color-${sessionId}`}
          type="button"
          className="flex h-5 w-5 items-center justify-center rounded shrink-0 text-wc-text-faint hover:text-wc-text-secondary"
          onClick={(e) => {
            e.stopPropagation();
            setColorPickerOpen((prev) => !prev);
          }}
          title="Set header color"
        >
          <PaintBucket
            className="h-3 w-3"
            style={
              headerColor !== "transparent"
                ? { color: headerColor }
                : undefined
            }
          />
        </button>
        {colorPickerOpen && (
          <div
            data-testid={`terminal-header-color-picker-${sessionId}`}
            className="absolute right-0 top-6 z-20 flex flex-wrap gap-1 rounded border border-wc-default bg-wc-surface-raised p-1.5 shadow-lg"
            style={{ width: "120px" }}
          >
            {/* Transparent option */}
            <button
              type="button"
              className={cn(
                "h-5 w-5 rounded-full border",
                headerColor === "transparent"
                  ? "border-wc-accent"
                  : "border-wc-default",
              )}
              style={{ background: "rgb(var(--wc-surface-input))" }}
              onClick={(e) => {
                e.stopPropagation();
                setPaneColor(sessionId, "transparent");
                setColorPickerOpen(false);
              }}
              title="No color"
            />
            {HEADER_COLORS.map((color) => (
              <button
                key={color}
                type="button"
                className={cn(
                  "h-5 w-5 rounded-full border",
                  headerColor === color
                    ? "border-wc-accent"
                    : "border-wc-default",
                )}
                style={{ backgroundColor: color }}
                onClick={(e) => {
                  e.stopPropagation();
                  setPaneColor(sessionId, color);
                  setColorPickerOpen(false);
                }}
                title={color}
              />
            ))}
          </div>
        )}
      </div>

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
