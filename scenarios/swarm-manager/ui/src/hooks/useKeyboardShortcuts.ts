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
  /** Tab navigation by number key (1-5). Return true if handled. */
  onTabNav?: (key: string) => boolean;
  /** Ctrl/Cmd+K — Global search / switcher. Return true if handled. */
  onSearch?: () => boolean;
}

function isInputElement(el: HTMLElement): boolean {
  return (
    el.tagName === "INPUT" ||
    el.tagName === "TEXTAREA" ||
    el.isContentEditable ||
    el.closest(".monaco-editor") !== null
  );
}

export function useKeyboardShortcuts(handlers: ShortcutHandlers): void {
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

      // Number keys 1-5 — Tab navigation
      if (!mod && /^[1-5]$/.test(event.key)) {
        const handled = handlers.onTabNav?.(event.key) ?? false;
        if (!handled) {
          emitShortcutIntent({
            action: HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER,
            outcome: "noop",
            chord: event.key,
            source: "keyboard",
          });
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
