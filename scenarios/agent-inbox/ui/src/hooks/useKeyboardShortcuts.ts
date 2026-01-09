import { useEffect, useCallback } from "react";

export interface KeyboardShortcut {
  key: string;
  ctrlKey?: boolean;
  metaKey?: boolean;
  shiftKey?: boolean;
  description: string;
  action: () => void;
  /** If true, prevent default browser behavior */
  preventDefault?: boolean;
  /** Category for grouping in help display */
  category?: "navigation" | "chat" | "general";
}

interface UseKeyboardShortcutsOptions {
  /** Disable all shortcuts (e.g., when a modal is open) */
  disabled?: boolean;
}

/**
 * Hook to register and handle keyboard shortcuts.
 * Shortcuts only fire when no input/textarea is focused (unless specified).
 */
export function useKeyboardShortcuts(
  shortcuts: KeyboardShortcut[],
  options: UseKeyboardShortcutsOptions = {}
) {
  const { disabled = false } = options;

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (disabled) return;

      // Don't trigger shortcuts when typing in inputs (except for specific cases)
      const target = e.target as HTMLElement;
      const isInput = target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable;

      for (const shortcut of shortcuts) {
        const keyMatch = e.key.toLowerCase() === shortcut.key.toLowerCase();
        const ctrlMatch = shortcut.ctrlKey ? (e.ctrlKey || e.metaKey) : !e.ctrlKey && !e.metaKey;
        const shiftMatch = shortcut.shiftKey ? e.shiftKey : !e.shiftKey;

        if (keyMatch && ctrlMatch && shiftMatch) {
          // Allow Escape to work even in inputs
          if (isInput && e.key !== "Escape") {
            continue;
          }

          if (shortcut.preventDefault !== false) {
            e.preventDefault();
          }
          shortcut.action();
          return;
        }
      }
    },
    [shortcuts, disabled]
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);
}

// Helper to format shortcut key for display
export function formatShortcutKey(shortcut: KeyboardShortcut): string {
  const parts: string[] = [];

  if (shortcut.ctrlKey) {
    // Show Cmd on Mac, Ctrl on Windows/Linux
    const isMac = typeof navigator !== "undefined" && navigator.platform.includes("Mac");
    parts.push(isMac ? "Cmd" : "Ctrl");
  }

  if (shortcut.shiftKey) {
    parts.push("Shift");
  }

  // Format special keys nicely
  let key = shortcut.key;
  if (key === " ") key = "Space";
  if (key === "Escape") key = "Esc";
  if (key.length === 1) key = key.toUpperCase();

  parts.push(key);

  return parts.join(" + ");
}
