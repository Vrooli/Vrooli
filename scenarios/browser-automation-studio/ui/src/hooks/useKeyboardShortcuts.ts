import { useEffect, useCallback, useRef } from "react";

export interface KeyboardShortcut {
  key: string;
  modifiers?: ("ctrl" | "meta" | "alt" | "shift")[];
  description: string;
  category: string;
  action: () => void;
  enabled?: boolean;
}

interface UseKeyboardShortcutsOptions {
  enabled?: boolean;
}

/**
 * Hook for managing keyboard shortcuts across the application.
 * Supports modifier keys (Ctrl, Cmd, Alt, Shift) and provides
 * a consistent way to register and handle shortcuts.
 */
export function useKeyboardShortcuts(
  shortcuts: KeyboardShortcut[],
  options: UseKeyboardShortcutsOptions = {}
) {
  const { enabled = true } = options;
  const shortcutsRef = useRef(shortcuts);
  shortcutsRef.current = shortcuts;

  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (!enabled) return;

      // Don't trigger shortcuts when typing in inputs, textareas, or contenteditable
      const target = event.target as HTMLElement;
      if (
        target.tagName === "INPUT" ||
        target.tagName === "TEXTAREA" ||
        target.isContentEditable
      ) {
        // Exception: Escape should work in inputs
        if (event.key !== "Escape") {
          return;
        }
      }

      const key = event.key.toLowerCase();
      const hasCtrl = event.ctrlKey;
      const hasMeta = event.metaKey;
      const hasAlt = event.altKey;
      const hasShift = event.shiftKey;

      for (const shortcut of shortcutsRef.current) {
        if (shortcut.enabled === false) continue;

        const shortcutKey = shortcut.key.toLowerCase();
        const modifiers = shortcut.modifiers || [];

        // Check if key matches (handle special keys like ?)
        let keyMatches = key === shortcutKey;
        // Handle shift+key combinations (e.g., ? is shift+/)
        if (!keyMatches && shortcutKey === "?" && key === "/" && hasShift) {
          keyMatches = true;
        }
        if (!keyMatches && shortcutKey === "/" && key === "/") {
          keyMatches = true;
        }

        if (!keyMatches) continue;

        // Check modifiers
        const needsCtrl = modifiers.includes("ctrl");
        const needsMeta = modifiers.includes("meta");
        const needsAlt = modifiers.includes("alt");
        const needsShift = modifiers.includes("shift");

        // For cross-platform support, treat ctrl and meta as interchangeable
        const hasCtrlOrMeta = hasCtrl || hasMeta;
        const needsCtrlOrMeta = needsCtrl || needsMeta;

        const modifiersMatch =
          hasCtrlOrMeta === needsCtrlOrMeta &&
          hasAlt === needsAlt &&
          (needsShift ? hasShift : true); // Shift check is special for ? key

        if (modifiersMatch) {
          event.preventDefault();
          event.stopPropagation();
          shortcut.action();
          return;
        }
      }
    },
    [enabled]
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);
}

/**
 * Returns the appropriate modifier key label based on the platform.
 * Mac uses Cmd (⌘), Windows/Linux uses Ctrl.
 */
export function getModifierKey(): string {
  const isMac =
    typeof navigator !== "undefined" &&
    /Mac|iPhone|iPad|iPod/.test(navigator.platform);
  return isMac ? "⌘" : "Ctrl";
}

/**
 * Formats a keyboard shortcut for display.
 */
export function formatShortcut(
  key: string,
  modifiers?: ("ctrl" | "meta" | "alt" | "shift")[]
): string {
  const parts: string[] = [];
  const mod = getModifierKey();

  if (modifiers?.includes("ctrl") || modifiers?.includes("meta")) {
    parts.push(mod);
  }
  if (modifiers?.includes("alt")) {
    parts.push("Alt");
  }
  if (modifiers?.includes("shift")) {
    parts.push("Shift");
  }

  // Format special keys
  let displayKey = key.toUpperCase();
  if (key === "escape") displayKey = "Esc";
  if (key === "enter") displayKey = "Enter";
  if (key === "arrowup") displayKey = "↑";
  if (key === "arrowdown") displayKey = "↓";
  if (key === "arrowleft") displayKey = "←";
  if (key === "arrowright") displayKey = "→";
  if (key === "/") displayKey = "/";
  if (key === "?") displayKey = "?";

  parts.push(displayKey);

  return parts.join(" + ");
}

export default useKeyboardShortcuts;
