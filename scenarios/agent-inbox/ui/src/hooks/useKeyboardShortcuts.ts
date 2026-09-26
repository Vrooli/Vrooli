import { useEffect, useCallback } from "react";

/**
 * Action handler signature.
 *
 * We use an interface with an explicit call signature returning `void` so
 * TypeScript accepts handlers that return nothing, boolean, or anything
 * else — the return value is discarded unless it strictly equals `false`.
 * This avoids `void | boolean` which runs afoul of no-invalid-void-type.
 */
export interface KeyboardShortcutAction {
  (): unknown;
}

export interface KeyboardShortcut {
  key: string;
  ctrlKey?: boolean;
  metaKey?: boolean;
  shiftKey?: boolean;
  description: string;
  /**
   * Action handler. Return `false` to mark the shortcut as unhandled
   * (triggers `onUnhandledShortcut`). Any other return value (including
   * nothing) counts as handled.
   */
  action: KeyboardShortcutAction;
  /** If true, prevent default browser behavior */
  preventDefault?: boolean;
  /** Allow this shortcut to fire while an input/textarea/contentEditable has focus */
  allowInInput?: boolean;
  /** Category for grouping in help display */
  category?: "navigation" | "chat" | "general";
}

interface UseKeyboardShortcutsOptions {
  /** Disable all shortcuts (e.g., when a modal is open) */
  disabled?: boolean;
  /** Called when a shortcut matches but action reports unhandled/no-op via `false` return */
  onUnhandledShortcut?: (shortcut: KeyboardShortcut, event: KeyboardEvent) => void;
}

/**
 * Hook to register and handle keyboard shortcuts.
 * Shortcuts only fire when no input/textarea is focused (unless specified).
 */
export function useKeyboardShortcuts(
  shortcuts: KeyboardShortcut[],
  options: UseKeyboardShortcutsOptions = {}
) {
  const { disabled = false, onUnhandledShortcut } = options;

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (disabled) return;

      // Default behavior: avoid non-modifier shortcuts while typing in editable fields.
      const target = e.target as HTMLElement;
      const isInput = target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable;

      for (const shortcut of shortcuts) {
        const keyMatch = e.key.toLowerCase() === shortcut.key.toLowerCase();
        const ctrlMatch = shortcut.ctrlKey ? (e.ctrlKey || e.metaKey) : !e.ctrlKey && !e.metaKey;
        const shiftMatch = shortcut.shiftKey ? e.shiftKey : !e.shiftKey;

        if (keyMatch && ctrlMatch && shiftMatch) {
          const hasModifier = e.ctrlKey || e.metaKey;
          const canRunInInput = shortcut.allowInInput === true || shortcut.key === "Escape" || hasModifier;
          if (isInput && !canRunInInput) {
            continue;
          }

          if (shortcut.preventDefault !== false) {
            e.preventDefault();
          }
          const handled = shortcut.action();
          if (handled === false) {
            onUnhandledShortcut?.(shortcut, e);
          }
          return;
        }
      }
    },
    [shortcuts, disabled, onUnhandledShortcut]
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
    // Prefer userAgentData (modern) with userAgent fallback (widely supported).
    const nav: { userAgentData?: { platform?: string }; userAgent?: string } =
      typeof navigator !== "undefined" ? navigator : {};
    const platformString = nav.userAgentData?.platform ?? nav.userAgent ?? "";
    const isMac = platformString.toLowerCase().includes("mac");
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
