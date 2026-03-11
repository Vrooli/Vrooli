import { useEffect, useCallback } from "react";
import {
  emitShortcutIntent,
  HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER,
  type BridgeShortcutOutcome,
} from "@vrooli/iframe-bridge";

/**
 * Central keyboard shortcut handler for this scenario.
 *
 * Architecture rules (see vrooli-ui-interop skill):
 * - This is the ONE place that owns window keydown listeners
 * - Components do NOT add their own keydown listeners for app shortcuts
 *   (dialog-local Escape handlers are fine)
 * - Unhandled shortcuts are relayed to the host via iframe-bridge
 */

interface ShortcutHandlers {
  /** Return true if the shortcut was handled locally */
  onSearch?: () => boolean;
  /** Called when Escape is pressed (e.g., close modal) */
  onEscape?: () => boolean;
  /** Called when Ctrl/Cmd+S is pressed (e.g., save) */
  onSave?: () => boolean;
}

/**
 * Check if the active element is an input that should receive keystrokes
 */
function isInputElement(el: HTMLElement): boolean {
  return (
    el.tagName === "INPUT" ||
    el.tagName === "TEXTAREA" ||
    el.isContentEditable ||
    el.closest(".monaco-editor") !== null
  );
}

/**
 * Hook for centralized keyboard shortcut handling.
 *
 * This hook provides:
 * 1. Single root keydown listener at app shell level
 * 2. Input suppression (skip shortcuts when in input/textarea/contentEditable)
 * 3. Local-first handling with iframe relay for unhandled shortcuts
 * 4. Shared host action constants from @vrooli/iframe-bridge
 *
 * @example
 * ```tsx
 * // In MainLayout.tsx or App.tsx
 * useKeyboardShortcuts({
 *   onSearch: () => {
 *     setSearchOpen(true);
 *     return true;
 *   },
 * });
 * ```
 */
export function useKeyboardShortcuts(handlers: ShortcutHandlers = {}): void {
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      const target = event.target as HTMLElement;
      if (isInputElement(target)) return;

      const mod = event.metaKey || event.ctrlKey;

      // Ctrl/Cmd+K — Global search / switcher
      if (mod && event.key === "k") {
        event.preventDefault();
        const handled = handlers.onSearch?.() ?? false;
        const outcome: BridgeShortcutOutcome = handled ? "handled" : "noop";
        if (outcome !== "handled") {
          emitShortcutIntent({
            action: HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER,
            outcome,
            chord: "mod+k",
            source: "keyboard",
          });
        }
        return;
      }

      // Ctrl/Cmd+S — Save
      if (mod && event.key === "s") {
        event.preventDefault();
        const handled = handlers.onSave?.() ?? false;
        const outcome: BridgeShortcutOutcome = handled ? "handled" : "noop";
        if (outcome !== "handled") {
          emitShortcutIntent({
            action: "save",
            outcome,
            chord: "mod+s",
            source: "keyboard",
          });
        }
        return;
      }

      // Escape — Close/cancel
      if (event.key === "Escape") {
        const handled = handlers.onEscape?.() ?? false;
        // Don't relay Escape to host - it's typically UI-local
        if (handled) {
          event.preventDefault();
        }
        return;
      }
    },
    [handlers],
  );

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);
}

export default useKeyboardShortcuts;
