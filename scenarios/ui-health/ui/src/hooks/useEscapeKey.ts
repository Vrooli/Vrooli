import { useEffect } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

/** Calls the latest callback when an enabled dismissible surface receives Escape. */
export function useEscapeKey(enabled: boolean, onEscape: () => void) {
  useEffect(() => {
    if (!enabled) return;

    const handler = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      onEscape();
      emitShortcutIntent({
        action: "dialog.close",
        outcome: "handled",
        chord: "Escape",
        source: "keyboard",
      });
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [enabled, onEscape]);
}
