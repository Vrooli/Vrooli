import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { useFloatingPosition } from "../hooks/useFloatingPosition";

interface TerminalContextMenuProps {
  /** Viewport coordinates where the menu should appear. */
  position: { x: number; y: number };
  /** Whether a text selection is active (controls Copy visibility). */
  hasSelection: boolean;
  onCopy: () => void;
  /** Called with clipboard text after a successful read. */
  onPaste: (text: string) => void;
  onSelectAll: () => void;
  onClear: () => void;
  onUploadImage?: () => void;
  onSpeak?: () => void;
  onClose: () => void;
}

export default function TerminalContextMenu({
  position,
  hasSelection,
  onCopy,
  onPaste,
  onSelectAll,
  onClear,
  onUploadImage,
  onSpeak,
  onClose,
}: TerminalContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);
  const [menuSize, setMenuSize] = useState<{ width: number; height: number } | null>(null);
  const [pasteError, setPasteError] = useState(false);
  const { clampPosition } = useFloatingPosition();

  // Measure menu after first render
  useLayoutEffect(() => {
    const el = menuRef.current;
    if (!el) return;
    setMenuSize({ width: el.offsetWidth, height: el.offsetHeight });
  }, []);

  // Dismiss on Escape
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        onClose();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

  const handlePaste = useCallback(async () => {
    try {
      const text = await navigator.clipboard.readText();
      if (text) {
        onPaste(text);
      }
      onClose();
    } catch {
      setPasteError(true);
      setTimeout(() => setPasteError(false), 2000);
    }
  }, [onPaste, onClose]);

  // Compute clamped position (invisible until measured)
  const clamped = menuSize
    ? clampPosition(position.x, position.y, menuSize)
    : null;

  const itemClass =
    "w-full text-left px-3 py-2 text-sm text-wc-text-primary hover:bg-white/10 transition-colors first:rounded-t-lg last:rounded-t-none last:rounded-b-lg";

  return (
    <>
      {/* Backdrop */}
      <div
        data-testid="ctx-backdrop"
        className="fixed inset-0 z-40"
        onClick={onClose}
      />
      {/* Menu */}
      <div
        ref={menuRef}
        data-testid="terminal-context-menu"
        className="fixed z-50 min-w-[140px] rounded-lg border border-wc-default bg-wc-surface-raised shadow-xl py-1"
        style={
          clamped
            ? { left: clamped.x, top: clamped.y }
            : { left: position.x, top: position.y, opacity: 0, pointerEvents: "none" as const }
        }
      >
        {hasSelection && (
          <button
            data-testid="ctx-copy"
            className={itemClass}
            onClick={onCopy}
          >
            Copy
          </button>
        )}
        {hasSelection && onSpeak && (
          <button
            data-testid="ctx-speak"
            className={itemClass}
            onClick={onSpeak}
          >
            Speak
          </button>
        )}
        <button
          data-testid="ctx-paste"
          className={itemClass}
          onClick={handlePaste}
        >
          {pasteError ? "Use Ctrl+V to paste" : "Paste"}
        </button>
        {onUploadImage && (
          <button
            data-testid="ctx-upload-image"
            className={itemClass}
            onClick={onUploadImage}
          >
            Upload Image
          </button>
        )}
        <button
          data-testid="ctx-select-all"
          className={itemClass}
          onClick={onSelectAll}
        >
          Select All
        </button>
        <button
          data-testid="ctx-clear"
          className={itemClass}
          onClick={onClear}
        >
          Clear Terminal
        </button>
      </div>
    </>
  );
}
